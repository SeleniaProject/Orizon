package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
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
		corpusOut  string
		outPath    string
		lang       string
		minimize   string
		targetKind string
		covOut     string
		covStats   bool
	)
	flag.DurationVar(&dur, "duration", 5*time.Second, "fuzzing duration")
	flag.Int64Var(&seed, "seed", 0, "random seed (0=time)")
	flag.IntVar(&max, "max", 4096, "max input size")
	flag.IntVar(&par, "p", 1, "parallel workers")
	flag.StringVar(&corpusPath, "corpus", "", "optional corpus file (one input per line, hex or raw)")
	flag.StringVar(&corpusOut, "corpus-out", "", "directory to save interesting inputs (new coverage)")
	flag.StringVar(&outPath, "out", "", "optional crashes output file")
	flag.StringVar(&lang, "lang", "en", "message language (ja|en)")
	flag.StringVar(&minimize, "minimize", "", "minimize a crashing input from file to --out (skips fuzz loop)")
	flag.StringVar(&targetKind, "target", "noop", "target selector (noop|parser|lexer|astbridge|custom)")
	flag.StringVar(&covOut, "covout", "", "write token-edge coverage to file during fuzzing")
	flag.BoolVar(&covStats, "covstats", false, "print coverage summary (unique token-edge count)")
	flag.Parse()

	L := getLocale(lang)

	// choose target
	var target fuzz.Target
	switch strings.ToLower(targetKind) {
	case "noop":
		target = func(data []byte) error { _ = data; return nil }
	case "parser":
		target = func(data []byte) error {
			// Convert input to string (UTF-8 assumption for now)
			src := string(data)
			// Use internal lexer and parser to attempt a parse
			lx := lexer.NewWithFilename(src, "fuzz_input.oriz")
			ps := parser.NewParser(lx, "fuzz_input.oriz")
			_, errs := ps.Parse()
			if len(errs) > 0 {
				return fmt.Errorf("parse failed: %v", errs[0])
			}
			return nil
		}
	case "lexer":
		// Scan all tokens and report if TokenError is encountered
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
		// Parse successfully, then run AST optimization bridge round-trip
		target = func(data []byte) error {
			src := string(data)
			lx := lexer.NewWithFilename(src, "fuzz_astbridge.oriz")
			ps := parser.NewParser(lx, "fuzz_astbridge.oriz")
			prog, errs := ps.Parse()
			if len(errs) > 0 || prog == nil {
				return fmt.Errorf("parse failed: %v", errs)
			}
			if _, err := parser.OptimizeViaAstPipe(prog, "default"); err != nil {
				return fmt.Errorf("ast bridge failed: %v", err)
			}
			return nil
		}
	default:
		target = func(data []byte) error { _ = data; return nil }
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
		if err := os.WriteFile(outPath, min, 0644); err != nil {
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
			// Try to decode hex input; fallback to raw on failure
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

	var w io.Writer
	if outPath != "" {
		f, err := os.Create(outPath)
		if err != nil {
			fatal(L, "failed to open output: ", err)
		}
		defer f.Close()
		w = f
	}

	// Optional coverage collection wrapper (thread-safe log + unique set)
	wrapped := target
	var covMu sync.Mutex
	var covSeen = make(map[uint64]struct{})
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
			edges := fuzz.TokenEdgeCoverage(string(data))
			if covOut != "" {
				covMu.Lock()
				for _, e := range edges {
					// write as hex per line
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
			// Save interesting inputs based on new edge discovery
			if corpusOut != "" {
				covMu.Lock()
				base := len(covSeen)
				for _, e := range edges {
					covSeen[e] = struct{}{}
				}
				grew := len(covSeen) > base
				covMu.Unlock()
				if grew {
					// Derive filename from edge hash and time
					name := time.Now().Format("20060102_150405.000000000") + ".bin"
					_ = os.MkdirAll(corpusOut, 0o755)
					_ = os.WriteFile(corpusOut+string(os.PathSeparator)+name, data, 0o644)
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

	fuzz.Run(fuzz.Options{Duration: dur, Seed: seed, MaxInput: max, Concurrency: par}, corpus, wrapped, nil, w)
	if covStats {
		covMu.Lock()
		n := len(covSeen)
		covMu.Unlock()
		fmt.Println(L.cov(n))
	}
	println(L.done())
}

type locale struct {
	done func() string
	cov  func(n int) string
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
