// Package codegen wires HIR -> MIR -> LIR lowering stages.
package codegen

import (
	"fmt"
	"os"

	"github.com/orizon-lang/orizon/internal/hir"
	"github.com/orizon-lang/orizon/internal/lir"
	"github.com/orizon-lang/orizon/internal/mir"
)

// getBinOpFromAssignOp converts assignment operators to binary operations.
func getBinOpFromAssignOp(assignOp string) (mir.BinOpKind, bool) {
	switch assignOp {
	case "+=":
		return mir.OpAdd, true
	case "-=":
		return mir.OpSub, true
	case "*=":
		return mir.OpMul, true
	case "/=":
		return mir.OpDiv, true
	case "%=":
		return mir.OpMod, true
	case "&=":
		return mir.OpAnd, true
	case "|=":
		return mir.OpOr, true
	case "^=":
		return mir.OpXor, true
	case "<<=":
		return mir.OpShl, true
	case ">>=":
		return mir.OpShr, true
	default:
		return 0, false // Simple assignment or unsupported
	}
}

// LowerToMIR converts HIR to a simple MIR module.
// Currently supports lowering function declarations with return statements of literal values.
// TODO: Extend to full expression/statement coverage, control flow, and SSA values.
func LowerToMIR(p *hir.HIRProgram) *mir.Module {
	if p == nil {
		return &mir.Module{Name: "<nil>"}
	}

	m := &mir.Module{Name: "main"}

	// Traverse modules and declarations to find functions.
	for _, mod := range p.Modules {
		if mod == nil {
			continue
		}
		// Prefer module name if available.
		if mod.Name != "" {
			m.Name = mod.Name
		}

		for _, d := range mod.Declarations {
			fd, ok := d.(*hir.HIRFunctionDeclaration)
			if !ok || fd == nil {
				continue
			}

			f := &mir.Function{Name: fd.Name}
			// パラメータを MIR の Parameters に登録（名前参照用）.
			for _, hp := range fd.Parameters {
				if hp != nil {
					f.Parameters = append(f.Parameters, mir.Value{Kind: mir.ValRef, Ref: hp.Name})
				}
			}

			temp := 0
			newTemp := func() string {
				t := fmt.Sprintf("%%t%d", temp)
				temp++
				return t
			}

			// entry ブロック作成.
			entry := &mir.BasicBlock{Name: "entry"}

			// ローカル・パラメータのスタックアロケーション（アドレスを名前に紐付け）.
			// 各パラメータに %<name>.addr を割り当て、最初に store する
			env := make(map[string]bool)

			for _, hp := range fd.Parameters {
				if hp == nil || hp.Name == "" {
					continue
				}

				addr := fmt.Sprintf("%%%s.addr", hp.Name)
				entry.Instr = append(entry.Instr, mir.Alloca{Dst: addr, Name: hp.Name})
				// store %param -> %param.addr
				entry.Instr = append(entry.Instr, mir.Store{Addr: mir.Value{Kind: mir.ValRef, Ref: addr}, Val: mir.Value{Kind: mir.ValRef, Ref: hp.Name}})
				env[hp.Name] = true
			}

			blocks := []*mir.BasicBlock{entry}

			// 関数本文の lowering.
			if fd.Body != nil {
				var ctx *lowerCtx = nil

				lowerHIRStmtBlock(fd.Body, &blocks, &entry, newTemp, env, ctx)
			}

			// 関数末尾に ret を保証.
			ensureTerminator(entry)

			f.Blocks = blocks
			m.Functions = append(m.Functions, f)
		}
	}

	// If no functions discovered, synthesize a trivial main returning 0.
	if len(m.Functions) == 0 {
		f := &mir.Function{Name: "main"}
		bb := &mir.BasicBlock{Name: "entry"}
		zero := mir.Value{Kind: mir.ValConstInt, Int64: 0}
		bb.Instr = append(bb.Instr, mir.Ret{Val: &zero})
		f.Blocks = []*mir.BasicBlock{bb}
		m.Functions = []*mir.Function{f}
	}

	return m
}

// lowerHIRExprToValue lowers a subset of HIR expressions to MIR immediate values.
// Returns (value, true) on success; otherwise (zero, false).
func lowerHIRExprToValue(e hir.HIRExpression) (mir.Value, bool) {
	switch v := e.(type) {
	case *hir.HIRLiteral:
		switch lit := v.Value.(type) {
		case int:
			return mir.Value{Kind: mir.ValConstInt, Int64: int64(lit), Class: mir.ClassInt}, true
		case int32:
			return mir.Value{Kind: mir.ValConstInt, Int64: int64(lit), Class: mir.ClassInt}, true
		case int64:
			return mir.Value{Kind: mir.ValConstInt, Int64: lit, Class: mir.ClassInt}, true
		case uint:
			return mir.Value{Kind: mir.ValConstInt, Int64: int64(lit), Class: mir.ClassInt}, true
		case uint32:
			return mir.Value{Kind: mir.ValConstInt, Int64: int64(lit), Class: mir.ClassInt}, true
		case uint64:
			// best-effort: clamp to int64.
			return mir.Value{Kind: mir.ValConstInt, Int64: int64(lit), Class: mir.ClassInt}, true
		case float32:
			return mir.Value{Kind: mir.ValConstFloat, Float64: float64(lit), Class: mir.ClassFloat}, true
		case float64:
			return mir.Value{Kind: mir.ValConstFloat, Float64: lit, Class: mir.ClassFloat}, true
		case bool:
			if lit {
				return mir.Value{Kind: mir.ValConstInt, Int64: 1, Class: mir.ClassInt}, true
			}

			return mir.Value{Kind: mir.ValConstInt, Int64: 0, Class: mir.ClassInt}, true
		case string:
			// String literals are now supported in MIR.
			return mir.Value{Kind: mir.ValConstString, StrVal: lit, Class: mir.ClassString}, true
		default:
			return mir.Value{}, false
		}
	default:
		return mir.Value{}, false
	}
}

// typeToClass maps HIR TypeInfo to MIR ValueClass (best-effort).
func typeToClass(t hir.TypeInfo) mir.ValueClass {
	switch t.Kind {
	case hir.TypeKindFloat:
		return mir.ClassFloat
	case hir.TypeKindInteger, hir.TypeKindPointer, hir.TypeKindBoolean:
		return mir.ClassInt
	default:
		return mir.ClassUnknown
	}
}

// typeToArgClassStr: more precise class label for calls: "f32","f64","int".
func typeToArgClassStr(t hir.TypeInfo) string {
	if t.Kind == hir.TypeKindFloat {
		if t.Size == 4 {
			return "f32"
		}
		// default f64 for floats.
		return "f64"
	}
	// Treat bool/pointer/integer as int class
	return "int"
}

