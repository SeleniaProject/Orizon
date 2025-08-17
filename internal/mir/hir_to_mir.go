// HIR to MIR conversion for Orizon language
// This file implements the transformation from High-level Intermediate Representation (HIR)
// to Mid-level Intermediate Representation (MIR). The transformation:
// 1. Converts HIR control flow to basic blocks with SSA-like form
// 2. Lowers high-level constructs to basic MIR operations
// 3. Performs basic optimizations (constant propagation, dead code elimination)
// 4. Prepares for subsequent code generation

package mir

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/orizon-lang/orizon/internal/parser"
)

// ====== HIR to MIR Transformer ======

// HIRToMIRTransformer converts HIR nodes to MIR nodes
type HIRToMIRTransformer struct {
	currentModule   *Module
	currentFunction *Function
	currentBlock    *BasicBlock
	valueCounter    int
	blockCounter    int
	symbolTable     map[string]string // HIR symbol -> MIR value mapping
	tempValues      map[string]Value  // temporary value storage
	optimizations   OptimizationFlags
	errors          []error

	// Memory safety components
	lifetimeManager  *LifetimeManager
	borrowChecker    *BorrowChecker
	ownershipManager *OwnershipManager
}

// OptimizationFlags controls which optimizations to apply
type OptimizationFlags struct {
	ConstantPropagation bool
	DeadCodeElimination bool
	BasicBlockMerging   bool
	CommonSubexpression bool
}

// DefaultOptimizations returns standard optimization settings
func DefaultOptimizations() OptimizationFlags {
	return OptimizationFlags{
		ConstantPropagation: true,
		DeadCodeElimination: true,
		BasicBlockMerging:   true,
		CommonSubexpression: false, // More complex, disabled for now
	}
}

// NewHIRToMIRTransformer creates a new HIR to MIR transformer
func NewHIRToMIRTransformer() *HIRToMIRTransformer {
	lm := NewLifetimeManager()
	bc := NewBorrowChecker(lm)
	om := NewOwnershipManager(bc)

	return &HIRToMIRTransformer{
		symbolTable:      make(map[string]string),
		tempValues:       make(map[string]Value),
		optimizations:    DefaultOptimizations(),
		errors:           make([]error, 0),
		lifetimeManager:  lm,
		borrowChecker:    bc,
		ownershipManager: om,
	}
}

// GetErrors returns transformation errors
func (t *HIRToMIRTransformer) GetErrors() []error {
	return t.errors
}

// GetLifetimeManager returns the lifetime manager
func (t *HIRToMIRTransformer) GetLifetimeManager() *LifetimeManager {
	return t.lifetimeManager
}

// GetBorrowChecker returns the borrow checker
func (t *HIRToMIRTransformer) GetBorrowChecker() *BorrowChecker {
	return t.borrowChecker
}

// GetOwnershipManager returns the ownership manager
func (t *HIRToMIRTransformer) GetOwnershipManager() *OwnershipManager {
	return t.ownershipManager
}

// ====== Module Transformation ======

// TransformModule converts a HIR module to MIR module
func (t *HIRToMIRTransformer) TransformModule(hirModule *parser.HIRModule) (*Module, error) {
	if hirModule == nil {
		return nil, fmt.Errorf("HIR module is nil")
	}

	t.currentModule = &Module{
		Name:      hirModule.Name,
		Functions: make([]*Function, 0),
	}

	// Transform all functions
	for _, hirFunc := range hirModule.Functions {
		mirFunc, err := t.transformFunction(hirFunc)
		if err != nil {
			t.errors = append(t.errors, fmt.Errorf("function %s: %v", hirFunc.Name, err))
			// Continue processing even if this function failed
			continue
		}
		if mirFunc != nil {
			t.currentModule.Functions = append(t.currentModule.Functions, mirFunc)
		} else {
			t.errors = append(t.errors, fmt.Errorf("function %s: transformation returned nil", hirFunc.Name))
		}
	}

	// Apply module-level optimizations
	if len(t.errors) == 0 {
		t.optimizeModule()
	}

	// Perform memory safety validation
	if len(t.errors) == 0 {
		// Validate lifetimes
		if err := t.lifetimeManager.ValidateLifetimes(t.currentModule); err != nil {
			t.errors = append(t.errors, fmt.Errorf("lifetime validation failed: %v", err))
		}

		// Validate borrow rules
		if err := t.borrowChecker.ValidateBorrowRules(t.currentModule); err != nil {
			t.errors = append(t.errors, fmt.Errorf("borrow validation failed: %v", err))
		}

		// Validate ownership
		if err := t.ownershipManager.ValidateOwnership(t.currentModule); err != nil {
			t.errors = append(t.errors, fmt.Errorf("ownership validation failed: %v", err))
		}
	}

	// Return module even if there were errors, so we can see what was produced
	return t.currentModule, nil
}

// ====== Function Transformation ======

