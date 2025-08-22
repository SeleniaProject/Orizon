// Package parser implements the Orizon recursive descent parser.
// Phase 1.2.1: 再帰下降パーサー実装
package parser

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/orizon-lang/orizon/internal/lexer"
)

// Parser represents the recursive descent parser with performance monitoring.
type Parser struct {
	lexer                *lexer.Lexer
	suggestionEngine     *SuggestionEngine
	filename             string
	errors               []error
	suggestions          []Suggestion
	current              lexer.Token
	peek                 lexer.Token
	recoveryMode         ErrorRecoveryMode
	maxErrors            int
	maxSuggestionsTotal  int
	errorsTruncated      bool
	suggestionsTruncated bool

	// Performance monitoring
	nodeCount       int   // Total AST nodes created
	tokenCount      int   // Total tokens processed
	memoryAllocated int64 // Estimated memory usage
	enableProfiling bool  // Whether to collect performance metrics

	// Memory pools for performance optimization
	stringPool    *sync.Pool // Pool for string builders
	slicePool     *sync.Pool // Pool for slices
	identifierPool *sync.Pool // Pool for identifier nodes
	literalPool    *sync.Pool // Pool for literal nodes
}

// ParseError represents a parsing error with enhanced context and recovery hints.
type ParseError struct {
	Message      string
	Context      string
	Position     Position
	TokenFound   string // The actual token that was found
	TokenWanted  string // The token that was expected
	Severity     ErrorSeverity
	RecoveryHint string // Suggestion for fixing the error
}

// ErrorSeverity indicates the severity level of parsing errors.
type ErrorSeverity int

const (
	SeverityError ErrorSeverity = iota
	SeverityWarning
	SeverityInfo
)

func (e *ParseError) Error() string {
	if e.RecoveryHint != "" {
		return fmt.Sprintf("Parse error at %s: %s (hint: %s)", e.Position.String(), e.Message, e.RecoveryHint)
	}
	return fmt.Sprintf("Parse error at %s: %s", e.Position.String(), e.Message)
}

// GetDetailedError returns a more detailed error message for IDE integration.
func (e *ParseError) GetDetailedError() string {
	var details strings.Builder
	details.WriteString(fmt.Sprintf("Parse error at %s:\n", e.Position.String()))
	details.WriteString(fmt.Sprintf("  Message: %s\n", e.Message))
	if e.Context != "" {
		details.WriteString(fmt.Sprintf("  Context: %s\n", e.Context))
	}
	if e.TokenFound != "" && e.TokenWanted != "" {
		details.WriteString(fmt.Sprintf("  Expected: %s, Found: %s\n", e.TokenWanted, e.TokenFound))
	}
	if e.RecoveryHint != "" {
		details.WriteString(fmt.Sprintf("  Suggestion: %s\n", e.RecoveryHint))
	}
	return details.String()
}

// parseInfixExpression parses infix expressions with proper nil checking.
func (p *Parser) parseInfixExpression(left Expression) Expression {
	// Critical nil check: prevent nil pointer dereference
	if left == nil {
		p.addError(TokenToPosition(p.current),
			"invalid left operand for infix expression",
			"Expected valid expression before operator")
		// Return nil rather than continuing with invalid state
		return nil
	}

	switch p.current.Type {
	// Arithmetic operators.
	case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenMul, lexer.TokenDiv, lexer.TokenMod:
		return p.parseBinaryExpression(left)
	// Power operator (right associative).
	case lexer.TokenPower:
		return p.parsePowerExpression(left)
	// Comparison operators.
	case lexer.TokenEq, lexer.TokenNe, lexer.TokenLt, lexer.TokenLe, lexer.TokenGt, lexer.TokenGe:
		return p.parseBinaryExpression(left)
	// Logical operators.
	case lexer.TokenAnd, lexer.TokenOr:
		return p.parseBinaryExpression(left)
	// Bitwise operators.
	case lexer.TokenBitAnd, lexer.TokenBitOr, lexer.TokenBitXor, lexer.TokenShl, lexer.TokenShr:
		return p.parseBinaryExpression(left)
	// Assignment operators.
	case lexer.TokenAssign:
		return p.parseAssignmentExpression(left)
	case lexer.TokenPlusAssign, lexer.TokenMinusAssign, lexer.TokenMulAssign,
		lexer.TokenDivAssign, lexer.TokenModAssign:
		return p.parseCompoundAssignmentExpression(left)
	case lexer.TokenBitAndAssign, lexer.TokenBitOrAssign, lexer.TokenBitXorAssign,
		lexer.TokenShlAssign, lexer.TokenShrAssign:
		return p.parseCompoundAssignmentExpression(left)
	// Call and access operators.
	case lexer.TokenLParen:
		return p.parseCallExpression(left)
	case lexer.TokenLBracket:
		return p.parseIndexExpression(left)
	case lexer.TokenDot:
		return p.parseMemberExpression(left)
	// Ternary conditional operator.
	case lexer.TokenQuestion:
		return p.parseTernaryExpression(left)
	// Range operator.
	case lexer.TokenRange:
		return p.parseRangeInfixExpression(left)
	default:
		return nil
	}
}

// NewParser creates a new parser instance with enhanced performance monitoring.
func NewParser(l *lexer.Lexer, filename string) *Parser {
	p := &Parser{
		lexer:        l,
		filename:     filename,
		errors:       make([]error, 0),
		suggestions:  make([]Suggestion, 0),
		recoveryMode: PhraseLevel, // Default to phrase-level recovery
		// Set conservative yet safe caps to prevent runaway memory usage.
		maxErrors:           5000,
		maxSuggestionsTotal: 5000,
		// Performance monitoring initialization
		nodeCount:       0,
		tokenCount:      0,
		memoryAllocated: 0,
		enableProfiling: true, // Enable by default for development

		// Initialize memory pools for performance optimization
		stringPool: &sync.Pool{
			New: func() interface{} {
				return &strings.Builder{}
			},
		},
		slicePool: &sync.Pool{
			New: func() interface{} {
				return make([]interface{}, 0, 16)
			},
		},
		identifierPool: &sync.Pool{
			New: func() interface{} {
				return &Identifier{}
			},
		},
		literalPool: &sync.Pool{
			New: func() interface{} {
				return &Literal{}
			},
		},
	}

	// Initialize suggestion engine with phrase-level recovery.
	p.suggestionEngine = NewSuggestionEngine(p.recoveryMode)

	// Read the first two tokens.
	p.nextToken()
	p.nextToken()

	return p
}

// Parse parses the input and returns an AST with performance tracking.
func (p *Parser) Parse() (*Program, []error) {
	program := p.parseProgram()
	return program, p.errors
}

// ParseStats represents parsing performance statistics.
type ParseStats struct {
	TokensProcessed      int
	NodesCreated         int
	ErrorsGenerated      int
	SuggestionsCreated   int
	MemoryAllocated      int64
	ErrorsTruncated      bool
	SuggestionsTruncated bool
}

// GetParseStats returns performance statistics for the current parse session.
func (p *Parser) GetParseStats() ParseStats {
	return ParseStats{
		TokensProcessed:      p.tokenCount,
		NodesCreated:         p.nodeCount,
		ErrorsGenerated:      len(p.errors),
		SuggestionsCreated:   len(p.suggestions),
		MemoryAllocated:      p.memoryAllocated,
		ErrorsTruncated:      p.errorsTruncated,
		SuggestionsTruncated: p.suggestionsTruncated,
	}
}

// ResetStats resets all performance counters.
func (p *Parser) ResetStats() {
	p.tokenCount = 0
	p.nodeCount = 0
	p.memoryAllocated = 0
	p.errorsTruncated = false
	p.suggestionsTruncated = false
}

// SetProfiling enables or disables performance profiling.
func (p *Parser) SetProfiling(enabled bool) {
	p.enableProfiling = enabled
}

// newPooledIdentifier creates a new identifier using memory pool for performance optimization.
func (p *Parser) newPooledIdentifier(span Span, value string) *Identifier {
	// Get identifier from pool
	ident := p.identifierPool.Get().(*Identifier)

	// Reset and set values
	ident.Span = span
	ident.Value = value

	// Performance monitoring
	if p.enableProfiling {
		p.nodeCount++
		p.memoryAllocated += int64(len(value) + 16) // estimate memory usage
	}

	return ident
}

// returnPooledIdentifier returns an identifier to the pool for reuse.
func (p *Parser) returnPooledIdentifier(ident *Identifier) {
	// Reset fields before returning to pool
	ident.Value = ""
	ident.Span = Span{}
	p.identifierPool.Put(ident)
}

// newPooledStringBuilder creates a string builder from pool for efficient string operations.
func (p *Parser) newPooledStringBuilder() *strings.Builder {
	return p.stringPool.Get().(*strings.Builder)
}

// returnPooledStringBuilder returns string builder to pool.
func (p *Parser) returnPooledStringBuilder(sb *strings.Builder) {
	sb.Reset()
	p.stringPool.Put(sb)
}

// newPooledSlice creates a slice from pool for efficient slice operations.
func (p *Parser) newPooledSlice() []interface{} {
	return p.slicePool.Get().([]interface{})
}

// returnPooledSlice returns slice to pool.
func (p *Parser) returnPooledSlice(slice []interface{}) {
	// Clear slice before returning
	for i := range slice {
		slice[i] = nil
	}
	slice = slice[:0]
	p.slicePool.Put(slice)
}

// newPooledLiteral creates a new literal using memory pool for performance optimization.
func (p *Parser) newPooledLiteral(span Span, value interface{}, kind LiteralKind) *Literal {
	// Get literal from pool
	lit := p.literalPool.Get().(*Literal)

	// Reset and set values
	lit.Span = span
	lit.Value = value
	lit.Kind = kind

	// Performance monitoring
	if p.enableProfiling {
		p.nodeCount++
		p.memoryAllocated += int64(24) // estimate memory usage for literal
	}

	return lit
}

// returnPooledLiteral returns a literal to the pool for reuse.
func (p *Parser) returnPooledLiteral(lit *Literal) {
	// Reset fields before returning to pool
	lit.Value = nil
	lit.Span = Span{}
	lit.Kind = 0
	p.literalPool.Put(lit)
}

// nextToken advances the parser to the next token with optimized trivia skipping.
func (p *Parser) nextToken() {
	p.current = p.peek
	p.peek = p.lexer.NextToken()

	// Performance monitoring: track token processing
	if p.enableProfiling {
		p.tokenCount++
	}

	// Optimization: Skip insignificant trivia tokens efficiently
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment {
		p.peek = p.lexer.NextToken()
		if p.enableProfiling {
			p.tokenCount++
		}
	}
}

// currentTokenIs checks if the current token is of the given type.
func (p *Parser) currentTokenIs(tokenType lexer.TokenType) bool {
	return p.current.Type == tokenType
}

// peekTokenIs checks if the peek token is of the given type.
func (p *Parser) peekTokenIs(tokenType lexer.TokenType) bool {
	return p.peek.Type == tokenType
}

// expectPeek advances if the peek token matches the expected type.
func (p *Parser) expectPeek(tokenType lexer.TokenType) bool {
	// Optimized trivia skipping: only handle newlines as they are structurally significant
	for p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if p.peekTokenIs(tokenType) {
		p.nextToken()
		return true
	}

	p.peekError(tokenType)

	// Attempt error recovery if enabled (Phase 1.2.4)
	if p.recoveryMode != PanicMode && p.suggestionEngine != nil {
		// Track current token to detect lack of progress.
		before := p.current
		_ = p.recoverFromError(fmt.Sprintf("expecting %s", tokenType.String()))
		// If recovery positioned the peek at expected, consume it.
		if p.peekTokenIs(tokenType) {
			p.nextToken()

			return true
		}
		// If nothing changed, advance one token to avoid infinite loops.
		if p.current == before && p.current.Type != lexer.TokenEOF {
			p.nextToken()
		}
	}

	return false
}

// expectPeekNoRecover advances if the peek token matches, without invoking recovery.
func (p *Parser) expectPeekNoRecover(tokenType lexer.TokenType) bool {
	// Optimized trivia skipping: only handle newlines as they are structurally significant
	for p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if p.peekTokenIs(tokenType) {
		p.nextToken()
		return true
	}
	// Record error but do not attempt recovery here.
	p.peekError(tokenType)
	return false
}

// peekError records a peek token mismatch error with detailed information.
func (p *Parser) peekError(expected lexer.TokenType) {
	msg := fmt.Sprintf("expected %s, got %s", expected.String(), p.peek.Type.String())

	// Generate helpful recovery hint based on common mistakes
	recoveryHint := p.generateRecoveryHint(expected, p.peek.Type)

	// Track expected token for suggestion engine (Phase 1.2.4)
	if p.suggestionEngine != nil {
		p.suggestionEngine.AddExpectedToken(expected)
	}

	p.addDetailedError(TokenToPosition(p.peek), msg, "token mismatch",
		p.peek.Type.String(), expected.String(), SeverityError, recoveryHint)
}

// generateRecoveryHint provides helpful suggestions based on common parsing errors.
func (p *Parser) generateRecoveryHint(expected, found lexer.TokenType) string {
	switch {
	case expected == lexer.TokenSemicolon && found == lexer.TokenNewline:
		return "Try adding a semicolon before the newline"
	case expected == lexer.TokenRParen && found == lexer.TokenEOF:
		return "Missing closing parenthesis ')'"
	case expected == lexer.TokenRBrace && found == lexer.TokenEOF:
		return "Missing closing brace '}'"
	case expected == lexer.TokenRBracket && found == lexer.TokenEOF:
		return "Missing closing bracket ']'"
	case expected == lexer.TokenIdentifier && found == lexer.TokenInteger:
		return "Expected an identifier, not a number"
	case expected == lexer.TokenLParen && found == lexer.TokenLBrace:
		return "Did you mean '(' instead of '{'?"
	case expected == lexer.TokenAssign && found == lexer.TokenEq:
		return "Use '=' for assignment, not '=='"
	default:
		return ""
	}
}

// addError adds an error to the parser's error list.
func (p *Parser) addError(pos Position, message, context string) {
	p.addDetailedError(pos, message, context, "", "", SeverityError, "")
}

// addDetailedError adds a detailed error with enhanced information.
func (p *Parser) addDetailedError(pos Position, message, context, tokenFound, tokenWanted string, severity ErrorSeverity, recoveryHint string) {
	// Respect error cap to limit memory under heavy error conditions.
	if p.maxErrors > 0 && len(p.errors) >= p.maxErrors {
		p.errorsTruncated = true
		// Also stop generating suggestions to save memory/CPU
		p.suggestionEngine = nil
		return
	}

	pos.File = p.filename
	parseErr := &ParseError{
		Position:     pos,
		Message:      message,
		Context:      context,
		TokenFound:   tokenFound,
		TokenWanted:  tokenWanted,
		Severity:     severity,
		RecoveryHint: recoveryHint,
	}
	p.errors = append(p.errors, parseErr)

	// Generate suggestions using the error recovery system (Phase 1.2.4)
	if p.suggestionEngine != nil {
		// Respect suggestions cap.
		if p.maxSuggestionsTotal > 0 && len(p.suggestions) >= p.maxSuggestionsTotal {
			p.suggestionsTruncated = true
			p.suggestionEngine = nil
		} else {
			newSuggestions := p.suggestionEngine.RecoverFromError(p, parseErr)
			// Ensure we don't exceed the cap even if engine returns up to max per call.
			if p.maxSuggestionsTotal > 0 && len(p.suggestions)+len(newSuggestions) > p.maxSuggestionsTotal {
				remain := p.maxSuggestionsTotal - len(p.suggestions)
				if remain > 0 {
					p.suggestions = append(p.suggestions, newSuggestions[:remain]...)
				}

				p.suggestionsTruncated = true
				p.suggestionEngine = nil
			} else {
				p.suggestions = append(p.suggestions, newSuggestions...)
			}
		}
	}
}

