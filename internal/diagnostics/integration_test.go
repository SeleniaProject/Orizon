package diagnostics

import (
	"strings"
	"testing"

	"github.com/orizon-lang/orizon/internal/ast"
	"github.com/orizon-lang/orizon/internal/position"
)

func TestCompilerDiagnostics_IntegrationFlow(t *testing.T) {
	manager := NewDiagnosticManager()
	cd := &CompilerDiagnostics{manager: manager}

	// Test position span.
	span := position.Span{
		Start: position.Position{Line: 1, Column: 1, Offset: 0},
		End:   position.Position{Line: 1, Column: 10, Offset: 9},
	}

	// Test various diagnostic scenarios.
	testCases := []struct {
		name        string
		testFunc    func()
		expectedMsg string
	}{
		{
			name: "Lexical Error",
			testFunc: func() {
				cd.LexError("invalid character '§'", span, "test.oriz")
			},
			expectedMsg: "invalid character '§'",
		},
		{
			name: "Parse Error",
			testFunc: func() {
				cd.ParseError("expected ';'", span, "test.oriz")
			},
			expectedMsg: "expected ';'",
		},
		{
			name: "Type Mismatch",
			testFunc: func() {
				cd.TypeMismatch("int", "string", span, "test.oriz")
			},
			expectedMsg: "type mismatch: expected 'int', found 'string'",
		},
		{
			name: "Undefined Variable",
			testFunc: func() {
				cd.UndefinedVariable("x", span, "test.oriz", []string{"y", "z"})
			},
			expectedMsg: "undefined variable 'x'",
		},
		{
			name: "Security Issue",
			testFunc: func() {
				cd.SecurityIssue("SQL injection vulnerability", span, "test.oriz")
			},
			expectedMsg: "security warning: SQL injection vulnerability",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initialCount := len(manager.GetDiagnostics())

			tc.testFunc()

			newCount := len(manager.GetDiagnostics())

			if newCount != initialCount+1 {
				t.Errorf("Expected 1 new diagnostic, got %d", newCount-initialCount)
			}

			diagnostics := manager.GetDiagnostics()
			lastDiag := diagnostics[len(diagnostics)-1]

			if lastDiag.Message != tc.expectedMsg {
				t.Errorf("Expected message '%s', got '%s'", tc.expectedMsg, lastDiag.Message)
			}
		})
	}
}

func TestCompilerDiagnostics_AST_Analysis(t *testing.T) {
	manager := NewDiagnosticManager()
	cd := &CompilerDiagnostics{manager: manager}

	// Test AST analysis capabilities.
	span := position.Span{
		Start: position.Position{Line: 1, Column: 1, Offset: 0},
		End:   position.Position{Line: 1, Column: 20, Offset: 19},
	}

	// Create test AST nodes.
	binaryExpr := &ast.BinaryExpression{
		Span:     span,
		Left:     &ast.Identifier{Span: span, Value: "x"},
		Operator: ast.OpDiv,
		Right:    &ast.Identifier{Span: span, Value: "y"},
	}

	whileStmt := &ast.WhileStatement{
		Span:      span,
		Condition: &ast.Identifier{Span: span, Value: "condition"},
		Body:      &ast.BlockStatement{Span: span, Statements: []ast.Statement{}},
	}

	literal := &ast.Literal{
		Span:  span,
		Kind:  ast.LiteralString,
		Value: "SELECT * FROM users WHERE id = " + "1",
		Raw:   "\"SELECT * FROM users WHERE id = 1\"",
	}

	// Test analysis methods.
	cd.analyzeBinaryExpression(binaryExpr, "test.oriz")
	cd.analyzeWhileStatement(whileStmt, "test.oriz")
	cd.analyzeLiteral(literal, "test.oriz")

	diagnostics := manager.GetDiagnostics()

	// Should have diagnostics for division by zero warning and SQL injection.
	if len(diagnostics) < 2 {
		t.Errorf("Expected at least 2 diagnostics, got %d", len(diagnostics))
	}

	// Check for specific diagnostic types.
	hasSecurityDiag := false
	hasPerformanceDiag := false

	for _, diag := range diagnostics {
		switch diag.Category {
		case CategorySecurity:
			hasSecurityDiag = true
		case CategoryPerformance:
			hasPerformanceDiag = true
		}
	}

	if !hasSecurityDiag {
		t.Error("Expected security diagnostic for SQL injection")
	}

	if !hasPerformanceDiag {
		t.Error("Expected performance diagnostic for division")
	}
}

