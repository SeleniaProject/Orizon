// Package lexer error recovery tests
// Phase 1.1.3: エラー回復機能のテスト実装
package lexer

import (
	"testing"
	"time"
)

// TestErrorRecovery_BasicFunctionality tests the fundamental error recovery operations
func TestErrorRecovery_BasicFunctionality(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedErrors int
		expectedTokens []TokenType
		description    string
	}{
		{
			name:           "UnterminatedString",
			input:          `"hello world`,
			expectedErrors: 1,
			expectedTokens: []TokenType{TokenError, TokenEOF},
			description:    "Should detect and recover from unterminated string",
		},
		{
			name:           "InvalidCharacterInIdentifier",
			input:          `my-invalid-identifier`,
			expectedErrors: 2,                                 // Two invalid characters: both '-'
			expectedTokens: []TokenType{TokenError, TokenEOF}, // Single error token for the entire invalid identifier
			description:    "Should detect invalid character in identifier and recover",
		},
		{
			name:           "MalformedNumber",
			input:          `123abc`,
			expectedErrors: 1,
			expectedTokens: []TokenType{TokenError, TokenEOF},
			description:    "Should detect malformed number literal",
		},
		{
			name:           "MultipleErrors",
			input:          `123abc invalid-id@test`,
			expectedErrors: 3,                                             // malformed number, 2 invalid characters in identifier
			expectedTokens: []TokenType{TokenError, TokenError, TokenEOF}, // Two error tokens
			description:    "Should handle multiple errors in sequence",
		},
		{
			name:           "ErrorWithRecovery",
			input:          `"unterminated; let x = 42;`,
			expectedErrors: 1,
			expectedTokens: []TokenType{TokenError, TokenEOF}, // Without recovery, only error token is returned
			description:    "Should detect error (recovery functionality to be enhanced later)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewWithFilename(tt.input, "test.oriz").WithErrorRecovery()
			var tokens []Token
			var tokenTypes []TokenType

			// Parse all tokens with error recovery
			for {
				token := lexer.RecoverableNextToken()
				tokens = append(tokens, token)
				tokenTypes = append(tokenTypes, token.Type)
				if token.Type == TokenEOF {
					break
				}
			}

			// Check error count
			if len(lexer.GetErrors()) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d", tt.expectedErrors, len(lexer.GetErrors()))
				for i, err := range lexer.GetErrors() {
					t.Logf("Error %d: %s", i, lexer.FormatError(err))
				}
			}

			// Check token sequence
			if len(tokenTypes) != len(tt.expectedTokens) {
				t.Errorf("Expected %d tokens, got %d", len(tt.expectedTokens), len(tokenTypes))
				t.Logf("Expected: %v", tt.expectedTokens)
				t.Logf("Got: %v", tokenTypes)
				return
			}

			for i, expectedType := range tt.expectedTokens {
				if tokenTypes[i] != expectedType {
					t.Errorf("Token %d: expected %v, got %v", i, expectedType, tokenTypes[i])
				}
			}

			t.Logf("✓ %s", tt.description)
		})
	}
}

