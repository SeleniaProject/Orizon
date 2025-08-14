// Package main provides the entry point for the Orizon compiler.
// Phase 1.1.1: Unicode対応字句解析器の基本実装
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/orizon-lang/orizon/internal/astbridge"
	"github.com/orizon-lang/orizon/internal/debug"
	"github.com/orizon-lang/orizon/internal/hir"
	"github.com/orizon-lang/orizon/internal/lexer"
	p "github.com/orizon-lang/orizon/internal/parser"
)

var (
	version = "0.1.0-alpha"
	commit  = "dev"
)

func main() {
	var (
		showVersion = flag.Bool("version", false, "show version information")
		showHelp    = flag.Bool("help", false, "show help information")
		debugLexer  = flag.Bool("debug-lexer", false, "enable lexer debug output")
		doParse     = flag.Bool("parse", false, "parse the input and print AST (parser AST)")
		optLevel    = flag.String("optimize-level", "", "optimize via ast pipeline: none|basic|default|aggressive")
		emitDebug   = flag.Bool("emit-debug", false, "emit debug info JSON and DWARF sections (stdout)")
		emitSrcMap  = flag.Bool("emit-sourcemap", false, "emit source map JSON (stdout)")
		debugOut    = flag.String("debug-out", "", "write debug JSON to file instead of stdout")
		smOut       = flag.String("sourcemap-out", "", "write source map JSON to file instead of stdout")
		dwarfDir    = flag.String("dwarf-out-dir", "", "write DWARF sections to directory as raw binary files")
		outELF      = flag.String("emit-elf", "", "write minimal ELF64 object bundling DWARF to the given path")
		outCOFF     = flag.String("emit-coff", "", "write minimal COFF object bundling DWARF to the given path")
		outMachO    = flag.String("emit-macho", "", "write minimal Mach-O object bundling DWARF to the given path")
	)

	flag.Parse()

	if *showVersion {
		fmt.Printf("Orizon Compiler v%s (%s)\n", version, commit)
		fmt.Println("The Future of Systems Programming")
		return
	}

	if *showHelp {
		showUsage()
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Error: No input file specified")
		showUsage()
		os.Exit(1)
	}

	inputFile := args[0]
	if err := compileFile(inputFile, *debugLexer, *doParse, *optLevel, *emitDebug, *emitSrcMap, *debugOut, *smOut, *dwarfDir, *outELF, *outCOFF, *outMachO); err != nil {
		log.Fatalf("Compilation failed: %v", err)
	}
}

func showUsage() {
	fmt.Println("Orizon Compiler - The Future of Systems Programming")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("    orizon-compiler [OPTIONS] <INPUT_FILE>")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("    --version         Show version information")
	fmt.Println("    --help           Show this help message")
	fmt.Println("    --debug-lexer    Enable lexer debug output")
	fmt.Println("    --optimize-level Level: none|basic|default|aggressive")
	fmt.Println("    --emit-debug     Emit debug JSON and DWARF sections")
	fmt.Println("    --emit-sourcemap Emit source map JSON")
	fmt.Println("    --debug-out      Write debug JSON to file")
	fmt.Println("    --sourcemap-out  Write source map JSON to file")
	fmt.Println("    --dwarf-out-dir  Write DWARF sections to directory")
	fmt.Println("    --emit-elf       Write minimal ELF64 object bundling DWARF")
	fmt.Println("    --emit-coff      Write minimal COFF (AMD64) object bundling DWARF")
	fmt.Println("    --emit-macho     Write minimal Mach-O (x86_64) object bundling DWARF")
	fmt.Println("    env ORIZON_DEBUG_OBJ_OUT, ORIZON_DEBUG_OBJ_FORMAT={auto|elf|coff|macho} can auto-emit when not specified")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("    orizon-compiler hello.oriz")
	fmt.Println("    orizon-compiler --emit-debug hello.oriz")
	fmt.Println("    orizon-compiler --emit-debug --debug-out dbg.json --dwarf-out-dir out/dwarf hello.oriz")
}

