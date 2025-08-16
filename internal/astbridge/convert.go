package astbridge

import (
	"fmt"

	ast "github.com/orizon-lang/orizon/internal/ast"
	p "github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/position"
)

// FromParserProgram converts parser.Program to ast.Program.
func FromParserProgram(src *p.Program) (*ast.Program, error) {
	if src == nil {
		return nil, fmt.Errorf("nil program")
	}
	dst := &ast.Program{Span: fromParserSpan(src.Span), Declarations: make([]ast.Declaration, 0, len(src.Declarations))}
	for _, d := range src.Declarations {
		conv, err := fromParserDecl(d)
		if err != nil {
			return nil, err
		}
		if conv != nil {
			dst.Declarations = append(dst.Declarations, conv)
		}
	}
	return dst, nil
}

// ToParserProgram converts ast.Program to parser.Program.
func ToParserProgram(src *ast.Program) (*p.Program, error) {
	if src == nil {
		return nil, fmt.Errorf("nil program")
	}
	dst := &p.Program{Span: toParserSpan(src.Span), Declarations: make([]p.Declaration, 0, len(src.Declarations))}
	for _, d := range src.Declarations {
		conv, err := toParserDecl(d)
		if err != nil {
			return nil, err
		}
		if conv != nil {
			dst.Declarations = append(dst.Declarations, conv)
		}
	}
	return dst, nil
}

// ===== Declarations =====

func fromParserDecl(d p.Declaration) (ast.Declaration, error) {
	switch n := d.(type) {
	case *p.FunctionDeclaration:
		return fromParserFunc(n)
	case *p.VariableDeclaration:
		return fromParserVar(n)
	case *p.ExpressionStatement:
		s, err := fromParserStmt(n)
		if err != nil {
			return nil, err
		}
		if decl, ok := s.(ast.Declaration); ok {
			return decl, nil
		}
		return &ast.TypeDeclaration{Span: position.Span{}, Name: &ast.Identifier{Span: position.Span{}, Value: "_"}, Type: &ast.BasicType{Kind: ast.BasicVoid}}, nil
	default:
		return nil, fmt.Errorf("unsupported parser declaration type %T", d)
	}
}

func toParserDecl(d ast.Declaration) (p.Declaration, error) {
	switch n := d.(type) {
	case *ast.FunctionDeclaration:
		return toParserFunc(n)
	case *ast.VariableDeclaration:
		return toParserVar(n)
	default:
		return nil, fmt.Errorf("unsupported ast declaration type %T", d)
	}
}

func fromParserFunc(fn *p.FunctionDeclaration) (*ast.FunctionDeclaration, error) {
	params := make([]*ast.Parameter, 0, len(fn.Parameters))
	for _, pp := range fn.Parameters {
		t, err := fromParserType(pp.TypeSpec)
		if err != nil {
			return nil, err
		}
		params = append(params, &ast.Parameter{Span: fromParserSpan(pp.Span), Name: &ast.Identifier{Span: fromParserSpan(pp.Span), Value: pp.Name.Value}, Type: t})
	}
	var ret ast.Type
	var err error
	if fn.ReturnType != nil {
		ret, err = fromParserType(fn.ReturnType)
		if err != nil {
			return nil, err
		}
	}
	bodyStmt, err := fromParserStmt(fn.Body)
	if err != nil {
		return nil, err
	}
	body := ensureBlock(bodyStmt)
	return &ast.FunctionDeclaration{Span: fromParserSpan(fn.Span), Name: &ast.Identifier{Span: fromParserSpan(fn.Name.Span), Value: fn.Name.Value}, Parameters: params, ReturnType: ret, Body: body, Attributes: nil, IsExported: fn.IsPublic, Comments: nil}, nil
}

