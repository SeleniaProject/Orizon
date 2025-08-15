package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/orizon-lang/orizon/internal/lexer"
	"github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/testrunner/fuzz"
)

func main() {
	var (
		in         string
		out        string
		seed       int64
		lang       string
		dur        time.Duration
		targetKind string
	)
	flag.StringVar(&in, "in", "", "input file to reproduce")
	flag.StringVar(&out, "out", "", "optional minimized output path")
	flag.Int64Var(&seed, "seed", 0, "random seed (0=time)")
	flag.StringVar(&lang, "lang", "en", "message language (ja|en)")
	flag.DurationVar(&dur, "budget", 3*time.Second, "minimization time budget")
	flag.StringVar(&targetKind, "target", "parser", "target selector (noop|parser|lexer|astbridge)")
	flag.Parse()

	L := getLocale(lang)
	if in == "" {
		fatal(L, "--in is required")
	}
	b, err := os.ReadFile(in)
	if err != nil {
		fatal(L, "failed to read input: ", err)
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
