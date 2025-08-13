package debug

import (
	"encoding/json"
	"testing"

	"github.com/orizon-lang/orizon/internal/hir"
	"github.com/orizon-lang/orizon/internal/position"
)

// minimal stubs to construct a function decl with spans
type dummyExpr struct {
	id hir.NodeID
	sp position.Span
}

func (d dummyExpr) GetID() hir.NodeID                 { return d.id }
func (d dummyExpr) GetSpan() position.Span            { return d.sp }
func (d dummyExpr) GetType() hir.TypeInfo             { return hir.TypeInfo{Kind: hir.TypeKindVoid, Name: "void"} }
func (d dummyExpr) GetEffects() hir.EffectSet         { return hir.NewEffectSet() }
func (d dummyExpr) GetRegions() hir.RegionSet         { return hir.NewRegionSet() }
func (d dummyExpr) String() string                    { return "dummy" }
func (d dummyExpr) Accept(hir.HIRVisitor) interface{} { return nil }
func (d dummyExpr) GetChildren() []hir.HIRNode        { return nil }
func (d dummyExpr) hirExpressionNode()                {}

func TestEmitter_EmitAndSerialize(t *testing.T) {
	p := hir.NewHIRProgram()
	m := &hir.HIRModule{ID: 1, ModuleID: 1, Name: "main"}
	// Fake function decl with parameters using current HIRFunctionDeclaration shape
	var i32 hir.HIRType = &hir.HIRBasicType{ID: 10, Kind: hir.TypeKindInteger, Name: "i32", Type: hir.TypeInfo{Name: "i32", Kind: hir.TypeKindInteger}}
	pA := &hir.HIRParameter{ID: 20, Name: "a", Type: i32, Span: position.Span{Start: position.Position{Filename: "main.oriz", Line: 1, Column: 5, Offset: 4}, End: position.Position{Filename: "main.oriz", Line: 1, Column: 6, Offset: 5}}}
	pB := &hir.HIRParameter{ID: 21, Name: "b", Type: i32}
	fn := &hir.HIRFunctionDeclaration{
		ID:         2,
		Name:       "add",
		Span:       position.Span{Start: position.Position{Filename: "main.oriz", Line: 1, Column: 1, Offset: 0}, End: position.Position{Filename: "main.oriz", Line: 3, Column: 1, Offset: 20}},
		Parameters: []*hir.HIRParameter{pA, pB},
		Body:       &hir.HIRBlockStatement{ID: 3, Span: position.Span{Start: position.Position{Filename: "main.oriz", Line: 1, Column: 10, Offset: 8}, End: position.Position{Filename: "main.oriz", Line: 3, Column: 1, Offset: 20}}, Statements: []hir.HIRStatement{&hir.HIRExpressionStatement{ID: 4, Span: position.Span{Start: position.Position{Filename: "main.oriz", Line: 2, Column: 5, Offset: 13}, End: position.Position{Filename: "main.oriz", Line: 2, Column: 10, Offset: 18}}, Expression: &hir.HIRLiteral{ID: 5, Type: hir.TypeInfo{Name: "i32", Kind: hir.TypeKindInteger}, Span: position.Span{Start: position.Position{Filename: "main.oriz", Line: 2, Column: 5, Offset: 13}, End: position.Position{Filename: "main.oriz", Line: 2, Column: 10, Offset: 18}}}}}},
	}
	m.Declarations = []hir.HIRDeclaration{fn}
	p.Modules[1] = m

	em := NewEmitter()
	info, err := em.Emit(p)
	if err != nil {
		t.Fatalf("emit failed: %v", err)
	}
	if len(info.Modules) != 1 || len(info.Modules[0].Functions) != 1 {
		t.Fatalf("unexpected modules/functions")
	}
	out, err := Serialize(info)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	// Must be valid JSON
	var tmp map[string]any
	if err := json.Unmarshal(out, &tmp); err != nil {
		t.Fatalf("json invalid: %v", err)
	}
}