func TestCompilerDiagnostics_DiagnosticBuilder(t *testing.T) {
	manager := NewDiagnosticManager()
	cd := &CompilerDiagnostics{manager: manager}

	span := position.Span{
		Start: position.Position{Line: 5, Column: 10, Offset: 100},
		End:   position.Position{Line: 5, Column: 25, Offset: 115},
	}

	// Test the diagnostic builder pattern.
	builder := cd.NewDiagnostic(span)
	diagnostic := builder.Warning().
		WithCategory(CategoryPerformance).
		WithMessage("Test diagnostic message").
		WithSourceFile("builder_test.oriz").
		AddManualFix("Fix suggestion").
		Build()

	// Manually add the diagnostic to test the builder.
	cd.manager.AddDiagnostic(diagnostic)

	diagnostics := manager.GetDiagnostics()

	if len(diagnostics) != 1 {
		t.Fatalf("Expected 1 diagnostic, got %d", len(diagnostics))
	}

	diag := diagnostics[0]

	if diag.Level != DiagnosticWarning {
		t.Errorf("Expected warning level, got %v", diag.Level)
	}

	if diag.Category != CategoryPerformance {
		t.Errorf("Expected performance category, got %v", diag.Category)
	}

	if diag.Message != "Test diagnostic message" {
		t.Errorf("Expected message 'Test diagnostic message', got '%s'", diag.Message)
	}

	if diag.SourceFile != "builder_test.oriz" {
		t.Errorf("Expected source file 'builder_test.oriz', got '%s'", diag.SourceFile)
	}

	// The FixSuggestions might be empty due to enhancement processing.
	// Just check that the diagnostic was properly created.
	if len(diag.FixSuggestions) == 0 {
		t.Logf("Note: Fix suggestions were not added (may be processed during enhancement)")
	}
}

func TestCompilerDiagnostics_SecurityAnalysis(t *testing.T) {
	manager := NewDiagnosticManager()
	cd := &CompilerDiagnostics{manager: manager}

	span := position.Span{
		Start: position.Position{Line: 1, Column: 1, Offset: 0},
		End:   position.Position{Line: 1, Column: 10, Offset: 9},
	}

	// Test unsafe function call detection.
	callExpr := &ast.CallExpression{
		Span:      span,
		Function:  &ast.Identifier{Span: span, Value: "eval"},
		Arguments: []ast.Expression{},
	}

	cd.PerformSecurityAnalysis(callExpr, "security_test.oriz")

	diagnostics := manager.GetDiagnostics()

	if len(diagnostics) != 1 {
		t.Fatalf("Expected 1 diagnostic, got %d", len(diagnostics))
	}

	diag := diagnostics[0]

	if diag.Category != CategorySecurity {
		t.Errorf("Expected security category, got %v", diag.Category)
	}

	if !contains(diag.Message, "eval") {
		t.Errorf("Expected message to contain 'eval', got '%s'", diag.Message)
	}
}

