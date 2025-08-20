package parser

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/lexer"
)

// Test parsing of import/export/struct/enum/trait/impl declarations and basic HIR mapping.
func TestDeclarationParsingAndHIR(t *testing.T) {
	cases := []struct {
		check  func(t *testing.T, prog *Program, hir *HIRModule)
		name   string
		source string
	}{
		{
			name:   "Import wildcard (public)",
			source: "pub import core::io::*;",
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
				if imp.Alias != nil {
					t.Fatalf("did not expect alias for wildcard import")
				}
				if hir == nil {
					t.Fatalf("expected HIR module")
				}
				if len(hir.Imports) != 1 {
					t.Fatalf("expected 1 HIR import, got %d", len(hir.Imports))
				}
				hi := hir.Imports[0]
				if hi.ModuleName != "core::io" || !hi.IsPublic {
					t.Fatalf("unexpected HIR import: %+v", hi)
				}
			},
		},
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
				if hir == nil {
					t.Fatalf("expected HIR module")
				}
				// Now impls are represented explicitly in HIR.
				if len(hir.Impls) != 1 {
					t.Fatalf("expected 1 HIR impl, got %d", len(hir.Impls))
				}
				if hir.Impls[0].Kind != HIRImplInherent {
					t.Fatalf("expected inherent impl kind")
				}
				if len(hir.Impls[0].Methods) != 1 || hir.Impls[0].Methods[0].Name != "norm" {
					t.Fatalf("expected 1 method 'norm' in impl")
				}
			},
		},
		{
			name:   "Newtype declaration -> HIR",
			source: "newtype UserId = i64;",
			check: func(t *testing.T, prog *Program, hir *HIRModule) {
				if len(prog.Declarations) != 1 {
					t.Fatalf("expected 1 decl, got %d", len(prog.Declarations))
				}
				nd, ok := prog.Declarations[0].(*NewtypeDeclaration)
				if !ok {
					t.Fatalf("expected NewtypeDeclaration, got %T", prog.Declarations[0])
				}
				if nd.Name.Value != "UserId" {
					t.Fatalf("unexpected newtype name: %s", nd.Name.Value)
				}
				if hir == nil {
					t.Fatalf("expected HIR module")
				}
				if len(hir.Types) != 1 {
					t.Fatalf("expected 1 HIR type def, got %d", len(hir.Types))
				}
				if hir.Types[0].Kind != TypeDefNewtype {
					t.Fatalf("expected TypeDefNewtype, got %v", hir.Types[0].Kind)
				}
			},
		},
		{
			name:   "Newtype without semicolon is tolerated (optional)",
			source: "newtype Tag = i32\n",
			check: func(t *testing.T, prog *Program, hir *HIRModule) {
				if len(hir.Types) != 1 || hir.Types[0].Kind != TypeDefNewtype {
					t.Fatalf("expected one newtype in HIR")
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

// New tests: block-level error recovery inside declaration bodies.
func TestRecovery_StructFieldMissingColon_AllowsFollowingFields(t *testing.T) {
	src := "struct S { a int, b: int, c: int }"
	l := lexer.New(src)
	p := NewParser(l, "test.oriz")

	prog, errs := p.Parse()
	if len(prog.Declarations) != 1 {
		t.Fatalf("expected 1 decl, got %d", len(prog.Declarations))
	}

	if len(errs) == 0 {
		t.Fatalf("expected parse error for missing ':' in struct field")
	}

	sd, ok := prog.Declarations[0].(*StructDeclaration)
	if !ok {
		t.Fatalf("expected StructDeclaration, got %T", prog.Declarations[0])
	}
	// We should still have parsed at least the valid trailing fields.
	if len(sd.Fields) < 2 {
		t.Fatalf("expected to recover and parse following fields, got %d", len(sd.Fields))
	}
}

func TestRecovery_EnumVariantStructFieldMissingColon_AllowsFollowing(t *testing.T) {
	src := "enum E { V{ x int, y: int } }"
	l := lexer.New(src)
	p := NewParser(l, "test.oriz")

	prog, errs := p.Parse()
	if len(errs) == 0 {
		t.Fatalf("expected parse error for missing ':' in enum variant field")
	}

	if len(prog.Declarations) != 1 {
		t.Fatalf("expected 1 decl, got %d", len(prog.Declarations))
	}

	ed, ok := prog.Declarations[0].(*EnumDeclaration)
	if !ok {
		t.Fatalf("expected EnumDeclaration, got %T", prog.Declarations[0])
	}

	if len(ed.Variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(ed.Variants))
	}
	// Should still capture at least the valid field 'y'.
	foundValid := false

	for _, f := range ed.Variants[0].Fields {
		if f.Name != nil && f.Name.Value == "y" {
			foundValid = true

			break
		}
	}

	if !foundValid {
		t.Fatalf("expected to recover and parse valid field after error")
	}
}

func TestRecovery_TraitItemErrors_DoNotPreventSubsequentItems(t *testing.T) {
	src := "trait T { type ; func f(x: int); func g() -> int; }"
	l := lexer.New(src)
	p := NewParser(l, "test.oriz")

	prog, errs := p.Parse()
	if len(errs) == 0 {
		t.Fatalf("expected errors in malformed trait items")
	}

	if len(prog.Declarations) != 1 {
		t.Fatalf("expected 1 decl, got %d", len(prog.Declarations))
	}

	td, ok := prog.Declarations[0].(*TraitDeclaration)
	if !ok {
		t.Fatalf("expected TraitDeclaration, got %T", prog.Declarations[0])
	}
	// associated type malformed should be skipped; methods f and g should parse.
	if len(td.Methods) < 2 {
		t.Fatalf("expected to recover and parse subsequent methods, got %d", len(td.Methods))
	}
}

func TestRecovery_ImplBlock_UnexpectedItem_SkipsToNextFunc(t *testing.T) {
	src := "impl S { let x = 1; func ok() { return; } }"
	l := lexer.New(src)
	p := NewParser(l, "test.oriz")

	prog, errs := p.Parse()
	if len(prog.Declarations) != 1 {
		t.Fatalf("expected 1 decl, got %d", len(prog.Declarations))
	}

	if len(errs) == 0 {
		t.Fatalf("expected error for unexpected item in impl block")
	}

	ib, ok := prog.Declarations[0].(*ImplBlock)
	if !ok {
		t.Fatalf("expected ImplBlock, got %T", prog.Declarations[0])
	}

	if len(ib.Items) != 1 || ib.Items[0].Name.Value != "ok" {
		t.Fatalf("expected to recover and parse following function in impl block")
	}
}

func TestNewtypeParseErrorRecovery(t *testing.T) {
	// malformed newtype should produce error but parser should continue on next decl.
	src := "newtype = i32;\ntrait T { }"
	l := lexer.New(src)
	p := NewParser(l, "test.oriz")

	prog, errs := p.Parse()
	if len(errs) == 0 {
		t.Fatalf("expected errors for malformed newtype")
	}

	if len(prog.Declarations) == 0 {
		t.Fatalf("expected at least one declaration after recovery")
	}
}

func TestParserRecoverySyncPoints_TraitAfterBadImport(t *testing.T) {
	// Malformed import followed by a trait; parser should recover and parse the trait.
	src := "import core::; trait T { }"
	l := lexer.New(src)
	p := NewParser(l, "test.oriz")

	prog, errs := p.Parse()
	if prog == nil {
		t.Fatalf("expected program, got nil")
	}

	if len(errs) == 0 {
		t.Fatalf("expected parse errors due to malformed import, got none")
	}

	if len(prog.Declarations) != 1 {
		t.Fatalf("expected 1 decl (trait) after recovery, got %d", len(prog.Declarations))
	}

	if _, ok := prog.Declarations[0].(*TraitDeclaration); !ok {
		t.Fatalf("expected TraitDeclaration after recovery, got %T", prog.Declarations[0])
	}
}

func TestParseStructWithGenerics_ASTOnly(t *testing.T) {
	src := "struct S<T> { x: T, }"
	l := lexer.New(src)
	p := NewParser(l, "test.oriz")

	prog, errs := p.Parse()
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	if len(prog.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(prog.Declarations))
	}

	sd, ok := prog.Declarations[0].(*StructDeclaration)
	if !ok {
		t.Fatalf("expected StructDeclaration, got %T", prog.Declarations[0])
	}

	if len(sd.Generics) != 1 || sd.Generics[0].Kind != GenericParamType || sd.Generics[0].Name.Value != "T" {
		t.Fatalf("unexpected generics: %+v", sd.Generics)
	}

	if len(sd.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(sd.Fields))
	}
}

