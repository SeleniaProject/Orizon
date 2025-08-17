// Package parser implements advanced error recovery and suggestion system
// Phase 1.2.4: エラー回復とサジェスト機能実装
package parser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/orizon-lang/orizon/internal/lexer"
)

// ErrorRecoveryMode defines different recovery strategies
type ErrorRecoveryMode int

const (
	// PanicMode performs rapid token skipping to synchronization points
	PanicMode ErrorRecoveryMode = iota
	// PhraseLevel attempts local error correction within statements
	PhraseLevel
	// GlobalCorrection performs comprehensive error analysis and correction
	GlobalCorrection
)

// SuggestionEngine provides intelligent code completion and error correction
type SuggestionEngine struct {
	// Recovery strategy configuration
	mode           ErrorRecoveryMode
	maxSuggestions int
	confidence     float64

	// Context tracking for intelligent suggestions
	currentScope   *ScopeContext
	recentTokens   []lexer.Token
	expectedTokens []lexer.TokenType

	// Pattern matching for common errors
	errorPatterns []ErrorPattern
	fixTemplates  []FixTemplate

	// Statistical data for suggestion ranking
	tokenFrequency map[lexer.TokenType]int
	pairFrequency  map[TokenPair]int
}

// ScopeContext tracks the current parsing context for contextual suggestions
type ScopeContext struct {
	Level        int                  // Current nesting level
	Type         ScopeType            // Current scope type (function, struct, etc.)
	Identifiers  map[string]IdentInfo // Available identifiers in scope
	ExpectedNext []lexer.TokenType    // Tokens that make sense in this context
	Parent       *ScopeContext        // Parent scope for identifier resolution
}

type ScopeType int

const (
	GlobalScope ScopeType = iota
	FunctionScope
	StructScope
	BlockScope
	ExpressionScope
	MacroScope
)

// IdentInfo stores information about available identifiers
type IdentInfo struct {
	Name       string
	Type       string
	Kind       IdentifierKind
	Confidence float64
	LastUsed   Position
}

type IdentifierKind int

const (
	VariableIdent IdentifierKind = iota
	FunctionIdent
	TypeIdent
	ConstantIdent
	MacroIdent
)

// TokenPair represents consecutive token combinations for pattern analysis
type TokenPair struct {
	First  lexer.TokenType
	Second lexer.TokenType
}

// ErrorPattern defines common syntax error patterns and their fixes
type ErrorPattern struct {
	Name        string
	Description string
	Pattern     []lexer.TokenType // Token sequence that indicates this error
	Context     ScopeType         // Scope where this error commonly occurs
	Confidence  float64           // How confident we are this is the intended pattern
	Fixes       []FixTemplate     // Possible fixes for this pattern
}

// FixTemplate defines an automatic code correction template
type FixTemplate struct {
	Name        string
	Description string
	Replacement []lexer.TokenType // Tokens to insert/replace
	InsertPos   int               // Position to insert relative to error (-1 = replace)
	Confidence  float64           // Confidence in this fix
	Example     string            // Example of the fix applied
}

// Suggestion represents a single code suggestion or error fix
type Suggestion struct {
	Type        SuggestionType
	Message     string
	Position    Position
	Replacement string
	Confidence  float64
	Category    SuggestionCategory
	Fix         *FixTemplate
}

type SuggestionType int

const (
	ErrorFix SuggestionType = iota
	Completion
	Refactoring
	StyleImprovement
)

type SuggestionCategory int

const (
	SyntaxError SuggestionCategory = iota
	TypeError
	NameError
	ScopeError
	StyleError
	PerformanceHint
)

// NewSuggestionEngine creates a new error recovery and suggestion engine
func NewSuggestionEngine(mode ErrorRecoveryMode) *SuggestionEngine {
	engine := &SuggestionEngine{
		mode:           mode,
		maxSuggestions: 10,
		confidence:     0.5, // Lower threshold to include more suggestions
		recentTokens:   make([]lexer.Token, 0, 10),
		expectedTokens: make([]lexer.TokenType, 0),
		tokenFrequency: make(map[lexer.TokenType]int),
		pairFrequency:  make(map[TokenPair]int),
	}

	// Initialize with common error patterns
	engine.initializeErrorPatterns()
	engine.initializeFixTemplates()

	return engine
}

