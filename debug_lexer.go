package main

import (
	"fmt"
	"os"

	"github.com/orizon-lang/orizon/internal/lexer"
)

func main() {
	var input string
	if len(os.Args) > 1 {
		input = os.Args[1]
	} else {
		input = "func main() { print(\"Hello, Orizon!\"); }"
	}

	fmt.Printf("Input: '%s'\n", input)
	fmt.Println("Tokens:")

	l := lexer.New(input)

	for {
		token := l.NextToken()
		fmt.Printf("  Type: %s, Literal: %q, Position: %d:%d\n",
			tokenNames[token.Type], token.Literal, token.Line, token.Column)

		if token.Type == lexer.TokenEOF {
			break
		}
	}
}

// tokenNames provides string representations for token types for debugging
var tokenNames = map[lexer.TokenType]string{
	lexer.TokenEOF:        "EOF",
	lexer.TokenError:      "ERROR",
	lexer.TokenNewline:    "NEWLINE",
	lexer.TokenWhitespace: "WHITESPACE",
	lexer.TokenComment:    "COMMENT",

	lexer.TokenIdentifier: "IDENTIFIER",
	lexer.TokenInteger:    "INTEGER",
	lexer.TokenFloat:      "FLOAT",
	lexer.TokenString:     "STRING",
	lexer.TokenChar:       "CHAR",
	lexer.TokenBool:       "BOOL",

	lexer.TokenFunc:     "FUNC",
	lexer.TokenLet:      "LET",
	lexer.TokenVar:      "VAR",
	lexer.TokenConst:    "CONST",
	lexer.TokenStruct:   "STRUCT",
	lexer.TokenEnum:     "ENUM",
	lexer.TokenTrait:    "TRAIT",
	lexer.TokenImpl:     "IMPL",
	lexer.TokenIf:       "IF",
	lexer.TokenElse:     "ELSE",
	lexer.TokenFor:      "FOR",
	lexer.TokenWhile:    "WHILE",
	lexer.TokenLoop:     "LOOP",
	lexer.TokenMatch:    "MATCH",
	lexer.TokenReturn:   "RETURN",
	lexer.TokenBreak:    "BREAK",
	lexer.TokenContinue: "CONTINUE",
	lexer.TokenAsync:    "ASYNC",
	lexer.TokenAwait:    "AWAIT",
	lexer.TokenActor:    "ACTOR",
	lexer.TokenSpawn:    "SPAWN",
	lexer.TokenImport:   "IMPORT",
	lexer.TokenExport:   "EXPORT",
	lexer.TokenModule:   "MODULE",
	lexer.TokenPub:      "PUB",
	lexer.TokenMut:      "MUT",
	lexer.TokenAs:       "AS",
	lexer.TokenIn:       "IN",
	lexer.TokenWhere:    "WHERE",
	lexer.TokenUnsafe:   "UNSAFE",

	lexer.TokenPlus:         "PLUS",
	lexer.TokenMinus:        "MINUS",
	lexer.TokenMul:          "MUL",
	lexer.TokenDiv:          "DIV",
	lexer.TokenMod:          "MOD",
	lexer.TokenPower:        "POWER",
	lexer.TokenAssign:       "ASSIGN",
	lexer.TokenPlusAssign:   "PLUS_ASSIGN",
	lexer.TokenMinusAssign:  "MINUS_ASSIGN",
	lexer.TokenMulAssign:    "MUL_ASSIGN",
	lexer.TokenDivAssign:    "DIV_ASSIGN",
	lexer.TokenModAssign:    "MOD_ASSIGN",
	lexer.TokenEq:           "EQ",
	lexer.TokenNe:           "NE",
	lexer.TokenLt:           "LT",
	lexer.TokenLe:           "LE",
	lexer.TokenGt:           "GT",
	lexer.TokenGe:           "GE",
	lexer.TokenAnd:          "AND",
	lexer.TokenOr:           "OR",
	lexer.TokenNot:          "NOT",
	lexer.TokenBitAnd:       "BIT_AND",
	lexer.TokenBitOr:        "BIT_OR",
	lexer.TokenBitXor:       "BIT_XOR",
	lexer.TokenBitNot:       "BIT_NOT",
	lexer.TokenShl:          "SHL",
	lexer.TokenShr:          "SHR",
	lexer.TokenBitAndAssign: "BIT_AND_ASSIGN",
	lexer.TokenBitOrAssign:  "BIT_OR_ASSIGN",
	lexer.TokenBitXorAssign: "BIT_XOR_ASSIGN",
	lexer.TokenShlAssign:    "SHL_ASSIGN",
	lexer.TokenShrAssign:    "SHR_ASSIGN",

	lexer.TokenLParen:      "LPAREN",
	lexer.TokenRParen:      "RPAREN",
	lexer.TokenLBrace:      "LBRACE",
	lexer.TokenRBrace:      "RBRACE",
	lexer.TokenLBracket:    "LBRACKET",
	lexer.TokenRBracket:    "RBRACKET",
	lexer.TokenSemicolon:   "SEMICOLON",
	lexer.TokenComma:       "COMMA",
	lexer.TokenDot:         "DOT",
	lexer.TokenColon:       "COLON",
	lexer.TokenDoubleColon: "DOUBLE_COLON",
	lexer.TokenArrow:       "ARROW",
	lexer.TokenFatArrow:    "FAT_ARROW",
	lexer.TokenQuestion:    "QUESTION",
	lexer.TokenAt:          "AT",
	lexer.TokenHash:        "HASH",
	lexer.TokenDollar:      "DOLLAR",
	lexer.TokenTilde:       "TILDE",
	lexer.TokenBackslash:   "BACKSLASH",
	lexer.TokenPipe:        "PIPE",
	lexer.TokenAmpersand:   "AMPERSAND",
	lexer.TokenCaret:       "CARET",
	lexer.TokenPercent:     "PERCENT",
	lexer.TokenExclamation: "EXCLAMATION",
}