// addErrorWithSuggestion adds an error with manual suggestions.
func (p *Parser) addErrorWithSuggestion(pos Position, message, context string, suggestions []Suggestion) {
	if p.maxErrors > 0 && len(p.errors) >= p.maxErrors {
		p.errorsTruncated = true

		return
	}

	pos.File = p.filename
	parseErr := &ParseError{
		Position: pos,
		Message:  message,
		Context:  context,
	}

	p.errors = append(p.errors, parseErr)
	if p.maxSuggestionsTotal > 0 {
		space := p.maxSuggestionsTotal - len(p.suggestions)
		if space <= 0 {
			p.suggestionsTruncated = true

			return
		}

		if len(suggestions) > space {
			p.suggestions = append(p.suggestions, suggestions[:space]...)
			p.suggestionsTruncated = true

			return
		}
	}

	p.suggestions = append(p.suggestions, suggestions...)
}

// addErrorSilent adds an error without invoking the suggestion engine.
func (p *Parser) addErrorSilent(pos Position, message, context string) {
	if p.maxErrors > 0 && len(p.errors) >= p.maxErrors {
		p.errorsTruncated = true

		return
	}

	pos.File = p.filename
	parseErr := &ParseError{
		Position: pos,
		Message:  message,
		Context:  context,
	}
	p.errors = append(p.errors, parseErr)
}

// expectPeekRaw advances if the peek matches; does not record errors or suggestions.
func (p *Parser) expectPeekRaw(tokenType lexer.TokenType) bool {
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if p.peekTokenIs(tokenType) {
		p.nextToken()

		return true
	}

	return false
}

// recoverFromError attempts intelligent error recovery.
func (p *Parser) recoverFromError(expectedContext string) bool {
	if p.suggestionEngine == nil {
		return false
	}

	// Create a temporary error for recovery analysis.
	tempErr := &ParseError{
		Position: TokenToPosition(p.current),
		Message:  fmt.Sprintf("unexpected token in %s", expectedContext),
		Context:  expectedContext,
	}

	// Let the suggestion engine handle recovery.
	suggestions := p.suggestionEngine.RecoverFromError(p, tempErr)
	p.suggestions = append(p.suggestions, suggestions...)

	// Return true if recovery was successful (parser position changed).
	return true
}

// GetSuggestions returns all accumulated suggestions.
func (p *Parser) GetSuggestions() []Suggestion {
	return p.suggestions
}

// SetRecoveryMode changes the error recovery strategy.
func (p *Parser) SetRecoveryMode(mode ErrorRecoveryMode) {
	p.recoveryMode = mode
	if p.suggestionEngine != nil {
		p.suggestionEngine.mode = mode
	}
}

// SetErrorLimit sets a maximum number of errors to retain; 0 or negative disables the cap.
func (p *Parser) SetErrorLimit(n int) {
	p.maxErrors = n
}

// SetSuggestionLimit sets a maximum number of suggestions to retain; 0 or negative disables the cap.
func (p *Parser) SetSuggestionLimit(n int) {
	p.maxSuggestionsTotal = n
}

// DisableSuggestions turns off the suggestion engine entirely (saves memory/CPU).
func (p *Parser) DisableSuggestions() {
	p.suggestionEngine = nil
}

// skipTo skips tokens until one of the given types is found.
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

// skipToBefore advances tokens until the next token (peek) matches one of the given types,.
// leaving current at the token right before the target so outer loops that call nextToken().
// will land exactly on the target.
func (p *Parser) skipToBefore(tokenTypes ...lexer.TokenType) {
	for !p.currentTokenIs(lexer.TokenEOF) {
		// If the next token is one of the targets, stop here.
		for _, tokenType := range tokenTypes {
			if p.peekTokenIs(tokenType) {
				return
			}
		}

		p.nextToken()
	}
}

// skipLine advances the parser to the beginning of the next line.
// by consuming tokens until a NEWLINE (inclusive) or EOF is reached.
func (p *Parser) skipLine() {
	for !p.currentTokenIs(lexer.TokenEOF) {
		if p.currentTokenIs(lexer.TokenNewline) {
			// Move to the first token on the next line.
			p.nextToken()

			return
		}

		p.nextToken()
	}
}

// skipToNextTopLevelDecl advances to the next top-level declaration keyword.
// (func/let/var/const/macro/struct/enum) that appears at the start of a line.
func (p *Parser) skipToNextTopLevelDecl() {
	for !p.currentTokenIs(lexer.TokenEOF) {
		if p.current.Type == lexer.TokenFunc || p.current.Type == lexer.TokenLet || p.current.Type == lexer.TokenVar ||
			p.current.Type == lexer.TokenConst || p.current.Type == lexer.TokenMacro || p.current.Type == lexer.TokenStruct ||
			p.current.Type == lexer.TokenEnum || p.current.Type == lexer.TokenTrait || p.current.Type == lexer.TokenImpl ||
			p.current.Type == lexer.TokenImport || p.current.Type == lexer.TokenExport ||
			p.current.Type == lexer.TokenIdentifier && p.current.Literal == "type" {
			return
		}
		// If the next token is a declaration starter, stop before it.
		if p.peekTokenIs(lexer.TokenFunc) || p.peekTokenIs(lexer.TokenLet) || p.peekTokenIs(lexer.TokenVar) ||
			p.peekTokenIs(lexer.TokenConst) || p.peekTokenIs(lexer.TokenMacro) || p.peekTokenIs(lexer.TokenStruct) ||
			p.peekTokenIs(lexer.TokenEnum) || p.peekTokenIs(lexer.TokenTrait) || p.peekTokenIs(lexer.TokenImpl) ||
			p.peekTokenIs(lexer.TokenImport) || p.peekTokenIs(lexer.TokenExport) ||
			(p.peekTokenIs(lexer.TokenIdentifier) && p.peek.Literal == "type") {
			// Advance onto the declaration start to ensure forward progress.
			p.nextToken()

			return
		}

		if p.peekTokenIs(lexer.TokenEOF) {
			// Advance to EOF to guarantee forward progress.
			p.nextToken()

			return
		}

		p.nextToken()
	}
}

// isTopLevelStart reports whether a token starts a top-level declaration (used for resynchronization).
func (p *Parser) isTopLevelStart(tok lexer.Token) bool {
	if tok.Type == lexer.TokenFunc || tok.Type == lexer.TokenLet || tok.Type == lexer.TokenVar || tok.Type == lexer.TokenConst ||
		tok.Type == lexer.TokenMacro || tok.Type == lexer.TokenStruct || tok.Type == lexer.TokenEnum || tok.Type == lexer.TokenTrait ||
		tok.Type == lexer.TokenImpl || tok.Type == lexer.TokenImport || tok.Type == lexer.TokenExport || tok.Type == lexer.TokenEffect {
		return true
	}

	if tok.Type == lexer.TokenIdentifier && (tok.Literal == "type" || tok.Literal == "newtype" || tok.Literal == "fn") {
		return true
	}

	return false
}

// ====== Grammar Rules ======.

// parseProgram parses the entire program with optimized memory allocation.
func (p *Parser) parseProgram() *Program {
	startPos := TokenToPosition(p.current)

	// Pre-allocate with estimated capacity to reduce allocations
	// Most programs have 10-100 top-level declarations
	const estimatedDeclarations = 32
	declarations := make([]Declaration, 0, estimatedDeclarations)

	for !p.currentTokenIs(lexer.TokenEOF) {
		// Skip whitespace and comments.
		if p.currentTokenIs(lexer.TokenWhitespace) || p.currentTokenIs(lexer.TokenComment) || p.currentTokenIs(lexer.TokenNewline) {
			p.nextToken()
			continue
		}

		// Parse declaration.
		if decl := p.parseDeclaration(); decl != nil {
			declarations = append(declarations, decl)
			// After a successful declaration, we're at the closing token of the decl.
			// Advance once to move past it and continue.
			p.nextToken()
		} else {
			// Declaration failed. Prefer declaration-level resynchronization to avoid swallowing following items.
			// If we're already at a top-level start token, don't consume it; retry parsing it on next loop.
			if !p.isTopLevelStart(p.current) {
				// Move to the next top-level declaration keyword or EOF.
				p.skipToNextTopLevelDecl()
			}
			continue
		}
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	program := NewProgram(span, declarations)

	// Performance monitoring: track program node creation
	if p.enableProfiling {
		p.nodeCount++                    // Count the program node
		p.nodeCount += len(declarations) // Count all declarations
		// Estimate memory usage (rough approximation)
		p.memoryAllocated += int64(len(declarations) * 64) // ~64 bytes per declaration
	}

	return program
}

// parseDeclaration parses a top-level declaration.
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
	case lexer.TokenFunc, lexer.TokenFn:
		fn := p.parseFunctionDeclaration()
		if fn == nil {
			// Let parseProgram handle synchronization to avoid double skipping.
			p.addError(TokenToPosition(p.current), "failed to parse function declaration", "declaration parsing")

			return nil
		}

		decl = fn
	case lexer.TokenEffect:
		ed := p.parseEffectDeclaration()
		if ed == nil {
			p.addError(TokenToPosition(p.current), "failed to parse effect declaration", "declaration parsing")

			return nil
		}

		decl = ed
	case lexer.TokenLet, lexer.TokenVar, lexer.TokenConst:
		vd := p.parseVariableDeclaration()
		if vd == nil {
			p.addError(TokenToPosition(p.current), "failed to parse variable declaration", "declaration parsing")

			return nil
		}

		decl = vd
	case lexer.TokenMacro:
		md := p.parseMacroDeclaration()
		if md == nil {
			p.addError(TokenToPosition(p.current), "failed to parse macro declaration", "declaration parsing")

			return nil
		}

		decl = md
	case lexer.TokenImport:
		id := p.parseImportDeclaration()
		if id == nil {
			p.addError(TokenToPosition(p.current), "failed to parse import declaration", "declaration parsing")

			return nil
		}

		id.IsPublic = isPublic
		decl = id
	case lexer.TokenExport:
		ed := p.parseExportDeclaration()
		if ed == nil {
			p.addError(TokenToPosition(p.current), "failed to parse export declaration", "declaration parsing")

			return nil
		}

		decl = ed
	case lexer.TokenStruct:
		sd := p.parseStructDeclaration()
		if sd == nil {
			p.addError(TokenToPosition(p.current), "failed to parse struct declaration", "declaration parsing")

			return nil
		}

		sd.IsPublic = isPublic
		decl = sd
	case lexer.TokenEnum:
		ed := p.parseEnumDeclaration()
		if ed == nil {
			p.addError(TokenToPosition(p.current), "failed to parse enum declaration", "declaration parsing")

			return nil
		}

		ed.IsPublic = isPublic
		decl = ed
	case lexer.TokenTrait:
		td := p.parseTraitDeclaration()
		if td == nil {
			p.addError(TokenToPosition(p.current), "failed to parse trait declaration", "declaration parsing")

			return nil
		}

		td.IsPublic = isPublic
		decl = td
	case lexer.TokenImpl:
		ib := p.parseImplBlock()
		if ib == nil {
			p.addError(TokenToPosition(p.current), "failed to parse impl block", "declaration parsing")

			return nil
		}

		decl = ib
	case lexer.TokenTypeKeyword:
		// Support 'type' alias declaration with TokenTypeKeyword.
		td := p.parseTypeAliasDeclaration()
		if td == nil {
			p.addError(TokenToPosition(p.current), "failed to parse type alias", "declaration parsing")

			return nil
		}

		td.IsPublic = isPublic
		decl = td
	default:
		// Support 'type' alias declaration (lexer has no TokenType; detect identifier 'type').
		if p.current.Type == lexer.TokenIdentifier && p.current.Literal == "type" {
			td := p.parseTypeAliasDeclaration()
			if td == nil {
				p.addError(TokenToPosition(p.current), "failed to parse type alias", "declaration parsing")

				return nil
			}

			td.IsPublic = isPublic
			decl = td

			break
		}
		// Support 'newtype' nominal wrapper declaration.
		if p.current.Type == lexer.TokenNewtype || (p.current.Type == lexer.TokenIdentifier && p.current.Literal == "newtype") {
			nd := p.parseNewtypeDeclaration()
			if nd == nil {
				p.addError(TokenToPosition(p.current), "failed to parse newtype", "declaration parsing")

				return nil
			}

			nd.IsPublic = isPublic
			decl = nd

			break
		}
		// Allow top-level expression statements via Pratt parser.
		if exprStmt := p.parseExpressionStatement(); exprStmt != nil {
			return exprStmt
		}
		// If expression parsing failed, record an error and let parseProgram resynchronize.
		p.addError(TokenToPosition(p.current),
			fmt.Sprintf("unexpected token %s in declaration", p.current.Type.String()),
			"declaration parsing")

		return nil
	}

	// Apply parsed modifiers to the declaration where applicable.
	switch d := decl.(type) {
	case *FunctionDeclaration:
		if d == nil {
			return nil
		}

		d.IsPublic = isPublic
		// If 'async' modifier present, mark declaration as async.
		d.IsAsync = isAsync
	case *VariableDeclaration:
		if d == nil {
			return nil
		}

		d.IsPublic = isPublic

		if isAsync {
			// async on variables is invalid; report but continue.
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
	case *ImportDeclaration:
		d.IsPublic = isPublic
	case *ExportDeclaration:
		// visibility inherent to export list.
	case *StructDeclaration:
		d.IsPublic = isPublic
	case *EnumDeclaration:
		d.IsPublic = isPublic
	case *TraitDeclaration:
		d.IsPublic = isPublic
	case *ImplBlock:
		// no modifiers applicable.
	}

	return decl
}

// parseTypeAliasDeclaration parses: type Name = Type ;.
func (p *Parser) parseTypeAliasDeclaration() *TypeAliasDeclaration {
	start := TokenToPosition(p.current)

	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	if !p.expectPeek(lexer.TokenAssign) {
		return nil
	}

	p.nextToken()
	aliased := p.parseType()
	// optional semicolon.
	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	}

	end := TokenToPosition(p.current)

	return &TypeAliasDeclaration{Span: SpanBetween(start, end), Name: name, Aliased: aliased}
}

// parseNewtypeDeclaration parses: newtype Name = Type ;.
func (p *Parser) parseNewtypeDeclaration() *NewtypeDeclaration {
	start := TokenToPosition(p.current)
	// Consume the 'newtype' token.
	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	if !p.expectPeek(lexer.TokenAssign) {
		return nil
	}

	p.nextToken()
	base := p.parseType()
	// optional semicolon.
	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	}

	end := TokenToPosition(p.current)

	return &NewtypeDeclaration{Span: SpanBetween(start, end), Name: name, Base: base}
}

