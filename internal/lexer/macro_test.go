package lexer

import (
	"testing"
)

// TestMacroTokens tests lexer support for macro-related tokens.
func TestMacroTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "Macro keyword",
			input: "macro",
			expected: []Token{
				{Type: TokenMacro, Literal: "macro"},
				{Type: TokenEOF, Literal: ""},
			},
		},
		{
			name:  "Macro invocation operator",
			input: "println!",
			expected: []Token{
				{Type: TokenIdentifier, Literal: "println"},
				{Type: TokenMacroInvoke, Literal: "!"},
				{Type: TokenEOF, Literal: ""},
			},
		},
		{
			name:  "Macro definition",
			input: "macro test() { x }",
			expected: []Token{
				{Type: TokenMacro, Literal: "macro"},
				{Type: TokenIdentifier, Literal: "test"},
				{Type: TokenLParen, Literal: "("},
				{Type: TokenRParen, Literal: ")"},
				{Type: TokenLBrace, Literal: "{"},
				{Type: TokenIdentifier, Literal: "x"},
				{Type: TokenRBrace, Literal: "}"},
				{Type: TokenEOF, Literal: ""},
			},
		},
		{
			name:  "Macro invocation with arguments",
			input: "println!(42, \"hello\")",
			expected: []Token{
				{Type: TokenIdentifier, Literal: "println"},
				{Type: TokenMacroInvoke, Literal: "!"},
				{Type: TokenLParen, Literal: "("},
				{Type: TokenInteger, Literal: "42"},
				{Type: TokenComma, Literal: ","},
				{Type: TokenString, Literal: "hello"},
				{Type: TokenRParen, Literal: ")"},
				{Type: TokenEOF, Literal: ""},
			},
		},
		{
			name:  "Logical NOT vs Macro invocation",
			input: "!flag println!",
			expected: []Token{
				{Type: TokenNot, Literal: "!"},
				{Type: TokenIdentifier, Literal: "flag"},
				{Type: TokenIdentifier, Literal: "println"},
				{Type: TokenMacroInvoke, Literal: "!"},
				{Type: TokenEOF, Literal: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input)

			for i, expectedToken := range tt.expected {
				tok := l.NextToken()

				if tok.Type != expectedToken.Type {
					t.Errorf("Token %d: expected type %v, got %v",
						i, expectedToken.Type, tok.Type)
				}

				if tok.Literal != expectedToken.Literal {
					t.Errorf("Token %d: expected literal %q, got %q",
						i, expectedToken.Literal, tok.Literal)
				}
			}

			t.Logf("✅ Macro token test %s: all tokens match", tt.name)
		})
	}
}

// TestMacroKeywordRecognition tests that 'macro' is recognized as a keyword.
func TestMacroKeywordRecognition(t *testing.T) {
	input := "macro test_macro func"
	l := New(input)

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenMacro, "macro"},
		{TokenIdentifier, "test_macro"},
		{TokenFunc, "func"},
		{TokenEOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Errorf("Token %d: expected type %v, got %v",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Errorf("Token %d: expected literal %q, got %q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}

	t.Log("✅ Macro keyword recognition test passed")
}

// TestMacroInvokeOperator tests the macro invocation operator recognition.
func TestMacroInvokeOperator(t *testing.T) {
	input := "ident!("
	l := New(input)

	// Should get: IDENT, MACRO_INVOKE, LPAREN, EOF.
	tok1 := l.NextToken()
	if tok1.Type != TokenIdentifier || tok1.Literal != "ident" {
		t.Errorf("Expected TokenIdentifier 'ident', got %v %q", tok1.Type, tok1.Literal)
	}

	tok2 := l.NextToken()
	if tok2.Type != TokenMacroInvoke || tok2.Literal != "!" {
		t.Errorf("Expected TokenMacroInvoke '!', got %v %q", tok2.Type, tok2.Literal)
	}

	tok3 := l.NextToken()
	if tok3.Type != TokenLParen || tok3.Literal != "(" {
		t.Errorf("Expected TokenLParen '(', got %v %q", tok3.Type, tok3.Literal)
	}

	t.Log("✅ Macro invoke operator test passed")
}
