// Package parser implements the Orizon recursive descent parser
// Phase 1.2.1: 再帰下降パーサー実装
package parser

import (
	"fmt"
	"strconv"
	"strings"

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

	// Memory guards: cap error/suggestion accumulation to avoid unbounded growth on pathological inputs
	maxErrors           int
	maxSuggestionsTotal int
	// internal flags for truncation (could be exposed later if needed)
	errorsTruncated      bool
	suggestionsTruncated bool
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

// NewParser creates a new parser instance
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
	// Set conservative yet safe caps to prevent runaway memory usage
	maxErrors:           5000,
	maxSuggestionsTotal: 5000,
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
	// Skip trivia on the peek side to allow newlines/whitespace/comments between tokens
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if p.peekTokenIs(tokenType) {
		p.nextToken()
		return true
	}
	p.peekError(tokenType)

	// Attempt error recovery if enabled (Phase 1.2.4)
	if p.recoveryMode != PanicMode && p.suggestionEngine != nil {
		// Track current token to detect lack of progress
		before := p.current
		_ = p.recoverFromError(fmt.Sprintf("expecting %s", tokenType.String()))
		// If recovery positioned the peek at expected, consume it
		if p.peekTokenIs(tokenType) {
			p.nextToken()
			return true
		}
		// If nothing changed, advance one token to avoid infinite loops
		if p.current == before && p.current.Type != lexer.TokenEOF {
			p.nextToken()
		}
	}

	return false
}

// expectPeekNoRecover advances if the peek token matches, without invoking recovery
func (p *Parser) expectPeekNoRecover(tokenType lexer.TokenType) bool {
	// Skip trivia on the peek side
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}

	if p.peekTokenIs(tokenType) {
		p.nextToken()
		return true
	}
	// Record error but do not attempt recovery here
	p.peekError(tokenType)
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
	// Respect error cap to limit memory under heavy error conditions
	if p.maxErrors > 0 && len(p.errors) >= p.maxErrors {
		p.errorsTruncated = true
		// Also stop generating suggestions to save memory/CPU
		p.suggestionEngine = nil
		return
	}
	pos.File = p.filename
	parseErr := &ParseError{
		Position: pos,
		Message:  message,
		Context:  context,
	}
	p.errors = append(p.errors, parseErr)

	// Generate suggestions using the error recovery system (Phase 1.2.4)
	if p.suggestionEngine != nil {
		// Respect suggestions cap
		if p.maxSuggestionsTotal > 0 && len(p.suggestions) >= p.maxSuggestionsTotal {
			p.suggestionsTruncated = true
			p.suggestionEngine = nil
		} else {
			newSuggestions := p.suggestionEngine.RecoverFromError(p, parseErr)
			// Ensure we don't exceed the cap even if engine returns up to max per call
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

// addErrorWithSuggestion adds an error with manual suggestions
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

// addErrorSilent adds an error without invoking the suggestion engine
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

// expectPeekRaw advances if the peek matches; does not record errors or suggestions
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

// SetErrorLimit sets a maximum number of errors to retain; 0 or negative disables the cap
func (p *Parser) SetErrorLimit(n int) {
	p.maxErrors = n
}

// SetSuggestionLimit sets a maximum number of suggestions to retain; 0 or negative disables the cap
func (p *Parser) SetSuggestionLimit(n int) {
	p.maxSuggestionsTotal = n
}

// DisableSuggestions turns off the suggestion engine entirely (saves memory/CPU)
func (p *Parser) DisableSuggestions() {
	p.suggestionEngine = nil
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

// skipToBefore advances tokens until the next token (peek) matches one of the given types,
// leaving current at the token right before the target so outer loops that call nextToken()
// will land exactly on the target.
func (p *Parser) skipToBefore(tokenTypes ...lexer.TokenType) {
	for !p.currentTokenIs(lexer.TokenEOF) {
		// If the next token is one of the targets, stop here
		for _, tokenType := range tokenTypes {
			if p.peekTokenIs(tokenType) {
				return
			}
		}
		p.nextToken()
	}
}

// skipLine advances the parser to the beginning of the next line
// by consuming tokens until a NEWLINE (inclusive) or EOF is reached.
func (p *Parser) skipLine() {
	for !p.currentTokenIs(lexer.TokenEOF) {
		if p.currentTokenIs(lexer.TokenNewline) {
			// Move to the first token on the next line
			p.nextToken()
			return
		}
		p.nextToken()
	}
}

// skipToNextTopLevelDecl advances to the next top-level declaration keyword
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
		// If the next token is a declaration starter, stop before it
		if p.peekTokenIs(lexer.TokenFunc) || p.peekTokenIs(lexer.TokenLet) || p.peekTokenIs(lexer.TokenVar) ||
			p.peekTokenIs(lexer.TokenConst) || p.peekTokenIs(lexer.TokenMacro) || p.peekTokenIs(lexer.TokenStruct) ||
			p.peekTokenIs(lexer.TokenEnum) || p.peekTokenIs(lexer.TokenTrait) || p.peekTokenIs(lexer.TokenImpl) ||
			p.peekTokenIs(lexer.TokenImport) || p.peekTokenIs(lexer.TokenExport) ||
			(p.peekTokenIs(lexer.TokenIdentifier) && p.peek.Literal == "type") {
			// Advance onto the declaration start to ensure forward progress
			p.nextToken()
			return
		}
		if p.peekTokenIs(lexer.TokenEOF) {
			// Advance to EOF to guarantee forward progress
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
		tok.Type == lexer.TokenImpl || tok.Type == lexer.TokenImport || tok.Type == lexer.TokenExport {
		return true
	}
	if tok.Type == lexer.TokenIdentifier && (tok.Literal == "type" || tok.Literal == "newtype" || tok.Literal == "fn") {
		return true
	}
	return false
}

// ====== Grammar Rules ======

// parseProgram parses the entire program
func (p *Parser) parseProgram() *Program {
	startPos := TokenToPosition(p.current)
	declarations := make([]Declaration, 0)

	for !p.currentTokenIs(lexer.TokenEOF) {
		// Skip whitespace and comments
		if p.currentTokenIs(lexer.TokenWhitespace) || p.currentTokenIs(lexer.TokenComment) || p.currentTokenIs(lexer.TokenNewline) {
			p.nextToken()
			continue
		}

		// Parse declaration
		if decl := p.parseDeclaration(); decl != nil {
			declarations = append(declarations, decl)
			// After a successful declaration, we're at the closing token of the decl
			// Advance once to move past it and continue
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
		fn := p.parseFunctionDeclaration()
		if fn == nil {
			// Let parseProgram handle synchronization to avoid double skipping
			p.addError(TokenToPosition(p.current), "failed to parse function declaration", "declaration parsing")
			return nil
		}
		decl = fn
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
	default:
		// Support 'type' alias declaration (lexer has no TokenType; detect identifier 'type')
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
		// Support 'newtype' nominal wrapper declaration (identifier literal)
		if p.current.Type == lexer.TokenIdentifier && p.current.Literal == "newtype" {
			nd := p.parseNewtypeDeclaration()
			if nd == nil {
				p.addError(TokenToPosition(p.current), "failed to parse newtype", "declaration parsing")
				return nil
			}
			nd.IsPublic = isPublic
			decl = nd
			break
		}
		// Allow top-level expression statements via Pratt parser
		if exprStmt := p.parseExpressionStatement(); exprStmt != nil {
			return exprStmt
		}
		// If expression parsing failed, record an error and let parseProgram resynchronize
		p.addError(TokenToPosition(p.current),
			fmt.Sprintf("unexpected token %s in declaration", p.current.Type.String()),
			"declaration parsing")
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
	case *ImportDeclaration:
		d.IsPublic = isPublic
	case *ExportDeclaration:
		// visibility inherent to export list
	case *StructDeclaration:
		d.IsPublic = isPublic
	case *EnumDeclaration:
		d.IsPublic = isPublic
	case *TraitDeclaration:
		d.IsPublic = isPublic
	case *ImplBlock:
		// no modifiers applicable
	}
	return decl
}

// parseTypeAliasDeclaration parses: type Name = Type ;
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
	// optional semicolon
	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	}
	end := TokenToPosition(p.current)
	return &TypeAliasDeclaration{Span: SpanBetween(start, end), Name: name, Aliased: aliased}
}

// parseNewtypeDeclaration parses: newtype Name = Type ;
func (p *Parser) parseNewtypeDeclaration() *NewtypeDeclaration {
	start := TokenToPosition(p.current)
	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}
	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
	if !p.expectPeek(lexer.TokenAssign) {
		return nil
	}
	p.nextToken()
	base := p.parseType()
	// optional semicolon
	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	}
	end := TokenToPosition(p.current)
	return &NewtypeDeclaration{Span: SpanBetween(start, end), Name: name, Base: base}
}