// SelectToLIR performs naive selection from MIR to target-agnostic LIR.
func SelectToLIR(m *mir.Module) *lir.Module {
	lm := &lir.Module{Name: m.Name}

	for _, f := range m.Functions {
		lf := &lir.Function{Name: f.Name}

		for _, bb := range f.Blocks {
			lb := &lir.BasicBlock{Label: bb.Name}

			for _, in := range bb.Instr {
				switch v := in.(type) {
				case mir.Ret:
					var src string
					if v.Val != nil {
						src = v.Val.String()
					}

					lb.Insns = append(lb.Insns, lir.Ret{Src: src})
				case mir.BinOp:
					dst := v.Dst
					if dst == "" {
						dst = "%t0"
					}

					switch v.Op.String() {
					case "add":
						lb.Insns = append(lb.Insns, lir.Add{Dst: dst, LHS: v.LHS.String(), RHS: v.RHS.String()})
					case "sub":
						lb.Insns = append(lb.Insns, lir.Sub{Dst: dst, LHS: v.LHS.String(), RHS: v.RHS.String()})
					case "mul":
						lb.Insns = append(lb.Insns, lir.Mul{Dst: dst, LHS: v.LHS.String(), RHS: v.RHS.String()})
					case "div":
						lb.Insns = append(lb.Insns, lir.Div{Dst: dst, LHS: v.LHS.String(), RHS: v.RHS.String()})
					}
				case mir.Call:
					args := make([]string, 0, len(v.Args))
					argClasses := make([]string, 0, len(v.Args))

					for _, a := range v.Args {
						args = append(args, a.String())
						// Prefer MIR call's ArgClasses if present later; here derive from value class.
						switch a.Class {
						case mir.ClassFloat:
							// fallback float width unknown -> f64.
							argClasses = append(argClasses, "f64")
						case mir.ClassInt:
							argClasses = append(argClasses, "int")
						default:
							argClasses = append(argClasses, "")
						}
					}

					callee := v.Callee
					if v.CalleeVal != nil {
						callee = v.CalleeVal.String()
					}
					// If MIR call carries ArgClasses/RetClass, prefer them
					retClass := v.RetClass
					// Try to use v.ArgClasses if provided
					useArgClasses := argClasses
					if len(v.ArgClasses) > 0 {
						useArgClasses = v.ArgClasses
					}

					lb.Insns = append(lb.Insns, lir.Call{Dst: v.Dst, Callee: callee, Args: args, ArgClasses: useArgClasses, RetClass: retClass})
				case mir.Alloca:
					lb.Insns = append(lb.Insns, lir.Alloc{Dst: v.Dst, Name: v.Name})
				case mir.Load:
					lb.Insns = append(lb.Insns, lir.Load{Dst: v.Dst, Addr: v.Addr.String()})
				case mir.Store:
					lb.Insns = append(lb.Insns, lir.Store{Addr: v.Addr.String(), Val: v.Val.String()})
				case mir.Cmp:
					lb.Insns = append(lb.Insns, lir.Cmp{Dst: v.Dst, Pred: v.Pred.String(), LHS: v.LHS.String(), RHS: v.RHS.String()})
				case mir.Br:
					lb.Insns = append(lb.Insns, lir.Br{Target: v.Target})
				case mir.CondBr:
					lb.Insns = append(lb.Insns, lir.BrCond{Cond: v.Cond.String(), True: v.True, False: v.False})
				default:
					// ignore unknown for now.
				}
			}

			lf.Blocks = append(lf.Blocks, lb)
		}

		lm.Functions = append(lm.Functions, lf)
	}

	return lm
}

