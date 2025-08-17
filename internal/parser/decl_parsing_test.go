package parser

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/lexer"
)

// Test parsing of import/export/struct/enum/trait/impl declarations and basic HIR mapping
func TestDeclarationParsingAndHIR(t *testing.T) {
	cases := []struct {
		name   string
		source string
		check  func(t *testing.T, prog *Program, hir *HIRModule)
	}{
		{
			name:   "Import with alias (public)",
			source: "pub import core::io as cio;",
			check: func(t *testing.T, prog *Program, hir *HIRModule) {
				if len(prog.Declarations) != 1 {
					t.Fatalf("expected 1 decl, got %d", len(prog.Declarations))
				}
				imp, ok := prog.Declarations[0].(*ImportDeclaration)
				if !ok {
					t.Fatalf("expected ImportDeclaration, got %T", prog.Declarations[0])
				}
				if !imp.IsPublic {
					t.Fatalf("expected import IsPublic=true")
				}
				if len(imp.Path) != 2 || imp.Path[0].Value != "core" || imp.Path[1].Value != "io" {
					t.Fatalf("unexpected path: %+v", imp.Path)
				}
				if imp.Alias == nil || imp.Alias.Value != "cio" {
					t.Fatalf("unexpected alias: %+v", imp.Alias)
				}
				if hir == nil {
					t.Fatalf("expected HIR module")
				}
				if len(hir.Imports) != 1 {
					t.Fatalf("expected 1 HIR import, got %d", len(hir.Imports))
				}
				hi := hir.Imports[0]
				if hi.ModuleName != "core::io" || hi.Alias != "cio" || !hi.IsPublic {
					t.Fatalf("unexpected HIR import: %+v", hi)
				}
			},
		},
		{
			name:   "Export list",
			source: "export { foo, bar };",
			check: func(t *testing.T, prog *Program, hir *HIRModule) {
				if len(prog.Declarations) != 1 {
					t.Fatalf("expected 1 decl, got %d", len(prog.Declarations))
				}
				ed, ok := prog.Declarations[0].(*ExportDeclaration)
				if !ok {
					t.Fatalf("expected ExportDeclaration, got %T", prog.Declarations[0])
				}
				if len(ed.Items) != 2 {
					t.Fatalf("expected 2 export items, got %d", len(ed.Items))
				}
				if ed.Items[0].Name.Value != "foo" || ed.Items[1].Name.Value != "bar" {
					t.Fatalf("unexpected export items")
				}
				if hir == nil {
					t.Fatalf("expected HIR module")
				}
				if len(hir.Exports) != 2 {
					t.Fatalf("expected 2 HIR exports, got %d", len(hir.Exports))
				}
				if hir.Exports[0].ItemName != "foo" || hir.Exports[1].ItemName != "bar" {
					t.Fatalf("unexpected HIR export names")
				}
			},
		},
		{
			name:   "Struct declaration",
			source: "pub struct Point { x: int, pub y: int }",
			check: func(t *testing.T, prog *Program, hir *HIRModule) {
				if len(prog.Declarations) != 1 {
					t.Fatalf("expected 1 decl, got %d", len(prog.Declarations))
				}
				sd, ok := prog.Declarations[0].(*StructDeclaration)
				if !ok {
					t.Fatalf("expected StructDeclaration, got %T", prog.Declarations[0])
				}
				if !sd.IsPublic {
					t.Fatalf("expected struct IsPublic=true")
				}
				if sd.Name.Value != "Point" {
					t.Fatalf("unexpected struct name: %s", sd.Name.Value)
				}
				if len(sd.Fields) != 2 {
					t.Fatalf("expected 2 fields, got %d", len(sd.Fields))
				}
				if sd.Fields[0].Name.Value != "x" || sd.Fields[1].Name.Value != "y" {
					t.Fatalf("unexpected field names")
				}
				if hir == nil {
					t.Fatalf("expected HIR module")
				}
				if len(hir.Types) != 1 {
					t.Fatalf("expected 1 HIR type, got %d", len(hir.Types))
				}
				td := hir.Types[0]
				if td.Name != "Point" || td.Kind != TypeDefStruct {
					t.Fatalf("unexpected HIR type def: %+v", td)
				}
				st, ok := td.Data.(*HIRStructType)
				if !ok {
					t.Fatalf("expected HIRStructType, got %T", td.Data)
				}
				if len(st.Fields) != 2 {
					t.Fatalf("expected 2 struct fields in HIR, got %d", len(st.Fields))
				}
				if st.Fields[0].Name != "x" || st.Fields[1].Name != "y" {
					t.Fatalf("unexpected HIR field names")
				}
			},
		},
		{
			name:   "Enum declaration",
			source: "enum Result { Ok(int), Err{ code: int } }",
			check: func(t *testing.T, prog *Program, hir *HIRModule) {
				if len(prog.Declarations) != 1 {
					t.Fatalf("expected 1 decl, got %d", len(prog.Declarations))
				}
				ed, ok := prog.Declarations[0].(*EnumDeclaration)
				if !ok {
					t.Fatalf("expected EnumDeclaration, got %T", prog.Declarations[0])
				}
				if ed.Name.Value != "Result" {
					t.Fatalf("unexpected enum name: %s", ed.Name.Value)
				}
				if len(ed.Variants) != 2 {
					t.Fatalf("expected 2 variants, got %d", len(ed.Variants))
				}
				if ed.Variants[0].Name.Value != "Ok" || ed.Variants[1].Name.Value != "Err" {
					t.Fatalf("unexpected variant names")
				}
				if hir == nil {
					t.Fatalf("expected HIR module")
				}
				if len(hir.Types) != 1 {
					t.Fatalf("expected 1 HIR type, got %d", len(hir.Types))
				}
				td := hir.Types[0]
				if td.Name != "Result" || td.Kind != TypeDefEnum {
					t.Fatalf("unexpected HIR enum def: %+v", td)
				}
				et, ok := td.Data.(*HIREnumType)
				if !ok {
					t.Fatalf("expected HIREnumType, got %T", td.Data)
				}
				if len(et.Variants) != 2 {
					t.Fatalf("expected 2 HIR variants, got %d", len(et.Variants))
				}
				if et.Variants[0].Name != "Ok" || et.Variants[1].Name != "Err" {
					t.Fatalf("unexpected HIR variant names")
				}
			},
		},
		{
			name:   "Trait declaration",
			source: "trait Display { func fmt(x: int) -> int; }",
			check: func(t *testing.T, prog *Program, hir *HIRModule) {
				if len(prog.Declarations) != 1 {
					t.Fatalf("expected 1 decl, got %d", len(prog.Declarations))
				}
				td, ok := prog.Declarations[0].(*TraitDeclaration)
				if !ok {
					t.Fatalf("expected TraitDeclaration, got %T", prog.Declarations[0])
				}
				if td.Name.Value != "Display" {
					t.Fatalf("unexpected trait name: %s", td.Name.Value)
				}
				if len(td.Methods) != 1 || td.Methods[0].Name.Value != "fmt" {
					t.Fatalf("unexpected methods in trait")
				}
				if hir == nil {
					t.Fatalf("expected HIR module")
				}
				if len(hir.Types) != 1 {
					t.Fatalf("expected 1 HIR type, got %d", len(hir.Types))
				}
				htd := hir.Types[0]
				if htd.Name != "Display" || htd.Kind != TypeDefTrait {
					t.Fatalf("unexpected HIR trait def: %+v", htd)
				}
				tt, ok := htd.Data.(*HIRTraitType)
				if !ok {
					t.Fatalf("expected HIRTraitType, got %T", htd.Data)
				}
				if len(tt.Methods) != 1 || tt.Methods[0].Name != "fmt" {
					t.Fatalf("unexpected HIR trait methods")
				}
			},
		},
		{
			name:   "Impl block with one method",
			source: "impl Point { func norm() -> int { return 0; } }",
			check: func(t *testing.T, prog *Program, hir *HIRModule) {
				if len(prog.Declarations) != 1 {
					t.Fatalf("expected 1 decl, got %d", len(prog.Declarations))
				}
				ib, ok := prog.Declarations[0].(*ImplBlock)
				if !ok {
					t.Fatalf("expected ImplBlock, got %T", prog.Declarations[0])
				}
				if ib.ForType == nil || ib.Items == nil || len(ib.Items) != 1 {
					t.Fatalf("unexpected impl block contents")
				}
				// Current transformer ignores impl blocks (methods are not attached yet)
				if hir == nil {
					t.Fatalf("expected HIR module")
				}
				if len(hir.Functions) != 0 {
					t.Fatalf("expected no top-level functions in HIR from impl, got %d", len(hir.Functions))
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := lexer.New(tc.source)
			p := NewParser(l, "test.oriz")
			prog, errs := p.Parse()
			if prog == nil {
				t.Fatalf("expected program, got nil")
			}
			if len(errs) != 0 {
				t.Fatalf("unexpected parse errors: %v", errs)
			}
			transformer := NewASTToHIRTransformer()
			hir, terrs := transformer.TransformProgram(prog)
			if len(terrs) != 0 {
				t.Fatalf("unexpected transform errors: %v", terrs)
			}
			tc.check(t, prog, hir)
		})
	}
}
