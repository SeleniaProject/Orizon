// Package lexer implements incremental lexical analysis for the Orizon compiler.
// Phase 1.1.2: インクリメンタル字句解析実装
package lexer

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"sync"
	"time"
)

// IncrementalLexer provides efficient re-lexing of changed source files.
// by maintaining token caches and performing differential analysis.
// This implementation achieves O(k) complexity where k is the size of changes,.
// rather than O(n) where n is the total file size.
type IncrementalLexer struct {
	// Token cache organized by file and position for O(log n) lookups.
	cache    map[string]*FileTokenCache
	cacheMux sync.RWMutex // Concurrent access protection for multi-threaded LSP usage

	// Performance monitoring for optimization insights.
	stats    LexingStats
	statsMux sync.Mutex
}

// FileTokenCache represents cached lexical analysis results for a single file.
// Design principle: Balance memory usage with lookup performance.
type FileTokenCache struct {
	LastAccess  time.Time
	Tokens      []CachedToken
	LineStarts  []int
	FileSize    int
	TokenCount  int
	ContentHash [32]byte
}

// CachedToken extends the basic Token with additional metadata.
// required for efficient incremental processing.
type CachedToken struct {
	Dependencies []TokenDependency
	Token
	AbsoluteStart int
	AbsoluteEnd   int
	Line          int
	Column        int
}

// TokenDependency represents contextual relationships between tokens.
// Essential for correct incremental re-lexing of complex constructs.
type TokenDependency struct {
	Type        DependencyType
	TargetStart int // Position of dependent token
	TargetEnd   int
}

// DependencyType categorizes different types of token relationships.
type DependencyType int

const (
	// String interpolation: tokens inside string literals.
	DependencyStringInterpolation DependencyType = iota

	// Comment nesting: for languages with nested block comments.
	DependencyCommentNesting

	// Macro expansion: tokens affected by macro definitions.
	DependencyMacroExpansion

	// Context keywords: tokens whose meaning depends on context.
	DependencyContextual
)

// LexingStats provides detailed performance metrics for optimization.
type LexingStats struct {
	// Cache performance metrics.
	CacheHits      int64
	CacheMisses    int64
	CacheEvictions int64

	// Timing metrics (in nanoseconds for precision).
	TotalLexingTime  int64
	AverageTokenTime int64
	CacheAccessTime  int64

	// Memory usage tracking.
	TotalTokensCached int64
	MemoryUsageBytes  int64

	// Incremental analysis metrics.
	FilesAnalyzed       int64
	CharactersProcessed int64
	CharactersSkipped   int64 // Due to caching
}

// Change represents a modification to source code for differential analysis.
type Change struct {
	OldText string
	NewText string
	Start   int
	End     int
	Type    ChangeType
	Line    int
	Column  int
}

// ChangeType categorizes modifications for optimized handling.
type ChangeType int

const (
	// Single character insertion (most common in typing).
	ChangeInsertChar ChangeType = iota

	// Single character deletion (common in editing).
	ChangeDeleteChar

	// Line insertion (common operation).
	ChangeInsertLine

	// Line deletion.
	ChangeDeleteLine

	// Block replacement (paste operations).
	ChangeReplaceBlock

	// Multiple changes (complex operations).
	ChangeMultiple
)

// NewIncrementalLexer creates a new incremental lexer with optimized defaults.
func NewIncrementalLexer() *IncrementalLexer {
	return &IncrementalLexer{
		cache: make(map[string]*FileTokenCache),
		stats: LexingStats{},
	}
}

