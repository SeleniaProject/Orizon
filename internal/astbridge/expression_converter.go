package astbridge

import (
	"fmt"

	ast "github.com/orizon-lang/orizon/internal/ast"
	p "github.com/orizon-lang/orizon/internal/parser"
)

// ExpressionConverter handles conversion between AST and parser expression types.
// This specialized converter focuses on expression-level transformations,
// ensuring proper handling of all expression constructs in the Orizon language.
type ExpressionConverter struct {
	// typeConverter handles type-specific conversions within expressions
	typeConverter *TypeConverter
}

// NewExpressionConverter creates a new expression converter with type converter dependency.
// This constructor ensures proper initialization and maintains conversion consistency.
func NewExpressionConverter(typeConverter *TypeConverter) *ExpressionConverter {
	return &ExpressionConverter{
		typeConverter: typeConverter,
	}
}

// FromParserExpression converts a parser.Expression to ast.Expression.
// This method provides comprehensive expression conversion with proper error handling
// for all supported expression types in the Orizon language.
func (ec *ExpressionConverter) FromParserExpression(expr p.Expression) (ast.Expression, error) {
	if expr == nil {
		return nil, fmt.Errorf("cannot convert nil parser expression")
	}

	switch concrete := expr.(type) {
	case *p.Identifier:
		return ec.fromParserIdentifier(concrete)
	case *p.Literal:
		return ec.fromParserLiteral(concrete)
	case *p.BinaryExpression:
		return ec.fromParserBinaryExpression(concrete)
	case *p.UnaryExpression:
		return ec.fromParserUnaryExpression(concrete)
	case *p.CallExpression:
		return ec.fromParserCallExpression(concrete)
	case *p.MemberExpression:
		return ec.fromParserMemberExpression(concrete)
	default:
		return nil, fmt.Errorf("unsupported parser expression type: %T", expr)
	}
}

// ToParserExpression converts an ast.Expression to parser.Expression.
// This method provides the inverse conversion with comprehensive error handling
// and maintains symmetry with FromParserExpression for bidirectional compatibility.
func (ec *ExpressionConverter) ToParserExpression(expr ast.Expression) (p.Expression, error) {
	if expr == nil {
		return nil, fmt.Errorf("cannot convert nil AST expression")
	}

	switch concrete := expr.(type) {
	case *ast.Identifier:
		return ec.toParserIdentifier(concrete)
	case *ast.Literal:
		return ec.toParserLiteral(concrete)
	case *ast.BinaryExpression:
		return ec.toParserBinaryExpression(concrete)
	case *ast.UnaryExpression:
		return ec.toParserUnaryExpression(concrete)
	case *ast.CallExpression:
		return ec.toParserCallExpression(concrete)
	case *ast.MemberExpression:
		return ec.toParserMemberExpression(concrete)
	default:
		return nil, fmt.Errorf("unsupported AST expression type: %T", expr)
	}
}

// fromParserIdentifier converts parser Identifier to AST Identifier.
// This method handles identifier conversion with proper span preservation.
func (ec *ExpressionConverter) fromParserIdentifier(ident *p.Identifier) (*ast.Identifier, error) {
	if ident == nil {
		return nil, fmt.Errorf("cannot convert nil parser identifier")
	}

	return &ast.Identifier{
		Span:  fromParserSpan(ident.Span),
		Value: ident.Value,
	}, nil
}

// toParserIdentifier converts AST Identifier to parser Identifier.
// This method provides the inverse conversion with proper span handling.
func (ec *ExpressionConverter) toParserIdentifier(ident *ast.Identifier) (*p.Identifier, error) {
	if ident == nil {
		return nil, fmt.Errorf("cannot convert nil AST identifier")
	}

	return &p.Identifier{
		Span:  toParserSpan(ident.Span),
		Value: ident.Value,
	}, nil
}

// fromParserLiteral converts parser Literal to AST Literal.
// This method handles all literal types with proper value and kind mapping.
func (ec *ExpressionConverter) fromParserLiteral(literal *p.Literal) (*ast.Literal, error) {
	if literal == nil {
		return nil, fmt.Errorf("cannot convert nil parser literal")
	}

	// Map parser literal kind to AST literal kind
	astKind, err := ec.mapParserLiteralKind(literal.Kind)
	if err != nil {
		return nil, fmt.Errorf("failed to map literal kind: %w", err)
	}

	// Use string representation as raw value since parser doesn't store raw text
	rawValue := fmt.Sprintf("%v", literal.Value)

	return &ast.Literal{
		Span:  fromParserSpan(literal.Span),
		Kind:  astKind,
		Value: literal.Value,
		Raw:   rawValue,
	}, nil
}

