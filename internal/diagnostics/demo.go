package diagnostics

import (
	"fmt"
	"log"
	"strings"

	"github.com/orizon-lang/orizon/internal/ast"
	"github.com/orizon-lang/orizon/internal/position"
)

func RunComprehensiveDiagnosticsDemo() {
	fmt.Println("=== Orizon Compiler - Comprehensive Diagnostics System Demo ===")

	// Initialize the diagnostic system
	manager := NewDiagnosticManager()
	cd := &CompilerDiagnostics{manager: manager}

	// Create various diagnostic scenarios
	demonstrateLexicalAnalysis(cd)
	demonstrateParsingAnalysis(cd)
	demonstrateTypeAnalysis(cd)
	demonstrateSecurityAnalysis(cd)
	demonstratePerformanceAnalysis(cd)
	demonstrateAST_Analysis(cd)

	// Display comprehensive results
	displayDiagnosticSummary(manager)
	displayDetailedDiagnostics(manager)
}

func demonstrateLexicalAnalysis(cd *CompilerDiagnostics) {
	fmt.Println("📝 Lexical Analysis Diagnostics:")

	span := position.Span{
		Start: position.Position{Line: 1, Column: 15, Offset: 14},
		End:   position.Position{Line: 1, Column: 16, Offset: 15},
	}

	cd.LexError("unexpected character '§' in identifier", span, "example.oriz")
	cd.LexError("unterminated string literal", span, "example.oriz")

	fmt.Println("✓ Generated lexical error diagnostics")
}

func demonstrateParsingAnalysis(cd *CompilerDiagnostics) {
	fmt.Println("🔍 Parser Analysis Diagnostics:")

	span := position.Span{
		Start: position.Position{Line: 5, Column: 20, Offset: 120},
		End:   position.Position{Line: 5, Column: 21, Offset: 121},
	}

	cd.ParseError("expected ';' after statement", span, "example.oriz")
	cd.UnexpectedTokenError("}", "{", span, "example.oriz")
	cd.MissingSemicolonError(span, "example.oriz")

	fmt.Println("✓ Generated parsing error diagnostics")
}

func demonstrateTypeAnalysis(cd *CompilerDiagnostics) {
	fmt.Println("🎯 Type System Analysis Diagnostics:")

	span := position.Span{
		Start: position.Position{Line: 8, Column: 10, Offset: 200},
		End:   position.Position{Line: 8, Column: 15, Offset: 205},
	}

	cd.TypeMismatch("int", "string", span, "example.oriz")
	cd.UndefinedVariable("undefinedVar", span, "example.oriz", []string{"definedVar", "myVar"})
	cd.UndefinedFunction("undefinedFunc", span, "example.oriz", []string{"println", "print"})
	cd.UndefinedType("UndefinedType", span, "example.oriz")

	fmt.Println("✓ Generated type system diagnostics")
}

func demonstrateSecurityAnalysis(cd *CompilerDiagnostics) {
	fmt.Println("🔒 Security Analysis Diagnostics:")

	span := position.Span{
		Start: position.Position{Line: 12, Column: 5, Offset: 300},
		End:   position.Position{Line: 12, Column: 25, Offset: 320},
	}

	// Test with unsafe function call
	callExpr := &ast.CallExpression{
		Span:      span,
		Function:  &ast.Identifier{Span: span, Value: "eval"},
		Arguments: []ast.Expression{},
	}

	cd.PerformSecurityAnalysis(callExpr, "example.oriz")

	// Test with SQL injection pattern
	sqlLiteral := &ast.Literal{
		Span:  span,
		Kind:  ast.LiteralString,
		Value: "SELECT * FROM users WHERE id = " + "1; DROP TABLE users;",
		Raw:   "\"SELECT * FROM users WHERE id = 1; DROP TABLE users;\"",
	}

	cd.PerformSecurityAnalysis(sqlLiteral, "example.oriz")

	fmt.Println("✓ Generated security analysis diagnostics")
}

