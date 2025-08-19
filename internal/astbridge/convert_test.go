package astbridge

import (
	"testing"

	aast "github.com/orizon-lang/orizon/internal/ast"
	p "github.com/orizon-lang/orizon/internal/parser"
)

func TestFromParserType_MappingAndPreservation(t *testing.T) {
	// Helper to build a simple program with one var decl
	mkProg := func(name string, typ p.Type) *p.Program {
		return &p.Program{Declarations: []p.Declaration{
			&p.VariableDeclaration{Name: &p.Identifier{Value: name}, TypeSpec: typ},
		}}
	}

	// 1) Known basic type -> ast.BasicType
	prog1 := mkProg("a", &p.BasicType{Name: "int"})
	a1, err := FromParserProgram(prog1)
	if err != nil {
		t.Fatalf("FromParserProgram error: %v", err)
	}
	v1, _ := a1.Declarations[0].(*aast.VariableDeclaration)
	if _, ok := v1.Type.(*aast.BasicType); !ok {
		t.Fatalf("expected ast.BasicType for 'int', got %T", v1.Type)
	}

	// 2) Unknown basic name should not default to BasicInt; keep as IdentifierType
	prog2 := mkProg("b", &p.BasicType{Name: "MyType"})
	a2, err := FromParserProgram(prog2)
	if err != nil {
		t.Fatalf("FromParserProgram error: %v", err)
	}
	v2, _ := a2.Declarations[0].(*aast.VariableDeclaration)
	if id, ok := v2.Type.(*aast.IdentifierType); !ok || id.Name.Value != "MyType" {
		t.Fatalf("expected IdentifierType 'MyType', got %T %#v", v2.Type, v2.Type)
	}

	// 3) Reference type with lifetime and mut should preserve textual form
	rt := &p.ReferenceType{Inner: &p.BasicType{Name: "int"}, IsMutable: true, Lifetime: "a"}
	prog3 := mkProg("c", rt)
	a3, err := FromParserProgram(prog3)
	if err != nil {
		t.Fatalf("FromParserProgram error: %v", err)
	}
	v3, _ := a3.Declarations[0].(*aast.VariableDeclaration)
	if id, ok := v3.Type.(*aast.IdentifierType); !ok || id.Name.Value != "&'a mut int" {
		t.Fatalf("expected IdentifierType %q, got %T %v", "&'a mut int", v3.Type, v3.Type)
	}

	// 4) Pointer type with mut
	pt := &p.PointerType{Inner: &p.BasicType{Name: "float"}, IsMutable: true}
	prog4 := mkProg("d", pt)
	a4, err := FromParserProgram(prog4)
	if err != nil {
		t.Fatalf("FromParserProgram error: %v", err)
	}
	v4, _ := a4.Declarations[0].(*aast.VariableDeclaration)
	if id, ok := v4.Type.(*aast.IdentifierType); !ok || id.Name.Value != "*mut float" {
		t.Fatalf("expected IdentifierType '*mut float', got %T %v", v4.Type, v4.Type)
	}

	// 5) Array type [int; 3]
	at := &p.ArrayType{ElementType: &p.BasicType{Name: "int"}, Size: &p.Literal{Value: 3}, IsDynamic: false}
	prog5 := mkProg("e", at)
	a5, err := FromParserProgram(prog5)
	if err != nil {
		t.Fatalf("FromParserProgram error: %v", err)
	}
	v5, _ := a5.Declarations[0].(*aast.VariableDeclaration)
	if id, ok := v5.Type.(*aast.IdentifierType); !ok || id.Name.Value != "[int; 3]" {
		t.Fatalf("expected IdentifierType '[int; 3]', got %T %v", v5.Type, v5.Type)
	}

	// 6) Generic type Result<int, string>
	gt := &p.GenericType{BaseType: &p.BasicType{Name: "Result"}, TypeParameters: []p.Type{
		&p.BasicType{Name: "int"}, &p.BasicType{Name: "string"},
	}}
	prog6 := mkProg("f", gt)
	a6, err := FromParserProgram(prog6)
	if err != nil {
		t.Fatalf("FromParserProgram error: %v", err)
	}
	v6, _ := a6.Declarations[0].(*aast.VariableDeclaration)
	if id, ok := v6.Type.(*aast.IdentifierType); !ok || id.Name.Value != "Result<int, string>" {
		t.Fatalf("expected IdentifierType 'Result<int, string>', got %T %v", v6.Type, v6.Type)
	}

	// 7) Function type (x: int, y: *mut float) -> &'a int
	ft := &p.FunctionType{Parameters: []*p.FunctionTypeParameter{
		{Name: "x", Type: &p.BasicType{Name: "int"}},
		{Name: "y", Type: &p.PointerType{Inner: &p.BasicType{Name: "float"}, IsMutable: true}},
	}, ReturnType: &p.ReferenceType{Inner: &p.BasicType{Name: "int"}, Lifetime: "a"}}
	prog7 := mkProg("g", ft)
	a7, err := FromParserProgram(prog7)
	if err != nil {
		t.Fatalf("FromParserProgram error: %v", err)
	}
	v7, _ := a7.Declarations[0].(*aast.VariableDeclaration)
	if id, ok := v7.Type.(*aast.IdentifierType); !ok || id.Name.Value != "(x: int, y: *mut float) -> &'a int" {
		t.Fatalf("expected IdentifierType %q, got %T %v", "(x: int, y: *mut float) -> &'a int", v7.Type, v7.Type)
	}
}

