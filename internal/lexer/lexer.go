// Package lexer implements the Orizon lexical analyzer.
// Phase 1.1.1: Unicode対応字句解析器実装 (修正版)
package lexer

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// TokenType represents the type of a token
type TokenType int

// String returns a string representation of the token type
func (tt TokenType) String() string {
	if name, ok := tokenNames[tt]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN(%d)", int(tt))
}

// Token types - Orizon言語のトークン定義
const (
	// 特殊トークン
	TokenEOF TokenType = iota
	TokenError
	TokenNewline
	TokenWhitespace
	TokenComment

	// リテラル
	TokenIdentifier
	TokenInteger
	TokenFloat
	TokenString
	TokenChar
	TokenBool

	// キーワード
	TokenFunc
	TokenLet
	TokenVar
	TokenConst
	TokenStruct
	TokenEnum
	TokenTrait
	TokenImpl
	TokenIf
	TokenElse
	TokenFor
	TokenWhile
	TokenLoop
	TokenMatch
	TokenReturn
	TokenBreak
	TokenContinue
	TokenAsync
	TokenAwait
	TokenActor
	TokenSpawn
	TokenImport
	TokenExport
	TokenModule
	TokenPub
	TokenMut
	TokenAs
	TokenIn
	TokenWhere
	TokenUnsafe

	// マクロ関連
	TokenMacro
	TokenMacroInvoke  // !
	TokenBackquote    // `
	TokenMacroPattern // Pattern matching markers
	TokenMacroRepeat  // Repetition markers (* + ?)
	TokenMacroGroup   // Grouping markers

	// 演算子
	TokenPlus
	TokenMinus
	TokenMul
	TokenDiv
	TokenMod
	TokenPower
	TokenAssign
	TokenPlusAssign
	TokenMinusAssign
	TokenMulAssign
	TokenDivAssign
	TokenModAssign
	TokenEq
	TokenNe
	TokenLt
	TokenLe
	TokenGt
	TokenGe
	TokenAnd
	TokenOr
	TokenNot
	TokenBitAnd
	TokenBitOr
	TokenBitXor
	TokenBitNot
	TokenShl
	TokenShr
	TokenBitAndAssign
	TokenBitOrAssign
	TokenBitXorAssign
	TokenShlAssign
	TokenShrAssign

	// 記号
	TokenLParen
	TokenRParen
	TokenLBrace
	TokenRBrace
	TokenLBracket
	TokenRBracket
	TokenSemicolon
	TokenComma
	TokenDot
	TokenColon
	TokenDoubleColon
	TokenArrow
	TokenFatArrow
	TokenQuestion
	TokenAt
	TokenHash
	TokenDollar
	TokenTilde
	TokenBackslash
	TokenPipe
	TokenAmpersand
	TokenCaret
	TokenPercent
	TokenExclamation
)

// Position represents a position in the source code
type Position struct {
	Line   int // 1-based line number
	Column int // 1-based column number
	Offset int // 0-based byte offset in source
}

// Span represents a range in the source code
type Span struct {
	Start Position
	End   Position
}

// Token represents a lexical token with position information
type Token struct {
	Type    TokenType
	Literal string
	Span    Span // Source code span for this token

	// Legacy compatibility fields (deprecated - use Span instead)
	Line   int
	Column int
}

// String returns a string representation of the token
func (t Token) String() string {
	return fmt.Sprintf("{Type: %s, Literal: %q, Line: %d, Column: %d}",
		tokenNames[t.Type], t.Literal, t.Line, t.Column)
}