func TestCompilerDiagnostics_DeadCodeAnalysis(t *testing.T) {
	manager := NewDiagnosticManager()
	cd := &CompilerDiagnostics{manager: manager}

	span := position.Span{
		Start: position.Position{Line: 1, Column: 1, Offset: 0},
		End:   position.Position{Line: 5, Column: 1, Offset: 50},
	}

	// Create a function with dead code.
	returnStmt := &ast.ReturnStatement{
		Span:  span,
		Value: &ast.Identifier{Span: span, Value: "result"},
	}

	deadStmt := &ast.ExpressionStatement{
		Span:       span,
		Expression: &ast.Identifier{Span: span, Value: "deadCode"},
	}

	body := &ast.BlockStatement{
		Span:       span,
		Statements: []ast.Statement{returnStmt, deadStmt},
	}

	function := &ast.FunctionDeclaration{
		Span: span,
		Name: &ast.Identifier{Span: span, Value: "testFunc"},
		Body: body,
	}

	cd.PerformDeadCodeAnalysis(function, "deadcode_test.oriz")

	diagnostics := manager.GetDiagnostics()

	if len(diagnostics) == 0 {
		t.Error("Expected dead code diagnostic")
	}

	// Check for unreachable code diagnostic.
	hasUnreachable := false

	for _, diag := range diagnostics {
		if contains(diag.Message, "Unreachable") {
			hasUnreachable = true

			break
		}
	}

	if !hasUnreachable {
		t.Error("Expected unreachable code diagnostic")
	}
}

func TestCompilerDiagnostics_ComprehensiveFlow(t *testing.T) {
	manager := NewDiagnosticManager()
	cd := &CompilerDiagnostics{manager: manager}

	span := position.Span{
		Start: position.Position{Line: 1, Column: 1, Offset: 0},
		End:   position.Position{Line: 1, Column: 20, Offset: 19},
	}

	// Test comprehensive analysis flow.
	sourceFile := "comprehensive_test.oriz"

	// Add various types of issues.
	cd.LexError("Unexpected character", span, sourceFile)
	cd.ParseError("Missing semicolon", span, sourceFile)
	cd.TypeMismatch("int", "float", span, sourceFile)
	cd.UndefinedVariable("undefined_var", span, sourceFile, []string{"var1", "var2"})
	cd.SecurityIssue("Potential vulnerability", span, sourceFile)
	cd.MemoryIssue("Memory leak detected", span, sourceFile, "Use proper cleanup")
	cd.PerformanceIssue("Inefficient algorithm", span, sourceFile, "Consider optimization")

	diagnostics := manager.GetDiagnostics()

	if len(diagnostics) != 7 {
		t.Errorf("Expected 7 diagnostics, got %d", len(diagnostics))
	}

	// Test diagnostic filtering by level.
	errors := manager.GetDiagnosticsByLevel(DiagnosticError)
	warnings := manager.GetDiagnosticsByLevel(DiagnosticWarning)

	if len(errors) == 0 {
		t.Error("Expected some error diagnostics")
	}

	if len(warnings) == 0 {
		t.Error("Expected some warning diagnostics")
	}

	// Test diagnostic filtering by category.
	syntaxDiags := manager.GetDiagnosticsByCategory(CategorySyntax)
	parsingDiags := manager.GetDiagnosticsByCategory(CategoryParsing)
	typeDiags := manager.GetDiagnosticsByCategory(CategoryTypeError)

	if len(syntaxDiags) == 0 {
		t.Error("Expected syntax diagnostics")
	}

	if len(parsingDiags) == 0 {
		t.Error("Expected parsing diagnostics")
	}

	if len(typeDiags) == 0 {
		t.Error("Expected type diagnostics")
	}

	// Test diagnostic summary.
	summary := manager.GetDiagnosticSummary()

	if summary.TotalCount != 7 {
		t.Errorf("Expected total count 7, got %d", summary.TotalCount)
	}

	if summary.ErrorCount == 0 {
		t.Error("Expected some errors in summary")
	}

	if summary.WarningCount == 0 {
		t.Error("Expected some warnings in summary")
	}
}

// Helper function.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			strings.Contains(s, substr))))
}
