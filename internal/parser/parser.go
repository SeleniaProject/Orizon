// Package parser implements the Orizon recursive descent parser
// Phase 1.2.1: 再帰下降パーサー実装
package parser

import (
	"fmt"
	"strconv"

	"github.com/orizon-lang/orizon/internal/lexer"
)

// Parser represents the recursive descent parser
type Parser struct {
	lexer   *lexer.Lexer
	current lexer.Token
	peek    lexer.Token
	errors  []error

	// Parser state
	filename string

	// Error recovery and suggestion system (Phase 1.2.4)
	suggestionEngine *SuggestionEngine
	suggestions      []Suggestion
	recoveryMode     ErrorRecoveryMode
}

// ParseError represents a parsing error with context
type ParseError struct {
	Position Position
	Message  string
	Context  string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("Parse error at %s: %s", e.Position.String(), e.Message)
}

// NewParser creates a ne// parseInfixExpression parses infix expressions with extended operator support
func (p *Parser) parseInfixExpression(left Expression) Expression {
	switch p.current.Type {
	// Arithmetic operators
	case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenMul, lexer.TokenDiv, lexer.TokenMod:
		return p.parseBinaryExpression(left)
	// Power operator (right associative)
	case lexer.TokenPower:
		return p.parsePowerExpression(left)
	// Comparison operators
	case lexer.TokenEq, lexer.TokenNe, lexer.TokenLt, lexer.TokenLe, lexer.TokenGt, lexer.TokenGe:
		return p.parseBinaryExpression(left)
	// Logical operators
	case lexer.TokenAnd, lexer.TokenOr:
		return p.parseBinaryExpression(left)
	// Bitwise operators
	case lexer.TokenBitAnd, lexer.TokenBitOr, lexer.TokenBitXor, lexer.TokenShl, lexer.TokenShr:
		return p.parseBinaryExpression(left)
	// Assignment operators
	case lexer.TokenAssign:
		return p.parseAssignmentExpression(left)
	case lexer.TokenPlusAssign, lexer.TokenMinusAssign, lexer.TokenMulAssign,
		lexer.TokenDivAssign, lexer.TokenModAssign:
		return p.parseCompoundAssignmentExpression(left)
	case lexer.TokenBitAndAssign, lexer.TokenBitOrAssign, lexer.TokenBitXorAssign,
		lexer.TokenShlAssign, lexer.TokenShrAssign:
		return p.parseCompoundAssignmentExpression(left)
	// Call and access operators
	case lexer.TokenLParen:
		return p.parseCallExpression(left)
	case lexer.TokenLBracket:
		return p.parseIndexExpression(left)
	case lexer.TokenDot:
		return p.parseMemberExpression(left)
	// Ternary conditional operator
	case lexer.TokenQuestion:
		return p.parseTernaryExpression(left)
	default:
		return nil
	}
}

// NewParser creates a new parser instance
func NewParser(l *lexer.Lexer, filename string) *Parser {
	p := &Parser{
		lexer:        l,
		filename:     filename,
		errors:       make([]error, 0),
		suggestions:  make([]Suggestion, 0),
		recoveryMode: PhraseLevel, // Default to phrase-level recovery
	}

	// Initialize suggestion engine with phrase-level recovery
	p.suggestionEngine = NewSuggestionEngine(p.recoveryMode)

	// Read the first two tokens
	p.nextToken()
	p.nextToken()

	return p
}

// Parse parses the input and returns an AST
func (p *Parser) Parse() (*Program, []error) {
	program := p.parseProgram()
	return program, p.errors
}

// nextToken advances the parser to the next token
func (p *Parser) nextToken() {
	p.current = p.peek
	p.peek = p.lexer.NextToken()
}

// currentTokenIs checks if the current token is of the given type
func (p *Parser) currentTokenIs(tokenType lexer.TokenType) bool {
	return p.current.Type == tokenType
}

// peekTokenIs checks if the peek token is of the given type
func (p *Parser) peekTokenIs(tokenType lexer.TokenType) bool {
	return p.peek.Type == tokenType
}

// expectPeek advances if the peek token matches the expected type
func (p *Parser) expectPeek(tokenType lexer.TokenType) bool {
	if p.peekTokenIs(tokenType) {
		p.nextToken()
		return true
	}
	p.peekError(tokenType)

	// Attempt error recovery if enabled (Phase 1.2.4)
	if p.recoveryMode != PanicMode && p.suggestionEngine != nil {
		recovered := p.recoverFromError(fmt.Sprintf("expecting %s", tokenType.String()))
		if recovered {
			// Check if we can continue after recovery
			if p.peekTokenIs(tokenType) {
				p.nextToken()
				return true
			}
		}
	}

	return false
}

// peekError records a peek token mismatch error
func (p *Parser) peekError(expected lexer.TokenType) {
	msg := fmt.Sprintf("expected %s, got %s", expected.String(), p.peek.Type.String())

	// Track expected token for suggestion engine (Phase 1.2.4)
	if p.suggestionEngine != nil {
		p.suggestionEngine.AddExpectedToken(expected)
	}

	p.addError(TokenToPosition(p.peek), msg, "token mismatch")
}

