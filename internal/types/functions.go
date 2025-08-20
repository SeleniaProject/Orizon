// Function type system implementation for Orizon language.
// This module provides advanced function types, closures, and higher-order function support.

package types

import (
	"fmt"
	"strings"
)

// ====== Advanced Function Types ======.

// ClosureType represents a closure type with captured variables.
type ClosureType struct {
	BaseFunction *FunctionType
	Environment  *ClosureEnvironment
	CapturedVars []CapturedVariable
	CaptureMode  CaptureMode
}

// CapturedVariable represents a variable captured by a closure.
type CapturedVariable struct {
	Type        *Type
	Source      *Type
	Name        string
	CaptureKind CaptureKind
}

// CaptureKind represents how a variable is captured.
type CaptureKind int

const (
	CaptureByValue CaptureKind = iota
	CaptureByReference
	CaptureByMove
)

// CaptureMode represents the overall capture strategy.
type CaptureMode int

const (
	CaptureModeExplicit CaptureMode = iota
	CaptureModeImplicit
	CaptureModeMove
)

// ClosureEnvironment represents the runtime environment of a closure.
type ClosureEnvironment struct {
	Variables map[string]*Type
	Parent    *ClosureEnvironment
	Size      int // Runtime size of environment
}

// HigherOrderType represents higher-order function types.
type HigherOrderType struct {
	BaseFunction *FunctionType
	TypeParams   []GenericParameter
	Constraints  []*Type
}

// PartiallyAppliedType represents a partially applied function.
type PartiallyAppliedType struct {
	OriginalFunction *FunctionType
	ResultType       *Type
	AppliedArgs      []*Type
	RemainingParams  []*Type
}

// AsyncFunctionType represents asynchronous function types.
type AsyncFunctionType struct {
	BaseFunction *FunctionType
	AwaitType    *Type // Type that can be awaited
	ErrorType    *Type // Possible error type
}

// GeneratorType represents generator function types.
type GeneratorType struct {
	YieldType  *Type
	ReturnType *Type
	SendType   *Type // Type that can be sent to generator
}

// ====== Function Type Construction ======.

// NewClosureType creates a new closure type.
func NewClosureType(baseFunc *FunctionType, capturedVars []CapturedVariable, mode CaptureMode) *Type {
	// Calculate closure size (function pointer + environment).
	envSize := 8 // Base environment pointer
	for _, captured := range capturedVars {
		envSize += captured.Type.Size
	}

	return &Type{
		Kind: TypeKindFunction,
		Size: 16, // Function pointer + environment pointer
		Data: &ClosureType{
			BaseFunction: baseFunc,
			CapturedVars: capturedVars,
			CaptureMode:  mode,
			Environment: &ClosureEnvironment{
				Variables: make(map[string]*Type),
				Parent:    nil,
				Size:      envSize,
			},
		},
	}
}

// NewHigherOrderType creates a new higher-order function type.
func NewHigherOrderType(baseFunc *FunctionType, typeParams []GenericParameter) *Type {
	return &Type{
		Kind: TypeKindFunction,
		Size: 8, // Function pointer
		Data: &HigherOrderType{
			BaseFunction: baseFunc,
			TypeParams:   typeParams,
			Constraints:  []*Type{},
		},
	}
}

// NewPartiallyAppliedType creates a new partially applied function type.
func NewPartiallyAppliedType(originalFunc *FunctionType, appliedArgs []*Type) *Type {
	if len(appliedArgs) >= len(originalFunc.Parameters) {
		// Fully applied.
		return originalFunc.ReturnType
	}

	remainingParams := originalFunc.Parameters[len(appliedArgs):]

	return &Type{
		Kind: TypeKindFunction,
		Size: 16, // Function pointer + captured arguments
		Data: &PartiallyAppliedType{
			OriginalFunction: originalFunc,
			AppliedArgs:      appliedArgs,
			RemainingParams:  remainingParams,
			ResultType:       NewFunctionType(remainingParams, originalFunc.ReturnType, false, false),
		},
	}
}

// NewAsyncFunctionType creates a new async function type.
func NewAsyncFunctionType(baseFunc *FunctionType, awaitType *Type, errorType *Type) *Type {
	return &Type{
		Kind: TypeKindFunction,
		Size: 8,
		Data: &AsyncFunctionType{
			BaseFunction: baseFunc,
			AwaitType:    awaitType,
			ErrorType:    errorType,
		},
	}
}