// tokenNames provides string representations for token types
var tokenNames = map[TokenType]string{
	TokenEOF:        "EOF",
	TokenError:      "ERROR",
	TokenNewline:    "NEWLINE",
	TokenWhitespace: "WHITESPACE",
	TokenComment:    "COMMENT",

	TokenIdentifier: "IDENTIFIER",
	TokenInteger:    "INTEGER",
	TokenFloat:      "FLOAT",
	TokenString:     "STRING",
	TokenChar:       "CHAR",
	TokenBool:       "BOOL",

	TokenFunc:     "FUNC",
	TokenLet:      "LET",
	TokenVar:      "VAR",
	TokenConst:    "CONST",
	TokenStruct:   "STRUCT",
	TokenEnum:     "ENUM",
	TokenTrait:    "TRAIT",
	TokenImpl:     "IMPL",
	TokenIf:       "IF",
	TokenElse:     "ELSE",
	TokenFor:      "FOR",
	TokenWhile:    "WHILE",
	TokenLoop:     "LOOP",
	TokenMatch:    "MATCH",
	TokenReturn:   "RETURN",
	TokenBreak:    "BREAK",
	TokenContinue: "CONTINUE",
	TokenAsync:    "ASYNC",
	TokenAwait:    "AWAIT",
	TokenActor:    "ACTOR",
	TokenSpawn:    "SPAWN",
	TokenImport:   "IMPORT",
	TokenExport:   "EXPORT",
	TokenModule:   "MODULE",
	TokenPub:      "PUB",
	TokenMut:      "MUT",
	TokenAs:       "AS",
	TokenIn:       "IN",
	TokenWhere:    "WHERE",
	TokenUnsafe:   "UNSAFE",

	// マクロ関連
	TokenMacro:        "MACRO",
	TokenMacroInvoke:  "MACRO_INVOKE",
	TokenBackquote:    "BACKQUOTE",
	TokenMacroPattern: "MACRO_PATTERN",
	TokenMacroRepeat:  "MACRO_REPEAT",
	TokenMacroGroup:   "MACRO_GROUP",

	TokenPlus:         "PLUS",
	TokenMinus:        "MINUS",
	TokenMul:          "MUL",
	TokenDiv:          "DIV",
	TokenMod:          "MOD",
	TokenPower:        "POWER",
	TokenAssign:       "ASSIGN",
	TokenPlusAssign:   "PLUS_ASSIGN",
	TokenMinusAssign:  "MINUS_ASSIGN",
	TokenMulAssign:    "MUL_ASSIGN",
	TokenDivAssign:    "DIV_ASSIGN",
	TokenModAssign:    "MOD_ASSIGN",
	TokenEq:           "EQ",
	TokenNe:           "NE",
	TokenLt:           "LT",
	TokenLe:           "LE",
	TokenGt:           "GT",
	TokenGe:           "GE",
	TokenAnd:          "AND",
	TokenOr:           "OR",
	TokenNot:          "NOT",
	TokenBitAnd:       "BIT_AND",
	TokenBitOr:        "BIT_OR",
	TokenBitXor:       "BIT_XOR",
	TokenBitNot:       "BIT_NOT",
	TokenShl:          "SHL",
	TokenShr:          "SHR",
	TokenBitAndAssign: "BIT_AND_ASSIGN",
	TokenBitOrAssign:  "BIT_OR_ASSIGN",
	TokenBitXorAssign: "BIT_XOR_ASSIGN",
	TokenShlAssign:    "SHL_ASSIGN",
	TokenShrAssign:    "SHR_ASSIGN",

	TokenLParen:      "LPAREN",
	TokenRParen:      "RPAREN",
	TokenLBrace:      "LBRACE",
	TokenRBrace:      "RBRACE",
	TokenLBracket:    "LBRACKET",
	TokenRBracket:    "RBRACKET",
	TokenSemicolon:   "SEMICOLON",
	TokenComma:       "COMMA",
	TokenDot:         "DOT",
	TokenColon:       "COLON",
	TokenDoubleColon: "DOUBLE_COLON",
	TokenArrow:       "ARROW",
	TokenFatArrow:    "FAT_ARROW",
	TokenQuestion:    "QUESTION",
	TokenAt:          "AT",
	TokenHash:        "HASH",
	TokenDollar:      "DOLLAR",
	TokenTilde:       "TILDE",
	TokenBackslash:   "BACKSLASH",
	TokenPipe:        "PIPE",
	TokenAmpersand:   "AMPERSAND",
	TokenCaret:       "CARET",
	TokenPercent:     "PERCENT",
	TokenExclamation: "EXCLAMATION",
}

