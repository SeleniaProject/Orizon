// Package parser implements the Orizon macro system foundation
// Phase 1.2.3: マクロシステム基盤実装
package parser

import (
	"fmt"
	"sync/atomic"
)

// MacroEngine represents the core macro expansion engine
type MacroEngine struct {
	definitions    map[string]*MacroDefinition
	scopeCounter   uint64
	expansionDepth int
	maxDepth       int
	hygienicMode   bool
	builtins       *MacroBuiltins
}

// NewMacroEngine creates a new macro engine instance
func NewMacroEngine() *MacroEngine {
	return &MacroEngine{
		definitions:    make(map[string]*MacroDefinition),
		scopeCounter:   0,
		expansionDepth: 0,
		maxDepth:       1000, // Prevent infinite recursion
		hygienicMode:   true,
		builtins:       NewMacroBuiltins(),
	}
}

// RegisterMacro registers a macro definition in the engine
func (me *MacroEngine) RegisterMacro(macro *MacroDefinition) error {
	if macro == nil || macro.Name == nil {
		return fmt.Errorf("invalid macro definition")
	}

	name := macro.Name.Value
	if _, exists := me.definitions[name]; exists {
		return fmt.Errorf("macro '%s' already defined", name)
	}

	me.definitions[name] = macro
	return nil
}

// GetMacro retrieves a macro definition by name
func (me *MacroEngine) GetMacro(name string) (*MacroDefinition, bool) {
	macro, exists := me.definitions[name]
	return macro, exists
}

// ExpandMacro expands a macro invocation into AST nodes
func (me *MacroEngine) ExpandMacro(invocation *MacroInvocation) ([]Statement, error) {
	if me.expansionDepth >= me.maxDepth {
		return nil, fmt.Errorf("macro expansion depth limit exceeded")
	}

	macro, exists := me.GetMacro(invocation.Name.Value)
	if !exists {
		// Try builtins before failing
		if me.builtins != nil {
			if fn, ok := me.builtins.GetBuiltin(invocation.Name.Value); ok {
				me.expansionDepth++
				defer func() { me.expansionDepth-- }()
				return fn(invocation.Arguments)
			}
		}
		return nil, fmt.Errorf("unknown macro: %s", invocation.Name.Value)
	}

	me.expansionDepth++
	defer func() { me.expansionDepth-- }()

	// Create hygienic context if needed
	var context *MacroContext
	if macro.IsHygienic && me.hygienicMode {
		context = me.createHygienicContext(invocation)
	}

	// Match macro templates and expand
	template, bindings, err := me.matchTemplate(macro, invocation)
	if err != nil {
		return nil, fmt.Errorf("macro expansion failed: %v", err)
	}

	// Expand the matched template
	result, err := me.expandTemplate(template, bindings, context)
	if err != nil {
		return nil, fmt.Errorf("template expansion failed: %v", err)
	}

	return result, nil
}

// createHygienicContext creates a new hygienic context for macro expansion
func (me *MacroEngine) createHygienicContext(invocation *MacroInvocation) *MacroContext {
	scopeId := atomic.AddUint64(&me.scopeCounter, 1)

	return &MacroContext{
		Span:           invocation.Span,
		ScopeId:        scopeId,
		CapturedNames:  make(map[string]string),
		ExpansionDepth: me.expansionDepth,
		SourceLocation: invocation.Span.Start,
	}
}

// matchTemplate finds the best matching template for a macro invocation
func (me *MacroEngine) matchTemplate(macro *MacroDefinition, invocation *MacroInvocation) (*MacroTemplate, map[string]interface{}, error) {
	if macro.Body == nil || len(macro.Body.Templates) == 0 {
		return nil, nil, fmt.Errorf("macro has no templates")
	}

	var bestTemplate *MacroTemplate
	var bestBindings map[string]interface{}
	bestScore := -1

	// Try each template in priority order
	for _, template := range macro.Body.Templates {
		bindings, score := me.matchPattern(template.Pattern, invocation.Arguments)
		if score > bestScore {
			// Check guard condition if present
			if template.Guard != nil {
				if !me.evaluateGuard(template.Guard, bindings) {
					continue
				}
			}
			bestTemplate = template
			bestBindings = bindings
			bestScore = score
		}
	}

	if bestTemplate == nil {
		return nil, nil, fmt.Errorf("no matching template found")
	}

	return bestTemplate, bestBindings, nil
}

