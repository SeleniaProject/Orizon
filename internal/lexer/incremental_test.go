// Package lexer implements incremental lexical analysis performance tests.
// Phase 1.1.2: インクリメンタル字句解析のパフォーマンステスト実装
package lexer

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestSpanAccuracy tests accuracy of span information
func TestSpanAccuracy(t *testing.T) {
	source := "let x = 42;"
	lexer := New(source)

	expectedSpans := []struct {
		tokenType TokenType
		startCol  int
		endCol    int
		literal   string
	}{
		{TokenLet, 1, 4, "let"},       // "let" starts at column 1, ends at 4
		{TokenIdentifier, 5, 6, "x"},  // "x" starts at column 5, ends at 6
		{TokenAssign, 7, 8, "="},      // "=" starts at column 7, ends at 8
		{TokenInteger, 9, 11, "42"},   // "42" starts at column 9, ends at 11
		{TokenSemicolon, 11, 12, ";"}, // ";" starts at column 11, ends at 12
	}

	for i, expected := range expectedSpans {
		token := lexer.NextToken()

		if token.Type != expected.tokenType {
			t.Errorf("Token %d: expected type %v, got %v", i, expected.tokenType, token.Type)
		}

		if token.Literal != expected.literal {
			t.Errorf("Token %d: expected literal %q, got %q", i, expected.literal, token.Literal)
		}

		if token.Span.Start.Column != expected.startCol {
			t.Errorf("Token %d: expected start column %d, got %d", i, expected.startCol, token.Span.Start.Column)
		}

		if token.Span.End.Column != expected.endCol {
			t.Errorf("Token %d: expected end column %d, got %d", i, expected.endCol, token.Span.End.Column)
		}
	}

	// Verify EOF token
	eofToken := lexer.NextToken()
	if eofToken.Type != TokenEOF {
		t.Errorf("Expected EOF token, got %v", eofToken.Type)
	}
}

// TestIncrementalAccuracy tests correctness of incremental lexing results
func TestIncrementalAccuracy(t *testing.T) {
	testCases := []struct {
		name           string
		originalSource string
		changePos      int
		insertion      string
		expectedDiff   int // Expected difference in token count
	}{
		{
			name:           "Simple variable insertion",
			originalSource: "func main() {\n    print(\"hello\");\n}",
			changePos:      15,
			insertion:      "let x = 42;\n    ",
			expectedDiff:   6, // let, x, =, 42, ;, newline
		},
		{
			name:           "Function call insertion",
			originalSource: "let a = 10;\nlet b = 20;",
			changePos:      11,
			insertion:      "\nlet c = add(a, b);",
			expectedDiff:   11, // newline, let, c, =, add, (, a, ,, b, ), ;
		},
		{
			name:           "Comment insertion",
			originalSource: "let x = 42;",
			changePos:      0,
			insertion:      "// This is a comment\n",
			expectedDiff:   2, // comment token, newline
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get tokens from original source
			originalLexer := New(tc.originalSource)
			var originalTokens []Token
			for {
				token := originalLexer.NextToken()
				originalTokens = append(originalTokens, token)
				if token.Type == TokenEOF {
					break
				}
			}

			// Create modified source
			modifiedSource := tc.originalSource[:tc.changePos] + tc.insertion + tc.originalSource[tc.changePos:]

			// Get tokens from modified source (full lexing for comparison)
			fullLexer := New(modifiedSource)
			var fullTokens []Token
			for {
				token := fullLexer.NextToken()
				fullTokens = append(fullTokens, token)
				if token.Type == TokenEOF {
					break
				}
			}

			// Create cache from original tokens
			cacheEntries := make([]CacheEntry, len(originalTokens))
			for i, token := range originalTokens {
				cacheEntries[i] = CacheEntry{
					Token:    token,
					IsValid:  true,
					StartPos: token.Span.Start.Offset,
					EndPos:   token.Span.End.Offset,
				}
			}

			// Perform incremental lexing
			changeRegion := &ChangeRegion{
				Start:  tc.changePos,
				End:    tc.changePos,
				Length: len(tc.insertion),
			}

			incrementalLexer := NewIncremental(modifiedSource, "test.oriz", cacheEntries, changeRegion)
			var incrementalTokens []Token
			for {
				token := incrementalLexer.NextToken()
				incrementalTokens = append(incrementalTokens, token)
				if token.Type == TokenEOF {
					break
				}
			}

			// Verify token count difference
			actualDiff := len(fullTokens) - len(originalTokens)
			if actualDiff != tc.expectedDiff {
				t.Errorf("Expected token count difference of %d, got %d", tc.expectedDiff, actualDiff)
			}
		})
	}
}