// NewGeneratorType creates a new generator type.
func NewGeneratorType(yieldType, returnType, sendType *Type) *Type {
	return &Type{
		Kind: TypeKindFunction,
		Size: 16, // Generator state + environment
		Data: &GeneratorType{
			YieldType:  yieldType,
			ReturnType: returnType,
			SendType:   sendType,
		},
	}
}

// ====== Function Type Operations ======.

// IsCallableWith checks if a function can be called with given argument types.
func (t *Type) IsCallableWith(argTypes []*Type) bool {
	if !t.IsCallable() {
		return false
	}

	switch data := t.Data.(type) {
	case *FunctionType:
		return t.checkFunctionCallability(data, argTypes)

	case *ClosureType:
		return t.checkFunctionCallability(data.BaseFunction, argTypes)

	case *HigherOrderType:
		return t.checkHigherOrderCallability(data, argTypes)

	case *PartiallyAppliedType:
		return t.checkPartiallyAppliedCallability(data, argTypes)

	case *AsyncFunctionType:
		return t.checkFunctionCallability(data.BaseFunction, argTypes)

	case *GeneratorType:
		// Generators are not directly callable with arguments.
		return false

	default:
		return false
	}
}

// checkFunctionCallability checks if a function type can be called with given arguments.
func (t *Type) checkFunctionCallability(funcType *FunctionType, argTypes []*Type) bool {
	if funcType.IsVariadic {
		// Variadic function: at least N-1 args, where N is param count.
		if len(argTypes) < len(funcType.Parameters)-1 {
			return false
		}
		// For variadic, last parameter can accept multiple arguments.
		requiredParams := len(funcType.Parameters) - 1
		if len(argTypes) < requiredParams {
			return false
		}
	} else {
		// Regular function: exact parameter count.
		if len(argTypes) != len(funcType.Parameters) {
			return false
		}
	}

	// Check parameter types.
	for i, paramType := range funcType.Parameters {
		if i >= len(argTypes) {
			if !funcType.IsVariadic || i < len(funcType.Parameters)-1 {
				return false
			}

			break // Variadic parameter can be omitted
		}

		argType := argTypes[i]

		// For variadic parameter (last parameter), check if it's compatible.
		if funcType.IsVariadic && i == len(funcType.Parameters)-1 {
			// Variadic parameter accepts any number of arguments of compatible type.
			for j := i; j < len(argTypes); j++ {
				if !argTypes[j].IsAssignableFrom(paramType) && !argTypes[j].CanConvertTo(paramType) {
					return false
				}
			}

			break
		}

		// Regular parameter type check.
		if !argType.IsAssignableFrom(paramType) && !argType.CanConvertTo(paramType) {
			return false
		}
	}

	return true
}

// checkHigherOrderCallability checks higher-order function callability.
func (t *Type) checkHigherOrderCallability(hoType *HigherOrderType, argTypes []*Type) bool {
	// For higher-order functions, we need to check if type parameters can be inferred.
	// This is a simplified check - full implementation would involve constraint solving.
	return t.checkFunctionCallability(hoType.BaseFunction, argTypes)
}

// checkPartiallyAppliedCallability checks partially applied function callability.
func (t *Type) checkPartiallyAppliedCallability(paType *PartiallyAppliedType, argTypes []*Type) bool {
	return len(argTypes) <= len(paType.RemainingParams)
}

// ApplyPartially applies arguments to a function type, returning a new function type.
func (t *Type) ApplyPartially(argTypes []*Type) (*Type, error) {
	if !t.IsCallable() {
		return nil, fmt.Errorf("type %s is not callable", t.String())
	}

	switch data := t.Data.(type) {
	case *FunctionType:
		if len(argTypes) >= len(data.Parameters) {
			// Fully applied.
			return data.ReturnType, nil
		}

		return NewPartiallyAppliedType(data, argTypes), nil

	case *ClosureType:
		if len(argTypes) >= len(data.BaseFunction.Parameters) {
			// Fully applied.
			return data.BaseFunction.ReturnType, nil
		}

		return NewPartiallyAppliedType(data.BaseFunction, argTypes), nil

	case *PartiallyAppliedType:
		totalApplied := append(data.AppliedArgs, argTypes...)

		return NewPartiallyAppliedType(data.OriginalFunction, totalApplied), nil

	default:
		return nil, fmt.Errorf("partial application not supported for type %s", t.String())
	}
} // GetCallResultType returns the result type of calling this function with given arguments
func (t *Type) GetCallResultType(argTypes []*Type) (*Type, error) {
	if !t.IsCallableWith(argTypes) {
		return nil, fmt.Errorf("function not callable with given arguments")
	}

	switch data := t.Data.(type) {
	case *FunctionType:
		return data.ReturnType, nil

	case *ClosureType:
		return data.BaseFunction.ReturnType, nil

	case *PartiallyAppliedType:
		if len(argTypes) == len(data.RemainingParams) {
			return data.OriginalFunction.ReturnType, nil
		}
		// Return partially applied function.
		return t.ApplyPartially(argTypes)

	case *AsyncFunctionType:
		// Async functions return Promise/Future-like types
		return data.AwaitType, nil

	default:
		return nil, fmt.Errorf("unsupported callable type")
	}
}