// transformFunction converts a HIR function to MIR function
func (t *HIRToMIRTransformer) transformFunction(hirFunc *parser.HIRFunction) (*Function, error) {
	if hirFunc == nil {
		return nil, fmt.Errorf("HIR function is nil")
	}

	// Reset transformer state for new function
	t.valueCounter = 0
	t.blockCounter = 0
	t.symbolTable = make(map[string]string)
	t.tempValues = make(map[string]Value)

	t.currentFunction = &Function{
		Name:       hirFunc.Name,
		Parameters: make([]Value, 0),
		Blocks:     make([]*BasicBlock, 0),
	}

	// Transform parameters
	for _, param := range hirFunc.Parameters {
		mirParam := t.transformParameter(param)
		t.currentFunction.Parameters = append(t.currentFunction.Parameters, mirParam)
		// Map parameter to its MIR representation
		t.symbolTable[param.Name] = fmt.Sprintf("%%param_%s", param.Name)
	}

	// Create entry basic block
	entryBlock := t.createBasicBlock("entry")
	t.currentFunction.Blocks = append(t.currentFunction.Blocks, entryBlock)
	t.currentBlock = entryBlock

	// Transform function body
	if hirFunc.Body != nil {
		err := t.transformBlock(hirFunc.Body)
		if err != nil {
			return nil, fmt.Errorf("body transformation failed: %v", err)
		}
	} else {
		// If no body, add a simple return
		t.currentBlock.Instr = append(t.currentBlock.Instr, Ret{Val: nil})
	}

	// Ensure function ends with return
	t.ensureFunctionReturn()

	// Apply function-level optimizations
	t.optimizeFunction()

	return t.currentFunction, nil
}

// transformParameter converts HIR parameter to MIR value
func (t *HIRToMIRTransformer) transformParameter(hirParam *parser.HIRParameter) Value {
	return Value{
		Kind:  ValRef,
		Ref:   fmt.Sprintf("%%param_%s", hirParam.Name),
		Class: t.getValueClass(hirParam.Type),
	}
}

// ====== Block Transformation ======

// transformBlock converts HIR block to MIR instructions
func (t *HIRToMIRTransformer) transformBlock(hirBlock *parser.HIRBlock) error {
	if hirBlock == nil {
		return nil
	}

	// Transform all statements in the block
	for _, stmt := range hirBlock.Statements {
		err := t.transformStatement(stmt)
		if err != nil {
			return err
		}
	}

	// Transform trailing expression if present
	if hirBlock.Expression != nil {
		_, err := t.transformExpression(hirBlock.Expression)
		return err
	}

	return nil
}

// ====== Statement Transformation ======

// transformStatement converts HIR statement to MIR instructions
func (t *HIRToMIRTransformer) transformStatement(hirStmt *parser.HIRStatement) error {
	if hirStmt == nil {
		return nil
	}

	switch hirStmt.Kind {
	case parser.HIRStmtExpression:
		if exprData, ok := hirStmt.Data.(*parser.HIRExpression); ok {
			_, err := t.transformExpression(exprData)
			return err
		}
		return fmt.Errorf("invalid expression statement data")

	case parser.HIRStmtLet:
		if letData, ok := hirStmt.Data.(*parser.HIRLetStatement); ok {
			return t.transformLetStatement(letData)
		}
		return fmt.Errorf("invalid let statement data")

	case parser.HIRStmtAssign:
		if assignData, ok := hirStmt.Data.(*parser.HIRAssignStatement); ok {
			return t.transformAssignStatement(assignData)
		}
		return fmt.Errorf("invalid assign statement data")

	case parser.HIRStmtReturn:
		if retData, ok := hirStmt.Data.(*parser.HIRReturnStatement); ok {
			return t.transformReturnStatement(retData)
		}
		return fmt.Errorf("invalid return statement data")

	case parser.HIRStmtIf:
		if ifData, ok := hirStmt.Data.(*parser.HIRIfStatement); ok {
			return t.transformIfStatement(ifData)
		}
		return fmt.Errorf("invalid if statement data")

	case parser.HIRStmtWhile:
		if whileData, ok := hirStmt.Data.(*parser.HIRWhileStatement); ok {
			return t.transformWhileStatement(whileData)
		}
		return fmt.Errorf("invalid while statement data")

	case parser.HIRStmtFor:
		if forData, ok := hirStmt.Data.(*parser.HIRForStatement); ok {
			return t.transformForStatement(forData)
		}
		return fmt.Errorf("invalid for statement data")

	default:
		return fmt.Errorf("unsupported HIR statement kind: %v", hirStmt.Kind)
	}
}

// transformLetStatement converts HIR let statement to MIR
func (t *HIRToMIRTransformer) transformLetStatement(stmt *parser.HIRLetStatement) error {
	if stmt.Variable == nil {
		return fmt.Errorf("let statement missing variable")
	}

	// Allocate stack space for the variable
	allocaName := t.getNextValue()
	alloca := Alloca{
		Dst:  allocaName,
		Name: stmt.Variable.Name,
	}
	t.currentBlock.Instr = append(t.currentBlock.Instr, alloca)

	// Map variable name to its stack address
	t.symbolTable[stmt.Variable.Name] = allocaName

	// If there's an initializer, store the value
	if stmt.Initializer != nil {
		initVal, err := t.transformExpression(stmt.Initializer)
		if err != nil {
			return err
		}

		store := Store{
			Addr: Value{Kind: ValRef, Ref: allocaName, Class: ClassInt},
			Val:  initVal,
		}
		t.currentBlock.Instr = append(t.currentBlock.Instr, store)
	}

	return nil
}

