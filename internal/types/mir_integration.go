package types

import (
	"github.com/orizon-lang/orizon/internal/mir"
)

// CoreTypeMIRIntegration provides MIR-level code generation for core type operations.
type CoreTypeMIRIntegration struct {
	module *mir.Module
}

// NewCoreTypeMIRIntegration creates a new core type MIR integration.
func NewCoreTypeMIRIntegration(module *mir.Module) *CoreTypeMIRIntegration {
	return &CoreTypeMIRIntegration{
		module: module,
	}
}

// GenerateOptionOperations creates MIR functions for Option type operations.
func (ctm *CoreTypeMIRIntegration) GenerateOptionOperations() []*mir.Function {
	// Placeholder for Option operations.
	someFunc := &mir.Function{
		Name: "option_some",
		Parameters: []mir.Value{
			{Kind: mir.ValRef, Ref: "value", Class: mir.ClassInt},
		},
		Blocks: []*mir.BasicBlock{
			{
				Name: "entry",
				Instr: []mir.Instr{
					&mir.Alloca{Dst: "option.addr", Name: "option"},
					&mir.Ret{Val: nil},
				},
			},
		},
	}

	return []*mir.Function{someFunc}
}

// GenerateResultOperations creates MIR functions for Result type operations.
func (ctm *CoreTypeMIRIntegration) GenerateResultOperations() []*mir.Function {
	okFunc := &mir.Function{
		Name: "result_ok",
		Parameters: []mir.Value{
			{Kind: mir.ValRef, Ref: "value", Class: mir.ClassInt},
		},
		Blocks: []*mir.BasicBlock{
			{
				Name: "entry",
				Instr: []mir.Instr{
					&mir.Alloca{Dst: "result.addr", Name: "result"},
					&mir.Ret{Val: nil},
				},
			},
		},
	}

	return []*mir.Function{okFunc}
}

// GetAllCoreFunctions returns all core type MIR functions.
func (ctm *CoreTypeMIRIntegration) GetAllCoreFunctions() []*mir.Function {
	var functions []*mir.Function

	functions = append(functions, ctm.GenerateOptionOperations()...)
	functions = append(functions, ctm.GenerateResultOperations()...)

	return functions
}

// RegisterCoreFunctions registers all core type functions with the MIR module.
func (ctm *CoreTypeMIRIntegration) RegisterCoreFunctions() {
	functions := ctm.GetAllCoreFunctions()
	ctm.module.Functions = append(ctm.module.Functions, functions...)
}