// parseImportDeclaration: import path [as alias] ;? (semicolon optional)
func (p *Parser) parseImportDeclaration() *ImportDeclaration {
	start := TokenToPosition(p.current)
	// Parse path segments: ident { :: ident }
	if !p.expectPeekRaw(lexer.TokenIdentifier) {
		// Local recovery: malformed import head. Sync to ';' or next top-level start and bail.
		p.addErrorSilent(TokenToPosition(p.peek), "expected module path after 'import'", "import parsing")
		p.skipToBefore(
			lexer.TokenSemicolon,
			lexer.TokenFunc, lexer.TokenLet, lexer.TokenVar, lexer.TokenConst,
			lexer.TokenMacro, lexer.TokenStruct, lexer.TokenEnum, lexer.TokenTrait, lexer.TokenImpl,
			lexer.TokenImport, lexer.TokenExport,
		)
		// Optionally consume semicolon if present to finish the bad import
		if p.peekTokenIs(lexer.TokenSemicolon) {
			p.nextToken()
		}
		return nil
	}
	path := []*Identifier{NewIdentifier(TokenToSpan(p.current), p.current.Literal)}
	for p.peekTokenIs(lexer.TokenDoubleColon) {
		p.nextToken() // consume ::
		// Support minimal wildcard import: module::*
		// If the next token is '*', consume it and treat as importing all items from the module.
		// For MVP we don't need to store this in the AST, as HIR import with Items=nil already
		// represents importing all; we only need to accept the syntax and advance tokens.
		if p.peekTokenIs(lexer.TokenMul) {
			p.nextToken() // consume '*'
			break         // stop extending the path; wildcard ends the path
		}
		if !p.expectPeekRaw(lexer.TokenIdentifier) {
			// Malformed segment after '::'. Report and sync locally, then bail to let outer loop recover.
			p.addErrorSilent(TokenToPosition(p.peek), "expected identifier after '::' in import path", "import parsing")
			// Sync to ';' or before the next top-level start so we don't swallow following declarations
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
	// Optional alias: as ident
	var alias *Identifier
	if p.peekTokenIs(lexer.TokenAs) {
		p.nextToken() // move to 'as'
		if !p.expectPeekRaw(lexer.TokenIdentifier) {
			// Recover: missing alias name, sync to end of import and bail
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
	// Optional semicolon
	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	}
	end := TokenToPosition(p.current)
	return &ImportDeclaration{Span: SpanBetween(start, end), Path: path, Alias: alias}
}

// parseExportDeclaration: export { id[, id]* } ;?  (for now only list form)
func (p *Parser) parseExportDeclaration() *ExportDeclaration {
	start := TokenToPosition(p.current)
	// Support two forms later: export <item>; and export { a, b }
	if !p.expectPeek(lexer.TokenLBrace) {
		// For MVP, allow single identifier: export foo;
		if p.peekTokenIs(lexer.TokenIdentifier) {
			p.nextToken()
			name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
			// Optional alias via 'as' (not in EBNF list form, but keep for future)
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
	// parse list
	items := make([]*ExportItem, 0)
	for {
		if !p.expectPeek(lexer.TokenIdentifier) {
			return nil
		}
		name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
		items = append(items, &ExportItem{Span: name.Span, Name: name})
		// trailing comma or closing brace
		if p.peekTokenIs(lexer.TokenComma) {
			p.nextToken()
			continue
		}
		if p.peekTokenIs(lexer.TokenRBrace) {
			p.nextToken()
			break
		}
		// tolerate newlines
		if p.peekTokenIs(lexer.TokenNewline) {
			p.nextToken()
			continue
		}
		// otherwise error
		p.addError(TokenToPosition(p.peek), "expected ',' or '}' in export list", "export parsing")
		return nil
	}
	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	}
	end := TokenToPosition(p.current)
	return &ExportDeclaration{Span: SpanBetween(start, end), Items: items}
}

// parseStructDeclaration: struct Name { fields }
func (p *Parser) parseStructDeclaration() *StructDeclaration {
	start := TokenToPosition(p.current)
	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}
	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
	// Optional generics
	gens := p.parseOptionalGenericParameters()
	// Optional body or ';' forward decl; MVP: require body
	if !p.expectPeek(lexer.TokenLBrace) {
		return nil
	}
	fields := make([]*StructField, 0)
	closed := false
	// Parse zero or more field declarations until '}'
	for {
		p.nextToken()
		if p.currentTokenIs(lexer.TokenRBrace) {
			closed = true
			break
		}
		if p.currentTokenIs(lexer.TokenEOF) {
			// EOF hit before closing brace
			p.addErrorSilent(TokenToPosition(p.current), "missing '}' to close struct block", "struct parsing")
			return nil
		}
		// Skip trivia
		if p.currentTokenIs(lexer.TokenWhitespace) || p.currentTokenIs(lexer.TokenNewline) || p.currentTokenIs(lexer.TokenComment) {
			continue
		}
		// If we see a token that starts a new top-level declaration, assume the struct block was not closed and bail.
		if p.isTopLevelStart(p.current) {
			p.addErrorSilent(TokenToPosition(p.current), "unexpected top-level declaration inside struct; missing '}'?", "struct parsing")
			return nil
		}
		// Optional 'pub'
		isPub := false
		if p.currentTokenIs(lexer.TokenPub) {
			isPub = true
			p.nextToken()
		}
		if !p.currentTokenIs(lexer.TokenIdentifier) {
			p.addErrorSilent(TokenToPosition(p.current), "expected field name", "struct field parsing")
			// try skip to next comma or '}'
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
			// Improve block-level recovery: report and sync to next field or end of struct
			p.addErrorSilent(TokenToPosition(p.peek), "expected ':' after struct field name", "struct field parsing")
			p.skipTo(lexer.TokenComma, lexer.TokenRBrace)
			if p.currentTokenIs(lexer.TokenRBrace) {
				closed = true
				break
			}
			// If we landed on a comma, continue to next field
			if p.currentTokenIs(lexer.TokenComma) {
				continue
			}
			continue
		}
		p.nextToken()
		ftype := p.parseType()
		field := &StructField{Span: SpanBetween(fname.Span.Start, ftype.GetSpan().End), Name: fname, Type: ftype, IsPublic: isPub}
		fields = append(fields, field)
		// Expect comma or '}'
		if p.peekTokenIs(lexer.TokenComma) {
			p.nextToken()
			continue
		}
		if p.peekTokenIs(lexer.TokenRBrace) {
			p.nextToken()
			closed = true
			break
		}
		// tolerate newline
		if p.peekTokenIs(lexer.TokenNewline) {
			p.nextToken()
			continue
		}
	}
	if !closed {
		// As a safety, if we somehow exited without marking closed, report and fail
		p.addErrorSilent(TokenToPosition(p.current), "unterminated struct declaration", "struct parsing")
		return nil
	}
	end := TokenToPosition(p.current)
	return &StructDeclaration{Span: SpanBetween(start, end), Name: name, Fields: fields, Generics: gens}
}

// parseEnumDeclaration: enum Name { Variants }
func (p *Parser) parseEnumDeclaration() *EnumDeclaration {
	start := TokenToPosition(p.current)
	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}
	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
	// Optional generics
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
			// attempt to sync
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
		// Optional data: (types) or { fields }
		if p.peekTokenIs(lexer.TokenLParen) {
			// tuple-like
			p.nextToken() // '('
			fields := make([]*StructField, 0)
			// parse type list
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
			// struct-like
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
					// Recover within variant field list
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
				// tolerate newline and other trivia; otherwise try to resync
				if p.peekTokenIs(lexer.TokenNewline) || p.peekTokenIs(lexer.TokenWhitespace) || p.peekTokenIs(lexer.TokenComment) {
					p.nextToken()
					continue
				}
				// Unexpected token: resync to next field or end
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
		// trailing comma or '}'
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

// parseTraitDeclaration: trait Name { method signatures }
func (p *Parser) parseTraitDeclaration() *TraitDeclaration {
	start := TokenToPosition(p.current)
	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}
	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
	// Optional generics
	gens := p.parseOptionalGenericParameters()
	// Optional bounds after ':' (not fully used yet)
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
		// Disallow unrelated top-level starts inside trait, but allow valid trait items: 'func' and associated 'type'
		if p.isTopLevelStart(p.current) && !(p.current.Type == lexer.TokenFunc || (p.current.Type == lexer.TokenIdentifier && p.current.Literal == "type")) {
			p.addErrorSilent(TokenToPosition(p.current), "unexpected top-level declaration inside trait; missing '}'?", "trait parsing")
			return nil
		}
		// Associated type: 'type' identifier [ : bounds ] ;
		// Lexer doesn't have a dedicated 'type' token yet; detect identifier literal "type".
		if p.currentTokenIs(lexer.TokenIdentifier) && p.current.Literal == "type" {
			if !p.expectPeekRaw(lexer.TokenIdentifier) {
				// Recover to ';' or '}' to continue parsing remaining items
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
			// Optional method-level generics
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
			// Optional return type
			var ret Type
			for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
				p.nextToken()
			}
			if p.peekTokenIs(lexer.TokenArrow) {
				p.nextToken()
				p.nextToken()
				ret = p.parseType()
			}
			// optional semicolon
			if p.peekTokenIs(lexer.TokenSemicolon) {
				p.nextToken()
			}
			methods = append(methods, &TraitMethod{Span: mname.Span, Name: mname, Parameters: params, ReturnType: ret, Generics: gens})
			continue
		}
		p.addErrorSilent(TokenToPosition(p.current), "expected 'func' or 'type' in trait body", "trait parsing")
		// sync to next possible item
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

// parseImplBlock: impl [Trait for] Type { func ... }
func (p *Parser) parseImplBlock() *ImplBlock {
	start := TokenToPosition(p.current)
	// Optional generics: impl <T, ...>
	gens := p.parseOptionalGenericParameters()
	// Two forms: impl Type { ... }  |  impl Trait for Type { ... }
	// Peek to decide if a trait path appears before 'for'
	// Move to next token to start type/trait
	if !p.expectPeek(lexer.TokenIdentifier) {
		return nil
	}
	// Parse a simple path or type name as first part, then optional generic suffix
	firstType := p.parsePathOrBasicType()
	firstType = p.parseGenericSuffixOn(firstType)
	var trait Type
	var forType Type
	if p.peekTokenIs(lexer.TokenFor) {
		// trait for type form
		trait = firstType
		p.nextToken() // move to 'for'
		if !p.expectPeek(lexer.TokenIdentifier) {
			return nil
		}
		forType = p.parsePathOrBasicType()
		forType = p.parseGenericSuffixOn(forType)
	} else {
		// inherent impl
		forType = firstType
	}
	// Optional where clause
	where := p.parseOptionalWhereClause()
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
		// Expect function declarations only for MVP
		if p.currentTokenIs(lexer.TokenFunc) || (p.currentTokenIs(lexer.TokenIdentifier) && p.current.Literal == "fn") {
			fn := p.parseFunctionDeclaration()
			if fn != nil {
				items = append(items, fn)
			}
			continue
		}
		// skip unknown item until next '}' or 'func'
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
		// If we stopped because '}' is next, the outer loop will handle it on next iteration
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

	// Accept both 'func' keyword and 'fn' identifier as an alias for function declarations
	isFuncKw := p.currentTokenIs(lexer.TokenFunc) || (p.currentTokenIs(lexer.TokenIdentifier) && p.current.Literal == "fn")
	if !isFuncKw {
		return nil
	}

	if !p.expectPeekRaw(lexer.TokenIdentifier) {
		p.addErrorSilent(TokenToPosition(p.peek), "expected identifier after 'func'", "function declaration")
		// consume rest of line to avoid getting stuck at random identifiers
		for !p.currentTokenIs(lexer.TokenEOF) && !p.currentTokenIs(lexer.TokenNewline) {
			p.nextToken()
		}
		return nil
	}

	name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)

	// Optional generics after function name
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

	// Optional return type
	var returnType Type
	// Allow trivia (including newlines) between ')' and '->'
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}
	if p.peekTokenIs(lexer.TokenArrow) {
		p.nextToken() // consume arrow
		p.nextToken() // move to type
		returnType = p.parseType()
	}

	// Allow trivia before function body '{'
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
		Span:       span,
		Name:       name,
		Parameters: parameters,
		ReturnType: returnType,
		Body:       body,
		IsPublic:   false,
		IsAsync:    false,
		Generics:   gens,
	}
}

// parseParameterList parses function parameters
func (p *Parser) parseParameterList() []*Parameter {
	parameters := make([]*Parameter, 0)

	// Allow empty parameter list with intervening trivia
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}
	if p.peekTokenIs(lexer.TokenRParen) {
		return parameters
	}

	// Move to first parameter token (skipping trivia already handled above)
	p.nextToken()

	// Parse first parameter
	param := p.parseParameter()
	if param != nil {
		parameters = append(parameters, param)
	}

	// Parse remaining parameters
	for {
		// Skip trivia before checking for comma or closing paren
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}
		if !p.peekTokenIs(lexer.TokenComma) {
			break
		}
		p.nextToken() // consume comma
		// Skip trivia before next parameter
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

	// Expect semicolon (or allow newline with suggestion). Handle cases where semicolon is current.
	if p.currentTokenIs(lexer.TokenSemicolon) {
		// already at semicolon (e.g., after recovering), keep as is
	} else if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	} else if p.peekTokenIs(lexer.TokenNewline) {
		// Provide suggestion to insert semicolon before newline
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
		// Public visibility is applied by parseDeclaration based on leading modifiers
		IsPublic: false,
	}
}

