// Package lexer implements robust error recovery mechanisms for the Orizon lexer.
// Phase 1.1.3: エラー回復機能実装
package lexer

import (
	"fmt"
	"regexp"
	"sort"
)

// ErrorRecovery provides sophisticated error handling and recovery capabilities.
// for lexical analysis, ensuring that a single syntax error doesn't prevent.
// the analysis of the entire source file.
type ErrorRecovery struct {
	syncPoints       map[TokenType]bool
	suggestionEngine *SuggestionEngine
	errorHistory     map[string]*ErrorFrequency
	errorPatterns    []ErrorPattern
	config           ErrorConfig
}

// ErrorPattern defines patterns for common lexical errors and their recovery strategies.
type ErrorPattern struct {
	Pattern         *regexp.Regexp
	SuggestionFunc  func(context string) []string
	Name            string
	Description     string
	MessageTemplate string
	SyncTokens      []TokenType
	RecoveryType    RecoveryType
	SkipCharacters  int
	Severity        ErrorSeverity
	Category        ErrorCategory
	Frequency       int
}

// RecoveryType defines different error recovery strategies.
type RecoveryType int

const (
	// Skip to next synchronization token (panic mode).
	RecoveryPanicMode RecoveryType = iota

	// Insert missing character(s).
	RecoveryInsertChar

	// Delete erroneous character(s).
	RecoveryDeleteChar

	// Replace character(s) with suggestion.
	RecoveryReplaceChar

	// Skip current invalid sequence.
	RecoverySkipSequence

	// Context-aware recovery using surrounding tokens.
	RecoveryContextual
)

// ErrorSeverity classifies the severity of lexical errors.
type ErrorSeverity int

const (
	SeverityInfo     ErrorSeverity = iota // Informational, doesn't prevent compilation
	SeverityWarning                       // Warning, compilation continues
	SeverityError                         // Error, compilation fails
	SeverityCritical                      // Critical error, immediate abort
)

// ErrorCategory categorizes types of lexical errors for better organization.
type ErrorCategory int

const (
	CategoryUnicodeError       ErrorCategory = iota // Unicode encoding/decoding issues
	CategoryUnterminatedString                      // Unclosed string literals
	CategoryInvalidCharacter                        // Invalid characters in identifiers/numbers
	CategoryMalformedNumber                         // Invalid number formats
	CategoryInvalidEscape                           // Invalid escape sequences
	CategoryCommentError                            // Unclosed comments
	CategoryEncodingError                           // Character encoding issues
)

// ErrorFrequency tracks frequency and context of recurring errors.
type ErrorFrequency struct {
	Context     string
	Suggestions []string
	Count       int
	LastSeen    int64
}

// SuggestionEngine generates intelligent suggestions for error correction.
type SuggestionEngine struct {
	// Dictionary of common identifiers and keywords for typo correction.
	vocabulary map[string]int

	// Pattern-based suggestions for common mistakes.
	commonMistakes map[string]string

	// Context-aware suggestion algorithms.
	contextRules []ContextRule
}

// ContextRule defines context-aware suggestion rules.
type ContextRule struct {
	Pattern     *regexp.Regexp
	Condition   func(context LexerContext) bool
	Suggestions func(context LexerContext) []string
	Name        string
}

// LexerContext provides context information for error recovery and suggestions.
type LexerContext struct {
	PrevChars    string
	NextChars    string
	Filename     string
	LineContent  string
	ErrorMessage string
	PrevTokens   []Token
	CurrentToken Token
	Position     Position
	ErrorType    ErrorCategory
	CurrentChar  rune
}

// ErrorConfig configures error reporting and recovery behavior.
type ErrorConfig struct {
	MaxErrors          int
	MaxSuggestions     int
	SuggestionMinScore float64
	ContinueOnCritical bool
	EnableSuggestions  bool
	VerboseMessages    bool
	ShowContext        bool
	ShowSuggestions    bool
	ColorizeOutput     bool
	AggressiveRecovery bool
	PreservePrevious   bool
}

// LexicalError represents a detailed lexical error with recovery information.
type LexicalError struct {
	Code              string
	Source            string
	Message           string
	LineContent       string
	Context           LexerContext
	SyncTokensUsed    []TokenType
	Suggestions       []ErrorSuggestion
	RelatedErrors     []*LexicalError
	Span              Span
	Position          Position
	Severity          ErrorSeverity
	Type              ErrorCategory
	RecoveryType      RecoveryType
	CharactersSkipped int
	Timestamp         int64
}

