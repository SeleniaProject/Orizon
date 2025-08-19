package astbridge

import (
	"fmt"
	"strings"

	ast "github.com/orizon-lang/orizon/internal/ast"
	p "github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/position"
)

// TypeConverter handles conversion between AST and parser type representations.
// This specialized converter ensures type safety and consistency across the entire
// compilation pipeline while maintaining clear separation of type-related concerns.
type TypeConverter struct{}

// NewTypeConverter creates a new type converter instance.
// This constructor provides a clean initialization point for type conversion functionality.
func NewTypeConverter() *TypeConverter {
	return &TypeConverter{}
}

// FromParserType converts a parser.Type to ast.Type.
// This method provides comprehensive type conversion with proper error handling
// for all supported type constructs in the Orizon language.
func (tc *TypeConverter) FromParserType(parserType p.Type) (ast.Type, error) {
	if parserType == nil {
		return nil, fmt.Errorf("cannot convert nil parser type")
	}

	switch concrete := parserType.(type) {
	case *p.BasicType:
		return tc.fromParserBasicType(concrete)
	case *p.TupleType:
		return tc.fromParserTupleType(concrete)
	default:
		return nil, fmt.Errorf("unsupported parser type: %T", parserType)
	}
}

// ToParserType converts an ast.Type to parser.Type.
// This method provides the inverse conversion with comprehensive error handling
// and maintains symmetry with FromParserType for bidirectional compatibility.
func (tc *TypeConverter) ToParserType(astType ast.Type) (p.Type, error) {
	if astType == nil {
		return nil, fmt.Errorf("cannot convert nil AST type")
	}

	switch concrete := astType.(type) {
	case *ast.BasicType:
		return tc.toParserBasicType(concrete)
	case *ast.IdentifierType:
		return tc.toParserIdentifierType(concrete)
	default:
		return nil, fmt.Errorf("unsupported AST type: %T", astType)
	}
}

// PrettyPrintParserType renders parser types with comprehensive formatting.
// This utility method provides detailed type representation for debugging and error messages.
func (tc *TypeConverter) PrettyPrintParserType(parserType p.Type) string {
	if parserType == nil {
		return "<nil-type>"
	}

	switch concrete := parserType.(type) {
	case *p.BasicType:
		return concrete.Name
	case *p.TupleType:
		return tc.formatTupleType(concrete)
	default:
		if stringer, ok := parserType.(fmt.Stringer); ok {
			return stringer.String()
		}
		return "<unknown-type>"
	}
}

// formatTupleType formats tuple types with proper element representation.
// This method handles both unit types and multi-element tuples.
func (tc *TypeConverter) formatTupleType(tupleType *p.TupleType) string {
	if len(tupleType.Elements) == 0 {
		return "()"
	}

	elements := make([]string, 0, len(tupleType.Elements))
	for _, element := range tupleType.Elements {
		elements = append(elements, tc.PrettyPrintParserType(element))
	}

	return fmt.Sprintf("(%s)", strings.Join(elements, ", "))
}

// Helper methods for specific type conversions

// fromParserBasicType converts parser BasicType to AST BasicType.
// This method maps between the different basic type representations and ensures
// proper handling of primitive type semantics.
func (tc *TypeConverter) fromParserBasicType(basicType *p.BasicType) (*ast.BasicType, error) {
	if basicType == nil {
		return nil, fmt.Errorf("cannot convert nil parser basic type")
	}

	// Map parser basic type names to AST basic kinds
	kind := tc.mapBasicTypeName(basicType.Name)

	return &ast.BasicType{
		Span: fromParserSpan(basicType.Span),
		Kind: kind,
	}, nil
}

// toParserBasicType converts AST BasicType to parser BasicType.
// This method provides the inverse conversion with proper kind mapping.
func (tc *TypeConverter) toParserBasicType(basicType *ast.BasicType) (*p.BasicType, error) {
	if basicType == nil {
		return nil, fmt.Errorf("cannot convert nil AST basic type")
	}

	// Map AST basic kind to parser type name
	name := tc.mapBasicTypeKind(basicType.Kind)

	return &p.BasicType{
		Span: toParserSpan(basicType.Span),
		Name: name,
	}, nil
}

