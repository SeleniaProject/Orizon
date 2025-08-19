// Integration of advanced diagnostic system with parser
// Provides enhanced error reporting and static analysis for parsing

package parser

import (
	"fmt"
	"strings"

	"github.com/orizon-lang/orizon/internal/diagnostic"
	"github.com/orizon-lang/orizon/internal/lexer"
	"github.com/orizon-lang/orizon/internal/position"
)

// DiagnosticParser extends the base parser with enhanced diagnostics
type DiagnosticParser struct {
	*Parser
	engine *diagnostic.DiagnosticEngine
}

// NewDiagnosticParser creates a new parser with diagnostic capabilities
func NewDiagnosticParser(l *lexer.Lexer, filename string) *DiagnosticParser {
	config := diagnostic.DiagnosticConfig{
		MaxErrors:         50,
		WarningsAsErrors:  false,
		ShowSuggestions:   true,
		ShowRelatedInfo:   true,
		EnablePerformance: true,
		EnableStyle:       true,
		EnableSecurity:    true,
		VerboseOutput:     true,
	}

	return &DiagnosticParser{
		Parser: NewParser(l, filename),
		engine: diagnostic.NewDiagnosticEngine(config),
	}
}

// GetDiagnostics returns the diagnostic engine
func (dp *DiagnosticParser) GetDiagnostics() *diagnostic.DiagnosticEngine {
	return dp.engine
}

// Enhanced error reporting methods

// reportUnexpectedToken reports an unexpected token with suggestions
func (dp *DiagnosticParser) reportUnexpectedToken(expected string, actual lexer.Token) {
	span := position.Span{
		Start: position.Position{
			Filename: dp.filename,
			Line:     actual.Line,
			Column:   actual.Column,
			Offset:   0, // Would need to calculate from lexer
		},
		End: position.Position{
			Filename: dp.filename,
			Line:     actual.Line,
			Column:   actual.Column + len(actual.Literal),
			Offset:   len(actual.Literal),
		},
	}

	diag := diagnostic.NewDiagnostic().
		Error().
		Syntax().
		Code("E1001").
		Title("Unexpected token").
		Message(fmt.Sprintf("Expected '%s', found '%s'", expected, actual.Literal)).
		Span(span)

	// Add suggestions based on common mistakes
	dp.addTokenSuggestions(diag, expected, actual)

	dp.engine.AddDiagnostic(diag.Build())
}

// addTokenSuggestions adds contextual suggestions for token errors
func (dp *DiagnosticParser) addTokenSuggestions(diag *diagnostic.DiagnosticBuilder, expected string, actual lexer.Token) {
	switch {
	case expected == ";" && actual.Type == lexer.TokenNewline:
		diag.Suggest("Add semicolon", "Add a semicolon before the newline")
	case expected == ")" && actual.Type == lexer.TokenEOF:
		diag.Suggest("Add closing parenthesis", "Add a closing parenthesis")
	case expected == "}" && actual.Type == lexer.TokenEOF:
		diag.Suggest("Add closing brace", "Add a closing brace")
	case strings.Contains(expected, "identifier") && isKeyword(actual.Type):
		diag.Suggest("Use different name", fmt.Sprintf("'%s' is a keyword, try a different identifier", actual.Literal))
	case expected == "=" && actual.Literal == "==":
		diag.Suggest("Use assignment operator", "Use '=' for assignment, '==' is for comparison")
	}
}

// isKeyword checks if a token type is a keyword
func isKeyword(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenFunc, lexer.TokenLet, lexer.TokenVar, lexer.TokenConst,
		lexer.TokenStruct, lexer.TokenEnum, lexer.TokenTrait, lexer.TokenImpl,
		lexer.TokenIf, lexer.TokenElse:
		return true
	default:
		return false
	}
}

// reportUndefinedSymbol reports an undefined symbol with helpful suggestions
func (dp *DiagnosticParser) reportUndefinedSymbol(name string, span position.Span) {
	diag := diagnostic.NewDiagnostic().
		Error().
		Semantic().
		Code("E2001").
		Title("Undefined symbol").
		Message(fmt.Sprintf("Symbol '%s' is not defined in this scope", name)).
		Span(span)

	// Add suggestions for common typos or similar names
	dp.addSymbolSuggestions(diag, name)

	dp.engine.AddDiagnostic(diag.Build())
}