// lowerHIRExpr は MIR 命令を必要に応じて追加しつつ、値を返す簡易評価器.
func lowerHIRExpr(e hir.HIRExpression, newTemp func() string, bb *mir.BasicBlock, env map[string]bool) (mir.Value, bool) {
	// 1) まず即値に落とせるなら即値.
	if v, ok := lowerHIRExprToValue(e); ok {
		return v, true
	}
	// 2) 単項 -x: 即値なら畳み込み、一般の場合は 0 - x または -1 * x.
	if ue, ok := e.(*hir.HIRUnaryExpression); ok && ue.Operator == "-" {
		if vv, ok := lowerHIRExprToValue(ue.Operand); ok {
			switch vv.Kind {
			case mir.ValConstInt:
				vv.Int64 = -vv.Int64

				return vv, true
			case mir.ValConstFloat:
				vv.Float64 = -vv.Float64

				return vv, true
			}
		}
		// 一般ケース.
		ov, ok := lowerHIRExpr(ue.Operand, newTemp, bb, env)
		if !ok {
			return mir.Value{}, false
		}
		// 型からクラスを決定.
		cls := typeToClass(ue.Operand.GetType())
		dst := newTemp()

		if cls == mir.ClassFloat {
			// 0.0 - x
			zero := mir.Value{Kind: mir.ValConstFloat, Float64: 0.0, Class: mir.ClassFloat}
			bb.Instr = append(bb.Instr, mir.BinOp{Dst: dst, Op: mir.OpSub, LHS: zero, RHS: ov})

			return mir.Value{Kind: mir.ValRef, Ref: dst, Class: mir.ClassFloat}, true
		}
		// 整数: 0 - x.
		zero := mir.Value{Kind: mir.ValConstInt, Int64: 0, Class: mir.ClassInt}
		bb.Instr = append(bb.Instr, mir.BinOp{Dst: dst, Op: mir.OpSub, LHS: zero, RHS: ov})

		return mir.Value{Kind: mir.ValRef, Ref: dst, Class: mir.ClassInt}, true
	}
	// 2.0) 単項 +x は恒等（そのまま下ろす）
	if ue, ok := e.(*hir.HIRUnaryExpression); ok && ue.Operator == "+" {
		return lowerHIRExpr(ue.Operand, newTemp, bb, env)
	}
	// 2.1) 単項 論理否定 !x を値として評価（0/1 を返す）
	if ue, ok := e.(*hir.HIRUnaryExpression); ok && ue.Operator == "!" {
		return lowerHIRCond(e, newTemp, bb, env)
	}
	// 2.1.1) 単項 ビット反転 ~x を値として評価（xor -1）
	if ue, ok := e.(*hir.HIRUnaryExpression); ok && ue.Operator == "~" {
		// try immediate.
		if vv, ok := lowerHIRExprToValue(ue.Operand); ok && vv.Kind == mir.ValConstInt {
			return mir.Value{Kind: mir.ValConstInt, Int64: ^vv.Int64, Class: mir.ClassInt}, true
		}
		// general case: tmp = x ^ -1.
		v, ok := lowerHIRExpr(ue.Operand, newTemp, bb, env)
		if !ok {
			return mir.Value{}, false
		}

		dst := newTemp()
		bb.Instr = append(bb.Instr, mir.BinOp{Dst: dst, Op: mir.OpXor, LHS: v, RHS: mir.Value{Kind: mir.ValConstInt, Int64: -1, Class: mir.ClassInt}})

		return mir.Value{Kind: mir.ValRef, Ref: dst, Class: mir.ClassInt}, true
	}
	// 2.2) アドレス演算子 &x（lvalue に限定）
	if ue, ok := e.(*hir.HIRUnaryExpression); ok && ue.Operator == "&" {
		// &identifier.
		if id, ok := ue.Operand.(*hir.HIRIdentifier); ok {
			if id.Name != "" {
				if env != nil && env[id.Name] {
					return mir.Value{Kind: mir.ValRef, Ref: fmt.Sprintf("%%%s.addr", id.Name), Class: mir.ClassInt}, true
				}
				// 非ローカルはシンボル参照として扱う（ベストエフォート）.
				return mir.Value{Kind: mir.ValRef, Ref: id.Name, Class: mir.ClassInt}, true
			}

			return mir.Value{}, false
		}
		// &(a[i]) / &(a.b)
		if addr, ok := lowerAddressOf(ue.Operand, newTemp, bb, env); ok {
			return addr, true
		}
		// &*p == p の簡易最適化.
		if innerU, ok := ue.Operand.(*hir.HIRUnaryExpression); ok && innerU.Operator == "*" {
			return lowerHIRExpr(innerU.Operand, newTemp, bb, env)
		}

		return mir.Value{}, false
	}
	// 2.3) デリファレンス *p
	if ue, ok := e.(*hir.HIRUnaryExpression); ok && ue.Operator == "*" {
		ptr, okp := lowerHIRExpr(ue.Operand, newTemp, bb, env)
		if !okp {
			return mir.Value{}, false
		}

		tmp := newTemp()
		bb.Instr = append(bb.Instr, mir.Load{Dst: tmp, Addr: ptr})
		// Determine class from pointer target type if available.
		cls := ptr.Class
		pt := ue.Operand.GetType()

		if pt.Kind == hir.TypeKindPointer && len(pt.Parameters) > 0 {
			cls = typeToClass(pt.Parameters[0])
		}

		return mir.Value{Kind: mir.ValRef, Ref: tmp, Class: cls}, true
	}
	// 3) 二項式（即値畳み込み、それ以外はオペランドを評価して BinOp を発行）.
	if be, ok := e.(*hir.HIRBinaryExpression); ok {
		if os.Getenv("ORIZON_DEBUG_LOWER_OPS") != "" {
			fmt.Fprintf(os.Stderr, "[lower] HIRBinaryExpression op=%q\n", be.Operator)
		}
		// 論理演算子は値コンテキストでも0/1の値として生成
		if be.Operator == "&&" || be.Operator == "||" {
			return lowerHIRCond(e, newTemp, bb, env)
		}
		// 代入および複合代入（式コンテキスト）.
		if be.Operator == "=" || be.Operator == "+=" || be.Operator == "-=" || be.Operator == "*=" || be.Operator == "/=" || be.Operator == "%=" || be.Operator == "&=" || be.Operator == "|=" || be.Operator == "^=" || be.Operator == "<<=" || be.Operator == ">>=" {
			// 左辺のアドレス算出（識別子・添字・フィールド・*ptr に対応）.
			// helper: address of LHS.
			getLHSAddr := func(lhs hir.HIRExpression) (mir.Value, bool) {
				switch t := lhs.(type) {
				case *hir.HIRIdentifier:
					if t.Name == "" {
						return mir.Value{}, false
					}

					addr := fmt.Sprintf("%%%s.addr", t.Name)
					if !env[t.Name] {
						// 未宣言なら割り当て（ベストエフォート）.
						bb.Instr = append(bb.Instr, mir.Alloca{Dst: addr, Name: t.Name})
						env[t.Name] = true
					}

					return mir.Value{Kind: mir.ValRef, Ref: addr, Class: mir.ClassInt}, true
				case *hir.HIRIndexExpression, *hir.HIRFieldExpression:
					return lowerAddressOf(t, newTemp, bb, env)
				case *hir.HIRUnaryExpression:
					if t.Operator == "*" {
						return lowerAddressOf(t.Operand, newTemp, bb, env)
					}

					return mir.Value{}, false
				default:
					return mir.Value{}, false
				}
			}

			addr, okA := getLHSAddr(be.Left)
			if !okA {
				return mir.Value{}, false
			}
			// RHS を評価.
			var rhs mir.Value
			if v, ok := lowerHIRExprToValue(be.Right); ok {
				rhs = v
			} else if v, ok := lowerHIRExpr(be.Right, newTemp, bb, env); ok {
				rhs = v
			} else {
				return mir.Value{}, false
			}
			// 複合代入なら load -> binop.
			if be.Operator != "=" {
				tmp := newTemp()
				bb.Instr = append(bb.Instr, mir.Load{Dst: tmp, Addr: addr})
				curVal := mir.Value{Kind: mir.ValRef, Ref: tmp, Class: mir.ClassInt}
				// map operator to BinOp.
				var op mir.BinOpKind

				switch be.Operator {
				case "+=":
					op = mir.OpAdd
				case "-=":
					op = mir.OpSub
				case "*=":
					op = mir.OpMul
				case "/=":
					op = mir.OpDiv
				case "%=":
					op = mir.OpMod
				case "&=":
					op = mir.OpAnd
				case "|=":
					op = mir.OpOr
				case "^=":
					op = mir.OpXor
				case "<<=":
					op = mir.OpShl
				case ">>=":
					op = mir.OpShr
				}

				dst := newTemp()
				bb.Instr = append(bb.Instr, mir.BinOp{Dst: dst, Op: op, LHS: curVal, RHS: rhs})
				// store back.
				bb.Instr = append(bb.Instr, mir.Store{Addr: addr, Val: mir.Value{Kind: mir.ValRef, Ref: dst, Class: mir.ClassInt}})

				return mir.Value{Kind: mir.ValRef, Ref: dst, Class: mir.ClassInt}, true
			}
			// 単純代入.
			bb.Instr = append(bb.Instr, mir.Store{Addr: addr, Val: rhs})

			return rhs, true
		}

		lhs, lOK := lowerHIRExprToValue(be.Left)
		rhs, rOK := lowerHIRExprToValue(be.Right)

		if lOK && rOK {
			// 簡易定数畳み込み.
			switch be.Operator {
			case "+":
				if lhs.Kind == mir.ValConstInt && rhs.Kind == mir.ValConstInt {
					return mir.Value{Kind: mir.ValConstInt, Int64: lhs.Int64 + rhs.Int64}, true
				}

				if lhs.Kind == mir.ValConstFloat && rhs.Kind == mir.ValConstFloat {
					return mir.Value{Kind: mir.ValConstFloat, Float64: lhs.Float64 + rhs.Float64}, true
				}
			case "-":
				if lhs.Kind == mir.ValConstInt && rhs.Kind == mir.ValConstInt {
					return mir.Value{Kind: mir.ValConstInt, Int64: lhs.Int64 - rhs.Int64}, true
				}

				if lhs.Kind == mir.ValConstFloat && rhs.Kind == mir.ValConstFloat {
					return mir.Value{Kind: mir.ValConstFloat, Float64: lhs.Float64 - rhs.Float64}, true
				}
			case "*":
				if lhs.Kind == mir.ValConstInt && rhs.Kind == mir.ValConstInt {
					return mir.Value{Kind: mir.ValConstInt, Int64: lhs.Int64 * rhs.Int64}, true
				}

				if lhs.Kind == mir.ValConstFloat && rhs.Kind == mir.ValConstFloat {
					return mir.Value{Kind: mir.ValConstFloat, Float64: lhs.Float64 * rhs.Float64}, true
				}
			case "/":
				if lhs.Kind == mir.ValConstInt && rhs.Kind == mir.ValConstInt && rhs.Int64 != 0 {
					return mir.Value{Kind: mir.ValConstInt, Int64: lhs.Int64 / rhs.Int64}, true
				}

				if lhs.Kind == mir.ValConstFloat && rhs.Kind == mir.ValConstFloat && rhs.Float64 != 0 {
					return mir.Value{Kind: mir.ValConstFloat, Float64: lhs.Float64 / rhs.Float64}, true
				}
			case "%":
				if lhs.Kind == mir.ValConstInt && rhs.Kind == mir.ValConstInt && rhs.Int64 != 0 {
					return mir.Value{Kind: mir.ValConstInt, Int64: lhs.Int64 % rhs.Int64}, true
				}
			case "&":
				if lhs.Kind == mir.ValConstInt && rhs.Kind == mir.ValConstInt {
					return mir.Value{Kind: mir.ValConstInt, Int64: lhs.Int64 & rhs.Int64}, true
				}
			case "|":
				if lhs.Kind == mir.ValConstInt && rhs.Kind == mir.ValConstInt {
					return mir.Value{Kind: mir.ValConstInt, Int64: lhs.Int64 | rhs.Int64}, true
				}
			case "^":
				if lhs.Kind == mir.ValConstInt && rhs.Kind == mir.ValConstInt {
					return mir.Value{Kind: mir.ValConstInt, Int64: lhs.Int64 ^ rhs.Int64}, true
				}
			case "<<":
				if lhs.Kind == mir.ValConstInt && rhs.Kind == mir.ValConstInt {
					return mir.Value{Kind: mir.ValConstInt, Int64: lhs.Int64 << uint64(rhs.Int64), Class: mir.ClassInt}, true
				}
			case ">>":
				if lhs.Kind == mir.ValConstInt && rhs.Kind == mir.ValConstInt {
					return mir.Value{Kind: mir.ValConstInt, Int64: lhs.Int64 >> uint64(rhs.Int64), Class: mir.ClassInt}, true
				}
			}
		}
		// オペランドを一般に評価.
		lhv, okL := lowerHIRExpr(be.Left, newTemp, bb, env)
		rhv, okR := lowerHIRExpr(be.Right, newTemp, bb, env)

		if !okL || !okR {
			return mir.Value{}, false
		}
		// 比較演算子に対応.
		switch be.Operator {
		case "==", "!=", "<", "<=", ">", ">=":
			pred := selectPredByType(be.Operator, be.Left.GetType(), be.Right.GetType())
			dst := newTemp()
			bb.Instr = append(bb.Instr, mir.Cmp{Dst: dst, Pred: pred, LHS: lhv, RHS: rhv})

			return mir.Value{Kind: mir.ValRef, Ref: dst, Class: mir.ClassInt}, true
		case "+", "-", "*", "/", "%", "&", "|", "^", "<<", ">>":
			var op mir.BinOpKind

			switch be.Operator {
			case "+":
				op = mir.OpAdd
			case "-":
				op = mir.OpSub
			case "*":
				op = mir.OpMul
			case "/":
				op = mir.OpDiv
			case "%":
				op = mir.OpMod
			case "&":
				op = mir.OpAnd
			case "|":
				op = mir.OpOr
			case "^":
				op = mir.OpXor
			case "<<":
				op = mir.OpShl
			case ">>":
				op = mir.OpShr
			}

			dst := newTemp()

			cls := mir.ClassInt
			if lhv.Class == mir.ClassFloat || rhv.Class == mir.ClassFloat {
				cls = mir.ClassFloat
			}

			bb.Instr = append(bb.Instr, mir.BinOp{Dst: dst, Op: op, LHS: lhv, RHS: rhv})

			return mir.Value{Kind: mir.ValRef, Ref: dst, Class: cls}, true
		default:
			return mir.Value{}, false
		}
	}
	// 3.5) 添字/フィールドアクセス（RValue）: 新しいIndexLoad/FieldLoad命令を使用
	if ie, ok := e.(*hir.HIRIndexExpression); ok {
		arrayVal, arrayOk := lowerHIRExpr(ie.Array, newTemp, bb, env)
		indexVal, indexOk := lowerHIRExpr(ie.Index, newTemp, bb, env)

		if arrayOk && indexOk {
			tmp := newTemp()
			bb.Instr = append(bb.Instr, mir.IndexLoad{
				Dst:   tmp,
				Array: arrayVal,
				Index: indexVal,
			})

			return mir.Value{Kind: mir.ValRef, Ref: tmp, Class: typeToClass(ie.GetType())}, true
		}

		return mir.Value{}, false
	}

	if fe, ok := e.(*hir.HIRFieldExpression); ok {
		objectVal, objectOk := lowerHIRExpr(fe.Object, newTemp, bb, env)
		if objectOk {
			tmp := newTemp()
			bb.Instr = append(bb.Instr, mir.FieldLoad{
				Dst:    tmp,
				Object: objectVal,
				Field:  fe.Field,
			})

			return mir.Value{Kind: mir.ValRef, Ref: tmp, Class: typeToClass(fe.GetType())}, true
		}

		return mir.Value{}, false
	}
	// 4) 識別子参照：ローカルスロットがある場合は load、それ以外は参照をそのまま返す.
	if id, ok := e.(*hir.HIRIdentifier); ok {
		if id.Name != "" {
			if env != nil && env[id.Name] {
				addrRef := fmt.Sprintf("%%%s.addr", id.Name)
				tmp := newTemp()
				bb.Instr = append(bb.Instr, mir.Load{Dst: tmp, Addr: mir.Value{Kind: mir.ValRef, Ref: addrRef}})

				return mir.Value{Kind: mir.ValRef, Ref: tmp, Class: typeToClass(id.GetType())}, true
			}

			return mir.Value{Kind: mir.ValRef, Ref: id.Name, Class: typeToClass(id.GetType())}, true
		}
	}
	// 5) 関数呼び出し.
	if ce, ok := e.(*hir.HIRCallExpression); ok {
		// callee: 識別子なら名前、そうでなければ値として間接呼び出し.
		callee := ""

		var calleeVal *mir.Value

		if id, ok := ce.Function.(*hir.HIRIdentifier); ok {
			callee = id.Name
			// Check if it's a built-in function and map to assembly name.
			if IsBuiltinFunction(callee) {
				if builtin, exists := GetBuiltinFunction(callee); exists {
					callee = builtin.AssemblyName
				}
			}
		} else {
			if v, ok := lowerHIRExpr(ce.Function, newTemp, bb, env); ok {
				calleeVal = &v
			} else {
				return mir.Value{}, false
			}
		}
		// 引数を順に評価（必要なら先に命令を発行）.
		args := make([]mir.Value, 0, len(ce.Arguments))
		argClasses := make([]string, 0, len(ce.Arguments))
		tempArgs := 0
		// newTemp は呼び出しの中でも利用する（外のカウンタと衝突しないよう一時的に別カウンタを使用）.
		localNewTemp := func() string {
			t := fmt.Sprintf("%%ac%d", tempArgs)
			tempArgs++
			return t
		}

		for _, a := range ce.Arguments {
			if v, ok := lowerHIRExprToValue(a); ok {
				args = append(args, v)
				argClasses = append(argClasses, typeToArgClassStr(a.GetType()))
			} else if v, ok := lowerHIRExpr(a, localNewTemp, bb, env); ok {
				args = append(args, v)
				argClasses = append(argClasses, typeToArgClassStr(a.GetType()))
			} else {
				return mir.Value{}, false
			}
		}
		// 戻り値用に一時を確保して Call を発行.
		dst := localNewTemp()
		// Determine return class from HIR call expression type (string).
		rclsStr := typeToArgClassStr(ce.GetType())
		// MIR.Callに詳細クラスを付与
		bb.Instr = append(bb.Instr, mir.Call{Dst: dst, Callee: callee, CalleeVal: calleeVal, Args: args, ArgClasses: argClasses, RetClass: rclsStr})
		// 値としては大分類のClassも設定.
		rvc := typeToClass(ce.GetType())

		return mir.Value{Kind: mir.ValRef, Ref: dst, Class: rvc}, true
	}
	// 6) キャスト（当面は値をそのまま通す）.
	if ce, ok := e.(*hir.HIRCastExpression); ok {
		return lowerHIRExpr(ce.Expression, newTemp, bb, env)
	}

	return mir.Value{}, false
}