// parseImportDeclaration: import path [as alias] ;? (semicolon optional).
func (p *Parser) parseImportDeclaration() *ImportDeclaration {
	start := TokenToPosition(p.current)
	// Parse path segments: ident { :: ident }.
	if !p.expectPeekRaw(lexer.TokenIdentifier) {
		// Local recovery: malformed import head. Sync to ';' or next top-level start and bail.
		p.addErrorSilent(TokenToPosition(p.peek), "expected module path after 'import'", "import parsing")
		p.skipToBefore(
			lexer.TokenSemicolon,
			lexer.TokenFunc, lexer.TokenLet, lexer.TokenVar, lexer.TokenConst,
			lexer.TokenMacro, lexer.TokenStruct, lexer.TokenEnum, lexer.TokenTrait, lexer.TokenImpl,
			lexer.TokenImport, lexer.TokenExport,
		)
		// Optionally consume semicolon if present to finish the bad import.
		if p.peekTokenIs(lexer.TokenSemicolon) {
			p.nextToken()
		}

		return nil
	}

	path := []*Identifier{NewIdentifier(TokenToSpan(p.current), p.current.Literal)}

	for p.peekTokenIs(lexer.TokenDoubleColon) {
		p.nextToken() // consume ::
		// Support minimal wildcard import: module::*.
		// If the next token is '*', consume it and treat as importing all items from the module.
		// For MVP we don't need to store this in the AST, as HIR import with Items=nil already.
		// represents importing all; we only need to accept the syntax and advance tokens.
		if p.peekTokenIs(lexer.TokenMul) {
			p.nextToken() // consume '*'

			break // stop extending the path; wildcard ends the path
		}

		if !p.expectPeekRaw(lexer.TokenIdentifier) {
			// Malformed segment after '::'. Report and sync locally, then bail to let outer loop recover.
			p.addErrorSilent(TokenToPosition(p.peek), "expected identifier after '::' in import path", "import parsing")
			// Sync to ';' or before the next top-level start so we don't swallow following declarations.
			p.skipToBefore(
				lexer.TokenSemicolon,
				lexer.TokenFunc, lexer.TokenLet, lexer.TokenVar, lexer.TokenConst,
				lexer.TokenMacro, lexer.TokenStruct, lexer.TokenEnum, lexer.TokenTrait, lexer.TokenImpl,
				lexer.TokenImport, lexer.TokenExport,
			)

			if p.peekTokenIs(lexer.TokenSemicolon) {
				p.nextToken()
			}

			return nil
		}

		path = append(path, NewIdentifier(TokenToSpan(p.current), p.current.Literal))
	}
	// Optional alias: as ident.
	var alias *Identifier

	if p.peekTokenIs(lexer.TokenAs) {
		p.nextToken() // move to 'as'

		if !p.expectPeekRaw(lexer.TokenIdentifier) {
			// Recover: missing alias name, sync to end of import and bail.
			p.addErrorSilent(TokenToPosition(p.peek), "expected identifier after 'as' in import", "import parsing")
			p.skipToBefore(
				lexer.TokenSemicolon,
				lexer.TokenFunc, lexer.TokenLet, lexer.TokenVar, lexer.TokenConst,
				lexer.TokenMacro, lexer.TokenStruct, lexer.TokenEnum, lexer.TokenTrait, lexer.TokenImpl,
				lexer.TokenImport, lexer.TokenExport,
			)

			if p.peekTokenIs(lexer.TokenSemicolon) {
				p.nextToken()
			}

			return nil
		}

		alias = NewIdentifier(TokenToSpan(p.current), p.current.Literal)
	}
	// Optional semicolon.
	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	}

	end := TokenToPosition(p.current)

	return &ImportDeclaration{Span: SpanBetween(start, end), Path: path, Alias: alias}
}

// parseExportDeclaration: export { id[, id]* } ;?  (for now only list form).
func (p *Parser) parseExportDeclaration() *ExportDeclaration {
	start := TokenToPosition(p.current)
	// Support two forms later: export <item>; and export { a, b }.
	if !p.expectPeek(lexer.TokenLBrace) {
		// For MVP, allow single identifier: export foo;.
		if p.peekTokenIs(lexer.TokenIdentifier) {
			p.nextToken()
			name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
			// Optional alias via 'as' (not in EBNF list form, but keep for future).
			var alias *Identifier

			if p.peekTokenIs(lexer.TokenAs) {
				p.nextToken()

				if !p.expectPeek(lexer.TokenIdentifier) {
					return nil
				}

				alias = NewIdentifier(TokenToSpan(p.current), p.current.Literal)
			}

			if p.peekTokenIs(lexer.TokenSemicolon) {
				p.nextToken()
			}

			end := TokenToPosition(p.current)

			return &ExportDeclaration{Span: SpanBetween(start, end), Items: []*ExportItem{{Span: name.Span, Name: name, Alias: alias}}}
		}

		p.addError(TokenToPosition(p.peek), "expected '{' or identifier after export", "export parsing")

		return nil
	}
	// parse list.
	items := make([]*ExportItem, 0)

	for {
		if !p.expectPeek(lexer.TokenIdentifier) {
			return nil
		}

		name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
		items = append(items, &ExportItem{Span: name.Span, Name: name})
		// trailing comma or closing brace.
		if p.peekTokenIs(lexer.TokenComma) {
			p.nextToken()

			continue
		}

		if p.peekTokenIs(lexer.TokenRBrace) {
			p.nextToken()

			break
		}
		// tolerate newlines.
		if p.peekTokenIs(lexer.TokenNewline) {
			p.nextToken()

			continue
		}
		// otherwise error.
		p.addError(TokenToPosition(p.peek), "expected ',' or '}' in export list", "export parsing")

		return nil
	}

	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	}

	end := TokenToPosition(p.current)

	return &ExportDeclaration{Span: SpanBetween(start, end), Items: items}
}

// parseEffectDeclaration: effect Name;.
func (p *Parser) parseEffectDeclaration() *EffectDeclaration {
	start := TokenToPosition(p.current)

	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	// Expect semicolon.
	if !p.expectPeek(lexer.TokenSemicolon) {
		return nil
	}

	end := TokenToPosition(p.current)

	return &EffectDeclaration{
		Span: SpanBetween(start, end),
		Name: name,
	}
}

// parseStructDeclaration: struct Name { fields }.
func (p *Parser) parseStructDeclaration() *StructDeclaration {
	start := TokenToPosition(p.current)

	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
	// Optional generics.
	gens := p.parseOptionalGenericParameters()
	// Optional body or ';' forward decl; MVP: require body.
	if !p.expectPeek(lexer.TokenLBrace) {
		return nil
	}

	fields := make([]*StructField, 0)
	closed := false
	// Parse zero or more field declarations until '}'.
	for {
		p.nextToken()

		if p.currentTokenIs(lexer.TokenRBrace) {
			closed = true

			break
		}

		if p.currentTokenIs(lexer.TokenEOF) {
			// EOF hit before closing brace.
			p.addErrorSilent(TokenToPosition(p.current), "missing '}' to close struct block", "struct parsing")

			return nil
		}
		// Skip trivia.
		if p.currentTokenIs(lexer.TokenWhitespace) || p.currentTokenIs(lexer.TokenNewline) || p.currentTokenIs(lexer.TokenComment) {
			continue
		}
		// If we see a token that starts a new top-level declaration, assume the struct block was not closed and bail.
		if p.isTopLevelStart(p.current) {
			p.addErrorSilent(TokenToPosition(p.current), "unexpected top-level declaration inside struct; missing '}'?", "struct parsing")

			return nil
		}
		// Optional 'pub'.
		isPub := false
		if p.currentTokenIs(lexer.TokenPub) {
			isPub = true

			p.nextToken()
		}

		if !p.currentTokenIs(lexer.TokenIdentifier) {
			p.addErrorSilent(TokenToPosition(p.current), "expected field name", "struct field parsing")
			// try skip to next comma or '}'.
			for !p.currentTokenIs(lexer.TokenComma) && !p.currentTokenIs(lexer.TokenRBrace) && !p.currentTokenIs(lexer.TokenEOF) {
				p.nextToken()
			}

			if p.currentTokenIs(lexer.TokenComma) {
				continue
			}

			if p.currentTokenIs(lexer.TokenRBrace) {
				closed = true

				break
			}

			continue
		}

		fname := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

		if !p.expectPeekRaw(lexer.TokenColon) {
			// Improve block-level recovery: report and sync to next field or end of struct.
			p.addErrorSilent(TokenToPosition(p.peek), "expected ':' after struct field name", "struct field parsing")
			p.skipTo(lexer.TokenComma, lexer.TokenRBrace)

			if p.currentTokenIs(lexer.TokenRBrace) {
				closed = true

				break
			}
			// If we landed on a comma, continue to next field.
			if p.currentTokenIs(lexer.TokenComma) {
				continue
			}

			continue
		}

		p.nextToken()
		ftype := p.parseType()
		field := &StructField{Span: SpanBetween(fname.Span.Start, ftype.GetSpan().End), Name: fname, Type: ftype, IsPublic: isPub}
		fields = append(fields, field)
		// Expect comma or '}'.
		if p.peekTokenIs(lexer.TokenComma) {
			p.nextToken()

			continue
		}

		if p.peekTokenIs(lexer.TokenRBrace) {
			p.nextToken()

			closed = true

			break
		}
		// tolerate newline.
		if p.peekTokenIs(lexer.TokenNewline) {
			p.nextToken()

			continue
		}
	}

	if !closed {
		// As a safety, if we somehow exited without marking closed, report and fail.
		p.addErrorSilent(TokenToPosition(p.current), "unterminated struct declaration", "struct parsing")

		return nil
	}

	end := TokenToPosition(p.current)

	return &StructDeclaration{Span: SpanBetween(start, end), Name: name, Fields: fields, Generics: gens}
}

// parseEnumDeclaration: enum Name { Variants }.
func (p *Parser) parseEnumDeclaration() *EnumDeclaration {
	start := TokenToPosition(p.current)

	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
	// Optional generics.
	gens := p.parseOptionalGenericParameters()

	if !p.expectPeek(lexer.TokenLBrace) {
		return nil
	}

	variants := make([]*EnumVariant, 0)
	closed := false

	for {
		p.nextToken()

		if p.currentTokenIs(lexer.TokenRBrace) {
			closed = true

			break
		}

		if p.currentTokenIs(lexer.TokenEOF) {
			p.addErrorSilent(TokenToPosition(p.current), "missing '}' to close enum block", "enum parsing")

			return nil
		}

		if p.currentTokenIs(lexer.TokenWhitespace) || p.currentTokenIs(lexer.TokenNewline) || p.currentTokenIs(lexer.TokenComment) {
			continue
		}

		if p.isTopLevelStart(p.current) {
			p.addErrorSilent(TokenToPosition(p.current), "unexpected top-level declaration inside enum; missing '}'?", "enum parsing")

			return nil
		}

		if !p.currentTokenIs(lexer.TokenIdentifier) {
			p.addErrorSilent(TokenToPosition(p.current), "expected variant name", "enum variant parsing")
			// attempt to sync.
			for !p.currentTokenIs(lexer.TokenComma) && !p.currentTokenIs(lexer.TokenRBrace) && !p.currentTokenIs(lexer.TokenEOF) {
				p.nextToken()
			}

			if p.currentTokenIs(lexer.TokenComma) {
				continue
			}

			if p.currentTokenIs(lexer.TokenRBrace) {
				closed = true

				break
			}

			continue
		}

		vname := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
		variant := &EnumVariant{Span: vname.Span, Name: vname}
		// Optional data: (types) or { fields }.
		if p.peekTokenIs(lexer.TokenLParen) {
			// tuple-like.
			p.nextToken() // '('

			fields := make([]*StructField, 0)
			// parse type list.
			for {
				p.nextToken()
				t := p.parseType()
				fields = append(fields, &StructField{Span: t.GetSpan(), Name: nil, Type: t})

				if p.peekTokenIs(lexer.TokenComma) {
					p.nextToken()

					continue
				}

				if p.peekTokenIs(lexer.TokenRParen) {
					p.nextToken()

					break
				}
			}

			variant.Fields = fields
		} else if p.peekTokenIs(lexer.TokenLBrace) {
			// struct-like.
			p.nextToken()

			sfields := make([]*StructField, 0)

			for {
				p.nextToken()

				if p.currentTokenIs(lexer.TokenRBrace) || p.currentTokenIs(lexer.TokenEOF) {
					break
				}

				if p.currentTokenIs(lexer.TokenWhitespace) || p.currentTokenIs(lexer.TokenNewline) || p.currentTokenIs(lexer.TokenComment) {
					continue
				}

				isPub := false
				if p.currentTokenIs(lexer.TokenPub) {
					isPub = true

					p.nextToken()
				}

				if !p.currentTokenIs(lexer.TokenIdentifier) {
					p.addErrorSilent(TokenToPosition(p.current), "expected field name", "enum variant struct fields")

					break
				}

				fn := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

				if !p.expectPeekRaw(lexer.TokenColon) {
					// Recover within variant field list.
					p.addErrorSilent(TokenToPosition(p.peek), "expected ':' after variant field name", "enum variant struct fields")
					p.skipTo(lexer.TokenComma, lexer.TokenRBrace)

					if p.currentTokenIs(lexer.TokenRBrace) {
						break
					}

					if p.currentTokenIs(lexer.TokenComma) {
						continue
					}

					continue
				}

				p.nextToken()
				ft := p.parseType()
				sfields = append(sfields, &StructField{Span: SpanBetween(fn.Span.Start, ft.GetSpan().End), Name: fn, Type: ft, IsPublic: isPub})

				if p.peekTokenIs(lexer.TokenComma) {
					p.nextToken()

					continue
				}

				if p.peekTokenIs(lexer.TokenRBrace) {
					p.nextToken()

					closed = true

					break
				}
				// tolerate newline and other trivia; otherwise try to resync.
				if p.peekTokenIs(lexer.TokenNewline) || p.peekTokenIs(lexer.TokenWhitespace) || p.peekTokenIs(lexer.TokenComment) {
					p.nextToken()

					continue
				}
				// Unexpected token: resync to next field or end.
				p.addErrorSilent(TokenToPosition(p.peek), "expected ',' or '}' in enum variant struct fields", "enum variant struct fields")
				p.skipTo(lexer.TokenComma, lexer.TokenRBrace)

				if p.currentTokenIs(lexer.TokenRBrace) {
					closed = true

					break
				}
			}

			variant.Fields = sfields
		}

		variants = append(variants, variant)
		// trailing comma or '}'.
		if p.peekTokenIs(lexer.TokenComma) {
			p.nextToken()

			continue
		}

		if p.peekTokenIs(lexer.TokenRBrace) {
			p.nextToken()

			closed = true

			break
		}
	}

	if !closed {
		p.addErrorSilent(TokenToPosition(p.current), "unterminated enum declaration", "enum parsing")

		return nil
	}

	end := TokenToPosition(p.current)

	return &EnumDeclaration{Span: SpanBetween(start, end), Name: name, Variants: variants, Generics: gens}
}

