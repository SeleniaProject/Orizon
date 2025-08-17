package parser

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/lexer"
)

func TestParseSliceAndArrayTypes(t *testing.T) {
	input := `
    func f() {
        let a: [int];
        let b: [int; 4];
    }`
	l := lexer.New(input)
	p := NewParser(l, "test_types.oriz")
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	fn := program.Declarations[0].(*FunctionDeclaration)
	if len(fn.Body.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(fn.Body.Statements))
	}
	v1 := fn.Body.Statements[0].(*VariableDeclaration)
	if _, ok := v1.TypeSpec.(*ArrayType); !ok || !v1.TypeSpec.(*ArrayType).IsDynamic {
		t.Fatalf("expected slice ArrayType(IsDynamic), got %T %#v", v1.TypeSpec, v1.TypeSpec)
	}
	v2 := fn.Body.Statements[1].(*VariableDeclaration)
	if at, ok := v2.TypeSpec.(*ArrayType); !ok || at.IsDynamic || at.Size == nil {
		t.Fatalf("expected static ArrayType with size, got %T %#v", v2.TypeSpec, v2.TypeSpec)
	}
}

func TestParseFunctionTypes(t *testing.T) {
	input := `
    func f() {
        let f1: func(int, float) -> bool;
        let f2: async func(x: int) -> void;
        let f3: func();
    }`
	l := lexer.New(input)
	p := NewParser(l, "test_funtypes.oriz")
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	fn := program.Declarations[0].(*FunctionDeclaration)
	if len(fn.Body.Statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(fn.Body.Statements))
	}
	v1 := fn.Body.Statements[0].(*VariableDeclaration)
	if _, ok := v1.TypeSpec.(*FunctionType); !ok {
		t.Fatalf("expected FunctionType for f1, got %T", v1.TypeSpec)
	}
	v2 := fn.Body.Statements[1].(*VariableDeclaration)
	if ft, ok := v2.TypeSpec.(*FunctionType); !ok || !ft.IsAsync {
		t.Fatalf("expected async FunctionType for f2, got %T %#v", v2.TypeSpec, v2.TypeSpec)
	}
	v3 := fn.Body.Statements[2].(*VariableDeclaration)
	if ft, ok := v3.TypeSpec.(*FunctionType); !ok || ft.ReturnType != nil {
		t.Fatalf("expected FunctionType with nil return (implicit unit) for f3, got %T %#v", v3.TypeSpec, v3.TypeSpec)
	}
}

func TestParseGenericTypes(t *testing.T) {
	input := `
    func f() {
        let g: Result<int, string>;
    }`
	l := lexer.New(input)
	p := NewParser(l, "test_generics.oriz")
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	fn := program.Declarations[0].(*FunctionDeclaration)
	if len(fn.Body.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(fn.Body.Statements))
	}
	v := fn.Body.Statements[0].(*VariableDeclaration)
	if _, ok := v.TypeSpec.(*GenericType); !ok {
		t.Fatalf("expected GenericType, got %T", v.TypeSpec)
	}
}

func TestParseReferenceTypes(t *testing.T) {
	input := `
	func f() {
		let r1: &int;
		let r2: &mut float;
		let r3: &Result<int, string>;
		let r4: &'a int;
		let r5: &'b mut Result<int, string>;
	}`
	l := lexer.New(input)
	p := NewParser(l, "test_refs.oriz")
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	fn := program.Declarations[0].(*FunctionDeclaration)
	if len(fn.Body.Statements) != 5 {
		t.Fatalf("expected 5 statements, got %d", len(fn.Body.Statements))
	}
	v1 := fn.Body.Statements[0].(*VariableDeclaration)
	if rt, ok := v1.TypeSpec.(*ReferenceType); !ok || rt.IsMutable || rt.Inner == nil {
		t.Fatalf("expected &T ReferenceType (immutable), got %T %#v", v1.TypeSpec, v1.TypeSpec)
	}
	v2 := fn.Body.Statements[1].(*VariableDeclaration)
	if rt, ok := v2.TypeSpec.(*ReferenceType); !ok || !rt.IsMutable || rt.Inner == nil {
		t.Fatalf("expected &mut T ReferenceType (mutable), got %T %#v", v2.TypeSpec, v2.TypeSpec)
	}
	v3 := fn.Body.Statements[2].(*VariableDeclaration)
	if rt, ok := v3.TypeSpec.(*ReferenceType); !ok {
		t.Fatalf("expected ReferenceType, got %T", v3.TypeSpec)
	} else {
		if _, ok := rt.Inner.(*GenericType); !ok {
			t.Fatalf("expected inner GenericType, got %T", rt.Inner)
		}
	}
	v4 := fn.Body.Statements[3].(*VariableDeclaration)
	if rt, ok := v4.TypeSpec.(*ReferenceType); !ok || rt.Lifetime != "a" {
		t.Fatalf("expected &'a T ReferenceType, got %T %#v", v4.TypeSpec, v4.TypeSpec)
	}
	v5 := fn.Body.Statements[4].(*VariableDeclaration)
	if rt, ok := v5.TypeSpec.(*ReferenceType); !ok || rt.Lifetime != "b" || !rt.IsMutable {
		t.Fatalf("expected &'b mut T ReferenceType, got %T %#v", v5.TypeSpec, v5.TypeSpec)
	}
}

func TestParsePointerTypes(t *testing.T) {
	input := `
	func f() {
		let p1: *int;
		let p2: *mut float;
		let p3: *Result<int, string>;
	}`
	l := lexer.New(input)
	p := NewParser(l, "test_ptrs.oriz")
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	fn := program.Declarations[0].(*FunctionDeclaration)
	if len(fn.Body.Statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(fn.Body.Statements))
	}
	v1 := fn.Body.Statements[0].(*VariableDeclaration)
	if pt, ok := v1.TypeSpec.(*PointerType); !ok || pt.IsMutable || pt.Inner == nil {
		t.Fatalf("expected *T PointerType (immutable), got %T %#v", v1.TypeSpec, v1.TypeSpec)
	}
	v2 := fn.Body.Statements[1].(*VariableDeclaration)
	if pt, ok := v2.TypeSpec.(*PointerType); !ok || !pt.IsMutable || pt.Inner == nil {
		t.Fatalf("expected *mut T PointerType (mutable), got %T %#v", v2.TypeSpec, v2.TypeSpec)
	}
	v3 := fn.Body.Statements[2].(*VariableDeclaration)
	if pt, ok := v3.TypeSpec.(*PointerType); !ok {
		t.Fatalf("expected PointerType, got %T", v3.TypeSpec)
	} else {
		if _, ok := pt.Inner.(*GenericType); !ok {
			t.Fatalf("expected inner GenericType, got %T", pt.Inner)
		}
	}
}