// parseType parses a type specification
func (p *Parser) parseType() Type {
	// Skip trivia at type start if present
	for p.current.Type == lexer.TokenWhitespace || p.current.Type == lexer.TokenComment || p.current.Type == lexer.TokenNewline {
		p.nextToken()
	}

	// Helper to optionally parse generic arguments after a base type
	parseGenericSuffix := func(base Type) Type {
		// Skip trivia
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}
		if !p.peekTokenIs(lexer.TokenLt) {
			return base
		}
		// consume '<'
		p.nextToken()
		// move to first type or '>'
		p.nextToken()
		typeParams := make([]Type, 0)
		// Allow empty generic arg list is invalid; enforce at least one type
		if !p.currentTokenIs(lexer.TokenGt) {
			// parse first type
			tp := p.parseType()
			if tp != nil {
				typeParams = append(typeParams, tp)
			}
			// parse remaining , type
			for {
				// Skip trivia before checking comma or '>'
				for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
					p.nextToken()
				}
				if !p.peekTokenIs(lexer.TokenComma) {
					break
				}
				p.nextToken() // consume comma
				// Skip trivia then move to next type
				for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
					p.nextToken()
				}
				p.nextToken()
				tp = p.parseType()
				if tp != nil {
					typeParams = append(typeParams, tp)
				}
			}
			// Expect '>'
			if !p.expectPeekRaw(lexer.TokenGt) {
				p.addError(TokenToPosition(p.peek), "expected '>' to close generic arguments", "type parsing")
				return base
			}
		}
		// current is now at '>' after expectPeekRaw
		span := SpanBetween(base.(TypeSafeNode).GetSpan().Start, TokenToSpan(p.current).End)
		return &GenericType{Span: span, BaseType: base, TypeParameters: typeParams}
	}

	switch p.current.Type {
	case lexer.TokenBitAnd:
		// reference type: &T, &mut T, &'a T, &'a mut T
		start := TokenToPosition(p.current)
		// optional whitespace/comments/newlines after '&'
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}
		// consume next token (potentially 'mut' or start of type)
		p.nextToken()
		// optional lifetime token
		lifetime := ""
		if p.current.Type == lexer.TokenLifetime {
			lifetime = p.current.Literal
			// skip trivia after lifetime
			for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
				p.nextToken()
			}
			p.nextToken()
		}
		isMut := false
		if p.current.Type == lexer.TokenMut {
			isMut = true
			// move to the inner type start
			p.nextToken()
		}
		inner := p.parseType()
		end := TokenToPosition(p.current)
		return &ReferenceType{Span: SpanBetween(start, end), Inner: inner, IsMutable: isMut, Lifetime: lifetime}
	case lexer.TokenMul:
		// pointer type: *T or *mut T
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
		// Support path types: A::B::C and apply generic suffix to the tail
		base := p.parsePathOrBasicType()
		return parseGenericSuffix(base)
	case lexer.TokenLBracket:
		// Array or slice type: [T] (slice) or [T; N] (array)
		start := TokenToPosition(p.current)
		p.nextToken() // move to element type
		elem := p.parseType()
		// After element type, allow trivia
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}
		if p.peekTokenIs(lexer.TokenSemicolon) {
			// static array with size expression
			p.nextToken() // consume ';'
			p.nextToken() // move to size expression
			sizeExpr := p.parseExpression(LOWEST)
			if !p.expectPeekRaw(lexer.TokenRBracket) {
				p.addError(TokenToPosition(p.peek), "expected ']' to close array type", "type parsing")
			}
			end := TokenToPosition(p.current)
			return &ArrayType{Span: SpanBetween(start, end), ElementType: elem, Size: sizeExpr, IsDynamic: false}
		}
		// Expect ']' for slice type
		if !p.expectPeekRaw(lexer.TokenRBracket) {
			p.addError(TokenToPosition(p.peek), "expected ']' to close slice type", "type parsing")
		}
		end := TokenToPosition(p.current)
		return &ArrayType{Span: SpanBetween(start, end), ElementType: elem, Size: nil, IsDynamic: true}
	case lexer.TokenAsync, lexer.TokenFunc:
		// function type: [async] func(paramList) [-> type]
		start := TokenToPosition(p.current)
		isAsync := false
		if p.current.Type == lexer.TokenAsync {
			isAsync = true
			// ensure 'func' follows
			for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
				p.nextToken()
			}
			if !p.peekTokenIs(lexer.TokenFunc) {
				p.addError(TokenToPosition(p.current), "expected 'func' after 'async' in type", "type parsing")
				return nil
			}
			p.nextToken() // move to TokenFunc
		}
		if !p.expectPeekRaw(lexer.TokenLParen) {
			p.addError(TokenToPosition(p.peek), "expected '(' after 'func' in type", "type parsing")
			return nil
		}
		params := make([]*FunctionTypeParameter, 0)
		// Allow empty parameter list
		// Skip trivia
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}
		if !p.peekTokenIs(lexer.TokenRParen) {
			// Move to first token within params
			p.nextToken()
			// parse first param
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
			// remaining params
			for {
				// Skip trivia before comma or ')'
				for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
					p.nextToken()
				}
				if !p.peekTokenIs(lexer.TokenComma) {
					break
				}
				p.nextToken() // consume ','
				// Skip trivia then move to next param start
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
		// Optional return type: allow trivia then '->'
		// Skip trivia
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}
		var ret Type
		if p.peekTokenIs(lexer.TokenArrow) {
			p.nextToken() // consume '->'
			p.nextToken() // move to type
			ret = p.parseType()
		} else {
			// If unspecified, leave nil (interpreted as unit later)
			ret = nil
		}
		end := TokenToPosition(p.current)
		return &FunctionType{Span: SpanBetween(start, end), Parameters: params, ReturnType: ret, IsAsync: isAsync}
	default:
		p.addError(TokenToPosition(p.current),
			fmt.Sprintf("unexpected token %s in type", p.current.Type.String()),
			"type parsing")
		return nil
	}
}