func TestToParserType_IdentifierRoundTrip(t *testing.T) {
	// ast.IdentifierType should map back to parser.BasicType with same name string
	astProg := &aast.Program{Declarations: []aast.Declaration{
		&aast.VariableDeclaration{Name: &aast.Identifier{Value: "x"}, Type: &aast.IdentifierType{Name: &aast.Identifier{Value: "&'a mut int"}}},
	}}
	pprog, err := ToParserProgram(astProg)
	if err != nil {
		t.Fatalf("ToParserProgram error: %v", err)
	}
	v, _ := pprog.Declarations[0].(*p.VariableDeclaration)
	if rt, ok := v.TypeSpec.(*p.ReferenceType); !ok || !rt.IsMutable || rt.Lifetime != "a" {
		t.Fatalf("expected parser.ReferenceType with lifetime 'a and mut, got %T %#v", v.TypeSpec, v.TypeSpec)
	}
}

func TestToParserType_PointerMutRoundTrip(t *testing.T) {
	astProg := &aast.Program{Declarations: []aast.Declaration{
		&aast.VariableDeclaration{Name: &aast.Identifier{Value: "y"}, Type: &aast.IdentifierType{Name: &aast.Identifier{Value: "*mut float"}}},
	}}
	pprog, err := ToParserProgram(astProg)
	if err != nil {
		t.Fatalf("ToParserProgram error: %v", err)
	}
	v, _ := pprog.Declarations[0].(*p.VariableDeclaration)
	if pt, ok := v.TypeSpec.(*p.PointerType); !ok || !pt.IsMutable {
		t.Fatalf("expected parser.PointerType(mut), got %T %#v", v.TypeSpec, v.TypeSpec)
	}
}