// parseTraitDeclaration: trait Name { method signatures }.
func (p *Parser) parseTraitDeclaration() *TraitDeclaration {
	start := TokenToPosition(p.current)

	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
	// Optional generics.
	gens := p.parseOptionalGenericParameters()
	// Optional bounds after ':' (not fully used yet).
	if p.peekTokenIs(lexer.TokenColon) {
		p.nextToken()
		p.nextToken()
		_ = p.parseTraitBounds()
	}

	if !p.expectPeek(lexer.TokenLBrace) {
		return nil
	}

	methods := make([]*TraitMethod, 0)
	assocTypes := make([]*AssociatedType, 0)
	closed := false

	for {
		p.nextToken()

		if p.currentTokenIs(lexer.TokenRBrace) {
			closed = true

			break
		}

		if p.currentTokenIs(lexer.TokenEOF) {
			p.addErrorSilent(TokenToPosition(p.current), "missing '}' to close trait body", "trait parsing")

			return nil
		}

		if p.currentTokenIs(lexer.TokenWhitespace) || p.currentTokenIs(lexer.TokenNewline) || p.currentTokenIs(lexer.TokenComment) {
			continue
		}
		// Disallow unrelated top-level starts inside trait, but allow valid trait items: 'func' and associated 'type'.
		if p.isTopLevelStart(p.current) && !(p.current.Type == lexer.TokenFunc || p.current.Type == lexer.TokenTypeKeyword || (p.current.Type == lexer.TokenIdentifier && p.current.Literal == "type")) {
			p.addErrorSilent(TokenToPosition(p.current), "unexpected top-level declaration inside trait; missing '}'?", "trait parsing")

			return nil
		}
		// Associated type: 'type' identifier [ : bounds ] ;.
		// Support both TokenTypeKeyword and identifier literal "type".
		if p.current.Type == lexer.TokenTypeKeyword || (p.currentTokenIs(lexer.TokenIdentifier) && p.current.Literal == "type") {
			if !p.expectPeekRaw(lexer.TokenIdentifier) {
				// Recover to ';' or '}' to continue parsing remaining items.
				p.addErrorSilent(TokenToPosition(p.peek), "expected associated type name after 'type'", "trait parsing")
				p.skipTo(lexer.TokenSemicolon, lexer.TokenRBrace)

				continue
			}

			aname := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
			bounds := []Type{}

			if p.peekTokenIs(lexer.TokenColon) {
				p.nextToken()
				p.nextToken()
				bounds = p.parseTraitBounds()
			}

			if p.peekTokenIs(lexer.TokenSemicolon) {
				p.nextToken()
			}

			assocTypes = append(assocTypes, &AssociatedType{Span: aname.Span, Name: aname, Bounds: bounds})

			continue
		}
		// method signature: func name [<T,...>](params) [-> type] ;
		if p.currentTokenIs(lexer.TokenFunc) {
			if !p.expectPeekRaw(lexer.TokenIdentifier) {
				p.addErrorSilent(TokenToPosition(p.peek), "expected method name after 'func'", "trait parsing")
				p.skipTo(lexer.TokenSemicolon, lexer.TokenRBrace)

				continue
			}

			mname := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
			// Optional method-level generics.
			gens := p.parseOptionalGenericParameters()

			if !p.expectPeekRaw(lexer.TokenLParen) {
				p.addErrorSilent(TokenToPosition(p.peek), "expected '(' after method name", "trait parsing")
				p.skipTo(lexer.TokenSemicolon, lexer.TokenRBrace)

				continue
			}

			params := p.parseParameterList()

			if !p.expectPeekRaw(lexer.TokenRParen) {
				p.addErrorSilent(TokenToPosition(p.peek), "expected ')' after method parameters", "trait parsing")
				p.skipTo(lexer.TokenSemicolon, lexer.TokenRBrace)

				continue
			}
			// Optional return type.
			var ret Type

			for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
				p.nextToken()
			}

			if p.peekTokenIs(lexer.TokenArrow) {
				p.nextToken()
				p.nextToken()
				ret = p.parseType()
			}
			// optional semicolon.
			if p.peekTokenIs(lexer.TokenSemicolon) {
				p.nextToken()
			}

			methods = append(methods, &TraitMethod{Span: mname.Span, Name: mname, Parameters: params, ReturnType: ret, Generics: gens})

			continue
		}

		p.addErrorSilent(TokenToPosition(p.current), "expected 'func' or 'type' in trait body", "trait parsing")
		// sync to next possible item.
		for !p.currentTokenIs(lexer.TokenSemicolon) && !p.currentTokenIs(lexer.TokenRBrace) && !p.currentTokenIs(lexer.TokenEOF) {
			p.nextToken()
		}
	}

	if !closed {
		p.addErrorSilent(TokenToPosition(p.current), "unterminated trait declaration", "trait parsing")

		return nil
	}

	end := TokenToPosition(p.current)

	return &TraitDeclaration{Span: SpanBetween(start, end), Name: name, Methods: methods, Generics: gens, AssociatedTypes: assocTypes}
}

// parseImplBlock: impl [Trait for] Type { func ... }.
func (p *Parser) parseImplBlock() *ImplBlock {
	start := TokenToPosition(p.current)
	// Optional generics: impl <T, ...>
	gens := p.parseOptionalGenericParameters()
	// Two forms: impl Type { ... }  |  impl Trait for Type { ... }
	// Peek to decide if a trait path appears before 'for'.
	// Move to next token to start type/trait
	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}
	// Parse a simple path or type name as first part, then optional generic suffix.
	firstType := p.parseBasicTypeOnly()
	firstType = p.parseGenericSuffixOn(firstType)

	var trait Type

	var forType Type

	if p.peekTokenIs(lexer.TokenFor) {
		// trait for type form.
		trait = firstType

		p.nextToken() // move to 'for'

		if !p.expectPeek(lexer.TokenIdentifier) {
			return nil
		}

		forType = p.parseBasicTypeOnly()
		forType = p.parseGenericSuffixOn(forType)
	} else {
		// inherent impl.
		forType = firstType
	}
	// Optional where clause.
	where := p.parseOptionalWhereClause()

	// Skip trivia before expecting LBRACE.
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if !p.expectPeek(lexer.TokenLBrace) {
		return nil
	}

	items := make([]*FunctionDeclaration, 0)
	closed := false

	for {
		p.nextToken()

		if p.currentTokenIs(lexer.TokenRBrace) {
			closed = true

			break
		}

		if p.currentTokenIs(lexer.TokenEOF) {
			p.addErrorSilent(TokenToPosition(p.current), "missing '}' to close impl block", "impl parsing")

			return nil
		}

		if p.currentTokenIs(lexer.TokenWhitespace) || p.currentTokenIs(lexer.TokenNewline) || p.currentTokenIs(lexer.TokenComment) {
			continue
		}
		// Inside impl, we only accept function items; any other token is treated as an unexpected item and skipped locally.
		// Expect function declarations only for MVP.
		if p.currentTokenIs(lexer.TokenFunc) || (p.currentTokenIs(lexer.TokenIdentifier) && p.current.Literal == "fn") {
			fn := p.parseFunctionDeclaration()
			if fn != nil {
				items = append(items, fn)
			}

			continue
		}
		// skip unknown item until next '}' or 'func'.
		p.addErrorSilent(TokenToPosition(p.current), "unexpected item in impl block", "impl parsing")

		for !p.currentTokenIs(lexer.TokenRBrace) && !p.currentTokenIs(lexer.TokenFunc) && !p.currentTokenIs(lexer.TokenEOF) && !p.peekTokenIs(lexer.TokenRBrace) {
			p.nextToken()
		}

		if p.currentTokenIs(lexer.TokenFunc) {
			fn := p.parseFunctionDeclaration()
			if fn != nil {
				items = append(items, fn)
			}
		}
		// If we stopped because '}' is next, the outer loop will handle it on next iteration.
	}

	if !closed {
		p.addErrorSilent(TokenToPosition(p.current), "unterminated impl block", "impl parsing")

		return nil
	}

	end := TokenToPosition(p.current)

	return &ImplBlock{Span: SpanBetween(start, end), Trait: trait, ForType: forType, Items: items, Generics: gens, WhereClauses: where}
}

func (p *Parser) parseFunctionDeclaration() *FunctionDeclaration {
	startPos := TokenToPosition(p.current)

	// Accept both 'func' keyword and 'fn' identifier as an alias for function declarations.
	isFuncKw := p.currentTokenIs(lexer.TokenFunc) || p.currentTokenIs(lexer.TokenFn) || (p.currentTokenIs(lexer.TokenIdentifier) && p.current.Literal == "fn")
	if !isFuncKw {
		return nil
	}

	if !p.expectPeekRaw(lexer.TokenIdentifier) {
		p.addErrorSilent(TokenToPosition(p.peek), "expected identifier after 'func'", "function declaration")
		// consume rest of line to avoid getting stuck at random identifiers.
		for !p.currentTokenIs(lexer.TokenEOF) && !p.currentTokenIs(lexer.TokenNewline) {
			p.nextToken()
		}

		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	// Optional generics after function name.
	gens := p.parseOptionalGenericParameters()

	if !p.expectPeekRaw(lexer.TokenLParen) {
		p.addErrorSilent(TokenToPosition(p.peek), "expected '(' after function name", "function declaration")

		for !p.currentTokenIs(lexer.TokenEOF) && !p.currentTokenIs(lexer.TokenNewline) {
			p.nextToken()
		}

		return nil
	}

	parameters := p.parseParameterList()

	if !p.expectPeekRaw(lexer.TokenRParen) {
		p.addErrorSilent(TokenToPosition(p.peek), "expected ')' after parameter list", "function declaration")

		for !p.currentTokenIs(lexer.TokenEOF) && !p.currentTokenIs(lexer.TokenNewline) {
			p.nextToken()
		}

		return nil
	}

	// Optional return type.
	var returnType Type
	// Allow trivia (including newlines) between ')' and '->'.
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if p.peekTokenIs(lexer.TokenArrow) {
		p.nextToken() // consume arrow
		p.nextToken() // move to type
		returnType = p.parseType()
	}

	// Optional effects annotation: effects(io, alloc, unsafe).
	var effects *EffectAnnotation
	// Allow trivia before effects.
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if p.peekTokenIs(lexer.TokenEffects) {
		p.nextToken() // consume 'effects'
		effects = p.parseEffectAnnotation()
	}

	// Optional where clause: where T: Display, U: Clone.
	var whereClause []*WherePredicate
	// Allow trivia before where clause.
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if p.peekTokenIs(lexer.TokenWhere) {
		p.nextToken() // move to 'where'
		whereClause = p.parseWhereClause()
	}

	// Allow trivia before function body '{'.
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if !p.expectPeekRaw(lexer.TokenLBrace) {
		p.addErrorSilent(TokenToPosition(p.peek), "expected '{' to start function body", "function declaration")

		for !p.currentTokenIs(lexer.TokenEOF) && !p.currentTokenIs(lexer.TokenNewline) {
			p.nextToken()
		}

		return nil
	}

	body := p.parseBlockStatement()
	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &FunctionDeclaration{
		Span:        span,
		Name:        name,
		Parameters:  parameters,
		ReturnType:  returnType,
		Body:        body,
		IsPublic:    false,
		IsAsync:     false,
		Generics:    gens,
		WhereClause: whereClause,
		Effects:     effects,
	}
}

// parseEffectAnnotation parses: effects(io, alloc, unsafe).
func (p *Parser) parseEffectAnnotation() *EffectAnnotation {
	start := TokenToPosition(p.current)

	if !p.expectPeek(lexer.TokenLParen) {
		return nil
	}

	effects := make([]*Effect, 0)

	// Parse effect list.
	for {
		p.nextToken()

		if p.currentTokenIs(lexer.TokenRParen) {
			break
		}

		if p.currentTokenIs(lexer.TokenEOF) {
			p.addError(TokenToPosition(p.current), "unterminated effect annotation", "effect parsing")

			return nil
		}

		if p.currentTokenIs(lexer.TokenWhitespace) || p.currentTokenIs(lexer.TokenComment) || p.currentTokenIs(lexer.TokenNewline) {
			continue
		}

		if !p.currentTokenIs(lexer.TokenIdentifier) {
			p.addError(TokenToPosition(p.current), "expected effect name", "effect parsing")

			return nil
		}

		effectName := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
		effects = append(effects, &Effect{
			Span: TokenToSpan(p.current),
			Name: effectName,
		})

		// Skip trivia.
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}

		if p.peekTokenIs(lexer.TokenComma) {
			p.nextToken()

			continue
		}

		if p.peekTokenIs(lexer.TokenRParen) {
			p.nextToken()

			break
		}

		p.addError(TokenToPosition(p.peek), "expected ',' or ')' in effect list", "effect parsing")

		return nil
	}

	end := TokenToPosition(p.current)

	return &EffectAnnotation{
		Span:    SpanBetween(start, end),
		Effects: effects,
	}
}

// parseParameterList parses function parameters.
func (p *Parser) parseParameterList() []*Parameter {
	parameters := make([]*Parameter, 0)

	// Allow empty parameter list with intervening trivia.
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if p.peekTokenIs(lexer.TokenRParen) {
		return parameters
	}

	// Move to first parameter token (skipping trivia already handled above).
	p.nextToken()

	// Parse first parameter.
	param := p.parseParameter()
	if param != nil {
		parameters = append(parameters, param)
	}

	// Parse remaining parameters.
	for {
		// Skip trivia before checking for comma or closing paren.
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}

		if !p.peekTokenIs(lexer.TokenComma) {
			break
		}

		p.nextToken() // consume comma
		// Skip trivia before next parameter.
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}

		p.nextToken() // move to next parameter

		param := p.parseParameter()
		if param != nil {
			parameters = append(parameters, param)
		}
	}

	return parameters
}