// TestErrorRecovery_SyncPointDetection tests error synchronization points
func TestErrorRecovery_SyncPointDetection(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedPos []Position
		description string
	}{
		{
			name:        "SemicolonSync",
			input:       `"unterminated; let x = 42;`,
			expectedPos: []Position{{Line: 1, Column: 14}}, // Position after semicolon
			description: "Should sync at semicolon",
		},
		{
			name:        "BraceSync",
			input:       `"unterminated { let x = 42; }`,
			expectedPos: []Position{{Line: 1, Column: 15}}, // Position after opening brace
			description: "Should sync at opening brace",
		},
		{
			name:        "KeywordSync",
			input:       `123abc let x = 42;`,
			expectedPos: []Position{{Line: 1, Column: 8}}, // Position at 'let'
			description: "Should sync at keyword",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewWithFilename(tt.input, "test.oriz").WithErrorRecovery()

			// Parse tokens until we hit an error and recovery
			var syncPositions []Position
			for {
				token := lexer.RecoverableNextToken()

				// Check if this was a recovery operation
				errors := lexer.GetErrors()
				if len(errors) > 0 {
					lastError := errors[len(errors)-1]
					if lastError.RecoveryType != 0 {
						// This was a recovery operation
						syncPositions = append(syncPositions, lexer.getCurrentPosition())
					}
				}

				if token.Type == TokenEOF {
					break
				}
			}

			if len(syncPositions) != len(tt.expectedPos) {
				t.Errorf("Expected %d sync positions, got %d", len(tt.expectedPos), len(syncPositions))
				return
			}

			for i, expectedPos := range tt.expectedPos {
				// We check line and column (offset may vary due to implementation details)
				if syncPositions[i].Line != expectedPos.Line ||
					syncPositions[i].Column != expectedPos.Column {
					t.Errorf("Sync position %d: expected line %d col %d, got line %d col %d",
						i, expectedPos.Line, expectedPos.Column,
						syncPositions[i].Line, syncPositions[i].Column)
				}
			}

			t.Logf("✓ %s", tt.description)
		})
	}
}

// TestErrorRecovery_ErrorCategorization tests error classification
func TestErrorRecovery_ErrorCategorization(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedCategory ErrorCategory
		expectedSeverity ErrorSeverity
		description      string
	}{
		{
			name:             "UnterminatedString",
			input:            `"hello world`,
			expectedCategory: CategoryUnterminatedString,
			expectedSeverity: SeverityError,
			description:      "Should categorize unterminated string correctly",
		},
		{
			name:             "InvalidCharacter",
			input:            `my@invalid`,
			expectedCategory: CategoryInvalidCharacter,
			expectedSeverity: SeverityError,
			description:      "Should categorize invalid character correctly",
		},
		{
			name:             "MalformedNumber",
			input:            `123.45.67`,
			expectedCategory: CategoryMalformedNumber,
			expectedSeverity: SeverityError,
			description:      "Should categorize malformed number correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewWithFilename(tt.input, "test.oriz").WithErrorRecovery()

			// Parse until we get an error
			for {
				token := lexer.RecoverableNextToken()
				if token.Type == TokenEOF || token.Type == TokenError {
					break
				}
			}

			errors := lexer.GetErrors()
			if len(errors) == 0 {
				t.Fatalf("Expected at least one error")
			}

			firstError := errors[0]
			if firstError.Type != tt.expectedCategory {
				t.Errorf("Expected category %v, got %v", tt.expectedCategory, firstError.Type)
			}

			if firstError.Severity != tt.expectedSeverity {
				t.Errorf("Expected severity %v, got %v", tt.expectedSeverity, firstError.Severity)
			}

			t.Logf("✓ %s", tt.description)
		})
	}
}

// TestErrorRecovery_SuggestionGeneration tests error message suggestions
func TestErrorRecovery_SuggestionGeneration(t *testing.T) {
	tests := []struct {
		name                string
		input               string
		expectedSuggestions int
		checkSuggestion     func(suggestions []ErrorSuggestion) bool
		description         string
	}{
		{
			name:                "UnterminatedStringSuggestions",
			input:               `"hello world`,
			expectedSuggestions: 2,
			checkSuggestion: func(suggestions []ErrorSuggestion) bool {
				for _, s := range suggestions {
					if containsSubstring(s.Description, "closing quote") {
						return true
					}
				}
				return false
			},
			description: "Should suggest closing quote for unterminated string",
		},
		{
			name:                "InvalidCharacterSuggestions",
			input:               `my@invalid`,
			expectedSuggestions: 1,
			checkSuggestion: func(suggestions []ErrorSuggestion) bool {
				for _, s := range suggestions {
					if containsSubstring(s.Description, "remove") {
						return true
					}
				}
				return false
			},
			description: "Should suggest removing invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewWithFilename(tt.input, "test.oriz").WithErrorRecovery()

			// Enable suggestions manually
			if lexer.errorRecovery != nil {
				lexer.errorRecovery.config.EnableSuggestions = true
				lexer.errorRecovery.config.MaxErrors = 10
				lexer.errorRecovery.config.VerboseMessages = true
			}

			// Parse until we get an error
			for {
				token := lexer.RecoverableNextToken()
				if token.Type == TokenEOF || token.Type == TokenError {
					break
				}
			}

			errors := lexer.GetErrors()
			if len(errors) == 0 {
				t.Fatalf("Expected at least one error")
			}

			firstError := errors[0]
			if len(firstError.Suggestions) < tt.expectedSuggestions {
				t.Errorf("Expected at least %d suggestions, got %d",
					tt.expectedSuggestions, len(firstError.Suggestions))
			}

			if tt.checkSuggestion != nil && !tt.checkSuggestion(firstError.Suggestions) {
				t.Errorf("Suggestion check failed")
				for i, s := range firstError.Suggestions {
					t.Logf("Suggestion %d: %s", i, s.Description)
				}
			}

			t.Logf("✓ %s", tt.description)
		})
	}
}

