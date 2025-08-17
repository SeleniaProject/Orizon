package astbridge

import (
	"fmt"
	"strconv"
	"strings"

	ast "github.com/orizon-lang/orizon/internal/ast"
	p "github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/position"
)

// prettyPrintParserType renders parser types including details missing from their String(),
// notably lifetimes on ReferenceType. Fallbacks to String() for others.
func prettyPrintParserType(t p.Type) string {
	switch n := t.(type) {
	case *p.ReferenceType:
		prefix := "&"
		if n.Lifetime != "" {
			prefix = "&'" + n.Lifetime + " "
		}
		if n.IsMutable {
			// include mut after potential lifetime
			if n.Lifetime != "" {
				prefix += "mut "
			} else {
				prefix = "&mut "
			}
		}
		return prefix + prettyPrintParserType(n.Inner)
	case *p.FunctionType:
		// [async ](params) -> return
		params := make([]string, 0, len(n.Parameters))
		for _, prm := range n.Parameters {
			pt := prettyPrintParserType(prm.Type)
			if prm.Name != "" {
				params = append(params, prm.Name+": "+pt)
			} else {
				params = append(params, pt)
			}
		}
		ret := "void"
		if n.ReturnType != nil {
			ret = prettyPrintParserType(n.ReturnType)
		}
		prefix := ""
		if n.IsAsync {
			prefix = "async "
		}
		return prefix + "(" + strings.Join(params, ", ") + ") -> " + ret
	case *p.GenericType:
		base := prettyPrintParserType(n.BaseType)
		args := make([]string, 0, len(n.TypeParameters))
		for _, a := range n.TypeParameters {
			args = append(args, prettyPrintParserType(a))
		}
		return base + "<" + strings.Join(args, ", ") + ">"
	case *p.PointerType:
		if n.IsMutable {
			return "*mut " + prettyPrintParserType(n.Inner)
		}
		return "*" + prettyPrintParserType(n.Inner)
	case *p.ArrayType:
		elem := prettyPrintParserType(n.ElementType)
		if n.IsDynamic {
			return "[" + elem + "]"
		}
		// size expression best-effort string
		sizeStr := "?"
		if n.Size != nil {
			if s, ok := any(n.Size).(fmt.Stringer); ok {
				sizeStr = s.String()
			}
		}
		return "[" + elem + "; " + sizeStr + "]"
	default:
		if s, ok := t.(fmt.Stringer); ok {
			return s.String()
		}
		return "<type>"
	}
}

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
	case *p.TypeAliasDeclaration:
		// Map parser type alias to core AST TypeDeclaration (IsAlias=true)
		var at ast.Type
		var err error
		if n.Aliased != nil {
			at, err = fromParserType(n.Aliased)
			if err != nil {
				return nil, err
			}
		}
		return &ast.TypeDeclaration{
			Span:       fromParserSpan(n.Span),
			Name:       &ast.Identifier{Span: fromParserSpan(n.Name.Span), Value: n.Name.Value},
			Type:       at,
			IsAlias:    true,
			IsExported: n.IsPublic,
		}, nil
	case *p.NewtypeDeclaration:
		// Map parser newtype to core AST TypeDeclaration (IsAlias=false)
		var bt ast.Type
		var err error
		if n.Base != nil {
			bt, err = fromParserType(n.Base)
			if err != nil {
				return nil, err
			}
		}
		return &ast.TypeDeclaration{
			Span:       fromParserSpan(n.Span),
			Name:       &ast.Identifier{Span: fromParserSpan(n.Name.Span), Value: n.Name.Value},
			Type:       bt,
			IsAlias:    false,
			IsExported: n.IsPublic,
		}, nil
	case *p.ImportDeclaration:
		// Map parser ImportDeclaration to core AST ImportDeclaration
		path := make([]*ast.Identifier, 0, len(n.Path))
		for _, seg := range n.Path {
			path = append(path, &ast.Identifier{Span: fromParserSpan(seg.Span), Value: seg.Value})
		}
		var alias *ast.Identifier
		if n.Alias != nil {
			alias = &ast.Identifier{Span: fromParserSpan(n.Alias.Span), Value: n.Alias.Value}
		}
		return &ast.ImportDeclaration{Span: fromParserSpan(n.Span), Path: path, Alias: alias, IsExported: n.IsPublic}, nil
	case *p.ExportDeclaration:
		items := make([]*ast.ExportItem, 0, len(n.Items))
		for _, it := range n.Items {
			var alias *ast.Identifier
			if it.Alias != nil { alias = &ast.Identifier{Span: fromParserSpan(it.Alias.Span), Value: it.Alias.Value} }
			items = append(items, &ast.ExportItem{Span: fromParserSpan(it.Span), Name: &ast.Identifier{Span: fromParserSpan(it.Name.Span), Value: it.Name.Value}, Alias: alias})
		}
	return &ast.ExportDeclaration{Span: fromParserSpan(n.Span), Items: items}, nil
	case *p.MacroDefinition:
		// Macros are compile-time only; skip them in AST bridge for now
		return nil, nil
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
	case *ast.TypeDeclaration:
		// Map core AST TypeDeclaration to parser alias/newtype decl
		var pt p.Type
		var err error
		if n.Type != nil {
			pt, err = toParserType(n.Type)
			if err != nil {
				return nil, err
			}
		}
		if n.IsAlias {
			return &p.TypeAliasDeclaration{
				Span:     toParserSpan(n.Span),
				Name:     &p.Identifier{Span: toParserSpan(n.Name.Span), Value: n.Name.Value},
				Aliased:  pt,
				IsPublic: n.IsExported,
			}, nil
		}
		return &p.NewtypeDeclaration{
			Span:     toParserSpan(n.Span),
			Name:     &p.Identifier{Span: toParserSpan(n.Name.Span), Value: n.Name.Value},
			Base:     pt,
			IsPublic: n.IsExported,
		}, nil
	case *ast.ImportDeclaration:
		path := make([]*p.Identifier, 0, len(n.Path))
		for _, seg := range n.Path {
			path = append(path, &p.Identifier{Span: toParserSpan(seg.Span), Value: seg.Value})
		}
		var alias *p.Identifier
		if n.Alias != nil {
			alias = &p.Identifier{Span: toParserSpan(n.Alias.Span), Value: n.Alias.Value}
		}
		return &p.ImportDeclaration{Span: toParserSpan(n.Span), Path: path, Alias: alias, IsPublic: n.IsExported}, nil
	case *ast.ExportDeclaration:
		items := make([]*p.ExportItem, 0, len(n.Items))
		for _, it := range n.Items {
			var alias *p.Identifier
			if it.Alias != nil { alias = &p.Identifier{Span: toParserSpan(it.Alias.Span), Value: it.Alias.Value} }
			items = append(items, &p.ExportItem{Span: toParserSpan(it.Span), Name: &p.Identifier{Span: toParserSpan(it.Name.Span), Value: it.Name.Value}, Alias: alias})
		}
		return &p.ExportDeclaration{Span: toParserSpan(n.Span), Items: items}, nil
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
	var val ast.Expression
	if v.Initializer != nil {
		val, err = fromParserExpr(v.Initializer)
		if err != nil {
			return nil, err
		}
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
	case *p.VariableDeclaration:
		// VariableDeclaration is both a declaration and a statement in the parser AST
		vd, err := fromParserVar(n)
		if err != nil {
			return nil, err
		}
		return vd, nil
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
	case *p.MatchStatement:
		// Lower match into a chain of if/else if/else.
		// Semantics: for each arm (pattern, [guard], body):
		//   if (scrutinee == pattern) && (guard?) { body }
		// Note: This is a simplistic lowering; real pattern matching would need richer semantics.
		scrutinee, err := fromParserExpr(n.Expression)
		if err != nil {
			return nil, err
		}
		// Build a nested if/else chain
		var chain ast.Statement
		// Iterate arms in reverse to nest properly
		for i := len(n.Arms) - 1; i >= 0; i-- {
			arm := n.Arms[i]
			pat, err := fromParserExpr(arm.Pattern)
			if err != nil {
				return nil, err
			}
			// Build equality: scrutinee == pat
			eq := &ast.BinaryExpression{Span: fromParserSpan(arm.Span), Left: scrutinee, Operator: ast.OpEq, Right: pat}
			cond := ast.Expression(eq)
			if arm.Guard != nil {
				g, err := fromParserExpr(arm.Guard)
				if err != nil {
					return nil, err
				}
				cond = &ast.BinaryExpression{Span: fromParserSpan(arm.Span), Left: cond, Operator: ast.OpAnd, Right: g}
			}
			bodyStmt, err := fromParserStmt(arm.Body)
			if err != nil {
				return nil, err
			}
			if chain == nil {
				chain = &ast.IfStatement{Span: fromParserSpan(arm.Span), Condition: cond, ThenBlock: ensureBlock(bodyStmt), ElseBlock: nil}
			} else {
				chain = &ast.IfStatement{Span: fromParserSpan(arm.Span), Condition: cond, ThenBlock: ensureBlock(bodyStmt), ElseBlock: chain}
			}
		}
		if chain == nil {
			// Empty match; lower to empty block
			return &ast.BlockStatement{Span: fromParserSpan(n.Span), Statements: nil}, nil
		}
		return chain, nil
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
	case *p.MacroInvocation:
		// Placeholder: no macro expansion in AST bridge yet.
		// Represent as a benign null literal so downstream passes can proceed.
		return &ast.Literal{Span: fromParserSpan(n.Span), Kind: ast.LiteralNull, Value: nil, Raw: "null"}, nil
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
		// Map well-known basic names to ast.BasicType, otherwise preserve as IdentifierType
		switch n.Name {
		case "int", "float", "string", "bool", "char", "void":
			return &ast.BasicType{Span: fromParserSpan(n.Span), Kind: mapBasicTypeName(n.Name)}, nil
		default:
			return &ast.IdentifierType{Span: fromParserSpan(n.Span), Name: &ast.Identifier{Span: fromParserSpan(n.Span), Value: n.Name}}, nil
		}
	// For complex parser types that don't have direct equivalents in internal/ast,
	// preserve their textual form as an IdentifierType to avoid losing information.
	case *p.ArrayType, *p.FunctionType, *p.StructType, *p.EnumType, *p.TraitType,
		*p.GenericType, *p.ReferenceType, *p.PointerType:
		// Preserve textual representation (including lifetime for references).
		sp := n.(p.Node).GetSpan()
		return &ast.IdentifierType{Span: fromParserSpan(sp), Name: &ast.Identifier{Span: fromParserSpan(sp), Value: prettyPrintParserType(n)}}, nil
	default:
		// Fallback: try String() if available; otherwise, report unsupported
		if s, ok := t.(fmt.Stringer); ok {
			// Attempt best-effort carry of the textual type into IdentifierType
			// Span information: use zero span if node doesn't expose it
			var sp p.Span
			if n2, ok2 := t.(p.Node); ok2 {
				sp = n2.GetSpan()
			}
			return &ast.IdentifierType{Span: fromParserSpan(sp), Name: &ast.Identifier{Span: fromParserSpan(sp), Value: s.String()}}, nil
		}
		return nil, fmt.Errorf("unsupported parser type %T", t)
	}
}

