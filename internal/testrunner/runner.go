package testrunner

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Event mirrors the subset of fields produced by `go test -json`.
// This is sufficient for a rich, colored summary and machine-readable output.
type Event struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test,omitempty"`
	Elapsed float64   `json:"Elapsed,omitempty"`
	Output  string    `json:"Output,omitempty"`
}

// Result aggregates outcomes per package and overall.
type Result struct {
	Packages []PackageResult
	Total    int
	Passed   int
	Failed   int
	Skipped  int
	Duration time.Duration
}

// PackageResult captures a single package execution summary.
type PackageResult struct {
	Name     string
	Passed   int
	Failed   int
	Skipped  int
	Duration time.Duration
	Output   []Event
}

// Options control the behavior of the test runner.
type Options struct {
	Packages   []string      // e.g., ["./..."]
	RunPattern string        // -run regex forwarded to go test
	Parallel   int           // number of concurrent packages
	JSON       bool          // stream raw JSON events to writer
	Short      bool          // pass -short to go test
	Race       bool          // pass -race to go test
	Timeout    time.Duration // pass -timeout to go test
	Env        []string      // additional env in KEY=VAL form
	Color      bool          // colorize human-readable output
	ExtraArgs  []string      // extra args to append to `go test`
}

// Runner executes `go test -json` per package with concurrency and aggregates results.
type Runner struct {
	opts Options
}

// New creates a test runner with sane defaults.
func New(opts Options) *Runner {
	if len(opts.Packages) == 0 {
		opts.Packages = []string{"./..."}
	}
	if opts.Parallel <= 0 {
		opts.Parallel = runtime.NumCPU()
		if opts.Parallel <= 0 {
			opts.Parallel = 1
		}
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Minute
	}
	return &Runner{opts: opts}
}

// Run executes tests and writes human-readable output to `out` unless JSON mode is enabled.
func (r *Runner) Run(ctx context.Context, out io.Writer) (Result, error) {
	start := time.Now()
	pkgs := append([]string(nil), r.opts.Packages...)
	// Resolve ./... expansion via go list for stable ordering
	expanded, err := r.goList(ctx, pkgs)
	if err != nil {
		return Result{}, err
	}
	sort.Strings(expanded)
	type item struct {
		pkg string
		res PackageResult
		err error
	}
	workCh := make(chan string)
	resCh := make(chan item)
	var wg sync.WaitGroup
	workerCount := r.opts.Parallel
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range workCh {
				pr, er := r.runOne(ctx, p, out)
				resCh <- item{pkg: p, res: pr, err: er}
			}
		}()
	}
	go func() {
		for _, p := range expanded {
			workCh <- p
		}
		close(workCh)
		wg.Wait()
		close(resCh)
	}()

	results := Result{Packages: make([]PackageResult, 0, len(expanded))}
	for it := range resCh {
		if it.err != nil {
			// Record as failed package with error message in output
			results.Packages = append(results.Packages, PackageResult{
				Name: it.pkg, Failed: 1, Output: []Event{{Time: time.Now(), Action: "output", Package: it.pkg, Output: it.err.Error()}},
			})
			results.Total++
			results.Failed++
			continue
		}
		results.Packages = append(results.Packages, it.res)
		results.Total += it.res.Passed + it.res.Failed + it.res.Skipped
		results.Passed += it.res.Passed
		results.Failed += it.res.Failed
		results.Skipped += it.res.Skipped
	}
	results.Duration = time.Since(start)
	// Human-readable summary when not JSON streaming
	if !r.opts.JSON && out != nil {
		r.writeSummary(out, results)
	}
	if results.Failed > 0 {
		return results, errors.New("test failures")
	}
	return results, nil
}

// runOne executes `go test -json` for a single package and aggregates its result.
func (r *Runner) runOne(ctx context.Context, pkg string, out io.Writer) (PackageResult, error) {
	args := []string{"test", "-json"}
	if r.opts.Short {
		args = append(args, "-short")
	}
	if r.opts.Race {
		args = append(args, "-race")
	}
	if r.opts.Timeout > 0 {
		args = append(args, "-timeout", r.opts.Timeout.String())
	}
	if r.opts.RunPattern != "" {
		args = append(args, "-run", r.opts.RunPattern)
	}
	args = append(args, r.opts.ExtraArgs...)
	args = append(args, pkg)
	cmd := exec.CommandContext(ctx, "go", args...)
	if len(r.opts.Env) > 0 {
		cmd.Env = append(cmd.Env, r.opts.Env...)
	}
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return PackageResult{}, err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return PackageResult{}, err
	}

	pr := PackageResult{Name: pkg, Output: make([]Event, 0, 128)}
	dec := newJSONEventDecoder(pipe)
	for dec.Next() {
		ev := dec.Event()
		pr.Output = append(pr.Output, ev)
		// Stream raw JSON if requested
		if r.opts.JSON && out != nil {
			// Marshal the original event again to ensure proper framing per line
			b, _ := json.Marshal(ev)
			_, _ = out.Write(b)
			_, _ = out.Write([]byte("\n"))
		} else if out != nil && ev.Action == "output" && ev.Output != "" {
			// In human-readable mode, forward test output lines with optional colors
			r.writeLine(out, ev)
		}
		// Aggregate counts on pass/fail/skip events
		switch ev.Action {
		case "pass":
			if ev.Test != "" {
				pr.Passed++
			} else if ev.Elapsed > 0 {
				pr.Duration += time.Duration(ev.Elapsed * float64(time.Second))
			}
		case "fail":
			if ev.Test != "" {
				pr.Failed++
			} else if ev.Elapsed > 0 {
				pr.Duration += time.Duration(ev.Elapsed * float64(time.Second))
			}
		case "skip":
			if ev.Test != "" {
				pr.Skipped++
			}
		}
	}
	if err := dec.Err(); err != nil {
		_ = cmd.Process.Kill()
		return pr, err
	}
	if err := cmd.Wait(); err != nil {
		// go test exits non-zero on package failures; allow counts to indicate failure
	}
	return pr, nil
}