// parsePathOrBasicType parses a possibly qualified path type like A::B and returns a BasicType with joined name for now
func (p *Parser) parsePathOrBasicType() Type {
	// current is identifier at head
	start := TokenToPosition(p.current)
	parts := []string{p.current.Literal}
	// Accumulate segments
	for p.peekTokenIs(lexer.TokenDoubleColon) {
		p.nextToken() // consume '::'
		if !p.expectPeek(lexer.TokenIdentifier) {
			break
		}
		parts = append(parts, p.current.Literal)
	}
	// For MVP, represent path as BasicType with qualified name (resolution later)
	bt := &BasicType{Span: SpanBetween(start, TokenToPosition(p.current)), Name: strings.Join(parts, "::")}
	return bt
}

// parseGenericSuffixOn applies a generic argument suffix like <T, U> to the given base type if the next token is '<'.
func (p *Parser) parseGenericSuffixOn(base Type) Type {
	// Skip trivia
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}
	if !p.peekTokenIs(lexer.TokenLt) {
		return base
	}
	// consume '<'
	p.nextToken()
	// move to first type or '>'
	p.nextToken()
	typeParams := make([]Type, 0)
	if !p.currentTokenIs(lexer.TokenGt) {
		// parse first type
		tp := p.parseType()
		if tp != nil {
			typeParams = append(typeParams, tp)
		}
		// parse remaining , type
		for {
			// Skip trivia before checking comma or '>'
			for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
				p.nextToken()
			}
			if !p.peekTokenIs(lexer.TokenComma) {
				break
			}
			p.nextToken() // consume comma
			// Skip trivia then move to next type
			for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
				p.nextToken()
			}
			p.nextToken()
			tp = p.parseType()
			if tp != nil {
				typeParams = append(typeParams, tp)
			}
		}
		// Expect '>'
		if !p.expectPeekRaw(lexer.TokenGt) {
			p.addError(TokenToPosition(p.peek), "expected '>' to close generic arguments", "type parsing")
			return base
		}
	}
	span := SpanBetween(base.(TypeSafeNode).GetSpan().Start, TokenToSpan(p.current).End)
	return &GenericType{Span: span, BaseType: base, TypeParameters: typeParams}
}

