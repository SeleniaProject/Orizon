// Package lexer error recovery integration
// Phase 1.1.3: エラー回復機能実装 - Lexer統合部分
package lexer

import (
	"fmt"
	"time"
)

// WithErrorRecovery adds error recovery capabilities to the lexer
func (l *Lexer) WithErrorRecovery() *Lexer {
	if l.errorRecovery == nil {
		l.errorRecovery = NewErrorRecovery()
	}
	return l
}

// RecoverableNextToken wraps NextToken with error recovery capabilities
func (l *Lexer) RecoverableNextToken() Token {
	if l.errorRecovery == nil {
		return l.NextToken()
	}

	errorCountBefore := len(l.GetErrors())

	// Attempt normal token parsing
	token := l.NextToken()

	// Check if we encountered an error token
	if token.Type == TokenError {
		// Use the error message from the token if no new errors were added
		errorMessage := token.Literal

		var lexError *LexicalError
		if len(l.GetErrors()) == errorCountBefore {
			// No new errors were added, create one based on the token
			if errorMessage == "unterminated string literal" {
				lexError = l.CreateUnterminatedStringError()
			} else {
				lexError = l.createLexicalError(errorMessage)
			}
			l.addError(lexError)
		} else {
			// Use the last error that was added
			lexError = l.GetErrors()[len(l.GetErrors())-1]
		}

		// Attempt error recovery
		// For now, just return the error token - recovery can be added later
		// if recoveredToken, err := l.errorRecovery.RecoverFromError(l, lexError); err == nil && recoveredToken != nil {
		//     return *recoveredToken
		// }

		// Return the error token
		return token
	}

	return token
} // createLexicalError creates a detailed error object from an error token
func (l *Lexer) createLexicalError(message string) *LexicalError {
	if l.errorRecovery == nil {
		// Return a basic error if no error recovery system
		return &LexicalError{
			Position: l.getCurrentPosition(),
			Message:  message,
			Type:     CategoryInvalidCharacter,
			Severity: SeverityError,
		}
	}

	// Use error recovery system to create detailed error
	return l.errorRecovery.GenerateError(l, l.categorizeError(message), message)
}

// categorizeError attempts to categorize the error based on the message
func (l *Lexer) categorizeError(message string) ErrorCategory {
	// Simple heuristic-based categorization
	switch {
	case contains(message, "unterminated", "string"):
		return CategoryUnterminatedString
	case contains(message, "invalid", "character"):
		return CategoryInvalidCharacter
	case contains(message, "malformed", "number"):
		return CategoryMalformedNumber
	case contains(message, "escape", "sequence"):
		return CategoryInvalidEscape
	case contains(message, "comment"):
		return CategoryCommentError
	case contains(message, "unicode", "encoding"):
		return CategoryUnicodeError
	default:
		return CategoryInvalidCharacter
	}
}

// contains checks if any of the keywords appear in the message (case-insensitive)
func contains(message string, keywords ...string) bool {
	messageLower := toLower(message)
	for _, keyword := range keywords {
		if containsSubstring(messageLower, toLower(keyword)) {
			return true
		}
	}
	return false
}

