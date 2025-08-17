package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/orizon-lang/orizon/internal/lir"
)

const scratchXMM = "xmm7" // スタック上の浮動小数引数退避に利用（非callee-saved、テストもこれを前提）

// EmitX64 emits a very naive Windows x64-like assembly text from LIR.
// It assigns each SSA value (%tN) a stack slot and uses RAX/RBX as scratch.
// This is for diagnostics only and not ABI-correct; arguments/returns are best-effort.
func EmitX64(m *lir.Module) string {
	var b strings.Builder
	fmt.Fprintf(&b, "; module %s\n", m.Name)
	for _, f := range m.Functions {
		emitFunc(&b, f)
	}
	return b.String()
}

func emitFunc(b *strings.Builder, f *lir.Function) {
	fmt.Fprintf(b, "%s:\n", f.Name)
	// Collect SSA destinations for stack slots
	slots := collectSlots(f)
	frameSize := int64(len(slots)) * 8
	// Align frame to 16 bytes for better call alignment (best-effort)
	if rem := frameSize % 16; rem != 0 {
		frameSize += 16 - rem
	}
	// Prologue
	b.WriteString("  push rbp\n")
	b.WriteString("  mov rbp, rsp\n")
	if frameSize > 0 {
		fmt.Fprintf(b, "  sub rsp, %d\n", frameSize)
	}
	// Emit blocks
	for _, bb := range f.Blocks {
		if bb.Label != "" {
			fmt.Fprintf(b, "%s:\n", bb.Label)
		}
		for _, ins := range bb.Insns {
			switch v := ins.(type) {
			case lir.Mov:
				// Generic move
				loadValue(b, slots, v.Src, "rax")
				storeValue(b, slots, v.Dst, "rax")
			case lir.Add:
				loadValue(b, slots, v.LHS, "rax")
				loadValue(b, slots, v.RHS, "r10")
				b.WriteString("  add rax, r10\n")
				storeValue(b, slots, v.Dst, "rax")
			case lir.Sub:
				loadValue(b, slots, v.LHS, "rax")
				loadValue(b, slots, v.RHS, "r10")
				b.WriteString("  sub rax, r10\n")
				storeValue(b, slots, v.Dst, "rax")
			case lir.Mul:
				loadValue(b, slots, v.LHS, "rax")
				loadValue(b, slots, v.RHS, "r10")
				b.WriteString("  imul rax, r10\n")
				storeValue(b, slots, v.Dst, "rax")
			case lir.Div:
				// Very naive: rax/lhs, rbx/rhs, idiv rbx
				loadValue(b, slots, v.LHS, "rax")
				loadValue(b, slots, v.RHS, "r10")
				b.WriteString("  cqo\n")
				b.WriteString("  idiv r10\n")
				storeValue(b, slots, v.Dst, "rax")
			case lir.Load:
				// Treat Addr as memory symbol or stack slot
				addr := v.Addr
				if off, ok := slots[addr]; ok {
					fmt.Fprintf(b, "  mov rax, qword ptr [rbp-%d]\n", off)
				} else if isImmediateInt(addr) {
					fmt.Fprintf(b, "  mov rax, %s\n", addr)
				} else {
					fmt.Fprintf(b, "  mov rax, qword ptr [%s]\n", addr)
				}
				storeValue(b, slots, v.Dst, "rax")
			case lir.Store:
				// Store Val to Addr
				loadValue(b, slots, v.Val, "rax")
				if off, ok := slots[v.Addr]; ok {
					fmt.Fprintf(b, "  mov qword ptr [rbp-%d], rax\n", off)
				} else {
					fmt.Fprintf(b, "  mov qword ptr [%s], rax\n", v.Addr)
				}
			case lir.Cmp:
				loadValue(b, slots, v.LHS, "rax")
				loadValue(b, slots, v.RHS, "r10")
				b.WriteString("  cmp rax, r10\n")
				// Map predicate to setcc
				setcc := mapCmpToSetcc(v.Pred)
				fmt.Fprintf(b, "  %s al\n", setcc)
				b.WriteString("  movzx rax, al\n")
				storeValue(b, slots, v.Dst, "rax")
			case lir.Br:
				fmt.Fprintf(b, "  jmp %s\n", v.Target)
			case lir.BrCond:
				// Evaluate cond into rax (0/1)
				loadValue(b, slots, v.Cond, "rax")
				b.WriteString("  test rax, rax\n")
				fmt.Fprintf(b, "  jnz %s\n", v.True)
				fmt.Fprintf(b, "  jmp %s\n", v.False)
			case lir.Call:
				// Win64-like call: GPR rcx,rdx,r8,r9 for ints/ptrs; XMM0-3 for floats; 32-byte shadow space
				gprRegs := []string{"rcx", "rdx", "r8", "r9"}
				xmmRegs := []string{"xmm0", "xmm1", "xmm2", "xmm3"}
				stackArgs := 0
				if len(v.Args) > 4 {
					stackArgs = len(v.Args) - 4
				}
				// Reserve shadow space + stack args, rounded up to 16B to keep RSP 16B-aligned at call.
				reserve := int64(32 + stackArgs*8)
				if rem := reserve % 16; rem != 0 {
					reserve += 16 - rem
				}
				if reserve > 0 {
					fmt.Fprintf(b, "  sub rsp, %d\n", reserve)
				}
				// Place stack args at [rsp+32 + i*8]
				for i := 4; i < len(v.Args); i++ {
					off := 32 + (i-4)*8
					cls := ""
					if i < len(v.ArgClasses) {
						cls = v.ArgClasses[i]
					}
					if cls == "f32" {
						loadValue(b, slots, v.Args[i], "rax")
						fmt.Fprintf(b, "  movd %s, eax\n", scratchXMM)
						fmt.Fprintf(b, "  movss dword ptr [rsp+%d], %s\n", off, scratchXMM)
					} else if cls == "f64" {
						loadValue(b, slots, v.Args[i], "rax")
						fmt.Fprintf(b, "  movq %s, rax\n", scratchXMM)
						fmt.Fprintf(b, "  movsd qword ptr [rsp+%d], %s\n", off, scratchXMM)
					} else {
						loadValue(b, slots, v.Args[i], "rax")
						fmt.Fprintf(b, "  mov qword ptr [rsp+%d], rax\n", off)
					}
				}
				// Load first four args into appropriate registers by class
				g := 0
				x := 0
				for i := 0; i < len(v.Args) && (g < len(gprRegs) || x < len(xmmRegs)); i++ {
					cls := ""
					if i < len(v.ArgClasses) {
						cls = v.ArgClasses[i]
					}
					if cls == "f32" || cls == "f64" {
						if x < len(xmmRegs) {
							// load into rax then mov to xmm by width
							loadValue(b, slots, v.Args[i], "rax")
							if cls == "f32" {
								fmt.Fprintf(b, "  movd %s, eax\n", xmmRegs[x])
							} else {
								fmt.Fprintf(b, "  movq %s, rax\n", xmmRegs[x])
							}
							x++
						}
					} else {
						if g < len(gprRegs) {
							loadValue(b, slots, v.Args[i], gprRegs[g])
							g++
						}
					}
				}
				// Direct vs indirect call
				callee := v.Callee
				if strings.HasPrefix(callee, "%") {
					// Indirect via SSA value -> load into r11 and call r11
					loadValue(b, slots, callee, "r11")
					b.WriteString("  call r11\n")
				} else {
					fmt.Fprintf(b, "  call %s\n", callee)
				}
				if reserve > 0 {
					fmt.Fprintf(b, "  add rsp, %d\n", reserve)
				}
				if v.Dst != "" {
					// Return value: f32/f64 -> xmm0 to rax/eax appropriately; else rax
					if v.RetClass == "f32" {
						b.WriteString("  movd eax, xmm0\n")
					} else if v.RetClass == "f64" {
						b.WriteString("  movq rax, xmm0\n")
					}
					storeValue(b, slots, v.Dst, "rax")
				}
			case lir.Ret:
				if v.Src != "" {
					loadValue(b, slots, v.Src, "rax")
				}
				// Epilogue
				b.WriteString("  mov rsp, rbp\n")
				b.WriteString("  pop rbp\n")
				b.WriteString("  ret\n")
			case lir.Alloc:
				// No-op here; stack slot per SSA already reserved. Keep as comment.
				fmt.Fprintf(b, "; alloca %s -> %s\n", v.Name, v.Dst)
			default:
				// Unknown: emit comment
				if s, ok := any(ins).(fmt.Stringer); ok {
					fmt.Fprintf(b, "; unknown: %s\n", s.String())
				} else {
					fmt.Fprintf(b, "; unknown op %s\n", ins.Op())
				}
			}
		}
	}
	// Ensure epilogue if fallthrough
	needTail := true
	for i := len(f.Blocks) - 1; i >= 0 && needTail; i-- {
		bb := f.Blocks[i]
		if len(bb.Insns) == 0 {
			continue
		}
		switch bb.Insns[len(bb.Insns)-1].(type) {
		case lir.Ret:
			needTail = false
		default:
			needTail = true
		}
		break
	}
	if needTail {
		b.WriteString("  mov rsp, rbp\n  pop rbp\n  ret\n")
	}
}

