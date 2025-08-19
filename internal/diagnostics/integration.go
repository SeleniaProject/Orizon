// Package diagnostics - Integration with compiler components
package diagnostics

import (
	"fmt"
	"strings"

	"github.com/orizon-lang/orizon/internal/ast"
	"github.com/orizon-lang/orizon/internal/position"
)

// CompilerDiagnostics provides diagnostic integration for all compiler phases
type CompilerDiagnostics struct {
	manager *DiagnosticManager
}

// NewCompilerDiagnostics creates a new compiler diagnostics system
func NewCompilerDiagnostics() *CompilerDiagnostics {
	return &CompilerDiagnostics{
		manager: NewDiagnosticManager(),
	}
}

// GetManager returns the underlying diagnostic manager
func (cd *CompilerDiagnostics) GetManager() *DiagnosticManager {
	return cd.manager
}

// === Lexer Integration ===

// LexerError reports a lexical analysis error
func (cd *CompilerDiagnostics) LexerError(message string, span position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Error().
		WithCode("L001").
		WithCategory(CategorySyntax).
		WithMessage(message).
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanation("The lexer encountered an invalid token or character sequence.").
		AddSeeAlso("lexical-analysis").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// InvalidCharacterError reports an invalid character error
func (cd *CompilerDiagnostics) InvalidCharacterError(char rune, span position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Error().
		WithCode("L002").
		WithCategory(CategorySyntax).
		WithMessagef("invalid character '%c' (U+%04X)", char, char).
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanationf("The character '%c' is not valid in this context.", char).
		AddManualFix("Remove or replace the invalid character").
		AddSeeAlso("character-encoding").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// UnterminatedStringError reports an unterminated string literal
func (cd *CompilerDiagnostics) UnterminatedStringError(span position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Error().
		WithCode("L003").
		WithCategory(CategorySyntax).
		WithMessage("unterminated string literal").
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanation("String literals must be properly closed with matching quotes.").
		AddAutomaticFix("Add closing quote", "\"", span).
		AddExample("\"properly terminated string\"").
		AddSeeAlso("string-literals").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// === Parser Integration ===

// ParseError reports a parsing error
func (cd *CompilerDiagnostics) ParseError(message string, span position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Error().
		WithCode("P001").
		WithCategory(CategoryParsing).
		WithMessage(message).
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanation("The parser encountered a syntax error.").
		AddSeeAlso("syntax").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// UnexpectedTokenError reports an unexpected token error
func (cd *CompilerDiagnostics) UnexpectedTokenError(expected, actual string, span position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Error().
		WithCode("P002").
		WithCategory(CategoryParsing).
		WithMessagef("expected '%s', found '%s'", expected, actual).
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanationf("The parser expected a '%s' token but encountered '%s' instead.", expected, actual).
		AddManualFix("Insert the expected token").
		AddManualFix("Check for missing punctuation").
		AddSeeAlso("syntax").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// MissingSemicolonError reports a missing semicolon
func (cd *CompilerDiagnostics) MissingSemicolonError(span position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Error().
		WithCode("P003").
		WithCategory(CategoryParsing).
		WithMessage("missing semicolon").
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanation("Statements must be terminated with a semicolon.").
		AddAutomaticFix("Add semicolon", ";", span).
		AddExample("let x = 42; // Semicolon required").
		AddSeeAlso("statement-syntax").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// === Type System Integration ===

// TypeError reports a type system error
func (cd *CompilerDiagnostics) TypeError(message string, span position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Error().
		WithCode("T001").
		WithCategory(CategoryTypeError).
		WithMessage(message).
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanation("The type system detected an error.").
		AddSeeAlso("type-system").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// TypeMismatch reports a type mismatch
func (cd *CompilerDiagnostics) TypeMismatch(expected, actual string, span position.Span, sourceFile string) {
	diagnostic := TypeMismatchError(expected, actual, span)
	diagnostic.SourceFile = sourceFile
	cd.manager.AddDiagnostic(diagnostic)
}

// UndefinedVariable reports an undefined variable
func (cd *CompilerDiagnostics) UndefinedVariable(name string, span position.Span, sourceFile string, suggestions []string) {
	diagnostic := UndefinedVariableError(name, span, suggestions)
	diagnostic.SourceFile = sourceFile
	cd.manager.AddDiagnostic(diagnostic)
}

// UndefinedFunction reports an undefined function
func (cd *CompilerDiagnostics) UndefinedFunction(name string, span position.Span, sourceFile string, suggestions []string) {
	builder := NewDiagnosticBuilder().
		Error().
		WithCode("T002").
		WithCategory(CategoryUndefinedFunction).
		WithMessagef("undefined function '%s'", name).
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanationf("The function '%s' is called but has not been declared.", name)

	if len(suggestions) > 0 {
		builder.AddManualFix("Did you mean: " + suggestions[0] + "?")
	}

	diagnostic := builder.Build()
	cd.manager.AddDiagnostic(diagnostic)
}

// UndefinedType reports an undefined type
func (cd *CompilerDiagnostics) UndefinedType(name string, span position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Error().
		WithCode("T003").
		WithCategory(CategoryUndefinedType).
		WithMessagef("undefined type '%s'", name).
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanationf("The type '%s' is used but has not been declared.", name).
		AddManualFix("Import the module containing this type").
		AddManualFix("Define the type").
		AddSeeAlso("type-declarations").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// RedefinitionError reports a redefinition error
func (cd *CompilerDiagnostics) RedefinitionError(name string, span position.Span, originalSpan position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Error().
		WithCode("T004").
		WithCategory(CategoryRedefinition).
		WithMessagef("redefinition of '%s'", name).
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanationf("The identifier '%s' has already been defined in this scope.", name).
		AddRelatedInfof(originalSpan, "previous definition of '%s' was here", name).
		AddManualFix("Use a different name").
		AddManualFix("Remove one of the definitions").
		AddSeeAlso("scoping-rules").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// === Control Flow Analysis ===

// UnreachableCode reports unreachable code
func (cd *CompilerDiagnostics) UnreachableCode(span position.Span, sourceFile string) {
	diagnostic := DeadCodeWarning(span)
	diagnostic.SourceFile = sourceFile
	cd.manager.AddDiagnostic(diagnostic)
}

// MissingReturn reports a missing return statement
func (cd *CompilerDiagnostics) MissingReturn(functionName string, span position.Span, sourceFile string, returnType string) {
	diagnostic := MissingReturnError(functionName, span, returnType)
	diagnostic.SourceFile = sourceFile
	cd.manager.AddDiagnostic(diagnostic)
}

// InfiniteLoop reports a potential infinite loop
func (cd *CompilerDiagnostics) InfiniteLoop(span position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Warning().
		WithCode("C001").
		WithCategory(CategoryInfiniteLoop).
		WithMessage("potential infinite loop detected").
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanation("This loop may run indefinitely because the loop condition never changes.").
		AddManualFix("Add a break condition").
		AddManualFix("Modify the loop variable").
		AddSeeAlso("loop-analysis").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// === Code Quality Analysis ===

// UnusedVariable reports an unused variable
func (cd *CompilerDiagnostics) UnusedVariable(name string, span position.Span, sourceFile string) {
	diagnostic := UnusedVariableWarning(name, span)
	diagnostic.SourceFile = sourceFile
	cd.manager.AddDiagnostic(diagnostic)
}

// UnusedFunction reports an unused function
func (cd *CompilerDiagnostics) UnusedFunction(name string, span position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Warning().
		WithCode("Q001").
		WithCategory(CategoryUnusedFunction).
		WithMessagef("unused function '%s'", name).
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanationf("The function '%s' is defined but never called.", name).
		AddAutomaticFix("Remove unused function", "", span).
		AddManualFix("Export the function if it's part of the public API").
		AddSeeAlso("dead-code-elimination").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// UnusedImport reports an unused import
func (cd *CompilerDiagnostics) UnusedImport(name string, span position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Warning().
		WithCode("Q002").
		WithCategory(CategoryUnusedImport).
		WithMessagef("unused import '%s'", name).
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanationf("The import '%s' is not used in this file.", name).
		AddAutomaticFix("Remove unused import", "", span).
		AddSeeAlso("import-system").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// === Performance Analysis ===

// PerformanceIssue reports a performance issue
func (cd *CompilerDiagnostics) PerformanceIssue(issue string, span position.Span, sourceFile string, suggestion string) {
	diagnostic := PerformanceWarning(issue, span, suggestion)
	diagnostic.SourceFile = sourceFile
	cd.manager.AddDiagnostic(diagnostic)
}

// InEfficientAlgorithm reports an inefficient algorithm
func (cd *CompilerDiagnostics) InEfficientAlgorithm(description string, span position.Span, sourceFile string, betterApproach string) {
	diagnostic := NewDiagnosticBuilder().
		Warning().
		WithCode("P001").
		WithCategory(CategoryPerformance).
		WithMessagef("inefficient algorithm: %s", description).
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanation("This algorithm has poor time or space complexity.").
		AddManualFix("Consider using: " + betterApproach).
		AddSeeAlso("algorithm-optimization").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// === Memory Safety Analysis ===

// MemoryLeak reports a potential memory leak
func (cd *CompilerDiagnostics) MemoryLeak(resource string, span position.Span, sourceFile string) {
	diagnostic := MemoryLeakWarning(resource, span)
	diagnostic.SourceFile = sourceFile
	cd.manager.AddDiagnostic(diagnostic)
}

// UseAfterFree reports a use-after-free error
func (cd *CompilerDiagnostics) UseAfterFree(variable string, span position.Span, freeSpan position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Error().
		WithCode("M001").
		WithCategory(CategoryUseAfterFree).
		WithMessagef("use of freed memory: variable '%s'", variable).
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanationf("The variable '%s' is used after its memory has been freed.", variable).
		AddRelatedInfof(freeSpan, "memory was freed here").
		AddManualFix("Ensure the variable is not used after being freed").
		AddSeeAlso("memory-safety").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// DanglingPointer reports a dangling pointer
func (cd *CompilerDiagnostics) DanglingPointer(pointer string, span position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Error().
		WithCode("M002").
		WithCategory(CategoryDanglingPointer).
		WithMessagef("dangling pointer: '%s'", pointer).
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanationf("The pointer '%s' points to memory that is no longer valid.", pointer).
		AddManualFix("Ensure the pointed-to memory remains valid").
		AddSeeAlso("pointer-safety").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// === Security Analysis ===

// SecurityIssue reports a security issue
func (cd *CompilerDiagnostics) SecurityIssue(issue string, span position.Span, sourceFile string) {
	diagnostic := SecurityWarning(issue, span)
	diagnostic.SourceFile = sourceFile
	cd.manager.AddDiagnostic(diagnostic)
}

// BufferOverflow reports a potential buffer overflow
func (cd *CompilerDiagnostics) BufferOverflow(span position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Error().
		WithCode("S001").
		WithCategory(CategoryBufferOverflow).
		WithMessage("potential buffer overflow").
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanation("This operation may write beyond the bounds of the buffer.").
		AddManualFix("Add bounds checking").
		AddManualFix("Use safe string functions").
		AddSeeAlso("buffer-safety").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// === Style and Convention Analysis ===

// NamingConvention reports a naming convention violation
func (cd *CompilerDiagnostics) NamingConvention(name string, span position.Span, sourceFile string, expectedPattern string) {
	diagnostic := NewDiagnosticBuilder().
		Warning().
		WithCode("S003").
		WithCategory(CategoryNaming).
		WithMessagef("naming convention: '%s' should follow pattern '%s'", name, expectedPattern).
		WithSpan(span).
		WithSourceFile(sourceFile).
		WithExplanation("This identifier does not follow the recommended naming convention.").
		AddManualFix("Rename to follow the convention").
		AddSeeAlso("naming-conventions").
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// === AST Analysis Integration ===

// AnalyzeASTNode performs comprehensive analysis on an AST node
func (cd *CompilerDiagnostics) AnalyzeASTNode(node ast.Node, sourceFile string) {
	if node == nil {
		return
	}

	span := node.GetSpan()

	// Analyze based on node type
	switch n := node.(type) {
	case *ast.FunctionDeclaration:
		cd.analyzeFunctionDeclaration(n, sourceFile)
	case *ast.VariableDeclaration:
		cd.analyzeVariableDeclaration(n, sourceFile)
	case *ast.IfStatement:
		cd.analyzeIfStatement(n, sourceFile)
	case *ast.WhileStatement:
		cd.analyzeWhileStatement(n, sourceFile)
	case *ast.BinaryExpression:
		cd.analyzeBinaryExpression(n, sourceFile)
	}

	// Generic analysis for all nodes
	cd.analyzeNodeForCommonIssues(node, span, sourceFile)
}

// analyzeFunctionDeclaration analyzes function declarations
func (cd *CompilerDiagnostics) analyzeFunctionDeclaration(fn *ast.FunctionDeclaration, sourceFile string) {
	// Check for missing documentation
	if fn.Name.Value != "main" { // Skip main function
		// In a real implementation, check for doc comments
		cd.StyleIssue("missing documentation comment", fn.GetSpan(), sourceFile, "Add documentation comment")
	}

	// Check for long parameter lists
	if len(fn.Parameters) > 5 {
		cd.PerformanceIssue("function has many parameters", fn.GetSpan(), sourceFile, "Consider using a struct for parameters")
	}
}

// analyzeVariableDeclaration analyzes variable declarations
func (cd *CompilerDiagnostics) analyzeVariableDeclaration(vardecl *ast.VariableDeclaration, sourceFile string) {
	name := vardecl.Name.Value

	// Check naming conventions
	if len(name) == 1 && name != "i" && name != "j" && name != "k" {
		cd.NamingConvention(name, vardecl.GetSpan(), sourceFile, "descriptive names")
	}
}

// analyzeIfStatement analyzes if statements
func (cd *CompilerDiagnostics) analyzeIfStatement(ifStmt *ast.IfStatement, sourceFile string) {
	// Check for empty then/else blocks
	// This would require analyzing the block structure
}

// analyzeWhileStatement analyzes while statements
func (cd *CompilerDiagnostics) analyzeWhileStatement(whileStmt *ast.WhileStatement, sourceFile string) {
	builder := cd.NewDiagnostic(whileStmt.GetSpan())

	// Check for potential infinite loops by analyzing condition
	if whileStmt.Condition != nil {
		// This is a placeholder for more sophisticated analysis
		// In a real implementation, we'd check if the condition can ever become false
		cd.analyzeExpression(whileStmt.Condition, sourceFile)
	} else {
		builder.Warning().WithCategory(CategoryPerformance).
			WithMessage("While loop has no condition - potential infinite loop").
			AddManualFix("Add a proper termination condition").
			Build()
	}

	// Analyze the body
	if whileStmt.Body != nil {
		cd.analyzeStatement(whileStmt.Body, sourceFile)
	}
}

// analyzeBinaryExpression analyzes binary expressions
func (cd *CompilerDiagnostics) analyzeBinaryExpression(binExpr *ast.BinaryExpression, sourceFile string) {
	// Check for potential division by zero
	if binExpr.Operator == ast.OpDiv {
		// This would require constant folding analysis
		cd.WarnIssue("Potential division by zero", binExpr.GetSpan(), sourceFile, "Check divisor before division")
	}

	// Check for comparison with floating point equality
	if binExpr.Operator == ast.OpEq || binExpr.Operator == ast.OpNe {
		// This would require type analysis to check if operands are floating point
		// For now, just a placeholder
	}
}

// analyzeNodeForCommonIssues performs common analysis on any node
func (cd *CompilerDiagnostics) analyzeNodeForCommonIssues(node ast.Node, span position.Span, sourceFile string) {
	// Check for very long lines (style issue)
	if span.End.Column-span.Start.Column > 120 {
		cd.StyleIssue("line too long", span, sourceFile, "Break long lines")
	}
}

// NewDiagnostic creates a new diagnostic builder
func (cd *CompilerDiagnostics) NewDiagnostic(span position.Span) *DiagnosticBuilder {
	return NewDiagnosticBuilder().WithSpan(span)
}

// WarnIssue creates a warning diagnostic
func (cd *CompilerDiagnostics) WarnIssue(message string, span position.Span, sourceFile string, help string) {
	diagnostic := NewDiagnosticBuilder().
		Warning().
		WithCategory(CategoryPerformance).
		WithMessage(message).
		WithSpan(span).
		WithSourceFile(sourceFile).
		AddManualFix(help).
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// StyleIssue creates a style-related diagnostic
func (cd *CompilerDiagnostics) StyleIssue(message string, span position.Span, sourceFile string, help string) {
	diagnostic := NewDiagnosticBuilder().
		Info().
		WithCategory(CategoryStyle).
		WithMessage(message).
		WithSpan(span).
		WithSourceFile(sourceFile).
		AddManualFix(help).
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// analyzeExpression analyzes expressions recursively
func (cd *CompilerDiagnostics) analyzeExpression(expr ast.Expression, sourceFile string) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.BinaryExpression:
		cd.analyzeBinaryExpression(e, sourceFile)
		cd.analyzeExpression(e.Left, sourceFile)
		cd.analyzeExpression(e.Right, sourceFile)
	case *ast.UnaryExpression:
		cd.analyzeExpression(e.Operand, sourceFile)
	case *ast.CallExpression:
		cd.analyzeExpression(e.Function, sourceFile)
		for _, arg := range e.Arguments {
			cd.analyzeExpression(arg, sourceFile)
		}
	case *ast.Identifier:
		// Check for undefined variables (would require symbol table)
		// Placeholder for variable resolution
	case *ast.Literal:
		// Literals are generally safe, but could check for overflow
		cd.analyzeLiteral(e, sourceFile)
	}
}

// analyzeStatement analyzes statements recursively
func (cd *CompilerDiagnostics) analyzeStatement(stmt ast.Statement, sourceFile string) {
	if stmt == nil {
		return
	}

	switch s := stmt.(type) {
	case *ast.BlockStatement:
		for _, stmt := range s.Statements {
			cd.analyzeStatement(stmt, sourceFile)
		}
	case *ast.ExpressionStatement:
		cd.analyzeExpression(s.Expression, sourceFile)
	case *ast.IfStatement:
		cd.analyzeIfStatement(s, sourceFile)
	case *ast.WhileStatement:
		cd.analyzeWhileStatement(s, sourceFile)
	case *ast.ReturnStatement:
		if s.Value != nil {
			cd.analyzeExpression(s.Value, sourceFile)
		}
	case *ast.VariableDeclaration:
		cd.analyzeVariableDeclaration(s, sourceFile)
		if s.Value != nil {
			cd.analyzeExpression(s.Value, sourceFile)
		}
	}
}

// analyzeLiteral analyzes literal expressions
func (cd *CompilerDiagnostics) analyzeLiteral(lit *ast.Literal, sourceFile string) {
	// Check for potential overflow in integer literals
	if lit.Kind == ast.LiteralInteger {
		// This would require constant value analysis
	}

	// Check for SQL injection patterns in string literals
	if lit.Kind == ast.LiteralString {
		if value, ok := lit.Value.(string); ok {
			cd.checkStringLiteralSecurity(value, lit.GetSpan(), sourceFile)
		}
	}
}

// checkStringLiteralSecurity checks string literals for security issues
func (cd *CompilerDiagnostics) checkStringLiteralSecurity(value string, span position.Span, sourceFile string) {
	lowerValue := strings.ToLower(value)

	// Check for SQL injection patterns
	sqlKeywords := []string{"select ", "insert ", "update ", "delete ", "drop ", "alter "}
	for _, keyword := range sqlKeywords {
		if strings.Contains(lowerValue, keyword) {
			cd.SecurityIssue("Potential SQL injection vulnerability", span, sourceFile)
			break
		}
	}
}

// Advanced Analysis Methods

// PerformDeadCodeAnalysis performs dead code analysis on a function
func (cd *CompilerDiagnostics) PerformDeadCodeAnalysis(fn *ast.FunctionDeclaration, sourceFile string) {
	// This would require control flow graph analysis
	// Placeholder implementation
	if fn.Body != nil {
		cd.analyzeDeadCodeInBlock(fn.Body, sourceFile)
	}
}

// analyzeDeadCodeInBlock checks for unreachable code in a block
func (cd *CompilerDiagnostics) analyzeDeadCodeInBlock(block *ast.BlockStatement, sourceFile string) {
	hasReturn := false

	for _, stmt := range block.Statements {
		if hasReturn {
			// Code after return is unreachable
			cd.WarnIssue("Unreachable code", stmt.GetSpan(), sourceFile,
				"Remove unreachable code after return statement")
		}

		if _, isReturn := stmt.(*ast.ReturnStatement); isReturn {
			hasReturn = true
		}

		// Check for empty blocks
		if blockStmt, ok := stmt.(*ast.BlockStatement); ok {
			if len(blockStmt.Statements) == 0 {
				cd.StyleIssue("Empty block", blockStmt.GetSpan(), sourceFile,
					"Consider removing empty block or adding a comment")
			}
		}
	}
}

// PerformSecurityAnalysis performs security-related analysis
func (cd *CompilerDiagnostics) PerformSecurityAnalysis(node ast.Node, sourceFile string) {
	switch n := node.(type) {
	case *ast.CallExpression:
		cd.analyzeUnsafeCalls(n, sourceFile)
	case *ast.Literal:
		if n.Kind == ast.LiteralString {
			if value, ok := n.Value.(string); ok {
				cd.checkStringLiteralSecurity(value, n.GetSpan(), sourceFile)
			}
		}
	}
}

// analyzeUnsafeCalls checks for potentially unsafe function calls
func (cd *CompilerDiagnostics) analyzeUnsafeCalls(call *ast.CallExpression, sourceFile string) {
	if fn, ok := call.Function.(*ast.Identifier); ok {
		unsafeFunctions := []string{"eval", "exec", "system", "shell"}
		for _, unsafe := range unsafeFunctions {
			if fn.Value == unsafe {
				cd.SecurityIssue(fmt.Sprintf("Use of potentially unsafe function '%s'", unsafe),
					call.GetSpan(), sourceFile)
			}
		}
	}
}

// LexError reports a lexical error
func (cd *CompilerDiagnostics) LexError(message string, span position.Span, sourceFile string) {
	diagnostic := NewDiagnosticBuilder().
		Error().
		WithCategory(CategorySyntax).
		WithMessage(message).
		WithSpan(span).
		WithSourceFile(sourceFile).
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}

// MemoryIssue reports a memory-related issue
func (cd *CompilerDiagnostics) MemoryIssue(message string, span position.Span, sourceFile string, help string) {
	diagnostic := NewDiagnosticBuilder().
		Warning().
		WithCategory(CategoryMemoryLeak).
		WithMessage(message).
		WithSpan(span).
		WithSourceFile(sourceFile).
		AddManualFix(help).
		Build()

	cd.manager.AddDiagnostic(diagnostic)
}
