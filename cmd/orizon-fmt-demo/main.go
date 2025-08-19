package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orifmt "github.com/orizon-lang/orizon/internal/format"
)

// Enhanced orizon-fmt demo with AST formatting and diff support
func main() {
	var (
		showDemo   bool
		testFile   string
		useAST     bool
		showDiff   bool
		indentSize int
		useTabs    bool
		maxLine    int
	)

	flag.BoolVar(&showDemo, "demo", false, "show formatting demo")
	flag.StringVar(&testFile, "file", "", "file to format")
	flag.BoolVar(&useAST, "ast", false, "use AST-based formatting")
	flag.BoolVar(&showDiff, "diff", false, "show differences")
	flag.IntVar(&indentSize, "indent", 4, "indentation size")
	flag.BoolVar(&useTabs, "tabs", false, "use tabs for indentation")
	flag.IntVar(&maxLine, "maxline", 100, "maximum line length")

	flag.Parse()

	fmt.Println("Orizon Enhanced Formatter Demo")
	fmt.Println("==============================")

	if showDemo {
		runDemo()
		return
	}

	if testFile != "" {
		processTestFile(testFile, useAST, showDiff, indentSize, useTabs, maxLine)
		return
	}

	fmt.Println("Usage:")
	fmt.Println("  -demo         Show formatting demo")
	fmt.Println("  -file <path>  Format specific file")
	fmt.Println("  -ast          Use AST-based formatting")
	fmt.Println("  -diff         Show differences")
	fmt.Println("  -indent <n>   Indentation size")
	fmt.Println("  -tabs         Use tabs")
	fmt.Println("  -maxline <n>  Maximum line length")
}

func runDemo() {
	// Demo source code with formatting issues
	demoSource := `
// Sample Orizon code with formatting issues
struct   Point {
x:i32,
y:   i32   ,
}

fn   main( ){
let   p=Point{x:10,y:20};
println( "Point: ({}, {})", p.x,p.y );
}

enum Option<T> {
None,
Some(T),
}

trait Display{
fn display(&self)->String;
}
`

	fmt.Println("Original source:")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println(demoSource)
	fmt.Println(strings.Repeat("-", 50))

	// Basic formatting
	basicOptions := orifmt.Options{PreserveNewlineStyle: false}
	basicFormatted := orifmt.FormatBytes([]byte(demoSource), basicOptions)

	fmt.Println("\nBasic formatted:")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println(string(basicFormatted))
	fmt.Println(strings.Repeat("-", 50))

	// AST formatting (would be used if AST parsing was working)
	fmt.Println("\nAST formatting would provide:")
	fmt.Println("- Proper indentation alignment")
	fmt.Println("- Consistent spacing around operators")
	fmt.Println("- Aligned struct fields")
	fmt.Println("- Proper line breaks and formatting")

	// Diff demo
	diffOptions := orifmt.DiffOptions{
		Mode:        orifmt.DiffModeUnified,
		Context:     3,
		ShowNumbers: true,
	}

	diffFormatter := orifmt.NewDiffFormatter(diffOptions)
	result := diffFormatter.GenerateDiff("demo.oriz", demoSource, string(basicFormatted))

	if result.HasChanges {
		fmt.Println("\nDiff output:")
		fmt.Println(strings.Repeat("-", 50))
		diff := diffFormatter.FormatDiff("demo.oriz", result)
		fmt.Print(diff)
		fmt.Println(strings.Repeat("-", 50))

		fmt.Printf("\nStatistics: +%d -%d lines\n",
			result.Stats.LinesAdded, result.Stats.LinesRemoved)
	}
}

func processTestFile(filename string, useAST, showDiff bool, indentSize int, useTabs bool, maxLine int) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Printf("Error: File %s does not exist\n", filename)
		os.Exit(1)
	}

	// Check if it's an Orizon file
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".oriz" && ext != ".orizon" {
		fmt.Printf("Warning: %s doesn't appear to be an Orizon source file\n", filename)
	}

	// Read file
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	original := string(data)
	fmt.Printf("Processing file: %s\n", filename)
	fmt.Printf("Original size: %d bytes, %d lines\n",
		len(original), strings.Count(original, "\n")+1)

	var formatted string

	if useAST {
		fmt.Println("Using AST-based formatting...")

		astOptions := orifmt.ASTFormattingOptions{
			IndentSize:                   indentSize,
			PreferTabs:                   useTabs,
			MaxLineLength:                maxLine,
			AlignFields:                  true,
			SpaceAroundOperators:         true,
			TrailingComma:                true,
			EmptyLineBetweenDeclarations: true,
		}

		astFormatted, err := orifmt.FormatSourceWithAST(original, astOptions)
		if err != nil {
			fmt.Printf("AST formatting failed: %v\n", err)
			fmt.Println("Falling back to basic formatting...")

			basicOptions := orifmt.Options{PreserveNewlineStyle: true}
			basicFormatted := orifmt.FormatBytes(data, basicOptions)
			formatted = string(basicFormatted)
		} else {
			formatted = astFormatted
		}
	} else {
		fmt.Println("Using basic formatting...")
		basicOptions := orifmt.Options{PreserveNewlineStyle: true}
		basicFormatted := orifmt.FormatBytes(data, basicOptions)
		formatted = string(basicFormatted)
	}

	fmt.Printf("Formatted size: %d bytes, %d lines\n",
		len(formatted), strings.Count(formatted, "\n")+1)

	if formatted == original {
		fmt.Println("No changes needed - file is already properly formatted")
		return
	}

	if showDiff {
		fmt.Println("\nDifferences found:")
		fmt.Println(strings.Repeat("-", 50))

		diffOptions := orifmt.DiffOptions{
			Mode:        orifmt.DiffModeUnified,
			Context:     3,
			ShowNumbers: true,
		}

		diffFormatter := orifmt.NewDiffFormatter(diffOptions)
		result := diffFormatter.GenerateDiff(filename, original, formatted)
		diff := diffFormatter.FormatDiff(filename, result)
		fmt.Print(diff)

		fmt.Printf("\nStats: +%d -%d lines changed\n",
			result.Stats.LinesAdded, result.Stats.LinesRemoved)
	} else {
		fmt.Println("\nFormatted output:")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Print(formatted)
		fmt.Println(strings.Repeat("-", 50))
	}
}
