package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var version = "dev"

type options struct {
	jobs     int
	depth    int
	timeout  time.Duration
	failFast bool
	dryRun   bool
	noColor  bool
}

var exitCode int

func main() {
	opts := &options{}
	root := &cobra.Command{
		Use:   "gitrecursive [flags] <git-command> [git-args...]",
		Short: "Run a git command in every repository under the current directory",
		Long: `gitrecursive discovers every git repository under the current directory
and runs the given git command in each of them in parallel.

Flags must come before the git command; everything after the first
non-flag argument is passed to git verbatim.`,
		Example: `  gitrecursive fetch --all
  gitrecursive -j 4 pull --ff-only
  gitrecursive --depth 1 status -sb
  gitrecursive --dry-run`,
		Version:       version,
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, opts, args)
		},
	}
	f := root.Flags()
	f.SetInterspersed(false)
	f.IntVarP(&opts.jobs, "jobs", "j", runtime.NumCPU(), "number of repositories processed in parallel")
	f.IntVarP(&opts.depth, "depth", "d", -1, "limit discovery depth (0 = current directory only, -1 = unlimited)")
	f.DurationVarP(&opts.timeout, "timeout", "t", 0, "per-repository timeout, e.g. 30s (0 = none)")
	f.BoolVar(&opts.failFast, "fail-fast", false, "cancel remaining repositories after the first failure")
	f.BoolVar(&opts.dryRun, "dry-run", false, "list discovered repositories without running anything")
	f.BoolVar(&opts.noColor, "no-color", false, "disable colored output")

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "gitrecursive:", err)
		os.Exit(1)
	}
	os.Exit(exitCode)
}

func run(cmd *cobra.Command, opts *options, gitArgs []string) error {
	if len(gitArgs) == 0 && !opts.dryRun {
		return cmd.Help()
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	repos, err := findRepos(cwd, opts.depth)
	if err != nil {
		return err
	}
	if len(repos) == 0 {
		fmt.Fprintln(os.Stderr, "no git repositories found under", cwd)
		return nil
	}
	u := newUI(cwd, repos, opts.noColor)
	if opts.dryRun {
		u.printRepoList()
		return nil
	}
	if opts.jobs < 1 {
		opts.jobs = runtime.NumCPU()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	events := make(chan event, 2*len(repos)+4)
	go runAll(ctx, repos, gitArgs, opts, u.color, events)

	start := time.Now()
	var tick <-chan time.Time
	if u.live {
		t := time.NewTicker(90 * time.Millisecond)
		defer t.Stop()
		tick = t.C
	}
	u.renderStatus()
loop:
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				break loop
			}
			u.handle(ev)
		case <-tick:
			u.frame++
			u.redraw()
		}
	}
	if u.finish(time.Since(start)) > 0 {
		exitCode = 1
	}
	return nil
}