// initializeErrorPatterns sets up common syntax error recognition patterns
func (se *SuggestionEngine) initializeErrorPatterns() {
	se.errorPatterns = []ErrorPattern{
		{
			Name:        "MissingSemicolon",
			Description: "Missing semicolon at end of statement",
			Pattern:     []lexer.TokenType{lexer.TokenIdentifier, lexer.TokenNewline},
			Context:     BlockScope,
			Confidence:  0.9,
			Fixes: []FixTemplate{{
				Name:        "InsertSemicolon",
				Description: "Insert missing semicolon",
				Replacement: []lexer.TokenType{lexer.TokenSemicolon},
				InsertPos:   0,
				Confidence:  0.95,
				Example:     "let x = 5; // <- semicolon inserted",
			}},
		},
		{
			Name:        "MissingCloseBrace",
			Description: "Missing closing brace for block",
			Pattern:     []lexer.TokenType{lexer.TokenLBrace, lexer.TokenEOF},
			Context:     BlockScope,
			Confidence:  0.95,
			Fixes: []FixTemplate{{
				Name:        "InsertCloseBrace",
				Description: "Insert missing closing brace",
				Replacement: []lexer.TokenType{lexer.TokenRBrace},
				InsertPos:   0,
				Confidence:  0.9,
				Example:     "} // <- closing brace inserted",
			}},
		},
		{
			Name:        "MissingCommaInList",
			Description: "Missing comma between list elements",
			Pattern:     []lexer.TokenType{lexer.TokenIdentifier, lexer.TokenIdentifier},
			Context:     ExpressionScope,
			Confidence:  0.8,
			Fixes: []FixTemplate{{
				Name:        "InsertComma",
				Description: "Insert comma between elements",
				Replacement: []lexer.TokenType{lexer.TokenComma},
				InsertPos:   0,
				Confidence:  0.85,
				Example:     "func(a, b) // <- comma inserted",
			}},
		},
		{
			Name:        "TypoInKeyword",
			Description: "Possible typo in language keyword",
			Pattern:     []lexer.TokenType{lexer.TokenIdentifier},
			Context:     GlobalScope,
			Confidence:  0.7,
			Fixes: []FixTemplate{
				{
					Name:        "CorrectFunction",
					Description: "Did you mean 'func'?",
					Replacement: []lexer.TokenType{lexer.TokenFunc},
					InsertPos:   -1,
					Confidence:  0.8,
					Example:     "func main() { // <- 'func' instead of 'function'",
				},
				{
					Name:        "CorrectLet",
					Description: "Did you mean 'let'?",
					Replacement: []lexer.TokenType{lexer.TokenLet},
					InsertPos:   -1,
					Confidence:  0.8,
					Example:     "let x = 5; // <- 'let' instead of 'var'",
				},
			},
		},
		{
			Name:        "MismatchedParentheses",
			Description: "Unbalanced parentheses in expression",
			Pattern:     []lexer.TokenType{lexer.TokenLParen, lexer.TokenSemicolon},
			Context:     ExpressionScope,
			Confidence:  0.9,
			Fixes: []FixTemplate{{
				Name:        "InsertCloseParen",
				Description: "Insert missing closing parenthesis",
				Replacement: []lexer.TokenType{lexer.TokenRParen},
				InsertPos:   0,
				Confidence:  0.9,
				Example:     "func(x) // <- closing parenthesis inserted",
			}},
		},
	}
}

// initializeFixTemplates sets up common code fix templates
func (se *SuggestionEngine) initializeFixTemplates() {
	se.fixTemplates = []FixTemplate{
		{
			Name:        "AddFunctionKeyword",
			Description: "Add 'func' keyword before function definition",
			Replacement: []lexer.TokenType{lexer.TokenFunc},
			InsertPos:   0,
			Confidence:  0.9,
			Example:     "func main() { ... }",
		},
		{
			Name:        "AddVariableKeyword",
			Description: "Add variable declaration keyword",
			Replacement: []lexer.TokenType{lexer.TokenLet},
			InsertPos:   0,
			Confidence:  0.85,
			Example:     "let x = 5;",
		},
		{
			Name:        "AddTypeAnnotation",
			Description: "Add explicit type annotation",
			Replacement: []lexer.TokenType{lexer.TokenColon, lexer.TokenIdentifier},
			InsertPos:   1,
			Confidence:  0.75,
			Example:     "let x: int = 5;",
		},
	}
}