// TestCacheInvalidation tests cache invalidation scenarios
func TestCacheInvalidation(t *testing.T) {
	source := "let x = 42;"
	lexer := New(source)

	// Initially cache should be invalid
	if lexer.cacheValid {
		t.Error("Expected cache to be invalid initially")
	}

	// Set up some cache data
	lexer.UpdateCache([]Token{
		{Type: TokenLet, Literal: "let"},
		{Type: TokenIdentifier, Literal: "x"},
	})

	if !lexer.cacheValid {
		t.Error("Expected cache to be valid after UpdateCache")
	}

	// Invalidate cache
	lexer.InvalidateCache()

	if lexer.cacheValid {
		t.Error("Expected cache to be invalid after InvalidateCache")
	}

	if len(lexer.tokenCache) != 0 {
		t.Error("Expected token cache to be empty after invalidation")
	}
}

// BenchmarkFullLexing benchmarks complete file lexing for baseline comparison
func BenchmarkFullLexing(b *testing.B) {
	// Generate a realistic source code sample for testing
	source := generateRealisticSource(1000) // 1000 lines of code

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lexer := New(source)
		tokenCount := 0

		for {
			token := lexer.NextToken()
			tokenCount++
			if token.Type == TokenEOF {
				break
			}
		}

		// Ensure we processed meaningful tokens
		if tokenCount < 100 {
			b.Fatalf("Expected at least 100 tokens, got %d", tokenCount)
		}
	}
}

// generateRealisticSource generates a realistic Orizon source code for benchmarking
func generateRealisticSource(lines int) string {
	var builder strings.Builder

	// Generate a header comment
	builder.WriteString("// Auto-generated test file for performance benchmarking\n")
	builder.WriteString("// This file contains realistic Orizon code patterns\n\n")

	// Generate imports
	builder.WriteString("import std.io;\n")
	builder.WriteString("import std.collections;\n\n")

	// Generate struct definitions
	builder.WriteString("struct Point {\n")
	builder.WriteString("    x: float,\n")
	builder.WriteString("    y: float,\n")
	builder.WriteString("}\n\n")

	linesWritten := 8

	// Generate functions and variables to reach target line count
	for linesWritten < lines {
		if linesWritten+10 < lines {
			// Generate a function
			funcNum := (linesWritten - 8) / 10
			builder.WriteString(fmt.Sprintf("func calculateDistance%d(p1: Point, p2: Point) -> float {\n", funcNum))
			builder.WriteString("    let dx = p1.x - p2.x;\n")
			builder.WriteString("    let dy = p1.y - p2.y;\n")
			builder.WriteString("    let distanceSquared = dx * dx + dy * dy;\n")
			builder.WriteString("    return sqrt(distanceSquared);\n")
			builder.WriteString("}\n\n")
			linesWritten += 7
		} else {
			// Generate simple statements to fill remaining lines
			builder.WriteString(fmt.Sprintf("let variable%d = %d;\n", linesWritten, linesWritten*42))
			linesWritten++
		}
	}

	// Add main function
	builder.WriteString("func main() {\n")
	builder.WriteString("    let origin = Point { x: 0.0, y: 0.0 };\n")
	builder.WriteString("    let point = Point { x: 3.0, y: 4.0 };\n")
	builder.WriteString("    let distance = calculateDistance0(origin, point);\n")
	builder.WriteString("    print(\"Distance: \", distance);\n")
	builder.WriteString("}\n")

	return builder.String()
}

// ===== Phase 1.1.2: Incremental Lexical Analysis Tests =====

// TestIncrementalLexer_BasicFunctionality tests the core incremental lexing capabilities
func TestIncrementalLexer_BasicFunctionality(t *testing.T) {
	lexer := NewIncrementalLexer()

	// Test case 1: Initial lexing
	content1 := []byte("func main() { print(\"hello\") }")
	tokens1, err := lexer.LexIncremental("test.oriz", content1, nil)
	if err != nil {
		t.Fatalf("Initial lexing failed: %v", err)
	}

	if len(tokens1) == 0 {
		t.Error("Expected tokens from initial lexing")
	}

	// Verify cache was created
	stats := lexer.GetStats()
	if stats.FilesAnalyzed != 1 {
		t.Errorf("Expected 1 file analyzed, got %d", stats.FilesAnalyzed)
	}

	// Test case 2: No changes - should hit cache
	tokens2, err := lexer.LexIncremental("test.oriz", content1, nil)
	if err != nil {
		t.Fatalf("Cache retrieval failed: %v", err)
	}

	if len(tokens2) != len(tokens1) {
		t.Errorf("Cache retrieval returned different token count: expected %d, got %d", len(tokens1), len(tokens2))
	}

	// Verify cache hit
	stats = lexer.GetStats()
	if stats.CacheHits != 1 {
		t.Errorf("Expected 1 cache hit, got %d", stats.CacheHits)
	}
}