// ====== Generics / Where / Bounds Parsing ======

// parseOptionalGenericParameters parses '<...>' if present
func (p *Parser) parseOptionalGenericParameters() []*GenericParameter {
	gens := []*GenericParameter{}
	// Skip trivia
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}
	if !p.peekTokenIs(lexer.TokenLt) {
		return gens
	}
	p.nextToken() // move to '<'
	// Move to first content or '>'
	p.nextToken()
	if p.currentTokenIs(lexer.TokenGt) {
		return gens
	}
	// Parse first param
	gp := p.parseGenericParameter()
	if gp != nil {
		gens = append(gens, gp)
	}
	// Remaining params
	for {
		// Skip trivia
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}
		if !p.peekTokenIs(lexer.TokenComma) {
			break
		}
		p.nextToken() // consume ','
		// Skip trivia then move to next param
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}
		p.nextToken()
		gp = p.parseGenericParameter()
		if gp != nil {
			gens = append(gens, gp)
		}
	}
	// Expect '>'
	if !p.expectPeekRaw(lexer.TokenGt) {
		p.addError(TokenToPosition(p.peek), "expected '>' to close generics", "generics parsing")
	}
	return gens
}

// parseGenericParameter parses one of: identifier [":" bounds] | const identifier ":" type | lifetime
func (p *Parser) parseGenericParameter() *GenericParameter {
	start := TokenToPosition(p.current)
	// Lifetime parameter: TokenLifetime
	if p.current.Type == lexer.TokenLifetime {
		return &GenericParameter{Span: SpanBetween(start, TokenToPosition(p.current)), Kind: GenericParamLifetime, Lifetime: p.current.Literal}
	}
	// Const parameter: 'const' ident ':' type
	if p.current.Type == lexer.TokenConst {
		if !p.expectPeek(lexer.TokenIdentifier) {
			return nil
		}
		name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
		if !p.expectPeek(lexer.TokenColon) {
			return nil
		}
		p.nextToken()
		ctype := p.parseType()
		return &GenericParameter{Span: SpanBetween(start, TokenToPosition(p.current)), Kind: GenericParamConst, Name: name, ConstType: ctype}
	}
	// Type parameter: ident [":" bounds]
	if p.current.Type == lexer.TokenIdentifier {
		name := NewIdentifier(TokenToSpan(p.current), p.current.Literal)
		bounds := []Type{}
		if p.peekTokenIs(lexer.TokenColon) {
			p.nextToken()
			p.nextToken()
			bounds = p.parseTraitBounds()
		}
		return &GenericParameter{Span: SpanBetween(start, TokenToPosition(p.current)), Kind: GenericParamType, Name: name, Bounds: bounds}
	}
	p.addError(TokenToPosition(p.current), "invalid generic parameter", "generics parsing")
	return nil
}