func toParserFunc(fn *ast.FunctionDeclaration) (*p.FunctionDeclaration, error) {
	params := make([]*p.Parameter, 0, len(fn.Parameters))
	for _, ap := range fn.Parameters {
		pt, err := toParserType(ap.Type)
		if err != nil {
			return nil, err
		}
		params = append(params, &p.Parameter{Span: toParserSpan(ap.Span), Name: &p.Identifier{Span: toParserSpan(ap.Name.Span), Value: ap.Name.Value}, TypeSpec: pt})
	}
	var ret p.Type
	var err error
	if fn.ReturnType != nil {
		ret, err = toParserType(fn.ReturnType)
		if err != nil {
			return nil, err
		}
	}
	bodyStmt, err := toParserStmt(fn.Body)
	if err != nil {
		return nil, err
	}
	blk, _ := bodyStmt.(*p.BlockStatement)
	if blk == nil {
		blk = &p.BlockStatement{}
	}
	return &p.FunctionDeclaration{Span: toParserSpan(fn.Span), Name: &p.Identifier{Span: toParserSpan(fn.Name.Span), Value: fn.Name.Value}, Parameters: params, ReturnType: ret, Body: blk, IsPublic: fn.IsExported}, nil
}

func fromParserVar(v *p.VariableDeclaration) (*ast.VariableDeclaration, error) {
	var t ast.Type
	var err error
	if v.TypeSpec != nil {
		t, err = fromParserType(v.TypeSpec)
		if err != nil {
			return nil, err
		}
	}
	val, err := fromParserExpr(v.Initializer)
	if err != nil {
		return nil, err
	}
	kind := ast.VarKindLet
	if v.IsMutable {
		kind = ast.VarKindVar
	}
	return &ast.VariableDeclaration{Span: fromParserSpan(v.Span), Name: &ast.Identifier{Span: fromParserSpan(v.Name.Span), Value: v.Name.Value}, Type: t, Value: val, Kind: kind, IsMutable: v.IsMutable, IsExported: v.IsPublic}, nil
}

func toParserVar(v *ast.VariableDeclaration) (*p.VariableDeclaration, error) {
	var t p.Type
	var err error
	if v.Type != nil {
		t, err = toParserType(v.Type)
		if err != nil {
			return nil, err
		}
	}
	return &p.VariableDeclaration{Span: toParserSpan(v.Span), Name: &p.Identifier{Span: toParserSpan(v.Name.Span), Value: v.Name.Value}, TypeSpec: t, Initializer: toParserExprOrNil(v.Value), IsMutable: v.IsMutable, IsPublic: v.IsExported}, nil
}

// ===== Statements =====

func fromParserStmt(s p.Statement) (ast.Statement, error) {
	switch n := s.(type) {
	case *p.BlockStatement:
		blk := &ast.BlockStatement{Span: fromParserSpan(n.Span), Statements: make([]ast.Statement, 0, len(n.Statements))}
		for _, st := range n.Statements {
			cs, err := fromParserStmt(st)
			if err != nil {
				return nil, err
			}
			blk.Statements = append(blk.Statements, cs)
		}
		return blk, nil
	case *p.ExpressionStatement:
		e, err := fromParserExpr(n.Expression)
		if err != nil {
			return nil, err
		}
		return &ast.ExpressionStatement{Span: fromParserSpan(n.Span), Expression: e}, nil
	case *p.ReturnStatement:
		var val ast.Expression
		var err error
		if n.Value != nil {
			val, err = fromParserExpr(n.Value)
			if err != nil {
				return nil, err
			}
		}
		return &ast.ReturnStatement{Span: fromParserSpan(n.Span), Value: val}, nil
	case *p.IfStatement:
		cond, err := fromParserExpr(n.Condition)
		if err != nil {
			return nil, err
		}
		thenStmt, err := fromParserStmt(n.ThenStmt)
		if err != nil {
			return nil, err
		}
		var elseStmt ast.Statement
		if n.ElseStmt != nil {
			elseStmt, err = fromParserStmt(n.ElseStmt)
			if err != nil {
				return nil, err
			}
		}
		return &ast.IfStatement{Span: fromParserSpan(n.Span), Condition: cond, ThenBlock: ensureBlock(thenStmt), ElseBlock: elseStmt}, nil
	case *p.WhileStatement:
		cond, err := fromParserExpr(n.Condition)
		if err != nil {
			return nil, err
		}
		body, err := fromParserStmt(n.Body)
		if err != nil {
			return nil, err
		}
		return &ast.WhileStatement{Span: fromParserSpan(n.Span), Condition: cond, Body: ensureBlock(body)}, nil
	default:
		return nil, fmt.Errorf("unsupported parser statement type %T", s)
	}
}

