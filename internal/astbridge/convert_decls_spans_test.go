package astbridge

import (
	"testing"

	aast "github.com/orizon-lang/orizon/internal/ast"
	p "github.com/orizon-lang/orizon/internal/parser"
)

func TestSpan_Struct_Field_And_Generic_RoundTrip(t *testing.T) {
	name := &p.Identifier{Value: "Point", Span: p.Span{Start: p.Position{Line: 1, Column: 8}, End: p.Position{Line: 1, Column: 13}}}
	fx := &p.StructField{Span: p.Span{Start: p.Position{Line: 2, Column: 3}, End: p.Position{Line: 2, Column: 12}}, Name: &p.Identifier{Value: "x", Span: p.Span{Start: p.Position{Line: 2, Column: 3}, End: p.Position{Line: 2, Column: 4}}}, Type: &p.BasicType{Name: "int"}}
	fy := &p.StructField{Span: p.Span{Start: p.Position{Line: 3, Column: 3}, End: p.Position{Line: 3, Column: 12}}, Name: &p.Identifier{Value: "y", Span: p.Span{Start: p.Position{Line: 3, Column: 3}, End: p.Position{Line: 3, Column: 4}}}, Type: &p.BasicType{Name: "int"}}
	gT := &p.GenericParameter{Span: p.Span{Start: p.Position{Line: 1, Column: 14}, End: p.Position{Line: 1, Column: 17}}, Kind: p.GenericParamType, Name: &p.Identifier{Value: "T", Span: p.Span{Start: p.Position{Line: 1, Column: 16}, End: p.Position{Line: 1, Column: 17}}}}

	sd := &p.StructDeclaration{Span: p.Span{Start: p.Position{Line: 1, Column: 1}, End: p.Position{Line: 4, Column: 1}}, Name: name, Fields: []*p.StructField{fx, fy}, Generics: []*p.GenericParameter{gT}, IsPublic: true}
	pprog := &p.Program{Declarations: []p.Declaration{sd}}

	ap, err := FromParserProgram(pprog)
	if err != nil {
		t.Fatalf("FromParserProgram err: %v", err)
	}
	as, ok := ap.Declarations[0].(*aast.StructDeclaration)
	if !ok {
		t.Fatalf("expected StructDeclaration")
	}

	// Name span
	if as.Name.Span.Start.Line != 1 || as.Name.Span.Start.Column != 8 {
		t.Fatalf("struct name span lost: %+v", as.Name.Span)
	}
	// Field name spans
	if len(as.Fields) != 2 {
		t.Fatalf("fields len")
	}
	if as.Fields[0].Name.Span.Start.Line != 2 || as.Fields[0].Name.Span.Start.Column != 3 {
		t.Fatalf("field x span: %+v", as.Fields[0].Name.Span)
	}
	if as.Fields[1].Name.Span.Start.Line != 3 || as.Fields[1].Name.Span.Start.Column != 3 {
		t.Fatalf("field y span: %+v", as.Fields[1].Name.Span)
	}
	// Generic param spans
	if len(as.Generics) != 1 || as.Generics[0].Name == nil {
		t.Fatalf("generic missing")
	}
	if as.Generics[0].Span.Start.Line != 1 || as.Generics[0].Span.Start.Column != 14 {
		t.Fatalf("generic span: %+v", as.Generics[0].Span)
	}
	if as.Generics[0].Name.Span.Start.Line != 1 || as.Generics[0].Name.Span.Start.Column != 16 {
		t.Fatalf("generic name span: %+v", as.Generics[0].Name.Span)
	}

	// Round trip
	back, err := ToParserProgram(ap)
	if err != nil {
		t.Fatalf("ToParserProgram err: %v", err)
	}
	bsd := back.Declarations[0].(*p.StructDeclaration)
	if bsd.Name.Span.Start.Line != 1 || bsd.Name.Span.Start.Column != 8 {
		t.Fatalf("back name span: %+v", bsd.Name.Span)
	}
	if bsd.Fields[0].Name.Span.Start.Line != 2 || bsd.Fields[0].Name.Span.Start.Column != 3 {
		t.Fatalf("back field x span: %+v", bsd.Fields[0].Name.Span)
	}
	if bsd.Fields[1].Name.Span.Start.Line != 3 || bsd.Fields[1].Name.Span.Start.Column != 3 {
		t.Fatalf("back field y span: %+v", bsd.Fields[1].Name.Span)
	}
	if len(bsd.Generics) != 1 || bsd.Generics[0].Name == nil {
		t.Fatalf("back generic missing")
	}
	if bsd.Generics[0].Span.Start.Line != 1 || bsd.Generics[0].Span.Start.Column != 14 {
		t.Fatalf("back generic span: %+v", bsd.Generics[0].Span)
	}
	if bsd.Generics[0].Name.Span.Start.Line != 1 || bsd.Generics[0].Name.Span.Start.Column != 16 {
		t.Fatalf("back generic name span: %+v", bsd.Generics[0].Name.Span)
	}
}