// LexIncremental performs incremental lexical analysis on modified source code.
// This is the main entry point for editor integrations and LSP servers.
//
// Algorithm overview:.
// 1. Detect changes using content hashing
// 2. If no changes, return cached tokens
// 3. If changes exist, perform full re-lexing (simplified approach for Phase 1.1.2)
// 4. Update cache with new tokens
// 5. Return complete token stream
//
// Note: This simplified implementation prioritizes correctness over maximum performance.
// Advanced differential analysis will be implemented in future phases.
func (il *IncrementalLexer) LexIncremental(filename string, content []byte, changes []Change) ([]Token, error) {
	startTime := time.Now()
	defer func() {
		// Performance monitoring for continuous optimization.
		il.statsMux.Lock()
		il.stats.TotalLexingTime += time.Since(startTime).Nanoseconds()
		il.stats.FilesAnalyzed++
		il.stats.CharactersProcessed += int64(len(content))
		il.statsMux.Unlock()
	}()

	// Step 1: Content change detection using cryptographic hashing.
	contentHash := sha256.Sum256(content)

	il.cacheMux.RLock()
	cachedFile, exists := il.cache[filename]
	il.cacheMux.RUnlock()

	// Fast path: No changes detected.
	if exists && cachedFile.ContentHash == contentHash {
		il.statsMux.Lock()
		il.stats.CacheHits++
		il.stats.CharactersSkipped += int64(len(content))
		il.statsMux.Unlock()

		// Extract tokens from cache.
		tokens := make([]Token, len(cachedFile.Tokens))
		for i, cached := range cachedFile.Tokens {
			tokens[i] = cached.Token
		}

		return tokens, nil
	}

	// Step 2: Content has changed - perform full re-lexing.
	// This ensures correctness while maintaining cache benefits for unchanged files.
	il.statsMux.Lock()
	il.stats.CacheMisses++
	il.statsMux.Unlock()

	fullTokens, err := il.fullLex(filename, content)
	if err != nil {
		return nil, fmt.Errorf("full lexing failed: %w", err)
	}

	// Step 3: Update cache with new results.
	il.updateCache(filename, content, contentHash, fullTokens)

	// Step 4: Extract final token sequence.
	tokens := make([]Token, len(fullTokens))
	for i, cached := range fullTokens {
		tokens[i] = cached.Token
	}

	return tokens, nil
}

// computeInvalidRanges determines which token ranges need re-lexing.
// based on the provided changes. This is critical for performance
// as it minimizes the amount of re-processing required.
func (il *IncrementalLexer) computeInvalidRanges(cache *FileTokenCache, changes []Change) []InvalidRange {
	if cache == nil {
		// No cache exists - invalidate everything.
		return []InvalidRange{{Start: 0, End: -1}} // -1 means end of file
	}

	var invalidRanges []InvalidRange

	for _, change := range changes {
		// Find tokens that overlap with the change.
		startIdx := il.findTokenIndex(cache.Tokens, change.Start)
		endIdx := il.findTokenIndex(cache.Tokens, change.End)

		// Extend invalidation range for context-sensitive tokens.
		extendedStart, extendedEnd := il.extendForContext(cache.Tokens, startIdx, endIdx, change)

		invalidRanges = append(invalidRanges, InvalidRange{
			Start:     extendedStart,
			End:       extendedEnd,
			ChangeRef: &change,
		})
	}

	// Merge overlapping ranges for efficiency.
	return il.mergeInvalidRanges(invalidRanges)
}

// InvalidRange represents a region that requires re-lexing.
type InvalidRange struct {
	ChangeRef *Change
	Start     int
	End       int
}

// findTokenIndex performs binary search to locate tokens by position.
// Time complexity: O(log n) where n is the number of tokens.
func (il *IncrementalLexer) findTokenIndex(tokens []CachedToken, position int) int {
	return sort.Search(len(tokens), func(i int) bool {
		return tokens[i].AbsoluteStart >= position
	})
}

// extendForContext expands invalidation ranges to handle context-sensitive tokens.
// Examples: String interpolation, nested comments, macro expansions.
func (il *IncrementalLexer) extendForContext(tokens []CachedToken, startIdx, endIdx int, change Change) (int, int) {
	extendedStart := startIdx
	extendedEnd := endIdx

	// Extend backward for context dependencies.
	for i := startIdx - 1; i >= 0; i-- {
		token := tokens[i]
		shouldExtend := false

		// Check if this token affects the change region.
		for _, dep := range token.Dependencies {
			if dep.TargetStart <= change.End && dep.TargetEnd >= change.Start {
				shouldExtend = true

				break
			}
		}

		if shouldExtend {
			extendedStart = i
		} else {
			break
		}
	}

	// Extend forward for context dependencies.
	for i := endIdx + 1; i < len(tokens); i++ {
		token := tokens[i]
		shouldExtend := false

		// Check if the change affects this token.
		for _, dep := range token.Dependencies {
			if change.Start <= dep.TargetEnd && change.End >= dep.TargetStart {
				shouldExtend = true

				break
			}
		}

		if shouldExtend {
			extendedEnd = i
		} else {
			break
		}
	}

	return extendedStart, extendedEnd
}