// Simple string operations to avoid imports
func toLower(s string) string {
	result := make([]byte, len(s))
	for i, r := range []byte(s) {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// addError adds an error to the lexer's error collection
func (l *Lexer) addError(err *LexicalError) {
	if l.errors == nil {
		l.errors = make([]any, 0)
	}

	// Set timestamp if not already set
	if err.Timestamp == 0 {
		err.Timestamp = time.Now().Unix()
	}

	l.errors = append(l.errors, err)
}

// GetErrors returns all accumulated lexical errors
func (l *Lexer) GetErrors() []*LexicalError {
	var result []*LexicalError
	for _, e := range l.errors {
		if err, ok := e.(*LexicalError); ok {
			result = append(result, err)
		}
	}
	return result
}

// HasErrors returns true if the lexer has encountered any errors
func (l *Lexer) HasErrors() bool {
	return len(l.errors) > 0
}

// ClearErrors clears all accumulated errors
func (l *Lexer) ClearErrors() {
	l.errors = l.errors[:0]
}

// GetErrorsByCategory returns errors filtered by category
func (l *Lexer) GetErrorsByCategory(category ErrorCategory) []*LexicalError {
	var filtered []*LexicalError
	for _, e := range l.errors {
		if err, ok := e.(*LexicalError); ok && err.Type == category {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

// GetErrorsBySeverity returns errors filtered by severity
func (l *Lexer) GetErrorsBySeverity(severity ErrorSeverity) []*LexicalError {
	var filtered []*LexicalError
	for _, e := range l.errors {
		if err, ok := e.(*LexicalError); ok && err.Severity == severity {
			filtered = append(filtered, err)
		}
	}
	return filtered
} // FormatError returns a formatted string representation of an error
func (l *Lexer) FormatError(err *LexicalError) string {
	return fmt.Sprintf("%s:%d:%d: %s: %s",
		err.Context.Filename,
		err.Position.Line,
		err.Position.Column,
		severityString(err.Severity),
		err.Message)
}

// FormatErrorDetailed returns a detailed formatted string with context
func (l *Lexer) FormatErrorDetailed(err *LexicalError) string {
	base := l.FormatError(err)

	if err.Context.LineContent != "" {
		base += fmt.Sprintf("\n  %s", err.Context.LineContent)

		// Add a caret indicator pointing to the error position
		padding := ""
		for i := 0; i < err.Position.Column-1; i++ {
			padding += " "
		}
		base += fmt.Sprintf("\n  %s^", padding)
	}

	// Add suggestions if available
	if len(err.Suggestions) > 0 {
		base += "\n  Suggestions:"
		for _, suggestion := range err.Suggestions {
			base += fmt.Sprintf("\n    - %s", suggestion.Description)
		}
	}

	return base
}

// severityString returns a string representation of error severity
func severityString(severity ErrorSeverity) string {
	switch severity {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Enhanced error creation methods for specific error types

// CreateUnterminatedStringError creates a specific error for unterminated strings
func (l *Lexer) CreateUnterminatedStringError() *LexicalError {
	if l.errorRecovery == nil {
		return l.createLexicalError("unterminated string literal")
	}

	err := l.errorRecovery.GenerateError(l, CategoryUnterminatedString,
		"Unterminated string literal. Expected closing quote (\").")

	// Add specific suggestions for string errors
	err.Suggestions = append(err.Suggestions, ErrorSuggestion{
		Description: "Add closing quote (\") at the end of the string",
		Confidence:  0.9,
		Category:    "syntax",
	}, ErrorSuggestion{
		Description: "Check for escaped quotes inside the string",
		Confidence:  0.7,
		Category:    "content",
	})

	return err
}

// CreateInvalidCharacterError creates a specific error for invalid characters
func (l *Lexer) CreateInvalidCharacterError(char rune) *LexicalError {
	message := fmt.Sprintf("Invalid character '%c' (U+%04X)", char, char)

	if l.errorRecovery == nil {
		return l.createLexicalError(message)
	}

	err := l.errorRecovery.GenerateError(l, CategoryInvalidCharacter, message)

	// Add character-specific suggestions
	err.Suggestions = append(err.Suggestions, ErrorSuggestion{
		Description: "Remove the invalid character",
		Confidence:  0.8,
		Category:    "syntax",
	})

	// Add specific suggestions for common character mistakes
	switch char {
	case '\u2018', '\u2019': // Smart single quotes
		err.Suggestions = append(err.Suggestions, ErrorSuggestion{
			Description: "Replace with straight single quote (')",
			Replacement: "'",
			Confidence:  0.9,
			Category:    "typography",
		})
	case '\u201C', '\u201D': // Smart double quotes
		err.Suggestions = append(err.Suggestions, ErrorSuggestion{
			Description: "Replace with straight double quote (\")",
			Replacement: "\"",
			Confidence:  0.9,
			Category:    "typography",
		})
	case '\u2014', '\u2013': // Em dash, en dash
		err.Suggestions = append(err.Suggestions, ErrorSuggestion{
			Description: "Replace with hyphen-minus (-)",
			Replacement: "-",
			Confidence:  0.8,
			Category:    "typography",
		})
	}

	return err
}

// CreateMalformedNumberError creates a specific error for malformed numbers
func (l *Lexer) CreateMalformedNumberError(literal string) *LexicalError {
	message := fmt.Sprintf("Malformed number literal: %s", literal)

	if l.errorRecovery == nil {
		return l.createLexicalError(message)
	}

	err := l.errorRecovery.GenerateError(l, CategoryMalformedNumber, message)

	// Add number-specific suggestions
	err.Suggestions = append(err.Suggestions, ErrorSuggestion{
		Description: "Remove letters from number literal",
		Confidence:  0.8,
		Category:    "syntax",
	}, ErrorSuggestion{
		Description: "Use only one decimal point in floating-point numbers",
		Confidence:  0.7,
		Category:    "syntax",
	})

	return err
}

// ErrorRecoveryConfig allows configuration of error recovery behavior
type ErrorRecoveryConfig struct {
	MaxErrors          int
	ContinueOnCritical bool
	EnableSuggestions  bool
	VerboseMessages    bool
}

// ConfigureErrorRecovery allows customization of error recovery behavior
func (l *Lexer) ConfigureErrorRecovery(config ErrorRecoveryConfig) {
	if l.errorRecovery == nil {
		l.errorRecovery = NewErrorRecovery()
	}

	l.errorRecovery.config.MaxErrors = config.MaxErrors
	l.errorRecovery.config.ContinueOnCritical = config.ContinueOnCritical
	l.errorRecovery.config.EnableSuggestions = config.EnableSuggestions
	l.errorRecovery.config.VerboseMessages = config.VerboseMessages
}