func TestDeclarations_TypeAlias_And_Newtype_RoundTrip(t *testing.T) {
	// Parser -> AST(TypeDeclaration alias) -> Parser
	{
		pprog := &p.Program{Declarations: []p.Declaration{
			&p.TypeAliasDeclaration{Name: &p.Identifier{Value: "MyInt"}, Aliased: &p.BasicType{Name: "i32"}, IsPublic: true},
		}}
		ap, err := FromParserProgram(pprog)
		if err != nil {
			t.Fatalf("FromParserProgram error: %v", err)
		}
		if len(ap.Declarations) != 1 {
			t.Fatalf("expected 1 decl")
		}
		td, ok := ap.Declarations[0].(*aast.TypeDeclaration)
		if !ok || !td.IsAlias || !td.IsExported || td.Name.Value != "MyInt" {
			t.Fatalf("unexpected TypeDeclaration alias: %#v", ap.Declarations[0])
		}
		// back to parser
		pback, err := ToParserProgram(ap)
		if err != nil {
			t.Fatalf("ToParserProgram error: %v", err)
		}
		if len(pback.Declarations) != 1 {
			t.Fatalf("expected 1 decl back")
		}
		if _, ok := pback.Declarations[0].(*p.TypeAliasDeclaration); !ok {
			t.Fatalf("expected TypeAliasDeclaration back, got %T", pback.Declarations[0])
		}
	}

	// Parser -> AST(TypeDeclaration newtype) -> Parser
	{
		pprog := &p.Program{Declarations: []p.Declaration{
			&p.NewtypeDeclaration{Name: &p.Identifier{Value: "UserId"}, Base: &p.BasicType{Name: "i64"}},
		}}
		ap, err := FromParserProgram(pprog)
		if err != nil {
			t.Fatalf("FromParserProgram error: %v", err)
		}
		if len(ap.Declarations) != 1 {
			t.Fatalf("expected 1 decl")
		}
		td, ok := ap.Declarations[0].(*aast.TypeDeclaration)
		if !ok || td.IsAlias || td.Name.Value != "UserId" {
			t.Fatalf("unexpected TypeDeclaration newtype: %#v", ap.Declarations[0])
		}
		// back to parser
		pback, err := ToParserProgram(ap)
		if err != nil {
			t.Fatalf("ToParserProgram error: %v", err)
		}
		if len(pback.Declarations) != 1 {
			t.Fatalf("expected 1 decl back")
		}
		if _, ok := pback.Declarations[0].(*p.NewtypeDeclaration); !ok {
			t.Fatalf("expected NewtypeDeclaration back, got %T", pback.Declarations[0])
		}
	}
}

func TestFromParserType_DynamicArray_And_AsyncFn_Pretty(t *testing.T) {
	// Dynamic array [T]
	{
		prog := &p.Program{Declarations: []p.Declaration{
			&p.VariableDeclaration{Name: &p.Identifier{Value: "arr"}, TypeSpec: &p.ArrayType{ElementType: &p.BasicType{Name: "int"}, IsDynamic: true}},
		}}
		ap, err := FromParserProgram(prog)
		if err != nil {
			t.Fatalf("FromParserProgram error: %v", err)
		}
		v, _ := ap.Declarations[0].(*aast.VariableDeclaration)
		if id, ok := v.Type.(*aast.IdentifierType); !ok || id.Name.Value != "[int]" {
			t.Fatalf("expected '[int]', got %T %#v", v.Type, v.Type)
		}
	}

	// async function type with unnamed param: async (int) -> void
	{
		ft := &p.FunctionType{IsAsync: true, Parameters: []*p.FunctionTypeParameter{{Name: "", Type: &p.BasicType{Name: "int"}}}, ReturnType: nil}
		prog := &p.Program{Declarations: []p.Declaration{
			&p.VariableDeclaration{Name: &p.Identifier{Value: "f"}, TypeSpec: ft},
		}}
		ap, err := FromParserProgram(prog)
		if err != nil {
			t.Fatalf("FromParserProgram error: %v", err)
		}
		v, _ := ap.Declarations[0].(*aast.VariableDeclaration)
		if id, ok := v.Type.(*aast.IdentifierType); !ok || id.Name.Value != "async (int) -> void" {
			t.Fatalf("expected 'async (int) -> void', got %T %#v", v.Type, v.Type)
		}
	}
}

