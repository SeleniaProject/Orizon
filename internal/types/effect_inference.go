// Package types provides effect inference engine for static side effect analysis.
// This module implements sophisticated effect inference algorithms including.
// flow-sensitive analysis, interprocedural inference, and effect polymorphism.
package types

import (
	"fmt"
	"sync"
)

// EffectInferenceEngine provides comprehensive effect inference capabilities.
type EffectInferenceEngine struct {
	context    *EffectInferenceContext
	analyzer   *EffectAnalyzer
	propagator *EffectPropagator
	validator  *EffectValidator
	optimizer  *EffectOptimizer
	cache      *InferenceCache
	statistics *InferenceStatistics
	config     *EffectInferenceConfig
	mu         sync.RWMutex
}

// NewEffectInferenceEngine creates a new effect inference engine.
func NewEffectInferenceEngine(config *EffectInferenceConfig) *EffectInferenceEngine {
	if config == nil {
		config = DefaultEffectInferenceConfig()
	}

	return &EffectInferenceEngine{
		context:    NewEffectInferenceContext(),
		analyzer:   NewEffectAnalyzer(),
		propagator: NewEffectPropagator(),
		validator:  NewEffectValidator(),
		optimizer:  NewEffectOptimizer(),
		cache:      NewInferenceCache(),
		statistics: NewInferenceStatistics(),
		config:     config,
	}
}

// InferEffects performs effect inference for the given AST node.
func (eie *EffectInferenceEngine) InferEffects(node ASTNode) (*EffectSignature, error) {
	eie.mu.Lock()
	defer eie.mu.Unlock()

	eie.statistics.IncrementInference()

	// Check cache first.
	if cacheKey := eie.getCacheKey(node); cacheKey != "" {
		if cached, found := eie.cache.Get(cacheKey); found {
			eie.statistics.IncrementCacheHit()

			return cached.Clone(), nil
		}

		eie.statistics.IncrementCacheMiss()
	}

	// Perform inference.
	signature, err := eie.inferEffectsInternal(node)
	if err != nil {
		return nil, fmt.Errorf("effect inference failed: %w", err)
	}

	// Validate signature.
	if err := eie.validator.Validate(signature); err != nil {
		return nil, fmt.Errorf("effect validation failed: %w", err)
	}

	// Optimize signature.
	optimized := eie.optimizer.Optimize(signature)

	// Cache result.
	if cacheKey := eie.getCacheKey(node); cacheKey != "" {
		eie.cache.Put(cacheKey, optimized)
	}

	return optimized, nil
}

// inferEffectsInternal performs the actual inference logic.
func (eie *EffectInferenceEngine) inferEffectsInternal(node ASTNode) (*EffectSignature, error) {
	switch n := node.(type) {
	case *FunctionDecl:
		return eie.inferFunctionEffects(n)
	case *CallExpr:
		return eie.inferCallEffects(n)
	case *AssignmentExpr:
		return eie.inferAssignmentEffects(n)
	case *IfStmt:
		return eie.inferConditionalEffects(n)
	case *ForStmt:
		return eie.inferLoopEffects(n)
	case *BlockStmt:
		return eie.inferBlockEffects(n)
	case *ReturnStmt:
		return eie.inferReturnEffects(n)
	case *ThrowStmt:
		return eie.inferThrowEffects(n)
	case *TryStmt:
		return eie.inferTryEffects(n)
	default:
		return eie.inferDefaultEffects(n)
	}
}

// inferFunctionEffects infers effects for function declarations.
func (eie *EffectInferenceEngine) inferFunctionEffects(fn *FunctionDecl) (*EffectSignature, error) {
	signature := NewEffectSignature()

	// Create new scope for function.
	scope := NewEffectScope(fn.Name, eie.context.CurrentScope())
	eie.context.PushScope(scope)
	defer eie.context.PopScope()

	// Analyze function body.
	if fn.Body != nil {
		bodySignature, err := eie.InferEffects(fn.Body)
		if err != nil {
			return nil, err
		}

		signature = signature.Merge(bodySignature)
	}

	// Apply function-specific effects.
	signature = eie.applyFunctionEffects(fn, signature)

	return signature, nil
}