// transformAssignStatement converts HIR assign statement to MIR
func (t *HIRToMIRTransformer) transformAssignStatement(stmt *parser.HIRAssignStatement) error {
	// For simple assignment, get the target address
	if stmt.Target.Kind == parser.HIRExprVariable {
		if varData, ok := stmt.Target.Data.(*parser.HIRVariableExpression); ok {
			targetAddr, exists := t.symbolTable[varData.Name]
			if !exists {
				return fmt.Errorf("undefined variable: %s", varData.Name)
			}

			// Transform the value expression
			val, err := t.transformExpression(stmt.Value)
			if err != nil {
				return err
			}

			// Generate store instruction
			store := Store{
				Addr: Value{Kind: ValRef, Ref: targetAddr, Class: ClassInt},
				Val:  val,
			}
			t.currentBlock.Instr = append(t.currentBlock.Instr, store)
			return nil
		}
	}

	return fmt.Errorf("unsupported assignment target")
}

// transformReturnStatement converts HIR return statement to MIR
func (t *HIRToMIRTransformer) transformReturnStatement(stmt *parser.HIRReturnStatement) error {
	var retVal *Value
	if stmt.Value != nil {
		val, err := t.transformExpression(stmt.Value)
		if err != nil {
			return err
		}
		retVal = &val
	}

	t.currentBlock.Instr = append(t.currentBlock.Instr, Ret{Val: retVal})
	return nil
}

// transformIfStatement converts HIR if statement to MIR control flow
func (t *HIRToMIRTransformer) transformIfStatement(stmt *parser.HIRIfStatement) error {
	// Transform condition
	condVal, err := t.transformExpression(stmt.Condition)
	if err != nil {
		return err
	}

	// Create basic blocks for then, else, and continuation
	thenBlock := t.createBasicBlock("if_then")
	var elseBlock *BasicBlock
	if stmt.ElseBlock != nil {
		elseBlock = t.createBasicBlock("if_else")
	}
	contBlock := t.createBasicBlock("if_cont")

	// Generate conditional branch
	var falseTarget string
	if elseBlock != nil {
		falseTarget = elseBlock.Name
	} else {
		falseTarget = contBlock.Name
	}

	condBr := CondBr{
		Cond:  condVal,
		True:  thenBlock.Name,
		False: falseTarget,
	}
	t.currentBlock.Instr = append(t.currentBlock.Instr, condBr)

	// Add blocks to function
	t.currentFunction.Blocks = append(t.currentFunction.Blocks, thenBlock)
	if elseBlock != nil {
		t.currentFunction.Blocks = append(t.currentFunction.Blocks, elseBlock)
	}
	t.currentFunction.Blocks = append(t.currentFunction.Blocks, contBlock)

	// Transform then block
	t.currentBlock = thenBlock
	err = t.transformBlock(stmt.ThenBlock)
	if err != nil {
		return err
	}
	// Branch to continuation
	t.currentBlock.Instr = append(t.currentBlock.Instr, Br{Target: contBlock.Name})

	// Transform else block if present
	if stmt.ElseBlock != nil && elseBlock != nil {
		t.currentBlock = elseBlock
		err = t.transformBlock(stmt.ElseBlock)
		if err != nil {
			return err
		}
		// Branch to continuation
		t.currentBlock.Instr = append(t.currentBlock.Instr, Br{Target: contBlock.Name})
	}

	// Continue with continuation block
	t.currentBlock = contBlock
	return nil
}

// transformWhileStatement converts HIR while statement to MIR loop
func (t *HIRToMIRTransformer) transformWhileStatement(stmt *parser.HIRWhileStatement) error {
	// Create basic blocks for loop header, body, and exit
	headerBlock := t.createBasicBlock("while_header")
	bodyBlock := t.createBasicBlock("while_body")
	exitBlock := t.createBasicBlock("while_exit")

	// Branch to header
	t.currentBlock.Instr = append(t.currentBlock.Instr, Br{Target: headerBlock.Name})

	// Add blocks to function
	t.currentFunction.Blocks = append(t.currentFunction.Blocks, headerBlock)
	t.currentFunction.Blocks = append(t.currentFunction.Blocks, bodyBlock)
	t.currentFunction.Blocks = append(t.currentFunction.Blocks, exitBlock)

	// Transform loop header (condition check)
	t.currentBlock = headerBlock
	condVal, err := t.transformExpression(stmt.Condition)
	if err != nil {
		return err
	}

	// Conditional branch based on condition
	condBr := CondBr{
		Cond:  condVal,
		True:  bodyBlock.Name,
		False: exitBlock.Name,
	}
	t.currentBlock.Instr = append(t.currentBlock.Instr, condBr)

	// Transform loop body
	t.currentBlock = bodyBlock
	err = t.transformBlock(stmt.Body)
	if err != nil {
		return err
	}
	// Branch back to header
	t.currentBlock.Instr = append(t.currentBlock.Instr, Br{Target: headerBlock.Name})

	// Continue with exit block
	t.currentBlock = exitBlock
	return nil
}