// addError adds an error to the parser's error list
func (p *Parser) addError(pos Position, message, context string) {
	pos.File = p.filename
	parseErr := &ParseError{
		Position: pos,
		Message:  message,
		Context:  context,
	}
	p.errors = append(p.errors, parseErr)

	// Generate suggestions using the error recovery system (Phase 1.2.4)
	if p.suggestionEngine != nil {
		newSuggestions := p.suggestionEngine.RecoverFromError(p, parseErr)
		p.suggestions = append(p.suggestions, newSuggestions...)
	}
}

// addErrorWithSuggestion adds an error with manual suggestions
func (p *Parser) addErrorWithSuggestion(pos Position, message, context string, suggestions []Suggestion) {
	pos.File = p.filename
	parseErr := &ParseError{
		Position: pos,
		Message:  message,
		Context:  context,
	}
	p.errors = append(p.errors, parseErr)
	p.suggestions = append(p.suggestions, suggestions...)
}

// recoverFromError attempts intelligent error recovery
func (p *Parser) recoverFromError(expectedContext string) bool {
	if p.suggestionEngine == nil {
		return false
	}

	// Create a temporary error for recovery analysis
	tempErr := &ParseError{
		Position: TokenToPosition(p.current),
		Message:  fmt.Sprintf("unexpected token in %s", expectedContext),
		Context:  expectedContext,
	}

	// Let the suggestion engine handle recovery
	suggestions := p.suggestionEngine.RecoverFromError(p, tempErr)
	p.suggestions = append(p.suggestions, suggestions...)

	// Return true if recovery was successful (parser position changed)
	return true
}

// GetSuggestions returns all accumulated suggestions
func (p *Parser) GetSuggestions() []Suggestion {
	return p.suggestions
}

// SetRecoveryMode changes the error recovery strategy
func (p *Parser) SetRecoveryMode(mode ErrorRecoveryMode) {
	p.recoveryMode = mode
	if p.suggestionEngine != nil {
		p.suggestionEngine.mode = mode
	}
}

// skipTo skips tokens until one of the given types is found
func (p *Parser) skipTo(tokenTypes ...lexer.TokenType) {
	for !p.currentTokenIs(lexer.TokenEOF) {
		for _, tokenType := range tokenTypes {
			if p.currentTokenIs(tokenType) {
				return
			}
		}
		p.nextToken()
	}
}

// ====== Grammar Rules ======