// addSymbolSuggestions suggests similar symbol names
func (dp *DiagnosticParser) addSymbolSuggestions(diag *diagnostic.DiagnosticBuilder, name string) {
	// This would typically check the symbol table for similar names
	// For now, we'll add some common suggestions

	commonVariants := []string{
		strings.ToLower(name),
		strings.ToUpper(name),
		strings.Title(name),
	}

	for _, variant := range commonVariants {
		if variant != name {
			diag.Suggest("Try alternative spelling", fmt.Sprintf("Did you mean '%s'?", variant))
		}
	}

	// Check for common typos
	if strings.HasSuffix(name, "s") && len(name) > 1 {
		singular := name[:len(name)-1]
		diag.Suggest("Try singular form", fmt.Sprintf("Did you mean '%s'?", singular))
	}
}

// reportTypeMismatch reports a type mismatch with conversion suggestions
func (dp *DiagnosticParser) reportTypeMismatch(expected, actual string, span position.Span) {
	diag := diagnostic.NewDiagnostic().
		Error().
		Type().
		Code("E3001").
		Title("Type mismatch").
		Message(fmt.Sprintf("Cannot assign value of type '%s' to variable of type '%s'", actual, expected)).
		Span(span)

	// Add conversion suggestions
	dp.addConversionSuggestions(diag, expected, actual)

	dp.engine.AddDiagnostic(diag.Build())
}

// addConversionSuggestions suggests type conversions
func (dp *DiagnosticParser) addConversionSuggestions(diag *diagnostic.DiagnosticBuilder, expected, actual string) {
	// Common type conversions
	conversions := map[string]map[string]string{
		"int": {
			"float":  "Use int(value) to convert float to int",
			"string": "Use strconv.Atoi(value) to convert string to int",
		},
		"float": {
			"int":    "Use float(value) to convert int to float",
			"string": "Use strconv.ParseFloat(value, 64) to convert string to float",
		},
		"string": {
			"int":   "Use strconv.Itoa(value) to convert int to string",
			"float": "Use strconv.FormatFloat(value, 'f', -1, 64) to convert float to string",
		},
	}

	if expectedConversions, exists := conversions[expected]; exists {
		if suggestion, exists := expectedConversions[actual]; exists {
			diag.Suggest("Add type conversion", suggestion)
		}
	}
}

// Static analysis methods

// analyzeUnusedVariables performs static analysis to find unused variables
func (dp *DiagnosticParser) analyzeUnusedVariables(program *Program) {
	// This is a simplified implementation
	// A real implementation would track variable usage throughout the AST

	visitor := &UnusedVariableAnalyzer{
		engine:   dp.engine,
		filename: dp.filename,
		declared: make(map[string]position.Span),
		used:     make(map[string]bool),
	}

	program.Accept(visitor)

	// Report unused variables
	for name, span := range visitor.declared {
		if !visitor.used[name] {
			diag := diagnostic.NewDiagnostic().
				Warning().
				Style().
				Code("W4001").
				Title("Unused variable").
				Message(fmt.Sprintf("Variable '%s' is declared but never used", name)).
				Span(span).
				Suggest("Remove variable", "Remove the unused variable declaration").
				Tag("unused")

			dp.engine.AddDiagnostic(diag.Build())
		}
	}
}

// UnusedVariableAnalyzer implements the Visitor interface to find unused variables
type UnusedVariableAnalyzer struct {
	engine   *diagnostic.DiagnosticEngine
	filename string
	declared map[string]position.Span
	used     map[string]bool
}

func (uva *UnusedVariableAnalyzer) VisitProgram(node *Program) {
	for _, stmt := range node.Statements {
		stmt.Accept(uva)
	}
}

func (uva *UnusedVariableAnalyzer) VisitVariableDeclaration(node *VariableDeclaration) {
	span := position.Span{
		Start: position.Position{
			Filename: uva.filename,
			Line:     1, // Would need actual position info
			Column:   1,
		},
		End: position.Position{
			Filename: uva.filename,
			Line:     1,
			Column:   10,
		},
	}
	uva.declared[node.Name.Value] = span

	if node.Initializer != nil {
		node.Initializer.Accept(uva)
	}
}

func (uva *UnusedVariableAnalyzer) VisitIdentifier(node *Identifier) {
	uva.used[node.Value] = true
}

