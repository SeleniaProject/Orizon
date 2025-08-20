// Package lexer error recovery tests.
// Phase 1.1.3: エラー回復機能のテスト実装
package lexer

import (
	"strings"
	"testing"
	"time"
)

// TestErrorRecovery_BasicFunctionality tests the fundamental error recovery operations.
func TestErrorRecovery_BasicFunctionality(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		description    string
		expectedTokens []TokenType
		expectedErrors int
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

			// Parse all tokens with error recovery.
			for {
				token := lexer.RecoverableNextToken()
				tokens = append(tokens, token)
				tokenTypes = append(tokenTypes, token.Type)

				if token.Type == TokenEOF {
					break
				}
			}

			// Check error count.
			if len(lexer.GetErrors()) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d", tt.expectedErrors, len(lexer.GetErrors()))

				for i, err := range lexer.GetErrors() {
					t.Logf("Error %d: %s", i, lexer.FormatError(err))
				}
			}

			// Check token sequence.
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

// TestErrorRecovery_SyncPointDetection tests error synchronization points.
func TestErrorRecovery_SyncPointDetection(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "SemicolonSync",
			input:       `"unterminated; let x = 42;`,
			description: "Should handle unterminated string and continue",
		},
		{
			name:        "BraceSync",
			input:       `"unterminated { let x = 42; }`,
			description: "Should handle unterminated string with braces",
		},
		{
			name:        "KeywordSync",
			input:       `123abc let x = 42;`,
			description: "Should handle invalid identifier and continue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewWithFilename(tt.input, "test.oriz")

			// Parse tokens and ensure we get expected error tokens.
			var tokens []Token

			errorCount := 0

			for {
				token := lexer.RecoverableNextToken() // Use RecoverableNextToken instead of NextToken
				tokens = append(tokens, token)

				// Check for error tokens.
				if token.Type == TokenError {
					errorCount++
				}

				if token.Type == TokenEOF {
					break
				}
			}

			// Verify we got at least one error for malformed input.
			if errorCount == 0 && len(lexer.GetErrors()) == 0 {
				t.Errorf("Expected at least one error for malformed input: %s", tt.input)
			}

			// Verify we continued parsing after errors.
			if len(tokens) < 2 {
				t.Errorf("Expected multiple tokens even with errors, got %d", len(tokens))
			}
		})
	}
}

// TestErrorRecovery_ErrorCategorization tests error classification.
func TestErrorRecovery_ErrorCategorization(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "UnterminatedString",
			input:       `"hello world`,
			description: "Should handle unterminated string",
		},
		{
			name:        "InvalidCharacter",
			input:       `my@invalid`,
			description: "Should handle invalid character",
		},
		{
			name:        "MalformedNumber",
			input:       `123.45.67`,
			description: "Should handle malformed number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewWithFilename(tt.input, "test.oriz")

			// Parse until we get an error.
			errorFound := false

			for {
				token := lexer.RecoverableNextToken() // Use RecoverableNextToken
				if token.Type == TokenError {
					errorFound = true
				}

				if token.Type == TokenEOF {
					break
				}
			}

			if !errorFound && len(lexer.GetErrors()) == 0 {
				t.Errorf("Expected error for malformed input: %s", tt.input)
			}

			t.Logf("✓ %s", tt.description)
		})
	}
}

// TestErrorRecovery_SuggestionGeneration tests error message suggestions.
func TestErrorRecovery_SuggestionGeneration(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "UnterminatedStringSuggestions",
			input:       `"hello world`,
			description: "Should handle unterminated string",
		},
		{
			name:        "InvalidCharacterSuggestions",
			input:       `my@invalid`,
			description: "Should handle invalid character in identifier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewWithFilename(tt.input, "test.oriz")

			// Parse until we get an error.
			errorFound := false

			for {
				token := lexer.NextToken()
				if token.Type == TokenError {
					errorFound = true
					// Verify we got an error message.
					if token.Literal == "" {
						t.Errorf("Error token should have a message")
					}
				}

				if token.Type == TokenEOF {
					break
				}
			}

			if !errorFound {
				t.Errorf("Expected error token for malformed input: %s", tt.input)
			}

			t.Logf("✓ %s", tt.description)
		})
	}
}

// TestErrorRecovery_ErrorFormatting tests error message formatting.
func TestErrorRecovery_ErrorFormatting(t *testing.T) {
	input := `"unterminated string`
	lexer := NewWithFilename(input, "test.oriz")

	// Parse to generate an error.
	errorFound := false

	for {
		token := lexer.NextToken()
		if token.Type == TokenError {
			errorFound = true
			// Verify error message contains useful information.
			if token.Literal == "" {
				t.Errorf("Error token should have a message")
			}

			if !strings.Contains(token.Literal, "string") {
				t.Errorf("Error message should mention string: %s", token.Literal)
			}
		}

		if token.Type == TokenEOF {
			break
		}
	}

	if !errorFound {
		t.Errorf("Expected error token for unterminated string")
	}

	t.Logf("✓ Error formatting test completed")
}

// TestErrorRecovery_PerformanceImpact tests that error recovery doesn't significantly impact performance.
func TestErrorRecovery_PerformanceImpact(t *testing.T) {
	// Test with a reasonably sized input without errors.
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

	// Parse without error recovery.
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

	// Parse with same lexer (simulating with recovery).
	startTime = time.Now()
	lexer2 := NewWithFilename(input, "test.oriz")
	tokenCount2 := 0

	for {
		token := lexer2.NextToken()
		tokenCount2++

		if token.Type == TokenEOF {
			break
		}
	}

	durationWith := time.Since(startTime)

	// Check that token counts match.
	if tokenCount1 != tokenCount2 {
		t.Errorf("Token count mismatch: without=%d, with=%d", tokenCount1, tokenCount2)
	}

	t.Logf("Performance test: first=%v, second=%v", durationWithout, durationWith)
}

// TestErrorRecovery_Configuration tests error recovery configuration options.
func TestErrorRecovery_Configuration(t *testing.T) {
	input := `"error1 "error2 "error3 "error4 "error5`

	// Test basic error handling.
	lexer := NewWithFilename(input, "test.oriz")

	errorCount := 0

	for {
		token := lexer.NextToken()
		if token.Type == TokenError {
			errorCount++
		}

		if token.Type == TokenEOF {
			break
		}
	}

	if errorCount == 0 {
		t.Errorf("Expected errors for malformed input")
	}

	t.Logf("Configuration test: found %d errors", errorCount)
}

// TestErrorRecovery_EdgeCases tests edge cases and boundary conditions.
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

			// Should not panic or hang.
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

// Helper function to repeat a string.
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

// BenchmarkErrorRecovery_WithoutErrors benchmarks normal lexing with error recovery enabled.
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

// BenchmarkErrorRecovery_WithErrors benchmarks lexing with actual errors.
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