// keywords maps string keywords to their token types
var keywords = map[string]TokenType{
	"func":     TokenFunc,
	"let":      TokenLet,
	"var":      TokenVar,
	"const":    TokenConst,
	"struct":   TokenStruct,
	"enum":     TokenEnum,
	"trait":    TokenTrait,
	"impl":     TokenImpl,
	"if":       TokenIf,
	"else":     TokenElse,
	"for":      TokenFor,
	"while":    TokenWhile,
	"loop":     TokenLoop,
	"match":    TokenMatch,
	"return":   TokenReturn,
	"break":    TokenBreak,
	"continue": TokenContinue,
	"async":    TokenAsync,
	"await":    TokenAwait,
	"actor":    TokenActor,
	"spawn":    TokenSpawn,
	"import":   TokenImport,
	"export":   TokenExport,
	"module":   TokenModule,
	"pub":      TokenPub,
	"mut":      TokenMut,
	"as":       TokenAs,
	"in":       TokenIn,
	"where":    TokenWhere,
	"unsafe":   TokenUnsafe,
	"macro":    TokenMacro,
	"true":     TokenBool,
	"false":    TokenBool,
}

// CacheEntry represents a cached token with its validity information
type CacheEntry struct {
	Token    Token
	IsValid  bool
	StartPos int // Start position in original input
	EndPos   int // End position in original input
}

// ChangeRegion represents a region that has been modified
type ChangeRegion struct {
	Start  int // Start offset of change
	End    int // End offset of change (in original text)
	Length int // Length of new text
}

// Lexer represents the lexical analyzer with incremental parsing support
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // current line number
	column       int  // current column number
	offset       int  // current byte offset in input

	// Incremental parsing state
	filename     string        // source filename for error reporting
	tokenCache   []CacheEntry  // cached tokens from previous parse
	changeRegion *ChangeRegion // region that has been modified
	cacheValid   bool          // whether cache is currently valid

	// Error recovery system (Phase 1.1.3)
	errorRecovery *ErrorRecovery // error recovery system
	errors        []any          // accumulated lexical errors (LexicalError)

	// Temporary state for error handling
	stringTerminated   bool // whether last string was properly terminated
	hasIdentifierError bool // whether identifier had invalid characters
}

// New creates a new lexer instance for complete parsing
func New(input string) *Lexer {
	return NewWithFilename(input, "")
}

// NewWithFilename creates a new lexer instance with filename for error reporting
func NewWithFilename(input, filename string) *Lexer {
	l := &Lexer{
		input:      input,
		line:       1,
		column:     0,
		offset:     0,
		filename:   filename,
		cacheValid: false,
	}

	// Initialize error recovery system
	l.errorRecovery = NewErrorRecovery()

	l.readChar()
	return l
}

// NewIncremental creates a lexer for incremental parsing
func NewIncremental(input, filename string, previousCache []CacheEntry, change *ChangeRegion) *Lexer {
	l := &Lexer{
		input:        input,
		line:         1,
		column:       0,
		offset:       0,
		filename:     filename,
		tokenCache:   previousCache,
		changeRegion: change,
		cacheValid:   change != nil,
	}
	l.readChar()
	return l
}

// readChar reads the next character and advances position
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII NUL character represents "EOF"
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.offset = l.position // Update byte offset

	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

// peekChar returns the next character without advancing position
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// skipWhitespace skips whitespace characters (except newlines)
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

