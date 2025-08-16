package parser

import (
	"fmt"

	aast "github.com/orizon-lang/orizon/internal/ast"
)

// OptimizeViaAstPipe converts a parser.Program into ast.Program, runs the ast optimization
// pipeline, and converts the result back to parser.Program.
// This enables using the richer optimization facilities in internal/ast without
// changing upstream parser users.
func OptimizeViaAstPipe(program *Program, level string) (*Program, error) {
	if program == nil {
		return nil, fmt.Errorf("nil program")
	}

	// Convert parser -> ast (local lightweight conversion to avoid import cycles)
	aProg, err := parserToAst(program)
	if err != nil {
		return nil, fmt.Errorf("convert to ast failed: %w", err)
	}

	// Select pipeline by level
	var pipeline *aast.OptimizationPipeline
	switch level {
	case "none":
		pipeline = aast.NewOptimizationPipeline()
		pipeline.SetOptimizationLevel(aast.OptimizationNone)
	case "basic":
		pipeline = aast.CreateBasicOptimizationPipeline()
	case "aggressive":
		pipeline = aast.CreateAggressiveOptimizationPipeline()
	case "default", "":
		fallthrough
	default:
		pipeline = aast.CreateStandardOptimizationPipeline()
	}

	// Run optimization
	optimizedNode, _, err := pipeline.Optimize(aProg)
	if err != nil {
		return nil, fmt.Errorf("ast optimization failed: %w", err)
	}

	// The root should remain a Program
	aProgOpt, ok := optimizedNode.(*aast.Program)
	if !ok {
		return nil, fmt.Errorf("unexpected optimized root type %T", optimizedNode)
	}

	// Convert back ast -> parser
	back, err := astToParser(aProgOpt)
	if err != nil {
		return nil, fmt.Errorf("convert back to parser failed: %w", err)
	}
	return back, nil
}

// Local minimal converters (subset) to avoid package cycles during optimization bridging
func parserToAst(src *Program) (*aast.Program, error) {
	if src == nil {
		return nil, fmt.Errorf("nil program")
	}
	dst := &aast.Program{Declarations: make([]aast.Declaration, 0, len(src.Declarations))}
	for _, d := range src.Declarations {
		switch n := d.(type) {
		case *FunctionDeclaration:
			// In parser AST, Body is *BlockStatement
			body, err := pBlockToAst(n.Body)
			if err != nil {
				return nil, err
			}
			dst.Declarations = append(dst.Declarations, &aast.FunctionDeclaration{
				Name: &aast.Identifier{Value: n.Name.Value},
				Body: body,
			})
		case *VariableDeclaration:
			dst.Declarations = append(dst.Declarations, &aast.VariableDeclaration{
				Name: &aast.Identifier{Value: n.Name.Value},
				Kind: aast.VarKindLet,
			})
		default:
			// Skip unsupported declarations
		}
	}
	return dst, nil
}

func astToParser(src *aast.Program) (*Program, error) {
	if src == nil {
		return nil, fmt.Errorf("nil program")
	}
	dst := &Program{Declarations: make([]Declaration, 0, len(src.Declarations))}
	for _, d := range src.Declarations {
		switch n := d.(type) {
		case *aast.FunctionDeclaration:
			body, err := aBlockToParser(n.Body)
			if err != nil {
				return nil, err
			}
			dst.Declarations = append(dst.Declarations, &FunctionDeclaration{Name: &Identifier{Value: n.Name.Value}, Body: body})
		case *aast.VariableDeclaration:
			// reconstruct as expression statement to avoid needing full var decl mapping
			dst.Declarations = append(dst.Declarations, &ExpressionStatement{Expression: &Identifier{Value: n.Name.Value}})
		default:
			// Skip
		}
	}
	return dst, nil
}

// ===== Helpers: Statement/Expression conversion =====

