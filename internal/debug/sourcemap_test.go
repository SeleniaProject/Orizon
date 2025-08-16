package debug

import (
	"encoding/json"
	"testing"

	"github.com/orizon-lang/orizon/internal/hir"
	"github.com/orizon-lang/orizon/internal/position"
)

func TestGenerateSourceMap_Minimal(t *testing.T) {
	p := hir.NewHIRProgram()
	m := &hir.HIRModule{ID: 1, ModuleID: 1, Name: "main"}
	// Function with one expression
	fn := &hir.HIRFunctionDeclaration{
		ID:         2,
		Name:       "f",
		Span:       position.Span{Start: position.Position{Filename: "main.oriz", Line: 1, Column: 1}, End: position.Position{Filename: "main.oriz", Line: 3, Column: 1}},
		Parameters: []*hir.HIRParameter{},
		Body: &hir.HIRBlockStatement{ID: 3, Span: position.Span{Start: position.Position{Filename: "main.oriz", Line: 1, Column: 1}, End: position.Position{Filename: "main.oriz", Line: 3, Column: 1}},
			Statements: []hir.HIRStatement{&hir.HIRExpressionStatement{ID: 4, Span: position.Span{Start: position.Position{Filename: "main.oriz", Line: 2, Column: 5}, End: position.Position{Filename: "main.oriz", Line: 2, Column: 10}},
				Expression: &hir.HIRLiteral{ID: 5, Type: hir.TypeInfo{Name: "i32", Kind: hir.TypeKindInteger}, Span: position.Span{Start: position.Position{Filename: "main.oriz", Line: 2, Column: 5}, End: position.Position{Filename: "main.oriz", Line: 2, Column: 10}}},
			}},
		},
	}
	m.Declarations = []hir.HIRDeclaration{fn}
	p.Modules[1] = m

	sm, err := GenerateSourceMap(p)
	if err != nil {
		t.Fatalf("GenerateSourceMap failed: %v", err)
	}
	if len(sm.Files) != 1 || sm.Files[0] != "main.oriz" {
		t.Fatalf("unexpected files: %+v", sm.Files)
	}
	if len(sm.Functions) != 1 || sm.Functions[0].Name != "f" || len(sm.Functions[0].Mappings) == 0 {
		t.Fatalf("unexpected functions: %+v", sm.Functions)
	}
	// JSON validity
	if _, err := json.Marshal(sm); err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}
}