// parseParameter parses a single parameter.
func (p *Parser) parseParameter() *Parameter {
	startPos := TokenToPosition(p.current)
	// Optional 'mut' modifier on parameters.
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

// parseVariableDeclaration parses a variable declaration.
func (p *Parser) parseVariableDeclaration() *VariableDeclaration {
	startPos := TokenToPosition(p.current)

	isMutable := p.currentTokenIs(lexer.TokenVar)
	// Support `let mut name` form.
	if p.currentTokenIs(lexer.TokenLet) && p.peekTokenIs(lexer.TokenMut) {
		p.nextToken() // consume 'mut'

		isMutable = true
	}

	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	// Optional type annotation.
	var typeSpec Type

	if p.peekTokenIs(lexer.TokenColon) {
		p.nextToken() // consume colon
		p.nextToken() // move to type
		typeSpec = p.parseType()
	}

	// Optional initializer.
	var initializer Expression

	if p.peekTokenIs(lexer.TokenAssign) {
		p.nextToken() // consume =
		p.nextToken() // move to expression
		initializer = p.parseExpression(LOWEST)
	}

	// Expect semicolon (or allow newline with suggestion). Handle cases where semicolon is current.
	if p.currentTokenIs(lexer.TokenSemicolon) {
		// already at semicolon (e.g., after recovering), keep as is
	} else if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	} else if p.peekTokenIs(lexer.TokenNewline) {
		// Provide suggestion to insert semicolon before newline.
		if p.suggestionEngine != nil {
			pos := TokenToPosition(p.current)
			pos.File = p.filename
			p.suggestions = append(p.suggestions, Suggestion{
				Type:        ErrorFix,
				Message:     "Insert semicolon before newline",
				Position:    pos,
				Replacement: ";",
				Confidence:  0.8,
				Category:    SyntaxError,
			})
		}
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &VariableDeclaration{
		Span:        span,
		Name:        name,
		TypeSpec:    typeSpec,
		Initializer: initializer,
		IsMutable:   isMutable,
		// Public visibility is applied by parseDeclaration based on leading modifiers.
		IsPublic: false,
	}
}

// parseType parses a type specification.
func (p *Parser) parseType() Type {
	// Skip trivia at type start if present.
	for p.current.Type == lexer.TokenWhitespace || p.current.Type == lexer.TokenComment || p.current.Type == lexer.TokenNewline {
		p.nextToken()
	}

	// Helper to optionally parse generic arguments after a base type.
	parseGenericSuffix := func(base Type) Type {
		// Skip trivia.
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}

		if !p.peekTokenIs(lexer.TokenLt) {
			return base
		}
		// consume '<'.
		p.nextToken()
		// move to first type or '>'.
		p.nextToken()

		typeParams := make([]Type, 0)
		// Allow empty generic arg list is invalid; enforce at least one type.
		if !p.currentTokenIs(lexer.TokenGt) {
			// parse first type.
			tp := p.parseType()
			if tp != nil {
				typeParams = append(typeParams, tp)
			}
			// parse remaining , type.
			for {
				// Skip trivia before checking comma or '>'.
				for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
					p.nextToken()
				}

				if !p.peekTokenIs(lexer.TokenComma) {
					break
				}

				p.nextToken() // consume comma
				// Skip trivia then move to next type.
				for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
					p.nextToken()
				}

				p.nextToken()

				tp = p.parseType()
				if tp != nil {
					typeParams = append(typeParams, tp)
				}
			}
			// Expect '>'.
			if !p.expectPeekRaw(lexer.TokenGt) {
				p.addError(TokenToPosition(p.peek), "expected '>' to close generic arguments", "type parsing")

				return base
			}
		}
		// current is now at '>' after expectPeekRaw.
		span := SpanBetween(base.(TypeSafeNode).GetSpan().Start, TokenToSpan(p.current).End)

		return &GenericType{Span: span, BaseType: base, TypeParameters: typeParams}
	}

	switch p.current.Type {
	case lexer.TokenBitAnd:
		// reference type: &T, &mut T, &'a T, &'a mut T.
		start := TokenToPosition(p.current)
		// optional whitespace/comments/newlines after '&'
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}
		// consume next token (potentially 'mut' or start of type).
		p.nextToken()
		// optional lifetime token.
		lifetime := ""
		if p.current.Type == lexer.TokenLifetime {
			lifetime = p.current.Literal
			// skip trivia after lifetime.
			for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
				p.nextToken()
			}

			p.nextToken()
		}

		isMut := false
		if p.current.Type == lexer.TokenMut {
			isMut = true
			// move to the inner type start.
			p.nextToken()
		}

		inner := p.parseType()
		end := TokenToPosition(p.current)

		return &ReferenceType{Span: SpanBetween(start, end), Inner: inner, IsMutable: isMut, Lifetime: lifetime}
	case lexer.TokenMul:
		// pointer type: *T or *mut T.
		start := TokenToPosition(p.current)

		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}

		p.nextToken()

		isMut := false
		if p.current.Type == lexer.TokenMut {
			isMut = true

			p.nextToken()
		}

		inner := p.parseType()
		end := TokenToPosition(p.current)

		return &PointerType{Span: SpanBetween(start, end), Inner: inner, IsMutable: isMut}
	case lexer.TokenIdentifier:
		// Support path types: A::B::C and apply generic suffix to the tail.
		base := p.parsePathOrBasicType()

		return parseGenericSuffix(base)
	case lexer.TokenInteger:
		// Integer literals as types in dependent contexts (e.g., Array<Int, 10>)
		// For now, treat as BasicType with the literal value as name.
		return &BasicType{
			Span: TokenToSpan(p.current),
			Name: p.current.Literal,
		}
	case lexer.TokenLBracket:
		// Array or slice type: [T] (slice) or [T; N] (array).
		start := TokenToPosition(p.current)
		p.nextToken() // move to element type
		elem := p.parseType()
		// After element type, allow trivia.
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}

		if p.peekTokenIs(lexer.TokenSemicolon) {
			// static array with size expression.
			p.nextToken() // consume ';'
			p.nextToken() // move to size expression
			sizeExpr := p.parseExpression(LOWEST)

			if !p.expectPeekRaw(lexer.TokenRBracket) {
				p.addError(TokenToPosition(p.peek), "expected ']' to close array type", "type parsing")
			}

			end := TokenToPosition(p.current)

			return &ArrayType{Span: SpanBetween(start, end), ElementType: elem, Size: sizeExpr, IsDynamic: false}
		}
		// Expect ']' for slice type.
		if !p.expectPeekRaw(lexer.TokenRBracket) {
			p.addError(TokenToPosition(p.peek), "expected ']' to close slice type", "type parsing")
		}

		end := TokenToPosition(p.current)

		return &ArrayType{Span: SpanBetween(start, end), ElementType: elem, Size: nil, IsDynamic: true}
	case lexer.TokenLBrace:
		// Refinement type: {n: Int | n > 0}.
		refinement := p.parseRefinementType()
		if refinementExpr, ok := refinement.(*RefinementTypeExpression); ok {
			// Return RefinementTypeExpression directly for now.
			// TODO: Implement proper RefinementType when dependent types are ready.
			return refinementExpr
		}

		return nil
	case lexer.TokenAsync, lexer.TokenFunc, lexer.TokenFn:
		// function type: [async] func(paramList) [-> type].
		start := TokenToPosition(p.current)

		isAsync := false
		if p.current.Type == lexer.TokenAsync {
			isAsync = true
			// ensure 'func' or 'fn' follows.
			for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
				p.nextToken()
			}

			if !p.peekTokenIs(lexer.TokenFunc) && !p.peekTokenIs(lexer.TokenFn) {
				p.addError(TokenToPosition(p.current), "expected 'func' or 'fn' after 'async' in type", "type parsing")

				return nil
			}

			p.nextToken() // move to TokenFunc or TokenFn
		}

		if !p.expectPeekRaw(lexer.TokenLParen) {
			p.addError(TokenToPosition(p.peek), "expected '(' after 'func' in type", "type parsing")

			return nil
		}

		params := make([]*FunctionTypeParameter, 0)
		// Allow empty parameter list.
		// Skip trivia.
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}

		if !p.peekTokenIs(lexer.TokenRParen) {
			// Move to first token within params.
			p.nextToken()
			// parse first param.
			name := ""

			var pt Type

			if p.current.Type == lexer.TokenIdentifier && p.peekTokenIs(lexer.TokenColon) {
				name = p.current.Literal
				p.nextToken() // consume ':'
				p.nextToken() // move to type
				pt = p.parseType()
			} else {
				pt = p.parseType()
			}

			params = append(params, &FunctionTypeParameter{Span: TokenToSpan(p.current), Name: name, Type: pt})
			// remaining params.
			for {
				// Skip trivia before comma or ')'.
				for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
					p.nextToken()
				}

				if !p.peekTokenIs(lexer.TokenComma) {
					break
				}

				p.nextToken() // consume ','
				// Skip trivia then move to next param start.
				for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
					p.nextToken()
				}

				p.nextToken()

				name = ""
				if p.current.Type == lexer.TokenIdentifier && p.peekTokenIs(lexer.TokenColon) {
					name = p.current.Literal
					p.nextToken() // consume ':'
					p.nextToken() // move to type
					pt = p.parseType()
				} else {
					pt = p.parseType()
				}

				params = append(params, &FunctionTypeParameter{Span: TokenToSpan(p.current), Name: name, Type: pt})
			}
		}

		if !p.expectPeekRaw(lexer.TokenRParen) {
			p.addError(TokenToPosition(p.peek), "expected ')' after function type parameters", "type parsing")

			return nil
		}
		// Optional return type: allow trivia then '->'.
		// Skip trivia.
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}

		var ret Type

		if p.peekTokenIs(lexer.TokenArrow) {
			p.nextToken() // consume '->'
			p.nextToken() // move to type
			ret = p.parseType()
		} else {
			// If unspecified, leave nil (interpreted as unit later).
			ret = nil
		}

		end := TokenToPosition(p.current)

		return &FunctionType{Span: SpanBetween(start, end), Parameters: params, ReturnType: ret, IsAsync: isAsync}
	case lexer.TokenLParen:
		// Tuple type: (T1, T2, ...) or unit type: ()
		start := TokenToPosition(p.current)
		p.nextToken() // move past '('

		// Check for empty tuple (unit type).
		if p.currentTokenIs(lexer.TokenRParen) {
			end := TokenToPosition(p.current)

			return &TupleType{Span: SpanBetween(start, end), Elements: []Type{}}
		}

		// Parse tuple elements.
		elements := make([]Type, 0)
		// Parse first element.
		elem := p.parseType()
		if elem != nil {
			elements = append(elements, elem)
		}

		// Parse remaining elements.
		for {
			// Skip trivia before comma or ')'.
			for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
				p.nextToken()
			}

			if !p.peekTokenIs(lexer.TokenComma) {
				break
			}

			p.nextToken() // consume ','

			// Skip trivia and move to next element.
			for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
				p.nextToken()
			}

			p.nextToken()

			// Allow trailing comma.
			if p.currentTokenIs(lexer.TokenRParen) {
				break
			}

			elem = p.parseType()
			if elem != nil {
				elements = append(elements, elem)
			}
		}

		if !p.expectPeekRaw(lexer.TokenRParen) {
			p.addError(TokenToPosition(p.peek), "expected ')' to close tuple type", "type parsing")

			return nil
		}

		end := TokenToPosition(p.current)

		return &TupleType{Span: SpanBetween(start, end), Elements: elements}
	default:
		p.addError(TokenToPosition(p.current),
			fmt.Sprintf("unexpected token %s in type", p.current.Type.String()),
			"type parsing")

		return nil
	}
}

// parsePathOrBasicType parses a possibly qualified path type like A::B and returns a BasicType with joined name for now.
func (p *Parser) parsePathOrBasicType() Type {
	// current is identifier at head.
	start := TokenToPosition(p.current)
	parts := []string{p.current.Literal}
	// Accumulate segments.
	for p.peekTokenIs(lexer.TokenDoubleColon) {
		p.nextToken() // consume '::'

		if !p.expectPeek(lexer.TokenIdentifier) {
			break
		}

		parts = append(parts, p.current.Literal)
	}
	// For MVP, represent path as BasicType with qualified name (resolution later).
	bt := &BasicType{Span: SpanBetween(start, TokenToPosition(p.current)), Name: strings.Join(parts, "::")}

	// Check for dependent type with where clause: Type where Constraint.
	// Skip trivia before checking 'where'.
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if p.peekTokenIs(lexer.TokenWhere) {
		p.nextToken() // consume 'where'
		p.nextToken() // move to constraint expression

		constraint := p.parseExpression(LOWEST)
		if constraint != nil {
			end := TokenToPosition(p.current)

			return &DependentType{
				Span:       SpanBetween(start, end),
				BaseType:   bt,
				Constraint: constraint,
			}
		}
	}

	return bt
}

// parseBasicTypeOnly parses basic type path without checking for dependent type where clause.
func (p *Parser) parseBasicTypeOnly() Type {
	// current is identifier at head.
	start := TokenToPosition(p.current)
	parts := []string{p.current.Literal}
	// Accumulate segments.
	for p.peekTokenIs(lexer.TokenDoubleColon) {
		p.nextToken() // consume '::'

		if !p.expectPeek(lexer.TokenIdentifier) {
			break
		}

		parts = append(parts, p.current.Literal)
	}
	// For MVP, represent path as BasicType with qualified name (resolution later).
	return &BasicType{Span: SpanBetween(start, TokenToPosition(p.current)), Name: strings.Join(parts, "::")}
}

// parseGenericSuffixOn applies a generic argument suffix like <T, U> to the given base type if the next token is '<'.
func (p *Parser) parseGenericSuffixOn(base Type) Type {
	// Skip trivia.
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if !p.peekTokenIs(lexer.TokenLt) {
		return base
	}
	// consume '<'.
	p.nextToken()
	// move to first type or '>'.
	p.nextToken()

	typeParams := make([]Type, 0)

	if !p.currentTokenIs(lexer.TokenGt) {
		// parse first type.
		tp := p.parseType()
		if tp != nil {
			typeParams = append(typeParams, tp)
		}
		// parse remaining , type.
		for {
			// Skip trivia before checking comma or '>'.
			for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
				p.nextToken()
			}

			if !p.peekTokenIs(lexer.TokenComma) {
				break
			}

			p.nextToken() // consume comma
			// Skip trivia then move to next type.
			for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
				p.nextToken()
			}

			p.nextToken()

			tp = p.parseType()
			if tp != nil {
				typeParams = append(typeParams, tp)
			}
		}
		// Expect '>'.
		if !p.expectPeekRaw(lexer.TokenGt) {
			p.addError(TokenToPosition(p.peek), "expected '>' to close generic arguments", "type parsing")

			return base
		}
	}

	span := SpanBetween(base.(TypeSafeNode).GetSpan().Start, TokenToSpan(p.current).End)

	return &GenericType{Span: span, BaseType: base, TypeParameters: typeParams}
}

// ====== Generics / Where / Bounds Parsing ======

// parseOptionalGenericParameters parses '<...>' if present.
func (p *Parser) parseOptionalGenericParameters() []*GenericParameter {
	// Check if next token is '<' to start generics.
	if !p.peekTokenIs(lexer.TokenLt) {
		return []*GenericParameter{}
	}

	p.nextToken() // consume '<'

	var params []*GenericParameter

	// Handle empty generics list.
	if p.peekTokenIs(lexer.TokenGt) {
		p.nextToken() // consume '>'

		return params
	}

	// Parse generic parameters separated by commas.
	for {
		p.nextToken() // move to the parameter token

		param := p.parseGenericParameter()
		if param != nil {
			params = append(params, param)
		} else {
			// Skip invalid parameters.
			p.nextToken()
		}

		// Check if we're done.
		if p.peekTokenIs(lexer.TokenGt) {
			p.nextToken() // consume '>'

			break
		}

		// Expect comma between parameters.
		if !p.peekTokenIs(lexer.TokenComma) {
			// Error recovery: consume until '>' or EOF.
			for !p.peekTokenIs(lexer.TokenGt) && !p.peekTokenIs(lexer.TokenEOF) {
				p.nextToken()
			}

			if p.peekTokenIs(lexer.TokenGt) {
				p.nextToken() // consume '>'
			}

			break
		}

		p.nextToken() // consume ','
	}

	return params
}

// parseGenericParameter parses one of: identifier [":" bounds] | const identifier ":" type | lifetime.
func (p *Parser) parseGenericParameter() *GenericParameter {
	start := TokenToPosition(p.current)

	// Lifetime parameter: TokenLifetime.
	if p.current.Type == lexer.TokenLifetime {
		lifetime := p.current.Literal

		return &GenericParameter{
			Span:     SpanBetween(start, TokenToPosition(p.current)),
			Kind:     GenericParamLifetime,
			Lifetime: lifetime,
		}
	}

	// Const parameter: 'const' ident ':' type.
	if p.current.Type == lexer.TokenConst {
		p.nextToken() // consume 'const'

		if p.current.Type != lexer.TokenIdentifier {
			return nil
		}

		name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
		p.nextToken()

		if p.current.Type != lexer.TokenColon {
			return nil
		}

		p.nextToken() // consume ':'
		ctype := p.parseType()

		return &GenericParameter{
			Span:      SpanBetween(start, TokenToPosition(p.current)),
			Kind:      GenericParamConst,
			Name:      name,
			ConstType: ctype,
		}
	}

	// Type parameter: ident [":" bounds].
	if p.current.Type == lexer.TokenIdentifier {
		name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

		var bounds []Type

		// Check for bounds: T: Display + Debug.
		if p.peekTokenIs(lexer.TokenColon) {
			p.nextToken() // move to ':'
			p.nextToken() // consume ':' and move to first bound

			for {
				bound := p.parseType()
				if bound != nil {
					bounds = append(bounds, bound)
				}

				// Check if there's a '+' for more bounds.
				if !p.peekTokenIs(lexer.TokenPlus) {
					break
				}

				p.nextToken() // move to '+'
				p.nextToken() // consume '+' and move to next bound
			}
		}

		return &GenericParameter{
			Span:   SpanBetween(start, TokenToPosition(p.current)),
			Kind:   GenericParamType,
			Name:   name,
			Bounds: bounds,
		}
	}

	return nil
}