// Implement other visitor methods with empty implementations for now
func (uva *UnusedVariableAnalyzer) VisitFunctionDeclaration(node *FunctionDeclaration)   {}
func (uva *UnusedVariableAnalyzer) VisitParameterDeclaration(node *ParameterDeclaration) {}
func (uva *UnusedVariableAnalyzer) VisitBlock(node *Block)                               {}
func (uva *UnusedVariableAnalyzer) VisitIfStatement(node *IfStatement)                   {}
func (uva *UnusedVariableAnalyzer) VisitWhileStatement(node *WhileStatement)             {}
func (uva *UnusedVariableAnalyzer) VisitForStatement(node *ForStatement)                 {}
func (uva *UnusedVariableAnalyzer) VisitReturnStatement(node *ReturnStatement)           {}
func (uva *UnusedVariableAnalyzer) VisitExpressionStatement(node *ExpressionStatement)   {}
func (uva *UnusedVariableAnalyzer) VisitBinaryExpression(node *BinaryExpression)         {}
func (uva *UnusedVariableAnalyzer) VisitUnaryExpression(node *UnaryExpression)           {}
func (uva *UnusedVariableAnalyzer) VisitCallExpression(node *CallExpression)             {}
func (uva *UnusedVariableAnalyzer) VisitMemberExpression(node *MemberExpression)         {}
func (uva *UnusedVariableAnalyzer) VisitIndexExpression(node *IndexExpression)           {}
func (uva *UnusedVariableAnalyzer) VisitLiteral(node *Literal)                           {}
func (uva *UnusedVariableAnalyzer) VisitArrayLiteral(node *ArrayLiteral)                 {}
func (uva *UnusedVariableAnalyzer) VisitObjectLiteral(node *ObjectLiteral)               {}
func (uva *UnusedVariableAnalyzer) VisitFunctionLiteral(node *FunctionLiteral)           {}
func (uva *UnusedVariableAnalyzer) VisitBasicType(node *BasicType)                       {}
func (uva *UnusedVariableAnalyzer) VisitArrayType(node *ArrayType)                       {}
func (uva *UnusedVariableAnalyzer) VisitFunctionType(node *FunctionType)                 {}
func (uva *UnusedVariableAnalyzer) VisitStructType(node *StructType)                     {}
func (uva *UnusedVariableAnalyzer) VisitFieldDeclaration(node *FieldDeclaration)         {}
func (uva *UnusedVariableAnalyzer) VisitInterfaceType(node *InterfaceType)               {}
func (uva *UnusedVariableAnalyzer) VisitMethodDeclaration(node *MethodDeclaration)       {}
func (uva *UnusedVariableAnalyzer) VisitMatchExpression(node *MatchExpression)           {}
func (uva *UnusedVariableAnalyzer) VisitMatchArm(node *MatchArm)                         {}
func (uva *UnusedVariableAnalyzer) VisitLiteralPattern(node *LiteralPattern)             {}
func (uva *UnusedVariableAnalyzer) VisitVariablePattern(node *VariablePattern)           {}
func (uva *UnusedVariableAnalyzer) VisitConstructorPattern(node *ConstructorPattern)     {}
func (uva *UnusedVariableAnalyzer) VisitGuardPattern(node *GuardPattern)                 {}
func (uva *UnusedVariableAnalyzer) VisitWildcardPattern(node *WildcardPattern)           {}

// Performance analysis methods

// analyzePerformance performs basic performance analysis
func (dp *DiagnosticParser) analyzePerformance(program *Program) {
	visitor := &PerformanceAnalyzer{
		engine:   dp.engine,
		filename: dp.filename,
	}

	program.Accept(visitor)
}

// PerformanceAnalyzer looks for common performance issues
type PerformanceAnalyzer struct {
	engine   *diagnostic.DiagnosticEngine
	filename string
}

func (pa *PerformanceAnalyzer) VisitProgram(node *Program) {
	for _, stmt := range node.Statements {
		stmt.Accept(pa)
	}
}

func (pa *PerformanceAnalyzer) VisitForStatement(node *ForStatement) {
	// Check for potential infinite loops or inefficient loops
	if node.Condition == nil {
		span := position.Span{
			Start: position.Position{Filename: pa.filename, Line: 1, Column: 1},
			End:   position.Position{Filename: pa.filename, Line: 1, Column: 10},
		}

		diag := diagnostic.NewDiagnostic().
			Warning().
			Performance().
			Code("W6001").
			Title("Potential infinite loop").
			Message("For loop without condition may run indefinitely").
			Span(span).
			Suggest("Add condition", "Add a termination condition to prevent infinite loop").
			Tag("performance")

		pa.engine.AddDiagnostic(diag.Build())
	}

	// Visit child nodes
	if node.Init != nil {
		node.Init.Accept(pa)
	}
	if node.Condition != nil {
		node.Condition.Accept(pa)
	}
	if node.Update != nil {
		node.Update.Accept(pa)
	}
	if node.Body != nil {
		node.Body.Accept(pa)
	}
}