func pBlockToAst(b *BlockStatement) (*aast.BlockStatement, error) {
	if b == nil {
		return &aast.BlockStatement{}, nil
	}
	out := &aast.BlockStatement{Statements: make([]aast.Statement, 0, len(b.Statements))}
	for _, st := range b.Statements {
		a, err := pStmtToAst(st)
		if err != nil {
			return nil, err
		}
		if a != nil {
			out.Statements = append(out.Statements, a)
		}
	}
	return out, nil
}

func pStmtToAst(s Statement) (aast.Statement, error) {
	switch n := s.(type) {
	case *ReturnStatement:
		if n.Value == nil {
			return &aast.ReturnStatement{}, nil
		}
		v, err := pExprToAst(n.Value)
		if err != nil {
			return nil, err
		}
		return &aast.ReturnStatement{Value: v}, nil
	case *ExpressionStatement:
		v, err := pExprToAst(n.Expression)
		if err != nil {
			return nil, err
		}
		return &aast.ExpressionStatement{Expression: v}, nil
	case *BlockStatement:
		return pBlockToAst(n)
	case *WhileStatement:
		cond, err := pExprToAst(n.Condition)
		if err != nil {
			return nil, err
		}
		// Normalize while body to an empty block when condition is a false literal
		if lit, ok := n.Condition.(*Literal); ok {
			if bv, ok2 := lit.Value.(bool); ok2 && !bv {
				return &aast.BlockStatement{}, nil
			}
		}
		var bodyAst *aast.BlockStatement
		if b, ok := n.Body.(*BlockStatement); ok {
			bodyAst, err = pBlockToAst(b)
			if err != nil {
				return nil, err
			}
		} else if n.Body != nil {
			stAst, err := pStmtToAst(n.Body)
			if err != nil {
				return nil, err
			}
			bodyAst = &aast.BlockStatement{}
			if stAst != nil {
				bodyAst.Statements = append(bodyAst.Statements, stAst)
			}
		} else {
			bodyAst = &aast.BlockStatement{}
		}
		return &aast.WhileStatement{Condition: cond, Body: bodyAst}, nil
	default:
		return nil, nil
	}
}

func pExprToAst(e Expression) (aast.Expression, error) {
	switch n := e.(type) {
	case *Identifier:
		return &aast.Identifier{Value: n.Value}, nil
	case *Literal:
		switch v := n.Value.(type) {
		case int:
			return &aast.Literal{Kind: aast.LiteralInteger, Value: int64(v), Raw: fmt.Sprintf("%d", v)}, nil
		case int64:
			return &aast.Literal{Kind: aast.LiteralInteger, Value: v, Raw: fmt.Sprintf("%d", v)}, nil
		case float64:
			return &aast.Literal{Kind: aast.LiteralFloat, Value: v, Raw: fmt.Sprintf("%g", v)}, nil
		case bool:
			return &aast.Literal{Kind: aast.LiteralBoolean, Value: v, Raw: fmt.Sprintf("%t", v)}, nil
		case string:
			return &aast.Literal{Kind: aast.LiteralString, Value: v, Raw: fmt.Sprintf("\"%s\"", v)}, nil
		default:
			return &aast.Literal{Kind: aast.LiteralNull, Value: nil, Raw: "null"}, nil
		}
	case *BinaryExpression:
		l, err := pExprToAst(n.Left)
		if err != nil {
			return nil, err
		}
		r, err := pExprToAst(n.Right)
		if err != nil {
			return nil, err
		}
		op, err := mapOpToAst(n.Operator)
		if err != nil {
			return nil, err
		}
		return &aast.BinaryExpression{Left: l, Operator: op, Right: r}, nil
	default:
		return nil, nil
	}
}

