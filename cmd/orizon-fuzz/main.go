package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/orizon-lang/orizon/internal/lexer"
	"github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/testrunner/fuzz"
)

func main() {
	var (
		dur        time.Duration
		seed       int64
		max        int
		par        int
		corpusPath string
		corpusDir  string
		corpusOut  string
		outPath    string
		crashDir   string
		lang       string
		minimize   string
		targetKind string
		covOut     string
		covStats   bool
		per        time.Duration
		minOnCrash bool
		minDir     string
		minBudget  time.Duration
		saveSeed   string
		printStats bool
		jsonStats  string
		intensity  float64
		autotune   bool
		covMode    string
		maxExecs   uint64
	)

	flag.DurationVar(&dur, "duration", 5*time.Second, "fuzzing duration")
	flag.Int64Var(&seed, "seed", 0, "random seed (0=time)")
	flag.IntVar(&max, "max", 4096, "max input size")
	flag.IntVar(&par, "p", 1, "parallel workers")
	flag.StringVar(&corpusPath, "corpus", "", "optional corpus file (one input per line, hex or raw)")
	flag.StringVar(&corpusOut, "corpus-out", "", "directory to save interesting inputs (new coverage)")
	flag.StringVar(&corpusDir, "corpus-dir", "", "optional corpus directory (each file is an input)")
	flag.StringVar(&outPath, "out", "", "optional crashes output file")
	flag.StringVar(&crashDir, "crash-dir", "", "optional directory to save each crashing input as a file")
	flag.StringVar(&lang, "lang", "en", "message language (ja|en)")
	flag.StringVar(&minimize, "minimize", "", "minimize a crashing input from file to --out (skips fuzz loop)")
	flag.StringVar(&targetKind, "target", "noop", "target selector (noop|parser|lexer|astbridge|hir|astbridge-hir|custom)")
	flag.StringVar(&covOut, "covout", "", "write token-edge coverage to file during fuzzing")
	flag.BoolVar(&covStats, "covstats", false, "print coverage summary (unique token-edge count)")
	flag.DurationVar(&per, "per", 0, "per-input timeout (0=none)")
	flag.BoolVar(&minOnCrash, "min-on-crash", false, "minimize crashing inputs to --min-dir")
	flag.StringVar(&minDir, "min-dir", "", "directory to write minimized crashes (default=./crashes_min)")
	flag.DurationVar(&minBudget, "min-budget", 2*time.Second, "time budget for per-crash minimization")
	flag.StringVar(&saveSeed, "save-seed", "", "optional path to write the used random seed")
	flag.BoolVar(&printStats, "stats", false, "print execution/crash statistics at end")
	flag.StringVar(&jsonStats, "json-stats", "", "write execution/crash stats as JSON to file")
	flag.Float64Var(&intensity, "intensity", 0, "mutation intensity factor (1.0=default). 0=auto")
	flag.BoolVar(&autotune, "autotune", false, "enable adaptive mutation intensity")
	flag.StringVar(&covMode, "cov-mode", "weighted", "coverage mode (edge|weighted|trigram|both)")
	flag.Uint64Var(&maxExecs, "max-execs", 0, "stop after this many executions (0=unlimited)")
	flag.Parse()

	L := getLocale(lang)

	// Determine final seed deterministically here so it is known to the user and reproducible.
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	if saveSeed != "" {
		_ = os.WriteFile(saveSeed, []byte(fmt.Sprintf("%d\n", seed)), 0o644)
	}

	// choose target.
	var target fuzz.Target

	switch strings.ToLower(targetKind) {
	case "noop":
		target = func(data []byte) error {
			_ = data
			return nil
		}
	case "parser":
		target = func(data []byte) error {
			// Convert input to string (UTF-8 assumption for now).
			src := string(data)
			// Use internal lexer and parser to attempt a parse.
			lx := lexer.NewWithFilename(src, "fuzz_input.oriz")
			ps := parser.NewParser(lx, "fuzz_input.oriz")

			_, errs := ps.Parse()
			if len(errs) > 0 {
				return fmt.Errorf("parse failed: %w", errs[0])
			}

			return nil
		}
	case "parser-lax":
		// Parse but do not treat syntax errors as crashes; only panics propagate.
		target = func(data []byte) error {
			src := string(data)
			lx := lexer.NewWithFilename(src, "fuzz_input_lax.oriz")
			ps := parser.NewParser(lx, "fuzz_input_lax.oriz")
			_, _ = ps.Parse()

			return nil
		}
	case "lexer":
		// Scan all tokens and report if TokenError is encountered.
		target = func(data []byte) error {
			lx := lexer.NewWithFilename(string(data), "fuzz_lex.oriz")

			for {
				tok := lx.NextToken()
				if tok.Type == lexer.TokenError {
					return fmt.Errorf("lexer error token: %q", tok.Literal)
				}

				if tok.Type == lexer.TokenEOF {
					break
				}
			}

			return nil
		}
	case "astbridge":
		// Parse successfully, then run AST optimization bridge round-trip.
		target = func(data []byte) error {
			src := string(data)
			lx := lexer.NewWithFilename(src, "fuzz_astbridge.oriz")
			ps := parser.NewParser(lx, "fuzz_astbridge.oriz")

			prog, errs := ps.Parse()
			if len(errs) > 0 || prog == nil {
				return fmt.Errorf("parse failed: %v", errs)
			}

			if _, err := parser.OptimizeViaAstPipe(prog, "default"); err != nil {
				return fmt.Errorf("ast bridge failed: %w", err)
			}

			return nil
		}
	case "hir":
		// Parse and then validate HIR.
		target = func(data []byte) error {
			src := string(data)
			lx := lexer.NewWithFilename(src, "fuzz_hir.oriz")
			ps := parser.NewParser(lx, "fuzz_hir.oriz")

			prog, errs := ps.Parse()
			if prog == nil || len(errs) > 0 {
				return fmt.Errorf("parse failed: %v", errs)
			}

			mod, terrs := parser.TransformASTToHIR(prog)
			if mod == nil || len(terrs) > 0 {
				return fmt.Errorf("transform failed: %v", terrs)
			}

			if verrs, _ := parser.ValidateHIR(mod); len(verrs) > 0 {
				return fmt.Errorf("hir validation failed: %d errors", len(verrs))
			}

			return nil
		}
	case "astbridge-hir":
		// Parse -> AST bridge optimize -> back to parser -> HIR transform -> HIR validate.
		target = func(data []byte) error {
			src := string(data)
			lx := lexer.NewWithFilename(src, "fuzz_ab_hir.oriz")
			ps := parser.NewParser(lx, "fuzz_ab_hir.oriz")

			prog, errs := ps.Parse()
			if prog == nil || len(errs) > 0 {
				return fmt.Errorf("parse failed: %v", errs)
			}

			bridged, err := parser.OptimizeViaAstPipe(prog, "default")
			if err != nil {
				return fmt.Errorf("ast bridge failed: %w", err)
			}

			mod, terrs := parser.TransformASTToHIR(bridged)
			if mod == nil || len(terrs) > 0 {
				return fmt.Errorf("transform failed: %v", terrs)
			}

			if verrs, _ := parser.ValidateHIR(mod); len(verrs) > 0 {
				return fmt.Errorf("hir validation failed: %d errors", len(verrs))
			}

			return nil
		}
	default:
		target = func(data []byte) error {
			_ = data
			return nil
		}
	}

	if minimize != "" {
		if outPath == "" {
			fatal(L, "--minimize requires --out destination")
		}

		b, err := os.ReadFile(minimize)
		if err != nil {
			fatal(L, "failed to read input: ", err)
		}

		min := fuzz.Minimize(seed, b, target, dur)
		if err := os.WriteFile(outPath, min, 0o644); err != nil {
			fatal(L, "failed to write output: ", err)
		}

		println(L.done())

		return
	}

	var corpus []fuzz.CorpusEntry

	if corpusPath != "" {
		b2, err2 := os.ReadFile(corpusPath)
		if err2 != nil {
			fatal(L, "failed to read corpus: ", err2)
		}

		for _, line := range strings.Split(string(b2), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// Try to decode hex input; fallback to raw on failure.
			l := line
			if strings.HasPrefix(l, "0x") || strings.HasPrefix(l, "0X") {
				l = l[2:]
			}

			if decoded, errh := hex.DecodeString(l); errh == nil && len(decoded) > 0 {
				corpus = append(corpus, decoded)
			} else {
				corpus = append(corpus, []byte(line))
			}
		}
	}
	// Load corpus from directory if provided (one file per input).
	if corpusDir != "" {
		entries, err := os.ReadDir(corpusDir)
		if err != nil {
			fatal(L, "failed to read corpus dir: ", err)
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}

			b, err := os.ReadFile(filepath.Join(corpusDir, e.Name()))
			if err == nil && len(b) > 0 {
				corpus = append(corpus, b)
			}
		}
	}

	var w io.Writer

	if outPath != "" {
		f, err := os.Create(outPath)
		if err != nil {
			fatal(L, "failed to open output: ", err)
		}

		defer f.Close()
		w = f
	}

	// Optional coverage collection wrapper (thread-safe log + unique set).
	wrapped := target

	var covMu sync.Mutex

	covSeen := make(map[uint64]struct{})

	if covOut != "" || covStats || corpusOut != "" {
		var logf io.Writer

		if covOut != "" {
			f, err := os.Create(covOut)
			if err != nil {
				fatal(L, "failed to open covout: ", err)
			}

			defer f.Close()
			logf = f
		}

		wrapped = func(data []byte) error {
			edges := fuzz.ComputeCoverage(covMode, string(data))

			if covOut != "" {
				covMu.Lock()
				for _, e := range edges {
					// write as hex per line.
					fmt.Fprintf(logf, "%016x\n", e)
				}
				covMu.Unlock()
			}

			if covStats {
				covMu.Lock()
				for _, e := range edges {
					covSeen[e] = struct{}{}
				}
				covMu.Unlock()
			}
			// Save interesting inputs based on new edge discovery.
			if corpusOut != "" {
				covMu.Lock()
				base := len(covSeen)

				for _, e := range edges {
					covSeen[e] = struct{}{}
				}

				grew := len(covSeen) > base
				covMu.Unlock()

				if grew {
					// Deduplicate by input hash and persist once.
					sum := sha256.Sum256(data)
					hexname := hex.EncodeToString(sum[:]) + ".bin"
					_ = os.MkdirAll(corpusOut, 0o755)
					path := filepath.Join(corpusOut, hexname)

					if _, err := os.Stat(path); err != nil {
						_ = os.WriteFile(path, data, 0o644)
					}
				}
			} else if covStats {
				covMu.Lock()
				for _, e := range edges {
					covSeen[e] = struct{}{}
				}
				covMu.Unlock()
			}

			return target(data)
		}
	}

	// Optional per-input timeout and on-crash minimization wrapper.
	effective := wrapped

	if per > 0 || minOnCrash {
		if minDir == "" {
			minDir = "crashes_min"
		}

		baseTarget := target // use raw target for minimization to avoid wrapper side effects
		effective = func(data []byte) error {
			// Apply per-input timeout if requested.
			var err error

			if per > 0 {
				ch := make(chan error, 1)
				go func() { ch <- wrapped(data) }()
				select {
				case e := <-ch:
					err = e
				case <-time.After(per):
					err = fmt.Errorf("per-input timeout")
				}
			} else {
				err = wrapped(data)
			}
			// On crash, optionally minimize and persist minimized input.
			if err != nil && minOnCrash {
				_ = os.MkdirAll(minDir, 0o755)
				min := fuzz.Minimize(seed, data, baseTarget, minBudget)
				name := time.Now().Format("20060102_150405.000000000") + ".min"
				_ = os.WriteFile(filepath.Join(minDir, name), min, 0o644)
			}

			return err
		}
	}

	opts := fuzz.Options{Duration: dur, Seed: seed, MaxInput: max, Concurrency: par}
	if per > 0 {
		opts.InputBudget = per
	}

	if intensity > 0 {
		opts.MutationIntensity = intensity
	}

	opts.AutoTune = autotune
	if maxExecs > 0 {
		opts.MaxExecs = maxExecs
	}
	// If a crash directory is specified, wrap crashes writer to emit per-file cases too.
	var crashWriter io.Writer = w

	if crashDir != "" {
		_ = os.MkdirAll(crashDir, 0o755)
		// wrap: on every write of a crash line, also save the input bytes to a timestamped file.
		crashWriter = &crashFileWriter{base: w, dir: crashDir}
	}

	start := time.Now()
	stats := fuzz.RunWithStats(opts, corpus, effective, nil, crashWriter)
	elapsed := time.Since(start)

	if covStats {
		covMu.Lock()
		n := len(covSeen)
		covMu.Unlock()
		fmt.Println(L.cov(n))
	}

	// Optional stats printing / JSON output
	if printStats {
		execsPerSec := float64(stats.Executions) / (elapsed.Seconds())
		fmt.Printf("executions=%d crashes=%d duration=%s execs_per_sec=%.2f\n", stats.Executions, stats.Crashes, elapsed.Truncate(time.Millisecond), execsPerSec)
	}

	if jsonStats != "" {
		_ = os.WriteFile(jsonStats, []byte(fmt.Sprintf("{\"executions\":%d,\"crashes\":%d,\"duration_ms\":%d}\n", stats.Executions, stats.Crashes, elapsed.Milliseconds())), 0o644)
	}

	println(L.done())
}