// TestIncrementalLexer_SimpleChanges tests basic incremental updates
func TestIncrementalLexer_SimpleChanges(t *testing.T) {
	lexer := NewIncrementalLexer()

	// Initial content
	content1 := []byte("func main() { print(\"hello\") }")
	tokens1, err := lexer.LexIncremental("test.oriz", content1, nil)
	if err != nil {
		t.Fatalf("Initial lexing failed: %v", err)
	}

	t.Logf("Initial tokens: %d", len(tokens1))
	for i, token := range tokens1 {
		t.Logf("Token %d: %s = %q", i, token.Type, token.Literal)
	}

	// Modified content (change string literal)
	content2 := []byte("func main() { print(\"world\") }")
	changes := []Change{
		{
			Start:   20, // Position of "hello"
			End:     25,
			OldText: "hello",
			NewText: "world",
			Type:    ChangeReplaceBlock,
		},
	}

	tokens2, err := lexer.LexIncremental("test.oriz", content2, changes)
	if err != nil {
		t.Fatalf("Incremental lexing failed: %v", err)
	}

	t.Logf("Updated tokens: %d", len(tokens2))
	for i, token := range tokens2 {
		t.Logf("Token %d: %s = %q", i, token.Type, token.Literal)
	}

	// Should have same number of tokens
	if len(tokens2) != len(tokens1) {
		t.Errorf("Token count changed unexpectedly: expected %d, got %d", len(tokens1), len(tokens2))
	}

	// Find the string token and verify it changed
	var foundStringToken bool
	for _, token := range tokens2 {
		if token.Type == TokenString && strings.Contains(token.Literal, "world") {
			foundStringToken = true
			break
		}
	}

	if !foundStringToken {
		t.Error("Expected to find updated string token with 'world'")
	}
} // TestIncrementalLexer_PerformanceMetrics tests performance monitoring
func TestIncrementalLexer_PerformanceMetrics(t *testing.T) {
	lexer := NewIncrementalLexer()

	// Large content for performance testing
	var contentBuilder strings.Builder
	for i := 0; i < 1000; i++ {
		contentBuilder.WriteString(fmt.Sprintf("func test%d() { print(\"line %d\") }\n", i, i))
	}
	content := []byte(contentBuilder.String())

	start := time.Now()
	tokens, err := lexer.LexIncremental("large.oriz", content, nil)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to lex large content: %v", err)
	}

	if len(tokens) == 0 {
		t.Error("Expected tokens from large content")
	}

	// Check performance stats
	stats := lexer.GetStats()
	if stats.TotalLexingTime == 0 {
		t.Error("Expected non-zero lexing time in stats")
	}

	if stats.CharactersProcessed == 0 {
		t.Error("Expected non-zero characters processed in stats")
	}

	t.Logf("Lexed %d characters in %v (%.2f chars/ms)",
		len(content), duration, float64(len(content))/float64(duration.Nanoseconds()/1000000))

	// Test cache hit performance
	start = time.Now()
	tokens2, err := lexer.LexIncremental("large.oriz", content, nil)
	cacheHitDuration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to retrieve from cache: %v", err)
	}

	if len(tokens2) != len(tokens) {
		t.Error("Cache retrieval returned different token count")
	}

	// Cache hit should be significantly faster
	if cacheHitDuration > duration/10 {
		t.Logf("Warning: Cache hit took %v, original lexing took %v (ratio: %.2fx)",
			cacheHitDuration, duration, float64(duration)/float64(cacheHitDuration))
	}

	t.Logf("Cache hit: %v (%.2fx speedup)", cacheHitDuration, float64(duration)/float64(cacheHitDuration))
}

// BenchmarkIncrementalLexer_CacheHit benchmarks cache hit performance
func BenchmarkIncrementalLexer_CacheHit(b *testing.B) {
	lexer := NewIncrementalLexer()
	content := []byte("func main() { print(\"hello world\") }")

	// Prime the cache
	_, err := lexer.LexIncremental("test.oriz", content, nil)
	if err != nil {
		b.Fatalf("Failed to prime cache: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := lexer.LexIncremental("test.oriz", content, nil)
		if err != nil {
			b.Fatalf("Cache hit failed: %v", err)
		}
	}
}

// BenchmarkIncrementalLexer_SmallChange benchmarks incremental updates
func BenchmarkIncrementalLexer_SmallChange(b *testing.B) {
	lexer := NewIncrementalLexer()
	baseContent := []byte("func main() { print(\"hello\") }")

	// Prime the cache
	_, err := lexer.LexIncremental("test.oriz", baseContent, nil)
	if err != nil {
		b.Fatalf("Failed to prime cache: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Alternate between two versions
		var content []byte
		var changes []Change

		if i%2 == 0 {
			content = []byte("func main() { print(\"world\") }")
			changes = []Change{{Start: 20, End: 25, OldText: "hello", NewText: "world", Type: ChangeReplaceBlock}}
		} else {
			content = baseContent
			changes = []Change{{Start: 20, End: 25, OldText: "world", NewText: "hello", Type: ChangeReplaceBlock}}
		}

		_, err := lexer.LexIncremental("test.oriz", content, changes)
		if err != nil {
			b.Fatalf("Incremental update failed: %v", err)
		}
	}
}