func TestSpan_Enum_Variant_RoundTrip(t *testing.T) {
	vSome := &p.EnumVariant{Span: p.Span{Start: p.Position{Line: 5, Column: 3}, End: p.Position{Line: 5, Column: 10}}, Name: &p.Identifier{Value: "Some", Span: p.Span{Start: p.Position{Line: 5, Column: 3}, End: p.Position{Line: 5, Column: 7}}}}
	vNone := &p.EnumVariant{Span: p.Span{Start: p.Position{Line: 6, Column: 3}, End: p.Position{Line: 6, Column: 10}}, Name: &p.Identifier{Value: "None", Span: p.Span{Start: p.Position{Line: 6, Column: 3}, End: p.Position{Line: 6, Column: 7}}}}
	en := &p.EnumDeclaration{Span: p.Span{Start: p.Position{Line: 4, Column: 1}, End: p.Position{Line: 7, Column: 1}}, Name: &p.Identifier{Value: "Option", Span: p.Span{Start: p.Position{Line: 4, Column: 6}, End: p.Position{Line: 4, Column: 12}}}, Variants: []*p.EnumVariant{vSome, vNone}}
	ap, err := FromParserProgram(&p.Program{Declarations: []p.Declaration{en}})
	if err != nil {
		t.Fatalf("From err: %v", err)
	}
	aen := ap.Declarations[0].(*aast.EnumDeclaration)
	if aen.Variants[0].Name.Span.Start.Line != 5 || aen.Variants[0].Name.Span.Start.Column != 3 {
		t.Fatalf("variant span lost: %+v", aen.Variants[0].Name.Span)
	}
	back, err := ToParserProgram(ap)
	if err != nil {
		t.Fatalf("To err: %v", err)
	}
	ben := back.Declarations[0].(*p.EnumDeclaration)
	if ben.Variants[1].Name.Span.Start.Line != 6 || ben.Variants[1].Name.Span.Start.Column != 3 {
		t.Fatalf("back variant span: %+v", ben.Variants[1].Name.Span)
	}
}

func TestSpan_Trait_Method_Param_RoundTrip(t *testing.T) {
	m := &p.TraitMethod{Span: p.Span{Start: p.Position{Line: 10, Column: 3}, End: p.Position{Line: 10, Column: 20}}, Name: &p.Identifier{Value: "next", Span: p.Span{Start: p.Position{Line: 10, Column: 3}, End: p.Position{Line: 10, Column: 7}}}, Parameters: []*p.Parameter{{Span: p.Span{Start: p.Position{Line: 10, Column: 8}, End: p.Position{Line: 10, Column: 17}}, Name: &p.Identifier{Value: "it", Span: p.Span{Start: p.Position{Line: 10, Column: 8}, End: p.Position{Line: 10, Column: 10}}}, TypeSpec: &p.BasicType{Name: "int"}}}}
	td := &p.TraitDeclaration{Span: p.Span{Start: p.Position{Line: 9, Column: 1}, End: p.Position{Line: 11, Column: 1}}, Name: &p.Identifier{Value: "Iterable", Span: p.Span{Start: p.Position{Line: 9, Column: 7}, End: p.Position{Line: 9, Column: 15}}}, Methods: []*p.TraitMethod{m}}
	ap, err := FromParserProgram(&p.Program{Declarations: []p.Declaration{td}})
	if err != nil {
		t.Fatalf("From err: %v", err)
	}
	atr := ap.Declarations[0].(*aast.TraitDeclaration)
	if atr.Methods[0].Name.Span.Start.Column != 3 {
		t.Fatalf("method name span: %+v", atr.Methods[0].Name.Span)
	}
	if atr.Methods[0].Parameters[0].Name.Span.Start.Column != 8 {
		t.Fatalf("param name span: %+v", atr.Methods[0].Parameters[0].Name.Span)
	}
	back, err := ToParserProgram(ap)
	if err != nil {
		t.Fatalf("To err: %v", err)
	}
	btr := back.Declarations[0].(*p.TraitDeclaration)
	if btr.Methods[0].Parameters[0].Name.Span.Start.Column != 8 {
		t.Fatalf("back param name span: %+v", btr.Methods[0].Parameters[0].Name.Span)
	}
}