// transformForStatement converts HIR for statement to MIR loop
func (t *HIRToMIRTransformer) transformForStatement(stmt *parser.HIRForStatement) error {
	// For now, implement a simplified for loop transformation
	// A complete implementation would handle pattern matching and iterators

	// Create basic blocks
	headerBlock := t.createBasicBlock("for_header")
	bodyBlock := t.createBasicBlock("for_body")
	exitBlock := t.createBasicBlock("for_exit")

	// Branch to header
	t.currentBlock.Instr = append(t.currentBlock.Instr, Br{Target: headerBlock.Name})

	// Add blocks to function
	t.currentFunction.Blocks = append(t.currentFunction.Blocks, headerBlock)
	t.currentFunction.Blocks = append(t.currentFunction.Blocks, bodyBlock)
	t.currentFunction.Blocks = append(t.currentFunction.Blocks, exitBlock)

	// Transform loop header
	t.currentBlock = headerBlock
	// For simplified implementation, always branch to body
	// A complete implementation would handle iterator protocol
	t.currentBlock.Instr = append(t.currentBlock.Instr, Br{Target: bodyBlock.Name})

	// Transform loop body
	t.currentBlock = bodyBlock
	err := t.transformBlock(stmt.Body)
	if err != nil {
		return err
	}
	// Branch back to header
	t.currentBlock.Instr = append(t.currentBlock.Instr, Br{Target: headerBlock.Name})

	// Continue with exit block
	t.currentBlock = exitBlock
	return nil
}

// ====== Expression Transformation ======

// transformExpression converts HIR expression to MIR value
func (t *HIRToMIRTransformer) transformExpression(hirExpr *parser.HIRExpression) (Value, error) {
	if hirExpr == nil {
		return Value{}, fmt.Errorf("HIR expression is nil")
	}

	switch hirExpr.Kind {
	case parser.HIRExprLiteral:
		if litData, ok := hirExpr.Data.(*parser.HIRLiteralExpression); ok {
			return t.transformLiteralExpression(litData)
		}
		return Value{}, fmt.Errorf("invalid literal expression data")

	case parser.HIRExprVariable:
		if varData, ok := hirExpr.Data.(*parser.HIRVariableExpression); ok {
			return t.transformVariableExpression(varData)
		}
		return Value{}, fmt.Errorf("invalid variable expression data")

	case parser.HIRExprBinary:
		if binData, ok := hirExpr.Data.(*parser.HIRBinaryExpression); ok {
			return t.transformBinaryExpression(binData)
		}
		return Value{}, fmt.Errorf("invalid binary expression data")

	case parser.HIRExprCall:
		if callData, ok := hirExpr.Data.(*parser.HIRCallExpression); ok {
			return t.transformCallExpression(callData)
		}
		return Value{}, fmt.Errorf("invalid call expression data")

	case parser.HIRExprFieldAccess:
		if fieldData, ok := hirExpr.Data.(*parser.HIRFieldAccessExpression); ok {
			return t.transformFieldAccessExpression(fieldData)
		}
		return Value{}, fmt.Errorf("invalid field access expression data")

	case parser.HIRExprIndex:
		if indexData, ok := hirExpr.Data.(*parser.HIRIndexExpression); ok {
			return t.transformIndexExpression(indexData)
		}
		return Value{}, fmt.Errorf("invalid index expression data")

	default:
		return Value{}, fmt.Errorf("unsupported HIR expression kind: %v", hirExpr.Kind)
	}
}

// transformLiteralExpression converts HIR literal to MIR value
func (t *HIRToMIRTransformer) transformLiteralExpression(expr *parser.HIRLiteralExpression) (Value, error) {
	switch expr.Kind {
	case parser.LiteralInteger:
		if strVal, ok := expr.Value.(string); ok {
			val, err := strconv.ParseInt(strVal, 10, 64)
			if err != nil {
				return Value{}, fmt.Errorf("invalid integer literal: %s", strVal)
			}
			return Value{Kind: ValConstInt, Int64: val, Class: ClassInt}, nil
		}
		if intVal, ok := expr.Value.(int64); ok {
			return Value{Kind: ValConstInt, Int64: intVal, Class: ClassInt}, nil
		}
		return Value{}, fmt.Errorf("invalid integer literal type")

	case parser.LiteralFloat:
		if strVal, ok := expr.Value.(string); ok {
			val, err := strconv.ParseFloat(strVal, 64)
			if err != nil {
				return Value{}, fmt.Errorf("invalid float literal: %s", strVal)
			}
			return Value{Kind: ValConstFloat, Float64: val, Class: ClassFloat}, nil
		}
		if floatVal, ok := expr.Value.(float64); ok {
			return Value{Kind: ValConstFloat, Float64: floatVal, Class: ClassFloat}, nil
		}
		return Value{}, fmt.Errorf("invalid float literal type")

	case parser.LiteralString:
		// For now, treat strings as pointer-like values
		// In a full implementation, this would involve string table management
		return Value{Kind: ValRef, Ref: fmt.Sprintf("str_%d", t.valueCounter), Class: ClassInt}, nil

	case parser.LiteralBool:
		var val int64
		if boolVal, ok := expr.Value.(bool); ok {
			if boolVal {
				val = 1
			}
		} else if strVal, ok := expr.Value.(string); ok {
			if strVal == "true" {
				val = 1
			}
		}
		return Value{Kind: ValConstInt, Int64: val, Class: ClassInt}, nil

	default:
		return Value{}, fmt.Errorf("unsupported literal kind: %v", expr.Kind)
	}
}