// Implement other visitor methods with empty implementations for now
func (pa *PerformanceAnalyzer) VisitVariableDeclaration(node *VariableDeclaration)   {}
func (pa *PerformanceAnalyzer) VisitFunctionDeclaration(node *FunctionDeclaration)   {}
func (pa *PerformanceAnalyzer) VisitParameterDeclaration(node *ParameterDeclaration) {}
func (pa *PerformanceAnalyzer) VisitBlock(node *Block)                               {}
func (pa *PerformanceAnalyzer) VisitIfStatement(node *IfStatement)                   {}
func (pa *PerformanceAnalyzer) VisitWhileStatement(node *WhileStatement)             {}
func (pa *PerformanceAnalyzer) VisitReturnStatement(node *ReturnStatement)           {}
func (pa *PerformanceAnalyzer) VisitExpressionStatement(node *ExpressionStatement)   {}
func (pa *PerformanceAnalyzer) VisitBinaryExpression(node *BinaryExpression)         {}
func (pa *PerformanceAnalyzer) VisitUnaryExpression(node *UnaryExpression)           {}
func (pa *PerformanceAnalyzer) VisitCallExpression(node *CallExpression)             {}
func (pa *PerformanceAnalyzer) VisitMemberExpression(node *MemberExpression)         {}
func (pa *PerformanceAnalyzer) VisitIndexExpression(node *IndexExpression)           {}
func (pa *PerformanceAnalyzer) VisitIdentifier(node *Identifier)                     {}
func (pa *PerformanceAnalyzer) VisitLiteral(node *Literal)                           {}
func (pa *PerformanceAnalyzer) VisitArrayLiteral(node *ArrayLiteral)                 {}
func (pa *PerformanceAnalyzer) VisitObjectLiteral(node *ObjectLiteral)               {}
func (pa *PerformanceAnalyzer) VisitFunctionLiteral(node *FunctionLiteral)           {}
func (pa *PerformanceAnalyzer) VisitBasicType(node *BasicType)                       {}
func (pa *PerformanceAnalyzer) VisitArrayType(node *ArrayType)                       {}
func (pa *PerformanceAnalyzer) VisitFunctionType(node *FunctionType)                 {}
func (pa *PerformanceAnalyzer) VisitStructType(node *StructType)                     {}
func (pa *PerformanceAnalyzer) VisitFieldDeclaration(node *FieldDeclaration)         {}
func (pa *PerformanceAnalyzer) VisitInterfaceType(node *InterfaceType)               {}
func (pa *PerformanceAnalyzer) VisitMethodDeclaration(node *MethodDeclaration)       {}
func (pa *PerformanceAnalyzer) VisitMatchExpression(node *MatchExpression)           {}
func (pa *PerformanceAnalyzer) VisitMatchArm(node *MatchArm)                         {}
func (pa *PerformanceAnalyzer) VisitLiteralPattern(node *LiteralPattern)             {}
func (pa *PerformanceAnalyzer) VisitVariablePattern(node *VariablePattern)           {}
func (pa *PerformanceAnalyzer) VisitConstructorPattern(node *ConstructorPattern)     {}
func (pa *PerformanceAnalyzer) VisitGuardPattern(node *GuardPattern)                 {}
func (pa *PerformanceAnalyzer) VisitWildcardPattern(node *WildcardPattern)           {}

// Enhanced parsing methods that use diagnostics

// ParseWithDiagnostics parses the source code and returns both AST and diagnostics
func (dp *DiagnosticParser) ParseWithDiagnostics() (*Program, *diagnostic.DiagnosticEngine) {
	program := dp.ParseProgram()

	// Perform static analysis
	dp.analyzeUnusedVariables(program)
	dp.analyzePerformance(program)

	return program, dp.engine
}

// FormatDiagnostics returns a formatted string of all diagnostics
func (dp *DiagnosticParser) FormatDiagnostics() string {
	return dp.engine.FormatDiagnostics()
}

// HasErrors returns true if there are any errors
func (dp *DiagnosticParser) HasErrors() bool {
	return dp.engine.HasErrors()
}