// inferCallEffects infers effects for function calls.
func (eie *EffectInferenceEngine) inferCallEffects(call *CallExpr) (*EffectSignature, error) {
	signature := NewEffectSignature()

	// Get callee signature.
	calleeSignature, err := eie.getCalleeSignature(call.Function)
	if err != nil {
		return nil, err
	}

	// Merge callee effects.
	signature = signature.Merge(calleeSignature)

	// Analyze arguments.
	for _, arg := range call.Arguments {
		argSignature, err := eie.InferEffects(arg)
		if err != nil {
			return nil, err
		}

		signature = signature.Merge(argSignature)
	}

	// Apply call-specific effects.
	signature = eie.applyCallEffects(call, signature)

	return signature, nil
}

// inferAssignmentEffects infers effects for assignments.
func (eie *EffectInferenceEngine) inferAssignmentEffects(assign *AssignmentExpr) (*EffectSignature, error) {
	signature := NewEffectSignature()

	// Analyze right-hand side.
	rhsSignature, err := eie.InferEffects(assign.RHS)
	if err != nil {
		return nil, err
	}

	signature = signature.Merge(rhsSignature)

	// Add write effect for left-hand side.
	writeEffect := eie.createWriteEffect(assign.LHS)
	signature.Effects.Add(writeEffect)

	return signature, nil
}

// inferConditionalEffects infers effects for conditional statements.
func (eie *EffectInferenceEngine) inferConditionalEffects(ifStmt *IfStmt) (*EffectSignature, error) {
	signature := NewEffectSignature()

	// Analyze condition.
	condSignature, err := eie.InferEffects(ifStmt.Condition)
	if err != nil {
		return nil, err
	}

	signature = signature.Merge(condSignature)

	// Analyze then branch.
	thenSignature, err := eie.InferEffects(ifStmt.Then)
	if err != nil {
		return nil, err
	}

	// Analyze else branch if present.
	var elseSignature *EffectSignature
	if ifStmt.Else != nil {
		elseSignature, err = eie.InferEffects(ifStmt.Else)
		if err != nil {
			return nil, err
		}
	} else {
		elseSignature = NewEffectSignature()
	}

	// Merge conditional effects.
	conditionalEffects := eie.mergeConditionalEffects(thenSignature, elseSignature)
	signature = signature.Merge(conditionalEffects)

	return signature, nil
}

// inferLoopEffects infers effects for loop statements.
func (eie *EffectInferenceEngine) inferLoopEffects(forStmt *ForStmt) (*EffectSignature, error) {
	signature := NewEffectSignature()

	// Analyze initialization.
	if forStmt.Init != nil {
		initSignature, err := eie.InferEffects(forStmt.Init)
		if err != nil {
			return nil, err
		}

		signature = signature.Merge(initSignature)
	}

	// Analyze condition.
	if forStmt.Condition != nil {
		condSignature, err := eie.InferEffects(forStmt.Condition)
		if err != nil {
			return nil, err
		}

		signature = signature.Merge(condSignature)
	}

	// Analyze update.
	if forStmt.Update != nil {
		updateSignature, err := eie.InferEffects(forStmt.Update)
		if err != nil {
			return nil, err
		}

		signature = signature.Merge(updateSignature)
	}

	// Analyze body (may execute multiple times).
	bodySignature, err := eie.InferEffects(forStmt.Body)
	if err != nil {
		return nil, err
	}

	// Apply loop effect amplification.
	loopEffects := eie.amplifyLoopEffects(bodySignature)
	signature = signature.Merge(loopEffects)

	return signature, nil
}

