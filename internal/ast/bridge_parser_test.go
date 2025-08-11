package ast

import (
    "testing"

    p "github.com/orizon-lang/orizon/internal/parser"
)

func pPos(file string, line, col, off int) p.Position {
    return p.Position{File: file, Line: line, Column: col, Offset: off}
}

func pSpanAt(file string, line, col int) p.Span {
    start := pPos(file, line, col, 0)
    end := pPos(file, line, col+1, 1)
    return p.Span{Start: start, End: end}
}

func TestBridge_FromParser_ToAst_FunctionReturnBinOp(t *testing.T) {
    file := "test.oriz"

    // Build parser AST: fn main(): int { return 1 + 2 }
    fn := &p.FunctionDeclaration{
        Span: pSpanAt(file, 1, 1),
        Name: &p.Identifier{Span: pSpanAt(file, 1, 5), Value: "main"},
        Parameters: []*p.Parameter{},
        ReturnType: &p.BasicType{Span: pSpanAt(file, 1, 12), Name: "int"},
        Body: &p.BlockStatement{Span: pSpanAt(file, 1, 16), Statements: []p.Statement{
            &p.ReturnStatement{Span: pSpanAt(file, 2, 3), Value: &p.BinaryExpression{
                Span:     pSpanAt(file, 2, 10),
                Left:     &p.Literal{Span: pSpanAt(file, 2, 10), Value: 1, Kind: p.LiteralInteger},
                Operator: &p.Operator{Span: pSpanAt(file, 2, 12), Value: "+", Kind: p.BinaryOp},
                Right:    &p.Literal{Span: pSpanAt(file, 2, 14), Value: 2, Kind: p.LiteralInteger},
            }},
        }},
        IsPublic: true,
    }
    prog := &p.Program{Span: pSpanAt(file, 1, 1), Declarations: []p.Declaration{fn}}

    aProg, err := FromParserProgram(prog)
    if err != nil {
        t.Fatalf("FromParserProgram error: %v", err)
    }
    if aProg == nil {
        t.Fatalf("FromParserProgram returned nil")
    }
    if len(aProg.Declarations) != 1 {
        t.Fatalf("expected 1 declaration, got %d", len(aProg.Declarations))
    }

    aFn, ok := aProg.Declarations[0].(*FunctionDeclaration)
    if !ok {
        t.Fatalf("expected FunctionDeclaration, got %T", aProg.Declarations[0])
    }
    if aFn.Name == nil || aFn.Name.Value != "main" {
        t.Fatalf("unexpected function name: %#v", aFn.Name)
    }
    if aFn.Body == nil || len(aFn.Body.Statements) != 1 {
        t.Fatalf("unexpected function body statements: %v", aFn.Body)
    }
    ret, ok := aFn.Body.Statements[0].(*ReturnStatement)
    if !ok {
        t.Fatalf("expected ReturnStatement, got %T", aFn.Body.Statements[0])
    }
    bin, ok := ret.Value.(*BinaryExpression)
    if !ok {
        t.Fatalf("expected BinaryExpression, got %T", ret.Value)
    }
    if bin.Operator != OpAdd {
        t.Fatalf("expected OpAdd, got %v", bin.Operator)
    }
}