func toParserStmt(s ast.Statement) (p.Statement, error) {
	switch n := s.(type) {
	case *ast.BlockStatement:
		out := &p.BlockStatement{Span: toParserSpan(n.Span), Statements: make([]p.Statement, 0, len(n.Statements))}
		for _, st := range n.Statements {
			cs, err := toParserStmt(st)
			if err != nil {
				return nil, err
			}
			out.Statements = append(out.Statements, cs)
		}
		return out, nil
	case *ast.ExpressionStatement:
		return &p.ExpressionStatement{Span: toParserSpan(n.Span), Expression: toParserExprOrNil(n.Expression)}, nil
	case *ast.ReturnStatement:
		var val p.Expression
		if n.Value != nil {
			val = toParserExprOrNil(n.Value)
		}
		return &p.ReturnStatement{Span: toParserSpan(n.Span), Value: val}, nil
	case *ast.IfStatement:
		var elseStmt p.Statement
		var err error
		if n.ElseBlock != nil {
			elseStmt, err = toParserStmt(n.ElseBlock)
			if err != nil {
				return nil, err
			}
		}
		thenStmt, err := toParserStmt(n.ThenBlock)
		if err != nil {
			return nil, err
		}
		return &p.IfStatement{Span: toParserSpan(n.Span), Condition: toParserExprOrNil(n.Condition), ThenStmt: thenStmt, ElseStmt: elseStmt}, nil
	case *ast.WhileStatement:
		body, err := toParserStmt(n.Body)
		if err != nil {
			return nil, err
		}
		return &p.WhileStatement{Span: toParserSpan(n.Span), Condition: toParserExprOrNil(n.Condition), Body: body}, nil
	default:
		return nil, fmt.Errorf("unsupported ast statement type %T", s)
	}
}

func ensureBlock(s ast.Statement) *ast.BlockStatement {
	if b, ok := s.(*ast.BlockStatement); ok {
		return b
	}
	return &ast.BlockStatement{Statements: []ast.Statement{s}}
}

// ===== Expressions =====

func fromParserExpr(e p.Expression) (ast.Expression, error) {
	switch n := e.(type) {
	case *p.Identifier:
		return &ast.Identifier{Span: fromParserSpan(n.Span), Value: n.Value}, nil
	case *p.Literal:
		return fromParserLiteral(n), nil
	case *p.BinaryExpression:
		left, err := fromParserExpr(n.Left)
		if err != nil {
			return nil, err
		}
		right, err := fromParserExpr(n.Right)
		if err != nil {
			return nil, err
		}
		op, err := fromParserOperator(n.Operator)
		if err != nil {
			return nil, err
		}
		return &ast.BinaryExpression{Span: fromParserSpan(n.Span), Left: left, Operator: op, Right: right}, nil
	case *p.UnaryExpression:
		operand, err := fromParserExpr(n.Operand)
		if err != nil {
			return nil, err
		}
		op, err := fromParserOperator(n.Operator)
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpression{Span: fromParserSpan(n.Span), Operator: op, Operand: operand}, nil
	case *p.CallExpression:
		fn, err := fromParserExpr(n.Function)
		if err != nil {
			return nil, err
		}
		args := make([]ast.Expression, 0, len(n.Arguments))
		for _, a := range n.Arguments {
			ca, err := fromParserExpr(a)
			if err != nil {
				return nil, err
			}
			args = append(args, ca)
		}
		return &ast.CallExpression{Span: fromParserSpan(n.Span), Function: fn, Arguments: args}, nil
	default:
		return nil, fmt.Errorf("unsupported parser expression type %T", e)
	}
}