// parseProgram parses the entire program
func (p *Parser) parseProgram() *Program {
	startPos := TokenToPosition(p.current)
	declarations := make([]Declaration, 0)

	for !p.currentTokenIs(lexer.TokenEOF) {
		// Skip whitespace and comments
		if p.currentTokenIs(lexer.TokenWhitespace) || p.currentTokenIs(lexer.TokenComment) {
			p.nextToken()
			continue
		}

		// Parse declaration
		if decl := p.parseDeclaration(); decl != nil {
			declarations = append(declarations, decl)
		}
		p.nextToken()
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return NewProgram(span, declarations)
}

// parseDeclaration parses a top-level declaration
func (p *Parser) parseDeclaration() Declaration {
	// Parse optional modifiers at declaration start: pub/async (order-insensitive)
	isPublic := false
	isAsync := false
	for {
		switch p.current.Type {
		case lexer.TokenPub:
			isPublic = true
			p.nextToken()
			continue
		case lexer.TokenAsync:
			isAsync = true
			p.nextToken()
			continue
		}
		break
	}

	var decl Declaration
	switch p.current.Type {
	case lexer.TokenFunc:
		decl = p.parseFunctionDeclaration()
	case lexer.TokenLet, lexer.TokenVar, lexer.TokenConst:
		decl = p.parseVariableDeclaration()
	case lexer.TokenMacro:
		decl = p.parseMacroDeclaration()
	default:
		// Try to parse as expression statement
		stmt := p.parseExpressionStatement()
		if stmt != nil {
			// Modifiers are not allowed for expression statements
			if isPublic || isAsync {
				p.addError(TokenToPosition(p.current), "modifiers not allowed for expression statements", "declaration modifiers")
			}
			return stmt
		}
		p.addError(TokenToPosition(p.current),
			fmt.Sprintf("unexpected token %s in declaration", p.current.Type.String()),
			"declaration parsing")
		p.skipTo(lexer.TokenFunc, lexer.TokenLet, lexer.TokenVar, lexer.TokenConst, lexer.TokenMacro, lexer.TokenStruct, lexer.TokenEnum, lexer.TokenEOF)
		return nil
	}

	// Apply parsed modifiers to the declaration where applicable
	switch d := decl.(type) {
	case *FunctionDeclaration:
		if d == nil {
			return nil
		}
		d.IsPublic = isPublic
		// If 'async' modifier present, mark declaration as async
		d.IsAsync = isAsync
	case *VariableDeclaration:
		if d == nil {
			return nil
		}
		d.IsPublic = isPublic
		if isAsync {
			// async on variables is invalid; report but continue
			p.addError(TokenToPosition(p.current), "async modifier is not valid for variable declarations", "declaration modifiers")
		}
	case *MacroDefinition:
		if d == nil {
			return nil
		}
		d.IsPublic = isPublic
		if isAsync {
			p.addError(TokenToPosition(p.current), "async modifier is not valid for macro declarations", "declaration modifiers")
		}
	}
	return decl
} // parseFunctionDeclaration parses a function declaration
func (p *Parser) parseFunctionDeclaration() *FunctionDeclaration {
	startPos := TokenToPosition(p.current)

	if !p.currentTokenIs(lexer.TokenFunc) {
		return nil
	}

	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	if !p.expectPeek(lexer.TokenLParen) {
		return nil
	}

	parameters := p.parseParameterList()

	if !p.expectPeek(lexer.TokenRParen) {
		return nil
	}

	// Optional return type
	var returnType Type
	if p.peekTokenIs(lexer.TokenArrow) {
		p.nextToken() // consume arrow
		p.nextToken() // move to type
		returnType = p.parseType()
	}

	if !p.expectPeek(lexer.TokenLBrace) {
		return nil
	}

	body := p.parseBlockStatement()
	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &FunctionDeclaration{
		Span:       span,
		Name:       name,
		Parameters: parameters,
		ReturnType: returnType,
		Body:       body,
		IsPublic:   false,
		IsAsync:    false,
	}
}

// parseParameterList parses function parameters
func (p *Parser) parseParameterList() []*Parameter {
	parameters := make([]*Parameter, 0)

	if p.peekTokenIs(lexer.TokenRParen) {
		return parameters
	}

	p.nextToken()

	// Parse first parameter
	param := p.parseParameter()
	if param != nil {
		parameters = append(parameters, param)
	}

	// Parse remaining parameters
	for p.peekTokenIs(lexer.TokenComma) {
		p.nextToken() // consume comma
		p.nextToken() // move to next parameter
		param := p.parseParameter()
		if param != nil {
			parameters = append(parameters, param)
		}
	}

	return parameters
}

// parseParameter parses a single parameter
func (p *Parser) parseParameter() *Parameter {
	startPos := TokenToPosition(p.current)
	// Optional 'mut' modifier on parameters
	isMut := false
	if p.currentTokenIs(lexer.TokenMut) {
		isMut = true
		p.nextToken()
	}

	if !p.currentTokenIs(lexer.TokenIdentifier) {
		p.addError(TokenToPosition(p.current),
			"expected parameter name", "parameter parsing")
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	if !p.expectPeek(lexer.TokenColon) {
		return nil
	}

	p.nextToken() // move to type
	typeSpec := p.parseType()

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &Parameter{
		Span:     span,
		Name:     name,
		TypeSpec: typeSpec,
		IsMut:    isMut,
	}
}

// parseVariableDeclaration parses a variable declaration
func (p *Parser) parseVariableDeclaration() *VariableDeclaration {
	startPos := TokenToPosition(p.current)

	isMutable := p.currentTokenIs(lexer.TokenVar)
	// Support `let mut name` form
	if p.currentTokenIs(lexer.TokenLet) && p.peekTokenIs(lexer.TokenMut) {
		p.nextToken() // consume 'mut'
		isMutable = true
	}

	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	// Optional type annotation
	var typeSpec Type
	if p.peekTokenIs(lexer.TokenColon) {
		p.nextToken() // consume colon
		p.nextToken() // move to type
		typeSpec = p.parseType()
	}

	// Optional initializer
	var initializer Expression
	if p.peekTokenIs(lexer.TokenAssign) {
		p.nextToken() // consume =
		p.nextToken() // move to expression
		initializer = p.parseExpression(LOWEST)
	}

	// Expect semicolon
	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &VariableDeclaration{
		Span:        span,
		Name:        name,
		TypeSpec:    typeSpec,
		Initializer: initializer,
		IsMutable:   isMutable,
		// Public visibility is applied by parseDeclaration based on leading modifiers
		IsPublic: false,
	}
}

// parseType parses a type specification
func (p *Parser) parseType() Type {
	switch p.current.Type {
	case lexer.TokenIdentifier:
		return &BasicType{
			Span: TokenToSpan(p.current),
			Name: p.current.Literal,
		}
	default:
		p.addError(TokenToPosition(p.current),
			fmt.Sprintf("unexpected token %s in type", p.current.Type.String()),
			"type parsing")
		return nil
	}
}

// parseBlockStatement parses a block statement
func (p *Parser) parseBlockStatement() *BlockStatement {
	startPos := TokenToPosition(p.current)
	statements := make([]Statement, 0)

	p.nextToken() // consume opening brace

	for !p.currentTokenIs(lexer.TokenRBrace) && !p.currentTokenIs(lexer.TokenEOF) {
		// Skip whitespace and comments
		if p.currentTokenIs(lexer.TokenWhitespace) || p.currentTokenIs(lexer.TokenComment) {
			p.nextToken()
			continue
		}

		stmt := p.parseStatement()
		if stmt != nil {
			statements = append(statements, stmt)
		}
		p.nextToken()
	}

	// Check if we hit EOF without finding closing brace
	if p.currentTokenIs(lexer.TokenEOF) {
		p.addError(TokenToPosition(p.current), "Unclosed block: missing closing brace", "block parsing")
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &BlockStatement{
		Span:       span,
		Statements: statements,
	}
}

// parseStatement parses a statement
func (p *Parser) parseStatement() Statement {
	switch p.current.Type {
	case lexer.TokenLet, lexer.TokenVar, lexer.TokenConst:
		return p.parseVariableDeclaration()
	case lexer.TokenReturn:
		return p.parseReturnStatement()
	case lexer.TokenIf:
		return p.parseIfStatement()
	case lexer.TokenWhile:
		return p.parseWhileStatement()
	case lexer.TokenLBrace:
		return p.parseBlockStatement()
	default:
		// Enhanced error recovery for unknown statement beginnings
		stmt := p.parseExpressionStatement()
		if stmt == nil && p.recoveryMode != PanicMode {
			p.recoverFromError("statement")
			// Try to continue parsing after recovery
			if p.current.Type != lexer.TokenEOF {
				p.nextToken()
				return p.parseStatement()
			}
		}
		return stmt
	}
}

// parseReturnStatement parses a return statement
func (p *Parser) parseReturnStatement() *ReturnStatement {
	startPos := TokenToPosition(p.current)

	var value Expression
	if !p.peekTokenIs(lexer.TokenSemicolon) && !p.peekTokenIs(lexer.TokenRBrace) {
		p.nextToken()
		value = p.parseExpression(LOWEST)
	}

	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &ReturnStatement{
		Span:  span,
		Value: value,
	}
}

// parseIfStatement parses an if statement
func (p *Parser) parseIfStatement() *IfStatement {
	startPos := TokenToPosition(p.current)

	if !p.expectPeek(lexer.TokenLParen) {
		return nil
	}

	p.nextToken()
	condition := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TokenRParen) {
		return nil
	}

	if !p.expectPeek(lexer.TokenLBrace) {
		return nil
	}

	thenStmt := p.parseBlockStatement()

	var elseStmt Statement
	if p.peekTokenIs(lexer.TokenElse) {
		p.nextToken()
		if p.peekTokenIs(lexer.TokenIf) {
			p.nextToken()
			elseStmt = p.parseIfStatement()
		} else if p.peekTokenIs(lexer.TokenLBrace) {
			p.nextToken()
			elseStmt = p.parseBlockStatement()
		}
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &IfStatement{
		Span:      span,
		Condition: condition,
		ThenStmt:  thenStmt,
		ElseStmt:  elseStmt,
	}
}

// parseWhileStatement parses a while statement
func (p *Parser) parseWhileStatement() *WhileStatement {
	startPos := TokenToPosition(p.current)

	if !p.expectPeek(lexer.TokenLParen) {
		return nil
	}

	p.nextToken()
	condition := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TokenRParen) {
		return nil
	}

	if !p.expectPeek(lexer.TokenLBrace) {
		return nil
	}

	body := p.parseBlockStatement()

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &WhileStatement{
		Span:      span,
		Condition: condition,
		Body:      body,
	}
}

// parseExpressionStatement parses an expression statement
func (p *Parser) parseExpressionStatement() *ExpressionStatement {
	startPos := TokenToPosition(p.current)

	expr := p.parseExpression(LOWEST)

	// Enhanced semicolon handling with error recovery (Phase 1.2.4)
	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	} else if p.peekTokenIs(lexer.TokenNewline) {
		// Missing semicolon before newline - common error
		if p.suggestionEngine != nil {
			pos := TokenToPosition(p.current)
			pos.File = p.filename
			suggestion := Suggestion{
				Type:        ErrorFix,
				Message:     "Insert semicolon before newline",
				Position:    pos,
				Replacement: ";",
				Confidence:  0.9,
				Category:    SyntaxError,
			}
			p.suggestions = append(p.suggestions, suggestion)
		}
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &ExpressionStatement{
		Span:       span,
		Expression: expr,
	}
}

// ====== Expression Parsing (Pratt Parser) ======

// Precedence levels for operators - Complete precedence hierarchy
type Precedence int

const (
	_ Precedence = iota
	LOWEST
	ASSIGN      // = += -= *= /= %= &= |= ^= <<= >>=
	TERNARY     // ? :
	LOGICAL_OR  // ||
	LOGICAL_AND // &&
	BITWISE_OR  // |
	BITWISE_XOR // ^
	BITWISE_AND // &
	EQUALS      // == !=
	LESSGREATER // < <= > >=
	SHIFT       // << >>
	SUM         // + -
	PRODUCT     // * / %
	PREFIX      // -X !X ~X
	POWER       // ** (right associative)
	POSTFIX     // X++ X-- X!
	CALL        // myFunction(X) X[Y] X.Y
)

// precedences maps token types to their precedence levels
// Following C-style operator precedence with some modern language improvements
var precedences = map[lexer.TokenType]Precedence{
	// Assignment operators (right associative)
	lexer.TokenAssign:       ASSIGN,
	lexer.TokenPlusAssign:   ASSIGN,
	lexer.TokenMinusAssign:  ASSIGN,
	lexer.TokenMulAssign:    ASSIGN,
	lexer.TokenDivAssign:    ASSIGN,
	lexer.TokenModAssign:    ASSIGN,
	lexer.TokenBitAndAssign: ASSIGN,
	lexer.TokenBitOrAssign:  ASSIGN,
	lexer.TokenBitXorAssign: ASSIGN,
	lexer.TokenShlAssign:    ASSIGN,
	lexer.TokenShrAssign:    ASSIGN,

	// Ternary conditional (right associative)
	lexer.TokenQuestion: TERNARY,

	// Logical operators
	lexer.TokenOr:  LOGICAL_OR,
	lexer.TokenAnd: LOGICAL_AND,

	// Bitwise operators
	lexer.TokenBitOr:  BITWISE_OR,
	lexer.TokenBitXor: BITWISE_XOR,
	lexer.TokenBitAnd: BITWISE_AND,

	// Equality and relational operators
	lexer.TokenEq: EQUALS,
	lexer.TokenNe: EQUALS,
	lexer.TokenLt: LESSGREATER,
	lexer.TokenLe: LESSGREATER,
	lexer.TokenGt: LESSGREATER,
	lexer.TokenGe: LESSGREATER,

	// Shift operators
	lexer.TokenShl: SHIFT,
	lexer.TokenShr: SHIFT,

	// Additive operators
	lexer.TokenPlus:  SUM,
	lexer.TokenMinus: SUM,

	// Multiplicative operators
	lexer.TokenMul: PRODUCT,
	lexer.TokenDiv: PRODUCT,
	lexer.TokenMod: PRODUCT,

	// Power operator (right associative)
	lexer.TokenPower: POWER,

	// Call and access operators
	lexer.TokenLParen:   CALL,
	lexer.TokenLBracket: CALL,
	lexer.TokenDot:      CALL,
}

// operatorAssociativity maps precedence levels to their associativity
var operatorAssociativity = map[Precedence]Associativity{
	TERNARY:     RightAssociative,
	ASSIGN:      RightAssociative,
	LOGICAL_OR:  LeftAssociative,
	LOGICAL_AND: LeftAssociative,
	BITWISE_OR:  LeftAssociative,
	BITWISE_XOR: LeftAssociative,
	BITWISE_AND: LeftAssociative,
	EQUALS:      LeftAssociative,
	LESSGREATER: LeftAssociative,
	SHIFT:       LeftAssociative,
	SUM:         LeftAssociative,
	PRODUCT:     LeftAssociative,
	POWER:       RightAssociative, // Important: 2^3^4 = 2^(3^4)
	PREFIX:      RightAssociative,
	POSTFIX:     LeftAssociative,
	CALL:        LeftAssociative,
}

// peekPrecedence returns the precedence of the peek token
func (p *Parser) peekPrecedence() Precedence {
	if p, ok := precedences[p.peek.Type]; ok {
		return p
	}
	return LOWEST
}

// currentPrecedence returns the precedence of the current token
func (p *Parser) currentPrecedence() Precedence {
	if p, ok := precedences[p.current.Type]; ok {
		return p
	}
	return LOWEST
}

// parseExpression parses expressions using enhanced Pratt parsing with associativity
func (p *Parser) parseExpression(precedence Precedence) Expression {
	// Parse prefix expression
	left := p.parsePrefixExpression()
	if left == nil {
		return nil
	}

	// Parse infix expressions with associativity consideration
	for !p.peekTokenIs(lexer.TokenSemicolon) && p.shouldContinueParsing(precedence) {
		p.nextToken()
		left = p.parseInfixExpression(left)
		if left == nil {
			return nil
		}
	}

	return left
}

// shouldContinueParsing determines if parsing should continue based on precedence and associativity
func (p *Parser) shouldContinueParsing(precedence Precedence) bool {
	peekPrec := p.peekPrecedence()

	// If peek precedence is lower, don't continue
	if precedence > peekPrec {
		return false
	}

	// If precedences are equal, check associativity
	if precedence == peekPrec {
		assoc, exists := operatorAssociativity[peekPrec]
		if !exists {
			return precedence < peekPrec // Default to left associative
		}

		switch assoc {
		case LeftAssociative:
			return false // Stop for left associative
		case RightAssociative:
			return true // Continue for right associative
		case NonAssociative:
			// Non-associative operators like comparison chains
			p.addError(TokenToPosition(p.peek),
				"non-associative operator cannot be chained",
				"expression parsing")
			return false
		}
	}

	return precedence < peekPrec
}

// parsePrefixExpression parses prefix expressions with extended operator support
func (p *Parser) parsePrefixExpression() Expression {
	switch p.current.Type {
	case lexer.TokenIdentifier:
		// Check if this is a macro invocation pattern (identifier followed by !)
		if p.peek.Type == lexer.TokenMacroInvoke {
			return p.parseMacroInvocationWithIdent()
		}
		return p.parseIdentifier()
	case lexer.TokenInteger:
		return p.parseIntegerLiteral()
	case lexer.TokenFloat:
		return p.parseFloatLiteral()
	case lexer.TokenString:
		return p.parseStringLiteral()
	case lexer.TokenBool:
		return p.parseBooleanLiteral()
	// Unary prefix operators
	case lexer.TokenMinus, lexer.TokenNot, lexer.TokenBitNot:
		return p.parseUnaryExpression()
	case lexer.TokenLParen:
		return p.parseGroupedExpression()
	// Macro invocation starting with !
	case lexer.TokenMacroInvoke:
		return p.parseMacroInvocation()
	default:
		p.addError(TokenToPosition(p.current),
			fmt.Sprintf("no prefix parse function for %s", p.current.Type.String()),
			"expression parsing")
		return nil
	}
}

// parseIdentifier parses an identifier
func (p *Parser) parseIdentifier() Expression {
	return NewIdentifier(TokenToSpan(p.current), p.current.Literal)
}

// parseIntegerLiteral parses an integer literal
func (p *Parser) parseIntegerLiteral() Expression {
	value, err := strconv.ParseInt(p.current.Literal, 0, 64)
	if err != nil {
		p.addError(TokenToPosition(p.current),
			fmt.Sprintf("could not parse %q as integer", p.current.Literal),
			"integer parsing")
		return nil
	}

	return NewLiteral(TokenToSpan(p.current), value, LiteralInteger)
}

// parseFloatLiteral parses a float literal
func (p *Parser) parseFloatLiteral() Expression {
	value, err := strconv.ParseFloat(p.current.Literal, 64)
	if err != nil {
		p.addError(TokenToPosition(p.current),
			fmt.Sprintf("could not parse %q as float", p.current.Literal),
			"float parsing")
		return nil
	}

	return NewLiteral(TokenToSpan(p.current), value, LiteralFloat)
}

// parseStringLiteral parses a string literal
func (p *Parser) parseStringLiteral() Expression {
	return NewLiteral(TokenToSpan(p.current), p.current.Literal, LiteralString)
}

// parseBooleanLiteral parses a boolean literal
func (p *Parser) parseBooleanLiteral() Expression {
	value := p.current.Literal == "true"
	return NewLiteral(TokenToSpan(p.current), value, LiteralBool)
}

// parseUnaryExpression parses unary expressions
func (p *Parser) parseUnaryExpression() Expression {
	startPos := TokenToPosition(p.current)
	operator := NewOperator(TokenToSpan(p.current), p.current.Literal, 0, RightAssociative, UnaryOp)

	p.nextToken()
	operand := p.parseExpression(PREFIX)

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &UnaryExpression{
		Span:     span,
		Operator: operator,
		Operand:  operand,
	}
}

// parseGroupedExpression parses grouped expressions
func (p *Parser) parseGroupedExpression() Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TokenRParen) {
		return nil
	}

	return exp
}