func TestToParserType_GenericAndArray_RoundTrip(t *testing.T) {
	// Generic Vec<Result<int, string>>
	{
		astProg := &aast.Program{Declarations: []aast.Declaration{
			&aast.VariableDeclaration{Name: &aast.Identifier{Value: "v"}, Type: &aast.IdentifierType{Name: &aast.Identifier{Value: "Vec<Result<int, string>>"}}},
		}}
		pprog, err := ToParserProgram(astProg)
		if err != nil {
			t.Fatalf("ToParserProgram error: %v", err)
		}
		v, _ := pprog.Declarations[0].(*p.VariableDeclaration)
		if gt, ok := v.TypeSpec.(*p.GenericType); !ok {
			t.Fatalf("expected parser.GenericType, got %T", v.TypeSpec)
		} else {
			// base should be BasicType("Vec") and first param GenericType base BasicType("Result")
			if b, ok := gt.BaseType.(*p.BasicType); !ok || b.Name != "Vec" {
				t.Fatalf("expected base Vec, got %#v", gt.BaseType)
			}
			if len(gt.TypeParameters) != 1 {
				t.Fatalf("expected 1 type param, got %d", len(gt.TypeParameters))
			}
		}
	}

	// Dynamic array [float]
	{
		astProg := &aast.Program{Declarations: []aast.Declaration{
			&aast.VariableDeclaration{Name: &aast.Identifier{Value: "a"}, Type: &aast.IdentifierType{Name: &aast.Identifier{Value: "[float]"}}},
		}}
		pprog, err := ToParserProgram(astProg)
		if err != nil {
			t.Fatalf("ToParserProgram error: %v", err)
		}
		v, _ := pprog.Declarations[0].(*p.VariableDeclaration)
		if at, ok := v.TypeSpec.(*p.ArrayType); !ok || !at.IsDynamic {
			t.Fatalf("expected dynamic parser.ArrayType, got %T %#v", v.TypeSpec, v.TypeSpec)
		}
	}
}

func TestDeclarations_Import_And_Export_RoundTrip(t *testing.T) {
	pImport := &p.ImportDeclaration{
		Path:     []*p.Identifier{{Value: "foo"}, {Value: "bar"}},
		Alias:    &p.Identifier{Value: "fb"},
		IsPublic: true,
	}
	pExport := &p.ExportDeclaration{
		Items: []*p.ExportItem{
			{Name: &p.Identifier{Value: "A"}},
			{Name: &p.Identifier{Value: "B"}, Alias: &p.Identifier{Value: "Bee"}},
		},
	}

	// Create ASTBridge and test its declaration conversion methods
	bridge := NewASTBridge()

	ad1, err := bridge.declarationConverter.FromParserDeclaration(pImport)
	if err != nil {
		t.Fatalf("FromParserDeclaration import err: %v", err)
	}
	ad2, err := bridge.declarationConverter.FromParserDeclaration(pExport)
	if err != nil {
		t.Fatalf("FromParserDeclaration export err: %v", err)
	}

	rd1, err := bridge.declarationConverter.ToParserDeclaration(ad1)
	if err != nil {
		t.Fatalf("toParserDecl import err: %v", err)
	}
	rd2, err := bridge.declarationConverter.ToParserDeclaration(ad2)
	if err != nil {
		t.Fatalf("toParserDecl export err: %v", err)
	}

	ri, ok := rd1.(*p.ImportDeclaration)
	if !ok {
		t.Fatalf("expected *ImportDeclaration, got %T", rd1)
	}
	if len(ri.Path) != 2 || ri.Path[0].Value != "foo" || ri.Path[1].Value != "bar" {
		t.Fatalf("import path mismatch: %#v", ri.Path)
	}
	if ri.Alias == nil || ri.Alias.Value != "fb" {
		t.Fatalf("import alias mismatch: %#v", ri.Alias)
	}
	if !ri.IsPublic {
		t.Fatalf("import IsPublic lost")
	}

	re, ok := rd2.(*p.ExportDeclaration)
	if !ok {
		t.Fatalf("expected *ExportDeclaration, got %T", rd2)
	}
	if len(re.Items) != 2 {
		t.Fatalf("export items length mismatch: %d", len(re.Items))
	}
	if re.Items[0].Name.Value != "A" {
		t.Fatalf("export item0 name mismatch: %s", re.Items[0].Name.Value)
	}
	if re.Items[1].Name.Value != "B" {
		t.Fatalf("export item1 name mismatch: %s", re.Items[1].Name.Value)
	}
	if re.Items[1].Alias == nil || re.Items[1].Alias.Value != "Bee" {
		t.Fatalf("export item1 alias mismatch: %#v", re.Items[1].Alias)
	}
}