// lowerHIRCond lowers a boolean-like HIR expression to a MIR value, emitting compares if needed.
func lowerHIRCond(e hir.HIRExpression, newTemp func() string, bb *mir.BasicBlock, env map[string]bool) (mir.Value, bool) {
	// If already an int/bool literal, reuse as condition (0=false, nonzero=true)
	if v, ok := lowerHIRExprToValue(e); ok {
		return v, true
	}
	// Logical NOT.
	if ue, ok := e.(*hir.HIRUnaryExpression); ok && ue.Operator == "!" {
		val, okv := lowerHIRCond(ue.Operand, newTemp, bb, env)
		if !okv {
			return mir.Value{}, false
		}
		// result = (val == 0).
		dst := newTemp()
		bb.Instr = append(bb.Instr, mir.Cmp{Dst: dst, Pred: mir.CmpEQ, LHS: val, RHS: mir.Value{Kind: mir.ValConstInt, Int64: 0, Class: mir.ClassInt}})

		return mir.Value{Kind: mir.ValRef, Ref: dst, Class: mir.ClassInt}, true
	}
	// Binary comparison operators.
	if be, ok := e.(*hir.HIRBinaryExpression); ok {
		// Short-circuit-less logical lowering using 0/1 arithmetic
		if be.Operator == "&&" || be.Operator == "||" {
			l, okL := lowerHIRCond(be.Left, newTemp, bb, env)
			r, okR := lowerHIRCond(be.Right, newTemp, bb, env)

			if !okL || !okR {
				return mir.Value{}, false
			}

			comb := newTemp()
			if be.Operator == "&&" {
				// and: multiply 0/1
				bb.Instr = append(bb.Instr, mir.BinOp{Dst: comb, Op: mir.OpMul, LHS: l, RHS: r})
			} else {
				// or: add 0/1
				bb.Instr = append(bb.Instr, mir.BinOp{Dst: comb, Op: mir.OpAdd, LHS: l, RHS: r})
			}
			// clamp to boolean 0/1: comb != 0
			zero := mir.Value{Kind: mir.ValConstInt, Int64: 0, Class: mir.ClassInt}
			dst := newTemp()
			bb.Instr = append(bb.Instr, mir.Cmp{Dst: dst, Pred: mir.CmpNE, LHS: mir.Value{Kind: mir.ValRef, Ref: comb}, RHS: zero})

			return mir.Value{Kind: mir.ValRef, Ref: dst, Class: mir.ClassInt}, true
		}

		var pred mir.CmpPred
		switch be.Operator {
		case "==", "!=", "<", "<=", ">", ">=":
			pred = selectPredByType(be.Operator, be.Left.GetType(), be.Right.GetType())
		default:
			return mir.Value{}, false
		}

		lhs, okL := lowerHIRExpr(be.Left, newTemp, bb, env)
		rhs, okR := lowerHIRExpr(be.Right, newTemp, bb, env)

		if !okL || !okR {
			return mir.Value{}, false
		}

		dst := newTemp()
		bb.Instr = append(bb.Instr, mir.Cmp{Dst: dst, Pred: pred, LHS: lhs, RHS: rhs})

		return mir.Value{Kind: mir.ValRef, Ref: dst, Class: mir.ClassInt}, true
	}
	// Fallback to general expression evaluation.
	if v, ok := lowerHIRExpr(e, newTemp, bb, env); ok {
		return v, true
	}

	return mir.Value{}, false
}