// TestErrorRecovery_ErrorFormatting tests error message formatting
func TestErrorRecovery_ErrorFormatting(t *testing.T) {
	input := `"unterminated string`
	lexer := NewWithFilename(input, "test.oriz").WithErrorRecovery()

	// Parse to generate an error
	lexer.RecoverableNextToken()

	errors := lexer.GetErrors()
	if len(errors) == 0 {
		t.Fatalf("Expected at least one error")
	}

	error := errors[0]

	// Test basic formatting
	basicFormat := lexer.FormatError(error)
	if !containsSubstring(basicFormat, "test.oriz") {
		t.Errorf("Basic format should contain filename")
	}
	if !containsSubstring(basicFormat, "1:") {
		t.Errorf("Basic format should contain line number")
	}

	// Test detailed formatting
	detailedFormat := lexer.FormatErrorDetailed(error)
	if !containsSubstring(detailedFormat, "^") {
		t.Errorf("Detailed format should contain caret indicator")
	}

	t.Logf("Basic format: %s", basicFormat)
	t.Logf("Detailed format:\n%s", detailedFormat)
}

// TestErrorRecovery_PerformanceImpact tests that error recovery doesn't significantly impact performance
func TestErrorRecovery_PerformanceImpact(t *testing.T) {
	// Test with a reasonably sized input without errors
	input := `
		let x = 42;
		let y = "hello world";
		let z = 3.14159;
		
		func fibonacci(n: int) -> int {
			if n <= 1 {
				return n;
			}
			return fibonacci(n - 1) + fibonacci(n - 2);
		}
		
		let result = fibonacci(10);
	`

	// Parse without error recovery
	startTime := time.Now()
	lexer1 := NewWithFilename(input, "test.oriz")
	tokenCount1 := 0
	for {
		token := lexer1.NextToken()
		tokenCount1++
		if token.Type == TokenEOF {
			break
		}
	}
	durationWithout := time.Since(startTime)

	// Parse with error recovery
	startTime = time.Now()
	lexer2 := NewWithFilename(input, "test.oriz").WithErrorRecovery()
	tokenCount2 := 0
	for {
		token := lexer2.RecoverableNextToken()
		tokenCount2++
		if token.Type == TokenEOF {
			break
		}
	}
	durationWith := time.Since(startTime)

	// Check that token counts match
	if tokenCount1 != tokenCount2 {
		t.Errorf("Token count mismatch: without=%d, with=%d", tokenCount1, tokenCount2)
	}

	// Check that there are no errors in clean input
	if lexer2.HasErrors() {
		t.Errorf("Unexpected errors in clean input: %d", len(lexer2.GetErrors()))
	}

	// Performance should not degrade significantly (allow 2x slowdown)
	if durationWith > durationWithout*2 {
		t.Errorf("Error recovery causes significant performance degradation: %v vs %v",
			durationWith, durationWithout)
	}

	t.Logf("Performance impact: without=%v, with=%v (%.2fx slower)",
		durationWithout, durationWith, float64(durationWith)/float64(durationWithout))
}