func toParserExprOrNil(e ast.Expression) p.Expression {
	if e == nil {
		return nil
	}
	out, _ := toParserExpr(e)
	return out
}

func toParserExpr(e ast.Expression) (p.Expression, error) {
	switch n := e.(type) {
	case *ast.Identifier:
		return &p.Identifier{Span: toParserSpan(n.Span), Value: n.Value}, nil
	case *ast.Literal:
		return toParserLiteral(n), nil
	case *ast.BinaryExpression:
		l, err := toParserExpr(n.Left)
		if err != nil {
			return nil, err
		}
		r, err := toParserExpr(n.Right)
		if err != nil {
			return nil, err
		}
		return &p.BinaryExpression{Span: toParserSpan(n.Span), Left: l, Operator: toParserOperator(n.Operator), Right: r}, nil
	case *ast.UnaryExpression:
		o, err := toParserExpr(n.Operand)
		if err != nil {
			return nil, err
		}
		return &p.UnaryExpression{Span: toParserSpan(n.Span), Operator: toParserOperator(n.Operator), Operand: o}, nil
	case *ast.CallExpression:
		f, err := toParserExpr(n.Function)
		if err != nil {
			return nil, err
		}
		args := make([]p.Expression, 0, len(n.Arguments))
		for _, a := range n.Arguments {
			ca, err := toParserExpr(a)
			if err != nil {
				return nil, err
			}
			args = append(args, ca)
		}
		return &p.CallExpression{Span: toParserSpan(n.Span), Function: f, Arguments: args}, nil
	default:
		return nil, fmt.Errorf("unsupported ast expression type %T", e)
	}
}

// ===== Types =====

func fromParserType(t p.Type) (ast.Type, error) {
	switch n := t.(type) {
	case *p.BasicType:
		return &ast.BasicType{Span: fromParserSpan(n.Span), Kind: mapBasicTypeName(n.Name)}, nil
	default:
		return nil, fmt.Errorf("unsupported parser type %T", t)
	}
}

func toParserType(t ast.Type) (p.Type, error) {
	switch n := t.(type) {
	case *ast.BasicType:
		return &p.BasicType{Span: toParserSpan(n.Span), Name: n.String()}, nil
	case *ast.IdentifierType:
		return &p.BasicType{Span: toParserSpan(n.Span), Name: n.String()}, nil
	default:
		return nil, fmt.Errorf("unsupported ast type %T", t)
	}
}

// ===== Literals and Operators =====

func fromParserLiteral(l *p.Literal) *ast.Literal {
	switch v := l.Value.(type) {
	case int:
		return &ast.Literal{Span: fromParserSpan(l.Span), Kind: ast.LiteralInteger, Value: int64(v), Raw: fmt.Sprintf("%d", v)}
	case int64:
		return &ast.Literal{Span: fromParserSpan(l.Span), Kind: ast.LiteralInteger, Value: v, Raw: fmt.Sprintf("%d", v)}
	case float64:
		return &ast.Literal{Span: fromParserSpan(l.Span), Kind: ast.LiteralFloat, Value: v, Raw: fmt.Sprintf("%g", v)}
	case string:
		return &ast.Literal{Span: fromParserSpan(l.Span), Kind: ast.LiteralString, Value: v, Raw: fmt.Sprintf("\"%s\"", v)}
	case bool:
		return &ast.Literal{Span: fromParserSpan(l.Span), Kind: ast.LiteralBoolean, Value: v, Raw: fmt.Sprintf("%t", v)}
	default:
		return &ast.Literal{Span: fromParserSpan(l.Span), Kind: ast.LiteralNull, Value: nil, Raw: "null"}
	}
}

