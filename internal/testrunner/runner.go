package testrunner

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
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
	Tests    map[string][]TestAttempt
}

// TestAttempt is a single attempt result for a test case.
type TestAttempt struct {
	Outcome string        // pass|fail|skip
	Time    time.Duration // duration of this attempt
	Output  string        // accumulated stdout/stderr for this test
}

// Options control the behavior of the test runner.
type Options struct {
	Packages     []string      // e.g., ["./..."]
	RunPattern   string        // -run regex forwarded to go test
	Parallel     int           // number of concurrent packages
	JSON         bool          // stream raw JSON events to writer
	Short        bool          // pass -short to go test
	Race         bool          // pass -race to go test
	Timeout      time.Duration // pass -timeout to go test
	Env          []string      // additional env in KEY=VAL form
	Color        bool          // colorize human-readable output
	ExtraArgs    []string      // extra args to append to `go test`
	JUnitPath    string        // optional JUnit XML output path
	Retries      int           // re-run failing tests up to N times
	FailFast     bool          // stop at first failing package
	PackageRegex string        // optional regex to filter package names after go list expansion
	SummaryJSON  string        // optional JSON summary output path
	AugmentJSON  bool          // when JSON is enabled, emit additional Orizon events (package attempts, flaky recovered)
    FileRegex    string        // optional regex to include only packages that have files matching this regex
    ListOnly     bool          // list test names (per package) without executing them
    FailOnFlaky  bool          // return non-zero if any test recovered from fail to pass after retries
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
	// Create a cancellable context so FailFast can stop in-flight packages.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	start := time.Now()
	pkgs := append([]string(nil), r.opts.Packages...)
	// Resolve ./... expansion via go list for stable ordering
	expanded, err := r.goList(ctx, pkgs)
	if err != nil {
		return Result{}, err
	}
    // Optional package regex filter
	if strings.TrimSpace(r.opts.PackageRegex) != "" {
		re, e := regexp.Compile(r.opts.PackageRegex)
		if e != nil {
			return Result{}, e
		}
		filtered := make([]string, 0, len(expanded))
		for _, p := range expanded {
			if re.MatchString(p) {
				filtered = append(filtered, p)
			}
		}
		expanded = filtered
	}
    // Optional file regex filter (keep only packages that contain a file path matching regex)
    if strings.TrimSpace(r.opts.FileRegex) != "" {
        re, e := regexp.Compile(r.opts.FileRegex)
        if e != nil { return Result{}, e }
        keep := make([]string, 0, len(expanded))
        for _, p := range expanded {
            // Query files via `go list -f` for this package
            ok, _ := r.packageHasMatchingFile(ctx, p, re)
            if ok { keep = append(keep, p) }
        }
        expanded = keep
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
                if r.opts.ListOnly {
                    r.listTests(ctx, p, out)
                    resCh <- item{pkg: p, res: PackageResult{Name: p}, err: nil}
                    continue
                }
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
	cancelled := false
	for it := range resCh {
		if it.err != nil {
			// Record as failed package with error message in output
			results.Packages = append(results.Packages, PackageResult{
				Name: it.pkg, Failed: 1, Output: []Event{{Time: time.Now(), Action: "output", Package: it.pkg, Output: it.err.Error()}},
			})
			results.Total++
			results.Failed++
			if r.opts.FailFast {
				cancelled = true
				cancel()
				break
			}
			continue
		}
		// Optionally retry failing tests up to Retries times and merge attempt history
		pkgRes := it.res
		if r.opts.Retries > 0 && pkgRes.Failed > 0 {
			merged := pkgRes
			for attempt := 1; attempt <= r.opts.Retries; attempt++ {
				rr, _ := r.runOne(ctx, it.pkg, out)
				// merge attempts
				if rr.Tests != nil {
					if merged.Tests == nil {
						merged.Tests = make(map[string][]TestAttempt)
					}
					for name, atts := range rr.Tests {
						merged.Tests[name] = append(merged.Tests[name], atts...)
					}
				}
				merged.Passed = rr.Passed
				merged.Failed = rr.Failed
				merged.Skipped = rr.Skipped
				merged.Duration += rr.Duration
				if rr.Failed == 0 {
					// Augmented JSON event to indicate flaky recovery for this package
					if r.opts.JSON && r.opts.AugmentJSON && out != nil {
						type aug struct {
							Orizon  string `json:"orizon"`
							Kind    string `json:"kind"`
							Package string `json:"package"`
							Retries int    `json:"retries"`
						}
						a := aug{Orizon: "test", Kind: "flaky_recovered", Package: it.pkg, Retries: attempt}
						bb, _ := json.Marshal(a)
						_, _ = out.Write(bb)
						_, _ = out.Write([]byte("\n"))
					}
					break
				}
			}
			pkgRes = merged
		}
		results.Packages = append(results.Packages, pkgRes)
		results.Total += pkgRes.Passed + pkgRes.Failed + pkgRes.Skipped
		results.Passed += pkgRes.Passed
		results.Failed += pkgRes.Failed
		results.Skipped += pkgRes.Skipped
		if r.opts.FailFast && pkgRes.Failed > 0 {
			cancelled = true
			cancel()
			break
		}
	}
	if cancelled {
		// drain remaining items
		for range resCh {
		}
	}
	results.Duration = time.Since(start)
	// Optional JSON summary with attempt history
	if strings.TrimSpace(r.opts.SummaryJSON) != "" {
		_ = r.writeSummaryJSON(r.opts.SummaryJSON, results)
	}
	// Always attempt to write JUnit output for CI consumption before returning.
	if r.opts.JUnitPath != "" {
		_ = writeJUnit(r.opts.JUnitPath, results)
	}
	// Human-readable summary when not JSON streaming
	if !r.opts.JSON && out != nil {
		r.writeSummary(out, results)
	}
    if results.Failed > 0 {
        return results, errors.New("test failures")
    }
    if r.opts.FailOnFlaky {
        // detect any test with fail then pass in attempts
        flaky := false
        for _, p := range results.Packages {
            for _, ats := range p.Tests {
                sawFail := false
                final := ""
                for _, a := range ats { if a.Outcome == "fail" { sawFail = true }; final = a.Outcome }
                if sawFail && final == "pass" { flaky = true; break }
            }
            if flaky { break }
        }
        if flaky { return results, errors.New("flaky tests detected") }
    }
	return results, nil
}

// writeSummaryJSON writes a machine-readable summary including per-test attempt history.
func (r *Runner) writeSummaryJSON(path string, res Result) error {
	type attempt struct {
		Outcome string `json:"outcome"`
		TimeMs  int64  `json:"time_ms"`
		Output  string `json:"output,omitempty"`
	}
	type testEntry struct {
		Package  string    `json:"package"`
		Name     string    `json:"name"`
		Attempts []attempt `json:"attempts"`
	}
	type summary struct {
		Total      int         `json:"total"`
		Passed     int         `json:"passed"`
		Failed     int         `json:"failed"`
		Skipped    int         `json:"skipped"`
		DurationMs int64       `json:"duration_ms"`
		Tests      []testEntry `json:"tests"`
	}
	sm := summary{Total: res.Total, Passed: res.Passed, Failed: res.Failed, Skipped: res.Skipped, DurationMs: int64(res.Duration / time.Millisecond)}
	for _, p := range res.Packages {
		if len(p.Tests) == 0 {
			continue
		}
		for name, ats := range p.Tests {
			te := testEntry{Package: p.Name, Name: name}
			for _, a := range ats {
				te.Attempts = append(te.Attempts, attempt{Outcome: a.Outcome, TimeMs: int64(a.Time / time.Millisecond), Output: a.Output})
			}
			sm.Tests = append(sm.Tests, te)
		}
	}
	b, err := json.MarshalIndent(sm, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
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
		// Preserve parent environment and append user-provided KEY=VAL entries.
		cmd.Env = append(os.Environ(), r.opts.Env...)
	}
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return PackageResult{}, err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return PackageResult{}, err
	}

	pr := PackageResult{Name: pkg, Output: make([]Event, 0, 128), Tests: make(map[string][]TestAttempt)}
	dec := newJSONEventDecoder(pipe)
	starts := make(map[string]time.Time)
	outs := make(map[string]*strings.Builder)
	for dec.Next() {
		ev := dec.Event()
		pr.Output = append(pr.Output, ev)
		// Stream raw JSON if requested (optionally augmented with Orizon metadata)
		if r.opts.JSON && out != nil {
			// Marshal the original event again to ensure proper framing per line
			b, _ := json.Marshal(ev)
			_, _ = out.Write(b)
			_, _ = out.Write([]byte("\n"))
		} else if out != nil && ev.Action == "output" && ev.Output != "" {
			// In human-readable mode, forward test output lines with optional colors
			r.writeLine(out, ev)
		}
		// Aggregate counts on pass/fail/skip events and capture attempts
		switch ev.Action {
		case "pass":
			if ev.Test != "" {
				pr.Passed++
				var dur time.Duration
				if st, ok := starts[ev.Test]; ok {
					dur = ev.Time.Sub(st)
					if dur < 0 {
						dur = 0
					}
				}
				var msg string
				if b := outs[ev.Test]; b != nil {
					msg = b.String()
				}
				pr.Tests[ev.Test] = append(pr.Tests[ev.Test], TestAttempt{Outcome: "pass", Time: dur, Output: msg})
				delete(starts, ev.Test)
			} else if ev.Elapsed > 0 {
				pr.Duration += time.Duration(ev.Elapsed * float64(time.Second))
			}
		case "fail":
			if ev.Test != "" {
				pr.Failed++
				var dur time.Duration
				if st, ok := starts[ev.Test]; ok {
					dur = ev.Time.Sub(st)
					if dur < 0 {
						dur = 0
					}
				}
				var msg string
				if b := outs[ev.Test]; b != nil {
					msg = b.String()
				}
				pr.Tests[ev.Test] = append(pr.Tests[ev.Test], TestAttempt{Outcome: "fail", Time: dur, Output: msg})
				delete(starts, ev.Test)
			} else if ev.Elapsed > 0 {
				pr.Duration += time.Duration(ev.Elapsed * float64(time.Second))
			}
		case "skip":
			if ev.Test != "" {
				pr.Skipped++
				var dur time.Duration
				if st, ok := starts[ev.Test]; ok {
					dur = ev.Time.Sub(st)
					if dur < 0 {
						dur = 0
					}
				}
				var msg string
				if b := outs[ev.Test]; b != nil {
					msg = b.String()
				}
				pr.Tests[ev.Test] = append(pr.Tests[ev.Test], TestAttempt{Outcome: "skip", Time: dur, Output: msg})
				delete(starts, ev.Test)
			}
		case "run":
			if ev.Test != "" {
				starts[ev.Test] = ev.Time
				if outs[ev.Test] == nil {
					outs[ev.Test] = &strings.Builder{}
				}
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

// packageHasMatchingFile reports whether `go list -json` for the package includes
// any file path matching the given regex.
func (r *Runner) packageHasMatchingFile(ctx context.Context, pkg string, re *regexp.Regexp) (bool, error) {
    cmd := exec.CommandContext(ctx, "go", "list", "-json", pkg)
    b, err := cmd.Output()
    if err != nil {
        return false, err
    }
    // Lightweight search to avoid introducing JSON struct definitions: just scan lines
    lines := strings.Split(string(b), "\n")
    for _, l := range lines {
        l = strings.TrimSpace(l)
        // Look through GoFiles, TestGoFiles, XTestGoFiles and other path-like lines
        if strings.HasPrefix(l, "\"") && strings.HasSuffix(l, "\",") {
            // strip quotes and trailing comma
            path := strings.TrimSuffix(strings.TrimPrefix(l, "\""), "\",")
            if re.MatchString(path) {
                return true, nil
            }
        }
        if re.MatchString(l) {
            return true, nil
        }
    }
    return false, nil
}

// listTests prints the list of tests in a package without executing them.
func (r *Runner) listTests(ctx context.Context, pkg string, out io.Writer) {
    args := []string{"test", "-list", ".", pkg}
    cmd := exec.CommandContext(ctx, "go", args...)
    b, err := cmd.CombinedOutput()
    if err != nil {
        if out != nil {
            _, _ = out.Write(b)
        }
        return
    }
    if out != nil {
        _, _ = out.Write(b)
    }
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
		// When retries were used, show flaky recoveries per package
		if r.opts.Retries > 0 && len(p.Tests) > 0 {
			// Collect tests that had at least one fail and ended up passing
			recovered := make([]string, 0, 8)
			for name, ats := range p.Tests {
				sawFail := false
				final := ""
				for _, a := range ats {
					if a.Outcome == "fail" {
						sawFail = true
					}
					final = a.Outcome
				}
				if sawFail && final == "pass" {
					recovered = append(recovered, name)
				}
			}
			if len(recovered) > 0 {
				sort.Strings(recovered)
				msg := fmt.Sprintf("  flaky recovered: %s\n", strings.Join(recovered, ", "))
				_, _ = io.WriteString(w, yellow(msg))
			}
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

// writeJUnit emits a very small JUnit XML summary file for CI consumption.
func writeJUnit(path string, res Result) error {
	// Detailed per-test JUnit built from `go test -json` events we stored per package.
	type testcase struct {
		Name      string
		Classname string
		Time      string
		Failure   *struct {
			Message string `xml:"message,attr"`
		} `xml:"failure,omitempty"`
		Skipped   *struct{} `xml:"skipped,omitempty"`
		SystemOut string    `xml:"system-out,omitempty"`
	}
	type testsuite struct {
		XMLName   struct{}   `xml:"testsuite"`
		Name      string     `xml:"name,attr"`
		Tests     int        `xml:"tests,attr"`
		Failures  int        `xml:"failures,attr"`
		Skipped   int        `xml:"skipped,attr"`
		Time      string     `xml:"time,attr"`
		Testcases []testcase `xml:"testcase"`
	}

	var cases []testcase
	total := 0
	failures := 0
	skipped := 0
	for _, p := range res.Packages {
		if len(p.Tests) > 0 {
			for name, attempts := range p.Tests {
				if len(attempts) == 0 {
					continue
				}
				total++
				last := attempts[len(attempts)-1]
				tc := testcase{Name: name, Classname: p.Name, Time: fmt.Sprintf("%.3f", last.Time.Seconds())}
				if out := strings.TrimSpace(last.Output); out != "" {
					tc.SystemOut = out
				}
				switch last.Outcome {
				case "skip":
					skipped++
					tc.Skipped = &struct{}{}
				case "fail":
					failures++
					msg := tc.SystemOut
					if msg == "" {
						msg = "test failed"
					}
					tc.Failure = &struct {
						Message string `xml:"message,attr"`
					}{Message: msg}
				}
				cases = append(cases, tc)
			}
			continue
		}
		// Fallback: reconstruct from events when per-test attempts are not available
		type st struct{ start time.Time }
		starts := map[string]st{}
		outs := map[string]*strings.Builder{}
		for _, ev := range p.Output {
			switch ev.Action {
			case "run":
				if ev.Test != "" {
					starts[ev.Test] = st{start: ev.Time}
				}
			case "output":
				if ev.Test != "" {
					b := outs[ev.Test]
					if b == nil {
						b = &strings.Builder{}
						outs[ev.Test] = b
					}
					b.WriteString(ev.Output)
				}
			case "pass", "fail", "skip":
				if ev.Test == "" {
					continue
				}
				total++
				tc := testcase{Name: ev.Test, Classname: p.Name, Time: "0"}
				if s, ok := starts[ev.Test]; ok {
					dur := ev.Time.Sub(s.start)
					if dur < 0 {
						dur = 0
					}
					tc.Time = fmt.Sprintf("%.3f", dur.Seconds())
				}
				if outb := outs[ev.Test]; outb != nil {
					tc.SystemOut = outb.String()
				}
				switch ev.Action {
				case "skip":
					skipped++
					tc.Skipped = &struct{}{}
				case "fail":
					failures++
					msg := tc.SystemOut
					if msg == "" {
						msg = "test failed"
					}
					tc.Failure = &struct {
						Message string `xml:"message,attr"`
					}{Message: msg}
				}
				cases = append(cases, tc)
			}
		}
	}

	suite := testsuite{
		Name:      "orizon-tests",
		Tests:     total,
		Failures:  failures,
		Skipped:   skipped,
		Time:      fmt.Sprintf("%.2f", res.Duration.Seconds()),
		Testcases: cases,
	}
	b := &strings.Builder{}
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	fmt.Fprintf(b, "<testsuite name=\"%s\" tests=\"%d\" failures=\"%d\" skipped=\"%d\" time=\"%s\">\n", suite.Name, suite.Tests, suite.Failures, suite.Skipped, suite.Time)
	for _, t := range suite.Testcases {
		fmt.Fprintf(b, "  <testcase name=\"%s\" classname=\"%s\" time=\"%s\">\n", xmlEscape(t.Name), xmlEscape(t.Classname), t.Time)
		if t.Failure != nil {
			fmt.Fprintf(b, "    <failure message=\"%s\"/>\n", xmlEscape(t.Failure.Message))
		}
		if t.Skipped != nil {
			b.WriteString("    <skipped/>\n")
		}
		if t.SystemOut != "" {
			fmt.Fprintf(b, "    <system-out>%s</system-out>\n", xmlEscape(t.SystemOut))
		}
		b.WriteString("  </testcase>\n")
	}
	b.WriteString("</testsuite>\n")
	return os.WriteFile(path, []byte(b.String()), 0644)
}

// xmlEscape performs minimal XML escaping for text and attribute values.
func xmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}
