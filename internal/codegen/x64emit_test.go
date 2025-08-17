package codegen

import (
	"strings"
	"testing"

	"github.com/orizon-lang/orizon/internal/lir"
)

func TestEmitX64_CallWin64(t *testing.T) {
	// Build a simple LIR module with a function that calls foo with 6 args
	f := &lir.Function{Name: "test_fn"}
	b0 := &lir.BasicBlock{Label: "entry"}
	// Create a dest to force at least one stack slot
	b0.Insns = append(b0.Insns, lir.Add{Dst: "%t0", LHS: "1", RHS: "2"})
	// Call with 6 integer-like args
	call := lir.Call{Dst: "%t1", Callee: "foo", Args: []string{"1", "2", "3", "4", "5", "6"}}
	b0.Insns = append(b0.Insns, call)
	b0.Insns = append(b0.Insns, lir.Ret{})
	f.Blocks = []*lir.BasicBlock{b0}
	m := &lir.Module{Name: "m", Functions: []*lir.Function{f}}

	asm := EmitX64(m)
	// First four args in rcx, rdx, r8, r9
	for _, want := range []string{
		"mov rcx, 1",
		"mov rdx, 2",
		"mov r8, 3",
		"mov r9, 4",
	} {
		if !strings.Contains(asm, want) {
			t.Fatalf("missing register arg move: %q\nASM:\n%s", want, asm)
		}
	}
	// Stack args at [rsp+32], [rsp+40]
	if !strings.Contains(asm, "mov qword ptr [rsp+32], rax") {
		t.Fatalf("missing stack arg store at rsp+32\nASM:\n%s", asm)
	}
	if !strings.Contains(asm, "mov qword ptr [rsp+40], rax") {
		t.Fatalf("missing stack arg store at rsp+40\nASM:\n%s", asm)
	}
	// Reserve should be 48 bytes (32 shadow + 16 stack args) and aligned
	if !strings.Contains(asm, "sub rsp, 48") {
		t.Fatalf("expected shadow+stack reserve of 48 bytes\nASM:\n%s", asm)
	}
}

func TestEmitX64_IndirectCall(t *testing.T) {
	f := &lir.Function{Name: "test_indirect"}
	b0 := &lir.BasicBlock{Label: "entry"}
	// Force slot
	b0.Insns = append(b0.Insns, lir.Alloc{Dst: "%t0", Name: "tmp"})
	// Indirect callee by using SSA-looking name
	b0.Insns = append(b0.Insns, lir.Call{Dst: "%t1", Callee: "%t0", Args: []string{"7"}})
	b0.Insns = append(b0.Insns, lir.Ret{})
	f.Blocks = []*lir.BasicBlock{b0}
	m := &lir.Module{Name: "m", Functions: []*lir.Function{f}}

	asm := EmitX64(m)
	if !strings.Contains(asm, "call r11") {
		t.Fatalf("expected indirect call via r11, got:\n%s", asm)
	}
}

func TestEmitX64_UnsignedCmpSetcc(t *testing.T) {
	f := &lir.Function{Name: "test_ucmp"}
	b0 := &lir.BasicBlock{Label: "entry"}
	// cmp.ult %t0, 1, 2
	b0.Insns = append(b0.Insns, lir.Cmp{Dst: "%t0", Pred: "ult", LHS: "1", RHS: "2"})
	b0.Insns = append(b0.Insns, lir.Ret{})
	f.Blocks = []*lir.BasicBlock{b0}
	m := &lir.Module{Name: "m", Functions: []*lir.Function{f}}

	asm := EmitX64(m)
	if !strings.Contains(asm, "setb al") {
		t.Fatalf("expected setb for unsigned <, got:\n%s", asm)
	}
}

func TestEmitX64_FrameAlignment(t *testing.T) {
	// One slot -> frame 8 -> aligned to 16
	f := &lir.Function{Name: "test_frame"}
	b0 := &lir.BasicBlock{Label: "entry"}
	b0.Insns = append(b0.Insns, lir.Alloc{Dst: "%t0", Name: "x"})
	b0.Insns = append(b0.Insns, lir.Ret{})
	f.Blocks = []*lir.BasicBlock{b0}
	m := &lir.Module{Name: "m", Functions: []*lir.Function{f}}

	asm := EmitX64(m)
	if !strings.Contains(asm, "sub rsp, 16") {
		t.Fatalf("expected 16-byte aligned frame subtraction, got:\n%s", asm)
	}
}

