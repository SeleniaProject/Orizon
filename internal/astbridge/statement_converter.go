package astbridge

import (
	"fmt"

	ast "github.com/orizon-lang/orizon/internal/ast"
	p "github.com/orizon-lang/orizon/internal/parser"
)

// StatementConverter handles conversion between AST and parser statement types.
// This specialized converter focuses on statement-level transformations,.
// ensuring proper handling of all statement constructs in the Orizon language.
type StatementConverter struct {
	// typeConverter handles type-specific conversions within statements.
	typeConverter *TypeConverter
	// exprConverter handles expression-specific conversions within statements.
	exprConverter *ExpressionConverter
}

// NewStatementConverter creates a new statement converter with required dependencies.
// This constructor ensures proper initialization and maintains conversion consistency.
func NewStatementConverter(typeConverter *TypeConverter, exprConverter *ExpressionConverter) *StatementConverter {
	return &StatementConverter{
		typeConverter: typeConverter,
		exprConverter: exprConverter,
	}
}

// FromParserStatement converts a parser.Statement to ast.Statement.
// This method provides comprehensive statement conversion with proper error handling.
// for all supported statement types in the Orizon language.
func (sc *StatementConverter) FromParserStatement(stmt p.Statement) (ast.Statement, error) {
	if stmt == nil {
		return nil, fmt.Errorf("cannot convert nil parser statement")
	}

	switch concrete := stmt.(type) {
	case *p.BlockStatement:
		return sc.fromParserBlockStatement(concrete)
	case *p.ExpressionStatement:
		return sc.fromParserExpressionStatement(concrete)
	case *p.ReturnStatement:
		return sc.fromParserReturnStatement(concrete)
	case *p.IfStatement:
		return sc.fromParserIfStatement(concrete)
	case *p.WhileStatement:
		return sc.fromParserWhileStatement(concrete)
	case *p.VariableDeclaration:
		return sc.fromParserVariableDeclaration(concrete)
	default:
		return nil, fmt.Errorf("unsupported parser statement type: %T", stmt)
	}
}

// ToParserStatement converts an ast.Statement to parser.Statement.
// This method provides the inverse conversion with comprehensive error handling.
// and maintains symmetry with FromParserStatement for bidirectional compatibility.
func (sc *StatementConverter) ToParserStatement(stmt ast.Statement) (p.Statement, error) {
	if stmt == nil {
		return nil, fmt.Errorf("cannot convert nil AST statement")
	}

	switch concrete := stmt.(type) {
	case *ast.BlockStatement:
		return sc.toParserBlockStatement(concrete)
	case *ast.ExpressionStatement:
		return sc.toParserExpressionStatement(concrete)
	case *ast.ReturnStatement:
		return sc.toParserReturnStatement(concrete)
	case *ast.IfStatement:
		return sc.toParserIfStatement(concrete)
	case *ast.WhileStatement:
		return sc.toParserWhileStatement(concrete)
	case *ast.VariableDeclaration:
		return sc.toParserVariableDeclaration(concrete)
	default:
		return nil, fmt.Errorf("unsupported AST statement type: %T", stmt)
	}
}

// fromParserBlockStatement converts parser BlockStatement to AST BlockStatement.
// This method handles block conversion with proper statement list processing.
func (sc *StatementConverter) fromParserBlockStatement(block *p.BlockStatement) (*ast.BlockStatement, error) {
	if block == nil {
		return nil, fmt.Errorf("cannot convert nil parser block statement")
	}

	statements := make([]ast.Statement, 0, len(block.Statements))

	for _, stmt := range block.Statements {
		convertedStmt, err := sc.FromParserStatement(stmt)
		if err != nil {
			return nil, fmt.Errorf("failed to convert statement in block: %w", err)
		}

		if convertedStmt != nil {
			statements = append(statements, convertedStmt)
		}
	}

	return &ast.BlockStatement{
		Span:       fromParserSpan(block.Span),
		Statements: statements,
	}, nil
}

// toParserBlockStatement converts AST BlockStatement to parser BlockStatement.
// This method provides the inverse conversion with proper statement handling.
func (sc *StatementConverter) toParserBlockStatement(block *ast.BlockStatement) (*p.BlockStatement, error) {
	if block == nil {
		return nil, fmt.Errorf("cannot convert nil AST block statement")
	}

	statements := make([]p.Statement, 0, len(block.Statements))

	for _, stmt := range block.Statements {
		convertedStmt, err := sc.ToParserStatement(stmt)
		if err != nil {
			return nil, fmt.Errorf("failed to convert statement in block: %w", err)
		}

		if convertedStmt != nil {
			statements = append(statements, convertedStmt)
		}
	}

	return &p.BlockStatement{
		Span:       toParserSpan(block.Span),
		Statements: statements,
	}, nil
}

