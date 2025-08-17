package codegen

import (
	"fmt"
	"testing"

	"github.com/orizon-lang/orizon/internal/hir"
	"github.com/orizon-lang/orizon/internal/mir"
)

func TestSelectPredByType(t *testing.T) {
	intS := hir.TypeInfo{Kind: hir.TypeKindInteger, Name: "i32"}
	intU := hir.TypeInfo{Kind: hir.TypeKindInteger, Name: "u32"}
	ptr := hir.TypeInfo{Kind: hir.TypeKindPointer, Name: "*i32"}
	flt := hir.TypeInfo{Kind: hir.TypeKindFloat, Name: "f32"}

	if got := selectPredByType("<", intS, intS); got != mir.CmpSLT {
		t.Fatalf("signed int <: want %v, got %v", mir.CmpSLT, got)
	}
	if got := selectPredByType("<", intU, intU); got != mir.CmpULT {
		t.Fatalf("unsigned int <: want %v, got %v", mir.CmpULT, got)
	}
	if got := selectPredByType("<=", ptr, ptr); got != mir.CmpULE {
		t.Fatalf("ptr <=: want %v, got %v", mir.CmpULE, got)
	}
	if got := selectPredByType(">", flt, flt); got != mir.CmpFGT {
		t.Fatalf("float >: want %v, got %v", mir.CmpFGT, got)
	}
	if got := selectPredByType("==", flt, flt); got != mir.CmpEQ {
		t.Fatalf("== uses eq: want %v, got %v", mir.CmpEQ, got)
	}
}

func TestLowerCondToBr_AndOr(t *testing.T) {
	// Build HIR: if (1 && 0) {}
	boolT := hir.TypeInfo{Kind: hir.TypeKindBoolean, Name: "bool"}
	lit1 := &hir.HIRLiteral{Value: true, Type: boolT}
	lit0 := &hir.HIRLiteral{Value: false, Type: boolT}
	andExpr := &hir.HIRBinaryExpression{Left: lit1, Operator: "&&", Right: lit0, Type: boolT}

	// Prepare MIR blocks
	temp := 0
	newTemp := func() string { t := temp; temp++; return fmt.Sprintf("%%t%d", t) }
	entry := &mir.BasicBlock{Name: "entry"}
	blocks := []*mir.BasicBlock{entry}
	trueLbl := "then"
	falseLbl := "else"

	lowerCondToBr(andExpr, &blocks, &entry, newTemp, map[string]bool{}, trueLbl, falseLbl)

	if len(blocks) < 2 {
		t.Fatalf("expected an extra RHS block for short-circuit, got %d blocks", len(blocks))
	}
	// First terminator must be CondBr to either rhs or false
	last := entry.Instr[len(entry.Instr)-1]
	br, ok := last.(mir.CondBr)
	if !ok {
		t.Fatalf("expected CondBr in entry, got %T", last)
	}
	if br.False != falseLbl {
		t.Fatalf("expected false branch to %s, got %s", falseLbl, br.False)
	}
}
