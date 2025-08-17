// Package mir defines a Mid-level IR used between HIR and LIR.
// It is SSA-lite and structured to enable target-agnostic optimizations.
package mir

import (
	"fmt"
	"strings"
)

// Module is a compilation unit of MIR.
type Module struct {
	Name      string
	Functions []*Function
}

// Function is a collection of basic blocks.
type Function struct {
	Name       string
	Parameters []Value
	Blocks     []*BasicBlock
}

// BasicBlock is a sequence of instructions ending with a terminator.
type BasicBlock struct {
	Name  string
	Instr []Instr
}

// Value represents an SSA-like value produced by an instruction or parameter.
type Value struct {
	Kind ValueKind
	// For constants
	Int64   int64
	Float64 float64
	// For instruction results (index into block/local numbering)
	Ref string
	// Lightweight type class hint for lowering (int/float/ptr)
	Class ValueClass
}

// ValueKind classifies the value category.
type ValueKind int

const (
	ValInvalid ValueKind = iota
	ValConstInt
	ValConstFloat
	ValRef
)

// Instr is implemented by all MIR instructions.
type Instr interface{ isInstr() }

// BinOp represents a binary arithmetic or logical operation.
type BinOp struct {
	Dst string
	Op  BinOpKind
	LHS Value
	RHS Value
}

// Ret returns from the current function with an optional value.
type Ret struct{ Val *Value }

// Call represents a function call.
type Call struct {
	Dst       string
	Callee    string // optional named callee
	CalleeVal *Value // optional value callee (for indirect calls)
	Args      []Value
	// Optional: per-arg classes (e.g., int, f32, f64) and result class
	ArgClasses []string
	RetClass   string
}

// Alloca allocates a local stack slot and returns its address (by reference name).
type Alloca struct {
	Dst  string // reference name for the slot address (e.g., %x.addr)
	Name string // optional source name for readability
}

// Load loads from an address into a destination value.
type Load struct {
	Dst  string
	Addr Value // expected to be a reference to an address
}

// Store stores a value into an address.
type Store struct {
	Addr Value
	Val  Value
}

// Cmp represents a comparison producing a boolean-like value (0/1).
type Cmp struct {
	Dst  string
	Pred CmpPred
	LHS  Value
	RHS  Value
}

// Unconditional branch to a target basic block label.
type Br struct{ Target string }

// Conditional branch based on a value treated as boolean (0=false, nonzero=true).
type CondBr struct {
	Cond  Value
	True  string
	False string
}

// BinOpKind enumerates supported binary operations at MIR level.
type BinOpKind int

const (
	OpAdd BinOpKind = iota
	OpSub
	OpMul
	OpDiv
)

func (BinOp) isInstr()  {}
func (Ret) isInstr()    {}
func (Call) isInstr()   {}
func (Alloca) isInstr() {}
func (Load) isInstr()   {}
func (Store) isInstr()  {}
func (Cmp) isInstr()    {}
func (Br) isInstr()     {}
func (CondBr) isInstr() {}

// ValueClass is a minimal type class for lowering decisions.
type ValueClass int

const (
	ClassUnknown ValueClass = iota
	ClassInt                // integers, pointers
	ClassFloat              // floating point
)

func (c ValueClass) String() string {
	switch c {
	case ClassInt:
		return "int"
	case ClassFloat:
		return "float"
	default:
		return "unknown"
	}
}

func (m *Module) String() string {
	if m == nil {
		return "<nil-mir-module>"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "module %s\n", m.Name)
	for _, f := range m.Functions {
		b.WriteString(f.String())
		b.WriteByte('\n')
	}
	return b.String()
}

func (f *Function) String() string {
	if f == nil {
		return "<nil-func>"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "func %s(", f.Name)
	for i := range f.Parameters {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(valString(f.Parameters[i]))
	}
	b.WriteString(") {\n")
	for _, bb := range f.Blocks {
		b.WriteString(bb.String())
	}
	b.WriteString("}\n")
	return b.String()
}