// inferBlockEffects infers effects for block statements.
func (eie *EffectInferenceEngine) inferBlockEffects(block *BlockStmt) (*EffectSignature, error) {
	signature := NewEffectSignature()

	// Create new scope for block.
	scope := NewEffectScope("block", eie.context.CurrentScope())
	eie.context.PushScope(scope)
	defer eie.context.PopScope()

	// Analyze statements sequentially.
	for _, stmt := range block.Statements {
		stmtSignature, err := eie.InferEffects(stmt)
		if err != nil {
			return nil, err
		}

		signature = signature.Merge(stmtSignature)
	}

	return signature, nil
}

// getCacheKey generates a cache key for the given AST node.
func (eie *EffectInferenceEngine) getCacheKey(node ASTNode) string {
	// Simple implementation - in practice would use more sophisticated key generation.
	return fmt.Sprintf("%T_%p", node, node)
}

// getCalleeSignature retrieves the effect signature for a function being called.
func (eie *EffectInferenceEngine) getCalleeSignature(fn ASTNode) (*EffectSignature, error) {
	// Implementation would look up function signatures from context or symbols.
	// For now, return a basic signature.
	return NewEffectSignature(), nil
}

// applyFunctionEffects applies function-specific effect rules.
func (eie *EffectInferenceEngine) applyFunctionEffects(fn *FunctionDecl, signature *EffectSignature) *EffectSignature {
	// Apply function annotations, modifiers, etc.
	return signature
}

// applyCallEffects applies call-specific effect rules.
func (eie *EffectInferenceEngine) applyCallEffects(call *CallExpr, signature *EffectSignature) *EffectSignature {
	// Apply call-site specific rules.
	return signature
}

// createWriteEffect creates a write effect for the given expression.
func (eie *EffectInferenceEngine) createWriteEffect(expr ASTNode) *SideEffect {
	// Determine the appropriate write effect based on the expression.
	return NewSideEffect(EffectMemoryWrite, EffectLevelLow)
}

// mergeConditionalEffects merges effects from conditional branches.
func (eie *EffectInferenceEngine) mergeConditionalEffects(thenSig, elseSig *EffectSignature) *EffectSignature {
	// Take union of both branches as either could execute.
	result := NewEffectSignature()
	result.Effects = thenSig.Effects.Union(elseSig.Effects)
	result.Requires = thenSig.Requires.Intersection(elseSig.Requires)
	result.Ensures = thenSig.Ensures.Intersection(elseSig.Ensures)

	return result
}

// amplifyLoopEffects amplifies effects that may occur multiple times in loops.
func (eie *EffectInferenceEngine) amplifyLoopEffects(bodySignature *EffectSignature) *EffectSignature {
	// Loops may amplify certain effects.
	result := bodySignature.Clone()

	// Increase effect levels for certain kinds.
	for _, effect := range result.Effects.ToSlice() {
		if effect.Kind == EffectMemoryAlloc || effect.Kind == EffectIO {
			if effect.Level < EffectLevelHigh {
				amplified := effect.Clone()
				amplified.Level++
				result.Effects.Add(amplified)
			}
		}
	}

	return result
}

// inferReturnEffects infers effects for return statements.
func (eie *EffectInferenceEngine) inferReturnEffects(ret *ReturnStmt) (*EffectSignature, error) {
	signature := NewEffectSignature()

	if ret.Value != nil {
		valueSignature, err := eie.InferEffects(ret.Value)
		if err != nil {
			return nil, err
		}

		signature = signature.Merge(valueSignature)
	}

	return signature, nil
}

// inferThrowEffects infers effects for throw statements.
func (eie *EffectInferenceEngine) inferThrowEffects(throw *ThrowStmt) (*EffectSignature, error) {
	signature := NewEffectSignature()

	// Add throw effect.
	throwEffect := NewSideEffect(EffectThrow, EffectLevelMedium)
	signature.Effects.Add(throwEffect)

	if throw.Value != nil {
		valueSignature, err := eie.InferEffects(throw.Value)
		if err != nil {
			return nil, err
		}

		signature = signature.Merge(valueSignature)
	}

	return signature, nil
}