// readIdentifier reads identifier or Unicode identifier - 修正版
func (l *Lexer) readIdentifier() string {
	position := l.position
	l.hasIdentifierError = false // Reset error flag

	// 最初の文字を処理
	if isLetter(l.ch) || l.ch == '_' {
		l.readChar()
	} else if l.ch >= 0x80 { // Unicode文字の開始
		_, size := utf8.DecodeRuneInString(l.input[l.position:])
		if size > 0 {
			r, _ := utf8.DecodeRuneInString(l.input[l.position:])
			if unicode.IsLetter(r) || unicode.IsSymbol(r) {
				for i := 0; i < size; i++ {
					l.readChar()
				}
			}
		}
	}

	// 残りの文字を読み続ける
	for {
		if isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
			l.readChar()
		} else if l.ch >= 0x80 {
			r, size := utf8.DecodeRuneInString(l.input[l.position:])
			if size == 0 {
				break
			}
			if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSymbol(r) {
				for i := 0; i < size; i++ {
					l.readChar()
				}
			} else {
				break
			}
		} else {
			// Check for invalid characters that might be typos in identifiers
			if l.ch == '-' || l.ch == '@' || l.ch == '#' || l.ch == '$' || l.ch == '%' || l.ch == '^' || l.ch == '&' || l.ch == '*' {
				if l.errorRecovery != nil {
					err := l.CreateInvalidCharacterError(rune(l.ch))
					l.addError(err)
				}
				l.hasIdentifierError = true
				l.readChar() // Skip the invalid character
				continue
			}
			break
		}
	}

	return l.input[position:l.position]
}

func (l *Lexer) readNumber() string {
	position := l.position
	hasDecimal := false

	// Read digits
	for isDigit(l.ch) {
		l.readChar()
	}

	// Handle decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		hasDecimal = true
		l.readChar() // consume '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	literal := l.input[position:l.position]

	// Check for malformed number (letters after digits)
	if isLetter(l.ch) || l.ch == '_' {
		// Malformed number - read the invalid part too
		invalidStart := l.position
		for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
			l.readChar()
		}

		if l.errorRecovery != nil {
			invalidPart := l.input[invalidStart:l.position]
			fullLiteral := literal + invalidPart
			err := l.CreateMalformedNumberError(fullLiteral)
			l.addError(err)
		}

		return l.input[position:l.position] // Return the full invalid literal
	}

	// Check for multiple decimal points
	if hasDecimal {
		remaining := l.input[l.position:]
		for i, r := range remaining {
			if r == '.' {
				// Found another decimal point - this is malformed
				if l.errorRecovery != nil {
					// Read the full malformed literal including the second decimal point
					extraPos := l.position + i + 1
					for extraPos < len(l.input) && isDigit(l.input[extraPos]) {
						extraPos++
					}
					malformedLiteral := l.input[position:extraPos]
					err := l.CreateMalformedNumberError(malformedLiteral)
					l.addError(err)
				}
				// Advance position to consume the malformed part
				for l.ch == '.' || isDigit(l.ch) {
					l.readChar()
				}
				return l.input[position:l.position]
			}
			if !isDigit(byte(r)) {
				break
			}
		}
	}

	return literal
}

func (l *Lexer) readString() string {
	position := l.position + 1 // 開始の引用符をスキップ
	terminated := false

	for {
		l.readChar()
		if l.ch == '"' {
			terminated = true
			break
		}
		if l.ch == 0 {
			// Unterminated string
			break
		}
		if l.ch == '\\' {
			l.readChar() // エスケープ文字をスキップ
		}
	}

	// Store termination status for NextToken to use
	l.stringTerminated = terminated

	return l.input[position:l.position]
}