// ====== Closure Analysis ======.

// AnalyzeClosure analyzes a closure and determines captured variables.
func AnalyzeClosure(funcType *FunctionType, environment *ClosureEnvironment,
	freeVars []string,
) (*Type, error) {
	var capturedVars []CapturedVariable

	for _, varName := range freeVars {
		// Look for variable in current environment or parent environments.
		var varType *Type

		currentEnv := environment
		for currentEnv != nil {
			if vType, exists := currentEnv.Variables[varName]; exists {
				varType = vType

				break
			}

			currentEnv = currentEnv.Parent
		}

		if varType != nil {
			// Determine capture kind based on variable usage.
			captureKind := CaptureByReference // Default

			// In a real implementation, you'd analyze usage patterns.
			// For now, we use simple heuristics.
			if varType.IsAggregate() {
				captureKind = CaptureByReference
			} else if varType.IsNumeric() || varType.Kind == TypeKindString || varType.Kind == TypeKindBool {
				captureKind = CaptureByValue
			}

			capturedVars = append(capturedVars, CapturedVariable{
				Name:        varName,
				Type:        varType,
				CaptureKind: captureKind,
				Source:      varType,
			})
		}
	}

	return NewClosureType(funcType, capturedVars, CaptureModeImplicit), nil
} // ====== Function Type Inference ======

// InferFunctionType infers the type of a function from usage context.
func InferFunctionType(paramTypes []*Type, returnType *Type, context *InferenceContext) *Type {
	// Basic function type inference.
	if returnType == nil {
		returnType = TypeVoid
	}

	funcType := NewFunctionType(paramTypes, returnType, false, false)

	// Check if this should be a higher-order function.
	hasGenericParams := false

	for _, paramType := range paramTypes {
		if paramType.Kind == TypeKindTypeVar || paramType.Kind == TypeKindGeneric {
			hasGenericParams = true

			break
		}
	}

	if hasGenericParams {
		// Extract generic parameters (avoid duplicates).
		var typeParams []GenericParameter

		seen := make(map[string]bool)

		for _, paramType := range paramTypes {
			if paramType.Kind == TypeKindGeneric {
				genericData := paramType.Data.(*GenericType)
				if !seen[genericData.Name] {
					typeParams = append(typeParams, GenericParameter{
						Name:        genericData.Name,
						Constraints: genericData.Constraints,
						Variance:    genericData.Variance,
					})
					seen[genericData.Name] = true
				}
			}
		}

		// Also check return type for generics.
		if returnType.Kind == TypeKindGeneric {
			genericData := returnType.Data.(*GenericType)
			if !seen[genericData.Name] {
				typeParams = append(typeParams, GenericParameter{
					Name:        genericData.Name,
					Constraints: genericData.Constraints,
					Variance:    genericData.Variance,
				})
				seen[genericData.Name] = true
			}
		}

		funcData := funcType.Data.(*FunctionType)

		return NewHigherOrderType(funcData, typeParams)
	}

	return funcType
}

// InferenceContext provides context for type inference.
type InferenceContext struct {
	Variables   map[string]*Type
	Functions   map[string]*Type
	TypeVars    map[string]*Type
	Constraints []Constraint
}

// Constraint represents a type constraint during inference.
type Constraint struct {
	Left  *Type
	Right *Type
	Kind  ConstraintKind
}

// ConstraintKind represents different kinds of type constraints.
type ConstraintKind int

const (
	ConstraintEqual ConstraintKind = iota
	ConstraintSubtype
	ConstraintSupertype
	ConstraintUnify
)

// ====== Function Composition ======.