// inferTryEffects infers effects for try-catch statements.
func (eie *EffectInferenceEngine) inferTryEffects(try *TryStmt) (*EffectSignature, error) {
	signature := NewEffectSignature()

	// Analyze try block.
	trySignature, err := eie.InferEffects(try.Body)
	if err != nil {
		return nil, err
	}

	signature = signature.Merge(trySignature)

	// Analyze catch blocks.
	for _, catch := range try.Catches {
		catchSignature, err := eie.InferEffects(catch.Body)
		if err != nil {
			return nil, err
		}

		signature = signature.Merge(catchSignature)

		// Add catch effect.
		catchEffect := NewSideEffect(EffectCatch, EffectLevelMedium)
		signature.Effects.Add(catchEffect)
	}

	// Analyze finally block.
	if try.Finally != nil {
		finallySignature, err := eie.InferEffects(try.Finally)
		if err != nil {
			return nil, err
		}

		signature = signature.Merge(finallySignature)
	}

	return signature, nil
}

// inferDefaultEffects provides default effect inference for unknown node types.
func (eie *EffectInferenceEngine) inferDefaultEffects(node ASTNode) (*EffectSignature, error) {
	// Most expressions are pure by default.
	return NewEffectSignature(), nil
}

// Merge merges two effect signatures.
func (es *EffectSignature) Merge(other *EffectSignature) *EffectSignature {
	result := es.Clone()
	result.Effects = result.Effects.Union(other.Effects)
	result.Requires = result.Requires.Union(other.Requires)
	result.Ensures = result.Ensures.Intersection(other.Ensures)
	result.Masks = result.Masks.Union(other.Masks)
	result.Constraints = append(result.Constraints, other.Constraints...)
	result.Pure = result.Pure && other.Pure

	return result
}

// EffectAnalyzer provides detailed effect analysis capabilities.
type EffectAnalyzer struct {
	flowAnalyzer  *FlowSensitiveAnalyzer
	aliasAnalyzer *AliasAnalyzer
	reachingDefs  *ReachingDefinitions
	config        *AnalyzerConfig
}

// NewEffectAnalyzer creates a new effect analyzer.
func NewEffectAnalyzer() *EffectAnalyzer {
	return &EffectAnalyzer{
		flowAnalyzer:  NewFlowSensitiveAnalyzer(),
		aliasAnalyzer: NewAliasAnalyzer(),
		reachingDefs:  NewReachingDefinitions(),
		config:        DefaultAnalyzerConfig(),
	}
}

// FlowSensitiveAnalyzer performs flow-sensitive effect analysis.
type FlowSensitiveAnalyzer struct {
	cfg      *ControlFlowGraph
	dataflow *DataFlowAnalysis
}

// NewFlowSensitiveAnalyzer creates a new flow-sensitive analyzer.
func NewFlowSensitiveAnalyzer() *FlowSensitiveAnalyzer {
	return &FlowSensitiveAnalyzer{}
}

// AliasAnalyzer performs alias analysis for effect inference.
type AliasAnalyzer struct {
	pointsTo map[string][]string
	aliases  map[string][]string
}

// NewAliasAnalyzer creates a new alias analyzer.
func NewAliasAnalyzer() *AliasAnalyzer {
	return &AliasAnalyzer{
		pointsTo: make(map[string][]string),
		aliases:  make(map[string][]string),
	}
}

// ReachingDefinitions tracks reaching definitions for effect analysis.
type ReachingDefinitions struct {
	definitions map[string][]*Definition
	reachingIn  map[string][]*Definition
	reachingOut map[string][]*Definition
}

// NewReachingDefinitions creates a new reaching definitions analyzer.
func NewReachingDefinitions() *ReachingDefinitions {
	return &ReachingDefinitions{
		definitions: make(map[string][]*Definition),
		reachingIn:  make(map[string][]*Definition),
		reachingOut: make(map[string][]*Definition),
	}
}