// parseBinaryExpression parses binary expressions
func (p *Parser) parseBinaryExpression(left Expression) Expression {
	startPos := left.GetSpan().Start
	operator := NewOperator(TokenToSpan(p.current), p.current.Literal,
		int(p.currentPrecedence()), LeftAssociative, BinaryOp)

	precedence := p.currentPrecedence()
	p.nextToken()
	right := p.parseExpression(precedence)

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &BinaryExpression{
		Span:     span,
		Left:     left,
		Operator: operator,
		Right:    right,
	}
}

// parseCallExpression parses function call expressions
func (p *Parser) parseCallExpression(function Expression) Expression {
	startPos := function.GetSpan().Start
	arguments := p.parseCallArguments()

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &CallExpression{
		Span:      span,
		Function:  function,
		Arguments: arguments,
	}
}

// parseCallArguments parses function call arguments
func (p *Parser) parseCallArguments() []Expression {
	args := make([]Expression, 0)

	if p.peekTokenIs(lexer.TokenRParen) {
		p.nextToken()
		return args
	}

	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.TokenComma) {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(lexer.TokenRParen) {
		return nil
	}

	return args
}

// parseAssignmentExpression parses assignment expressions
func (p *Parser) parseAssignmentExpression(left Expression) Expression {
	startPos := left.GetSpan().Start
	operator := NewOperator(TokenToSpan(p.current), p.current.Literal, 0, RightAssociative, AssignmentOp)

	p.nextToken()
	right := p.parseExpression(ASSIGN) // Right associative

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &AssignmentExpression{
		Span:     span,
		Left:     left,
		Operator: operator,
		Right:    right,
	}
}