// ErrorSuggestion represents a potential fix for a lexical error.
type ErrorSuggestion struct {
	Description string
	Replacement string
	Category    string
	Example     string
	StartPos    Position
	EndPos      Position
	Confidence  float64
}

// NewErrorRecovery creates a new error recovery system with optimized defaults.
func NewErrorRecovery() *ErrorRecovery {
	recovery := &ErrorRecovery{
		syncPoints:    make(map[TokenType]bool),
		errorPatterns: make([]ErrorPattern, 0),
		errorHistory:  make(map[string]*ErrorFrequency),
		config: ErrorConfig{
			MaxErrors:          50,
			ContinueOnCritical: true,
			EnableSuggestions:  true,
			MaxSuggestions:     5,
			SuggestionMinScore: 0.3,
			VerboseMessages:    false,
			ShowContext:        true,
			ShowSuggestions:    true,
			AggressiveRecovery: false,
			PreservePrevious:   true,
		},
	}

	// Initialize default synchronization points.
	recovery.initDefaultSyncPoints()

	// Initialize error patterns.
	recovery.initErrorPatterns()

	// Initialize suggestion engine.
	recovery.suggestionEngine = NewSuggestionEngine()

	return recovery
}

// initDefaultSyncPoints sets up the default token types used for error synchronization.
func (er *ErrorRecovery) initDefaultSyncPoints() {
	// Statement terminators and delimiters.
	er.syncPoints[TokenSemicolon] = true
	er.syncPoints[TokenNewline] = true

	// Block delimiters.
	er.syncPoints[TokenLBrace] = true
	er.syncPoints[TokenRBrace] = true

	// Expression delimiters.
	er.syncPoints[TokenLParen] = true
	er.syncPoints[TokenRParen] = true
	er.syncPoints[TokenLBracket] = true
	er.syncPoints[TokenRBracket] = true

	// Keywords that often start new constructs.
	er.syncPoints[TokenFunc] = true
	er.syncPoints[TokenLet] = true
	er.syncPoints[TokenVar] = true
	er.syncPoints[TokenConst] = true
	er.syncPoints[TokenStruct] = true
	er.syncPoints[TokenEnum] = true
	er.syncPoints[TokenIf] = true
	er.syncPoints[TokenFor] = true
	er.syncPoints[TokenWhile] = true
	er.syncPoints[TokenReturn] = true

	// Import and module keywords.
	er.syncPoints[TokenImport] = true
	er.syncPoints[TokenExport] = true
	er.syncPoints[TokenModule] = true
}