// transformVariableExpression converts HIR variable to MIR value
func (t *HIRToMIRTransformer) transformVariableExpression(expr *parser.HIRVariableExpression) (Value, error) {
	// Look up variable in symbol table
	addr, exists := t.symbolTable[expr.Name]
	if !exists {
		return Value{}, fmt.Errorf("undefined variable: %s", expr.Name)
	}

	// Generate load instruction
	loadDst := t.getNextValue()
	load := Load{
		Dst:  loadDst,
		Addr: Value{Kind: ValRef, Ref: addr, Class: ClassInt},
	}
	t.currentBlock.Instr = append(t.currentBlock.Instr, load)

	return Value{Kind: ValRef, Ref: loadDst, Class: ClassInt}, nil
}

// transformBinaryExpression converts HIR binary expression to MIR
func (t *HIRToMIRTransformer) transformBinaryExpression(expr *parser.HIRBinaryExpression) (Value, error) {
	// Transform operands
	lhs, err := t.transformExpression(expr.Left)
	if err != nil {
		return Value{}, err
	}

	rhs, err := t.transformExpression(expr.Right)
	if err != nil {
		return Value{}, err
	}

	// Generate appropriate MIR instruction based on operator
	switch expr.Operator {
	case parser.BinOpAdd:
		return t.generateBinaryOp(OpAdd, lhs, rhs)
	case parser.BinOpSub:
		return t.generateBinaryOp(OpSub, lhs, rhs)
	case parser.BinOpMul:
		return t.generateBinaryOp(OpMul, lhs, rhs)
	case parser.BinOpDiv:
		return t.generateBinaryOp(OpDiv, lhs, rhs)
	case parser.BinOpEq:
		return t.generateComparison(CmpEQ, lhs, rhs)
	case parser.BinOpNe:
		return t.generateComparison(CmpNE, lhs, rhs)
	case parser.BinOpLt:
		return t.generateComparison(CmpSLT, lhs, rhs)
	case parser.BinOpLe:
		return t.generateComparison(CmpSLE, lhs, rhs)
	case parser.BinOpGt:
		return t.generateComparison(CmpSGT, lhs, rhs)
	case parser.BinOpGe:
		return t.generateComparison(CmpSGE, lhs, rhs)
	default:
		return Value{}, fmt.Errorf("unsupported binary operator: %v", expr.Operator)
	}
}

// generateBinaryOp creates a binary operation instruction
func (t *HIRToMIRTransformer) generateBinaryOp(op BinOpKind, lhs, rhs Value) (Value, error) {
	dst := t.getNextValue()
	binOp := BinOp{
		Dst: dst,
		Op:  op,
		LHS: lhs,
		RHS: rhs,
	}
	t.currentBlock.Instr = append(t.currentBlock.Instr, binOp)

	return Value{Kind: ValRef, Ref: dst, Class: lhs.Class}, nil
}

// generateComparison creates a comparison instruction
func (t *HIRToMIRTransformer) generateComparison(pred CmpPred, lhs, rhs Value) (Value, error) {
	dst := t.getNextValue()
	cmp := Cmp{
		Dst:  dst,
		Pred: pred,
		LHS:  lhs,
		RHS:  rhs,
	}
	t.currentBlock.Instr = append(t.currentBlock.Instr, cmp)

	return Value{Kind: ValRef, Ref: dst, Class: ClassInt}, nil
}

// transformUnaryExpression converts HIR unary expression to MIR
func (t *HIRToMIRTransformer) transformUnaryExpression(expr *parser.HIRUnaryExpression) (Value, error) {
	operand, err := t.transformExpression(expr.Operand)
	if err != nil {
		return Value{}, err
	}

	switch expr.Operator {
	case parser.UnaryOpNeg:
		// Unary minus: 0 - operand
		zero := Value{Kind: ValConstInt, Int64: 0, Class: ClassInt}
		return t.generateBinaryOp(OpSub, zero, operand)
	case parser.UnaryOpNot:
		// Logical not: compare with 0
		zero := Value{Kind: ValConstInt, Int64: 0, Class: ClassInt}
		return t.generateComparison(CmpEQ, operand, zero)
	default:
		return Value{}, fmt.Errorf("unsupported unary operator: %v", expr.Operator)
	}
}

// transformCallExpression converts HIR call expression to MIR
func (t *HIRToMIRTransformer) transformCallExpression(expr *parser.HIRCallExpression) (Value, error) {
	// Transform arguments
	args := make([]Value, 0, len(expr.Arguments))
	for _, arg := range expr.Arguments {
		argVal, err := t.transformExpression(arg)
		if err != nil {
			return Value{}, err
		}
		args = append(args, argVal)
	}

	// Get function name (simplified - a complete implementation would handle function expressions)
	var funcName string
	if expr.Function.Kind == parser.HIRExprVariable {
		if varData, ok := expr.Function.Data.(*parser.HIRVariableExpression); ok {
			funcName = varData.Name
		}
	}

	// Generate call instruction
	dst := t.getNextValue()
	call := Call{
		Dst:    dst,
		Callee: funcName,
		Args:   args,
	}
	t.currentBlock.Instr = append(t.currentBlock.Instr, call)

	return Value{Kind: ValRef, Ref: dst, Class: ClassInt}, nil
}