// parseTraitBounds parses trait_bound { '+' trait_bound }
func (p *Parser) parseTraitBounds() []Type {
	bounds := []Type{}
	// first bound
	b := p.parseType()
	if b != nil {
		bounds = append(bounds, b)
	}
	for {
		// Skip trivia
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

// parseOptionalWhereClause parses where_clause if present: 'where' pred {',' pred}
func (p *Parser) parseOptionalWhereClause() []*WherePredicate {
	preds := []*WherePredicate{}
	// Skip trivia
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}
	if !p.peekTokenIs(lexer.TokenWhere) {
		return preds
	}
	p.nextToken() // move to 'where'
	// Move to first predicate
	p.nextToken()
	pred := p.parseWherePredicate()
	if pred != nil {
		preds = append(preds, pred)
	}
	for {
		// Skip trivia
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}
		if !p.peekTokenIs(lexer.TokenComma) {
			break
		}
		p.nextToken()
		// Skip trivia then move to next predicate
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}
		p.nextToken()
		pred = p.parseWherePredicate()
		if pred != nil {
			preds = append(preds, pred)
		}
	}
	return preds
}

// parseWherePredicate parses: type ':' trait_bounds
func (p *Parser) parseWherePredicate() *WherePredicate {
	start := TokenToPosition(p.current)
	t := p.parseType()
	if t == nil {
		return nil
	}
	if !p.expectPeek(lexer.TokenColon) {
		return nil
	}
	p.nextToken()
	bounds := p.parseTraitBounds()
	return &WherePredicate{Span: SpanBetween(start, TokenToPosition(p.current)), Target: t, Bounds: bounds}
}