// initErrorPatterns initializes common error patterns and their recovery strategies.
func (er *ErrorRecovery) initErrorPatterns() {
	er.errorPatterns = []ErrorPattern{
		{
			Name:            "UnterminatedString",
			Description:     "String literal not properly terminated",
			Pattern:         regexp.MustCompile(`"[^"]*$`),
			RecoveryType:    RecoveryPanicMode,
			SyncTokens:      []TokenType{TokenNewline, TokenSemicolon},
			MessageTemplate: "Unterminated string literal. Expected closing quote (\").",
			SuggestionFunc: func(context string) []string {
				return []string{
					"Add closing quote (\") at the end of the string",
					"Check for escaped quotes inside the string",
					"Consider using multi-line string syntax if needed",
				}
			},
			Severity: SeverityError,
			Category: CategoryUnterminatedString,
		},
		{
			Name:            "InvalidCharInIdentifier",
			Description:     "Invalid character in identifier",
			Pattern:         regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*[^a-zA-Z0-9_]+[a-zA-Z0-9_]*`),
			RecoveryType:    RecoveryDeleteChar,
			MessageTemplate: "Invalid character '%c' in identifier. Identifiers can only contain letters, digits, and underscores.",
			SuggestionFunc: func(context string) []string {
				return []string{
					"Remove invalid characters from identifier",
					"Use underscore (_) instead of hyphens or spaces",
					"Check for accidentally inserted symbols",
				}
			},
			Severity: SeverityError,
			Category: CategoryInvalidCharacter,
		},
		{
			Name:            "MalformedNumber",
			Description:     "Invalid number format",
			Pattern:         regexp.MustCompile(`\d+[a-zA-Z]+|\d*\.\d*\.\d*`),
			RecoveryType:    RecoverySkipSequence,
			MessageTemplate: "Malformed number literal. Numbers cannot contain letters or multiple decimal points.",
			SuggestionFunc: func(context string) []string {
				return []string{
					"Remove letters from number literal",
					"Use only one decimal point in floating-point numbers",
					"Consider using scientific notation (e.g., 1.23e4)",
				}
			},
			Severity: SeverityError,
			Category: CategoryMalformedNumber,
		},
		{
			Name:            "InvalidEscapeSequence",
			Description:     "Invalid escape sequence in string",
			Pattern:         regexp.MustCompile(`\\[^nrtbf\\'"0xuU]`),
			RecoveryType:    RecoveryReplaceChar,
			MessageTemplate: "Invalid escape sequence '\\%c'. Use valid escape sequences like \\n, \\t, \\r, \\\\, etc.",
			SuggestionFunc: func(context string) []string {
				return []string{
					"Use \\n for newline, \\t for tab, \\r for carriage return",
					"Use \\\\ for literal backslash, \\\" for quote",
					"Use \\u followed by 4 hex digits for Unicode characters",
				}
			},
			Severity: SeverityError,
			Category: CategoryInvalidEscape,
		},
		{
			Name:            "UnterminatedComment",
			Description:     "Block comment not properly closed",
			Pattern:         regexp.MustCompile(`/\*[^*/]*$`),
			RecoveryType:    RecoveryPanicMode,
			SyncTokens:      []TokenType{TokenNewline},
			MessageTemplate: "Unterminated block comment. Expected closing */ sequence.",
			SuggestionFunc: func(context string) []string {
				return []string{
					"Add */ to close the block comment",
					"Check for nested comments which may not be supported",
					"Consider using line comments (//) instead",
				}
			},
			Severity: SeverityError,
			Category: CategoryCommentError,
		},
	}

	// Sort patterns by frequency for optimization.
	sort.Slice(er.errorPatterns, func(i, j int) bool {
		return er.errorPatterns[i].Frequency > er.errorPatterns[j].Frequency
	})
}

// RecoverFromError attempts to recover from a lexical error and continue analysis.
func (er *ErrorRecovery) RecoverFromError(lexer *Lexer, err *LexicalError) (*Token, error) {
	// Update error history for learning.
	er.updateErrorHistory(err)

	// Find appropriate recovery strategy.
	pattern := er.findMatchingPattern(err.Context)
	if pattern == nil {
		// Use default panic mode recovery.
		return er.panicModeRecovery(lexer, err)
	}

	// Apply pattern-specific recovery.
	switch pattern.RecoveryType {
	case RecoveryPanicMode:
		return er.panicModeRecovery(lexer, err)

	case RecoveryInsertChar:
		return er.insertCharRecovery(lexer, err, pattern)

	case RecoveryDeleteChar:
		return er.deleteCharRecovery(lexer, err, pattern)

	case RecoveryReplaceChar:
		return er.replaceCharRecovery(lexer, err, pattern)

	case RecoverySkipSequence:
		return er.skipSequenceRecovery(lexer, err, pattern)

	case RecoveryContextual:
		return er.contextualRecovery(lexer, err, pattern)

	default:
		return er.panicModeRecovery(lexer, err)
	}
}

// panicModeRecovery implements classic panic mode recovery by skipping to sync points.
func (er *ErrorRecovery) panicModeRecovery(lexer *Lexer, err *LexicalError) (*Token, error) {
	skippedChars := 0

	// Skip characters until we find a synchronization point.
	for lexer.position < len(lexer.input) {
		// Try to tokenize from current position.
		lexer.readPosition = lexer.position + 1
		if lexer.readPosition < len(lexer.input) {
			lexer.ch = lexer.input[lexer.readPosition]
		} else {
			lexer.ch = 0
		}

		// Attempt to get next token.
		token := lexer.NextToken()

		// Check if this token is a sync point.
		if er.syncPoints[token.Type] {
			// Update recovery information.
			err.CharactersSkipped = skippedChars
			err.RecoveryType = RecoveryPanicMode

			return &token, nil
		}

		// Move to next character.
		lexer.position++
		skippedChars++

		// Safety limit to prevent infinite loops.
		if skippedChars > 1000 {
			err.CharactersSkipped = skippedChars
			err.RecoveryType = RecoveryPanicMode

			// Return a safe error token instead of nil to prevent cascading failures
			return &Token{
				Type:    TokenError,
				Literal: fmt.Sprintf("recovery_limit_exceeded_%d_chars", skippedChars),
				Line:    1,
				Column:  lexer.position,
			}, fmt.Errorf("panic mode recovery exceeded safety limit of 1000 characters")
		}
	}

	// Reached end of input.
	return &Token{Type: TokenEOF}, nil
}

// insertCharRecovery attempts recovery by inserting missing characters.
func (er *ErrorRecovery) insertCharRecovery(lexer *Lexer, err *LexicalError, pattern *ErrorPattern) (*Token, error) {
	// This is a placeholder for insert character recovery.
	// Implementation would depend on specific error patterns.
	return er.panicModeRecovery(lexer, err)
}

// deleteCharRecovery attempts recovery by deleting erroneous characters.
func (er *ErrorRecovery) deleteCharRecovery(lexer *Lexer, err *LexicalError, pattern *ErrorPattern) (*Token, error) {
	// Skip the problematic character and try again.
	if lexer.position < len(lexer.input) {
		lexer.readChar()

		err.CharactersSkipped = 1
		err.RecoveryType = RecoveryDeleteChar

		// Try to get next token from new position.
		token := lexer.NextToken()

		return &token, nil
	}

	return er.panicModeRecovery(lexer, err)
}

// replaceCharRecovery attempts recovery by replacing characters with suggestions.
func (er *ErrorRecovery) replaceCharRecovery(lexer *Lexer, err *LexicalError, pattern *ErrorPattern) (*Token, error) {
	// This is a placeholder for replace character recovery.
	// Implementation would involve trying suggested replacements.
	return er.panicModeRecovery(lexer, err)
}

// skipSequenceRecovery skips an entire invalid sequence.
func (er *ErrorRecovery) skipSequenceRecovery(lexer *Lexer, err *LexicalError, pattern *ErrorPattern) (*Token, error) {
	skippedChars := 0

	// Skip characters while they match the invalid pattern.
	for lexer.position < len(lexer.input) {
		remaining := lexer.input[lexer.position:]
		if !pattern.Pattern.MatchString(remaining) {
			break
		}

		lexer.readChar()

		skippedChars++

		// Safety limit.
		if skippedChars > 100 {
			break
		}
	}

	err.CharactersSkipped = skippedChars
	err.RecoveryType = RecoverySkipSequence

	// Try to get next token.
	if lexer.position < len(lexer.input) {
		token := lexer.NextToken()

		return &token, nil
	}

	return &Token{Type: TokenEOF}, nil
}

// contextualRecovery uses context information for intelligent recovery.
func (er *ErrorRecovery) contextualRecovery(lexer *Lexer, err *LexicalError, pattern *ErrorPattern) (*Token, error) {
	// This is a placeholder for context-aware recovery.
	// Implementation would analyze surrounding code for intelligent decisions.
	return er.panicModeRecovery(lexer, err)
}

// findMatchingPattern finds the best matching error pattern for the given context.
func (er *ErrorRecovery) findMatchingPattern(context LexerContext) *ErrorPattern {
	contextStr := context.PrevChars + string(context.CurrentChar) + context.NextChars

	for i := range er.errorPatterns {
		pattern := &er.errorPatterns[i]
		if pattern.Pattern.MatchString(contextStr) {
			pattern.Frequency++ // Update frequency for learning

			return pattern
		}
	}

	return nil
}

// updateErrorHistory tracks error patterns for learning and improvement.
func (er *ErrorRecovery) updateErrorHistory(err *LexicalError) {
	key := fmt.Sprintf("%d:%d:%d", int(err.Type), err.Position.Line, err.Position.Column)

	if freq, exists := er.errorHistory[key]; exists {
		freq.Count++
		freq.LastSeen = err.Timestamp
	} else {
		er.errorHistory[key] = &ErrorFrequency{
			Count:    1,
			LastSeen: err.Timestamp,
			Context:  err.LineContent,
		}
	}
}

// GenerateError creates a detailed error object with suggestions and recovery information.
func (er *ErrorRecovery) GenerateError(lexer *Lexer, errorType ErrorCategory, message string) *LexicalError {
	context := er.buildContext(lexer)

	err := &LexicalError{
		Position:    lexer.getCurrentPosition(),
		Span:        Span{Start: lexer.getCurrentPosition(), End: lexer.getCurrentPosition()},
		Message:     message,
		Type:        errorType,
		Severity:    er.determineSeverity(errorType),
		Code:        er.generateErrorCode(errorType),
		Context:     context,
		LineContent: er.getLineContent(lexer),
		Timestamp:   er.getCurrentTimestamp(),
		Source:      "lexer",
	}

	// Generate suggestions if enabled.
	if er.config.EnableSuggestions {
		err.Suggestions = er.generateSuggestions(context, errorType)
	}

	return err
}

// buildContext creates detailed context information for error reporting.
func (er *ErrorRecovery) buildContext(lexer *Lexer) LexerContext {
	return LexerContext{
		Position:    lexer.getCurrentPosition(),
		CurrentChar: rune(lexer.ch),
		PrevChars:   er.getPreviousChars(lexer, 10),
		NextChars:   er.getNextChars(lexer, 10),
		Filename:    lexer.filename,
		LineContent: er.getLineContent(lexer),
	}
}

// Helper methods for context building.
func (er *ErrorRecovery) getPreviousChars(lexer *Lexer, count int) string {
	start := lexer.position - count
	if start < 0 {
		start = 0
	}

	end := lexer.position
	if end > len(lexer.input) {
		end = len(lexer.input)
	}

	if start >= end {
		return ""
	}

	return string(lexer.input[start:end])
}

func (er *ErrorRecovery) getNextChars(lexer *Lexer, count int) string {
	start := lexer.position
	if start >= len(lexer.input) {
		return ""
	}

	end := lexer.position + count
	if end > len(lexer.input) {
		end = len(lexer.input)
	}

	if start >= end {
		return ""
	}

	return string(lexer.input[start:end])
}

func (er *ErrorRecovery) getLineContent(lexer *Lexer) string {
	// Find the start and end of the current line.
	start := lexer.position
	for start > 0 && start-1 < len(lexer.input) && lexer.input[start-1] != '\n' {
		start--
	}

	end := lexer.position
	for end < len(lexer.input) && lexer.input[end] != '\n' {
		end++
	}

	if start >= len(lexer.input) || end > len(lexer.input) || start >= end {
		return ""
	}

	return string(lexer.input[start:end])
}

func (er *ErrorRecovery) determineSeverity(errorType ErrorCategory) ErrorSeverity {
	switch errorType {
	case CategoryUnicodeError, CategoryEncodingError:
		return SeverityCritical
	case CategoryUnterminatedString, CategoryMalformedNumber:
		return SeverityError
	case CategoryInvalidCharacter, CategoryInvalidEscape:
		return SeverityError
	case CategoryCommentError:
		return SeverityWarning
	default:
		return SeverityError
	}
}

func (er *ErrorRecovery) generateErrorCode(errorType ErrorCategory) string {
	switch errorType {
	case CategoryUnicodeError:
		return "E001"
	case CategoryUnterminatedString:
		return "E002"
	case CategoryInvalidCharacter:
		return "E003"
	case CategoryMalformedNumber:
		return "E004"
	case CategoryInvalidEscape:
		return "E005"
	case CategoryCommentError:
		return "E006"
	case CategoryEncodingError:
		return "E007"
	default:
		return "E999"
	}
}

func (er *ErrorRecovery) getCurrentTimestamp() int64 {
	return 0 // Placeholder - would use time.Now().Unix() in real implementation
}

func (er *ErrorRecovery) generateSuggestions(context LexerContext, errorType ErrorCategory) []ErrorSuggestion {
	// This would integrate with the suggestion engine.
	// Placeholder implementation.
	return []ErrorSuggestion{}
}

// NewSuggestionEngine creates a new suggestion engine for error correction.
func NewSuggestionEngine() *SuggestionEngine {
	return &SuggestionEngine{
		vocabulary:     make(map[string]int),
		commonMistakes: make(map[string]string),
		contextRules:   make([]ContextRule, 0),
	}
}