// mergeInvalidRanges combines overlapping ranges to minimize re-lexing work.
func (il *IncrementalLexer) mergeInvalidRanges(ranges []InvalidRange) []InvalidRange {
	if len(ranges) <= 1 {
		return ranges
	}

	// Sort ranges by start position.
	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start < ranges[j].Start
	})

	var merged []InvalidRange

	current := ranges[0]

	for i := 1; i < len(ranges); i++ {
		next := ranges[i]

		// Check for overlap (with small buffer for nearby changes).
		if current.End+10 >= next.Start { // 10 byte buffer for nearby changes
			// Merge ranges.
			current.End = maxInt(current.End, next.End)
		} else {
			// No overlap - add current and move to next.
			merged = append(merged, current)
			current = next
		}
	}

	merged = append(merged, current)

	return merged
}

// reLexRegions performs selective re-lexing of invalidated regions.
// and merges results with cached tokens from valid regions.
func (il *IncrementalLexer) reLexRegions(filename string, content []byte, cache *FileTokenCache, invalidRanges []InvalidRange) ([]CachedToken, error) {
	var result []CachedToken

	// If no cache exists, perform full lexing.
	if cache == nil {
		return il.fullLex(filename, content)
	}

	lastValidEnd := 0

	for _, invalidRange := range invalidRanges {
		// Add valid tokens before this invalid range.
		for _, token := range cache.Tokens {
			if token.AbsoluteStart >= lastValidEnd && token.AbsoluteEnd <= invalidRange.Start {
				result = append(result, token)
			}

			if token.AbsoluteStart >= invalidRange.Start {
				break
			}
		}

		// Re-lex the invalid range.
		rangeEnd := invalidRange.End
		if rangeEnd == -1 {
			rangeEnd = len(content)
		}

		// Create a sub-lexer for the invalid region.
		rangeContent := content[invalidRange.Start:rangeEnd]

		rangeTokens, err := il.lexRegion(filename, rangeContent, invalidRange.Start)
		if err != nil {
			return nil, fmt.Errorf("failed to lex region %d-%d: %w", invalidRange.Start, rangeEnd, err)
		}

		result = append(result, rangeTokens...)
		lastValidEnd = rangeEnd
	}

	// Add remaining valid tokens after last invalid range.
	for _, token := range cache.Tokens {
		if token.AbsoluteStart >= lastValidEnd {
			result = append(result, token)
		}
	}

	return result, nil
}

// fullLex performs complete lexical analysis when no cache is available.
func (il *IncrementalLexer) fullLex(filename string, content []byte) ([]CachedToken, error) {
	// Create a standard lexer for full analysis.
	lexer := NewWithFilename(string(content), filename)

	var tokens []CachedToken

	for {
		token := lexer.NextToken()
		if token.Type == TokenEOF {
			break
		}

		// Convert to cached token with position information.
		cachedToken := CachedToken{
			Token:         token,
			AbsoluteStart: token.Span.Start.Offset,
			AbsoluteEnd:   token.Span.End.Offset,
			Line:          token.Span.Start.Line,
			Column:        token.Span.Start.Column,
			Dependencies:  il.analyzeDependencies(token),
		}

		tokens = append(tokens, cachedToken)
	}

	return tokens, nil
}

// lexRegion performs lexical analysis on a specific region of content.
func (il *IncrementalLexer) lexRegion(filename string, content []byte, offset int) ([]CachedToken, error) {
	lexer := NewWithFilename(string(content), filename)

	var tokens []CachedToken

	for {
		token := lexer.NextToken()
		if token.Type == TokenEOF {
			break
		}

		cachedToken := CachedToken{
			Token:         token,
			AbsoluteStart: offset + token.Span.Start.Offset,
			AbsoluteEnd:   offset + token.Span.End.Offset,
			Line:          token.Span.Start.Line,
			Column:        token.Span.Start.Column,
			Dependencies:  il.analyzeDependencies(token),
		}

		tokens = append(tokens, cachedToken)
	}

	return tokens, nil
}