// toParserIdentifierType converts AST IdentifierType to parser BasicType.
// Since parser doesn't have explicit IdentifierType, we represent it as BasicType.
func (tc *TypeConverter) toParserIdentifierType(identType *ast.IdentifierType) (*p.BasicType, error) {
	if identType == nil {
		return nil, fmt.Errorf("cannot convert nil AST identifier type")
	}

	return &p.BasicType{
		Span: toParserSpan(identType.Span),
		Name: identType.Name.Value,
	}, nil
}

// fromParserTupleType converts parser TupleType to appropriate AST type.
// This method handles both unit types and complex tuple structures.
func (tc *TypeConverter) fromParserTupleType(tupleType *p.TupleType) (ast.Type, error) {
	if tupleType == nil {
		return nil, fmt.Errorf("cannot convert nil parser tuple type")
	}

	// Unit type () maps to void
	if len(tupleType.Elements) == 0 {
		return &ast.BasicType{
			Span: fromParserSpan(tupleType.Span),
			Kind: ast.BasicVoid,
		}, nil
	}

	// For now, map tuple types to identifier types with tuple representation
	// This is a simplified approach - full tuple support would require extending AST
	return &ast.IdentifierType{
		Span: fromParserSpan(tupleType.Span),
		Name: &ast.Identifier{
			Span:  fromParserSpan(tupleType.Span),
			Value: tupleType.String(),
		},
	}, nil
}

// mapBasicTypeName maps parser basic type names to AST basic kinds.
// This centralized mapping ensures consistency across type conversions.
func (tc *TypeConverter) mapBasicTypeName(name string) ast.BasicKind {
	switch name {
	case "int", "i32", "i64":
		return ast.BasicInt
	case "float", "f32", "f64":
		return ast.BasicFloat
	case "string", "str":
		return ast.BasicString
	case "bool", "boolean":
		return ast.BasicBool
	case "char", "character":
		return ast.BasicChar
	case "void", "unit", "()":
		return ast.BasicVoid
	default:
		// Fallback for unknown types
		return ast.BasicVoid
	}
}

// mapBasicTypeKind maps AST basic kinds to parser type names.
// This provides the inverse mapping for bidirectional conversion.
func (tc *TypeConverter) mapBasicTypeKind(kind ast.BasicKind) string {
	switch kind {
	case ast.BasicInt:
		return "int"
	case ast.BasicFloat:
		return "float"
	case ast.BasicString:
		return "string"
	case ast.BasicBool:
		return "bool"
	case ast.BasicChar:
		return "char"
	case ast.BasicVoid:
		return "void"
	default:
		return "unknown"
	}
}

// Helper functions for span conversion

// fromParserSpan converts parser.Span to position.Span.
// This function ensures proper position tracking across conversion boundaries.
func fromParserSpan(parserSpan p.Span) position.Span {
	return position.Span{
		Start: position.Position{
			Filename: parserSpan.Start.File,
			Line:     parserSpan.Start.Line,
			Column:   parserSpan.Start.Column,
			Offset:   parserSpan.Start.Offset,
		},
		End: position.Position{
			Filename: parserSpan.End.File,
			Line:     parserSpan.End.Line,
			Column:   parserSpan.End.Column,
			Offset:   parserSpan.End.Offset,
		},
	}
}

// toParserSpan converts position.Span to parser.Span.
// This function provides the inverse conversion for bidirectional compatibility.
func toParserSpan(posSpan position.Span) p.Span {
	return p.Span{
		Start: p.Position{
			File:   posSpan.Start.Filename,
			Line:   posSpan.Start.Line,
			Column: posSpan.Start.Column,
			Offset: posSpan.Start.Offset,
		},
		End: p.Position{
			File:   posSpan.End.Filename,
			Line:   posSpan.End.Line,
			Column: posSpan.End.Column,
			Offset: posSpan.End.Offset,
		},
	}
}