func (l *Lexer) readChar2() string {
	position := l.position + 1 // 開始の引用符をスキップ
	for {
		l.readChar()
		if l.ch == '\'' || l.ch == 0 {
			break
		}
		if l.ch == '\\' {
			l.readChar() // エスケープ文字をスキップ
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) readComment() string {
	position := l.position
	if l.ch == '/' && l.peekChar() == '/' {
		// シングルラインコメント
		for l.ch != '\n' && l.ch != 0 {
			l.readChar()
		}
	} else if l.ch == '/' && l.peekChar() == '*' {
		// マルチラインコメント
		l.readChar() // '/'をスキップ
		l.readChar() // '*'をスキップ
		for {
			if l.ch == '*' && l.peekChar() == '/' {
				l.readChar() // '*'をスキップ
				l.readChar() // '/'をスキップ
				break
			}
			if l.ch == 0 {
				break
			}
			l.readChar()
		}
	}
	return l.input[position:l.position]
}

// isLetter checks if character is ASCII letter
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z'
}

// isDigit checks if character is ASCII digit
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// isAlphaNumeric checks if character is alphanumeric
func isAlphaNumeric(ch byte) bool {
	return isLetter(ch) || isDigit(ch)
}

// NextToken scans the input and returns the next token with full position information
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	// Store current position for token start
	startPos := l.getCurrentPosition()

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(TokenEq, string(ch)+string(l.ch))
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(TokenFatArrow, string(ch)+string(l.ch))
		} else {
			tok = l.newTokenFromChar(TokenAssign, l.ch)
		}
	case '+':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(TokenPlusAssign, string(ch)+string(l.ch))
		} else {
			tok = l.newTokenFromChar(TokenPlus, l.ch)
		}
	case '-':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(TokenMinusAssign, string(ch)+string(l.ch))
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(TokenArrow, string(ch)+string(l.ch))
		} else {
			tok = l.newTokenFromChar(TokenMinus, l.ch)
		}
	case '*':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(TokenMulAssign, string(ch)+string(l.ch))
		} else if l.peekChar() == '*' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(TokenPower, string(ch)+string(l.ch))
		} else {
			tok = l.newTokenFromChar(TokenMul, l.ch)
		}
	case '/':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(TokenDivAssign, string(ch)+string(l.ch))
		} else if l.peekChar() == '/' || l.peekChar() == '*' {
			commentText := l.readComment()
			tok = l.newTokenFromPosition(TokenComment, commentText, startPos)
			return tok
		} else {
			tok = l.newTokenFromChar(TokenDiv, l.ch)
		}
	case '%':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(TokenModAssign, string(ch)+string(l.ch))
		} else {
			tok = l.newTokenFromChar(TokenMod, l.ch)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(TokenNe, string(ch)+string(l.ch))
		} else {
			// Context-based disambiguation between logical NOT and macro invocation
			nextChar := l.peekChar()

			// Check if we're in a macro context by looking at surrounding tokens
			// If followed immediately by another operator or identifier at end of line, it's macro
			// If followed by space and then identifier/expression, it's NOT
			if nextChar == ' ' || nextChar == '\t' || nextChar == '\n' || nextChar == '\r' {
				tok = l.newTokenFromChar(TokenNot, l.ch)
			} else if isLetter(nextChar) || isDigit(nextChar) {
				// Directly followed by identifier or number - this is NOT
				tok = l.newTokenFromChar(TokenNot, l.ch)
			} else if nextChar == '!' || nextChar == '~' || nextChar == '-' || nextChar == '+' {
				// Followed by another operator - this is NOT (for chains like !!a)
				tok = l.newTokenFromChar(TokenNot, l.ch)
			} else {
				// At end of line or followed by punctuation - likely macro
				tok = l.newTokenFromChar(TokenMacroInvoke, l.ch)
			}
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(TokenLe, string(ch)+string(l.ch))
		} else if l.peekChar() == '<' {
			ch := l.ch
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = l.newToken(TokenShlAssign, "<<"+string(l.ch))
			} else {
				tok = l.newToken(TokenShl, string(ch)+string(l.ch))
			}
		} else {
			tok = l.newTokenFromChar(TokenLt, l.ch)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenGe, Literal: string(ch) + string(l.ch)}
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = Token{Type: TokenShrAssign, Literal: ">>" + string(l.ch)}
			} else {
				tok = Token{Type: TokenShr, Literal: string(ch) + string(l.ch)}
			}
		} else {
			tok = newToken(TokenGt, l.ch)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenAnd, Literal: string(ch) + string(l.ch)}
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenBitAndAssign, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(TokenBitAnd, l.ch)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenOr, Literal: string(ch) + string(l.ch)}
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenBitOrAssign, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(TokenBitOr, l.ch)
		}
	case '^':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenBitXorAssign, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(TokenBitXor, l.ch)
		}
	case '~':
		tok = newToken(TokenBitNot, l.ch)
	case ':':
		if l.peekChar() == ':' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenDoubleColon, Literal: string(ch) + string(l.ch)}
		} else {
			tok = l.newTokenFromChar(TokenColon, l.ch)
		}
	case ';':
		tok = l.newTokenFromChar(TokenSemicolon, l.ch)
	case ',':
		tok = l.newTokenFromChar(TokenComma, l.ch)
	case '.':
		tok = l.newTokenFromChar(TokenDot, l.ch)
	case '(':
		tok = l.newTokenFromChar(TokenLParen, l.ch)
	case ')':
		tok = l.newTokenFromChar(TokenRParen, l.ch)
	case '{':
		tok = l.newTokenFromChar(TokenLBrace, l.ch)
	case '}':
		tok = l.newTokenFromChar(TokenRBrace, l.ch)
	case '[':
		tok = l.newTokenFromChar(TokenLBracket, l.ch)
	case ']':
		tok = l.newTokenFromChar(TokenRBracket, l.ch)
	case '?':
		tok = l.newTokenFromChar(TokenQuestion, l.ch)
	case '@':
		tok = l.newTokenFromChar(TokenAt, l.ch)
	case '#':
		tok = l.newTokenFromChar(TokenHash, l.ch)
	case '$':
		tok = l.newTokenFromChar(TokenDollar, l.ch)
	case '\\':
		tok = l.newTokenFromChar(TokenBackslash, l.ch)
	case '"':
		stringLiteral := l.readString()
		// Check if string was properly terminated
		if !l.stringTerminated {
			// Unterminated string - return error token (error will be created by RecoverableNextToken)
			tok = l.newTokenFromPosition(TokenError, "unterminated string literal", startPos)
		} else {
			tok = l.newTokenFromPosition(TokenString, stringLiteral, startPos)
		}
	case '\'':
		charLiteral := l.readChar2()
		tok = l.newTokenFromPosition(TokenChar, charLiteral, startPos)
	case '\n':
		tok = l.newTokenFromChar(TokenNewline, l.ch)
	case 0:
		tok = l.newToken(TokenEOF, "")
	default:
		if isLetter(l.ch) || l.ch == '_' {
			identLiteral := l.readIdentifier()
			if l.hasIdentifierError {
				// Return error token for identifier with invalid characters
				tok = l.newTokenFromPosition(TokenError, "invalid character in identifier", startPos)
			} else {
				tok = l.newTokenFromPosition(lookupIdent(identLiteral), identLiteral, startPos)
			}
			return tok
		} else if isDigit(l.ch) {
			numberLiteral := l.readNumber()
			// Check if there were errors during number parsing
			if l.errorRecovery != nil && len(l.errors) > 0 {
				// Check if the last error was a malformed number
				if lastError, ok := l.errors[len(l.errors)-1].(*LexicalError); ok && lastError.Type == CategoryMalformedNumber {
					tok = l.newTokenFromPosition(TokenError, numberLiteral, startPos)
					return tok
				}
			}
			tok = l.newTokenFromPosition(TokenInteger, numberLiteral, startPos)
			return tok
		} else if l.ch >= 0x80 { // Unicode文字
			r, size := utf8.DecodeRuneInString(l.input[l.position:])
			if size > 0 && (unicode.IsLetter(r) || unicode.IsSymbol(r)) {
				identLiteral := l.readIdentifier()
				if l.hasIdentifierError {
					// Return error token for identifier with invalid characters
					tok = l.newTokenFromPosition(TokenError, "invalid character in identifier", startPos)
				} else {
					tok = l.newTokenFromPosition(lookupIdent(identLiteral), identLiteral, startPos)
				}
				return tok
			} else {
				tok = l.newTokenFromChar(TokenError, l.ch)
			}
		} else {
			tok = l.newTokenFromChar(TokenError, l.ch)
		}
	}

	l.readChar()
	return tok
}

