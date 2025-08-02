package lexer

import "testing"

func TestBasicTokens(t *testing.T) {
	input := `func main() {
	print("Hello, Orizon!");
}`

	tests := []struct {
		expectedType  TokenType
		expectedValue string
	}{
		{TokenFunc, "func"},
		{TokenIdentifier, "main"},
		{TokenLParen, "("},
		{TokenRParen, ")"},
		{TokenLBrace, "{"},
		{TokenNewline, "\n"},
		{TokenIdentifier, "print"},
		{TokenLParen, "("},
		{TokenString, "Hello, Orizon!"},
		{TokenRParen, ")"},
		{TokenSemicolon, ";"},
		{TokenNewline, "\n"},
		{TokenRBrace, "}"},
		{TokenEOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedValue {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedValue, tok.Literal)
		}
	}
}

func TestKeywords(t *testing.T) {
	input := `func let var const struct enum trait impl`

	tests := []struct {
		expectedType  TokenType
		expectedValue string
	}{
		{TokenFunc, "func"},
		{TokenLet, "let"},
		{TokenVar, "var"},
		{TokenConst, "const"},
		{TokenStruct, "struct"},
		{TokenEnum, "enum"},
		{TokenTrait, "trait"},
		{TokenImpl, "impl"},
		{TokenEOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedValue {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedValue, tok.Literal)
		}
	}
}