func collectSlots(f *lir.Function) map[string]int64 {
	slots := make(map[string]int64)
	next := int64(8)
	add := func(name string) {
		if name == "" {
			return
		}
		if _, ok := slots[name]; ok {
			return
		}
		slots[name] = next
		next += 8
	}
	for _, bb := range f.Blocks {
		for _, ins := range bb.Insns {
			switch v := ins.(type) {
			case lir.Add:
				add(v.Dst)
			case lir.Sub:
				add(v.Dst)
			case lir.Mul:
				add(v.Dst)
			case lir.Div:
				add(v.Dst)
			case lir.Load:
				add(v.Dst)
			case lir.Store:
				// address could be a slot; ensure it exists too
				add(v.Addr)
			case lir.Cmp:
				add(v.Dst)
			case lir.Call:
				add(v.Dst)
			case lir.Alloc:
				add(v.Dst)
			}
		}
	}
	return slots
}

func loadValue(b *strings.Builder, slots map[string]int64, src, reg string) {
	if src == "" {
		fmt.Fprintf(b, "  xor %s, %s\n", reg, reg)
		return
	}
	if isImmediateInt(src) {
		fmt.Fprintf(b, "  mov %s, %s\n", reg, src)
		return
	}
	if off, ok := slots[src]; ok {
		fmt.Fprintf(b, "  mov %s, qword ptr [rbp-%d]", reg, off)
		b.WriteString("\n")
		return
	}
	// Symbolic
	fmt.Fprintf(b, "  mov %s, qword ptr [%s]\n", reg, src)
}

func storeValue(b *strings.Builder, slots map[string]int64, dst, reg string) {
	if dst == "" {
		return
	}
	if off, ok := slots[dst]; ok {
		fmt.Fprintf(b, "  mov qword ptr [rbp-%d], %s\n", off, reg)
		return
	}
	// Symbolic
	fmt.Fprintf(b, "  mov qword ptr [%s], %s\n", dst, reg)
}

func isImmediateInt(s string) bool {
	if len(s) == 0 {
		return false
	}
	if s[0] == '-' {
		s = s[1:]
	}
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

func mapCmpToSetcc(pred string) string {
	switch pred {
	case "eq":
		return "sete"
	case "ne":
		return "setne"
	case "slt", "flt":
		return "setl"
	case "sle", "fle":
		return "setle"
	case "sgt", "fgt":
		return "setg"
	case "sge", "fge":
		return "setge"
	case "ult":
		return "setb" // below (unsigned <)
	case "ule":
		return "setbe"
	case "ugt":
		return "seta" // above (unsigned >)
	case "uge":
		return "setae"
	default:
		return "sete"
	}
}
