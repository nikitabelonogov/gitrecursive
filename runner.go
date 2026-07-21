package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

type state int

const (
	statePending state = iota
	stateRunning
	stateOK
	stateFailed
	stateTimeout
	stateSkipped
)

type result struct {
	state    state
	output   []byte
	exitCode int
	errMsg   string
	dur      time.Duration
}

type eventKind int

const (
	evStarted eventKind = iota
	evFinished
)

type event struct {
	idx  int
	kind eventKind
	res  *result
}

func runAll(ctx context.Context, repos []string, gitArgs []string, opts *options, forceColor bool, events chan<- event) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < opts.jobs; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				if runCtx.Err() != nil {
					events <- event{idx: idx, kind: evFinished, res: &result{state: stateSkipped, errMsg: "skipped"}}
					continue
				}
				events <- event{idx: idx, kind: evStarted}
				res := runOne(runCtx, repos[idx], gitArgs, opts, forceColor)
				if opts.failFast && (res.state == stateFailed || res.state == stateTimeout) {
					cancel()
				}
				events <- event{idx: idx, kind: evFinished, res: res}
			}
		}()
	}
	for i := range repos {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	close(events)
}

func runOne(ctx context.Context, repo string, gitArgs []string, opts *options, forceColor bool) *result {
	start := time.Now()
	cctx := ctx
	if opts.timeout > 0 {
		var cancel context.CancelFunc
		cctx, cancel = context.WithTimeout(ctx, opts.timeout)
		defer cancel()
	}
	args := []string{"-C", repo}
	if forceColor {
		args = append(args, "-c", "color.ui=always")
	}
	args = append(args, gitArgs...)
	cmd := exec.CommandContext(cctx, "git", args...)
	setupProcessGroup(cmd)
	// Don't let orphaned children (ssh, credential helpers) holding the
	// output pipe keep Wait blocked after a kill.
	cmd.WaitDelay = time.Second
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	// Output is captured, so interactive credential prompts would hang forever.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	err := cmd.Run()

	res := &result{output: buf.Bytes(), dur: time.Since(start), state: stateOK}
	if err == nil {
		return res
	}
	switch {
	case cctx.Err() == context.DeadlineExceeded:
		res.state = stateTimeout
		res.errMsg = fmt.Sprintf("timeout after %s", opts.timeout)
	case ctx.Err() != nil:
		res.state = stateSkipped
		res.errMsg = "cancelled"
	default:
		res.state = stateFailed
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.exitCode = exitErr.ExitCode()
			res.errMsg = fmt.Sprintf("exit %d", res.exitCode)
		} else {
			res.exitCode = -1
			res.errMsg = err.Error()
		}
	}
	return res
}