// transformFieldAccessExpression converts HIR field access to MIR
func (t *HIRToMIRTransformer) transformFieldAccessExpression(expr *parser.HIRFieldAccessExpression) (Value, error) {
	// For now, this is a simplified implementation
	// A complete implementation would need struct layout information
	baseVal, err := t.transformExpression(expr.Object)
	if err != nil {
		return Value{}, err
	}

	// Generate field access as a load with offset (simplified)
	dst := t.getNextValue()
	load := Load{
		Dst:  dst,
		Addr: baseVal,
	}
	t.currentBlock.Instr = append(t.currentBlock.Instr, load)

	return Value{Kind: ValRef, Ref: dst, Class: ClassInt}, nil
}

// transformIndexExpression converts HIR index expression to MIR
func (t *HIRToMIRTransformer) transformIndexExpression(expr *parser.HIRIndexExpression) (Value, error) {
	// Transform object and index
	objectVal, err := t.transformExpression(expr.Object)
	if err != nil {
		return Value{}, err
	}

	indexVal, err := t.transformExpression(expr.Index)
	if err != nil {
		return Value{}, err
	}

	// For now, this is a simplified implementation
	// A complete implementation would calculate the actual address
	dst := t.getNextValue()
	load := Load{
		Dst:  dst,
		Addr: objectVal,
	}
	t.currentBlock.Instr = append(t.currentBlock.Instr, load)

	// Note: indexVal should be used to calculate offset, but simplified here
	_ = indexVal

	return Value{Kind: ValRef, Ref: dst, Class: ClassInt}, nil
}

// ====== Utility Methods ======

// createBasicBlock creates a new basic block with a unique name
func (t *HIRToMIRTransformer) createBasicBlock(prefix string) *BasicBlock {
	name := fmt.Sprintf("%s_%d", prefix, t.blockCounter)
	t.blockCounter++
	return &BasicBlock{
		Name:  name,
		Instr: make([]Instr, 0),
	}
}

// getNextValue generates a unique value name
func (t *HIRToMIRTransformer) getNextValue() string {
	name := fmt.Sprintf("%%v%d", t.valueCounter)
	t.valueCounter++
	return name
}

// getValueClass determines the value class from HIR type
func (t *HIRToMIRTransformer) getValueClass(hirType *parser.HIRType) ValueClass {
	if hirType == nil {
		return ClassUnknown
	}

	switch hirType.Kind {
	case parser.HIRTypePrimitive:
		if primData, ok := hirType.Data.(*parser.HIRPrimitiveType); ok {
			switch primData.Name {
			case "int", "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64":
				return ClassInt
			case "float", "f32", "f64":
				return ClassFloat
			case "bool":
				return ClassInt
			default:
				return ClassUnknown
			}
		}
	case parser.HIRTypePointer, parser.HIRTypeReference:
		return ClassInt
	default:
		return ClassUnknown
	}

	return ClassUnknown
}

// ensureFunctionReturn ensures function ends with a return instruction
func (t *HIRToMIRTransformer) ensureFunctionReturn() {
	if t.currentBlock == nil || len(t.currentBlock.Instr) == 0 {
		t.currentBlock.Instr = append(t.currentBlock.Instr, Ret{Val: nil})
		return
	}

	// Check if last instruction is already a terminator
	lastInstr := t.currentBlock.Instr[len(t.currentBlock.Instr)-1]
	switch lastInstr.(type) {
	case Ret, Br, CondBr:
		// Already has terminator
		return
	default:
		// Add return
		t.currentBlock.Instr = append(t.currentBlock.Instr, Ret{Val: nil})
	}
}

// ====== Optimizations ======

// optimizeModule applies module-level optimizations
func (t *HIRToMIRTransformer) optimizeModule() {
	if !t.optimizations.DeadCodeElimination {
		return
	}

	// Remove unused functions (simplified implementation)
	usedFunctions := make(map[string]bool)

	// Mark main function as used if it exists
	hasMain := false
	for _, fn := range t.currentModule.Functions {
		if fn.Name == "main" {
			usedFunctions[fn.Name] = true
			hasMain = true
			break
		}
	}

	// If no main function, mark all functions as used (useful for libraries and tests)
	if !hasMain {
		for _, fn := range t.currentModule.Functions {
			usedFunctions[fn.Name] = true
		}
	}

	// Mark functions called by used functions
	for _, fn := range t.currentModule.Functions {
		if usedFunctions[fn.Name] {
			t.markCalledFunctions(fn, usedFunctions)
		}
	}

	// Keep only used functions
	filteredFunctions := make([]*Function, 0)
	for _, fn := range t.currentModule.Functions {
		if usedFunctions[fn.Name] {
			filteredFunctions = append(filteredFunctions, fn)
		}
	}
	t.currentModule.Functions = filteredFunctions
}