// ensureTerminator ensures the basic block ends with a terminator; if none, append `ret`.
func ensureTerminator(bb *mir.BasicBlock) {
	if bb == nil || len(bb.Instr) == 0 {
		bb.Instr = append(bb.Instr, mir.Ret{Val: nil})

		return
	}

	switch bb.Instr[len(bb.Instr)-1].(type) {
	case mir.Ret, mir.Br, mir.CondBr:
		return
	default:
		bb.Instr = append(bb.Instr, mir.Ret{Val: nil})
	}
}

// lowerHIRStmtBlock lowers a HIRBlockStatement into MIR, appending new blocks as needed.
func lowerHIRStmtBlock(bs *hir.HIRBlockStatement, blocks *[]*mir.BasicBlock, cur **mir.BasicBlock, newTemp func() string, env map[string]bool, ctx *lowerCtx) {
	if bs == nil {
		return
	}

	for _, st := range bs.Statements {
		lowerHIRStmt(st, blocks, cur, newTemp, env, ctx)
		// if current block was terminated (ret/br), create a fresh fallthrough block to continue
		if len((*cur).Instr) > 0 {
			switch (*cur).Instr[len((*cur).Instr)-1].(type) {
			case mir.Ret, mir.Br, mir.CondBr:
				nb := &mir.BasicBlock{Name: newBlockLabel("cont", newTemp)}
				*blocks = append(*blocks, nb)
				*cur = nb
			}
		}
	}
}

func newBlockLabel(prefix string, newTemp func() string) string {
	t := newTemp()
	// replace leading '%' for label friendliness.
	if len(t) > 0 && t[0] == '%' {
		t = t[1:]
	}

	return fmt.Sprintf("%s_%s", prefix, t)
}

// env for locals: map identifier -> address ref name.
// (reserved) localEnv: identifier -> address ref name (not used currently).

// loop context for break/continue targets.
type lowerCtx struct {
	breakLabel    string
	continueLabel string
}