func TestParseTraitWithAssocType_ASTOnly(t *testing.T) {
	src := "trait Tr<T> { type Item; func f(x: T); }"
	l := lexer.New(src)
	p := NewParser(l, "test.oriz")

	prog, errs := p.Parse()
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	if len(prog.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(prog.Declarations))
	}

	td, ok := prog.Declarations[0].(*TraitDeclaration)
	if !ok {
		t.Fatalf("expected TraitDeclaration, got %T", prog.Declarations[0])
	}

	if len(td.Generics) != 1 || td.Generics[0].Name.Value != "T" {
		t.Fatalf("unexpected generics: %+v", td.Generics)
	}

	if len(td.AssociatedTypes) != 1 || td.AssociatedTypes[0].Name.Value != "Item" {
		t.Fatalf("expected one associated type 'Item', got %+v", td.AssociatedTypes)
	}

	if len(td.Methods) != 1 || td.Methods[0].Name.Value != "f" {
		t.Fatalf("expected one method 'f', got %+v", td.Methods)
	}
}

func TestParseImplWithWhere_ASTOnly(t *testing.T) {
	src := "impl<T> S<T> where T: Eq { }"
	l := lexer.New(src)
	p := NewParser(l, "test.oriz")

	prog, errs := p.Parse()
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	if len(prog.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(prog.Declarations))
	}

	ib, ok := prog.Declarations[0].(*ImplBlock)
	if !ok {
		t.Fatalf("expected ImplBlock, got %T", prog.Declarations[0])
	}

	if len(ib.Generics) != 1 || ib.Generics[0].Name.Value != "T" {
		t.Fatalf("unexpected generics: %+v", ib.Generics)
	}

	if len(ib.WhereClauses) != 1 {
		t.Fatalf("expected 1 where clause, got %d", len(ib.WhereClauses))
	}
}