// ComposeFunction composes two function types f(g(x)).
func ComposeFunction(f, g *Type) (*Type, error) {
	if !f.IsCallable() || !g.IsCallable() {
		return nil, fmt.Errorf("both types must be callable for composition")
	}

	fFunc := getFunctionTypeData(f)
	gFunc := getFunctionTypeData(g)

	if fFunc == nil || gFunc == nil {
		return nil, fmt.Errorf("invalid function types for composition")
	}

	// Check compatibility: f's input must match g's output.
	if len(fFunc.Parameters) != 1 {
		return nil, fmt.Errorf("first function must take exactly one parameter")
	}

	if !gFunc.ReturnType.IsAssignableFrom(fFunc.Parameters[0]) {
		return nil, fmt.Errorf("function output/input types not compatible")
	}

	// Create composed function type: g's params -> f's return.
	return NewFunctionType(gFunc.Parameters, fFunc.ReturnType, false, false), nil
}

// getFunctionTypeData extracts FunctionType data from various function-like types.
func getFunctionTypeData(t *Type) *FunctionType {
	switch data := t.Data.(type) {
	case *FunctionType:
		return data
	case *ClosureType:
		return data.BaseFunction
	case *HigherOrderType:
		return data.BaseFunction
	case *PartiallyAppliedType:
		return data.ResultType.Data.(*FunctionType)
	default:
		return nil
	}
}

// ====== Function Type Utilities ======.

// GetArity returns the arity (number of parameters) of a function type.
func (t *Type) GetArity() int {
	if !t.IsCallable() {
		return -1
	}

	switch data := t.Data.(type) {
	case *FunctionType:
		return len(data.Parameters)
	case *ClosureType:
		return len(data.BaseFunction.Parameters)
	case *PartiallyAppliedType:
		return len(data.RemainingParams)
	default:
		return -1
	}
}

// IsPure checks if a function type represents a pure function.
func (t *Type) IsPure() bool {
	// In a full implementation, this would check for effect annotations.
	// For now, we use simple heuristics.
	if !t.IsCallable() {
		return false
	}

	switch data := t.Data.(type) {
	case *FunctionType:
		// Pure if no side effects (would need effect system).
		return true // Simplified
	case *ClosureType:
		// Closures with mutable captures are not pure.
		for _, captured := range data.CapturedVars {
			if captured.CaptureKind == CaptureByReference {
				return false
			}
		}

		return true
	case *AsyncFunctionType:
		// Async functions are generally not pure.
		return false
	default:
		return false
	}
}

// IsRecursive checks if a function type might be recursive.
func (t *Type) IsRecursive() bool {
	// This is a placeholder - real implementation would need call graph analysis.
	return false
}

// ====== Higher-Order Function Utilities ======.

// CreateMap creates a map function type: (T -> U) -> [T] -> [U].
func CreateMapType() *Type {
	// Create generic type parameters.
	tParam := NewGenericType("T", []*Type{}, VarianceInvariant)
	uParam := NewGenericType("U", []*Type{}, VarianceInvariant)

	// Create function type T -> U.
	mapperFunc := NewFunctionType([]*Type{tParam}, uParam, false, false)

	// Create array types.
	inputArray := NewSliceType(tParam)
	outputArray := NewSliceType(uParam)

	// Create map function: (T -> U) -> [T] -> [U].
	mapType := NewFunctionType([]*Type{mapperFunc, inputArray}, outputArray, false, false)

	return NewHigherOrderType(mapType.Data.(*FunctionType), []GenericParameter{
		{Name: "T", Constraints: []*Type{}, Variance: VarianceInvariant},
		{Name: "U", Constraints: []*Type{}, Variance: VarianceInvariant},
	})
}

// CreateFilter creates a filter function type: (T -> bool) -> [T] -> [T].
func CreateFilterType() *Type {
	tParam := NewGenericType("T", []*Type{}, VarianceInvariant)

	predicateFunc := NewFunctionType([]*Type{tParam}, TypeBool, false, false)
	arrayType := NewSliceType(tParam)

	filterType := NewFunctionType([]*Type{predicateFunc, arrayType}, arrayType, false, false)

	return NewHigherOrderType(filterType.Data.(*FunctionType), []GenericParameter{
		{Name: "T", Constraints: []*Type{}, Variance: VarianceInvariant},
	})
}

