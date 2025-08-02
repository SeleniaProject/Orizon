// Package main provides the entry point for the Orizon compiler.
// Phase 1.1.1: Unicode対応字句解析器の基本実装
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/orizon-lang/orizon/internal/lexer"
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
	if err := compileFile(inputFile, *debugLexer); err != nil {
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
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("    orizon-compiler hello.oriz")
	fmt.Println("    orizon-compiler --debug-lexer example.oriz")
}

func compileFile(filename string, debugLexer bool) error {
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
	} else {
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
	}

	fmt.Println("🎉 Phase 1.1.2: Incremental lexing capability successful!")
	fmt.Println("📝 Note: Full compilation pipeline coming in future phases")

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