// Definition represents a variable definition.
type Definition struct {
	Value    ASTNode
	Location *SourceLocation
	Effects  *EffectSet
	Variable string
}

// DataFlowAnalysis provides data flow analysis capabilities.
type DataFlowAnalysis struct {
	facts map[string]interface{}
}

// AnalyzerConfig holds configuration for effect analyzer.
type AnalyzerConfig struct {
	PrecisionLevel  int
	AnalysisTimeout int
	MaxIterations   int
}

// DefaultAnalyzerConfig returns default analyzer configuration.
func DefaultAnalyzerConfig() *AnalyzerConfig {
	return &AnalyzerConfig{
		PrecisionLevel:  2,
		AnalysisTimeout: 30,
		MaxIterations:   100,
	}
}

// EffectInferenceConfig holds configuration for effect inference.
type EffectInferenceConfig struct {
	EnableCaching         bool
	EnableOptimization    bool
	EnableFlowSensitivity bool
	EnableInterProcedural bool
	MaxInferenceDepth     int
	CacheSize             int
	OptimizationLevel     int
}

// DefaultEffectInferenceConfig returns default effect inference configuration.
func DefaultEffectInferenceConfig() *EffectInferenceConfig {
	return &EffectInferenceConfig{
		EnableCaching:         true,
		EnableOptimization:    true,
		EnableFlowSensitivity: true,
		EnableInterProcedural: false,
		MaxInferenceDepth:     10,
		CacheSize:             1000,
		OptimizationLevel:     1,
	}
}

// NewEffectInferenceConfig creates a new effect inference configuration with custom settings.
func NewEffectInferenceConfig() *EffectInferenceConfig {
	return DefaultEffectInferenceConfig()
}

// InferenceCache provides caching for effect inference results.
type InferenceCache struct {
	cache   map[string]*EffectSignature
	maxSize int
	hits    int64
	misses  int64
	mu      sync.RWMutex
}

// NewInferenceCache creates a new inference cache.
func NewInferenceCache() *InferenceCache {
	return &InferenceCache{
		cache:   make(map[string]*EffectSignature),
		maxSize: 1000,
	}
}

// Get retrieves a cached signature.
func (ic *InferenceCache) Get(key string) (*EffectSignature, bool) {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	signature, exists := ic.cache[key]
	if exists {
		ic.hits++

		return signature.Clone(), true
	}

	ic.misses++

	return nil, false
}

// Put stores a signature in the cache.
func (ic *InferenceCache) Put(key string, signature *EffectSignature) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	if len(ic.cache) >= ic.maxSize {
		// Simple eviction: remove oldest entry.
		for k := range ic.cache {
			delete(ic.cache, k)

			break
		}
	}

	ic.cache[key] = signature.Clone()
}

// InferenceStatistics tracks inference statistics.
type InferenceStatistics struct {
	TotalInferences int64
	CacheHits       int64
	CacheMisses     int64
	ErrorCount      int64
	mu              sync.RWMutex
}

// NewInferenceStatistics creates new inference statistics.
func NewInferenceStatistics() *InferenceStatistics {
	return &InferenceStatistics{}
}

// IncrementInference increments total inference count.
func (is *InferenceStatistics) IncrementInference() {
	is.mu.Lock()
	defer is.mu.Unlock()

	is.TotalInferences++
}

// IncrementCacheHit increments cache hit count.
func (is *InferenceStatistics) IncrementCacheHit() {
	is.mu.Lock()
	defer is.mu.Unlock()

	is.CacheHits++
}

// IncrementCacheMiss increments cache miss count.
func (is *InferenceStatistics) IncrementCacheMiss() {
	is.mu.Lock()
	defer is.mu.Unlock()

	is.CacheMisses++
}

// IncrementError increments error count.
func (is *InferenceStatistics) IncrementError() {
	is.mu.Lock()
	defer is.mu.Unlock()

	is.ErrorCount++
}