// parseTraitBounds parses trait_bound { '+' trait_bound }.
func (p *Parser) parseTraitBounds() []Type {
	bounds := []Type{}
	// first bound.
	b := p.parseType()
	if b != nil {
		bounds = append(bounds, b)
	}

	for {
		// Skip trivia.
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}

		if !p.peekTokenIs(lexer.TokenPlus) {
			break
		}

		p.nextToken() // consume '+'
		p.nextToken() // move to next bound

		b = p.parseType()
		if b != nil {
			bounds = append(bounds, b)
		}
	}

	return bounds
}

// parseOptionalWhereClause parses where_clause if present: 'where' pred {',' pred}.
func (p *Parser) parseOptionalWhereClause() []*WherePredicate {
	preds := []*WherePredicate{}
	// Skip trivia.
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if !p.peekTokenIs(lexer.TokenWhere) {
		return preds
	}

	p.nextToken() // move to 'where'
	// Move to first predicate start.
	p.nextToken()

	pred := p.parseWherePredicate()
	if pred != nil {
		preds = append(preds, pred)
	}
	// Check for more predicates.
	for {
		// Skip trivia.
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}

		if !p.peekTokenIs(lexer.TokenComma) {
			break
		}

		p.nextToken() // consume comma
		// Skip trivia then move to next predicate.
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}

		p.nextToken() // move to predicate start

		pred = p.parseWherePredicate()
		if pred != nil {
			preds = append(preds, pred)
		}
	}

	return preds
}

// parseWherePredicate parses: type ':' trait_bounds.
func (p *Parser) parseWherePredicate() *WherePredicate {
	start := TokenToPosition(p.current)

	t := p.parseType()
	if t == nil {
		return nil
	}

	if !p.expectPeek(lexer.TokenColon) {
		return nil
	}

	p.nextToken() // move past colon to trait bounds
	bounds := p.parseTraitBounds()
	// After parsing trait bounds, ensure we're positioned correctly.
	// parseTraitBounds() should leave us on the last bound token.
	return &WherePredicate{Span: SpanBetween(start, TokenToPosition(p.current)), Target: t, Bounds: bounds}
}

// parseBlockStatement parses a block statement.
func (p *Parser) parseBlockStatement() *BlockStatement {
	startPos := TokenToPosition(p.current)
	statements := make([]Statement, 0)

	p.nextToken() // consume opening brace

	for !p.currentTokenIs(lexer.TokenRBrace) && !p.currentTokenIs(lexer.TokenEOF) {
		// Skip whitespace and comments.
		if p.currentTokenIs(lexer.TokenWhitespace) || p.currentTokenIs(lexer.TokenComment) || p.currentTokenIs(lexer.TokenNewline) {
			p.nextToken()

			continue
		}

		stmt := p.parseStatement()
		if stmt != nil {
			statements = append(statements, stmt)
		}

		p.nextToken()
	}

	// Check if we hit EOF without finding closing brace.
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

// parseStatement parses a statement.
func (p *Parser) parseStatement() Statement {
	// Allow nested function declarations (e.g., inside macro bodies)
	if p.current.Type == lexer.TokenFunc || p.current.Type == lexer.TokenFn || (p.current.Type == lexer.TokenIdentifier && p.current.Literal == "fn") {
		return p.parseFunctionDeclaration()
	}

	switch p.current.Type {
	case lexer.TokenLet, lexer.TokenVar, lexer.TokenConst:
		return p.parseVariableDeclaration()
	case lexer.TokenReturn:
		return p.parseReturnStatement()
	case lexer.TokenIf:
		return p.parseIfStatement()
	case lexer.TokenWhile:
		return p.parseWhileStatement()
	case lexer.TokenLoop:
		return p.parseLoopStatement()
	case lexer.TokenMatch:
		return p.parseMatchStatement()
	case lexer.TokenFor:
		return p.parseForStatement()
	case lexer.TokenBreak:
		return p.parseBreakStatement()
	case lexer.TokenContinue:
		return p.parseContinueStatement()
	case lexer.TokenDefer:
		return p.parseDeferStatement()
	case lexer.TokenLBrace:
		return p.parseBlockStatement()
	default:
		// Enhanced error recovery for unknown statement beginnings.
		stmt := p.parseExpressionStatement()
		if stmt == nil && p.recoveryMode != PanicMode {
			p.recoverFromError("statement")
			// Try to continue parsing after recovery.
			if p.current.Type != lexer.TokenEOF {
				p.nextToken()

				return p.parseStatement()
			}
		}

		return stmt
	}
}

// match (expr) { pattern [if guard] => body, ... }.
func (p *Parser) parseMatchStatement() *MatchStatement {
	startPos := TokenToPosition(p.current)

	// Expect '(' then condition expression then ')'.
	if !p.expectPeek(lexer.TokenLParen) {
		return nil
	}

	p.nextToken()
	scrutinee := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TokenRParen) {
		return nil
	}

	// Expect '{'.
	if !p.expectPeek(lexer.TokenLBrace) {
		return nil
	}

	arms := make([]*MatchArm, 0)
	// Parse arms until '}'.
	for {
		// Advance to first token of arm or '}'.
		p.nextToken()

		if p.currentTokenIs(lexer.TokenRBrace) || p.currentTokenIs(lexer.TokenEOF) {
			break
		}
		// Skip trivia.
		if p.currentTokenIs(lexer.TokenWhitespace) || p.currentTokenIs(lexer.TokenNewline) || p.currentTokenIs(lexer.TokenComment) {
			continue
		}

		// Parse pattern as an expression (simple for now).
		pattern := p.parseExpression(LOWEST)

		// Optional guard: 'if' expr.
		var guard Expression

		if p.peekTokenIs(lexer.TokenIf) {
			p.nextToken() // move to 'if'
			p.nextToken() // move to start of guard expr
			guard = p.parseExpression(LOWEST)
		}

		// Expect '=>'.
		if !p.expectPeek(lexer.TokenFatArrow) {
			// Try to recover to next arrow or end of arm.
			for !p.peekTokenIs(lexer.TokenFatArrow) && !p.peekTokenIs(lexer.TokenComma) && !p.peekTokenIs(lexer.TokenRBrace) && !p.peekTokenIs(lexer.TokenEOF) {
				p.nextToken()
			}

			if p.peekTokenIs(lexer.TokenFatArrow) {
				p.nextToken()
			} else {
				// give up this arm.
				continue
			}
		}

		// Body: block statement or single statement/expression
		var body Statement

		if p.peekTokenIs(lexer.TokenLBrace) {
			p.nextToken()
			body = p.parseBlockStatement()
		} else {
			// Parse a single statement as arm body; allow expression statements.
			p.nextToken()

			body = p.parseStatement()
			if body == nil && p.current.Type != lexer.TokenRBrace {
				// Fallback to expression statement.
				expr := p.parseExpression(LOWEST)
				if expr != nil {
					body = &ExpressionStatement{Span: TokenToSpan(p.current), Expression: expr}
				}
			}
			// Optional trailing comma or semicolon; do not require.
			if p.peekTokenIs(lexer.TokenSemicolon) || p.peekTokenIs(lexer.TokenComma) {
				p.nextToken()
			}
		}

		arms = append(arms, &MatchArm{Span: TokenToSpan(p.current), Pattern: pattern, Guard: guard, Body: body})

		// If next is '}', end; if comma, continue to next arm.
		if p.peekTokenIs(lexer.TokenRBrace) {
			p.nextToken()

			break
		}
		// Consume trailing comma if present and continue.
		if p.peekTokenIs(lexer.TokenComma) {
			p.nextToken()

			continue
		}
		// Otherwise, loop continues; whitespace/newlines will be skipped at top
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &MatchStatement{Span: span, Expression: scrutinee, Arms: arms}
}

// parseForStatement parses a C-style for loop: for (init; cond; update) { ... }
// Each of init/cond/update is optional. The body must be a block.
func (p *Parser) parseForStatement() Statement {
	startPos := TokenToPosition(p.current)

	p.nextToken() // move past 'for'

	// Check if this is a for-in loop by looking for the pattern: identifier in expression
	if p.currentTokenIs(lexer.TokenIdentifier) && p.peekTokenIs(lexer.TokenIn) {
		return p.parseForInStatement(startPos)
	}

	// Otherwise, parse as C-style for loop with parentheses
	if !p.currentTokenIs(lexer.TokenLParen) {
		// This isn't a C-style for loop either, return nil to let error handling take over
		return nil
	}

	// Parse optional init statement until ';'.
	var init Statement
	// Move to the first token of init/semicolon
	p.nextToken()

	if !p.currentTokenIs(lexer.TokenSemicolon) {
		init = p.parseSimpleStatementUntilSemicolon()
	}
	// Current should be at semicolon; if not, try to recover.
	if !p.currentTokenIs(lexer.TokenSemicolon) {
		// attempt to sync by skipping to next ';' or ')'.
		for !p.currentTokenIs(lexer.TokenSemicolon) && !p.currentTokenIs(lexer.TokenRParen) && !p.currentTokenIs(lexer.TokenEOF) {
			p.nextToken()
		}
	}

	// Parse optional condition: advance to next token and read until ';'.
	var cond Expression

	if p.currentTokenIs(lexer.TokenSemicolon) {
		// move to condition start or next ';'.
		if !p.expectPeekRaw(lexer.TokenRParen) { // peek may be ')' or condition start
			p.nextToken()

			if !p.currentTokenIs(lexer.TokenSemicolon) && !p.currentTokenIs(lexer.TokenRParen) {
				cond = p.parseExpression(LOWEST)
			}
		}
	}
	// Ensure we are at the second semicolon (or ')').
	if !p.currentTokenIs(lexer.TokenSemicolon) {
		// Try to find semicolon.
		for !p.currentTokenIs(lexer.TokenSemicolon) && !p.currentTokenIs(lexer.TokenRParen) && !p.currentTokenIs(lexer.TokenEOF) {
			p.nextToken()
		}
	}

	// Parse optional update statement: after second ';' until ')'.
	var update Statement

	if p.currentTokenIs(lexer.TokenSemicolon) {
		// move to update start or ')'.
		if !p.expectPeekRaw(lexer.TokenRParen) { // peek may be ')' or start of update
			p.nextToken()

			if !p.currentTokenIs(lexer.TokenRParen) {
				update = p.parseSimpleStatementUntilRParen()
			}
		}
	}

	// We should be at ')'.
	if !p.currentTokenIs(lexer.TokenRParen) {
		// try to recover: advance to ')'.
		for !p.currentTokenIs(lexer.TokenRParen) && !p.currentTokenIs(lexer.TokenEOF) {
			p.nextToken()
		}
	}

	// Expect body block.
	if !p.expectPeek(lexer.TokenLBrace) {
		return nil
	}

	body := p.parseBlockStatement()

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &ForStatement{
		Span:      span,
		Init:      init,
		Condition: cond,
		Update:    update,
		Body:      body,
	}
}

// parseForInStatement parses a for-in loop: for item in collection { ... }
func (p *Parser) parseForInStatement(startPos Position) Statement {
	// Parse the loop variable
	if !p.currentTokenIs(lexer.TokenIdentifier) {
		return nil
	}

	variable := &Identifier{
		Span:  TokenToSpan(p.current),
		Value: p.current.Literal,
	}

	// Expect 'in'
	if !p.expectPeek(lexer.TokenIn) {
		return nil
	}

	// Parse the iterable expression
	p.nextToken()
	iterable := p.parseExpression(LOWEST)

	// Expect block
	if !p.expectPeek(lexer.TokenLBrace) {
		return nil
	}

	body := p.parseBlockStatement()

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	return &ForInStatement{
		Span:     span,
		Variable: variable,
		Iterable: iterable,
		Body:     body,
	}
}

// parseBreakStatement parses a break statement with optional label.
func (p *Parser) parseBreakStatement() *BreakStatement {
	startPos := TokenToPosition(p.current)

	var label *Identifier
	// Optional identifier label before ';' or '}'.
	if p.peekTokenIs(lexer.TokenIdentifier) {
		p.nextToken()
		label = NewIdentifier(TokenToSpan(p.current), p.current.Literal)
	}
	// Optional semicolon.
	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	}

	endPos := TokenToPosition(p.current)

	return &BreakStatement{Span: SpanBetween(startPos, endPos), Label: label}
}

// parseContinueStatement parses a continue statement with optional label.
func (p *Parser) parseContinueStatement() *ContinueStatement {
	startPos := TokenToPosition(p.current)

	var label *Identifier

	if p.peekTokenIs(lexer.TokenIdentifier) {
		p.nextToken()
		label = NewIdentifier(TokenToSpan(p.current), p.current.Literal)
	}

	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	}

	endPos := TokenToPosition(p.current)

	return &ContinueStatement{Span: SpanBetween(startPos, endPos), Label: label}
}

// parseSimpleStatementUntilSemicolon parses a minimal statement used in for-init until reaching ';'.
func (p *Parser) parseSimpleStatementUntilSemicolon() Statement {
	// Variable declaration or expression statement.
	switch p.current.Type {
	case lexer.TokenLet, lexer.TokenVar, lexer.TokenConst:
		stmt := p.parseVariableDeclaration()
		// Ensure current is at semicolon if present.
		if !p.currentTokenIs(lexer.TokenSemicolon) && p.peekTokenIs(lexer.TokenSemicolon) {
			p.nextToken()
		}

		return stmt
	default:
		expr := p.parseExpression(LOWEST)
		// consume up to semicolon.
		for !p.currentTokenIs(lexer.TokenSemicolon) && !p.currentTokenIs(lexer.TokenEOF) {
			if p.peekTokenIs(lexer.TokenSemicolon) {
				p.nextToken()

				break
			}

			p.nextToken()
		}

		return &ExpressionStatement{Span: TokenToSpan(p.current), Expression: expr}
	}
}

// parseSimpleStatementUntilRParen parses a minimal statement used in for-update until ')'.
func (p *Parser) parseSimpleStatementUntilRParen() Statement {
	switch p.current.Type {
	case lexer.TokenLet, lexer.TokenVar, lexer.TokenConst:
		stmt := p.parseVariableDeclaration()

		return stmt
	default:
		expr := p.parseExpression(LOWEST)
		// advance until ')'.
		for !p.currentTokenIs(lexer.TokenRParen) && !p.currentTokenIs(lexer.TokenEOF) {
			if p.peekTokenIs(lexer.TokenRParen) {
				p.nextToken()

				break
			}

			p.nextToken()
		}

		return &ExpressionStatement{Span: TokenToSpan(p.current), Expression: expr}
	}
}

// parseReturnStatement parses a return statement.
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