// lowerHIRStmt lowers a single HIR statement into MIR.
func lowerHIRStmt(st hir.HIRStatement, blocks *[]*mir.BasicBlock, cur **mir.BasicBlock, newTemp func() string, env map[string]bool, ctx *lowerCtx) {
	switch s := st.(type) {
	case *hir.HIRReturnStatement:
		var retVal *mir.Value

		if s.Expression != nil {
			if v, ok := lowerHIRExprToValue(s.Expression); ok {
				retVal = &v
			} else if v, ok := lowerHIRExpr(s.Expression, newTemp, *cur, env); ok {
				retVal = &v
			}
		}

		(*cur).Instr = append((*cur).Instr, mir.Ret{Val: retVal})
	case *hir.HIRExpressionStatement:
		// Evaluate expression for side effects; discard result.
		if s.Expression != nil {
			if _, ok := lowerHIRExpr(s.Expression, newTemp, *cur, env); !ok {
				// try fold immediate and ignore.
				_, _ = lowerHIRExprToValue(s.Expression)
			}
		}
	case *hir.HIRAssignStatement:
		// currently support simple identifier target.
		// compute RHS.
		var rhs mir.Value
		if v, ok := lowerHIRExprToValue(s.Value); ok {
			rhs = v
		} else if v, ok := lowerHIRExpr(s.Value, newTemp, *cur, env); ok {
			rhs = v
		}
		// target address.
		switch tgt := s.Target.(type) {
		case *hir.HIRIdentifier:
			name := tgt.Name
			addr := fmt.Sprintf("%%%s.addr", name)

			if !env[name] {
				// first time seeing this local; allocate.
				(*cur).Instr = append((*cur).Instr, mir.Alloca{Dst: addr, Name: name})
				env[name] = true
			}
			// compound assignment: load, op, store.
			if s.Operator != "=" {
				// load current.
				tmp := newTemp()
				(*cur).Instr = append((*cur).Instr, mir.Load{Dst: tmp, Addr: mir.Value{Kind: mir.ValRef, Ref: addr}})
				curVal := mir.Value{Kind: mir.ValRef, Ref: tmp}

				var op mir.BinOpKind
				switch s.Operator {
				case "+=":
					op = mir.OpAdd
				case "-=":
					op = mir.OpSub
				case "*=":
					op = mir.OpMul
				case "/=":
					op = mir.OpDiv
				default:
					// fallback to plain store.
					(*cur).Instr = append((*cur).Instr, mir.Store{Addr: mir.Value{Kind: mir.ValRef, Ref: addr}, Val: rhs})

					return
				}

				dst := newTemp()
				(*cur).Instr = append((*cur).Instr, mir.BinOp{Dst: dst, Op: op, LHS: curVal, RHS: rhs})
				(*cur).Instr = append((*cur).Instr, mir.Store{Addr: mir.Value{Kind: mir.ValRef, Ref: addr}, Val: mir.Value{Kind: mir.ValRef, Ref: dst}})

				return
			}

			(*cur).Instr = append((*cur).Instr, mir.Store{Addr: mir.Value{Kind: mir.ValRef, Ref: addr}, Val: rhs})
		case *hir.HIRIndexExpression:
			if addr, ok := lowerAddressOf(tgt, newTemp, *cur, env); ok {
				// compound ops on memory location.
				if s.Operator != "=" {
					tmp := newTemp()
					(*cur).Instr = append((*cur).Instr, mir.Load{Dst: tmp, Addr: addr})
					curVal := mir.Value{Kind: mir.ValRef, Ref: tmp}

					var op mir.BinOpKind
					switch s.Operator {
					case "+=":
						op = mir.OpAdd
					case "-=":
						op = mir.OpSub
					case "*=":
						op = mir.OpMul
					case "/=":
						op = mir.OpDiv
					default:
						(*cur).Instr = append((*cur).Instr, mir.Store{Addr: addr, Val: rhs})

						return
					}

					dst := newTemp()
					(*cur).Instr = append((*cur).Instr, mir.BinOp{Dst: dst, Op: op, LHS: curVal, RHS: rhs})
					(*cur).Instr = append((*cur).Instr, mir.Store{Addr: addr, Val: mir.Value{Kind: mir.ValRef, Ref: dst}})

					return
				}

				(*cur).Instr = append((*cur).Instr, mir.Store{Addr: addr, Val: rhs})
			}
		case *hir.HIRFieldExpression:
			if addr, ok := lowerAddressOf(tgt, newTemp, *cur, env); ok {
				if s.Operator != "=" {
					tmp := newTemp()
					(*cur).Instr = append((*cur).Instr, mir.Load{Dst: tmp, Addr: addr})
					curVal := mir.Value{Kind: mir.ValRef, Ref: tmp}

					var op mir.BinOpKind
					switch s.Operator {
					case "+=":
						op = mir.OpAdd
					case "-=":
						op = mir.OpSub
					case "*=":
						op = mir.OpMul
					case "/=":
						op = mir.OpDiv
					default:
						(*cur).Instr = append((*cur).Instr, mir.Store{Addr: addr, Val: rhs})

						return
					}

					dst := newTemp()
					(*cur).Instr = append((*cur).Instr, mir.BinOp{Dst: dst, Op: op, LHS: curVal, RHS: rhs})
					(*cur).Instr = append((*cur).Instr, mir.Store{Addr: addr, Val: mir.Value{Kind: mir.ValRef, Ref: dst}})

					return
				}

				(*cur).Instr = append((*cur).Instr, mir.Store{Addr: addr, Val: rhs})
			}
		case *hir.HIRUnaryExpression:
			// *ptr = rhs / compound
			if tgt.Operator == "*" {
				if addr, ok := lowerAddressOf(tgt.Operand, newTemp, *cur, env); ok {
					if s.Operator != "=" {
						tmp := newTemp()
						(*cur).Instr = append((*cur).Instr, mir.Load{Dst: tmp, Addr: addr})
						curVal := mir.Value{Kind: mir.ValRef, Ref: tmp}

						var op mir.BinOpKind
						switch s.Operator {
						case "+=":
							op = mir.OpAdd
						case "-=":
							op = mir.OpSub
						case "*=":
							op = mir.OpMul
						case "/=":
							op = mir.OpDiv
						default:
							(*cur).Instr = append((*cur).Instr, mir.Store{Addr: addr, Val: rhs})

							return
						}

						dst := newTemp()
						(*cur).Instr = append((*cur).Instr, mir.BinOp{Dst: dst, Op: op, LHS: curVal, RHS: rhs})
						(*cur).Instr = append((*cur).Instr, mir.Store{Addr: addr, Val: mir.Value{Kind: mir.ValRef, Ref: dst}})

						return
					}

					(*cur).Instr = append((*cur).Instr, mir.Store{Addr: addr, Val: rhs})
				}
			}
		default:
			// Handle field/index assignment
			rhs, _ := lowerHIRExpr(s.Value, newTemp, *cur, env)

			switch target := s.Target.(type) {
			case *hir.HIRIndexExpression:
				// Array index assignment: arr[idx] = value.
				arrayVal, _ := lowerHIRExpr(target.Array, newTemp, *cur, env)
				indexVal, _ := lowerHIRExpr(target.Index, newTemp, *cur, env)

				if s.Operator == "=" {
					// Simple assignment.
					(*cur).Instr = append((*cur).Instr, mir.IndexStore{
						Array: arrayVal,
						Index: indexVal,
						Val:   rhs,
					})
				} else {
					// Compound assignment (+=, -=, etc.)
					// First load the current value.
					tmpDst := newTemp()
					(*cur).Instr = append((*cur).Instr, mir.IndexLoad{
						Dst:   tmpDst,
						Array: arrayVal,
						Index: indexVal,
					})

					// Perform the operation.
					op, ok := getBinOpFromAssignOp(s.Operator)
					if ok {
						opDst := newTemp()
						currentVal := mir.Value{Kind: mir.ValRef, Ref: tmpDst}
						(*cur).Instr = append((*cur).Instr, mir.BinOp{
							Dst: opDst,
							Op:  op,
							LHS: currentVal,
							RHS: rhs,
						})

						// Store the result back.
						(*cur).Instr = append((*cur).Instr, mir.IndexStore{
							Array: arrayVal,
							Index: indexVal,
							Val:   mir.Value{Kind: mir.ValRef, Ref: opDst},
						})
					}
				}

			case *hir.HIRFieldExpression:
				// Struct field assignment: obj.field = value
				objectVal, _ := lowerHIRExpr(target.Object, newTemp, *cur, env)

				if s.Operator == "=" {
					// Simple assignment.
					(*cur).Instr = append((*cur).Instr, mir.FieldStore{
						Object: objectVal,
						Field:  target.Field,
						Val:    rhs,
					})
				} else {
					// Compound assignment (+=, -=, etc.)
					// First load the current value.
					tmpDst := newTemp()
					(*cur).Instr = append((*cur).Instr, mir.FieldLoad{
						Dst:    tmpDst,
						Object: objectVal,
						Field:  target.Field,
					})

					// Perform the operation.
					op, ok := getBinOpFromAssignOp(s.Operator)
					if ok {
						opDst := newTemp()
						currentVal := mir.Value{Kind: mir.ValRef, Ref: tmpDst}
						(*cur).Instr = append((*cur).Instr, mir.BinOp{
							Dst: opDst,
							Op:  op,
							LHS: currentVal,
							RHS: rhs,
						})

						// Store the result back.
						(*cur).Instr = append((*cur).Instr, mir.FieldStore{
							Object: objectVal,
							Field:  target.Field,
							Val:    mir.Value{Kind: mir.ValRef, Ref: opDst},
						})
					}
				}
			}
		}
	case *hir.HIRIfStatement:
		// build then/else/end blocks
		thenLbl := newBlockLabel("then", newTemp)
		elseLbl := newBlockLabel("else", newTemp)
		endLbl := newBlockLabel("endif", newTemp)
		// cond (with short-circuit CFG).
		lowerCondToBr(s.Condition, blocks, cur, newTemp, env, thenLbl, elseLbl)
		thenBB := &mir.BasicBlock{Name: thenLbl}
		*blocks = append(*blocks, thenBB)
		// lower then.
		curThen := thenBB
		if s.ThenBlock != nil {
			lowerHIRStmt(s.ThenBlock, blocks, &curThen, newTemp, env, ctx)
		}
		// if-then falls through to end when not terminated.
		if len(curThen.Instr) == 0 || !isTerminator(curThen.Instr[len(curThen.Instr)-1]) {
			curThen.Instr = append(curThen.Instr, mir.Br{Target: endLbl})
		}
		// lower else.
		elseBB := &mir.BasicBlock{Name: elseLbl}
		*blocks = append(*blocks, elseBB)
		curElse := elseBB

		if s.ElseBlock != nil {
			lowerHIRStmt(s.ElseBlock, blocks, &curElse, newTemp, env, ctx)
		}

		if len(curElse.Instr) == 0 || !isTerminator(curElse.Instr[len(curElse.Instr)-1]) {
			curElse.Instr = append(curElse.Instr, mir.Br{Target: endLbl})
		}
		// create end and fallthrough to end.
		endBB := &mir.BasicBlock{Name: endLbl}
		*blocks = append(*blocks, endBB)
		*cur = endBB
	case *hir.HIRWhileStatement:
		head := newBlockLabel("while_head", newTemp)
		body := newBlockLabel("while_body", newTemp)
		tail := newBlockLabel("while_end", newTemp)
		// jump to head.
		(*cur).Instr = append((*cur).Instr, mir.Br{Target: head})
		headBB := &mir.BasicBlock{Name: head}
		*blocks = append(*blocks, headBB)
		// cond in head (with short-circuit CFG).
		lowerCondToBr(s.Condition, blocks, &headBB, newTemp, env, body, tail)
		// body.
		bodyBB := &mir.BasicBlock{Name: body}
		*blocks = append(*blocks, bodyBB)
		curBody := bodyBB

		if s.Body != nil {
			loopCtx := &lowerCtx{breakLabel: tail, continueLabel: head}
			lowerHIRStmt(s.Body, blocks, &curBody, newTemp, env, loopCtx)
		}
		// loop back.
		curBody.Instr = append(curBody.Instr, mir.Br{Target: head})
		// tail/end
		endBB := &mir.BasicBlock{Name: tail}
		*blocks = append(*blocks, endBB)
		*cur = endBB
	case *hir.HIRForStatement:
		// for (init; cond; update) body.
		head := newBlockLabel("for_head", newTemp)
		body := newBlockLabel("for_body", newTemp)
		update := newBlockLabel("for_update", newTemp)
		tail := newBlockLabel("for_end", newTemp)
		// init.
		if s.Init != nil {
			lowerHIRStmt(s.Init, blocks, cur, newTemp, env, ctx)
		}
		// jump to head.
		(*cur).Instr = append((*cur).Instr, mir.Br{Target: head})
		headBB := &mir.BasicBlock{Name: head}
		*blocks = append(*blocks, headBB)
		// cond (optional, with short-circuit CFG).
		if s.Condition != nil {
			lowerCondToBr(s.Condition, blocks, &headBB, newTemp, env, body, tail)
		} else {
			headBB.Instr = append(headBB.Instr, mir.Br{Target: body})
		}
		// body.
		bodyBB := &mir.BasicBlock{Name: body}
		*blocks = append(*blocks, bodyBB)
		curBody := bodyBB
		// continue goes to update if exists else head.
		contTarget := head
		if s.Update != nil {
			contTarget = update
		}

		loopCtx := &lowerCtx{breakLabel: tail, continueLabel: contTarget}
		if s.Body != nil {
			lowerHIRStmt(s.Body, blocks, &curBody, newTemp, env, loopCtx)
		}
		// at end of body, branch to update or head.
		if s.Update != nil {
			curBody.Instr = append(curBody.Instr, mir.Br{Target: update})
			// update block.
			updBB := &mir.BasicBlock{Name: update}
			*blocks = append(*blocks, updBB)
			curUpd := updBB
			lowerHIRStmt(s.Update, blocks, &curUpd, newTemp, env, loopCtx)
			curUpd.Instr = append(curUpd.Instr, mir.Br{Target: head})
		} else {
			curBody.Instr = append(curBody.Instr, mir.Br{Target: head})
		}
		// tail/end becomes current
		endBB := &mir.BasicBlock{Name: tail}
		*blocks = append(*blocks, endBB)
		*cur = endBB
	case *hir.HIRBreakStatement:
		if ctx != nil && ctx.breakLabel != "" {
			(*cur).Instr = append((*cur).Instr, mir.Br{Target: ctx.breakLabel})
		}
	case *hir.HIRContinueStatement:
		if ctx != nil && ctx.continueLabel != "" {
			(*cur).Instr = append((*cur).Instr, mir.Br{Target: ctx.continueLabel})
		}
	case *hir.HIRBlockStatement:
		lowerHIRStmtBlock(s, blocks, cur, newTemp, env, ctx)
	case *hir.HIRVariableDeclaration:
		// Handle variable declarations (let statements).
		name := s.Name
		addr := fmt.Sprintf("%%%s.addr", name)

		// Allocate space for the variable.
		(*cur).Instr = append((*cur).Instr, mir.Alloca{Dst: addr, Name: name})
		env[name] = true

		// If there's an initializer, store its value.
		if s.Initializer != nil {
			var initVal mir.Value
			if v, ok := lowerHIRExprToValue(s.Initializer); ok {
				initVal = v
			} else if v, ok := lowerHIRExpr(s.Initializer, newTemp, *cur, env); ok {
				initVal = v
			} else {
				// Fallback to zero value.
				initVal = mir.Value{Kind: mir.ValConstInt, Int64: 0}
			}

			(*cur).Instr = append((*cur).Instr, mir.Store{Addr: mir.Value{Kind: mir.ValRef, Ref: addr}, Val: initVal})
		}
	default:
		// Variable declarations belong to declaration pass.
	}
}