func TestBridge_RoundTrip_Program(t *testing.T) {
    file := "roundtrip.oriz"

    // let x: int = 3; fn add(a:int, b:int): int { return a + b }
    v := &p.VariableDeclaration{
        Span:        pSpanAt(file, 1, 1),
        Name:        &p.Identifier{Span: pSpanAt(file, 1, 5), Value: "x"},
        TypeSpec:    &p.BasicType{Span: pSpanAt(file, 1, 8), Name: "int"},
        Initializer: &p.Literal{Span: pSpanAt(file, 1, 15), Value: 3, Kind: p.LiteralInteger},
        IsMutable:   false,
        IsPublic:    false,
    }
    add := &p.FunctionDeclaration{
        Span: pSpanAt(file, 2, 1),
        Name: &p.Identifier{Span: pSpanAt(file, 2, 5), Value: "add"},
        Parameters: []*p.Parameter{
            {Span: pSpanAt(file, 2, 9), Name: &p.Identifier{Span: pSpanAt(file, 2, 9), Value: "a"}, TypeSpec: &p.BasicType{Span: pSpanAt(file, 2, 12), Name: "int"}},
            {Span: pSpanAt(file, 2, 18), Name: &p.Identifier{Span: pSpanAt(file, 2, 18), Value: "b"}, TypeSpec: &p.BasicType{Span: pSpanAt(file, 2, 21), Name: "int"}},
        },
        ReturnType: &p.BasicType{Span: pSpanAt(file, 2, 27), Name: "int"},
        Body: &p.BlockStatement{Span: pSpanAt(file, 2, 31), Statements: []p.Statement{
            &p.ReturnStatement{Span: pSpanAt(file, 3, 3), Value: &p.BinaryExpression{
                Span:     pSpanAt(file, 3, 10),
                Left:     &p.Identifier{Span: pSpanAt(file, 3, 10), Value: "a"},
                Operator: &p.Operator{Span: pSpanAt(file, 3, 12), Value: "+", Kind: p.BinaryOp},
                Right:    &p.Identifier{Span: pSpanAt(file, 3, 14), Value: "b"},
            }},
        }},
        IsPublic: true,
    }
    prog := &p.Program{Span: pSpanAt(file, 1, 1), Declarations: []p.Declaration{v, add}}

    aProg, err := FromParserProgram(prog)
    if err != nil {
        t.Fatalf("FromParserProgram error: %v", err)
    }

    back, err := ToParserProgram(aProg)
    if err != nil {
        t.Fatalf("ToParserProgram error: %v", err)
    }

    if len(back.Declarations) != 2 {
        t.Fatalf("expected 2 decls after roundtrip, got %d", len(back.Declarations))
    }

    // Validate variable decl
    vb, ok := back.Declarations[0].(*p.VariableDeclaration)
    if !ok {
        t.Fatalf("expected VariableDeclaration, got %T", back.Declarations[0])
    }
    if vb.Name == nil || vb.Name.Value != "x" {
        t.Fatalf("unexpected var name: %#v", vb.Name)
    }
    if bt, ok := vb.TypeSpec.(*p.BasicType); !ok || bt.Name != "int" {
        t.Fatalf("unexpected var type: %#v", vb.TypeSpec)
    }
    if lit, ok := vb.Initializer.(*p.Literal); !ok || lit.Kind != p.LiteralInteger {
        t.Fatalf("unexpected initializer: %#v", vb.Initializer)
    }

    // Validate function
    fb, ok := back.Declarations[1].(*p.FunctionDeclaration)
    if !ok {
        t.Fatalf("expected FunctionDeclaration, got %T", back.Declarations[1])
    }
    if fb.Name == nil || fb.Name.Value != "add" {
        t.Fatalf("unexpected fn name: %#v", fb.Name)
    }
    if fb.Body == nil || len(fb.Body.Statements) != 1 {
        t.Fatalf("unexpected fn body: %#v", fb.Body)
    }
    r, ok := fb.Body.Statements[0].(*p.ReturnStatement)
    if !ok {
        t.Fatalf("expected ReturnStatement, got %T", fb.Body.Statements[0])
    }
    be, ok := r.Value.(*p.BinaryExpression)
    if !ok {
        t.Fatalf("expected BinaryExpression, got %T", r.Value)
    }
    if be.Operator == nil || be.Operator.Value != "+" {
        t.Fatalf("unexpected operator: %#v", be.Operator)
    }
}

func TestBridge_Span_Mapping(t *testing.T) {
    file := "span.oriz"
    sp := pSpanAt(file, 10, 20)
    pp := &p.Program{Span: sp, Declarations: []p.Declaration{}}
    a, err := FromParserProgram(pp)
    if err != nil {
        t.Fatalf("FromParserProgram error: %v", err)
    }
    if a.Span.Start.Line != 10 || a.Span.Start.Column != 20 {
        t.Fatalf("span not mapped: %#v", a.Span)
    }
    back, err := ToParserProgram(a)
    if err != nil {
        t.Fatalf("ToParserProgram error: %v", err)
    }
    if back.Span.Start.Line != 10 || back.Span.Start.Column != 20 || back.Span.Start.File != file {
        t.Fatalf("span not round-tripped: %#v", back.Span)
    }
}