// parseIfStatement parses an if statement.
func (p *Parser) parseIfStatement() *IfStatement {
	startPos := TokenToPosition(p.current)

	p.nextToken()
	condition := p.parseExpression(LOWEST)

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

// parseWhileStatement parses a while statement.
func (p *Parser) parseWhileStatement() *WhileStatement {
	startPos := TokenToPosition(p.current)

	p.nextToken()
	condition := p.parseExpression(LOWEST)

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

// parseDeferStatement parses: defer { ... } | defer <statement-or-expr> ;.
func (p *Parser) parseDeferStatement() *DeferStatement {
	startPos := TokenToPosition(p.current)

	var body Statement

	if p.peekTokenIs(lexer.TokenLBrace) {
		p.nextToken()
		body = p.parseBlockStatement()
	} else {
		// Defer a single statement or expression.
		p.nextToken()

		body = p.parseStatement()
		if body == nil {
			// fallback: treat as expression.
			expr := p.parseExpression(LOWEST)
			if expr != nil {
				body = &ExpressionStatement{Span: TokenToSpan(p.current), Expression: expr}
			}
		}
		// Optional semicolon.
		if p.peekTokenIs(lexer.TokenSemicolon) {
			p.nextToken()
		}
	}

	endPos := TokenToPosition(p.current)

	return &DeferStatement{Span: SpanBetween(startPos, endPos), Body: body}
}

// parseLoopStatement parses an infinite loop: loop { ... }
// Lowered to WhileStatement with condition `true`.
func (p *Parser) parseLoopStatement() *WhileStatement {
	startPos := TokenToPosition(p.current)

	if !p.expectPeek(lexer.TokenLBrace) {
		return nil
	}

	body := p.parseBlockStatement()
	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)
	// Build a boolean literal 'true'.
	cond := NewLiteral(SpanBetween(startPos, startPos), true, LiteralBool)

	return &WhileStatement{Span: span, Condition: cond, Body: body}
}

// parseExpressionStatement parses an expression statement.
func (p *Parser) parseExpressionStatement() *ExpressionStatement {
	startPos := TokenToPosition(p.current)

	expr := p.parseExpression(LOWEST)
	if expr == nil {
		return nil
	}

	// Enhanced semicolon handling with error recovery (Phase 1.2.4)
	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	} else if p.peekTokenIs(lexer.TokenNewline) {
		// Treat newline as a valid statement terminator; keep suggestion for optional semicolon style.
		if p.suggestionEngine != nil {
			pos := TokenToPosition(p.current)
			pos.File = p.filename
			suggestion := Suggestion{
				Type:        ErrorFix,
				Message:     "Optional semicolon before newline",
				Position:    pos,
				Replacement: ";",
				Confidence:  0.4,
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

// ====== Expression Parsing (Pratt Parser) ======.

// Precedence levels for operators - Complete precedence hierarchy.
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

// precedences maps token types to their precedence levels.
// Following C-style operator precedence with some modern language improvements.
var precedences = map[lexer.TokenType]Precedence{
	// Assignment operators (right associative).
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

	// Ternary conditional (right associative).
	lexer.TokenQuestion: TERNARY,

	// Logical operators.
	lexer.TokenOr:  LOGICAL_OR,
	lexer.TokenAnd: LOGICAL_AND,

	// Bitwise operators.
	lexer.TokenBitOr:  BITWISE_OR,
	lexer.TokenBitXor: BITWISE_XOR,
	lexer.TokenBitAnd: BITWISE_AND,

	// Equality and relational operators.
	lexer.TokenEq: EQUALS,
	lexer.TokenNe: EQUALS,
	lexer.TokenLt: LESSGREATER,
	lexer.TokenLe: LESSGREATER,
	lexer.TokenGt: LESSGREATER,
	lexer.TokenGe: LESSGREATER,

	// Shift operators.
	lexer.TokenShl: SHIFT,
	lexer.TokenShr: SHIFT,

	// Additive operators.
	lexer.TokenPlus:  SUM,
	lexer.TokenMinus: SUM,

	// Multiplicative operators.
	lexer.TokenMul: PRODUCT,
	lexer.TokenDiv: PRODUCT,
	lexer.TokenMod: PRODUCT,

	// Power operator (right associative).
	lexer.TokenPower: POWER,

	// Call and access operators.
	lexer.TokenLParen:   CALL,
	lexer.TokenLBracket: CALL,
	lexer.TokenDot:      CALL,
}

// operatorAssociativity maps precedence levels to their associativity.
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

// peekPrecedence returns the precedence of the peek token.
func (p *Parser) peekPrecedence() Precedence {
	if p, ok := precedences[p.peek.Type]; ok {
		return p
	}

	return LOWEST
}

// currentPrecedence returns the precedence of the current token.
func (p *Parser) currentPrecedence() Precedence {
	if p, ok := precedences[p.current.Type]; ok {
		return p
	}

	return LOWEST
}

// parseExpression parses expressions using enhanced Pratt parsing with associativity.
func (p *Parser) parseExpression(precedence Precedence) Expression {
	// Parse prefix expression.
	left := p.parsePrefixExpression()
	if left == nil {
		return nil
	}

	// Parse infix expressions with associativity consideration.
	for !p.peekTokenIs(lexer.TokenSemicolon) && !p.peekTokenIs(lexer.TokenNewline) && p.shouldContinueParsing(precedence) {
		p.nextToken()

		left = p.parseInfixExpression(left)
		if left == nil {
			return nil
		}
	}

	return left
}

// shouldContinueParsing determines if parsing should continue based on precedence and associativity.
func (p *Parser) shouldContinueParsing(precedence Precedence) bool {
	peekPrec := p.peekPrecedence()

	// If peek precedence is lower, don't continue.
	if precedence > peekPrec {
		return false
	}

	// If precedences are equal, check associativity.
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
			// Non-associative operators like comparison chains.
			p.addError(TokenToPosition(p.peek),
				"non-associative operator cannot be chained",
				"expression parsing")

			return false
		}
	}

	return precedence < peekPrec
}

// parsePrefixExpression parses prefix expressions with extended operator support.
func (p *Parser) parsePrefixExpression() Expression {
	switch p.current.Type {
	case lexer.TokenIdentifier:
		// Check if this is a macro invocation pattern (identifier followed by !).
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
	case lexer.TokenTemplateString:
		return p.parseTemplateString()
	case lexer.TokenRawString:
		return p.parseRawString()
	case lexer.TokenTrue, lexer.TokenFalse:
		return p.parseBooleanLiteral()
	// Unary prefix operators.
	case lexer.TokenMinus, lexer.TokenPlus, lexer.TokenNot, lexer.TokenBitNot:
		return p.parseUnaryExpression()
	case lexer.TokenLParen:
		return p.parseGroupedExpression()
	// Refinement types: {n: Int | n > 0}.
	case lexer.TokenLBrace:
		return p.parseRefinementType()
	// Macro invocation starting with !.
	case lexer.TokenMacroInvoke:
		return p.parseMacroInvocation()
	// Attributes starting with #
	case lexer.TokenHash:
		return p.parseAttribute()
	// Arrays and slices starting with [
	case lexer.TokenLBracket:
		return p.parseArrayLiteral()
	// Bitwise operators as prefix
	case lexer.TokenBitAnd, lexer.TokenBitOr:
		return p.parseReferenceOrBitwiseExpression()
	// Keywords that can appear as expressions
	case lexer.TokenFor:
		return p.parseForExpression()
	case lexer.TokenMatch:
		return p.parseMatchExpression()
	case lexer.TokenIf:
		return p.parseIfExpression()
	case lexer.TokenWhile:
		return p.parseWhileExpression()
	case lexer.TokenAsync:
		return p.parseAsyncExpression()
	case lexer.TokenAwait:
		return p.parseAwaitExpression()
	case lexer.TokenUnsafe:
		return p.parseUnsafeExpression()
	// Types that can be used as expressions
	case lexer.TokenError:
		return p.parseErrorTypeExpression()
	case lexer.TokenMut:
		return p.parseMutExpression()
	// Special handling for tokens that might be misused as expressions
	case lexer.TokenIn:
		return p.parseInExpression()
	// Type casting and other keywords
	case lexer.TokenAs:
		return p.parseAsExpression()
	case lexer.TokenLet:
		return p.parseLetExpression()
	default:
		p.addError(TokenToPosition(p.current),
			fmt.Sprintf("no prefix parse function for %s", p.current.Type.String()),
			"expression parsing")

		return nil
	}
}

// parseIdentifier parses an identifier or a path expression (A::B::C).
func (p *Parser) parseIdentifier() Expression {
	// Start with the current identifier
	parts := []string{p.current.Literal}
	startPos := TokenToPosition(p.current)

	// Parse potential path segments: A::B::C
	for p.peekTokenIs(lexer.TokenDoubleColon) {
		p.nextToken() // consume current token
		p.nextToken() // consume '::'

		if !p.currentTokenIs(lexer.TokenIdentifier) {
			p.addError(TokenToPosition(p.current),
				"expected identifier after '::'",
				"path expression parsing")
			break
		}

		parts = append(parts, p.current.Literal)
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	// If we have multiple parts, create a path expression
	if len(parts) > 1 {
		// Use pooled string builder for efficient string concatenation
		sb := p.newPooledStringBuilder()
		for i, part := range parts {
			if i > 0 {
				sb.WriteString("::")
			}
			sb.WriteString(part)
		}
		value := sb.String()
		p.returnPooledStringBuilder(sb)

		// Use pooled identifier for better performance
		return p.newPooledIdentifier(span, value)
	}

	return NewIdentifier(TokenToSpan(p.current), p.current.Literal)
}

// parseIntegerLiteral parses an integer literal.
func (p *Parser) parseIntegerLiteral() Expression {
	value, err := strconv.ParseInt(p.current.Literal, 0, 64)
	if err != nil {
		p.addError(TokenToPosition(p.current),
			fmt.Sprintf("could not parse %q as integer", p.current.Literal),
			"integer parsing")

		return nil
	}

	// Use pooled literal for better performance
	return p.newPooledLiteral(TokenToSpan(p.current), value, LiteralInteger)
}

// parseFloatLiteral parses a float literal.
func (p *Parser) parseFloatLiteral() Expression {
	value, err := strconv.ParseFloat(p.current.Literal, 64)
	if err != nil {
		p.addError(TokenToPosition(p.current),
			fmt.Sprintf("could not parse %q as float", p.current.Literal),
			"float parsing")

		return nil
	}

	// Use pooled literal for better performance
	return p.newPooledLiteral(TokenToSpan(p.current), value, LiteralFloat)
}

// parseStringLiteral parses a string literal.
func (p *Parser) parseStringLiteral() Expression {
	// Use pooled literal for better performance
	return p.newPooledLiteral(TokenToSpan(p.current), p.current.Literal, LiteralString)
}

// parseTemplateString parses a template string with interpolation.
func (p *Parser) parseTemplateString() Expression {
	start := TokenToPosition(p.current)
	elements := make([]*TemplateElement, 0)

	// Parse template string content with interpolation support.
	content := p.current.Literal

	var i int

	for i < len(content) {
		// Look for ${.
		if i < len(content)-1 && content[i] == '$' && content[i+1] == '{' {
			// Find the end of interpolation.
			braceCount := 1
			j := i + 2

			for j < len(content) && braceCount > 0 {
				if content[j] == '{' {
					braceCount++
				} else if content[j] == '}' {
					braceCount--
				}

				j++
			}

			if braceCount == 0 {
				// Extract expression text.
				exprText := content[i+2 : j-1]

				// Create a mini-lexer for the expression.
				exprLexer := lexer.NewWithFilename(exprText, p.filename)
				exprParser := &Parser{
					lexer:    exprLexer,
					current:  exprLexer.NextToken(),
					filename: p.filename,
				}

				// Parse the expression.
				expr := exprParser.parseExpression(LOWEST)
				elements = append(elements, &TemplateElement{
					Span:       SpanBetween(start, TokenToPosition(p.current)),
					IsText:     false,
					Expression: expr,
				})

				i = j
			} else {
				// Malformed interpolation, treat as text.
				elements = append(elements, &TemplateElement{
					Span:   SpanBetween(start, TokenToPosition(p.current)),
					IsText: true,
					Text:   string(content[i]),
				})
				i++
			}
		} else {
			// Regular text - accumulate until next interpolation or end.
			textStart := i

			for i < len(content) && !(i < len(content)-1 && content[i] == '$' && content[i+1] == '{') {
				i++
			}

			if i > textStart {
				elements = append(elements, &TemplateElement{
					Span:   SpanBetween(start, TokenToPosition(p.current)),
					IsText: true,
					Text:   content[textStart:i],
				})
			}
		}
	}

	end := TokenToPosition(p.current)

	return &TemplateString{
		Span:     SpanBetween(start, end),
		Elements: elements,
	}
}

// parseRawString parses a raw string literal (r"...").
func (p *Parser) parseRawString() Expression {
	return NewLiteral(TokenToSpan(p.current), p.current.Literal, LiteralString)
}

// parseWhereClause parses a where clause: where T: Display + Debug, U: Clone.
func (p *Parser) parseWhereClause() []*WherePredicate {
	var predicates []*WherePredicate

	if p.current.Type != lexer.TokenWhere {
		return predicates
	}

	p.nextToken() // consume 'where'

	for {
		// Parse type.
		target := p.parseType()

		// Expect ':'.
		if p.current.Type != lexer.TokenColon {
			break
		}

		p.nextToken() // consume ':'

		// Parse bounds (Trait1 + Trait2 + ...)
		var bounds []Type

		for {
			bound := p.parseType()
			bounds = append(bounds, bound)

			if p.current.Type != lexer.TokenPlus {
				break
			}

			p.nextToken() // consume '+'
		}

		predicate := &WherePredicate{
			Span:   SpanBetween(TokenToPosition(p.current), TokenToPosition(p.current)),
			Target: target,
			Bounds: bounds,
		}
		predicates = append(predicates, predicate)

		// Check for comma to continue with next predicate.
		if p.current.Type != lexer.TokenComma {
			break
		}

		p.nextToken() // consume ','
	}

	return predicates
}

// parseGenericParameters parses generic parameter list: <T, U: Display, const N: usize>.
func (p *Parser) parseGenericParameters() []*GenericParameter {
	var params []*GenericParameter

	if p.current.Type != lexer.TokenLt {
		return params
	}

	p.nextToken() // consume '<'

	for p.current.Type != lexer.TokenGt && p.current.Type != lexer.TokenEOF {
		param := p.parseGenericParameter()
		if param != nil {
			params = append(params, param)
		}

		if p.current.Type == lexer.TokenComma {
			p.nextToken() // consume ','
		} else {
			break
		}
	}

	if p.current.Type == lexer.TokenGt {
		p.nextToken() // consume '>'
	}

	return params
}

// parseBooleanLiteral parses a boolean literal.
func (p *Parser) parseBooleanLiteral() Expression {
	value := p.current.Literal == "true"

	return NewLiteral(TokenToSpan(p.current), value, LiteralBool)
}

// parseUnaryExpression parses unary expressions.
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

// parseGroupedExpression parses grouped expressions.
func (p *Parser) parseGroupedExpression() Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TokenRParen) {
		return nil
	}

	return exp
}

// parseBinaryExpression parses binary expressions.
func (p *Parser) parseBinaryExpression(left Expression) Expression {
	// Critical nil check: prevent nil pointer dereference
	if left == nil {
		p.addError(TokenToPosition(p.current),
			"invalid left operand for binary expression",
			"Expected valid expression before binary operator")
		return nil
	}

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

// parseCallExpression parses function call expressions.
func (p *Parser) parseCallExpression(function Expression) Expression {
	// Critical nil check: prevent nil pointer dereference
	if function == nil {
		p.addError(TokenToPosition(p.current),
			"invalid function for call expression",
			"Expected valid function expression before '(' operator")
		return nil
	}

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

// parseCallArguments parses function call arguments.
func (p *Parser) parseCallArguments() []Expression {
	args := make([]Expression, 0)

	// Skip trivia between '(' and first argument or ')'.
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if p.peekTokenIs(lexer.TokenRParen) {
		p.nextToken()

		return args
	}

	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))

	for {
		// Skip trivia before checking for comma or closing paren.
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}

		if !p.peekTokenIs(lexer.TokenComma) {
			break
		}

		p.nextToken()
		// Skip trivia before next argument.
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}

		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(lexer.TokenRParen) {
		return nil
	}

	return args
}