// parseBlockStatement parses a block statement
func (p *Parser) parseBlockStatement() *BlockStatement {
	startPos := TokenToPosition(p.current)
	statements := make([]Statement, 0)

	p.nextToken() // consume opening brace

	for !p.currentTokenIs(lexer.TokenRBrace) && !p.currentTokenIs(lexer.TokenEOF) {
		// Skip whitespace and comments
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
	// Allow nested function declarations (e.g., inside macro bodies)
	if p.current.Type == lexer.TokenFunc || (p.current.Type == lexer.TokenIdentifier && p.current.Literal == "fn") {
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

// parseMatchStatement parses a match statement of the form:
// match (expr) { pattern [if guard] => body, ... }
func (p *Parser) parseMatchStatement() *MatchStatement {
	startPos := TokenToPosition(p.current)

	// Expect '(' then condition expression then ')'
	if !p.expectPeek(lexer.TokenLParen) {
		return nil
	}
	p.nextToken()
	scrutinee := p.parseExpression(LOWEST)
	if !p.expectPeek(lexer.TokenRParen) {
		return nil
	}

	// Expect '{'
	if !p.expectPeek(lexer.TokenLBrace) {
		return nil
	}

	arms := make([]*MatchArm, 0)
	// Parse arms until '}'
	for {
		// Advance to first token of arm or '}'
		p.nextToken()
		if p.currentTokenIs(lexer.TokenRBrace) || p.currentTokenIs(lexer.TokenEOF) {
			break
		}
		// Skip trivia
		if p.currentTokenIs(lexer.TokenWhitespace) || p.currentTokenIs(lexer.TokenNewline) || p.currentTokenIs(lexer.TokenComment) {
			continue
		}

		// Parse pattern as an expression (simple for now)
		pattern := p.parseExpression(LOWEST)

		// Optional guard: 'if' expr
		var guard Expression
		if p.peekTokenIs(lexer.TokenIf) {
			p.nextToken() // move to 'if'
			p.nextToken() // move to start of guard expr
			guard = p.parseExpression(LOWEST)
		}

		// Expect '=>'
		if !p.expectPeek(lexer.TokenFatArrow) {
			// Try to recover to next arrow or end of arm
			for !p.peekTokenIs(lexer.TokenFatArrow) && !p.peekTokenIs(lexer.TokenComma) && !p.peekTokenIs(lexer.TokenRBrace) && !p.peekTokenIs(lexer.TokenEOF) {
				p.nextToken()
			}
			if p.peekTokenIs(lexer.TokenFatArrow) {
				p.nextToken()
			} else {
				// give up this arm
				continue
			}
		}

		// Body: block statement or single statement/expression
		var body Statement
		if p.peekTokenIs(lexer.TokenLBrace) {
			p.nextToken()
			body = p.parseBlockStatement()
		} else {
			// Parse a single statement as arm body; allow expression statements
			p.nextToken()
			body = p.parseStatement()
			if body == nil && p.current.Type != lexer.TokenRBrace {
				// Fallback to expression statement
				expr := p.parseExpression(LOWEST)
				if expr != nil {
					body = &ExpressionStatement{Span: TokenToSpan(p.current), Expression: expr}
				}
			}
			// Optional trailing comma or semicolon; do not require
			if p.peekTokenIs(lexer.TokenSemicolon) || p.peekTokenIs(lexer.TokenComma) {
				p.nextToken()
			}
		}

		arms = append(arms, &MatchArm{Span: TokenToSpan(p.current), Pattern: pattern, Guard: guard, Body: body})

		// If next is '}', end; if comma, continue to next arm
		if p.peekTokenIs(lexer.TokenRBrace) {
			p.nextToken()
			break
		}
		// Consume trailing comma if present and continue
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
func (p *Parser) parseForStatement() *ForStatement {
	startPos := TokenToPosition(p.current)

	// Expect '('
	if !p.expectPeek(lexer.TokenLParen) {
		return nil
	}

	// Parse optional init statement until ';'
	var init Statement
	// Move to the first token of init/semicolon
	p.nextToken()
	if !p.currentTokenIs(lexer.TokenSemicolon) {
		init = p.parseSimpleStatementUntilSemicolon()
	}
	// Current should be at semicolon; if not, try to recover
	if !p.currentTokenIs(lexer.TokenSemicolon) {
		// attempt to sync by skipping to next ';' or ')'
		for !p.currentTokenIs(lexer.TokenSemicolon) && !p.currentTokenIs(lexer.TokenRParen) && !p.currentTokenIs(lexer.TokenEOF) {
			p.nextToken()
		}
	}

	// Parse optional condition: advance to next token and read until ';'
	var cond Expression
	if p.currentTokenIs(lexer.TokenSemicolon) {
		// move to condition start or next ';'
		if !p.expectPeekRaw(lexer.TokenRParen) { // peek may be ')' or condition start
			p.nextToken()
			if !p.currentTokenIs(lexer.TokenSemicolon) && !p.currentTokenIs(lexer.TokenRParen) {
				cond = p.parseExpression(LOWEST)
			}
		}
	}
	// Ensure we are at the second semicolon (or ')')
	if !p.currentTokenIs(lexer.TokenSemicolon) {
		// Try to find semicolon
		for !p.currentTokenIs(lexer.TokenSemicolon) && !p.currentTokenIs(lexer.TokenRParen) && !p.currentTokenIs(lexer.TokenEOF) {
			p.nextToken()
		}
	}

	// Parse optional update statement: after second ';' until ')'
	var update Statement
	if p.currentTokenIs(lexer.TokenSemicolon) {
		// move to update start or ')'
		if !p.expectPeekRaw(lexer.TokenRParen) { // peek may be ')' or start of update
			p.nextToken()
			if !p.currentTokenIs(lexer.TokenRParen) {
				update = p.parseSimpleStatementUntilRParen()
			}
		}
	}

	// We should be at ')'
	if !p.currentTokenIs(lexer.TokenRParen) {
		// try to recover: advance to ')'
		for !p.currentTokenIs(lexer.TokenRParen) && !p.currentTokenIs(lexer.TokenEOF) {
			p.nextToken()
		}
	}

	// Expect body block
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

// parseBreakStatement parses a break statement with optional label
func (p *Parser) parseBreakStatement() *BreakStatement {
	startPos := TokenToPosition(p.current)
	var label *Identifier
	// Optional identifier label before ';' or '}'
	if p.peekTokenIs(lexer.TokenIdentifier) {
		p.nextToken()
		label = NewIdentifier(TokenToSpan(p.current), p.current.Literal)
	}
	// Optional semicolon
	if p.peekTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	}
	endPos := TokenToPosition(p.current)
	return &BreakStatement{Span: SpanBetween(startPos, endPos), Label: label}
}

// parseContinueStatement parses a continue statement with optional label
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

// parseSimpleStatementUntilSemicolon parses a minimal statement used in for-init until reaching ';'
func (p *Parser) parseSimpleStatementUntilSemicolon() Statement {
	// Variable declaration or expression statement
	switch p.current.Type {
	case lexer.TokenLet, lexer.TokenVar, lexer.TokenConst:
		stmt := p.parseVariableDeclaration()
		// Ensure current is at semicolon if present
		if !p.currentTokenIs(lexer.TokenSemicolon) && p.peekTokenIs(lexer.TokenSemicolon) {
			p.nextToken()
		}
		return stmt
	default:
		expr := p.parseExpression(LOWEST)
		// consume up to semicolon
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

// parseSimpleStatementUntilRParen parses a minimal statement used in for-update until ')'
func (p *Parser) parseSimpleStatementUntilRParen() Statement {
	switch p.current.Type {
	case lexer.TokenLet, lexer.TokenVar, lexer.TokenConst:
		stmt := p.parseVariableDeclaration()
		return stmt
	default:
		expr := p.parseExpression(LOWEST)
		// advance until ')'
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

// parseDeferStatement parses: defer { ... } | defer <statement-or-expr> ;
func (p *Parser) parseDeferStatement() *DeferStatement {
	startPos := TokenToPosition(p.current)
	var body Statement
	if p.peekTokenIs(lexer.TokenLBrace) {
		p.nextToken()
		body = p.parseBlockStatement()
	} else {
		// Defer a single statement or expression
		p.nextToken()
		body = p.parseStatement()
		if body == nil {
			// fallback: treat as expression
			expr := p.parseExpression(LOWEST)
			if expr != nil {
				body = &ExpressionStatement{Span: TokenToSpan(p.current), Expression: expr}
			}
		}
		// Optional semicolon
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
	// Build a boolean literal 'true'
	cond := NewLiteral(SpanBetween(startPos, startPos), true, LiteralBool)
	return &WhileStatement{Span: span, Condition: cond, Body: body}
}

// parseExpressionStatement parses an expression statement
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
		// Treat newline as a valid statement terminator; keep suggestion for optional semicolon style
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
	for !p.peekTokenIs(lexer.TokenSemicolon) && !p.peekTokenIs(lexer.TokenNewline) && p.shouldContinueParsing(precedence) {
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

	// Skip trivia between '(' and first argument or ')'
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
		// Skip trivia before checking for comma or closing paren
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}
		if !p.peekTokenIs(lexer.TokenComma) {
			break
		}
		p.nextToken()
		// Skip trivia before next argument
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

		// Detect variadic marker '...'
		dotCount := 0
		for p.peek.Type == lexer.TokenDot && dotCount < 3 {
			p.nextToken()
			dotCount++
		}
		if dotCount == 3 && len(params) > 0 {
			params[len(params)-1].IsVariadic = true
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
		// Skip trivia between templates
		if p.current.Type == lexer.TokenWhitespace || p.current.Type == lexer.TokenComment || p.current.Type == lexer.TokenNewline || p.current.Type == lexer.TokenSemicolon {
			p.nextToken()
			continue
		}

		// Guard again in case we advanced to closing brace via trivia
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

	// Ensure current token is the arrow '->'; accept if already at arrow,
	// otherwise require the next token to be arrow.
	if p.current.Type != lexer.TokenArrow {
		if !p.expectPeek(lexer.TokenArrow) {
			return nil
		}
	}

	// Move to the start of the template body (token after '->')
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

// parseMacroPatternElement parses a single pattern element
func (p *Parser) parseMacroPatternElement() *MacroPatternElement {
	switch p.current.Type {
	case lexer.TokenIdentifier:
		// '_' as wildcard; other identifiers as parameter
		if p.current.Literal == "_" {
			return &MacroPatternElement{Span: TokenToSpan(p.current), Kind: MacroPatternWildcard}
		}
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

	// Skip trivia between '(' and first argument or ')'
	for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
		p.nextToken()
	}
	if p.peek.Type == lexer.TokenRParen {
		return args
	}

	p.nextToken()
	for {
		if p.current.Type == lexer.TokenLBrace {
			// block literal as macro argument
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

		// Skip trivia before comma or closing paren
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}
		if p.peek.Type != lexer.TokenComma {
			break
		}
		p.nextToken() // consume comma
		// Skip trivia before next argument
		for p.peek.Type == lexer.TokenWhitespace || p.peek.Type == lexer.TokenComment || p.peek.Type == lexer.TokenNewline {
			p.nextToken()
		}
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
