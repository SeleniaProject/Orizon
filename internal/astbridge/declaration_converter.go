package astbridge

import (
	"fmt"

	ast "github.com/orizon-lang/orizon/internal/ast"
	p "github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/position"
)

// DeclarationConverter handles conversion between AST and parser declaration types.
// This specialized converter focuses solely on declaration-level transformations,
// ensuring separation of concerns and maintainable code organization.
type DeclarationConverter struct {
	// typeConverter handles type-specific conversions
	typeConverter *TypeConverter
	// exprConverter handles expression-specific conversions
	exprConverter *ExpressionConverter
	// stmtConverter handles statement-specific conversions
	stmtConverter *StatementConverter
}

// NewDeclarationConverter creates a new declaration converter with all necessary sub-converters.
// This constructor ensures proper initialization of all dependencies and maintains
// consistent conversion behavior across the entire AST bridge.
func NewDeclarationConverter() *DeclarationConverter {
	typeConverter := NewTypeConverter()
	exprConverter := NewExpressionConverter(typeConverter)
	stmtConverter := NewStatementConverter(typeConverter, exprConverter)

	return &DeclarationConverter{
		typeConverter: typeConverter,
		exprConverter: exprConverter,
		stmtConverter: stmtConverter,
	}
}

// FromParserDeclaration converts a parser.Declaration to ast.Declaration.
// This method provides comprehensive declaration conversion with proper error handling
// and type safety. It delegates to specific conversion methods based on the concrete
// declaration type, ensuring extensibility and maintainability.
func (dc *DeclarationConverter) FromParserDeclaration(decl p.Declaration) (ast.Declaration, error) {
	if decl == nil {
		return nil, fmt.Errorf("cannot convert nil parser declaration")
	}

	switch concrete := decl.(type) {
	case *p.FunctionDeclaration:
		return dc.fromParserFunction(concrete)
	case *p.VariableDeclaration:
		return dc.fromParserVariable(concrete)
	case *p.TypeAliasDeclaration:
		return dc.fromParserTypeAlias(concrete)
	case *p.NewtypeDeclaration:
		return dc.fromParserNewtype(concrete)
	case *p.StructDeclaration:
		return dc.fromParserStruct(concrete)
	case *p.EnumDeclaration:
		return dc.fromParserEnum(concrete)
	case *p.TraitDeclaration:
		return dc.fromParserTrait(concrete)
	case *p.ImplBlock:
		return dc.fromParserImplBlock(concrete)
	case *p.ImportDeclaration:
		return dc.fromParserImport(concrete)
	case *p.ExportDeclaration:
		return dc.fromParserExport(concrete)
	case *p.MacroDefinition:
		// Macros are compile-time only constructs and are not represented
		// in the runtime AST. They are processed during the preprocessing phase.
		return nil, nil
	case *p.ExpressionStatement:
		// Expression statements at the top level are treated as declarations
		// in certain contexts. We delegate to statement conversion and wrap if needed.
		stmt, err := dc.stmtConverter.FromParserStatement(concrete)
		if err != nil {
			return nil, fmt.Errorf("failed to convert expression statement: %w", err)
		}
		if astDecl, ok := stmt.(ast.Declaration); ok {
			return astDecl, nil
		}
		// Fallback: wrap as a placeholder type declaration for compatibility
		return dc.createPlaceholderDeclaration(), nil
	default:
		return nil, fmt.Errorf("unsupported parser declaration type: %T", decl)
	}
}

// ToParserDeclaration converts an ast.Declaration to parser.Declaration.
// This method provides the inverse conversion with comprehensive error handling
// and maintains symmetry with FromParserDeclaration for bidirectional compatibility.
func (dc *DeclarationConverter) ToParserDeclaration(decl ast.Declaration) (p.Declaration, error) {
	if decl == nil {
		return nil, fmt.Errorf("cannot convert nil AST declaration")
	}

	switch concrete := decl.(type) {
	case *ast.FunctionDeclaration:
		return dc.toParserFunction(concrete)
	case *ast.VariableDeclaration:
		return dc.toParserVariable(concrete)
	case *ast.TypeDeclaration:
		return dc.toParserTypeDeclaration(concrete)
	case *ast.StructDeclaration:
		return dc.toParserStruct(concrete)
	case *ast.EnumDeclaration:
		return dc.toParserEnum(concrete)
	case *ast.TraitDeclaration:
		return dc.toParserTrait(concrete)
	case *ast.ImplDeclaration:
		return dc.toParserImplBlock(concrete)
	case *ast.ImportDeclaration:
		return dc.toParserImport(concrete)
	case *ast.ExportDeclaration:
		return dc.toParserExport(concrete)
	default:
		return nil, fmt.Errorf("unsupported AST declaration type: %T", decl)
	}
}