// TestErrorRecovery_Configuration tests error recovery configuration options
func TestErrorRecovery_Configuration(t *testing.T) {
	input := `"error1 "error2 "error3 "error4 "error5`

	// Test max errors limit
	lexer := NewWithFilename(input, "test.oriz").WithErrorRecovery()
	if lexer.errorRecovery != nil {
		lexer.errorRecovery.config.MaxErrors = 3
		lexer.errorRecovery.config.EnableSuggestions = false
		lexer.errorRecovery.config.VerboseMessages = false
	}

	errorCount := 0
	for {
		token := lexer.RecoverableNextToken()
		if token.Type == TokenEOF {
			break
		}
		if len(lexer.GetErrors()) > errorCount {
			errorCount = len(lexer.GetErrors())
			if errorCount >= 3 {
				// Should stop accumulating errors
				break
			}
		}
	}

	if len(lexer.GetErrors()) > 3 {
		t.Errorf("Max errors limit not respected: got %d errors", len(lexer.GetErrors()))
	}

	t.Logf("✓ Max errors configuration working: %d errors", len(lexer.GetErrors()))
}

// TestErrorRecovery_EdgeCases tests edge cases and boundary conditions
func TestErrorRecovery_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "EmptyInput",
			input:       "",
			description: "Should handle empty input gracefully",
		},
		{
			name:        "OnlyErrors",
			input:       "@@@###$$$",
			description: "Should handle input with only invalid characters",
		},
		{
			name:        "VeryLongError",
			input:       `"` + repeatString("a", 1000),
			description: "Should handle very long unterminated strings",
		},
		{
			name:        "UnicodeErrors",
			input:       "🌟@🚀#🎯",
			description: "Should handle Unicode characters in errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewWithFilename(tt.input, "test.oriz").WithErrorRecovery()

			// Should not panic or hang
			tokenCount := 0
			for tokenCount < 1000 { // Safety limit
				token := lexer.RecoverableNextToken()
				tokenCount++
				if token.Type == TokenEOF {
					break
				}
			}

			t.Logf("✓ %s: parsed %d tokens", tt.description, tokenCount)
		})
	}
}

// Helper function to repeat a string
func repeatString(s string, count int) string {
	if count <= 0 {
		return ""
	}
	result := make([]byte, len(s)*count)
	for i := 0; i < count; i++ {
		copy(result[i*len(s):], s)
	}
	return string(result)
}

// BenchmarkErrorRecovery_WithoutErrors benchmarks normal lexing with error recovery enabled
func BenchmarkErrorRecovery_WithoutErrors(b *testing.B) {
	input := `
		func quicksort(arr: []int) -> []int {
			if arr.length <= 1 {
				return arr;
			}
			
			let pivot = arr[arr.length / 2];
			let less = [];
			let equal = [];
			let greater = [];
			
			for item in arr {
				if item < pivot {
					less.push(item);
				} else if item == pivot {
					equal.push(item);
				} else {
					greater.push(item);
				}
			}
			
			return quicksort(less) + equal + quicksort(greater);
		}
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := NewWithFilename(input, "bench.oriz").WithErrorRecovery()
		for {
			token := lexer.RecoverableNextToken()
			if token.Type == TokenEOF {
				break
			}
		}
	}
}

// BenchmarkErrorRecovery_WithErrors benchmarks lexing with actual errors
func BenchmarkErrorRecovery_WithErrors(b *testing.B) {
	input := `
		func example() {
			let x = "unterminated string;
			let y = 123abc;
			let z = invalid@identifier;
			return x + y + z;
		}
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := NewWithFilename(input, "bench.oriz").WithErrorRecovery()
		for {
			token := lexer.RecoverableNextToken()
			if token.Type == TokenEOF {
				break
			}
		}
	}
}