// matchPattern matches a macro pattern against arguments
func (me *MacroEngine) matchPattern(pattern *MacroPattern, args []*MacroArgument) (map[string]interface{}, int) {
	if pattern == nil {
		return nil, -1
	}

	bindings := make(map[string]interface{})
	score := 0
	argIndex := 0

	for _, element := range pattern.Elements {
		switch element.Kind {
		case MacroPatternLiteral:
			// Literal patterns must match exactly
			if argIndex >= len(args) {
				return nil, -1
			}
			// For now, skip literal matching implementation
			argIndex++
			score += 10

		case MacroPatternParameter:
			// Parameter patterns bind to arguments
			if argIndex >= len(args) {
				return nil, -1
			}
			bindings[element.Value] = args[argIndex]
			argIndex++
			score += 20

		case MacroPatternWildcard:
			// Wildcard patterns match anything
			if argIndex >= len(args) {
				return nil, -1
			}
			argIndex++
			score += 5

		case MacroPatternGroup:
			// Group patterns handle repetition
			if element.Repetition != nil {
				consumed := me.matchRepetition(element.Repetition, args[argIndex:])
				if consumed == -1 {
					return nil, -1
				}
				argIndex += consumed
				score += consumed * 15
			}
		}
	}

	// Ensure all arguments are consumed
	if argIndex != len(args) {
		return nil, -1
	}

	return bindings, score
}

// matchRepetition handles repetition patterns in macro matching
func (me *MacroEngine) matchRepetition(repetition *MacroRepetition, args []*MacroArgument) int {
	switch repetition.Kind {
	case MacroRepeatZeroOrMore:
		return len(args) // Consume all remaining arguments
	case MacroRepeatOneOrMore:
		if len(args) == 0 {
			return -1
		}
		return len(args)
	case MacroRepeatOptional:
		if len(args) > 0 {
			return 1
		}
		return 0
	case MacroRepeatExact:
		if len(args) == repetition.Min {
			return repetition.Min
		}
		return -1
	case MacroRepeatRange:
		if len(args) >= repetition.Min && (repetition.Max == -1 || len(args) <= repetition.Max) {
			return len(args)
		}
		return -1
	}
	return -1
}

// evaluateGuard evaluates a guard expression
func (me *MacroEngine) evaluateGuard(guard Expression, bindings map[string]interface{}) bool {
	// For now, implement basic guard evaluation
	// In a full implementation, this would be a complete expression evaluator
	switch g := guard.(type) {
	case *Literal:
		if b, ok := g.Value.(bool); ok {
			return b
		}
	case *Identifier:
		if val, exists := bindings[g.Value]; exists {
			if b, ok := val.(bool); ok {
				return b
			}
		}
	}
	return true // Default to true for unimplemented cases
}

// expandTemplate expands a macro template with bindings
func (me *MacroEngine) expandTemplate(template *MacroTemplate, bindings map[string]interface{}, context *MacroContext) ([]Statement, error) {
	if template == nil || len(template.Body) == 0 {
		return []Statement{}, nil
	}

	var result []Statement
	expander := &MacroExpander{
		engine:   me,
		bindings: bindings,
		context:  context,
	}

	for _, stmt := range template.Body {
		expanded, err := expander.expandStatement(stmt)
		if err != nil {
			return nil, err
		}
		result = append(result, expanded...)
	}

	return result, nil
}

// MacroExpander handles the actual expansion of macro templates
type MacroExpander struct {
	engine   *MacroEngine
	bindings map[string]interface{}
	context  *MacroContext
}

// expandStatement expands a single statement with macro substitutions
func (ex *MacroExpander) expandStatement(stmt Statement) ([]Statement, error) {
	switch s := stmt.(type) {
	case *ExpressionStatement:
		expr, err := ex.expandExpression(s.Expression)
		if err != nil {
			return nil, err
		}
		return []Statement{&ExpressionStatement{
			Span:       s.Span,
			Expression: expr,
		}}, nil

	case *VariableDeclaration:
		var typeSpec Type
		var initializer Expression
		var err error

		if s.TypeSpec != nil {
			typeSpec, err = ex.expandType(s.TypeSpec)
			if err != nil {
				return nil, err
			}
		}

		if s.Initializer != nil {
			initializer, err = ex.expandExpression(s.Initializer)
			if err != nil {
				return nil, err
			}
		}

		name := s.Name
		if ex.context != nil {
			name = ex.applyHygiene(s.Name)
		}

		return []Statement{&VariableDeclaration{
			Span:        s.Span,
			Name:        name,
			TypeSpec:    typeSpec,
			Initializer: initializer,
			IsMutable:   s.IsMutable,
			IsPublic:    s.IsPublic,
		}}, nil

	case *BlockStatement:
		var expandedStmts []Statement
		for _, blockStmt := range s.Statements {
			expanded, err := ex.expandStatement(blockStmt)
			if err != nil {
				return nil, err
			}
			expandedStmts = append(expandedStmts, expanded...)
		}
		return []Statement{&BlockStatement{
			Span:       s.Span,
			Statements: expandedStmts,
		}}, nil

	case *ReturnStatement:
		if s.Value != nil {
			value, err := ex.expandExpression(s.Value)
			if err != nil {
				return nil, err
			}
			return []Statement{&ReturnStatement{
				Span:  s.Span,
				Value: value,
			}}, nil
		}
		return []Statement{s}, nil

	default:
		// For unimplemented statement types, return as-is
		return []Statement{stmt}, nil
	}
}