func (bb *BasicBlock) String() string {
	if bb == nil {
		return ""
	}
	var b strings.Builder
	if bb.Name != "" {
		fmt.Fprintf(&b, "%s:\n", bb.Name)
	}
	for _, in := range bb.Instr {
		b.WriteString("  ")
		if s, ok := any(in).(fmt.Stringer); ok {
			b.WriteString(s.String())
		} else {
			b.WriteString("<instr>")
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func (v Value) String() string { return valString(v) }

func valString(v Value) string {
	switch v.Kind {
	case ValConstInt:
		return fmt.Sprintf("%d", v.Int64)
	case ValConstFloat:
		return fmt.Sprintf("%g", v.Float64)
	case ValRef:
		if v.Ref == "" {
			return "%ref?"
		}
		return v.Ref
	default:
		return "<invalid>"
	}
}

func (i BinOp) String() string {
	if i.Dst != "" {
		return fmt.Sprintf("%s = %s %s, %s", i.Dst, i.Op, i.LHS, i.RHS)
	}
	return fmt.Sprintf("%s %s, %s", i.Op, i.LHS, i.RHS)
}
func (i Ret) String() string {
	if i.Val == nil {
		return "ret"
	}
	return fmt.Sprintf("ret %s", i.Val.String())
}
func (i Call) String() string {
	var b strings.Builder
	if i.Dst != "" {
		fmt.Fprintf(&b, "%s = ", i.Dst)
	}
	// choose callee representation
	callee := i.Callee
	if i.CalleeVal != nil {
		callee = i.CalleeVal.String()
	}
	fmt.Fprintf(&b, "call %s(", callee)
	for idx, a := range i.Args {
		if idx > 0 {
			b.WriteString(", ")
		}
		b.WriteString(a.String())
	}
	b.WriteString(")")
	// Optionally annotate classes for diagnostics
	if len(i.ArgClasses) > 0 || i.RetClass != "" {
		b.WriteString(" ;")
		if len(i.ArgClasses) > 0 {
			b.WriteString(" args:")
			for j, c := range i.ArgClasses {
				if j > 0 {
					b.WriteString(",")
				}
				if c == "" {
					b.WriteString("?")
				} else {
					b.WriteString(c)
				}
			}
		}
		if i.RetClass != "" {
			fmt.Fprintf(&b, " ret:%s", i.RetClass)
		}
	}
	return b.String()
}

func (i Alloca) String() string {
	if i.Name != "" {
		return fmt.Sprintf("%s = alloca %s", i.Dst, i.Name)
	}
	return fmt.Sprintf("%s = alloca", i.Dst)
}

func (i Load) String() string {
	return fmt.Sprintf("%s = load %s", i.Dst, i.Addr.String())
}

func (i Store) String() string {
	return fmt.Sprintf("store %s, %s", i.Addr.String(), i.Val.String())
}

// CmpPred enumerates compare predicates.
type CmpPred int

const (
	// Generic equality
	CmpEQ CmpPred = iota
	CmpNE
	// Signed integer comparisons
	CmpSLT
	CmpSLE
	CmpSGT
	CmpSGE
	// Unsigned integer (and pointer) comparisons
	CmpULT
	CmpULE
	CmpUGT
	CmpUGE
	// Floating-point comparisons
	CmpFLT
	CmpFLE
	CmpFGT
	CmpFGE
)

func (p CmpPred) String() string {
	switch p {
	case CmpEQ:
		return "eq"
	case CmpNE:
		return "ne"
	case CmpSLT:
		return "slt"
	case CmpSLE:
		return "sle"
	case CmpSGT:
		return "sgt"
	case CmpSGE:
		return "sge"
	case CmpULT:
		return "ult"
	case CmpULE:
		return "ule"
	case CmpUGT:
		return "ugt"
	case CmpUGE:
		return "uge"
	case CmpFLT:
		return "flt"
	case CmpFLE:
		return "fle"
	case CmpFGT:
		return "fgt"
	case CmpFGE:
		return "fge"
	default:
		return "cmp?"
	}
}

func (i Cmp) String() string {
	if i.Dst != "" {
		return fmt.Sprintf("%s = cmp.%s %s, %s", i.Dst, i.Pred, i.LHS, i.RHS)
	}
	return fmt.Sprintf("cmp.%s %s, %s", i.Pred, i.LHS, i.RHS)
}

func (i Br) String() string { return fmt.Sprintf("br %s", i.Target) }

func (i CondBr) String() string {
	return fmt.Sprintf("brcond %s, %s, %s", i.Cond.String(), i.True, i.False)
}

func (k BinOpKind) String() string {
	switch k {
	case OpAdd:
		return "add"
	case OpSub:
		return "sub"
	case OpMul:
		return "mul"
	case OpDiv:
		return "div"
	default:
		return "binop?"
	}
}