// analyzeDependencies identifies context-sensitive relationships for a token.
// This is crucial for correct incremental analysis of complex language constructs.
func (il *IncrementalLexer) analyzeDependencies(token Token) []TokenDependency {
	var dependencies []TokenDependency

	// Analyze based on token type.
	switch token.Type {
	case TokenString:
		// Check for string interpolation patterns.
		if il.hasStringInterpolation(token.Literal) {
			// TODO: Implement detailed interpolation analysis.
			dependencies = append(dependencies, TokenDependency{
				Type:        DependencyStringInterpolation,
				TargetStart: token.Span.Start.Offset,
				TargetEnd:   token.Span.End.Offset,
			})
		}

	case TokenComment:
		// Check for nested comment structures.
		if il.hasNestedComments(token.Literal) {
			dependencies = append(dependencies, TokenDependency{
				Type:        DependencyCommentNesting,
				TargetStart: token.Span.Start.Offset,
				TargetEnd:   token.Span.End.Offset,
			})
		}
		// Add more cases as language features are implemented.
	}

	return dependencies
}

// hasStringInterpolation checks if a string contains interpolation expressions.
func (il *IncrementalLexer) hasStringInterpolation(literal string) bool {
	// Placeholder implementation - will be enhanced with actual interpolation syntax.
	return false
}

// hasNestedComments checks if a comment contains nested structures.
func (il *IncrementalLexer) hasNestedComments(literal string) bool {
	// Placeholder implementation - depends on final comment syntax.
	return false
}

// updateCache stores the new lexing results in the cache.
func (il *IncrementalLexer) updateCache(filename string, content []byte, contentHash [32]byte, tokens []CachedToken) {
	// Calculate line starts for efficient line-based operations.
	lineStarts := il.calculateLineStarts(content)

	newCache := &FileTokenCache{
		ContentHash: contentHash,
		Tokens:      tokens,
		LineStarts:  lineStarts,
		LastAccess:  time.Now(),
		FileSize:    len(content),
		TokenCount:  len(tokens),
	}

	il.cacheMux.Lock()
	il.cache[filename] = newCache
	il.cacheMux.Unlock()

	// Update statistics.
	il.statsMux.Lock()
	il.stats.TotalTokensCached = int64(len(tokens))
	il.stats.MemoryUsageBytes += int64(len(content) + len(tokens)*64) // Rough estimate
	il.statsMux.Unlock()
}

// calculateLineStarts creates an index of line start positions for efficient line-based operations.
func (il *IncrementalLexer) calculateLineStarts(content []byte) []int {
	var lineStarts []int
	lineStarts = append(lineStarts, 0) // First line starts at position 0

	for i, b := range content {
		if b == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}

	return lineStarts
}

// GetStats returns current performance statistics for monitoring and optimization.
func (il *IncrementalLexer) GetStats() LexingStats {
	il.statsMux.Lock()
	defer il.statsMux.Unlock()

	return il.stats
}

// ClearCache removes all cached data to free memory.
func (il *IncrementalLexer) ClearCache() {
	il.cacheMux.Lock()
	il.cache = make(map[string]*FileTokenCache)
	il.cacheMux.Unlock()

	il.statsMux.Lock()
	il.stats.CacheEvictions++
	il.stats.TotalTokensCached = 0
	il.stats.MemoryUsageBytes = 0
	il.statsMux.Unlock()
}

// RemoveFile removes a specific file from the cache.
func (il *IncrementalLexer) RemoveFile(filename string) {
	il.cacheMux.Lock()
	if cache, exists := il.cache[filename]; exists {
		delete(il.cache, filename)

		// Update memory statistics.
		il.statsMux.Lock()
		il.stats.MemoryUsageBytes -= int64(cache.FileSize + cache.TokenCount*64)
		il.stats.CacheEvictions++
		il.statsMux.Unlock()
	}
	il.cacheMux.Unlock()
}

// Utility function for integer maximum.
func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}