// markCalledFunctions marks functions called by the given function as used
func (t *HIRToMIRTransformer) markCalledFunctions(fn *Function, used map[string]bool) {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instr {
			if call, ok := instr.(Call); ok && call.Callee != "" {
				if !used[call.Callee] {
					used[call.Callee] = true
					// Recursively mark called functions
					for _, otherFn := range t.currentModule.Functions {
						if otherFn.Name == call.Callee {
							t.markCalledFunctions(otherFn, used)
							break
						}
					}
				}
			}
		}
	}
}

// optimizeFunction applies function-level optimizations
func (t *HIRToMIRTransformer) optimizeFunction() {
	if t.optimizations.ConstantPropagation {
		t.constantPropagation()
	}

	if t.optimizations.DeadCodeElimination {
		t.deadCodeElimination()
	}

	if t.optimizations.BasicBlockMerging {
		t.mergeBasicBlocks()
	}
}

// constantPropagation performs constant propagation optimization
func (t *HIRToMIRTransformer) constantPropagation() {
	constants := make(map[string]Value)

	for _, block := range t.currentFunction.Blocks {
		for i, instr := range block.Instr {
			switch inst := instr.(type) {
			case BinOp:
				// Try to fold constant binary operations
				if newVal, folded := t.foldConstantBinOp(inst, constants); folded {
					constants[inst.Dst] = newVal
					// Replace instruction with a simpler form if possible
					// (In a full implementation, we'd replace with move or eliminate)
				}

			case Load:
				// Track loaded values if they're constants
				if constVal, isConst := constants[inst.Addr.Ref]; isConst {
					constants[inst.Dst] = constVal
				}

			case Store:
				// Invalidate stored-to addresses
				for ref := range constants {
					if strings.HasPrefix(ref, inst.Addr.Ref) {
						delete(constants, ref)
					}
				}
			}

			// Update instruction if it was modified
			block.Instr[i] = instr
		}
	}
}

// foldConstantBinOp attempts to fold constant binary operations
func (t *HIRToMIRTransformer) foldConstantBinOp(binOp BinOp, constants map[string]Value) (Value, bool) {
	// Get constant values for operands
	var lhsConst, rhsConst Value
	var lhsIsConst, rhsIsConst bool

	if binOp.LHS.Kind == ValConstInt || binOp.LHS.Kind == ValConstFloat {
		lhsConst = binOp.LHS
		lhsIsConst = true
	} else if val, exists := constants[binOp.LHS.Ref]; exists {
		lhsConst = val
		lhsIsConst = true
	}

	if binOp.RHS.Kind == ValConstInt || binOp.RHS.Kind == ValConstFloat {
		rhsConst = binOp.RHS
		rhsIsConst = true
	} else if val, exists := constants[binOp.RHS.Ref]; exists {
		rhsConst = val
		rhsIsConst = true
	}

	// If both operands are constants, fold the operation
	if lhsIsConst && rhsIsConst {
		if lhsConst.Kind == ValConstInt && rhsConst.Kind == ValConstInt {
			var result int64
			switch binOp.Op {
			case OpAdd:
				result = lhsConst.Int64 + rhsConst.Int64
			case OpSub:
				result = lhsConst.Int64 - rhsConst.Int64
			case OpMul:
				result = lhsConst.Int64 * rhsConst.Int64
			case OpDiv:
				if rhsConst.Int64 != 0 {
					result = lhsConst.Int64 / rhsConst.Int64
				} else {
					return Value{}, false // Division by zero
				}
			default:
				return Value{}, false
			}
			return Value{Kind: ValConstInt, Int64: result, Class: ClassInt}, true
		}
	}

	return Value{}, false
}

// deadCodeElimination removes unused instructions and unreachable blocks
func (t *HIRToMIRTransformer) deadCodeElimination() {
	// Mark reachable blocks
	reachable := make(map[string]bool)
	if len(t.currentFunction.Blocks) > 0 {
		t.markReachableBlocks(t.currentFunction.Blocks[0], reachable)
	}

	// Remove unreachable blocks
	filteredBlocks := make([]*BasicBlock, 0)
	for _, block := range t.currentFunction.Blocks {
		if reachable[block.Name] {
			filteredBlocks = append(filteredBlocks, block)
		}
	}
	t.currentFunction.Blocks = filteredBlocks

	// Remove unused instructions within reachable blocks
	for _, block := range t.currentFunction.Blocks {
		t.removeUnusedInstructions(block)
	}
}

// markReachableBlocks marks all blocks reachable from the given block
func (t *HIRToMIRTransformer) markReachableBlocks(block *BasicBlock, reachable map[string]bool) {
	if reachable[block.Name] {
		return // Already visited
	}

	reachable[block.Name] = true

	// Find successor blocks
	if len(block.Instr) > 0 {
		lastInstr := block.Instr[len(block.Instr)-1]
		switch term := lastInstr.(type) {
		case Br:
			if successorBlock := t.findBlock(term.Target); successorBlock != nil {
				t.markReachableBlocks(successorBlock, reachable)
			}
		case CondBr:
			if trueBlock := t.findBlock(term.True); trueBlock != nil {
				t.markReachableBlocks(trueBlock, reachable)
			}
			if falseBlock := t.findBlock(term.False); falseBlock != nil {
				t.markReachableBlocks(falseBlock, reachable)
			}
		}
	}
}