func TestEmitX64_FloatArgsUseXMM(t *testing.T) {
	f := &lir.Function{Name: "test_float_args"}
	b0 := &lir.BasicBlock{Label: "entry"}
	// Force a slot
	b0.Insns = append(b0.Insns, lir.Alloc{Dst: "%t0", Name: "x"})
	// First two args are float class
	call := lir.Call{Dst: "%t1", Callee: "foo", Args: []string{"1.0", "2.0"}, ArgClasses: []string{"f64", "f64"}}
	b0.Insns = append(b0.Insns, call)
	b0.Insns = append(b0.Insns, lir.Ret{})
	f.Blocks = []*lir.BasicBlock{b0}
	m := &lir.Module{Name: "m", Functions: []*lir.Function{f}}

	asm := EmitX64(m)
	if !strings.Contains(asm, "movq xmm0, rax") {
		t.Fatalf("expected first float arg moved into xmm0, got:\n%s", asm)
	}
	if !strings.Contains(asm, "movq xmm1, rax") {
		t.Fatalf("expected second float arg moved into xmm1, got:\n%s", asm)
	}
}

func TestEmitX64_FloatReturnMovesFromXMM0(t *testing.T) {
	f := &lir.Function{Name: "test_float_ret"}
	b0 := &lir.BasicBlock{Label: "entry"}
	// Call returning float
	call := lir.Call{Dst: "%t0", Callee: "bar", Args: []string{}, ArgClasses: []string{}, RetClass: "f64"}
	b0.Insns = append(b0.Insns, call)
	b0.Insns = append(b0.Insns, lir.Ret{})
	f.Blocks = []*lir.BasicBlock{b0}
	m := &lir.Module{Name: "m", Functions: []*lir.Function{f}}

	asm := EmitX64(m)
	if !strings.Contains(asm, "movq rax, xmm0") {
		t.Fatalf("expected float return to move from xmm0 to rax, got:\n%s", asm)
	}
}

func TestEmitX64_FloatStackArgs_UseMovsd(t *testing.T) {
	f := &lir.Function{Name: "test_float_stack_args"}
	b0 := &lir.BasicBlock{Label: "entry"}
	// 6 args: 最初の2つfloatはXMM、残り2つfloatはスタック
	args := []string{"1.0", "2.0", "3.0", "4.0", "5.0", "6.0"}
	classes := []string{"f64", "f64", "f64", "f64", "f64", "f64"}
	b0.Insns = append(b0.Insns, lir.Call{Dst: "%t0", Callee: "foo", Args: args, ArgClasses: classes})
	b0.Insns = append(b0.Insns, lir.Ret{})
	f.Blocks = []*lir.BasicBlock{b0}
	m := &lir.Module{Name: "m", Functions: []*lir.Function{f}}

	asm := EmitX64(m)
	// 5th and 6th args go to [rsp+32], [rsp+40], stored via movsd
	if !strings.Contains(asm, "movsd qword ptr [rsp+32], xmm7") {
		t.Fatalf("expected movsd for float stack arg at rsp+32, got:\n%s", asm)
	}
	if !strings.Contains(asm, "movsd qword ptr [rsp+40], xmm7") {
		t.Fatalf("expected movsd for float stack arg at rsp+40, got:\n%s", asm)
	}
}

func TestEmitX64_F32ArgsAndReturn(t *testing.T) {
	f := &lir.Function{Name: "test_f32"}
	b0 := &lir.BasicBlock{Label: "entry"}
	// f32 args: two in XMM, two on stack
	args := []string{"1", "2", "3", "4", "5"}
	classes := []string{"f32", "f32", "f32", "f32", "f32"}
	b0.Insns = append(b0.Insns, lir.Call{Dst: "%t0", Callee: "foo32", Args: args, ArgClasses: classes, RetClass: "f32"})
	b0.Insns = append(b0.Insns, lir.Ret{})
	f.Blocks = []*lir.BasicBlock{b0}
	m := &lir.Module{Name: "m", Functions: []*lir.Function{f}}

	asm := EmitX64(m)
	// XMM loads should use movd eax→xmm
	if !strings.Contains(asm, "movd xmm0, eax") {
		t.Fatalf("expected movd into xmm0 for f32 arg, got:\n%s", asm)
	}
	// stack args use movss
	if !strings.Contains(asm, "movss dword ptr [rsp+32], xmm7") {
		t.Fatalf("expected movss for f32 stack arg at rsp+32, got:\n%s", asm)
	}
	// return uses movd eax, xmm0
	if !strings.Contains(asm, "movd eax, xmm0") {
		t.Fatalf("expected movd eax, xmm0 for f32 return, got:\n%s", asm)
	}
}