func toParserLiteral(l *ast.Literal) *p.Literal {
	switch l.Kind {
	case ast.LiteralInteger:
		switch v := l.Value.(type) {
		case int:
			return &p.Literal{Span: toParserSpan(l.Span), Value: v, Kind: p.LiteralInteger}
		case int64:
			return &p.Literal{Span: toParserSpan(l.Span), Value: int(v), Kind: p.LiteralInteger}
		default:
			return &p.Literal{Span: toParserSpan(l.Span), Value: 0, Kind: p.LiteralInteger}
		}
	case ast.LiteralFloat:
		if v, ok := l.Value.(float64); ok {
			return &p.Literal{Span: toParserSpan(l.Span), Value: v, Kind: p.LiteralFloat}
		}
		return &p.Literal{Span: toParserSpan(l.Span), Value: 0.0, Kind: p.LiteralFloat}
	case ast.LiteralString:
		if v, ok := l.Value.(string); ok {
			return &p.Literal{Span: toParserSpan(l.Span), Value: v, Kind: p.LiteralString}
		}
		return &p.Literal{Span: toParserSpan(l.Span), Value: "", Kind: p.LiteralString}
	case ast.LiteralBoolean:
		if v, ok := l.Value.(bool); ok {
			return &p.Literal{Span: toParserSpan(l.Span), Value: v, Kind: p.LiteralBool}
		}
		return &p.Literal{Span: toParserSpan(l.Span), Value: false, Kind: p.LiteralBool}
	default:
		return &p.Literal{Span: toParserSpan(l.Span), Value: nil, Kind: p.LiteralNull}
	}
}

func fromParserOperator(op *p.Operator) (ast.Operator, error) {
	switch op.Value {
	case "+":
		return ast.OpAdd, nil
	case "-":
		return ast.OpSub, nil
	case "*":
		return ast.OpMul, nil
	case "/":
		return ast.OpDiv, nil
	case "%":
		return ast.OpMod, nil
	case "==":
		return ast.OpEq, nil
	case "!=":
		return ast.OpNe, nil
	case "<":
		return ast.OpLt, nil
	case "<=":
		return ast.OpLe, nil
	case ">":
		return ast.OpGt, nil
	case ">=":
		return ast.OpGe, nil
	case "&&":
		return ast.OpAnd, nil
	case "||":
		return ast.OpOr, nil
	case "!":
		return ast.OpNot, nil
	default:
		return 0, fmt.Errorf("unsupported operator %q", op.Value)
	}
}

func toParserOperator(op ast.Operator) *p.Operator {
	return &p.Operator{Value: op.String(), Kind: p.BinaryOp}
}

func mapBasicTypeName(name string) ast.BasicKind {
	switch name {
	case "int":
		return ast.BasicInt
	case "float":
		return ast.BasicFloat
	case "string":
		return ast.BasicString
	case "bool":
		return ast.BasicBool
	case "char":
		return ast.BasicChar
	case "void":
		return ast.BasicVoid
	default:
		return ast.BasicInt
	}
}

// ==== Span Conversion ====

func fromParserSpan(s p.Span) position.Span {
	return position.Span{Start: position.Position{Filename: s.Start.File, Line: s.Start.Line, Column: s.Start.Column, Offset: s.Start.Offset}, End: position.Position{Filename: s.End.File, Line: s.End.Line, Column: s.End.Column, Offset: s.End.Offset}}
}
func toParserSpan(s position.Span) p.Span {
	return p.Span{Start: p.Position{File: s.Start.Filename, Line: s.Start.Line, Column: s.Start.Column, Offset: s.Start.Offset}, End: p.Position{File: s.End.Filename, Line: s.End.Line, Column: s.End.Column, Offset: s.End.Offset}}
}