// findBlock finds a block by name in the current function
func (t *HIRToMIRTransformer) findBlock(name string) *BasicBlock {
	for _, block := range t.currentFunction.Blocks {
		if block.Name == name {
			return block
		}
	}
	return nil
}

// removeUnusedInstructions removes instructions whose results are never used
func (t *HIRToMIRTransformer) removeUnusedInstructions(block *BasicBlock) {
	// Build use-def chains
	used := make(map[string]bool)

	// Mark all used values
	for _, instr := range block.Instr {
		switch inst := instr.(type) {
		case BinOp:
			t.markValueUsed(inst.LHS, used)
			t.markValueUsed(inst.RHS, used)
		case Load:
			t.markValueUsed(inst.Addr, used)
		case Store:
			t.markValueUsed(inst.Addr, used)
			t.markValueUsed(inst.Val, used)
		case Ret:
			if inst.Val != nil {
				t.markValueUsed(*inst.Val, used)
			}
		case Call:
			for _, arg := range inst.Args {
				t.markValueUsed(arg, used)
			}
		case Cmp:
			t.markValueUsed(inst.LHS, used)
			t.markValueUsed(inst.RHS, used)
		case CondBr:
			t.markValueUsed(inst.Cond, used)
		}
	}

	// Remove instructions that define unused values
	filteredInstr := make([]Instr, 0)
	for _, instr := range block.Instr {
		keep := true
		switch inst := instr.(type) {
		case BinOp:
			if !used[inst.Dst] {
				keep = false
			}
		case Load:
			if !used[inst.Dst] {
				keep = false
			}
		case Call:
			if inst.Dst != "" && !used[inst.Dst] {
				keep = false
			}
		case Cmp:
			if !used[inst.Dst] {
				keep = false
			}
		}

		if keep {
			filteredInstr = append(filteredInstr, instr)
		}
	}
	block.Instr = filteredInstr
}

// markValueUsed marks a value as used in the use-def analysis
func (t *HIRToMIRTransformer) markValueUsed(val Value, used map[string]bool) {
	if val.Kind == ValRef && val.Ref != "" {
		used[val.Ref] = true
	}
}

// mergeBasicBlocks merges basic blocks that can be combined
func (t *HIRToMIRTransformer) mergeBasicBlocks() {
	changed := true
	for changed {
		changed = false

		for _, block := range t.currentFunction.Blocks {
			if len(block.Instr) == 0 {
				continue
			}

			// Check if block ends with unconditional branch
			lastInstr := block.Instr[len(block.Instr)-1]
			if br, ok := lastInstr.(Br); ok {
				// Find target block
				targetBlock := t.findBlock(br.Target)
				if targetBlock != nil && t.canMergeBlocks(block, targetBlock) {
					// Merge blocks
					block.Instr = block.Instr[:len(block.Instr)-1] // Remove branch
					block.Instr = append(block.Instr, targetBlock.Instr...)

					// Update all references to target block
					t.replaceBlockReferences(targetBlock.Name, block.Name)

					// Remove target block
					t.removeBlock(targetBlock.Name)

					changed = true
					break
				}
			}
		}
	}
}

// canMergeBlocks checks if two blocks can be merged
func (t *HIRToMIRTransformer) canMergeBlocks(source, target *BasicBlock) bool {
	// Target block should only have one predecessor (the source block)
	predecessors := 0
	for _, block := range t.currentFunction.Blocks {
		if len(block.Instr) == 0 {
			continue
		}

		lastInstr := block.Instr[len(block.Instr)-1]
		switch term := lastInstr.(type) {
		case Br:
			if term.Target == target.Name {
				predecessors++
			}
		case CondBr:
			if term.True == target.Name || term.False == target.Name {
				predecessors++
			}
		}
	}

	return predecessors == 1
}

// replaceBlockReferences replaces all references to oldName with newName
func (t *HIRToMIRTransformer) replaceBlockReferences(oldName, newName string) {
	for _, block := range t.currentFunction.Blocks {
		if len(block.Instr) == 0 {
			continue
		}

		lastInstrIdx := len(block.Instr) - 1
		switch term := block.Instr[lastInstrIdx].(type) {
		case Br:
			if term.Target == oldName {
				block.Instr[lastInstrIdx] = Br{Target: newName}
			}
		case CondBr:
			newTerm := term
			if term.True == oldName {
				newTerm.True = newName
			}
			if term.False == oldName {
				newTerm.False = newName
			}
			block.Instr[lastInstrIdx] = newTerm
		}
	}
}

// removeBlock removes a block from the function
func (t *HIRToMIRTransformer) removeBlock(name string) {
	filteredBlocks := make([]*BasicBlock, 0)
	for _, block := range t.currentFunction.Blocks {
		if block.Name != name {
			filteredBlocks = append(filteredBlocks, block)
		}
	}
	t.currentFunction.Blocks = filteredBlocks
}