// parsePowerExpression parses power expressions (right associative)
func (p *Parser) parsePowerExpression(left Expression) Expression {
	startPos := left.GetSpan().Start
	operator := NewOperator(TokenToSpan(p.current), p.current.Literal,
		int(p.currentPrecedence()), RightAssociative, BinaryOp)

	precedence := p.currentPrecedence()
	p.nextToken()
	// Right associative: use same precedence - 1 to parse right operand
	right := p.parseExpression(precedence - 1)

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &BinaryExpression{
		Span:     span,
		Left:     left,
		Operator: operator,
		Right:    right,
	}
}

// parseCompoundAssignmentExpression parses compound assignment expressions (+=, -=, etc.)
func (p *Parser) parseCompoundAssignmentExpression(left Expression) Expression {
	startPos := left.GetSpan().Start
	operator := NewOperator(TokenToSpan(p.current), p.current.Literal, 0, RightAssociative, AssignmentOp)

	p.nextToken()
	right := p.parseExpression(ASSIGN) // Right associative

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &AssignmentExpression{
		Span:     span,
		Left:     left,
		Operator: operator,
		Right:    right,
	}
}

// parseIndexExpression parses array/slice index expressions
func (p *Parser) parseIndexExpression(left Expression) Expression {
	startPos := left.GetSpan().Start

	p.nextToken() // consume [
	index := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TokenRBracket) {
		return nil
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	// Create a specialized index expression using CallExpression structure
	return &CallExpression{
		Span:      span,
		Function:  left,
		Arguments: []Expression{index},
	}
}

