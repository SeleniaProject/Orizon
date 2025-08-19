package astbridge

import (
	"testing"

	aast "github.com/orizon-lang/orizon/internal/ast"
	p "github.com/orizon-lang/orizon/internal/parser"
)

func TestDeclarations_Struct_RoundTrip(t *testing.T) {
	pprog := &p.Program{Declarations: []p.Declaration{
		&p.StructDeclaration{
			Name: &p.Identifier{Value: "Point"},
			Fields: []*p.StructField{
				{Name: &p.Identifier{Value: "x"}, Type: &p.BasicType{Name: "int"}, IsPublic: true},
				{Name: &p.Identifier{Value: "y"}, Type: &p.BasicType{Name: "int"}, IsPublic: false},
			},
			Generics: []*p.GenericParameter{
				{Kind: p.GenericParamType, Name: &p.Identifier{Value: "T"}},
			},
			IsPublic: true,
		},
	}}

	ap, err := FromParserProgram(pprog)
	if err != nil {
		t.Fatalf("FromParserProgram error: %v", err)
	}
	if len(ap.Declarations) != 1 {
		t.Fatalf("expected 1 decl")
	}
	sd, ok := ap.Declarations[0].(*aast.StructDeclaration)
	if !ok {
		t.Fatalf("expected StructDeclaration, got %T", ap.Declarations[0])
	}
	if sd.Name.Value != "Point" || !sd.IsExported {
		t.Fatalf("struct name/export mismatch: %#v", sd)
	}
	if len(sd.Fields) != 2 || !sd.Fields[0].IsPublic || sd.Fields[1].IsPublic {
		t.Fatalf("fields/public flags mismatch: %#v", sd.Fields)
	}
	if len(sd.Generics) != 1 || sd.Generics[0].Kind != aast.GenericParamType || sd.Generics[0].Name.Value != "T" {
		t.Fatalf("generics mismatch: %#v", sd.Generics)
	}

	pback, err := ToParserProgram(ap)
	if err != nil {
		t.Fatalf("ToParserProgram error: %v", err)
	}
	bsd, ok := pback.Declarations[0].(*p.StructDeclaration)
	if !ok {
		t.Fatalf("expected parser StructDeclaration, got %T", pback.Declarations[0])
	}
	if bsd.Name.Value != "Point" || len(bsd.Fields) != 2 || !bsd.Fields[0].IsPublic || bsd.Fields[1].IsPublic {
		t.Fatalf("round-trip struct mismatch: %#v", bsd)
	}
}

func TestDeclarations_Enum_RoundTrip(t *testing.T) {
	pprog := &p.Program{Declarations: []p.Declaration{
		&p.EnumDeclaration{
			Name:     &p.Identifier{Value: "Option"},
			Generics: []*p.GenericParameter{{Kind: p.GenericParamType, Name: &p.Identifier{Value: "T"}}},
			Variants: []*p.EnumVariant{
				{Name: &p.Identifier{Value: "Some"}, Fields: []*p.StructField{{Name: &p.Identifier{Value: "value"}, Type: &p.BasicType{Name: "T"}}}},
				{Name: &p.Identifier{Value: "None"}},
			},
			IsPublic: true,
		},
	}}
	ap, err := FromParserProgram(pprog)
	if err != nil {
		t.Fatalf("FromParserProgram error: %v", err)
	}
	en, ok := ap.Declarations[0].(*aast.EnumDeclaration)
	if !ok {
		t.Fatalf("expected EnumDeclaration, got %T", ap.Declarations[0])
	}
	if en.Name.Value != "Option" || len(en.Variants) != 2 {
		t.Fatalf("enum shape mismatch: %#v", en)
	}

	pback, err := ToParserProgram(ap)
	if err != nil {
		t.Fatalf("ToParserProgram error: %v", err)
	}
	ben, ok := pback.Declarations[0].(*p.EnumDeclaration)
	if !ok || ben.Name.Value != "Option" || len(ben.Variants) != 2 {
		t.Fatalf("round-trip enum mismatch: %#v", pback.Declarations[0])
	}
}

func TestDeclarations_Trait_RoundTrip(t *testing.T) {
	pprog := &p.Program{Declarations: []p.Declaration{
		&p.TraitDeclaration{
			Name: &p.Identifier{Value: "Iterable"},
			Methods: []*p.TraitMethod{
				{Name: &p.Identifier{Value: "next"}, ReturnType: &p.BasicType{Name: "int"}, Parameters: []*p.Parameter{}},
			},
			AssociatedTypes: []*p.AssociatedType{
				{Name: &p.Identifier{Value: "Item"}, Bounds: []p.Type{&p.BasicType{Name: "Clone"}}},
			},
			IsPublic: true,
		},
	}}
	ap, err := FromParserProgram(pprog)
	if err != nil {
		t.Fatalf("FromParserProgram error: %v", err)
	}
	tr, ok := ap.Declarations[0].(*aast.TraitDeclaration)
	if !ok {
		t.Fatalf("expected TraitDeclaration, got %T", ap.Declarations[0])
	}
	if tr.Name.Value != "Iterable" || len(tr.Methods) != 1 || len(tr.AssociatedTypes) != 1 {
		t.Fatalf("trait shape mismatch: %#v", tr)
	}

	pback, err := ToParserProgram(ap)
	if err != nil {
		t.Fatalf("ToParserProgram error: %v", err)
	}
	btr, ok := pback.Declarations[0].(*p.TraitDeclaration)
	if !ok || btr.Name.Value != "Iterable" || len(btr.Methods) != 1 {
		t.Fatalf("round-trip trait mismatch: %#v", pback.Declarations[0])
	}
}

func TestDeclarations_Impl_RoundTrip(t *testing.T) {
	pprog := &p.Program{Declarations: []p.Declaration{
		&p.ImplBlock{
			Trait:        &p.BasicType{Name: "Iterable"},
			ForType:      &p.BasicType{Name: "Vec"},
			Items:        []*p.FunctionDeclaration{{Name: &p.Identifier{Value: "len"}, ReturnType: &p.BasicType{Name: "int"}, Parameters: []*p.Parameter{}, Body: &p.BlockStatement{}}},
			Generics:     []*p.GenericParameter{{Kind: p.GenericParamType, Name: &p.Identifier{Value: "T"}}},
			WhereClauses: []*p.WherePredicate{{Target: &p.BasicType{Name: "T"}, Bounds: []p.Type{&p.BasicType{Name: "Clone"}}}},
		},
	}}
	ap, err := FromParserProgram(pprog)
	if err != nil {
		t.Fatalf("FromParserProgram error: %v", err)
	}
	im, ok := ap.Declarations[0].(*aast.ImplDeclaration)
	if !ok {
		t.Fatalf("expected ImplDeclaration, got %T", ap.Declarations[0])
	}
	if im.Trait == nil || im.ForType == nil || len(im.Methods) != 1 || len(im.Generics) != 1 || len(im.WhereClauses) != 1 {
		t.Fatalf("impl shape mismatch: %#v", im)
	}

	pback, err := ToParserProgram(ap)
	if err != nil {
		t.Fatalf("ToParserProgram error: %v", err)
	}
	bim, ok := pback.Declarations[0].(*p.ImplBlock)
	if !ok || bim.Trait == nil || bim.ForType == nil || len(bim.Items) != 1 {
		t.Fatalf("round-trip impl mismatch: %#v", pback.Declarations[0])
	}
}
