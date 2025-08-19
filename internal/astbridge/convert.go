package astbridge

import (
	"fmt"

	ast "github.com/orizon-lang/orizon/internal/ast"
	p "github.com/orizon-lang/orizon/internal/parser"
)

// ASTBridge provides the main interface for converting between AST and parser representations.
// This facade pattern simplifies the complex conversion operations by providing a unified
// interface while maintaining the internal modular architecture for maintainability.
//
// The bridge ensures bidirectional conversion capabilities with proper error handling
// and maintains consistency across the entire Orizon compilation pipeline.
type ASTBridge struct {
	// declarationConverter handles all declaration-level conversions
	declarationConverter *DeclarationConverter
	// typeConverter handles all type-level conversions
	typeConverter *TypeConverter
	// exprConverter handles all expression-level conversions
	exprConverter *ExpressionConverter
	// stmtConverter handles all statement-level conversions
	stmtConverter *StatementConverter
}

// NewASTBridge creates a new AST bridge with all necessary converters properly initialized.
// This constructor ensures consistent initialization and maintains proper dependency relationships
// between the various conversion components.
func NewASTBridge() *ASTBridge {
	declarationConverter := NewDeclarationConverter()

	return &ASTBridge{
		declarationConverter: declarationConverter,
		typeConverter:        declarationConverter.typeConverter,
		exprConverter:        declarationConverter.exprConverter,
		stmtConverter:        declarationConverter.stmtConverter,
	}
}

// FromParserProgram converts a parser.Program to ast.Program.
// This method provides the primary entry point for converting parser ASTs to the unified
// AST representation used throughout the Orizon compilation pipeline.
//
// The conversion process:
// 1. Validates input program structure
// 2. Converts each declaration using specialized converters
// 3. Maintains source position information for debugging
// 4. Handles conversion errors gracefully with detailed error messages
func FromParserProgram(src *p.Program) (*ast.Program, error) {
	if src == nil {
		return nil, fmt.Errorf("cannot convert nil parser program")
	}

	bridge := NewASTBridge()
	return bridge.fromParserProgram(src)
}

// ToParserProgram converts an ast.Program to parser.Program.
// This method provides the inverse conversion for cases where parser representation
// is required (e.g., for certain optimization passes or serialization).
//
// The conversion process ensures that all AST nodes are properly mapped back to
// their parser equivalents while preserving semantic information and structure.
func ToParserProgram(src *ast.Program) (*p.Program, error) {
	if src == nil {
		return nil, fmt.Errorf("cannot convert nil AST program")
	}

	bridge := NewASTBridge()
	return bridge.toParserProgram(src)
}

// fromParserProgram performs the actual parser to AST program conversion.
// This internal method implements the core conversion logic with proper error handling.
func (ab *ASTBridge) fromParserProgram(src *p.Program) (*ast.Program, error) {
	// Create the target AST program with proper capacity allocation
	dst := &ast.Program{
		Span:         fromParserSpan(src.Span),
		Declarations: make([]ast.Declaration, 0, len(src.Declarations)),
		Comments:     nil, // Comments handled separately in advanced implementation
	}

	// Convert each declaration with comprehensive error handling
	for i, decl := range src.Declarations {
		convertedDecl, err := ab.declarationConverter.FromParserDeclaration(decl)
		if err != nil {
			return nil, fmt.Errorf("failed to convert declaration %d (%T): %w", i, decl, err)
		}

		// Only add non-nil declarations (some parser constructs may be compile-time only)
		if convertedDecl != nil {
			dst.Declarations = append(dst.Declarations, convertedDecl)
		}
	}

	return dst, nil
}

// toParserProgram performs the actual AST to parser program conversion.
// This internal method implements the inverse conversion logic with proper error handling.
func (ab *ASTBridge) toParserProgram(src *ast.Program) (*p.Program, error) {
	// Create the target parser program with proper capacity allocation
	dst := &p.Program{
		Span:         toParserSpan(src.Span),
		Declarations: make([]p.Declaration, 0, len(src.Declarations)),
	}

	// Convert each declaration with comprehensive error handling
	for i, decl := range src.Declarations {
		convertedDecl, err := ab.declarationConverter.ToParserDeclaration(decl)
		if err != nil {
			return nil, fmt.Errorf("failed to convert declaration %d (%T): %w", i, decl, err)
		}

		// Only add non-nil declarations
		if convertedDecl != nil {
			dst.Declarations = append(dst.Declarations, convertedDecl)
		}
	}

	return dst, nil
}

// Legacy function compatibility for existing codebase.
// These functions maintain backward compatibility while the codebase transitions
// to the new modular architecture.

// prettyPrintParserType provides a compatibility wrapper for the type converter's
// pretty printing functionality. This maintains the existing API while delegating
// to the specialized type converter implementation.
func prettyPrintParserType(t p.Type) string {
	converter := NewTypeConverter()
	return converter.PrettyPrintParserType(t)
}

// Error types for more precise error handling

// ConversionError represents errors that occur during AST conversion.
// This specialized error type provides additional context for debugging
// conversion issues and enables better error reporting in the compiler.
type ConversionError struct {
	// SourceType indicates the type of the source node that failed conversion
	SourceType string
	// TargetType indicates the expected target type for conversion
	TargetType string
	// Operation indicates the conversion operation that failed
	Operation string
	// Underlying contains the original error that caused the conversion failure
	Underlying error
}

// Error implements the error interface for ConversionError.
// This method provides comprehensive error messages for debugging conversion issues.
func (ce *ConversionError) Error() string {
	return fmt.Sprintf("conversion error: failed to convert %s to %s during %s operation: %v",
		ce.SourceType, ce.TargetType, ce.Operation, ce.Underlying)
}

// Unwrap provides access to the underlying error for error chain inspection.
// This enables advanced error handling and debugging in calling code.
func (ce *ConversionError) Unwrap() error {
	return ce.Underlying
}

// Utility functions for consistent error creation

// NewConversionError creates a new ConversionError with the specified details.
// This factory function ensures consistent error message formatting and structure.
func NewConversionError(sourceType, targetType, operation string, underlying error) *ConversionError {
	return &ConversionError{
		SourceType: sourceType,
		TargetType: targetType,
		Operation:  operation,
		Underlying: underlying,
	}
}