// CreateReduce creates a reduce function type: (acc -> T -> acc) -> acc -> [T] -> acc.
func CreateReduceType() *Type {
	tParam := NewGenericType("T", []*Type{}, VarianceInvariant)
	accParam := NewGenericType("Acc", []*Type{}, VarianceInvariant)

	reducerFunc := NewFunctionType([]*Type{accParam, tParam}, accParam, false, false)
	arrayType := NewSliceType(tParam)

	reduceType := NewFunctionType([]*Type{reducerFunc, accParam, arrayType}, accParam, false, false)

	return NewHigherOrderType(reduceType.Data.(*FunctionType), []GenericParameter{
		{Name: "T", Constraints: []*Type{}, Variance: VarianceInvariant},
		{Name: "Acc", Constraints: []*Type{}, Variance: VarianceInvariant},
	})
}

// ====== Function Type String Representation ======.

// Enhanced string representation for function types.
func (t *Type) FunctionString() string {
	if !t.IsCallable() {
		return t.String()
	}

	switch data := t.Data.(type) {
	case *FunctionType:
		return t.functionTypeString(data)

	case *ClosureType:
		captured := []string{}

		for _, cap := range data.CapturedVars {
			captureSymbol := ""

			switch cap.CaptureKind {
			case CaptureByValue:
				captureSymbol = "="
			case CaptureByReference:
				captureSymbol = "&"
			case CaptureByMove:
				captureSymbol = "move"
			}

			captured = append(captured, fmt.Sprintf("%s%s: %s", captureSymbol, cap.Name, cap.Type.String()))
		}

		return fmt.Sprintf("closure[%s] %s", strings.Join(captured, ", "),
			t.functionTypeString(data.BaseFunction))

	case *HigherOrderType:
		var typeParams []string
		for _, param := range data.TypeParams {
			typeParams = append(typeParams, param.Name)
		}

		return fmt.Sprintf("<%s> %s", strings.Join(typeParams, ", "),
			t.functionTypeString(data.BaseFunction))

	case *PartiallyAppliedType:
		var appliedTypes []string
		for _, arg := range data.AppliedArgs {
			appliedTypes = append(appliedTypes, arg.String())
		}

		return fmt.Sprintf("partial(%s) %s", strings.Join(appliedTypes, ", "),
			data.ResultType.String())

	case *AsyncFunctionType:
		return fmt.Sprintf("async %s -> Future<%s>",
			t.functionTypeString(data.BaseFunction),
			data.AwaitType.String())

	case *GeneratorType:
		return fmt.Sprintf("generator<yield: %s, return: %s, send: %s>",
			data.YieldType.String(), data.ReturnType.String(), data.SendType.String())

	default:
		return t.String()
	}
}

// functionTypeString creates string representation for basic function types.
func (t *Type) functionTypeString(funcType *FunctionType) string {
	var params []string
	for _, param := range funcType.Parameters {
		params = append(params, param.String())
	}

	paramStr := strings.Join(params, ", ")
	if funcType.IsVariadic {
		paramStr += ", ..."
	}

	asyncStr := ""
	if funcType.IsAsync {
		asyncStr = "async "
	}

	return fmt.Sprintf("%sfn(%s) -> %s", asyncStr, paramStr, funcType.ReturnType.String())
}

// ====== Function Type Registry ======.

// FunctionTypeRegistry manages common function types.
type FunctionTypeRegistry struct {
	CommonTypes map[string]*Type
}

// NewFunctionTypeRegistry creates a new function type registry.
func NewFunctionTypeRegistry() *FunctionTypeRegistry {
	registry := &FunctionTypeRegistry{
		CommonTypes: make(map[string]*Type),
	}

	// Register common higher-order function types.
	registry.CommonTypes["map"] = CreateMapType()
	registry.CommonTypes["filter"] = CreateFilterType()
	registry.CommonTypes["reduce"] = CreateReduceType()

	// Register common function signatures.
	registry.CommonTypes["unary_int"] = NewFunctionType([]*Type{TypeInt32}, TypeInt32, false, false)
	registry.CommonTypes["binary_int"] = NewFunctionType([]*Type{TypeInt32, TypeInt32}, TypeInt32, false, false)
	registry.CommonTypes["predicate"] = NewFunctionType([]*Type{TypeAny}, TypeBool, false, false)

	return registry
}

// GetCommonType retrieves a common function type by name.
func (ftr *FunctionTypeRegistry) GetCommonType(name string) (*Type, bool) {
	typeObj, exists := ftr.CommonTypes[name]

	return typeObj, exists
}

// RegisterCommonType registers a new common function type.
func (ftr *FunctionTypeRegistry) RegisterCommonType(name string, funcType *Type) {
	if funcType.IsCallable() {
		ftr.CommonTypes[name] = funcType
	}
}