// toParserLiteral converts AST Literal to parser Literal.
// This method provides the inverse conversion with proper kind mapping.
func (ec *ExpressionConverter) toParserLiteral(literal *ast.Literal) (*p.Literal, error) {
	if literal == nil {
		return nil, fmt.Errorf("cannot convert nil AST literal")
	}

	// Map AST literal kind to parser literal kind
	parserKind, err := ec.mapASTLiteralKind(literal.Kind)
	if err != nil {
		return nil, fmt.Errorf("failed to map literal kind: %w", err)
	}

	return &p.Literal{
		Span:  toParserSpan(literal.Span),
		Kind:  parserKind,
		Value: literal.Value,
	}, nil
}

// mapParserLiteralKind maps parser LiteralKind to AST LiteralKind.
// This method centralizes literal kind mapping for consistency.
func (ec *ExpressionConverter) mapParserLiteralKind(kind p.LiteralKind) (ast.LiteralKind, error) {
	switch kind {
	case p.LiteralInteger:
		return ast.LiteralInteger, nil
	case p.LiteralFloat:
		return ast.LiteralFloat, nil
	case p.LiteralString:
		return ast.LiteralString, nil
	case p.LiteralBool:
		return ast.LiteralBoolean, nil
	case p.LiteralNull:
		return ast.LiteralNull, nil
	default:
		return 0, fmt.Errorf("unsupported parser literal kind: %v", kind)
	}
}

// mapASTLiteralKind maps AST LiteralKind to parser LiteralKind.
// This method provides the inverse mapping for bidirectional conversion.
func (ec *ExpressionConverter) mapASTLiteralKind(kind ast.LiteralKind) (p.LiteralKind, error) {
	switch kind {
	case ast.LiteralInteger:
		return p.LiteralInteger, nil
	case ast.LiteralFloat:
		return p.LiteralFloat, nil
	case ast.LiteralString:
		return p.LiteralString, nil
	case ast.LiteralBoolean:
		return p.LiteralBool, nil
	case ast.LiteralCharacter:
		// Parser doesn't have separate char literals, map to string
		return p.LiteralString, nil
	case ast.LiteralNull:
		return p.LiteralNull, nil
	default:
		return 0, fmt.Errorf("unsupported AST literal kind: %v", kind)
	}
}

// Stub implementations for complex expressions (to be expanded)

func (ec *ExpressionConverter) fromParserBinaryExpression(expr *p.BinaryExpression) (*ast.BinaryExpression, error) {
	return nil, fmt.Errorf("binary expression conversion not yet implemented")
}

func (ec *ExpressionConverter) toParserBinaryExpression(expr *ast.BinaryExpression) (*p.BinaryExpression, error) {
	return nil, fmt.Errorf("binary expression conversion not yet implemented")
}

func (ec *ExpressionConverter) fromParserUnaryExpression(expr *p.UnaryExpression) (*ast.UnaryExpression, error) {
	return nil, fmt.Errorf("unary expression conversion not yet implemented")
}

func (ec *ExpressionConverter) toParserUnaryExpression(expr *ast.UnaryExpression) (*p.UnaryExpression, error) {
	return nil, fmt.Errorf("unary expression conversion not yet implemented")
}

func (ec *ExpressionConverter) fromParserCallExpression(expr *p.CallExpression) (*ast.CallExpression, error) {
	return nil, fmt.Errorf("call expression conversion not yet implemented")
}

func (ec *ExpressionConverter) toParserCallExpression(expr *ast.CallExpression) (*p.CallExpression, error) {
	return nil, fmt.Errorf("call expression conversion not yet implemented")
}

func (ec *ExpressionConverter) fromParserMemberExpression(expr *p.MemberExpression) (*ast.MemberExpression, error) {
	return nil, fmt.Errorf("member expression conversion not yet implemented")
}

func (ec *ExpressionConverter) toParserMemberExpression(expr *ast.MemberExpression) (*p.MemberExpression, error) {
	return nil, fmt.Errorf("member expression conversion not yet implemented")
}