// RecoverFromError attempts to recover from a parsing error using the configured strategy
func (se *SuggestionEngine) RecoverFromError(p *Parser, err *ParseError) []Suggestion {
	// Update context tracking
	se.updateContext(p)

	// Generate suggestions based on error type and context
	suggestions := se.generateSuggestions(p, err)

	// Perform error recovery based on mode
	switch se.mode {
	case PanicMode:
		se.performPanicRecovery(p, err)
	case PhraseLevel:
		se.performPhraseRecovery(p, err)
	case GlobalCorrection:
		se.performGlobalRecovery(p, err)
	}

	// Rank and filter suggestions (do this before filtering to get proper order)
	suggestions = se.rankSuggestions(suggestions)
	return se.filterSuggestions(suggestions)
}

// AddExpectedToken adds a token type to the expected tokens for suggestions
func (se *SuggestionEngine) AddExpectedToken(tokenType lexer.TokenType) {
	// Add to expected tokens if not already present
	found := false
	for _, expected := range se.expectedTokens {
		if expected == tokenType {
			found = true
			break
		}
	}
	if !found {
		se.expectedTokens = append(se.expectedTokens, tokenType)
	}
}

// updateContext tracks the current parsing context for intelligent suggestions
func (se *SuggestionEngine) updateContext(p *Parser) {
	// Track recent tokens for pattern matching
	if len(se.recentTokens) >= 10 {
		se.recentTokens = se.recentTokens[1:]
	}
	se.recentTokens = append(se.recentTokens, p.current)

	// Update token frequency statistics
	se.tokenFrequency[p.current.Type]++

	// Update token pair frequency
	if len(se.recentTokens) >= 2 {
		pair := TokenPair{
			First:  se.recentTokens[len(se.recentTokens)-2].Type,
			Second: se.recentTokens[len(se.recentTokens)-1].Type,
		}
		se.pairFrequency[pair]++
	}

	// Determine expected tokens based on current context
	se.updateExpectedTokens(p)
}

// updateExpectedTokens determines what tokens would be valid in current context
func (se *SuggestionEngine) updateExpectedTokens(p *Parser) {
	se.expectedTokens = se.expectedTokens[:0] // Clear slice

	// Based on recent tokens, predict what should come next
	if len(se.recentTokens) > 0 {
		current := se.recentTokens[len(se.recentTokens)-1]

		switch current.Type {
		case lexer.TokenLet, lexer.TokenVar, lexer.TokenConst:
			se.expectedTokens = append(se.expectedTokens, lexer.TokenIdentifier)
		case lexer.TokenFunc:
			se.expectedTokens = append(se.expectedTokens, lexer.TokenIdentifier)
		case lexer.TokenIdentifier:
			se.expectedTokens = append(se.expectedTokens,
				lexer.TokenAssign, lexer.TokenColon, lexer.TokenLParen, lexer.TokenSemicolon)
		case lexer.TokenLParen:
			se.expectedTokens = append(se.expectedTokens,
				lexer.TokenIdentifier, lexer.TokenRParen, lexer.TokenInteger, lexer.TokenString)
		case lexer.TokenLBrace:
			se.expectedTokens = append(se.expectedTokens,
				lexer.TokenIdentifier, lexer.TokenLet, lexer.TokenVar, lexer.TokenFunc, lexer.TokenRBrace)
		case lexer.TokenAssign:
			se.expectedTokens = append(se.expectedTokens,
				lexer.TokenIdentifier, lexer.TokenInteger, lexer.TokenFloat, lexer.TokenString, lexer.TokenBool)
		}
	}
}

