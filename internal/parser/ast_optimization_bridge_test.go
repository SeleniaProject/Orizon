package parser

import "testing"

func pos(file string, line, col, off int) Position {
	return Position{File: file, Line: line, Column: col, Offset: off}
}
func spanAt(file string, line, col int) Span {
	return Span{Start: pos(file, line, col, 0), End: pos(file, line, col+1, 1)}
}

func TestOptimizeViaAstPipe_ConstantFolding_ReturnLiteral(t *testing.T) {
	file := "bridge_cf.oriz"

	// return 2 * 3
	ret := &ReturnStatement{
		Span: spanAt(file, 2, 3),
		Value: &BinaryExpression{
			Span:     spanAt(file, 2, 10),
			Left:     &Literal{Span: spanAt(file, 2, 10), Value: 2, Kind: LiteralInteger},
			Operator: &Operator{Span: spanAt(file, 2, 12), Value: "*", Kind: BinaryOp},
			Right:    &Literal{Span: spanAt(file, 2, 14), Value: 3, Kind: LiteralInteger},
		},
	}

	fn := &FunctionDeclaration{
		Span:       spanAt(file, 1, 1),
		Name:       &Identifier{Span: spanAt(file, 1, 5), Value: "main"},
		Parameters: []*Parameter{},
		ReturnType: &BasicType{Span: spanAt(file, 1, 12), Name: "int"},
		Body:       &BlockStatement{Span: spanAt(file, 1, 16), Statements: []Statement{ret}},
		IsPublic:   true,
	}

	prog := &Program{Span: spanAt(file, 1, 1), Declarations: []Declaration{fn}}

	optimized, err := OptimizeViaAstPipe(prog, "default")
	if err != nil {
		t.Fatalf("OptimizeViaAstPipe error: %v", err)
	}
	if len(optimized.Declarations) != 1 {
		t.Fatalf("expected 1 decl, got %d", len(optimized.Declarations))
	}
	ofn, ok := optimized.Declarations[0].(*FunctionDeclaration)
	if !ok {
		t.Fatalf("expected FunctionDeclaration, got %T", optimized.Declarations[0])
	}
	if len(ofn.Body.Statements) != 1 {
		t.Fatalf("expected 1 statement in body, got %d", len(ofn.Body.Statements))
	}
	oret, ok := ofn.Body.Statements[0].(*ReturnStatement)
	if !ok {
		t.Fatalf("expected ReturnStatement, got %T", ofn.Body.Statements[0])
	}
	olit, ok := oret.Value.(*Literal)
	if !ok {
		t.Fatalf("expected Literal after folding, got %T", oret.Value)
	}
	if olit.Kind != LiteralInteger {
		t.Fatalf("expected integer literal, got kind=%v", olit.Kind)
	}
	// Value is int (parser literal)
	if v, ok := olit.Value.(int); !ok || v != 6 {
		t.Fatalf("expected value 6, got %#v", olit.Value)
	}
}

func TestOptimizeViaAstPipe_DeadCode_WhileFalseRemoved(t *testing.T) {
	file := "bridge_dc.oriz"

	// while (false) { 1 }  -> removed
	wl := &WhileStatement{
		Span:      spanAt(file, 2, 1),
		Condition: &Literal{Span: spanAt(file, 2, 8), Value: false, Kind: LiteralBool},
		Body:      &ExpressionStatement{Span: spanAt(file, 2, 15), Expression: &Literal{Span: spanAt(file, 2, 15), Value: 1, Kind: LiteralInteger}},
	}

	fn := &FunctionDeclaration{
		Span:       spanAt(file, 1, 1),
		Name:       &Identifier{Span: spanAt(file, 1, 5), Value: "main"},
		Parameters: []*Parameter{},
		ReturnType: &BasicType{Span: spanAt(file, 1, 12), Name: "void"},
		Body:       &BlockStatement{Span: spanAt(file, 1, 16), Statements: []Statement{wl}},
		IsPublic:   false,
	}

	prog := &Program{Span: spanAt(file, 1, 1), Declarations: []Declaration{fn}}

	optimized, err := OptimizeViaAstPipe(prog, "aggressive")
	if err != nil {
		t.Fatalf("OptimizeViaAstPipe error: %v", err)
	}

	ofn, ok := optimized.Declarations[0].(*FunctionDeclaration)
	if !ok {
		t.Fatalf("expected FunctionDeclaration, got %T", optimized.Declarations[0])
	}
	// While(false) should be removed by dead code elimination -> empty block
	if len(ofn.Body.Statements) != 0 {
		t.Fatalf("expected while(false) removed, statements=%d", len(ofn.Body.Statements))
	}
}