// newToken creates a new token with current position information
func (l *Lexer) newToken(tokenType TokenType, literal string) Token {
	startPos := Position{Line: l.line, Column: l.column, Offset: l.offset}
	endPos := Position{
		Line:   l.line,
		Column: l.column + len(literal),
		Offset: l.offset + len(literal),
	}

	return Token{
		Type:    tokenType,
		Literal: literal,
		Span: Span{
			Start: startPos,
			End:   endPos,
		},
		// Legacy compatibility
		Line:   l.line,
		Column: l.column,
	}
}

// newTokenFromChar creates a new token from a single character
func (l *Lexer) newTokenFromChar(tokenType TokenType, ch byte) Token {
	return l.newToken(tokenType, string(ch))
}

// newTokenFromPosition creates a token with explicit span information
func (l *Lexer) newTokenFromPosition(tokenType TokenType, literal string, startPos Position) Token {
	endPos := Position{
		Line:   l.line,
		Column: l.column,
		Offset: l.offset,
	}

	return Token{
		Type:    tokenType,
		Literal: literal,
		Span: Span{
			Start: startPos,
			End:   endPos,
		},
		// Legacy compatibility
		Line:   startPos.Line,
		Column: startPos.Column,
	}
}

// getCurrentPosition returns current position in source
func (l *Lexer) getCurrentPosition() Position {
	return Position{Line: l.line, Column: l.column, Offset: l.offset}
}