// goList expands package patterns using `go list`.
func (r *Runner) goList(ctx context.Context, patterns []string) ([]string, error) {
	args := append([]string{"list"}, patterns...)
	cmd := exec.CommandContext(ctx, "go", args...)
	b, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	return out, nil
}

// jsonEventDecoder reads newline-delimited JSON events from go test.
type jsonEventDecoder struct {
	r   *bufio.Reader
	cur Event
	err error
}

func newJSONEventDecoder(rd io.Reader) *jsonEventDecoder {
	return &jsonEventDecoder{r: bufio.NewReader(rd)}
}

func (d *jsonEventDecoder) Next() bool {
	if d.err != nil {
		return false
	}
	line, err := d.r.ReadBytes('\n')
	if len(line) == 0 && err != nil {
		if errors.Is(err, io.EOF) {
			return false
		}
		d.err = err
		return false
	}
	// Some toolchains may print non-JSON lines; skip those gracefully
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" {
		return d.Next()
	}
	var ev Event
	if e := json.Unmarshal([]byte(trimmed), &ev); e != nil {
		// Fallback: attempt to wrap plain output lines
		ev = Event{Time: time.Now(), Action: "output", Output: trimmed}
	}
	d.cur = ev
	return true
}

func (d *jsonEventDecoder) Event() Event { return d.cur }
func (d *jsonEventDecoder) Err() error   { return d.err }

// writeLine writes one output line with optional ANSI colors.
func (r *Runner) writeLine(w io.Writer, ev Event) {
	s := ev.Output
	if !r.opts.Color {
		_, _ = io.WriteString(w, s)
		return
	}
	// Color heuristics based on common prefixes
	green := "\x1b[32m"
	red := "\x1b[31m"
	yellow := "\x1b[33m"
	cyan := "\x1b[36m"
	reset := "\x1b[0m"
	switch {
	case strings.HasPrefix(s, "=== RUN"):
		s = cyan + s + reset
	case strings.HasPrefix(s, "--- PASS"):
		s = green + s + reset
	case strings.HasPrefix(s, "--- FAIL"):
		s = red + s + reset
	case strings.HasPrefix(s, "--- SKIP"):
		s = yellow + s + reset
	default:
		// highlight file:line patterns
		re := regexp.MustCompile(`(\w+\.go:\d+)`)
		s = re.ReplaceAllString(s, yellow+"$1"+reset)
	}
	_, _ = io.WriteString(w, s)
}

// writeSummary prints a final colored summary of all package outcomes.
func (r *Runner) writeSummary(w io.Writer, res Result) {
	if w == nil {
		return
	}
	bold := func(s string) string {
		if r.opts.Color {
			return "\x1b[1m" + s + "\x1b[0m"
		}
		return s
	}
	green := func(s string) string {
		if r.opts.Color {
			return "\x1b[32m" + s + "\x1b[0m"
		}
		return s
	}
	red := func(s string) string {
		if r.opts.Color {
			return "\x1b[31m" + s + "\x1b[0m"
		}
		return s
	}
	yellow := func(s string) string {
		if r.opts.Color {
			return "\x1b[33m" + s + "\x1b[0m"
		}
		return s
	}
	// Per-package one-liners
	for _, p := range res.Packages {
		status := "ok"
		if p.Failed > 0 {
			status = "fail"
		} else if p.Skipped > 0 && p.Passed == 0 {
			status = "skip"
		}
		line := fmt.Sprintf("%s\t%s\t%.2fs\t(pass:%d fail:%d skip:%d)\n", status, p.Name, p.Duration.Seconds(), p.Passed, p.Failed, p.Skipped)
		switch status {
		case "ok":
			_, _ = io.WriteString(w, green(line))
		case "fail":
			_, _ = io.WriteString(w, red(line))
		default:
			_, _ = io.WriteString(w, yellow(line))
		}
	}
	// Global summary
	sum := fmt.Sprintf("\n%s %d tests, %s %d, %s %d, %s %d in %.2fs\n",
		bold("SUMMARY:"), res.Total,
		green("passed:"), res.Passed,
		red("failed:"), res.Failed,
		yellow("skipped:"), res.Skipped,
		res.Duration.Seconds())
	_, _ = io.WriteString(w, sum)
}