func TestParseTypeAlias_ASTOnly(t *testing.T) {
	src := "type MyInt = i32;"
	l := lexer.New(src)
	p := NewParser(l, "test.oriz")

	prog, errs := p.Parse()
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	if len(prog.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(prog.Declarations))
	}

	td, ok := prog.Declarations[0].(*TypeAliasDeclaration)
	if !ok {
		t.Fatalf("expected TypeAliasDeclaration, got %T", prog.Declarations[0])
	}

	if td.Name.Value != "MyInt" {
		t.Fatalf("unexpected alias name: %s", td.Name.Value)
	}
}

func TestParseFunctionWithGenerics_ASTOnly(t *testing.T) {
	src := "func f<T>(x: T) { }"
	l := lexer.New(src)
	p := NewParser(l, "test.oriz")

	prog, errs := p.Parse()
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	if len(prog.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(prog.Declarations))
	}

	fd, ok := prog.Declarations[0].(*FunctionDeclaration)
	if !ok {
		t.Fatalf("expected FunctionDeclaration, got %T", prog.Declarations[0])
	}

	if fd.Name.Value != "f" {
		t.Fatalf("unexpected function name: %s", fd.Name.Value)
	}

	if len(fd.Generics) != 1 || fd.Generics[0].Name.Value != "T" {
		t.Fatalf("unexpected generics on function: %+v", fd.Generics)
	}
}

func TestParseTraitMethodWithGenerics_ASTOnly(t *testing.T) {
	src := "trait Tr { func m<T>(x: T); }"
	l := lexer.New(src)
	p := NewParser(l, "test.oriz")

	prog, errs := p.Parse()
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	if len(prog.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(prog.Declarations))
	}

	td, ok := prog.Declarations[0].(*TraitDeclaration)
	if !ok {
		t.Fatalf("expected TraitDeclaration, got %T", prog.Declarations[0])
	}

	if len(td.Methods) != 1 {
		t.Fatalf("expected 1 method, got %d", len(td.Methods))
	}

	m := td.Methods[0]
	if m.Name.Value != "m" {
		t.Fatalf("unexpected method name: %s", m.Name.Value)
	}

	if len(m.Generics) != 1 || m.Generics[0].Name.Value != "T" {
		t.Fatalf("unexpected generics on trait method: %+v", m.Generics)
	}
}

