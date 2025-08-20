// Package lir defines a Low-level IR close to the target ISA.
// It is suitable for straightforward instruction selection and regalloc.
package lir

import (
	"fmt"
	"strings"
)

// Module bundles functions for one object file.
type Module struct {
	Name      string
	Functions []*Function
}

// Function is a sequence of basic blocks of target-like instructions.
type Function struct {
	Name   string
	Blocks []*BasicBlock
}

// BasicBlock contains a linear list of target-like instructions.
type BasicBlock struct {
	Label string
	Insns []Insn
}

// Insn is a target-agnostic instruction representation.
type Insn interface{ Op() string }

// Mov, Add, Sub, Mul are minimal sample instructions with textual form.
type Mov struct{ Dst, Src string }

func (Mov) Op() string       { return "mov" }
func (m Mov) String() string { return fmt.Sprintf("mov %s, %s", m.Dst, m.Src) }

type Add struct{ Dst, LHS, RHS string }

func (Add) Op() string       { return "add" }
func (a Add) String() string { return fmt.Sprintf("add %s, %s, %s", a.Dst, a.LHS, a.RHS) }

type Sub struct{ Dst, LHS, RHS string }

func (Sub) Op() string       { return "sub" }
func (s Sub) String() string { return fmt.Sprintf("sub %s, %s, %s", s.Dst, s.LHS, s.RHS) }

type Mul struct{ Dst, LHS, RHS string }

func (Mul) Op() string       { return "mul" }
func (m Mul) String() string { return fmt.Sprintf("mul %s, %s, %s", m.Dst, m.LHS, m.RHS) }

type Div struct{ Dst, LHS, RHS string }

func (Div) Op() string       { return "div" }
func (d Div) String() string { return fmt.Sprintf("div %s, %s, %s", d.Dst, d.LHS, d.RHS) }

type Ret struct{ Src string }

func (Ret) Op() string { return "ret" }
func (r Ret) String() string {
	if r.Src == "" {
		return "ret"
	}

	return fmt.Sprintf("ret %s", r.Src)
}

type Call struct {
	Dst        string
	Callee     string
	RetClass   string
	Args       []string
	ArgClasses []string
}

func (Call) Op() string { return "call" }
func (c Call) String() string {
	var b strings.Builder
	if c.Dst != "" {
		fmt.Fprintf(&b, "%s = ", c.Dst)
	}

	fmt.Fprintf(&b, "call %s(", c.Callee)

	for i, a := range c.Args {
		if i > 0 {
			b.WriteString(", ")
		}

		b.WriteString(a)
	}

	b.WriteString(")")
	// Annotate classes as a comment for debugging.
	if len(c.ArgClasses) > 0 || c.RetClass != "" {
		b.WriteString(" ;")

		if len(c.ArgClasses) > 0 {
			b.WriteString(" args:")

			for i, cl := range c.ArgClasses {
				if i > 0 {
					b.WriteString(",")
				}

				if cl == "" {
					cl = "?"
				}

				b.WriteString(cl)
			}
		}

		if c.RetClass != "" {
			fmt.Fprintf(&b, " ret:%s", c.RetClass)
		}
	}

	return b.String()
}

// Compare and branching.
type Cmp struct{ Dst, Pred, LHS, RHS string }

func (Cmp) Op() string       { return "cmp" }
func (c Cmp) String() string { return fmt.Sprintf("cmp.%s %s, %s, %s", c.Pred, c.Dst, c.LHS, c.RHS) }

type Br struct{ Target string }

func (Br) Op() string       { return "br" }
func (b Br) String() string { return fmt.Sprintf("br %s", b.Target) }

type BrCond struct{ Cond, True, False string }

func (BrCond) Op() string       { return "brcond" }
func (b BrCond) String() string { return fmt.Sprintf("brcond %s, %s, %s", b.Cond, b.True, b.False) }

// Memory operations.
type Alloc struct{ Dst, Name string }

func (Alloc) Op() string { return "alloca" }
func (a Alloc) String() string {
	if a.Name != "" {
		return fmt.Sprintf("%s = alloca %s", a.Dst, a.Name)
	}

	return fmt.Sprintf("%s = alloca", a.Dst)
}

type Load struct{ Dst, Addr string }

func (Load) Op() string       { return "load" }
func (l Load) String() string { return fmt.Sprintf("%s = load %s", l.Dst, l.Addr) }

type Store struct{ Addr, Val string }

func (Store) Op() string       { return "store" }
func (s Store) String() string { return fmt.Sprintf("store %s, %s", s.Addr, s.Val) }

func (m *Module) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "module %s\n", m.Name)

	for _, f := range m.Functions {
		b.WriteString(f.String())
		b.WriteByte('\n')
	}

	return b.String()
}

func (f *Function) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "func %s() {\n", f.Name)

	for _, bb := range f.Blocks {
		if bb.Label != "" {
			fmt.Fprintf(&b, "%s:\n", bb.Label)
		}

		for _, ins := range bb.Insns {
			if s, ok := any(ins).(fmt.Stringer); ok {
				b.WriteString("  ")
				b.WriteString(s.String())
				b.WriteByte('\n')
			} else {
				fmt.Fprintf(&b, "  %s\n", ins.Op())
			}
		}
	}

	b.WriteString("}\n")

	return b.String()
}
