package format

import (
	"strings"

	"github.com/orizon-lang/orizon/internal/ast"
	"github.com/orizon-lang/orizon/internal/lexer"
)

// ASTFormattingOptions controls AST-based formatting
type ASTFormattingOptions struct {
	// IndentSize specifies the number of spaces for indentation
	IndentSize int
	// PreferTabs uses tabs instead of spaces for indentation
	PreferTabs bool
	// MaxLineLength is the preferred maximum line length
	MaxLineLength int
	// AlignFields aligns struct/enum fields vertically
	AlignFields bool
	// SpaceAroundOperators adds spaces around binary operators
	SpaceAroundOperators bool
	// TrailingComma adds trailing commas in lists
	TrailingComma bool
	// EmptyLineBetweenDeclarations adds empty lines between top-level declarations
	EmptyLineBetweenDeclarations bool
}

// DefaultASTFormattingOptions returns default AST formatting options
func DefaultASTFormattingOptions() ASTFormattingOptions {
	return ASTFormattingOptions{
		IndentSize:                   4,
		PreferTabs:                   false,
		MaxLineLength:                100,
		AlignFields:                  true,
		SpaceAroundOperators:         true,
		TrailingComma:                true,
		EmptyLineBetweenDeclarations: true,
	}
}

// ASTFormatter provides AST-based code formatting
type ASTFormatter struct {
	options ASTFormattingOptions
	indent  int
	buffer  strings.Builder
}

// NewASTFormatter creates a new AST formatter with the given options
func NewASTFormatter(options ASTFormattingOptions) *ASTFormatter {
	return &ASTFormatter{
		options: options,
		indent:  0,
	}
}

// FormatAST formats an AST and returns the formatted source code
func (f *ASTFormatter) FormatAST(node ast.Node) string {
	f.buffer.Reset()
	f.indent = 0

	if node != nil {
		f.formatNode(node)
	}

	return f.buffer.String()
}

// FormatSourceWithAST parses source code and formats it using AST
func FormatSourceWithAST(source string, options ASTFormattingOptions) (string, error) {
	// For now, return a notice that AST formatting is not fully implemented
	// This would be implemented once the parser interface is stabilized

	// Try basic token-based formatting as a fallback
	formatted := formatBasedOnTokens(source, options)
	return formatted, nil
}

// formatBasedOnTokens provides improved formatting using lexical analysis
func formatBasedOnTokens(source string, options ASTFormattingOptions) string {
	// Create lexer
	l := lexer.New(source)

	var result strings.Builder
	currentIndent := 0
	needNewline := false
	lastTokenWasKeyword := false

	for {
		token := l.NextToken()
		if token.Type == lexer.TokenEOF {
			break
		}

		// Handle indentation for specific tokens
		switch token.Type.String() {
		case "LBRACE", "{":
			if needNewline {
				result.WriteString("\n")
				writeIndentString(&result, currentIndent, options)
				needNewline = false
			}
			if options.SpaceAroundOperators && result.Len() > 0 {
				lastChar := result.String()[result.Len()-1]
				if lastChar != ' ' && lastChar != '\n' && lastChar != '\t' {
					result.WriteString(" ")
				}
			}
			result.WriteString("{")
			currentIndent++
			needNewline = true

		case "RBRACE", "}":
			currentIndent--
			if currentIndent < 0 {
				currentIndent = 0
			}
			result.WriteString("\n")
			writeIndentString(&result, currentIndent, options)
			result.WriteString("}")
			needNewline = true

		case "SEMICOLON", ";":
			result.WriteString(";")
			needNewline = true

		case "COMMA", ",":
			result.WriteString(",")
			if options.SpaceAroundOperators {
				result.WriteString(" ")
			}

		case "COLON", ":":
			result.WriteString(":")
			if options.SpaceAroundOperators {
				result.WriteString(" ")
			}

		case "ASSIGN", "=":
			if options.SpaceAroundOperators {
				result.WriteString(" = ")
			} else {
				result.WriteString("=")
			}

		case "PLUS", "+", "MINUS", "-", "MULTIPLY", "*", "DIVIDE", "/":
			if options.SpaceAroundOperators {
				result.WriteString(" " + token.Literal + " ")
			} else {
				result.WriteString(token.Literal)
			}

		default:
			// Handle newlines and indentation
			if needNewline {
				result.WriteString("\n")
				writeIndentString(&result, currentIndent, options)
				needNewline = false
			}

			// Add space before keywords if needed
			if isKeyword(token.Literal) && !lastTokenWasKeyword && result.Len() > 0 {
				lastChar := result.String()[result.Len()-1]
				if lastChar != ' ' && lastChar != '\n' && lastChar != '\t' {
					result.WriteString(" ")
				}
			}

			result.WriteString(token.Literal)
			lastTokenWasKeyword = isKeyword(token.Literal)
		}
	}

	// Ensure single trailing newline
	formatted := result.String()
	formatted = strings.TrimRight(formatted, " \t\r\n") + "\n"

	return formatted
}

// writeIndentString writes indentation to the buffer
func writeIndentString(buffer *strings.Builder, level int, options ASTFormattingOptions) {
	if options.PreferTabs {
		buffer.WriteString(strings.Repeat("\t", level))
	} else {
		buffer.WriteString(strings.Repeat(" ", level*options.IndentSize))
	}
}

// isKeyword checks if a token is a keyword
func isKeyword(literal string) bool {
	keywords := map[string]bool{
		"fn":       true,
		"struct":   true,
		"enum":     true,
		"trait":    true,
		"impl":     true,
		"let":      true,
		"var":      true,
		"const":    true,
		"if":       true,
		"else":     true,
		"while":    true,
		"for":      true,
		"match":    true,
		"return":   true,
		"break":    true,
		"continue": true,
		"pub":      true,
		"import":   true,
		"export":   true,
	}

	return keywords[literal]
}

// formatNode formats a specific AST node (placeholder for future implementation)
func (f *ASTFormatter) formatNode(node ast.Node) {
	// This would be implemented once AST structure is stabilized
	// For now, use string representation
	f.writeString(node.String())
}

// Writing helper functions
func (f *ASTFormatter) writeString(s string) {
	f.buffer.WriteString(s)
}

func (f *ASTFormatter) writeNewline() {
	f.buffer.WriteString("\n")
}

func (f *ASTFormatter) writeIndent() {
	if f.options.PreferTabs {
		f.buffer.WriteString(strings.Repeat("\t", f.indent))
	} else {
		f.buffer.WriteString(strings.Repeat(" ", f.indent*f.options.IndentSize))
	}
}