func TestHIR_TypeAlias_And_TraitAssoc_Minimal(t *testing.T) {
	// Type alias.
	{
		src := "type MyInt = i32;"
		l := lexer.New(src)
		p := NewParser(l, "test.oriz")

		prog, errs := p.Parse()
		if len(errs) != 0 {
			t.Fatalf("unexpected parse errors: %v", errs)
		}

		tr := NewASTToHIRTransformer()

		hir, terrs := tr.TransformProgram(prog)
		if len(terrs) != 0 {
			t.Fatalf("unexpected transform errors: %v", terrs)
		}

		if len(hir.Types) != 1 {
			t.Fatalf("expected 1 HIR type, got %d", len(hir.Types))
		}

		if hir.Types[0].Kind != TypeDefAlias {
			t.Fatalf("expected TypeDefAlias, got %v", hir.Types[0].Kind)
		}
	}
	// Trait associated type.
	{
		src := "trait Tr { type Item; }"
		l := lexer.New(src)
		p := NewParser(l, "test.oriz")

		prog, errs := p.Parse()
		if len(errs) != 0 {
			t.Fatalf("unexpected parse errors: %v", errs)
		}

		tr := NewASTToHIRTransformer()

		hir, terrs := tr.TransformProgram(prog)
		if len(terrs) != 0 {
			t.Fatalf("unexpected transform errors: %v", terrs)
		}

		if len(hir.Types) != 1 {
			t.Fatalf("expected 1 HIR type, got %d", len(hir.Types))
		}

		td := hir.Types[0]
		if td.Kind != TypeDefTrait {
			t.Fatalf("expected trait type def, got %v", td.Kind)
		}

		ht, ok := td.Data.(*HIRTraitType)
		if !ok {
			t.Fatalf("expected HIRTraitType, got %T", td.Data)
		}

		if len(ht.AssociatedTypes) != 1 || ht.AssociatedTypes[0].Name != "Item" {
			t.Fatalf("expected one associated type 'Item', got %+v", ht.AssociatedTypes)
		}
	}
}

func TestHIR_Impl_Trait_And_Constraints(t *testing.T) {
	// Trait impl should be represented with HIRImplTrait and method metadata.
	{
		src := "impl Display for Point { func fmt(x: int) -> int { return 1; } }"
		l := lexer.New(src)
		p := NewParser(l, "test.oriz")

		prog, errs := p.Parse()
		if len(errs) != 0 {
			t.Fatalf("unexpected parse errors in first test: %v", errs)
		}

		tr := NewASTToHIRTransformer()

		hir, terrs := tr.TransformProgram(prog)
		if len(terrs) != 0 {
			t.Fatalf("unexpected transform errors in first test: %v", terrs)
		}

		if len(hir.Impls) != 1 {
			t.Fatalf("expected 1 HIR impl, got %d", len(hir.Impls))
		}

		impl := hir.Impls[0]
		if impl.Kind != HIRImplTrait {
			t.Fatalf("expected HIRImplTrait, got %v", impl.Kind)
		}

		if impl.Trait == nil || impl.ForType == nil {
			t.Fatalf("expected both Trait and ForType to be set")
		}

		if len(impl.Methods) != 1 || !impl.Methods[0].IsMethod || impl.Methods[0].ImplementedTrait == nil {
			t.Fatalf("expected one trait method with ImplementedTrait set")
		}
	}

	// Impl with generics and where constraints should populate TypeParams and Constraints.
	{
		src := "impl<T> S where T: Eq { func id(x: int) -> int { return x; } }"
		l := lexer.New(src)
		p := NewParser(l, "test.oriz")

		prog, errs := p.Parse()
		if len(errs) != 0 {
			t.Fatalf("unexpected parse errors in second test: %v", errs)
		}

		tr := NewASTToHIRTransformer()

		hir, terrs := tr.TransformProgram(prog)
		if len(terrs) != 0 {
			t.Fatalf("unexpected transform errors: %v", terrs)
		}

		if len(hir.Impls) != 1 {
			t.Fatalf("expected 1 HIR impl, got %d", len(hir.Impls))
		}

		impl := hir.Impls[0]
		if impl.Kind != HIRImplInherent {
			t.Fatalf("expected inherent impl kind, got %v", impl.Kind)
		}

		if len(impl.TypeParams) != 1 || impl.TypeParams[0].Name != "T" {
			t.Fatalf("expected one type parameter T, got %+v", impl.TypeParams)
		}

		if len(impl.Constraints) != 1 || impl.Constraints[0].Trait == nil {
			t.Fatalf("expected one where constraint with trait bound")
		}

		if len(impl.Methods) != 1 || !impl.Methods[0].IsMethod || impl.Methods[0].MethodOfType == nil {
			t.Fatalf("expected one inherent method with receiver type set")
		}
	}
}