// generateSuggestions creates intelligent suggestions based on error context
func (se *SuggestionEngine) generateSuggestions(p *Parser, err *ParseError) []Suggestion {
	var suggestions []Suggestion

	// Check for pattern matches with known error types
	for _, pattern := range se.errorPatterns {
		if se.matchesPattern(pattern, se.recentTokens) {
			for _, fix := range pattern.Fixes {
				suggestion := Suggestion{
					Type:        ErrorFix,
					Message:     fmt.Sprintf("%s: %s", pattern.Name, fix.Description),
					Position:    err.Position,
					Replacement: se.generateReplacement(fix),
					Confidence:  fix.Confidence * pattern.Confidence,
					Category:    SyntaxError,
					Fix:         &fix,
				}
				suggestions = append(suggestions, suggestion)
			}
		}
	}

	// Generate completion suggestions based on expected tokens
	for _, expected := range se.expectedTokens {
		if se.isCompletionCandidate(expected, p.current.Type) {
			suggestion := Suggestion{
				Type:        Completion,
				Message:     fmt.Sprintf("Expected %s", tokenTypeToString(expected)),
				Position:    err.Position,
				Replacement: tokenTypeToString(expected),
				Confidence:  0.8,
				Category:    SyntaxError,
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	// If no specific expected tokens, generate common completion suggestions
	if len(se.expectedTokens) == 0 {
		commonTokens := []lexer.TokenType{
			lexer.TokenIdentifier, lexer.TokenLet, lexer.TokenVar, lexer.TokenFunc,
			lexer.TokenIf, lexer.TokenWhile, lexer.TokenReturn, lexer.TokenRBrace,
		}
		for _, token := range commonTokens {
			if se.isCompletionCandidate(token, p.current.Type) {
				suggestion := Suggestion{
					Type:        Completion,
					Message:     fmt.Sprintf("Expected %s", tokenTypeToString(token)),
					Position:    err.Position,
					Replacement: tokenTypeToString(token),
					Confidence:  0.8,
					Category:    SyntaxError,
				}
				suggestions = append(suggestions, suggestion)
			}
		}
	}

	// Generate typo corrections for identifiers
	if p.current.Type == lexer.TokenIdentifier {
		suggestions = append(suggestions, se.generateTypoCorrections(p.current.Literal)...)
	}

	// Generate scope-based suggestions
	suggestions = append(suggestions, se.generateScopeBasedSuggestions(p)...)

	return suggestions
}

// matchesPattern checks if recent tokens match a known error pattern
func (se *SuggestionEngine) matchesPattern(pattern ErrorPattern, tokens []lexer.Token) bool {
	if len(tokens) < len(pattern.Pattern) {
		return false
	}

	// Check if the last N tokens match the pattern
	start := len(tokens) - len(pattern.Pattern)
	for i, expectedType := range pattern.Pattern {
		if tokens[start+i].Type != expectedType {
			return false
		}
	}

	return true
}

// generateReplacement creates the replacement text for a fix template
func (se *SuggestionEngine) generateReplacement(fix FixTemplate) string {
	var parts []string
	for _, tokenType := range fix.Replacement {
		parts = append(parts, tokenTypeToString(tokenType))
	}
	return strings.Join(parts, " ")
}

// isCompletionCandidate determines if a token type is a valid completion option
func (se *SuggestionEngine) isCompletionCandidate(expected, current lexer.TokenType) bool {
	// Don't suggest if we already have the expected token
	if expected == current {
		return false
	}

	// Don't suggest whitespace or comments
	if expected == lexer.TokenWhitespace || expected == lexer.TokenComment {
		return false
	}

	return true
}

// generateTypoCorrections suggests corrections for potentially misspelled identifiers
func (se *SuggestionEngine) generateTypoCorrections(input string) []Suggestion {
	var suggestions []Suggestion

	// Common keyword typos
	keywordCorrections := map[string]string{
		"function": "func",
		"var":      "let",
		"def":      "func",
		"int32":    "int",
		"int64":    "int",
		"string":   "str",
		"boolean":  "bool",
		"true":     "true",
		"false":    "false",
		"nil":      "null",
		"None":     "null",
		"return":   "return",
		"if":       "if",
		"else":     "else",
		"for":      "for",
		"while":    "while",
	}

	// Check for exact matches first
	if correction, exists := keywordCorrections[input]; exists {
		suggestion := Suggestion{
			Type:        ErrorFix,
			Message:     fmt.Sprintf("Did you mean '%s'?", correction),
			Position:    Position{}, // Will be set by caller
			Replacement: correction,
			Confidence:  0.9,
			Category:    SyntaxError,
		}
		suggestions = append(suggestions, suggestion)
	}

	// Check for fuzzy matches using edit distance
	for typo, correction := range keywordCorrections {
		distance := editDistance(input, typo)
		if distance <= 2 && distance > 0 { // Allow up to 2 character differences
			confidence := 1.0 - (float64(distance) / float64(len(input)))
			if confidence >= 0.6 {
				suggestion := Suggestion{
					Type:        ErrorFix,
					Message:     fmt.Sprintf("Did you mean '%s'? (similar to '%s')", correction, typo),
					Position:    Position{},
					Replacement: correction,
					Confidence:  confidence,
					Category:    SyntaxError,
				}
				suggestions = append(suggestions, suggestion)
			}
		}
	}

	return suggestions
}

// generateScopeBasedSuggestions creates suggestions based on current scope context
func (se *SuggestionEngine) generateScopeBasedSuggestions(p *Parser) []Suggestion {
	var suggestions []Suggestion

	// If we have a current scope context
	if se.currentScope != nil {
		// Suggest available identifiers in scope
		for name, info := range se.currentScope.Identifiers {
			if info.Confidence > 0.5 {
				suggestion := Suggestion{
					Type:        Completion,
					Message:     fmt.Sprintf("Available %s: %s", identifierKindToString(info.Kind), name),
					Position:    Position{},
					Replacement: name,
					Confidence:  info.Confidence,
					Category:    ScopeError,
				}
				suggestions = append(suggestions, suggestion)
			}
		}
	}

	return suggestions
}

// performPanicRecovery implements panic-mode error recovery
func (se *SuggestionEngine) performPanicRecovery(p *Parser, err *ParseError) {
	// Define synchronization tokens where parsing can safely resume
	syncTokens := []lexer.TokenType{
		lexer.TokenSemicolon,
		lexer.TokenNewline,
		lexer.TokenRBrace,
		lexer.TokenFunc,
		lexer.TokenLet,
		lexer.TokenVar,
		lexer.TokenConst,
		lexer.TokenIf,
		lexer.TokenFor,
		lexer.TokenWhile,
		lexer.TokenReturn,
	}

	// Skip tokens until we find a synchronization point
	for p.current.Type != lexer.TokenEOF {
		for _, syncToken := range syncTokens {
			if p.current.Type == syncToken {
				return // Found synchronization point
			}
		}
		p.nextToken()
	}
}

// performPhraseRecovery implements phrase-level error recovery
func (se *SuggestionEngine) performPhraseRecovery(p *Parser, err *ParseError) {
	// Try to complete the current phrase intelligently
	switch p.current.Type {
	case lexer.TokenIdentifier:
		// If we're at a top-level declaration head like 'func <ident>' or 'let <ident>',
		// avoid aggressive scanning that can skip over valid declarations.
		if len(se.recentTokens) >= 2 {
			prev := se.recentTokens[len(se.recentTokens)-2]
			if prev.Type == lexer.TokenFunc || prev.Type == lexer.TokenLet || prev.Type == lexer.TokenVar || prev.Type == lexer.TokenConst || prev.Type == lexer.TokenMacro || prev.Type == lexer.TokenStruct || prev.Type == lexer.TokenEnum {
				break // fall through to panic recovery below
			}
		}
		// If we have an identifier, try to complete a declaration or expression
		if len(se.expectedTokens) > 0 {
			// Skip to the most likely next valid token
			for _, expected := range se.expectedTokens {
				if se.advanceToToken(p, expected) {
					return
				}
			}
		}
	case lexer.TokenLParen:
		// If we have an unclosed parenthesis, try to balance it
		se.balanceDelimiters(p, lexer.TokenLParen, lexer.TokenRParen)
	case lexer.TokenLBrace:
		// If we have an unclosed brace, try to balance it
		se.balanceDelimiters(p, lexer.TokenLBrace, lexer.TokenRBrace)
	}

	// Fall back to panic recovery if phrase recovery fails
	se.performPanicRecovery(p, err)
}

// performGlobalRecovery implements comprehensive error analysis and recovery
func (se *SuggestionEngine) performGlobalRecovery(p *Parser, err *ParseError) {
	// Analyze the entire error context and make intelligent decisions

	// First, try phrase-level recovery
	se.performPhraseRecovery(p, err)

	// If that doesn't work, update our understanding and try again
	se.updateErrorPatterns(err)

	// Finally, fall back to panic recovery
	se.performPanicRecovery(p, err)
}

// advanceToToken safely advances parser to a specific token type
func (se *SuggestionEngine) advanceToToken(p *Parser, target lexer.TokenType) bool {
	maxAdvance := 10 // Limit how far we look ahead

	for i := 0; i < maxAdvance && p.current.Type != lexer.TokenEOF; i++ {
		if p.current.Type == target {
			return true
		}
		p.nextToken()
	}

	return false
}

// balanceDelimiters attempts to balance unmatched delimiters
func (se *SuggestionEngine) balanceDelimiters(p *Parser, open, close lexer.TokenType) {
	depth := 1 // We've seen one opening delimiter

	for p.current.Type != lexer.TokenEOF && depth > 0 {
		p.nextToken()
		if p.current.Type == open {
			depth++
		} else if p.current.Type == close {
			depth--
		}
	}
}

// updateErrorPatterns learns from new errors to improve future suggestions
func (se *SuggestionEngine) updateErrorPatterns(err *ParseError) {
	// This would implement machine learning to improve error recognition
	// For now, we just log the error pattern for future analysis
	// In a production system, this could update confidence scores
	// or add new patterns based on recurring errors
}

// rankSuggestions sorts suggestions by confidence and relevance
func (se *SuggestionEngine) rankSuggestions(suggestions []Suggestion) []Suggestion {
	sort.Slice(suggestions, func(i, j int) bool {
		// Primary sort by confidence (descending)
		if suggestions[i].Confidence != suggestions[j].Confidence {
			return suggestions[i].Confidence > suggestions[j].Confidence
		}

		// Secondary sort by suggestion type (fixes first, then completions)
		return suggestions[i].Type < suggestions[j].Type
	})

	return suggestions
}

// filterSuggestions removes low-quality and duplicate suggestions
func (se *SuggestionEngine) filterSuggestions(suggestions []Suggestion) []Suggestion {
	var filtered []Suggestion
	seen := make(map[string]bool)

	for _, suggestion := range suggestions {
		// Skip low-confidence suggestions
		if suggestion.Confidence < se.confidence {
			continue
		}

		// Skip duplicates
		key := suggestion.Message + suggestion.Replacement
		if seen[key] {
			continue
		}
		seen[key] = true

		filtered = append(filtered, suggestion)

		// Limit number of suggestions
		if len(filtered) >= se.maxSuggestions {
			break
		}
	}

	return filtered
}

// Helper functions

// editDistance calculates the Levenshtein distance between two strings
func editDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create matrix
	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}

// tokenTypeToString converts token type to readable string
func tokenTypeToString(tokenType lexer.TokenType) string {
	switch tokenType {
	case lexer.TokenFunc:
		return "func"
	case lexer.TokenLet:
		return "let"
	case lexer.TokenVar:
		return "var"
	case lexer.TokenConst:
		return "const"
	case lexer.TokenIdentifier:
		return "identifier"
	case lexer.TokenInteger:
		return "integer"
	case lexer.TokenFloat:
		return "float"
	case lexer.TokenString:
		return "string"
	case lexer.TokenBool:
		return "boolean"
	case lexer.TokenSemicolon:
		return ";"
	case lexer.TokenComma:
		return ","
	case lexer.TokenColon:
		return ":"
	case lexer.TokenAssign:
		return "="
	case lexer.TokenLParen:
		return "("
	case lexer.TokenRParen:
		return ")"
	case lexer.TokenLBrace:
		return "{"
	case lexer.TokenRBrace:
		return "}"
	case lexer.TokenLBracket:
		return "["
	case lexer.TokenRBracket:
		return "]"
	default:
		return fmt.Sprintf("<%s>", tokenType.String())
	}
}

// identifierKindToString converts identifier kind to readable string
func identifierKindToString(kind IdentifierKind) string {
	switch kind {
	case VariableIdent:
		return "variable"
	case FunctionIdent:
		return "function"
	case TypeIdent:
		return "type"
	case ConstantIdent:
		return "constant"
	case MacroIdent:
		return "macro"
	default:
		return "identifier"
	}
}