// parseMemberExpression parses member access expressions (obj.field)
func (p *Parser) parseMemberExpression(left Expression) Expression {
	startPos := left.GetSpan().Start

	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	member := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	// Create a specialized member expression using BinaryExpression structure
	operator := NewOperator(Span{Start: startPos, End: endPos}, ".",
		int(CALL), LeftAssociative, BinaryOp)

	return &BinaryExpression{
		Span:     span,
		Left:     left,
		Operator: operator,
		Right:    member,
	}
}

// parseTernaryExpression parses ternary conditional expressions (condition ? true_expr : false_expr)
func (p *Parser) parseTernaryExpression(left Expression) Expression {
	startPos := left.GetSpan().Start

	p.nextToken() // consume ?
	trueExpr := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TokenColon) {
		return nil
	}

	p.nextToken()                               // consume :
	falseExpr := p.parseExpression(TERNARY - 1) // Right associative

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	// Create ternary expression using specialized AST node
	return &TernaryExpression{
		Span:      span,
		Condition: left,
		TrueExpr:  trueExpr,
		FalseExpr: falseExpr,
	}
}

// parseMacroDeclaration parses a macro definition
func (p *Parser) parseMacroDeclaration() *MacroDefinition {
	startPos := TokenToPosition(p.current)

	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	// Parse parameters if present
	var params []*MacroParameter
	if p.peek.Type == lexer.TokenLParen {
		p.nextToken()
		params = p.parseMacroParameters()
		if !p.expectPeek(lexer.TokenRParen) {
			return nil
		}
	}

	// Parse macro body
	if !p.expectPeek(lexer.TokenLBrace) {
		return nil
	}

	body := p.parseMacroBody()
	if body == nil {
		return nil
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &MacroDefinition{
		Span:       span,
		Name:       name,
		Parameters: params,
		Body:       body,
		IsHygienic: true, // Default to hygienic macros
	}
}

// parseMacroParameters parses macro parameter list
func (p *Parser) parseMacroParameters() []*MacroParameter {
	var params []*MacroParameter

	if p.peek.Type == lexer.TokenRParen {
		return params
	}

	p.nextToken()
	for {
		param := p.parseMacroParameter()
		if param != nil {
			params = append(params, param)
		}

		if p.peek.Type != lexer.TokenComma {
			break
		}
		p.nextToken() // consume comma
		p.nextToken() // move to next parameter
	}

	return params
}

// parseMacroParameter parses a single macro parameter
func (p *Parser) parseMacroParameter() *MacroParameter {
	if p.current.Type != lexer.TokenIdentifier {
		p.addError(TokenToPosition(p.current),
			"expected parameter name",
			"macro parameter parsing")
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	// Check for default value
	var defaultValue Expression
	if p.peek.Type == lexer.TokenAssign {
		p.nextToken() // consume =
		p.nextToken() // move to value
		defaultValue = p.parseExpression(LOWEST)
	}

	return &MacroParameter{
		Span:         name.Span,
		Name:         name,
		Constraint:   nil, // Type constraints not implemented yet
		DefaultValue: defaultValue,
	}
}

// parseMacroBody parses the body of a macro definition
func (p *Parser) parseMacroBody() *MacroBody {
	startPos := TokenToPosition(p.current)
	var templates []*MacroTemplate

	p.nextToken() // consume {

	for p.current.Type != lexer.TokenRBrace && p.current.Type != lexer.TokenEOF {
		template := p.parseMacroTemplate()
		if template != nil {
			templates = append(templates, template)
		}
		p.nextToken()
	}

	if p.current.Type != lexer.TokenRBrace {
		p.addError(TokenToPosition(p.current),
			"expected '}' to close macro body",
			"macro body parsing")
		return nil
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &MacroBody{
		Span:      span,
		Templates: templates,
	}
}

// parseMacroTemplate parses a single macro template
func (p *Parser) parseMacroTemplate() *MacroTemplate {
	startPos := TokenToPosition(p.current)

	// Parse pattern
	pattern := p.parseMacroPattern()
	if pattern == nil {
		return nil
	}

	// Parse guard if present
	var guard Expression
	if p.current.Type == lexer.TokenIf {
		p.nextToken()
		guard = p.parseExpression(LOWEST)
	}

	// Parse arrow
	if !p.expectPeek(lexer.TokenArrow) {
		return nil
	}

	// Parse template body
	p.nextToken()
	var body []Statement

	if p.current.Type == lexer.TokenLBrace {
		// Block body
		blockStmt := p.parseBlockStatement()
		if blockStmt != nil {
			body = blockStmt.Statements
		}
	} else {
		// Single statement or expression
		stmt := p.parseStatement()
		if stmt != nil {
			body = append(body, stmt)
		}
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &MacroTemplate{
		Span:     span,
		Pattern:  pattern,
		Guard:    guard,
		Body:     body,
		Priority: 0, // Default priority
	}
}

// parseMacroPattern parses a macro pattern
func (p *Parser) parseMacroPattern() *MacroPattern {
	startPos := TokenToPosition(p.current)
	var elements []*MacroPatternElement

	// For now, implement basic pattern parsing
	for p.current.Type != lexer.TokenArrow && p.current.Type != lexer.TokenIf && p.current.Type != lexer.TokenEOF {
		element := p.parseMacroPatternElement()
		if element != nil {
			elements = append(elements, element)
		}
		p.nextToken()
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &MacroPattern{
		Span:     span,
		Elements: elements,
	}
}

// parseMacroPatternElement parses a single pattern element
func (p *Parser) parseMacroPatternElement() *MacroPatternElement {
	switch p.current.Type {
	case lexer.TokenIdentifier:
		// Parameter pattern
		return &MacroPatternElement{
			Span:  TokenToSpan(p.current),
			Kind:  MacroPatternParameter,
			Value: p.current.Literal,
		}
	case lexer.TokenMul:
		// Wildcard pattern
		return &MacroPatternElement{
			Span: TokenToSpan(p.current),
			Kind: MacroPatternWildcard,
		}
	default:
		// Literal pattern
		return &MacroPatternElement{
			Span:  TokenToSpan(p.current),
			Kind:  MacroPatternLiteral,
			Value: p.current.Literal,
		}
	}
}

// parseMacroInvocation parses a macro invocation expression
func (p *Parser) parseMacroInvocation() *MacroInvocation {
	startPos := TokenToPosition(p.current)

	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	// Parse arguments
	if !p.expectPeek(lexer.TokenLParen) {
		return nil
	}

	args := p.parseMacroArguments()

	if !p.expectPeek(lexer.TokenRParen) {
		return nil
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &MacroInvocation{
		Span:      span,
		Name:      name,
		Arguments: args,
	}
}

// parseMacroArguments parses macro invocation arguments
func (p *Parser) parseMacroArguments() []*MacroArgument {
	var args []*MacroArgument

	if p.peek.Type == lexer.TokenRParen {
		return args
	}

	p.nextToken()
	for {
		arg := p.parseMacroArgument()
		if arg != nil {
			args = append(args, arg)
		}

		if p.peek.Type != lexer.TokenComma {
			break
		}
		p.nextToken() // consume comma
		p.nextToken() // move to next argument
	}

	return args
}

// parseMacroArgument parses a single macro argument
func (p *Parser) parseMacroArgument() *MacroArgument {
	startPos := TokenToPosition(p.current)

	// For now, treat all arguments as expressions
	expr := p.parseExpression(LOWEST)
	if expr == nil {
		return nil
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &MacroArgument{
		Span:  span,
		Kind:  MacroArgExpression,
		Value: expr,
	}
}

// parseMacroInvocationWithIdent parses macro invocation in the form: identifier!(args)
func (p *Parser) parseMacroInvocationWithIdent() *MacroInvocation {
	startPos := TokenToPosition(p.current)

	// Current token is identifier, next should be !
	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	if !p.expectPeek(lexer.TokenMacroInvoke) {
		return nil
	}

	// Parse arguments
	if !p.expectPeek(lexer.TokenLParen) {
		return nil
	}

	args := p.parseMacroArguments()

	if !p.expectPeek(lexer.TokenRParen) {
		return nil
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &MacroInvocation{
		Span:      span,
		Name:      name,
		Arguments: args,
	}
}