// helper: is terminator?.
func isTerminator(in mir.Instr) bool {
	switch in.(type) {
	case mir.Ret, mir.Br, mir.CondBr:
		return true
	default:
		return false
	}
}

// lowerCondToBr lowers a condition expression into branches with proper short-circuiting for && and ||.
// It emits control flow from the current block to either trueLbl or falseLbl.
func lowerCondToBr(e hir.HIRExpression, blocks *[]*mir.BasicBlock, cur **mir.BasicBlock, newTemp func() string, env map[string]bool, trueLbl, falseLbl string) {
	if e == nil {
		// No condition means always true (used by for (;;){}) callers would bypass this).
		(*cur).Instr = append((*cur).Instr, mir.Br{Target: trueLbl})

		return
	}
	// Handle logical negation by swapping targets.
	if ue, ok := e.(*hir.HIRUnaryExpression); ok && ue.Operator == "!" {
		lowerCondToBr(ue.Operand, blocks, cur, newTemp, env, falseLbl, trueLbl)

		return
	}
	// Short-circuit AND / OR
	if be, ok := e.(*hir.HIRBinaryExpression); ok {
		switch be.Operator {
		case "&&":
			rhsLbl := newBlockLabel("and_rhs", newTemp)
			// if LHS true -> evaluate RHS; else -> falseLbl.
			lowerCondToBr(be.Left, blocks, cur, newTemp, env, rhsLbl, falseLbl)
			// RHS block.
			rhsBB := &mir.BasicBlock{Name: rhsLbl}
			*blocks = append(*blocks, rhsBB)
			*cur = rhsBB
			lowerCondToBr(be.Right, blocks, cur, newTemp, env, trueLbl, falseLbl)

			return
		case "||":
			rhsLbl := newBlockLabel("or_rhs", newTemp)
			// if LHS true -> trueLbl; else -> evaluate RHS.
			lowerCondToBr(be.Left, blocks, cur, newTemp, env, trueLbl, rhsLbl)
			// RHS block.
			rhsBB := &mir.BasicBlock{Name: rhsLbl}
			*blocks = append(*blocks, rhsBB)
			*cur = rhsBB
			lowerCondToBr(be.Right, blocks, cur, newTemp, env, trueLbl, falseLbl)

			return
		}
	}
	// Fallback: compute condition value and emit conditional branch.
	if cond, ok := lowerHIRCond(e, newTemp, *cur, env); ok {
		(*cur).Instr = append((*cur).Instr, mir.CondBr{Cond: cond, True: trueLbl, False: falseLbl})
	} else {
		// If cannot lower, conservatively branch to false.
		(*cur).Instr = append((*cur).Instr, mir.Br{Target: falseLbl})
	}
}