// createPlaceholderDeclaration creates a placeholder type declaration for edge cases.
// This utility method provides a safe fallback when declaration conversion cannot
// be performed directly, maintaining AST consistency while preserving error information.
func (dc *DeclarationConverter) createPlaceholderDeclaration() *ast.TypeDeclaration {
	return &ast.TypeDeclaration{
		Span:    createEmptySpan(),
		Name:    &ast.Identifier{Span: createEmptySpan(), Value: "_placeholder"},
		Type:    &ast.BasicType{Kind: ast.BasicVoid},
		IsAlias: false,
	}
}

// createEmptySpan creates an empty position span for placeholder nodes.
// This helper method ensures consistent handling of position information in fallback cases.
func createEmptySpan() position.Span {
	return position.Span{
		Start: position.Position{},
		End:   position.Position{},
	}
}

// Stub implementations for specific declaration types (to be expanded)

func (dc *DeclarationConverter) fromParserFunction(fn *p.FunctionDeclaration) (*ast.FunctionDeclaration, error) {
	return nil, fmt.Errorf("function declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) toParserFunction(fn *ast.FunctionDeclaration) (*p.FunctionDeclaration, error) {
	return nil, fmt.Errorf("function declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) fromParserVariable(variable *p.VariableDeclaration) (*ast.VariableDeclaration, error) {
	return nil, fmt.Errorf("variable declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) toParserVariable(variable *ast.VariableDeclaration) (*p.VariableDeclaration, error) {
	return nil, fmt.Errorf("variable declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) fromParserTypeAlias(alias *p.TypeAliasDeclaration) (*ast.TypeDeclaration, error) {
	return nil, fmt.Errorf("type alias declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) fromParserNewtype(newtype *p.NewtypeDeclaration) (*ast.TypeDeclaration, error) {
	return nil, fmt.Errorf("newtype declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) toParserTypeDeclaration(typeDecl *ast.TypeDeclaration) (p.Declaration, error) {
	return nil, fmt.Errorf("type declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) fromParserStruct(structDecl *p.StructDeclaration) (*ast.StructDeclaration, error) {
	return nil, fmt.Errorf("struct declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) toParserStruct(structDecl *ast.StructDeclaration) (*p.StructDeclaration, error) {
	return nil, fmt.Errorf("struct declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) fromParserEnum(enumDecl *p.EnumDeclaration) (*ast.EnumDeclaration, error) {
	return nil, fmt.Errorf("enum declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) toParserEnum(enumDecl *ast.EnumDeclaration) (*p.EnumDeclaration, error) {
	return nil, fmt.Errorf("enum declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) fromParserTrait(traitDecl *p.TraitDeclaration) (*ast.TraitDeclaration, error) {
	return nil, fmt.Errorf("trait declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) toParserTrait(traitDecl *ast.TraitDeclaration) (*p.TraitDeclaration, error) {
	return nil, fmt.Errorf("trait declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) fromParserImplBlock(implBlock *p.ImplBlock) (*ast.ImplDeclaration, error) {
	return nil, fmt.Errorf("impl block conversion not yet fully implemented")
}

func (dc *DeclarationConverter) toParserImplBlock(implDecl *ast.ImplDeclaration) (*p.ImplBlock, error) {
	return nil, fmt.Errorf("impl declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) fromParserImport(importDecl *p.ImportDeclaration) (*ast.ImportDeclaration, error) {
	return nil, fmt.Errorf("import declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) toParserImport(importDecl *ast.ImportDeclaration) (*p.ImportDeclaration, error) {
	return nil, fmt.Errorf("import declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) fromParserExport(exportDecl *p.ExportDeclaration) (*ast.ExportDeclaration, error) {
	return nil, fmt.Errorf("export declaration conversion not yet fully implemented")
}

func (dc *DeclarationConverter) toParserExport(exportDecl *ast.ExportDeclaration) (*p.ExportDeclaration, error) {
	return nil, fmt.Errorf("export declaration conversion not yet fully implemented")
}