type locale struct {
	done func() string
	cov  func(n int) string
}

// crashFileWriter writes crash lines to an underlying writer and also extracts.
// the crashing input to store as an individual file in a directory.
type crashFileWriter struct {
	base io.Writer
	dir  string
	buf  []byte
}

func (w *crashFileWriter) Write(p []byte) (int, error) {
	// Pass-through.
	if w.base != nil {
		if _, err := w.base.Write(p); err != nil {
			// ignore pass-through error for extraction.
		}
	}
	// Buffer until newline.
	w.buf = append(w.buf, p...)
	// process complete lines.
	for {
		idx := -1

		for i := 0; i < len(w.buf); i++ {
			if w.buf[i] == '\n' {
				idx = i

				break
			}
		}

		if idx == -1 {
			break
		}

		line := w.buf[:idx]

		if len(w.buf) > idx+1 {
			w.buf = w.buf[idx+1:]
		} else {
			w.buf = w.buf[:0]
		}
		// Attempt to split the assembled line by tabs.
		parts := strings.SplitN(string(line), "\t", 3)
		if len(parts) >= 2 {
			raw := parts[1]
			// We expect hex-encoded input prefixed with 0x from the crash writer.
			if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
				raw = raw[2:]
			}

			if dec, err := hex.DecodeString(raw); err == nil && len(dec) > 0 {
				name := time.Now().Format("20060102_150405.000000000") + ".crash"
				_ = os.WriteFile(filepath.Join(w.dir, name), dec, 0o644)
			}
		}
	}

	return len(p), nil
}

func getLocale(lang string) locale {
	switch strings.ToLower(lang) {
	case "ja", "jp", "japanese":
		return locale{
			done: func() string { return "ファズ終了" },
			cov:  func(n int) string { return fmt.Sprintf("カバレッジユニークエッジ数: %d", n) },
		}
	default:
		return locale{
			done: func() string { return "Fuzzing finished" },
			cov:  func(n int) string { return fmt.Sprintf("Coverage unique edges: %d", n) },
		}
	}
}

func fatal(L locale, a ...any) {
	fmt.Fprintln(os.Stderr, a...)
	os.Exit(1)
}
