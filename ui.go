package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

const (
	cDim    = "2"
	cRed    = "31"
	cGreen  = "32"
	cYellow = "33"
	cCyan   = "36"
)

var spinnerFrames = []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")

// ui owns stdout: the runner goroutines only send events, and the main loop
// applies them here, so no locking is needed.
type ui struct {
	out       *os.File
	live      bool
	color     bool
	repos     []string
	rel       []string
	st        []state
	started   []time.Time
	results   []*result
	prevLines int
	frame     int
}

func newUI(cwd string, repos []string, noColor bool) *ui {
	live := term.IsTerminal(int(os.Stdout.Fd()))
	u := &ui{
		out:     os.Stdout,
		live:    live,
		color:   live && !noColor && os.Getenv("NO_COLOR") == "",
		repos:   repos,
		rel:     make([]string, len(repos)),
		st:      make([]state, len(repos)),
		started: make([]time.Time, len(repos)),
		results: make([]*result, len(repos)),
	}
	for i, r := range repos {
		if p, err := filepath.Rel(cwd, r); err == nil {
			u.rel[i] = p
		} else {
			u.rel[i] = r
		}
	}
	return u
}

func (u *ui) paint(code, s string) string {
	if !u.color {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func (u *ui) size() (int, int) {
	if w, h, err := term.GetSize(int(u.out.Fd())); err == nil && w > 0 {
		return w, h
	}
	return 80, 24
}

// clipRel shortens a repo path so a status line never wraps.
func (u *ui) clipRel(i int) string {
	rel := u.rel[i]
	w, _ := u.size()
	if max := w - 14; max > 4 && utf8.RuneCountInString(rel) > max {
		r := []rune(rel)
		rel = "…" + string(r[len(r)-max+1:])
	}
	return rel
}

func (u *ui) pathColored(i int) string {
	rel := u.clipRel(i)
	dir, base := filepath.Split(rel)
	if dir == "" {
		return u.paint(cYellow, rel)
	}
	return u.paint(cDim, dir) + u.paint(cYellow, base)
}

func (u *ui) handle(ev event) {
	switch ev.kind {
	case evStarted:
		u.st[ev.idx] = stateRunning
		u.started[ev.idx] = time.Now()
		u.redraw()
	case evFinished:
		u.st[ev.idx] = ev.res.state
		u.results[ev.idx] = ev.res
		if ev.res.state == stateSkipped {
			u.redraw()
			return
		}
		u.printBlock(ev.idx, ev.res)
	}
}

func (u *ui) redraw() {
	if !u.live {
		return
	}
	u.clearStatus()
	u.renderStatus()
}

func (u *ui) clearStatus() {
	if !u.live || u.prevLines == 0 {
		return
	}
	fmt.Fprintf(u.out, "\x1b[%dA\x1b[0J", u.prevLines)
	u.prevLines = 0
}

func (u *ui) renderStatus() {
	if !u.live {
		return
	}
	var idxs []int
	for i, s := range u.st {
		if s == statePending || s == stateRunning {
			idxs = append(idxs, i)
		}
	}
	if len(idxs) == 0 {
		return
	}
	_, h := u.size()
	max := h - 2
	if max < 1 {
		max = 1
	}
	var lines []string
	if len(idxs) > max {
		var keep []int
		for _, i := range idxs {
			if u.st[i] == stateRunning {
				keep = append(keep, i)
			}
		}
		omitted := len(idxs) - len(keep)
		if len(keep) > max-1 {
			omitted += len(keep) - (max - 1)
			keep = keep[:max-1]
		}
		for _, i := range keep {
			lines = append(lines, u.statusLine(i))
		}
		lines = append(lines, u.paint(cDim, fmt.Sprintf("… %d more", omitted)))
	} else {
		for _, i := range idxs {
			lines = append(lines, u.statusLine(i))
		}
	}
	for _, l := range lines {
		fmt.Fprintln(u.out, l)
	}
	u.prevLines = len(lines)
}

func (u *ui) statusLine(i int) string {
	if u.st[i] == stateRunning {
		frame := string(spinnerFrames[u.frame%len(spinnerFrames)])
		elapsed := fmtDur(time.Since(u.started[i]))
		return fmt.Sprintf("%s %s %s", u.paint(cCyan, frame), u.pathColored(i), u.paint(cDim, elapsed))
	}
	return u.paint(cDim, "· "+u.clipRel(i))
}

func (u *ui) printBlock(i int, res *result) {
	u.clearStatus()
	fmt.Fprintln(u.out, u.blockHeader(i, res))
	out := bytes.TrimRight(res.output, "\n")
	if len(out) > 0 {
		u.out.Write(out)
		fmt.Fprintln(u.out)
	} else if res.state == stateOK {
		fmt.Fprintln(u.out, u.paint(cDim, "(no output)"))
	}
	if res.exitCode == -1 && res.errMsg != "" {
		fmt.Fprintln(u.out, u.paint(cRed, res.errMsg))
	}
	fmt.Fprintln(u.out)
	u.renderStatus()
}

func (u *ui) blockHeader(i int, res *result) string {
	w, _ := u.size()
	if w > 100 {
		w = 100
	}
	glyph, gcolor, suffix := "✓", cGreen, ""
	switch res.state {
	case stateFailed:
		glyph, gcolor = "✗", cRed
		if res.exitCode == -1 {
			suffix = "error"
		} else {
			suffix = res.errMsg
		}
	case stateTimeout:
		glyph, gcolor = "✗", cRed
		suffix = res.errMsg
	}
	visible := 3 + 2 + utf8.RuneCountInString(u.clipRel(i)) + 1
	if suffix != "" {
		visible += utf8.RuneCountInString(suffix) + 3
	}
	fill := w - visible
	if fill < 3 {
		fill = 3
	}
	var b strings.Builder
	b.WriteString(u.paint(cDim, "── "))
	b.WriteString(u.paint(gcolor, glyph))
	b.WriteString(" ")
	b.WriteString(u.pathColored(i))
	b.WriteString(" ")
	b.WriteString(u.paint(cDim, strings.Repeat("─", fill)))
	if suffix != "" {
		b.WriteString(u.paint(cRed, " "+suffix))
	}
	return b.String()
}

func (u *ui) printRepoList() {
	for i := range u.repos {
		fmt.Fprintln(u.out, u.pathColored(i))
	}
	fmt.Fprintln(u.out, u.paint(cDim, fmt.Sprintf("%d repositories", len(u.repos))))
}

func (u *ui) finish(total time.Duration) (failed int) {
	u.clearStatus()
	wname := 0
	for i := range u.rel {
		if n := utf8.RuneCountInString(u.rel[i]); n > wname {
			wname = n
		}
	}
	if wname > 48 {
		wname = 48
	}
	var ok, skipped int
	fmt.Fprintln(u.out, u.paint(cDim, strings.Repeat("─", min(60, wname+16))))
	for i, res := range u.results {
		if res == nil {
			res = &result{state: stateSkipped, errMsg: "skipped"}
		}
		name := u.rel[i]
		pad := wname - utf8.RuneCountInString(name)
		if pad < 0 {
			pad = 0
		}
		var glyph, info string
		switch res.state {
		case stateOK:
			ok++
			glyph = u.paint(cGreen, "✓")
			info = u.paint(cDim, fmtDur(res.dur))
		case stateFailed:
			failed++
			glyph = u.paint(cRed, "✗")
			msg := res.errMsg
			if res.exitCode == -1 {
				msg = "error"
			}
			info = u.paint(cRed, msg) + u.paint(cDim, " · "+fmtDur(res.dur))
		case stateTimeout:
			failed++
			glyph = u.paint(cRed, "✗")
			info = u.paint(cRed, res.errMsg)
		default:
			skipped++
			glyph = u.paint(cDim, "•")
			info = u.paint(cDim, "skipped")
		}
		fmt.Fprintf(u.out, "%s %s%s  %s\n", glyph, name, strings.Repeat(" ", pad), info)
	}
	parts := []string{fmt.Sprintf("%d repos", len(u.repos))}
	parts = append(parts, u.paint(cGreen, fmt.Sprintf("%d ok", ok)))
	if failed > 0 {
		parts = append(parts, u.paint(cRed, fmt.Sprintf("%d failed", failed)))
	}
	if skipped > 0 {
		parts = append(parts, u.paint(cDim, fmt.Sprintf("%d skipped", skipped)))
	}
	parts = append(parts, fmtDur(total))
	fmt.Fprintln(u.out)
	fmt.Fprintln(u.out, strings.Join(parts, " · "))
	return failed
}

func fmtDur(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
}