// expandExpression expands an expression with macro substitutions
func (ex *MacroExpander) expandExpression(expr Expression) (Expression, error) {
	switch e := expr.(type) {
	case *Identifier:
		// Check for parameter substitution
		if val, exists := ex.bindings[e.Value]; exists {
			if arg, ok := val.(*MacroArgument); ok {
				if argExpr, ok := arg.Value.(Expression); ok {
					return argExpr, nil
				}
			}
		}
		// Apply hygiene if needed
		if ex.context != nil {
			return ex.applyHygiene(e), nil
		}
		return e, nil

	case *BinaryExpression:
		left, err := ex.expandExpression(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := ex.expandExpression(e.Right)
		if err != nil {
			return nil, err
		}
		return &BinaryExpression{
			Span:     e.Span,
			Left:     left,
			Operator: e.Operator,
			Right:    right,
		}, nil

	case *UnaryExpression:
		operand, err := ex.expandExpression(e.Operand)
		if err != nil {
			return nil, err
		}
		return &UnaryExpression{
			Span:     e.Span,
			Operator: e.Operator,
			Operand:  operand,
		}, nil

	case *CallExpression:
		function, err := ex.expandExpression(e.Function)
		if err != nil {
			return nil, err
		}
		var args []Expression
		for _, arg := range e.Arguments {
			expandedArg, err := ex.expandExpression(arg)
			if err != nil {
				return nil, err
			}
			args = append(args, expandedArg)
		}
		return &CallExpression{
			Span:      e.Span,
			Function:  function,
			Arguments: args,
		}, nil

	case *MacroInvocation:
		// Handle nested macro invocations
		expanded, err := ex.engine.ExpandMacro(e)
		if err != nil {
			return nil, err
		}
		if len(expanded) == 1 {
			if exprStmt, ok := expanded[0].(*ExpressionStatement); ok {
				return exprStmt.Expression, nil
			}
		}
		return nil, fmt.Errorf("macro expansion did not result in single expression")

	default:
		// For literal values and other expressions, return as-is
		return expr, nil
	}
}

// expandType expands a type with macro substitutions
func (ex *MacroExpander) expandType(t Type) (Type, error) {
	switch typ := t.(type) {
	case *BasicType:
		// Apply hygiene if needed
		if ex.context != nil && typ.Name != "" {
			if uniqueName, exists := ex.context.CapturedNames[typ.Name]; exists {
				return &BasicType{
					Span: typ.Span,
					Name: uniqueName,
				}, nil
			}
		}
		return typ, nil
	default:
		return t, nil
	}
}

// applyHygiene applies hygienic renaming to identifiers
func (ex *MacroExpander) applyHygiene(id *Identifier) *Identifier {
	if ex.context == nil {
		return id
	}

	originalName := id.Value
	if uniqueName, exists := ex.context.CapturedNames[originalName]; exists {
		return &Identifier{
			Span:  id.Span,
			Value: uniqueName,
		}
	}

	// Generate unique name
	uniqueName := fmt.Sprintf("%s__%d", originalName, ex.context.ScopeId)
	ex.context.CapturedNames[originalName] = uniqueName

	return &Identifier{
		Span:  id.Span,
		Value: uniqueName,
	}
}

// MacroValidator validates macro definitions for correctness
type MacroValidator struct {
	errors []error
}

// NewMacroValidator creates a new macro validator
func NewMacroValidator() *MacroValidator {
	return &MacroValidator{
		errors: make([]error, 0),
	}
}

// ValidateMacro validates a macro definition
func (mv *MacroValidator) ValidateMacro(macro *MacroDefinition) []error {
	mv.errors = make([]error, 0)

	if macro == nil {
		mv.addError("macro definition is nil")
		return mv.errors
	}

	if macro.Name == nil || macro.Name.Value == "" {
		mv.addError("macro must have a valid name")
	}

	if macro.Body == nil {
		mv.addError("macro must have a body")
	} else {
		mv.validateMacroBody(macro.Body)
	}

	mv.validateMacroParameters(macro.Parameters)

	return mv.errors
}

// validateMacroBody validates the body of a macro
func (mv *MacroValidator) validateMacroBody(body *MacroBody) {
	if len(body.Templates) == 0 {
		mv.addError("macro body must contain at least one template")
		return
	}

	for i, template := range body.Templates {
		mv.validateMacroTemplate(template, i)
	}
}

// validateMacroTemplate validates a macro template
func (mv *MacroValidator) validateMacroTemplate(template *MacroTemplate, index int) {
	if template.Pattern == nil {
		mv.addError(fmt.Sprintf("template %d must have a pattern", index))
	} else {
		mv.validateMacroPattern(template.Pattern)
	}

	if len(template.Body) == 0 {
		mv.addError(fmt.Sprintf("template %d must have a non-empty body", index))
	}
}

// validateMacroPattern validates a macro pattern
func (mv *MacroValidator) validateMacroPattern(pattern *MacroPattern) {
	if len(pattern.Elements) == 0 {
		mv.addError("macro pattern must contain at least one element")
		return
	}

	paramNames := make(map[string]bool)
	for _, element := range pattern.Elements {
		if element.Kind == MacroPatternParameter {
			if paramNames[element.Value] {
				mv.addError(fmt.Sprintf("duplicate parameter name: %s", element.Value))
			}
			paramNames[element.Value] = true
		}
	}
}

// validateMacroParameters validates macro parameters
func (mv *MacroValidator) validateMacroParameters(params []*MacroParameter) {
	names := make(map[string]bool)
	for _, param := range params {
		if param.Name == nil || param.Name.Value == "" {
			mv.addError("macro parameter must have a valid name")
			continue
		}

		name := param.Name.Value
		if names[name] {
			mv.addError(fmt.Sprintf("duplicate parameter name: %s", name))
		}
		names[name] = true
	}
}

// addError adds an error to the validator
func (mv *MacroValidator) addError(message string) {
	mv.errors = append(mv.errors, fmt.Errorf(message))
}

// MacroBuiltins provides built-in macro functions
type MacroBuiltins struct {
	functions map[string]func([]*MacroArgument) ([]Statement, error)
}

// NewMacroBuiltins creates a new macro builtins registry
func NewMacroBuiltins() *MacroBuiltins {
	mb := &MacroBuiltins{
		functions: make(map[string]func([]*MacroArgument) ([]Statement, error)),
	}

	// Register built-in macros
	mb.registerBuiltins()
	return mb
}

// registerBuiltins registers built-in macro functions
func (mb *MacroBuiltins) registerBuiltins() {
	mb.functions["println"] = mb.printlnMacro
	mb.functions["debug_print"] = mb.debugPrintMacro
	mb.functions["compile_time_assert"] = mb.compileTimeAssertMacro
}

// printlnMacro implements a simple println macro
func (mb *MacroBuiltins) printlnMacro(args []*MacroArgument) ([]Statement, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("println macro expects exactly 1 argument")
	}

	arg := args[0]
	if arg.Kind != MacroArgExpression {
		return nil, fmt.Errorf("println macro expects an expression argument")
	}

	expr, ok := arg.Value.(Expression)
	if !ok {
		return nil, fmt.Errorf("invalid expression argument")
	}

	// Create a function call to println
	printlnCall := &CallExpression{
		Span: arg.Span,
		Function: &Identifier{
			Span:  arg.Span,
			Value: "println",
		},
		Arguments: []Expression{expr},
	}

	return []Statement{&ExpressionStatement{
		Span:       arg.Span,
		Expression: printlnCall,
	}}, nil
}

