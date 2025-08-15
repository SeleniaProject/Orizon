package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"encoding/hex"

	"github.com/orizon-lang/orizon/internal/lexer"
	"github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/testrunner/fuzz"
)

func main() {
	var (
		in         string
		logPath    string
		lineNum    int
		out        string
		seed       int64
		lang       string
		dur        time.Duration
		targetKind string
	)
	flag.StringVar(&in, "in", "", "input file to reproduce")
	flag.StringVar(&logPath, "log", "", "optional crashes log (crashes.txt) to read from")
	flag.IntVar(&lineNum, "line", 0, "1-based line number in --log to reproduce (default=last non-empty line)")
	flag.StringVar(&out, "out", "", "optional minimized output path")
	flag.Int64Var(&seed, "seed", 0, "random seed (0=time)")
	flag.StringVar(&lang, "lang", "en", "message language (ja|en)")
	flag.DurationVar(&dur, "budget", 3*time.Second, "minimization time budget")
	flag.StringVar(&targetKind, "target", "parser", "target selector (noop|parser|lexer|astbridge|hir|astbridge-hir)")
	flag.Parse()

	L := getLocale(lang)
	var b []byte
	if logPath != "" {
		lb, err := os.ReadFile(logPath)
		if err != nil {
			fatal(L, "failed to read log: ", err)
		}
		lines := strings.Split(string(lb), "\n")
		// Pick last non-empty if lineNum==0, else 1-based index
		pick := -1
		if lineNum > 0 {
			if lineNum-1 < len(lines) {
				pick = lineNum - 1
			}
		} else {
			for i := len(lines) - 1; i >= 0; i-- {
				if strings.TrimSpace(lines[i]) != "" {
					pick = i
					break
				}
			}
		}
		if pick < 0 {
			fatal(L, "no usable lines in log")
		}
		s := strings.TrimSpace(lines[pick])
		// crash-line: ts \t 0xHEX \t msg
		parts := strings.SplitN(s, "\t", 3)
		if len(parts) >= 2 {
			h := parts[1]
			if strings.HasPrefix(h, "0x") || strings.HasPrefix(h, "0X") {
				h = h[2:]
			}
			if dec, er := hex.DecodeString(h); er == nil && len(dec) > 0 {
				b = dec
			}
		}
		if len(b) == 0 {
			fatal(L, "failed to decode crash line from log (expecting 0xHEX)")
		}
	} else {
		if in == "" {
			fatal(L, "--in or --log is required")
		}
		var err error
		b, err = os.ReadFile(in)
		if err != nil {
			fatal(L, "failed to read input: ", err)
		}
	}
	// Auto-detect crash-line or hex-encoded input and decode to raw bytes
	{
		s := strings.TrimSpace(string(b))
		if len(s) > 0 {
			// crash-line: ts \t 0xHEX \t msg
			if strings.Contains(s, "\t") {
				parts := strings.SplitN(s, "\t", 3)
				if len(parts) >= 2 {
					h := parts[1]
					if strings.HasPrefix(h, "0x") || strings.HasPrefix(h, "0X") {
						h = h[2:]
					}
					if dec, er := hex.DecodeString(h); er == nil && len(dec) > 0 {
						b = dec
					}
				}
			} else {
				// bare 0xHEX or HEX
				h := s
				if strings.HasPrefix(h, "0x") || strings.HasPrefix(h, "0X") {
					h = h[2:]
				}
				if dec, er := hex.DecodeString(h); er == nil && len(dec) > 0 {
					b = dec
				}
			}
		}
	}

	var target fuzz.Target
	switch strings.ToLower(targetKind) {
	case "noop":
		target = func(data []byte) error { _ = data; return nil }
	case "parser":
		target = func(data []byte) error {
			lx := lexer.NewWithFilename(string(data), "repro.oriz")
			ps := parser.NewParser(lx, "repro.oriz")
			_, errs := ps.Parse()
			if len(errs) > 0 {
				return fmt.Errorf("parse failed: %v", errs[0])
			}
			return nil
		}
	case "lexer":
		target = func(data []byte) error {
			lx := lexer.NewWithFilename(string(data), "repro_lex.oriz")
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
		target = func(data []byte) error {
			lx := lexer.NewWithFilename(string(data), "repro_ast.oriz")
			ps := parser.NewParser(lx, "repro_ast.oriz")
			prog, errs := ps.Parse()
			if len(errs) > 0 || prog == nil {
				return fmt.Errorf("parse failed: %v", errs)
			}
			if _, err := parser.OptimizeViaAstPipe(prog, "default"); err != nil {
				return fmt.Errorf("ast bridge failed: %v", err)
			}
			return nil
		}
	case "hir":
		target = func(data []byte) error {
			lx := lexer.NewWithFilename(string(data), "repro_hir.oriz")
			ps := parser.NewParser(lx, "repro_hir.oriz")
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
		target = func(data []byte) error {
			lx := lexer.NewWithFilename(string(data), "repro_ab_hir.oriz")
			ps := parser.NewParser(lx, "repro_ab_hir.oriz")
			prog, errs := ps.Parse()
			if prog == nil || len(errs) > 0 {
				return fmt.Errorf("parse failed: %v", errs)
			}
			bridged, err := parser.OptimizeViaAstPipe(prog, "default")
			if err != nil {
				return fmt.Errorf("ast bridge failed: %v", err)
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
		target = func(data []byte) error { _ = data; return nil }
	}

	if err := target(b); err != nil {
		fmt.Println(L.fail(err.Error()))
		if out != "" {
			min := fuzz.Minimize(seed, b, target, dur)
			if err := os.WriteFile(out, min, 0644); err != nil {
				fatal(L, "failed to write output: ", err)
			}
			fmt.Println(L.minDone(out))
		}
		return
	}
	fmt.Println(L.ok())
}

type locale struct {
	ok      func() string
	fail    func(msg string) string
	minDone func(path string) string
}

func getLocale(lang string) locale {
	switch lang {
	case "ja", "jp", "japanese":
		return locale{
			ok:      func() string { return "再現に失敗（問題なし）" },
			fail:    func(msg string) string { return "再現成功: " + msg },
			minDone: func(p string) string { return "最小化完了: " + p },
		}
	default:
		return locale{
			ok:      func() string { return "Reproduction failed (no issue)" },
			fail:    func(msg string) string { return "Reproduced: " + msg },
			minDone: func(p string) string { return "Minimized written: " + p },
		}
	}
}

func fatal(L locale, a ...any) {
	fmt.Fprintln(os.Stderr, a...)
	os.Exit(1)
}