func aBlockToParser(b *aast.BlockStatement) (*BlockStatement, error) {
	if b == nil {
		return &BlockStatement{}, nil
	}
	out := &BlockStatement{Statements: make([]Statement, 0, len(b.Statements))}
	for _, st := range b.Statements {
		p, err := aStmtToParser(st)
		if err != nil {
			return nil, err
		}
		if p != nil {
			if blk, ok := p.(*BlockStatement); ok && len(blk.Statements) == 0 {
				// Skip empty blocks produced by dead code elimination
				continue
			}
			out.Statements = append(out.Statements, p)
		}
	}
	return out, nil
}

func aStmtToParser(s aast.Statement) (Statement, error) {
	switch n := s.(type) {
	case *aast.ReturnStatement:
		if n.Value == nil {
			return &ReturnStatement{}, nil
		}
		v, err := aExprToParser(n.Value)
		if err != nil {
			return nil, err
		}
		return &ReturnStatement{Value: v}, nil
	case *aast.ExpressionStatement:
		v, err := aExprToParser(n.Expression)
		if err != nil {
			return nil, err
		}
		return &ExpressionStatement{Expression: v}, nil
	case *aast.WhileStatement:
		cond, err := aExprToParser(n.Condition)
		if err != nil {
			return nil, err
		}
		body, err := aBlockToParser(n.Body)
		if err != nil {
			return nil, err
		}
		return &WhileStatement{Condition: cond, Body: body}, nil
	case *aast.BlockStatement:
		return aBlockToParser(n)
	default:
		return nil, nil
	}
}

func aExprToParser(e aast.Expression) (Expression, error) {
	switch n := e.(type) {
	case *aast.Identifier:
		return &Identifier{Value: n.Value}, nil
	case *aast.Literal:
		switch n.Kind {
		case aast.LiteralInteger:
			if iv, ok := n.Value.(int64); ok {
				return &Literal{Value: int(iv), Kind: LiteralInteger}, nil
			}
			return &Literal{Value: 0, Kind: LiteralInteger}, nil
		case aast.LiteralFloat:
			if fv, ok := n.Value.(float64); ok {
				return &Literal{Value: fv, Kind: LiteralFloat}, nil
			}
			return &Literal{Value: 0.0, Kind: LiteralFloat}, nil
		case aast.LiteralBoolean:
			if bv, ok := n.Value.(bool); ok {
				return &Literal{Value: bv, Kind: LiteralBool}, nil
			}
			return &Literal{Value: false, Kind: LiteralBool}, nil
		case aast.LiteralString:
			if sv, ok := n.Value.(string); ok {
				return &Literal{Value: sv, Kind: LiteralString}, nil
			}
			return &Literal{Value: "", Kind: LiteralString}, nil
		default:
			return &Literal{Value: nil, Kind: LiteralNull}, nil
		}
	case *aast.BinaryExpression:
		l, err := aExprToParser(n.Left)
		if err != nil {
			return nil, err
		}
		r, err := aExprToParser(n.Right)
		if err != nil {
			return nil, err
		}
		return &BinaryExpression{Left: l, Operator: &Operator{Value: n.Operator.String(), Kind: BinaryOp}, Right: r}, nil
	default:
		return nil, nil
	}
}

func mapOpToAst(op *Operator) (aast.Operator, error) {
	if op == nil {
		return 0, fmt.Errorf("nil operator")
	}
	switch op.Value {
	case "+":
		return aast.OpAdd, nil
	case "-":
		return aast.OpSub, nil
	case "*":
		return aast.OpMul, nil
	case "/":
		return aast.OpDiv, nil
	case "%":
		return aast.OpMod, nil
	case "==":
		return aast.OpEq, nil
	case "!=":
		return aast.OpNe, nil
	case "<":
		return aast.OpLt, nil
	case "<=":
		return aast.OpLe, nil
	case ">":
		return aast.OpGt, nil
	case ">=":
		return aast.OpGe, nil
	case "&&":
		return aast.OpAnd, nil
	case "||":
		return aast.OpOr, nil
	case "!":
		return aast.OpNot, nil
	default:
		return 0, fmt.Errorf("unsupported operator %q", op.Value)
	}
}