// debugPrintMacro implements a debug print macro
func (mb *MacroBuiltins) debugPrintMacro(args []*MacroArgument) ([]Statement, error) {
	var statements []Statement

	for _, arg := range args {
		if arg.Kind != MacroArgExpression {
			continue
		}

		expr, ok := arg.Value.(Expression)
		if !ok {
			continue
		}

		// Create debug print statement
		debugCall := &CallExpression{
			Span: arg.Span,
			Function: &Identifier{
				Span:  arg.Span,
				Value: "debug_print",
			},
			Arguments: []Expression{expr},
		}

		statements = append(statements, &ExpressionStatement{
			Span:       arg.Span,
			Expression: debugCall,
		})
	}

	return statements, nil
}

// compileTimeAssertMacro implements a compile-time assertion macro
func (mb *MacroBuiltins) compileTimeAssertMacro(args []*MacroArgument) ([]Statement, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("compile_time_assert macro expects exactly 1 argument")
	}

	// For now, just return an empty statement
	// In a full implementation, this would evaluate the condition at compile time
	return []Statement{}, nil
}

// GetBuiltin returns a built-in macro function by name
func (mb *MacroBuiltins) GetBuiltin(name string) (func([]*MacroArgument) ([]Statement, error), bool) {
	fn, exists := mb.functions[name]
	return fn, exists
}
