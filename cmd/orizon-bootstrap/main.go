package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/orizon-lang/orizon/internal/astbridge"
	"github.com/orizon-lang/orizon/internal/codegen"
	"github.com/orizon-lang/orizon/internal/hir"
	"github.com/orizon-lang/orizon/internal/lexer"
	p "github.com/orizon-lang/orizon/internal/parser"
)

func main() {
	var (
		outDir       = flag.String("out-dir", "artifacts/selfhost", "output directory for snapshots")
		emitMIR      = flag.Bool("emit-mir", true, "emit MIR text snapshots")
		emitLIR      = flag.Bool("emit-lir", true, "emit LIR text snapshots")
		emitX64      = flag.Bool("emit-x64", true, "emit diagnostic x64 assembly snapshots")
		goldenDir    = flag.String("golden-dir", "", "golden snapshots directory (optional)")
		updateGolden = flag.Bool("update-golden", false, "update golden snapshots when diffs are found")
		expandMacros = flag.Bool("expand-macros", false, "expand macros before bridging (experimental)")
	)

	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: orizon-bootstrap [--out-dir DIR] [--golden-dir DIR] [--update-golden] <file.oriz|dir> [more files/dirs...]")
		os.Exit(2)
	}

	// Collect input files from all args.
	var files []string

	for _, in := range args {
		st, err := os.Stat(in)
		if err != nil {
			fmt.Fprintf(os.Stderr, "stat failed for %s: %v\n", in, err)
			os.Exit(1)
		}

		if st.IsDir() {
			filepath.WalkDir(in, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if d.IsDir() {
					return nil
				}

				if strings.HasSuffix(strings.ToLower(d.Name()), ".oriz") {
					files = append(files, path)
				}

				return nil
			})
		} else {
			// Accept only .oriz files explicitly
			if strings.HasSuffix(strings.ToLower(in), ".oriz") {
				files = append(files, in)
			}
		}
	}

	if len(files) == 0 {
		fmt.Println("no .oriz files found")
		os.Exit(1)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir out-dir failed: %v\n", err)
		os.Exit(1)
	}

	var failures int

	for _, f := range files {
		if err := processFile(f, *outDir, *emitMIR, *emitLIR, *emitX64, *goldenDir, *updateGolden, *expandMacros); err != nil {
			fmt.Fprintf(os.Stderr, "[FAIL] %s: %v\n", f, err)

			failures++
		} else {
			fmt.Printf("[OK] %s\n", f)
		}
	}

	if failures > 0 {
		fmt.Fprintf(os.Stderr, "bootstrap completed with %d failure(s)\n", failures)
		os.Exit(1)
	}
}

func processFile(filename, outDir string, doMIR, doLIR, doX64 bool, goldenDir string, update bool, doExpand bool) error {
	src, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// Parse.
	pr := p.NewParser(lexer.NewWithFilename(string(src), filename), filename)

	program, parseErrors := pr.Parse()
	if len(parseErrors) > 0 {
		for _, e := range parseErrors {
			fmt.Fprintf(os.Stderr, "parse error: %v\n", e)
		}

		return fmt.Errorf("parse failed: %d error(s)", len(parseErrors))
	}
	// Optional: Macro expansion at parser AST level before bridging.
	if doExpand {
		program = expandMacros(program)
	}

	// AST->HIR.
	astProg, err := astbridge.FromParserProgram(program)
	if err != nil {
		return fmt.Errorf("ast bridge failed: %w", err)
	}

	conv := hir.NewASTToHIRConverter()
	hirProg, _ := conv.ConvertProgram(astProg)
	// Lower.
	mirMod := codegen.LowerToMIR(hirProg)
	lirMod := codegen.SelectToLIR(mirMod)

	// Build a stable, path-prefixed base name to avoid collisions when the same.
	// file name exists in different directories (e.g., examples/simple.oriz vs bootstrap_samples/simple.oriz)
	rel := filename

	if wd, wderr := os.Getwd(); wderr == nil {
		if r, rerr := filepath.Rel(wd, filename); rerr == nil {
			rel = r
		}
	}
	// Drop extension and normalize separators.
	noext := strings.TrimSuffix(rel, filepath.Ext(rel))
	norm := filepath.ToSlash(noext)
	base := strings.ReplaceAll(norm, "/", ".")
	// Emit.
	type out struct{ name, content string }

	outs := []out{}
	if doMIR {
		outs = append(outs, out{base + ".mir.txt", mirMod.String()})
	}

	if doLIR {
		outs = append(outs, out{base + ".lir.txt", lirMod.String()})
	}

	if doX64 {
		outs = append(outs, out{base + ".x64.asm", codegen.EmitX64(lirMod)})
	}

	for _, o := range outs {
		pth := filepath.Join(outDir, o.name)
		if err := os.WriteFile(pth, []byte(o.content), 0o644); err != nil {
			return fmt.Errorf("write %s failed: %w", pth, err)
		}
		// Golden compare if provided.
		if goldenDir != "" {
			gp := filepath.Join(goldenDir, o.name)

			gb, gerr := os.ReadFile(gp)
			if gerr != nil {
				if update { // create
					if err := os.MkdirAll(goldenDir, 0o755); err != nil {
						return err
					}

					if err := os.WriteFile(gp, []byte(o.content), 0o644); err != nil {
						return err
					}
				} else {
					return fmt.Errorf("missing golden: %s", gp)
				}
			} else if string(gb) != o.content {
				if update {
					if err := os.WriteFile(gp, []byte(o.content), 0o644); err != nil {
						return err
					}
				} else {
					return fmt.Errorf("golden mismatch: %s", gp)
				}
			}
		}
	}

	return nil
}