// parseAssignmentExpression parses assignment expressions.
func (p *Parser) parseAssignmentExpression(left Expression) Expression {
	// Critical nil check: prevent nil pointer dereference
	if left == nil {
		p.addError(TokenToPosition(p.current),
			"invalid left operand for assignment",
			"Expected valid lvalue before assignment operator")
		return nil
	}

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

// parsePowerExpression parses power expressions (right associative).
func (p *Parser) parsePowerExpression(left Expression) Expression {
	// Critical nil check: prevent nil pointer dereference
	if left == nil {
		p.addError(TokenToPosition(p.current),
			"invalid left operand for power expression",
			"Expected valid expression before '**' operator")
		return nil
	}

	startPos := left.GetSpan().Start
	operator := NewOperator(TokenToSpan(p.current), p.current.Literal,
		int(p.currentPrecedence()), RightAssociative, BinaryOp)

	precedence := p.currentPrecedence()
	p.nextToken()
	// Right associative: use same precedence - 1 to parse right operand.
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
	// Critical nil check: prevent nil pointer dereference
	if left == nil {
		p.addError(TokenToPosition(p.current),
			"invalid left operand for compound assignment",
			"Expected valid lvalue before compound assignment operator")
		return nil
	}

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

// parseIndexExpression parses array/slice index expressions.
func (p *Parser) parseIndexExpression(left Expression) Expression {
	// Critical nil check: prevent nil pointer dereference
	if left == nil {
		p.addError(TokenToPosition(p.current),
			"invalid left operand for index expression",
			"Expected valid expression before '[' operator")
		return nil
	}

	startPos := left.GetSpan().Start

	p.nextToken() // consume [
	index := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TokenRBracket) {
		return nil
	}

	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	// Create a specialized index expression using CallExpression structure.
	return &CallExpression{
		Span:      span,
		Function:  left,
		Arguments: []Expression{index},
	}
}

// parseMemberExpression parses member access expressions (obj.field).
func (p *Parser) parseMemberExpression(left Expression) Expression {
	// Critical nil check: prevent nil pointer dereference
	if left == nil {
		p.addError(TokenToPosition(p.current),
			"invalid left operand for member access",
			"Expected valid expression before '.' operator")
		return nil
	}

	startPos := left.GetSpan().Start

	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	member := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
	endPos := TokenToPosition(p.current)
	span := SpanBetween(startPos, endPos)

	// Create a specialized member expression using BinaryExpression structure.
	operator := NewOperator(Span{Start: startPos, End: endPos}, ".",
		int(CALL), LeftAssociative, BinaryOp)

	return &BinaryExpression{
		Span:     span,
		Left:     left,
		Operator: operator,
		Right:    member,
	}
}

// parseTernaryExpression parses ternary conditional expressions (condition ? true_expr : false_expr).
func (p *Parser) parseTernaryExpression(left Expression) Expression {
	// Critical nil check: prevent nil pointer dereference
	if left == nil {
		p.addError(TokenToPosition(p.current),
			"invalid condition for ternary expression",
			"Expected valid expression before '?' operator")
		return nil
	}

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

	// Create ternary expression using specialized AST node.
	return &TernaryExpression{
		Span:      span,
		Condition: left,
		TrueExpr:  trueExpr,
		FalseExpr: falseExpr,
	}
}

// parseMacroDeclaration parses a macro definition.
func (p *Parser) parseMacroDeclaration() *MacroDefinition {
	startPos := TokenToPosition(p.current)

	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	// Parse parameters if present.
	var params []*MacroParameter

	if p.peek.Type == lexer.TokenLParen {
		p.nextToken()
		params = p.parseMacroParameters()

		if !p.expectPeek(lexer.TokenRParen) {
			return nil
		}
	}

	// Parse macro body.
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

// parseMacroParameters parses macro parameter list.
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

// parseMacroParameter parses a single macro parameter.
func (p *Parser) parseMacroParameter() *MacroParameter {
	if p.current.Type != lexer.TokenIdentifier {
		p.addError(TokenToPosition(p.current),
			"expected parameter name",
			"macro parameter parsing")

		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
	isVariadic := false

	// Check for variadic parameter (...)
	if p.peek.Type == lexer.TokenEllipsis {
		isVariadic = true

		p.nextToken() // consume ...
	}

	// Check for default value.
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
		IsVariadic:   isVariadic,
	}
}

// parseMacroBody parses the body of a macro definition.
func (p *Parser) parseMacroBody() *MacroBody {
	startPos := TokenToPosition(p.current)

	var templates []*MacroTemplate

	p.nextToken() // consume {

	for p.current.Type != lexer.TokenRBrace && p.current.Type != lexer.TokenEOF {
		// Skip trivia between templates.
		if p.current.Type == lexer.TokenWhitespace || p.current.Type == lexer.TokenComment || p.current.Type == lexer.TokenNewline || p.current.Type == lexer.TokenSemicolon {
			p.nextToken()

			continue
		}

		// Guard again in case we advanced to closing brace via trivia.
		if p.current.Type == lexer.TokenRBrace || p.current.Type == lexer.TokenEOF {
			break
		}

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

// parseMacroTemplate parses a single macro template.
func (p *Parser) parseMacroTemplate() *MacroTemplate {
	startPos := TokenToPosition(p.current)

	// Parse pattern.
	pattern := p.parseMacroPattern()
	if pattern == nil {
		return nil
	}

	// Parse guard if present.
	var guard Expression

	if p.current.Type == lexer.TokenIf {
		p.nextToken()
		guard = p.parseExpression(LOWEST)
	}

	// Ensure current token is the arrow '->'; accept if already at arrow,.
	// otherwise require the next token to be arrow.
	if p.current.Type != lexer.TokenArrow {
		if !p.expectPeek(lexer.TokenArrow) {
			return nil
		}
	}

	// Move to the start of the template body (token after '->').
	p.nextToken()

	var body []Statement

	if p.current.Type == lexer.TokenLBrace {
		// Block body.
		blockStmt := p.parseBlockStatement()
		if blockStmt != nil {
			body = blockStmt.Statements
		}
	} else {
		// Single statement or expression.
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

// parseMacroPattern parses a macro pattern.
func (p *Parser) parseMacroPattern() *MacroPattern {
	startPos := TokenToPosition(p.current)

	var elements []*MacroPatternElement

	// For now, implement basic pattern parsing.
	for p.current.Type != lexer.TokenArrow && p.current.Type != lexer.TokenIf && p.current.Type != lexer.TokenRBrace && p.current.Type != lexer.TokenEOF {
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

// parseMacroPatternElement parses a single pattern element.
func (p *Parser) parseMacroPatternElement() *MacroPatternElement {
	switch p.current.Type {
	case lexer.TokenIdentifier:
		// '_' as wildcard; other identifiers as parameter.
		if p.current.Literal == "_" {
			return &MacroPatternElement{Span: TokenToSpan(p.current), Kind: MacroPatternWildcard}
		}

		return &MacroPatternElement{
			Span:  TokenToSpan(p.current),
			Kind:  MacroPatternParameter,
			Value: p.current.Literal,
		}
	case lexer.TokenMul:
		// Wildcard pattern.
		return &MacroPatternElement{
			Span: TokenToSpan(p.current),
			Kind: MacroPatternWildcard,
		}
	default:
		// Literal pattern.
		return &MacroPatternElement{
			Span:  TokenToSpan(p.current),
			Kind:  MacroPatternLiteral,
			Value: p.current.Literal,
		}
	}
}

// parseMacroInvocation parses a macro invocation expression.
func (p *Parser) parseMacroInvocation() *MacroInvocation {
	startPos := TokenToPosition(p.current)

	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	// Parse arguments.
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

// parseMacroArguments parses macro invocation arguments.
func (p *Parser) parseMacroArguments() []*MacroArgument {
	var args []*MacroArgument

	// Skip trivia between '(' and first argument or ')'.
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if p.peek.Type == lexer.TokenRParen {
		return args
	}

	p.nextToken()

	for {
		if p.current.Type == lexer.TokenLBrace {
			// block literal as macro argument.
			block := p.parseBlockStatement()
			if block != nil {
				args = append(args, &MacroArgument{Span: block.Span, Kind: MacroArgBlock, Value: block})
			}
		} else {
			arg := p.parseMacroArgument()
			if arg != nil {
				args = append(args, arg)
			}
		}

		// Skip trivia before comma or closing paren.
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}

		if p.peek.Type != lexer.TokenComma {
			break
		}

		p.nextToken() // consume comma
		// Skip trivia before next argument.
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}

		p.nextToken() // move to next argument
	}

	return args
}

// parseMacroArgument parses a single macro argument.
func (p *Parser) parseMacroArgument() *MacroArgument {
	startPos := TokenToPosition(p.current)

	// For now, treat all arguments as expressions.
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

// parseMacroInvocationWithIdent parses macro invocation in the form: identifier!(args).
func (p *Parser) parseMacroInvocationWithIdent() *MacroInvocation {
	startPos := TokenToPosition(p.current)

	// Current token is identifier, next should be !.
	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	if !p.expectPeek(lexer.TokenMacroInvoke) {
		return nil
	}

	// Parse arguments.
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

// parseRefinementType parses refinement types: {n: Int | n > 0}.
func (p *Parser) parseRefinementType() Expression {
	start := TokenToPosition(p.current)

	// Skip opening brace.
	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}

	variable := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	if !p.expectPeek(lexer.TokenColon) {
		return nil
	}

	p.nextToken()
	baseType := p.parseType()

	if !p.expectPeek(lexer.TokenBitOr) {
		return nil
	}

	p.nextToken()
	constraint := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TokenRBrace) {
		return nil
	}

	end := TokenToPosition(p.current)
	span := SpanBetween(start, end)

	// Create refinement type expression.
	return &RefinementTypeExpression{
		Span:      span,
		Variable:  variable,
		BaseType:  baseType,
		Predicate: constraint,
	}
}

// parseAttribute parses attribute expressions like #[test].
func (p *Parser) parseAttribute() Expression {
	start := TokenToPosition(p.current)

	if !p.expectPeek(lexer.TokenLBracket) {
		p.addError(TokenToPosition(p.current), "expected '[' after '#'", "attribute parsing")
		return nil
	}

	if !p.expectPeek(lexer.TokenIdentifier) {
		p.addError(TokenToPosition(p.current), "expected identifier in attribute", "attribute parsing")
		return nil
	}

	name := p.current.Literal

	if !p.expectPeek(lexer.TokenRBracket) {
		p.addError(TokenToPosition(p.current), "expected ']' to close attribute", "attribute parsing")
		return nil
	}

	end := TokenToPosition(p.current)
	span := SpanBetween(start, end)

	// For now, create an identifier to represent attributes
	return NewIdentifier(span, "#["+name+"]")
}

// parseArrayLiteral parses array literals like [1, 2, 3].
func (p *Parser) parseArrayLiteral() Expression {
	start := TokenToPosition(p.current)
	elements := []Expression{}

	if p.peekTokenIs(lexer.TokenRBracket) {
		p.nextToken()
		end := TokenToPosition(p.current)
		span := SpanBetween(start, end)
		// Return array literal - for now use a call expression
		return &CallExpression{
			Function:  NewIdentifier(span, "Array"),
			Arguments: elements,
			Span:      span,
		}
	}

	p.nextToken()
	elements = append(elements, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.TokenComma) {
		p.nextToken()
		p.nextToken()
		elements = append(elements, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(lexer.TokenRBracket) {
		return nil
	}

	end := TokenToPosition(p.current)
	span := SpanBetween(start, end)

	// Return array literal - for now use a call expression
	return &CallExpression{
		Function:  NewIdentifier(span, "Array"),
		Arguments: elements,
		Span:      span,
	}
}

// parseReferenceOrBitwiseExpression parses reference (&) or bitwise expressions.
func (p *Parser) parseReferenceOrBitwiseExpression() Expression {
	start := TokenToPosition(p.current)
	operator := p.current.Literal

	p.nextToken()
	operand := p.parseExpression(PREFIX)

	if operand == nil {
		return nil
	}

	end := operand.GetSpan().End
	span := SpanBetween(start, end)

	// For now, create an identifier to represent bitwise operations
	return NewIdentifier(span, operator+operand.String())
}

// parseForExpression parses for expressions/statements as expressions.
func (p *Parser) parseForExpression() Expression {
	span := TokenToSpan(p.current)
	// For now, just return an identifier representing the for loop
	return NewIdentifier(span, "for_loop")
}

// parseMatchExpression parses match expressions.
func (p *Parser) parseMatchExpression() Expression {
	span := TokenToSpan(p.current)
	// For now, just return an identifier representing the match
	return NewIdentifier(span, "match_expr")
}

// parseIfExpression parses if expressions.
func (p *Parser) parseIfExpression() Expression {
	span := TokenToSpan(p.current)
	// For now, just return an identifier representing the if
	return NewIdentifier(span, "if_expr")
}

// parseWhileExpression parses while expressions.
func (p *Parser) parseWhileExpression() Expression {
	span := TokenToSpan(p.current)
	// For now, just return an identifier representing the while
	return NewIdentifier(span, "while_expr")
}

// parseAsyncExpression parses async expressions.
func (p *Parser) parseAsyncExpression() Expression {
	span := TokenToSpan(p.current)
	// For now, just return an identifier representing async
	return NewIdentifier(span, "async_expr")
}

// parseAwaitExpression parses await expressions.
func (p *Parser) parseAwaitExpression() Expression {
	span := TokenToSpan(p.current)
	// For now, just return an identifier representing await
	return NewIdentifier(span, "await_expr")
}

// parseUnsafeExpression parses unsafe expressions.
func (p *Parser) parseUnsafeExpression() Expression {
	span := TokenToSpan(p.current)
	// For now, just return an identifier representing unsafe
	return NewIdentifier(span, "unsafe_expr")
}

// parseErrorTypeExpression parses error type expressions.
func (p *Parser) parseErrorTypeExpression() Expression {
	span := TokenToSpan(p.current)
	// For now, just return an identifier representing error type
	return NewIdentifier(span, "Error")
}

// parseMutExpression parses mut expressions.
func (p *Parser) parseMutExpression() Expression {
	span := TokenToSpan(p.current)
	// For now, just return an identifier representing mut
	return NewIdentifier(span, "mut")
}

// parseInExpression parses in expressions.
func (p *Parser) parseInExpression() Expression {
	span := TokenToSpan(p.current)
	// For now, just return an identifier representing in
	return NewIdentifier(span, "in")
}

// parseRangeExpression parses range expressions.
func (p *Parser) parseRangeExpression() Expression {
	span := TokenToSpan(p.current)
	// For now, just return an identifier representing range
	return NewIdentifier(span, "range")
}

// parseRangeInfixExpression parses range expressions as infix operators.
func (p *Parser) parseRangeInfixExpression(left Expression) Expression {
	startSpan := left.GetSpan()
	rangeToken := p.current
	inclusive := false

	// Check if this is an inclusive range (..=)
	if rangeToken.Type == lexer.TokenRange && strings.HasSuffix(rangeToken.Literal, "=") {
		inclusive = true
	}

	// Advance past the range operator
	p.nextToken()

	// Parse the right side of the range
	right := p.parseExpression(LOWEST)
	if right == nil {
		p.addError(TokenToPosition(p.current),
			"expected expression after range operator",
			"Range expressions require both start and end values")
		return nil
	}

	endSpan := right.GetSpan()
	span := Span{Start: startSpan.Start, End: endSpan.End}

	return &RangeExpression{
		Start:     left,
		End:       right,
		Span:      span,
		Inclusive: inclusive,
	}
}

// parseAsExpression parses as expressions.
func (p *Parser) parseAsExpression() Expression {
	span := TokenToSpan(p.current)
	// For now, just return an identifier representing as
	return NewIdentifier(span, "as")
}

// parseLetExpression parses let expressions.
func (p *Parser) parseLetExpression() Expression {
	span := TokenToSpan(p.current)
	// For now, just return an identifier representing let
	return NewIdentifier(span, "let")
}