func toParserType(t ast.Type) (p.Type, error) {
	switch n := t.(type) {
	case *ast.BasicType:
		return &p.BasicType{Span: toParserSpan(n.Span), Name: n.String()}, nil
	case *ast.IdentifierType:
		// Try to reconstruct structured parser types from textual identifier forms
		if pt := parseIdentifierTypeStringToParserType(n.String()); pt != nil {
			return pt, nil
		}
		// Fallback: keep as named basic type
		return &p.BasicType{Span: toParserSpan(n.Span), Name: n.String()}, nil
	default:
		return nil, fmt.Errorf("unsupported ast type %T", t)
	}
}

// parseIdentifierTypeStringToParserType recognizes simple textual forms like
//
//	&T, &mut T, &'a T, &'a mut T, *T, *mut T
//
// and converts them into parser structured types. Returns nil if unrecognized.
func parseIdentifierTypeStringToParserType(s string) p.Type {
	// lightweight parsing without allocations-heavy regex
	trim := func(x string) string {
		i, j := 0, len(x)
		for i < j && (x[i] == ' ' || x[i] == '\t' || x[i] == '\n' || x[i] == '\r') {
			i++
		}
		for j > i && (x[j-1] == ' ' || x[j-1] == '\t' || x[j-1] == '\n' || x[j-1] == '\r') {
			j--
		}
		return x[i:j]
	}
	s = trim(s)
	if s == "" {
		return nil
	}

	// Generic forms: Base<T1, T2, ...>
	// Detect first '<' at depth 0 and trailing '>'
	// but avoid mistaking array/function brackets
	// We'll attempt generic parse before reference/pointer when base doesn't start with '&' or '*'
	if len(s) > 0 && s[0] != '&' && s[0] != '*' && s[0] != '[' {
		// find top-level '<'
		depthAngle, depthSquare, depthParen := 0, 0, 0
		idx := -1
		for i := 0; i < len(s); i++ {
			switch s[i] {
			case '<':
				if depthAngle == 0 && depthSquare == 0 && depthParen == 0 {
					idx = i
				}
				depthAngle++
			case '>':
				if depthAngle > 0 {
					depthAngle--
				}
			case '[':
				depthSquare++
			case ']':
				if depthSquare > 0 {
					depthSquare--
				}
			case '(':
				depthParen++
			case ')':
				if depthParen > 0 {
					depthParen--
				}
			}
		}
		if idx >= 0 && s[len(s)-1] == '>' {
			baseName := trim(s[:idx])
			argStr := s[idx+1 : len(s)-1]
			// split args by commas at top level
			args := make([]p.Type, 0)
			depthA, depthS, depthP := 0, 0, 0
			start := 0
			addArg := func(end int) {
				part := trim(argStr[start:end])
				if part == "" {
					return
				}
				t := parseIdentifierTypeStringToParserType(part)
				if t == nil {
					t = &p.BasicType{Name: part}
				}
				args = append(args, t)
			}
			for i := 0; i < len(argStr); i++ {
				switch argStr[i] {
				case '<':
					depthA++
				case '>':
					if depthA > 0 {
						depthA--
					}
				case '[':
					depthS++
				case ']':
					if depthS > 0 {
						depthS--
					}
				case '(':
					depthP++
				case ')':
					if depthP > 0 {
						depthP--
					}
				case ',':
					if depthA == 0 && depthS == 0 && depthP == 0 {
						addArg(i)
						start = i + 1
					}
				}
			}
			addArg(len(argStr))
			base := parseIdentifierTypeStringToParserType(baseName)
			if base == nil {
				base = &p.BasicType{Name: baseName}
			}
			return &p.GenericType{BaseType: base, TypeParameters: args}
		}
	}

	// Reference forms
	if s[0] == '&' {
		rest := trim(s[1:])
		lifetime := ""
		isMut := false
		// Lifetime like 'a ...
		if len(rest) > 0 && rest[0] == '\'' {
			// consume until space
			k := 1
			for k < len(rest) && rest[k] != ' ' {
				k++
			}
			lifetime = rest[1:k]
			rest = trim(rest[k:])
		}
		// Optional mut
		if len(rest) >= 4 && rest[:4] == "mut " {
			isMut = true
			rest = trim(rest[4:])
		}
		if rest == "" {
			return nil
		}
		// Recurse for inner type
		inner := parseIdentifierTypeStringToParserType(rest)
		if inner == nil {
			inner = &p.BasicType{Name: rest}
		}
		return &p.ReferenceType{Inner: inner, IsMutable: isMut, Lifetime: lifetime}
	}

	// Pointer forms
	if s[0] == '*' {
		rest := trim(s[1:])
		isMut := false
		if len(rest) >= 4 && rest[:4] == "mut " {
			isMut = true
			rest = trim(rest[4:])
		}
		if rest == "" {
			return nil
		}
		// Recurse for inner type
		inner := parseIdentifierTypeStringToParserType(rest)
		if inner == nil {
			inner = &p.BasicType{Name: rest}
		}
		return &p.PointerType{Inner: inner, IsMutable: isMut}
	}

	// Array forms: [T] or [T; N]
	if s[0] == '[' && s[len(s)-1] == ']' {
		inside := s[1 : len(s)-1]
		// split on ';' if present (only top-level)
		elemStr := inside
		var sizeExpr p.Expression = nil
		// find top-level ';'
		depthAngle, depthSquare := 0, 0
		semiIndex := -1
		for i := 0; i < len(inside); i++ {
			switch inside[i] {
			case '<':
				depthAngle++
			case '>':
				if depthAngle > 0 {
					depthAngle--
				}
			case '[':
				depthSquare++
			case ']':
				if depthSquare > 0 {
					depthSquare--
				}
			case ';':
				if depthAngle == 0 && depthSquare == 0 {
					semiIndex = i
					i = len(inside) // break
				}
			}
		}
		if semiIndex >= 0 {
			elemStr = inside[:semiIndex]
			sz := trim(inside[semiIndex+1:])
			if sz != "" {
				// try parse integer size
				if iv, err := strconv.Atoi(sz); err == nil {
					sizeExpr = &p.Literal{Value: iv}
				} else {
					// fallback: identifier expression
					sizeExpr = &p.Identifier{Value: sz}
				}
			}
		}
		elemStr = trim(elemStr)
		elem := parseIdentifierTypeStringToParserType(elemStr)
		if elem == nil {
			elem = &p.BasicType{Name: elemStr}
		}
		return &p.ArrayType{ElementType: elem, Size: sizeExpr, IsDynamic: semiIndex < 0}
	}

	// Function types: [async ](params) -> returnType
	// Accept optional leading 'async '
	{
		isAsync := false
		rest := s
		if len(rest) >= 6 && rest[:6] == "async " {
			isAsync = true
			rest = trim(rest[6:])
		}
		if len(rest) > 0 && rest[0] == '(' {
			// find matching ')'
			depth := 0
			end := -1
			for i := 0; i < len(rest); i++ {
				if rest[i] == '(' {
					depth++
				}
				if rest[i] == ')' {
					depth--
					if depth == 0 {
						end = i
						break
					}
				}
			}
			if end > 0 {
				paramsStr := rest[1:end]
				after := trim(rest[end+1:])
				if len(after) >= 2 && after[:2] == "->" {
					retStr := trim(after[2:])
					// parse params
					params := []*p.FunctionTypeParameter{}
					// split by commas at top-level
					depthA, depthS, depthP := 0, 0, 0
					start := 0
					addParam := func(end int) {
						part := trim(paramsStr[start:end])
						if part == "" {
							return
						}
						name := ""
						typStr := part
						// optional name: type (only split at first ':')
						// find top-level ':'
						dA, dS, dP := 0, 0, 0
						colon := -1
						for i := 0; i < len(part); i++ {
							switch part[i] {
							case '<':
								dA++
							case '>':
								if dA > 0 {
									dA--
								}
							case '[':
								dS++
							case ']':
								if dS > 0 {
									dS--
								}
							case '(':
								dP++
							case ')':
								if dP > 0 {
									dP--
								}
							case ':':
								if dA == 0 && dS == 0 && dP == 0 {
									colon = i
									i = len(part)
								}
							}
						}
						if colon >= 0 {
							name = trim(part[:colon])
							typStr = trim(part[colon+1:])
						}
						t := parseIdentifierTypeStringToParserType(typStr)
						if t == nil {
							t = &p.BasicType{Name: typStr}
						}
						params = append(params, &p.FunctionTypeParameter{Name: name, Type: t})
					}
					for i := 0; i < len(paramsStr); i++ {
						switch paramsStr[i] {
						case '<':
							depthA++
						case '>':
							if depthA > 0 {
								depthA--
							}
						case '[':
							depthS++
						case ']':
							if depthS > 0 {
								depthS--
							}
						case '(':
							depthP++
						case ')':
							if depthP > 0 {
								depthP--
							}
						case ',':
							if depthA == 0 && depthS == 0 && depthP == 0 {
								addParam(i)
								start = i + 1
							}
						}
					}
					addParam(len(paramsStr))
					// return type
					ret := parseIdentifierTypeStringToParserType(retStr)
					if ret == nil {
						ret = &p.BasicType{Name: retStr}
					}
					return &p.FunctionType{Parameters: params, ReturnType: ret, IsAsync: isAsync}
				}
			}
		}
	}

	return nil
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