func TestSpan_Import_Export_RoundTrip(t *testing.T) {
	imp := &p.ImportDeclaration{Span: p.Span{Start: p.Position{Line: 1, Column: 1}, End: p.Position{Line: 1, Column: 30}}, Path: []*p.Identifier{{Value: "foo", Span: p.Span{Start: p.Position{Line: 1, Column: 8}, End: p.Position{Line: 1, Column: 11}}}, {Value: "bar", Span: p.Span{Start: p.Position{Line: 1, Column: 12}, End: p.Position{Line: 1, Column: 15}}}}, Alias: &p.Identifier{Value: "fb", Span: p.Span{Start: p.Position{Line: 1, Column: 20}, End: p.Position{Line: 1, Column: 22}}}, IsPublic: true}
	exp := &p.ExportDeclaration{Span: p.Span{Start: p.Position{Line: 2, Column: 1}, End: p.Position{Line: 2, Column: 25}}, Items: []*p.ExportItem{{Span: p.Span{Start: p.Position{Line: 2, Column: 8}, End: p.Position{Line: 2, Column: 9}}, Name: &p.Identifier{Value: "A", Span: p.Span{Start: p.Position{Line: 2, Column: 8}, End: p.Position{Line: 2, Column: 9}}}}, {Span: p.Span{Start: p.Position{Line: 2, Column: 11}, End: p.Position{Line: 2, Column: 20}}, Name: &p.Identifier{Value: "B", Span: p.Span{Start: p.Position{Line: 2, Column: 11}, End: p.Position{Line: 2, Column: 12}}}, Alias: &p.Identifier{Value: "Bee", Span: p.Span{Start: p.Position{Line: 2, Column: 15}, End: p.Position{Line: 2, Column: 18}}}}}}

	ap, err := FromParserProgram(&p.Program{Declarations: []p.Declaration{imp, exp}})
	if err != nil {
		t.Fatalf("From err: %v", err)
	}

	// Check AST spans
	aimp := ap.Declarations[0].(*aast.ImportDeclaration)
	if aimp.Path[0].Span.Start.Column != 8 || aimp.Path[1].Span.Start.Column != 12 {
		t.Fatalf("import path spans: %+v, %+v", aimp.Path[0].Span, aimp.Path[1].Span)
	}
	if aimp.Alias == nil || aimp.Alias.Span.Start.Column != 20 {
		t.Fatalf("import alias span: %+v", aimp.Alias)
	}

	aexp := ap.Declarations[1].(*aast.ExportDeclaration)
	if aexp.Items[0].Name.Span.Start.Column != 8 {
		t.Fatalf("export item0 name span: %+v", aexp.Items[0].Name.Span)
	}
	if aexp.Items[1].Alias == nil || aexp.Items[1].Alias.Span.Start.Column != 15 {
		t.Fatalf("export item1 alias span: %+v", aexp.Items[1].Alias)
	}

	// Round trip
	back, err := ToParserProgram(ap)
	if err != nil {
		t.Fatalf("To err: %v", err)
	}
	bimp := back.Declarations[0].(*p.ImportDeclaration)
	if bimp.Path[0].Span.Start.Column != 8 || bimp.Path[1].Span.Start.Column != 12 {
		t.Fatalf("back import path spans: %+v, %+v", bimp.Path[0].Span, bimp.Path[1].Span)
	}
	if bimp.Alias == nil || bimp.Alias.Span.Start.Column != 20 {
		t.Fatalf("back import alias span: %+v", bimp.Alias)
	}
	bexp := back.Declarations[1].(*p.ExportDeclaration)
	if bexp.Items[1].Alias == nil || bexp.Items[1].Alias.Span.Start.Column != 15 {
		t.Fatalf("back export item1 alias: %+v", bexp.Items[1].Alias)
	}
}