// selectPredByType chooses a MIR compare predicate based on operator and operand types.
func selectPredByType(op string, lt, rt hir.TypeInfo) mir.CmpPred {
	// If either is float, use float compare.
	if lt.Kind == hir.TypeKindFloat || rt.Kind == hir.TypeKindFloat {
		switch op {
		case "<":
			return mir.CmpFLT
		case "<=":
			return mir.CmpFLE
		case ">":
			return mir.CmpFGT
		case ">=":
			return mir.CmpFGE
		case "==":
			return mir.CmpEQ
		case "!=":
			return mir.CmpNE
		}
	}
	// Pointer or unsigned integer: use unsigned predicates for ordering.
	isUnsignedLike := func(t hir.TypeInfo) bool {
		if t.Kind == hir.TypeKindPointer {
			return true
		}

		if t.Kind == hir.TypeKindInteger {
			// Heuristic: treat names starting with 'u' as unsigned (e.g., u32)
			if len(t.Name) > 0 && (t.Name[0] == 'u' || t.Name[0] == 'U') {
				return true
			}
		}

		return false
	}

	isUL := isUnsignedLike(lt) || isUnsignedLike(rt)
	switch op {
	case "<":
		if isUL {
			return mir.CmpULT
		}

		return mir.CmpSLT
	case "<=":
		if isUL {
			return mir.CmpULE
		}

		return mir.CmpSLE
	case ">":
		if isUL {
			return mir.CmpUGT
		}

		return mir.CmpSGT
	case ">=":
		if isUL {
			return mir.CmpUGE
		}

		return mir.CmpSGE
	case "==":
		return mir.CmpEQ
	case "!=":
		return mir.CmpNE
	default:
		return mir.CmpEQ
	}
}

// lowerAddressOf computes the address Value (mir.Value as reference) for index/field expressions.
func lowerAddressOf(e hir.HIRExpression, newTemp func() string, bb *mir.BasicBlock, env map[string]bool) (mir.Value, bool) {
	switch v := e.(type) {
	case *hir.HIRIdentifier:
		if v.Name == "" {
			return mir.Value{}, false
		}

		if env != nil && env[v.Name] {
			return mir.Value{Kind: mir.ValRef, Ref: fmt.Sprintf("%%%s.addr", v.Name), Class: mir.ClassInt}, true
		}
		// ベストエフォート: 非ローカルはシンボル参照をアドレスとして返す.
		return mir.Value{Kind: mir.ValRef, Ref: v.Name}, true
	case *hir.HIRUnaryExpression:
		if v.Operator == "*" { // address-of(deref) -> original pointer value
			return lowerHIRExpr(v.Operand, newTemp, bb, env)
		}
	case *hir.HIRIndexExpression:
		// base address resolution:.
		// 1) local identifier -> %name.addr
		// 2) recursively addressable -> lowerAddressOf(v.Array)
		// 3) expression value that is a pointer -> use as base address.
		// otherwise: fail (cannot take address).
		var baseVal mir.Value

		var haveBase bool

		if id, ok := v.Array.(*hir.HIRIdentifier); ok && env != nil && env[id.Name] {
			baseVal = mir.Value{Kind: mir.ValRef, Ref: fmt.Sprintf("%%%s.addr", id.Name), Class: mir.ClassInt}
			haveBase = true
		}

		if !haveBase {
			if addr, okA := lowerAddressOf(v.Array, newTemp, bb, env); okA {
				baseVal = addr
				haveBase = true
			}
		}

		if !haveBase {
			if val, okV := lowerHIRExpr(v.Array, newTemp, bb, env); okV {
				at := v.Array.GetType()
				if at.Kind == hir.TypeKindPointer {
					baseVal = val
					haveBase = true
				}
			}
		}

		if !haveBase {
			return mir.Value{}, false
		}
		// index value.
		idxVal, okI := lowerHIRExpr(v.Index, newTemp, bb, env)
		if !okI {
			return mir.Value{}, false
		}
		// element size: derive from the array/pointer/slice type on v.Array (fallback 1)
		elemSize := int64(1)

		if arrT := v.Array.GetType(); arrT.Kind == hir.TypeKindArray || arrT.Kind == hir.TypeKindSlice || arrT.Kind == hir.TypeKindPointer {
			if len(arrT.Parameters) > 0 {
				if sz := arrT.Parameters[0].Size; sz > 0 {
					elemSize = sz
				}
			}
		}
		// offset = idx * elemSize.
		idxTmp := idxVal
		// if size != 1, multiply.
		off := idxTmp

		if elemSize != 1 {
			szConst := mir.Value{Kind: mir.ValConstInt, Int64: elemSize, Class: mir.ClassInt}
			tmp := newTemp()
			bb.Instr = append(bb.Instr, mir.BinOp{Dst: tmp, Op: mir.OpMul, LHS: idxTmp, RHS: szConst})
			off = mir.Value{Kind: mir.ValRef, Ref: tmp}
		}
		// addr = base + off.
		addrTmp := newTemp()
		bb.Instr = append(bb.Instr, mir.BinOp{Dst: addrTmp, Op: mir.OpAdd, LHS: baseVal, RHS: off})

		return mir.Value{Kind: mir.ValRef, Ref: addrTmp, Class: mir.ClassInt}, true
	case *hir.HIRFieldExpression:
		// object/base address resolution:
		// 1) local identifier -> %name.addr
		// 2) recursively addressable -> lowerAddressOf(v.Object)
		// 3) expression value that is a pointer -> use as base address.
		var objVal mir.Value

		var haveBase bool

		if id, ok := v.Object.(*hir.HIRIdentifier); ok && env != nil && env[id.Name] {
			objVal = mir.Value{Kind: mir.ValRef, Ref: fmt.Sprintf("%%%s.addr", id.Name), Class: mir.ClassInt}
			haveBase = true
		}

		if !haveBase {
			if addr, okO := lowerAddressOf(v.Object, newTemp, bb, env); okO {
				objVal = addr
				haveBase = true
			}
		}

		if !haveBase {
			if val, okV := lowerHIRExpr(v.Object, newTemp, bb, env); okV {
				ot := v.Object.GetType()
				if ot.Kind == hir.TypeKindPointer {
					objVal = val
					haveBase = true
				}
			}
		}

		if !haveBase {
			return mir.Value{}, false
		}
		// find field offset if available.
		off := int64(0)

		if st := v.Object.GetType(); st.StructInfo != nil && len(st.StructInfo.Fields) > 0 {
			for _, f := range st.StructInfo.Fields {
				if f.Name == v.Field {
					off = f.Offset

					break
				}
			}
		} else if len(st.Fields) > 0 {
			for _, f := range st.Fields {
				if f.Name == v.Field {
					off = f.Offset

					break
				}
			}
		}

		if off == 0 {
			return objVal, true // base as address
		}

		offConst := mir.Value{Kind: mir.ValConstInt, Int64: off, Class: mir.ClassInt}
		addrTmp := newTemp()
		bb.Instr = append(bb.Instr, mir.BinOp{Dst: addrTmp, Op: mir.OpAdd, LHS: objVal, RHS: offConst})

		return mir.Value{Kind: mir.ValRef, Ref: addrTmp, Class: mir.ClassInt}, true
	default:
	}

	return mir.Value{}, false
}