func compileFile(filename string, debugLexer bool, doParse bool, optLevel string, emitDebug bool, emitSrcMap bool, debugOut string, smOut string, dwarfDir string, outELF string, outCOFF string, outMachO string) error {
	// ファイル存在チェック
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filename)
	}

	// ファイル読み込み
	source, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	fmt.Printf("🔥 Compiling %s...\n", filepath.Base(filename))

	// Phase 1.1.2: インクリメンタル対応字句解析器実行
	l := lexer.NewWithFilename(string(source), filename)

	if debugLexer {
		fmt.Println("🔍 Lexer Debug Output:")
		fmt.Println(strings.Repeat("=", 50))

		for {
			token := l.NextToken()
			fmt.Printf("Token: %-15s | Value: %-20s | Position: %d:%d\n",
				token.Type,
				token.Literal,
				token.Line,
				token.Column)

			if token.Type == lexer.TokenEOF {
				break
			}
		}
		fmt.Println(strings.Repeat("=", 50))
	} else if !doParse && optLevel == "" {
		// 通常のコンパイル（現在は字句解析のみ）
		tokenCount := 0
		for {
			token := l.NextToken()
			tokenCount++

			if token.Type == lexer.TokenEOF {
				break
			}

			// エラートークンのチェック
			if token.Type == lexer.TokenError {
				return fmt.Errorf("lexer error at %d:%d: %s",
					token.Line, token.Column, token.Literal)
			}
		}

		fmt.Printf("✅ Lexing completed: %d tokens processed\n", tokenCount)
	} else {
		// Parse phase (optional) and optional optimization via AST bridge
		pr := p.NewParser(lexer.NewWithFilename(string(source), filename), filename)
		program, parseErrors := pr.Parse()
		if len(parseErrors) > 0 {
			for _, e := range parseErrors {
				fmt.Fprintf(os.Stderr, "Parse error: %v\n", e)
			}
			return fmt.Errorf("parse failed with %d error(s)", len(parseErrors))
		}

		if doParse && optLevel == "" {
			// Print parser AST
			fmt.Println("📦 Parsed AST (parser):")
			fmt.Println(p.PrettyPrint(program))
		}

		if optLevel != "" {
			optimized, err := p.OptimizeViaAstPipe(program, strings.ToLower(optLevel))
			if err != nil {
				return fmt.Errorf("optimization failed: %w", err)
			}
			fmt.Printf("✨ Optimized via AST pipeline (level=%s)\n", strings.ToLower(optLevel))
			fmt.Println(p.PrettyPrint(optimized))
			program = optimized
		}

		// Convert parser AST -> internal AST -> HIR (once) for debug artifacts
		if emitDebug || emitSrcMap {
			astProg, err := astbridge.FromParserProgram(program)
			if err != nil {
				return fmt.Errorf("ast bridge failed: %w", err)
			}
			conv := hir.NewASTToHIRConverter()
			hirProg, _ := conv.ConvertProgram(astProg)

			if emitDebug {
				em := debug.NewEmitter()
				dbg, err := em.Emit(hirProg)
				if err != nil {
					return fmt.Errorf("emit debug failed: %w", err)
				}
				js, err := debug.Serialize(dbg)
				if err != nil {
					return fmt.Errorf("serialize debug failed: %w", err)
				}
				if debugOut != "" {
					if err := os.WriteFile(debugOut, js, 0o644); err != nil {
						return fmt.Errorf("write debug json failed: %w", err)
					}
					fmt.Printf("[debug-json] wrote %s (%d bytes)\n", debugOut, len(js))
				} else {
					fmt.Println("--- DEBUG-JSON ---")
					os.Stdout.Write(js)
					fmt.Println()
				}
				fmt.Println("--- DWARF SECTIONS ---")
				secs, err := debug.BuildDWARF(dbg)
				if err != nil {
					return fmt.Errorf("build dwarf failed: %w", err)
				}
				printSection := func(name string, b []byte) { fmt.Printf("[%s] %d bytes\n", name, len(b)) }
				printSection(".debug_abbrev", secs.Abbrev)
				printSection(".debug_info", secs.Info)
				printSection(".debug_line", secs.Line)
				printSection(".debug_str", secs.Str)
				if dwarfDir != "" || outELF != "" || outCOFF != "" || outMachO != "" || os.Getenv("ORIZON_DEBUG_OBJ_OUT") != "" {
					if err := os.MkdirAll(dwarfDir, 0o755); err != nil {
						return fmt.Errorf("mkdir dwarf dir failed: %w", err)
					}
					write := func(name string, b []byte) error {
						p := filepath.Join(dwarfDir, name)
						return os.WriteFile(p, b, 0o644)
					}
					if dwarfDir != "" {
						if err := write("debug_abbrev.bin", secs.Abbrev); err != nil {
							return err
						}
						if err := write("debug_info.bin", secs.Info); err != nil {
							return err
						}
						if err := write("debug_line.bin", secs.Line); err != nil {
							return err
						}
						if err := write("debug_str.bin", secs.Str); err != nil {
							return err
						}
						fmt.Printf("[dwarf] wrote raw sections to %s\n", dwarfDir)
					}
					// Auto-select object format by OS when ORIZON_DEBUG_OBJ_OUT is given
					if auto := os.Getenv("ORIZON_DEBUG_OBJ_OUT"); auto != "" {
						switch f := os.Getenv("ORIZON_DEBUG_OBJ_FORMAT"); f {
						case "elf":
							outELF = auto
						case "coff":
							outCOFF = auto
						case "macho":
							outMachO = auto
						default:
							switch runtime.GOOS {
							case "windows":
								outCOFF = auto
							case "darwin":
								outMachO = auto
							default:
								outELF = auto
							}
						}
					}
					if outELF != "" {
						if err := debug.WriteELFWithDWARF(outELF, secs); err != nil {
							return fmt.Errorf("write ELF failed: %w", err)
						}
						fmt.Printf("[dwarf] wrote ELF object: %s\n", outELF)
					}
					if outCOFF != "" {
						if err := debug.WriteCOFFWithDWARF(outCOFF, secs); err != nil {
							return fmt.Errorf("write COFF failed: %w", err)
						}
						fmt.Printf("[dwarf] wrote COFF object: %s\n", outCOFF)
					}
					if outMachO != "" {
						if err := debug.WriteMachOWithDWARF(outMachO, secs); err != nil {
							return fmt.Errorf("write Mach-O failed: %w", err)
						}
						fmt.Printf("[dwarf] wrote Mach-O object: %s\n", outMachO)
					}
				}
			}

			if emitSrcMap {
				sm, err := debug.GenerateSourceMap(hirProg)
				if err != nil {
					return fmt.Errorf("generate sourcemap failed: %w", err)
				}
				js, err := debug.SerializeSourceMap(sm)
				if err != nil {
					return fmt.Errorf("serialize sourcemap failed: %w", err)
				}
				if smOut != "" {
					if err := os.WriteFile(smOut, js, 0o644); err != nil {
						return fmt.Errorf("write sourcemap failed: %w", err)
					}
					fmt.Printf("[sourcemap] wrote %s (%d bytes)\n", smOut, len(js))
				} else {
					fmt.Println("--- SOURCE-MAP ---")
					os.Stdout.Write(js)
					fmt.Println()
				}
			}
		}
	}

	return nil
}

// repeat はGo 1.21以前での文字列繰り返し関数（現在は不要）
// func repeat(s string, count int) string {
// 	result := ""
// 	for i := 0; i < count; i++ {
// 		result += s
// 	}
// 	return result
// }