// fromParserExpressionStatement converts parser ExpressionStatement to AST ExpressionStatement.
// This method handles expression statement conversion with proper expression processing.
func (sc *StatementConverter) fromParserExpressionStatement(exprStmt *p.ExpressionStatement) (*ast.ExpressionStatement, error) {
	if exprStmt == nil {
		return nil, fmt.Errorf("cannot convert nil parser expression statement")
	}

	expression, err := sc.exprConverter.FromParserExpression(exprStmt.Expression)
	if err != nil {
		return nil, fmt.Errorf("failed to convert expression in expression statement: %w", err)
	}

	return &ast.ExpressionStatement{
		Span:       fromParserSpan(exprStmt.Span),
		Expression: expression,
	}, nil
}

// toParserExpressionStatement converts AST ExpressionStatement to parser ExpressionStatement.
// This method provides the inverse conversion with proper expression handling.
func (sc *StatementConverter) toParserExpressionStatement(exprStmt *ast.ExpressionStatement) (*p.ExpressionStatement, error) {
	if exprStmt == nil {
		return nil, fmt.Errorf("cannot convert nil AST expression statement")
	}

	expression, err := sc.exprConverter.ToParserExpression(exprStmt.Expression)
	if err != nil {
		return nil, fmt.Errorf("failed to convert expression in expression statement: %w", err)
	}

	return &p.ExpressionStatement{
		Span:       toParserSpan(exprStmt.Span),
		Expression: expression,
	}, nil
}

// fromParserReturnStatement converts parser ReturnStatement to AST ReturnStatement.
// This method handles return statement conversion with optional value processing.
func (sc *StatementConverter) fromParserReturnStatement(retStmt *p.ReturnStatement) (*ast.ReturnStatement, error) {
	if retStmt == nil {
		return nil, fmt.Errorf("cannot convert nil parser return statement")
	}

	var value ast.Expression

	if retStmt.Value != nil {
		var err error

		value, err = sc.exprConverter.FromParserExpression(retStmt.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert return value: %w", err)
		}
	}

	return &ast.ReturnStatement{
		Span:  fromParserSpan(retStmt.Span),
		Value: value,
	}, nil
}

// toParserReturnStatement converts AST ReturnStatement to parser ReturnStatement.
// This method provides the inverse conversion with proper value handling.
func (sc *StatementConverter) toParserReturnStatement(retStmt *ast.ReturnStatement) (*p.ReturnStatement, error) {
	if retStmt == nil {
		return nil, fmt.Errorf("cannot convert nil AST return statement")
	}

	var value p.Expression

	if retStmt.Value != nil {
		var err error

		value, err = sc.exprConverter.ToParserExpression(retStmt.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert return value: %w", err)
		}
	}

	return &p.ReturnStatement{
		Span:  toParserSpan(retStmt.Span),
		Value: value,
	}, nil
}

// Stub implementations for complex statements (to be expanded).

func (sc *StatementConverter) fromParserIfStatement(ifStmt *p.IfStatement) (*ast.IfStatement, error) {
	return nil, fmt.Errorf("if statement conversion not yet implemented")
}

func (sc *StatementConverter) toParserIfStatement(ifStmt *ast.IfStatement) (*p.IfStatement, error) {
	return nil, fmt.Errorf("if statement conversion not yet implemented")
}

func (sc *StatementConverter) fromParserWhileStatement(whileStmt *p.WhileStatement) (*ast.WhileStatement, error) {
	return nil, fmt.Errorf("while statement conversion not yet implemented")
}

func (sc *StatementConverter) toParserWhileStatement(whileStmt *ast.WhileStatement) (*p.WhileStatement, error) {
	return nil, fmt.Errorf("while statement conversion not yet implemented")
}

func (sc *StatementConverter) fromParserVariableDeclaration(varDecl *p.VariableDeclaration) (*ast.VariableDeclaration, error) {
	return nil, fmt.Errorf("variable declaration conversion not yet implemented")
}

func (sc *StatementConverter) toParserVariableDeclaration(varDecl *ast.VariableDeclaration) (*p.VariableDeclaration, error) {
	return nil, fmt.Errorf("variable declaration conversion not yet implemented")
}
