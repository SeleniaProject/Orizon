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
		lexer:    l,
		filename: filename,
		errors:   make([]error, 0),
	}

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
	return false
}

// peekError records a peek token mismatch error
func (p *Parser) peekError(expected lexer.TokenType) {
	msg := fmt.Sprintf("expected %s, got %s", expected.String(), p.peek.Type.String())
	p.addError(TokenToPosition(p.peek), msg, "token mismatch")
}

// addError adds an error to the parser's error list
func (p *Parser) addError(pos Position, message, context string) {
	pos.File = p.filename
	p.errors = append(p.errors, &ParseError{
		Position: pos,
		Message:  message,
		Context:  context,
	})
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
	switch p.current.Type {
	case lexer.TokenFunc:
		return p.parseFunctionDeclaration()
	case lexer.TokenLet, lexer.TokenVar, lexer.TokenConst:
		return p.parseVariableDeclaration()
	default:
		// Try to parse as expression statement
		stmt := p.parseExpressionStatement()
		if stmt != nil {
			return stmt
		}
		p.addError(TokenToPosition(p.current),
			fmt.Sprintf("unexpected token %s in declaration", p.current.Type.String()),
			"declaration parsing")
		p.skipTo(lexer.TokenFunc, lexer.TokenLet, lexer.TokenVar, lexer.TokenConst, lexer.TokenEOF)
		return nil
	}
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
		IsPublic:   false, // TODO: handle pub modifier
		IsAsync:    false, // TODO: handle async modifier
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
		IsMut:    false, // TODO: handle mut modifier
	}
}

// parseVariableDeclaration parses a variable declaration
func (p *Parser) parseVariableDeclaration() *VariableDeclaration {
	startPos := TokenToPosition(p.current)

	isMutable := p.currentTokenIs(lexer.TokenVar)

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
		IsPublic:    false, // TODO: handle pub modifier
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
		return p.parseExpressionStatement()
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

	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
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
	TERNARY     // ? :
	ASSIGN      // = += -= *= /= %= &= |= ^= <<= >>=
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
	POWER       // ** (right associative)
	PREFIX      // -X !X ~X
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