func demonstratePerformanceAnalysis(cd *CompilerDiagnostics) {
	fmt.Println("⚡ Performance Analysis Diagnostics:")

	span := position.Span{
		Start: position.Position{Line: 15, Column: 10, Offset: 400},
		End:   position.Position{Line: 18, Column: 1, Offset: 450},
	}

	cd.PerformanceIssue("inefficient nested loop detected", span, "example.oriz",
		"Consider algorithm optimization")
	cd.MemoryIssue("potential memory leak in loop", span, "example.oriz",
		"Ensure proper resource cleanup")

	fmt.Println("✓ Generated performance analysis diagnostics")
}

func demonstrateAST_Analysis(cd *CompilerDiagnostics) {
	fmt.Println("🌳 AST-based Analysis Diagnostics:")

	span := position.Span{
		Start: position.Position{Line: 20, Column: 1, Offset: 500},
		End:   position.Position{Line: 25, Column: 1, Offset: 600},
	}

	// Create a complex AST structure for analysis
	binaryExpr := &ast.BinaryExpression{
		Span:     span,
		Left:     &ast.Identifier{Span: span, Value: "x"},
		Operator: ast.OpDiv,
		Right:    &ast.Identifier{Span: span, Value: "y"},
	}

	whileStmt := &ast.WhileStatement{
		Span:      span,
		Condition: &ast.Identifier{Span: span, Value: "true"},
		Body:      &ast.BlockStatement{Span: span, Statements: []ast.Statement{}},
	}

	// Create function with dead code
	returnStmt := &ast.ReturnStatement{
		Span:  span,
		Value: &ast.Identifier{Span: span, Value: "result"},
	}

	deadStmt := &ast.ExpressionStatement{
		Span:       span,
		Expression: &ast.Identifier{Span: span, Value: "unreachableCode"},
	}

	body := &ast.BlockStatement{
		Span:       span,
		Statements: []ast.Statement{returnStmt, deadStmt},
	}

	function := &ast.FunctionDeclaration{
		Span: span,
		Name: &ast.Identifier{Span: span, Value: "exampleFunc"},
		Body: body,
	}

	// Perform comprehensive AST analysis
	cd.analyzeBinaryExpression(binaryExpr, "example.oriz")
	cd.analyzeWhileStatement(whileStmt, "example.oriz")
	cd.PerformDeadCodeAnalysis(function, "example.oriz")

	fmt.Println("✓ Generated AST-based analysis diagnostics")
}

func displayDiagnosticSummary(manager *DiagnosticManager) {
	fmt.Println("📊 Diagnostic Summary:")
	fmt.Println("=" + strings.Repeat("=", 50))

	summary := manager.GetDiagnosticSummary()

	fmt.Printf("Total Diagnostics: %d\n", summary.TotalCount)
	fmt.Printf("  Errors:   %d\n", summary.ErrorCount)
	fmt.Printf("  Warnings: %d\n", summary.WarningCount)
	fmt.Printf("  Info:     %d\n", summary.InfoCount)
	fmt.Printf("  Hints:    %d\n", summary.HintCount)

	fmt.Println()

	// Display summary by category
	categories := []DiagnosticCategory{
		CategorySyntax,
		CategoryParsing,
		CategoryTypeError,
		CategoryUndefinedVariable,
		CategoryUndefinedFunction,
		CategorySecurity,
		CategoryPerformance,
		CategoryMemoryLeak,
	}

	fmt.Println("Breakdown by Category:")
	for _, category := range categories {
		diags := manager.GetDiagnosticsByCategory(category)
		if len(diags) > 0 {
			fmt.Printf("  %s: %d\n", category.String(), len(diags))
		}
	}

	fmt.Println()
}

func displayDetailedDiagnostics(manager *DiagnosticManager) {
	fmt.Println("📋 Detailed Diagnostic Report:")
	fmt.Println("=" + strings.Repeat("=", 50))

	diagnostics_list := manager.GetDiagnostics()

	for i, diag := range diagnostics_list {
		fmt.Printf("%d. %s\n", i+1, manager.FormatDiagnostic(diag, true))
		fmt.Println()
	}

	// Display final summary
	fmt.Println(manager.FormatSummary())
}

// Initialize demonstration
func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