// expandMacros registers all macro definitions into an engine and expands macro invocations.
// within functions and top-level statements, returning a transformed program.
func expandMacros(prog *p.Program) *p.Program {
	if prog == nil {
		return prog
	}

	engine := p.NewMacroEngine()

	// 1) Register macros and collect non-macro declarations.
	var decls []p.Declaration

	for _, d := range prog.Declarations {
		switch md := d.(type) {
		case *p.MacroDefinition:
			_ = engine.RegisterMacro(md) // ignore duplicates for now
		default:
			decls = append(decls, d)
		}
	}

	// 2) Walk remaining declarations and expand bodies.
	outDecls := make([]p.Declaration, 0, len(decls))

	for _, d := range decls {
		switch fn := d.(type) {
		case *p.FunctionDeclaration:
			if fn.Body != nil {
				fn.Body = expandBlock(fn.Body, engine)
			}

			outDecls = append(outDecls, fn)
		case *p.VariableDeclaration:
			if fn.Initializer != nil {
				fn.Initializer = expandExpr(fn.Initializer, engine)
			}

			outDecls = append(outDecls, fn)
		case *p.ExpressionStatement:
			// Top-level expression statements are allowed.
			stmts := expandStmt(fn, engine)
			// Flatten back to declarations (keep ExpressionStatement as declaration like before).
			for _, s := range stmts {
				if es, ok := s.(*p.ExpressionStatement); ok {
					outDecls = append(outDecls, es)
				}
			}
		default:
			outDecls = append(outDecls, d)
		}
	}

	prog.Declarations = outDecls

	return prog
}

func expandBlock(b *p.BlockStatement, engine *p.MacroEngine) *p.BlockStatement {
	if b == nil {
		return b
	}

	out := &p.BlockStatement{Span: b.Span, Statements: make([]p.Statement, 0, len(b.Statements))}
	for _, s := range b.Statements {
		out.Statements = append(out.Statements, expandStmt(s, engine)...)
	}

	return out
}

func expandStmt(s p.Statement, engine *p.MacroEngine) []p.Statement {
	switch n := s.(type) {
	case *p.BlockStatement:
		return []p.Statement{expandBlock(n, engine)}
	case *p.ExpressionStatement:
		// If it's a macro invocation as a statement, expand to statements.
		if mi, ok := n.Expression.(*p.MacroInvocation); ok {
			expanded, err := engine.ExpandMacro(mi)
			if err == nil && len(expanded) > 0 {
				// Recursively expand the result in case nested invocations exist.
				var flat []p.Statement
				for _, es := range expanded {
					flat = append(flat, expandStmt(es, engine)...)
				}

				return flat
			}
		}
		// Otherwise, expand expression recursively.
		n.Expression = expandExpr(n.Expression, engine)

		return []p.Statement{n}
	case *p.ReturnStatement:
		if n.Value != nil {
			n.Value = expandExpr(n.Value, engine)
		}

		return []p.Statement{n}
	case *p.IfStatement:
		n.Condition = expandExpr(n.Condition, engine)
		if n.ThenStmt != nil {
			thenFlat := expandStmt(n.ThenStmt, engine)
			// Rewrap into a block to preserve a single Statement.
			n.ThenStmt = &p.BlockStatement{Statements: thenFlat}
		}

		if n.ElseStmt != nil {
			elseFlat := expandStmt(n.ElseStmt, engine)
			n.ElseStmt = &p.BlockStatement{Statements: elseFlat}
		}

		return []p.Statement{n}
	case *p.WhileStatement:
		n.Condition = expandExpr(n.Condition, engine)
		if n.Body != nil {
			bodyFlat := expandStmt(n.Body, engine)
			n.Body = &p.BlockStatement{Statements: bodyFlat}
		}

		return []p.Statement{n}
	case *p.VariableDeclaration:
		if n.Initializer != nil {
			n.Initializer = expandExpr(n.Initializer, engine)
		}

		return []p.Statement{n}
	default:
		return []p.Statement{s}
	}
}

func expandExpr(e p.Expression, engine *p.MacroEngine) p.Expression {
	switch x := e.(type) {
	case *p.MacroInvocation:
		// Try to expand to a single expression via expansion result.
		if stmts, err := engine.ExpandMacro(x); err == nil && len(stmts) == 1 {
			if es, ok := stmts[0].(*p.ExpressionStatement); ok && es.Expression != nil {
				// Further expand nested expressions.
				return expandExpr(es.Expression, engine)
			}
		}

		return e
	case *p.BinaryExpression:
		x.Left = expandExpr(x.Left, engine)
		x.Right = expandExpr(x.Right, engine)

		return x
	case *p.UnaryExpression:
		x.Operand = expandExpr(x.Operand, engine)

		return x
	case *p.CallExpression:
		x.Function = expandExpr(x.Function, engine)
		for i := range x.Arguments {
			x.Arguments[i] = expandExpr(x.Arguments[i], engine)
		}

		return x
	default:
		return e
	}
}