// Legacy function for backward compatibility - deprecated
func newToken(tokenType TokenType, ch byte) Token {
	return Token{
		Type:    tokenType,
		Literal: string(ch),
		// Note: Span information not available in legacy function
		Line:   0,
		Column: 0,
	}
}

// lookupIdent checks if identifier is keyword
func lookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return TokenIdentifier
}

// Incremental parsing functions

// CanUseCache determines if a cached token can be reused for current position
func (l *Lexer) CanUseCache(cacheEntry *CacheEntry) bool {
	if !l.cacheValid || l.changeRegion == nil {
		return false
	}

	// Check if this token is before the change region
	if cacheEntry.EndPos < l.changeRegion.Start {
		return true
	}

	// Check if this token is after the change region (with offset adjustment)
	if cacheEntry.StartPos > l.changeRegion.End {
		return true
	}

	return false
} // AdjustCachedToken adjusts a cached token's position for text changes
func (l *Lexer) AdjustCachedToken(token Token) Token {
	if !l.cacheValid || l.changeRegion == nil {
		return token
	}

	offsetDiff := l.changeRegion.Length - (l.changeRegion.End - l.changeRegion.Start)

	// Only adjust tokens that come after the change region
	if token.Span.Start.Offset > l.changeRegion.End {
		token.Span.Start.Offset += offsetDiff
		token.Span.End.Offset += offsetDiff

		// Note: Line and column adjustments would require more complex calculation
		// For now, we only adjust byte offsets for simplicity
	}

	return token
}

// InvalidateCache marks the token cache as invalid
func (l *Lexer) InvalidateCache() {
	l.cacheValid = false
	l.tokenCache = nil
	l.changeRegion = nil
}

// UpdateCache updates the token cache with new tokens
func (l *Lexer) UpdateCache(tokens []Token) {
	l.tokenCache = make([]CacheEntry, len(tokens))
	for i, token := range tokens {
		l.tokenCache[i] = CacheEntry{
			Token:    token,
			IsValid:  true,
			StartPos: token.Span.Start.Offset,
			EndPos:   token.Span.End.Offset,
		}
	}
	l.cacheValid = true
}

// GetCachedTokens returns all cached tokens
func (l *Lexer) GetCachedTokens() []CacheEntry {
	return l.tokenCache
}

// SetChangeRegion sets the region that has been modified for incremental parsing
func (l *Lexer) SetChangeRegion(start, end, newLength int) {
	l.changeRegion = &ChangeRegion{
		Start:  start,
		End:    end,
		Length: newLength,
	}
}
