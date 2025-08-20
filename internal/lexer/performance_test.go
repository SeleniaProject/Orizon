// Package lexer provides performance benchmarks for incremental lexical analysis.
// Phase 1.1.2: インクリメンタル字句解析パフォーマンステスト実装
package lexer

import (
	"fmt"
	"strings"
	"testing"
)

// Performance test data generation utilities.

// generateRealisticOrizonCode creates realistic Orizon source code for performance testing.
func generateRealisticOrizonCode(functions int, linesPerFunction int) string {
	var builder strings.Builder

	// Add imports.
	builder.WriteString("import \"std::io\";\n")
	builder.WriteString("import \"std::math\";\n")
	builder.WriteString("import \"std::collections\";\n\n")

	// Add type definitions.
	builder.WriteString("struct Point {\n")
	builder.WriteString("    x: f64,\n")
	builder.WriteString("    y: f64,\n")
	builder.WriteString("}\n\n")

	builder.WriteString("enum Color {\n")
	builder.WriteString("    Red,\n")
	builder.WriteString("    Green(u8),\n")
	builder.WriteString("    Blue { intensity: f32 },\n")
	builder.WriteString("}\n\n")

	// Generate functions with varied complexity.
	for i := 0; i < functions; i++ {
		// Function signature.
		builder.WriteString(fmt.Sprintf("func calculate_distance_%d(p1: Point, p2: Point) -> f64 {\n", i))

		// Function body with realistic code patterns.
		for j := 0; j < linesPerFunction; j++ {
			switch j % 8 {
			case 0:
				builder.WriteString(fmt.Sprintf("    let dx_%d = p2.x - p1.x;\n", j))
			case 1:
				builder.WriteString(fmt.Sprintf("    let dy_%d = p2.y - p1.y;\n", j))
			case 2:
				builder.WriteString(fmt.Sprintf("    let squared_%d = dx_%d * dx_%d + dy_%d * dy_%d;\n", j, j-2, j-2, j-1, j-1))
			case 3:
				builder.WriteString(fmt.Sprintf("    if squared_%d > 0.0 {\n", j-1))
			case 4:
				builder.WriteString(fmt.Sprintf("        print(\"Calculating distance for function %d\");\n", i))
			case 5:
				builder.WriteString("    }\n")
			case 6:
				builder.WriteString(fmt.Sprintf("    let result_%d = math::sqrt(squared_%d);\n", j, j-3))
			case 7:
				builder.WriteString(fmt.Sprintf("    // Comment line %d in function %d\n", j, i))
			}
		}

		builder.WriteString("    return result_6;\n")
		builder.WriteString("}\n\n")
	}

	// Add main function.
	builder.WriteString("func main() {\n")
	builder.WriteString("    let origin = Point { x: 0.0, y: 0.0 };\n")

	for i := 0; i < min(10, functions); i++ {
		builder.WriteString(fmt.Sprintf("    let point_%d = Point { x: %d.0, y: %d.0 };\n", i, i*3, i*4))
		builder.WriteString(fmt.Sprintf("    let distance_%d = calculate_distance_%d(origin, point_%d);\n", i, i, i))
		builder.WriteString(fmt.Sprintf("    print(\"Distance %d: \", distance_%d);\n", i, i))
	}

	builder.WriteString("}\n")

	return builder.String()
}

// Utility function for min.
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// Benchmarks for incremental lexer performance.

// BenchmarkIncrementalLexer_RealisticFile tests performance with realistic file sizes.
func BenchmarkIncrementalLexer_RealisticFile(b *testing.B) {
	lexer := NewIncrementalLexer()
	// Generate more realistic content (about 10KB).
	content := []byte(generateRealisticOrizonCode(100, 25)) // ~2500 lines

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		filename := fmt.Sprintf("realistic_%d.oriz", i%5)

		_, err := lexer.LexIncremental(filename, content, nil)
		if err != nil {
			b.Fatalf("Failed to lex realistic file: %v", err)
		}
	}
}

// BenchmarkIncrementalLexer_RealisticCacheHit tests performance on realistic cache hits.
func BenchmarkIncrementalLexer_RealisticCacheHit(b *testing.B) {
	lexer := NewIncrementalLexer()
	// Use more realistic content size (about 1KB).
	content := []byte(generateRealisticOrizonCode(10, 15)) // ~150 lines

	// Prime the cache.
	_, err := lexer.LexIncremental("test.oriz", content, nil)
	if err != nil {
		b.Fatalf("Failed to prime cache: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := lexer.LexIncremental("test.oriz", content, nil)
		if err != nil {
			b.Fatalf("Cache hit failed: %v", err)
		}
	}
} // BenchmarkIncrementalLexer_MediumFile tests performance on medium-sized files
func BenchmarkIncrementalLexer_MediumFile(b *testing.B) {
	lexer := NewIncrementalLexer()
	content := []byte(generateRealisticOrizonCode(50, 20)) // ~1000 lines

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		filename := fmt.Sprintf("medium_%d.oriz", i%10)

		_, err := lexer.LexIncremental(filename, content, nil)
		if err != nil {
			b.Fatalf("Failed to lex medium file: %v", err)
		}
	}
}

// BenchmarkIncrementalLexer_LargeFile tests performance on large files.
func BenchmarkIncrementalLexer_LargeFile(b *testing.B) {
	lexer := NewIncrementalLexer()
	content := []byte(generateRealisticOrizonCode(500, 50)) // ~25000 lines

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		filename := fmt.Sprintf("large_%d.oriz", i%3)

		_, err := lexer.LexIncremental(filename, content, nil)
		if err != nil {
			b.Fatalf("Failed to lex large file: %v", err)
		}
	}
}

// BenchmarkIncrementalLexer_CacheHitRatio tests cache effectiveness.
func BenchmarkIncrementalLexer_CacheHitRatio(b *testing.B) {
	lexer := NewIncrementalLexer()
	content := []byte(generateRealisticOrizonCode(20, 15))

	// Prime cache with multiple files.
	for i := 0; i < 10; i++ {
		filename := fmt.Sprintf("cached_%d.oriz", i)

		_, err := lexer.LexIncremental(filename, content, nil)
		if err != nil {
			b.Fatalf("Failed to prime cache: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 80% cache hits, 20% new files.
		filename := fmt.Sprintf("cached_%d.oriz", i%12) // Files 0-9 exist, 10-11 are new

		_, err := lexer.LexIncremental(filename, content, nil)
		if err != nil {
			b.Fatalf("Cache hit test failed: %v", err)
		}
	}
}

// BenchmarkIncrementalLexer_SmallChanges tests incremental update performance.
func BenchmarkIncrementalLexer_SmallChanges(b *testing.B) {
	lexer := NewIncrementalLexer()
	baseContent := generateRealisticOrizonCode(10, 10)
	content := []byte(baseContent)

	// Prime the cache.
	_, err := lexer.LexIncremental("test.oriz", content, nil)
	if err != nil {
		b.Fatalf("Failed to prime cache: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate typing by changing small parts.
		modifiedContent := strings.Replace(baseContent,
			fmt.Sprintf("calculate_distance_%d", i%10),
			fmt.Sprintf("calculate_distance_%d_modified", i%10), 1)

		changes := []Change{
			{
				Start:   strings.Index(baseContent, fmt.Sprintf("calculate_distance_%d", i%10)),
				End:     strings.Index(baseContent, fmt.Sprintf("calculate_distance_%d", i%10)) + 20,
				OldText: fmt.Sprintf("calculate_distance_%d", i%10),
				NewText: fmt.Sprintf("calculate_distance_%d_modified", i%10),
				Type:    ChangeReplaceBlock,
			},
		}

		_, err := lexer.LexIncremental("test.oriz", []byte(modifiedContent), changes)
		if err != nil {
			b.Fatalf("Incremental change failed: %v", err)
		}
	}
}

// BenchmarkIncrementalLexer_LineInsertion tests line insertion performance.
func BenchmarkIncrementalLexer_LineInsertion(b *testing.B) {
	lexer := NewIncrementalLexer()
	baseContent := generateRealisticOrizonCode(20, 10)
	content := []byte(baseContent)

	// Prime the cache.
	_, err := lexer.LexIncremental("test.oriz", content, nil)
	if err != nil {
		b.Fatalf("Failed to prime cache: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Insert a new line in the middle.
		insertPos := len(baseContent) / 2
		newLine := fmt.Sprintf("\n    let new_var_%d = %d;\n", i, i)
		modifiedContent := baseContent[:insertPos] + newLine + baseContent[insertPos:]

		changes := []Change{
			{
				Start:   insertPos,
				End:     insertPos,
				OldText: "",
				NewText: newLine,
				Type:    ChangeInsertLine,
			},
		}

		_, err := lexer.LexIncremental("test.oriz", []byte(modifiedContent), changes)
		if err != nil {
			b.Fatalf("Line insertion failed: %v", err)
		}
	}
}

// BenchmarkIncrementalLexer_ConcurrentAccess tests thread safety performance.
func BenchmarkIncrementalLexer_ConcurrentAccess(b *testing.B) {
	lexer := NewIncrementalLexer()
	content := []byte(generateRealisticOrizonCode(10, 8))

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		fileID := 0
		for pb.Next() {
			filename := fmt.Sprintf("concurrent_%d.oriz", fileID%20)

			_, err := lexer.LexIncremental(filename, content, nil)
			if err != nil {
				b.Fatalf("Concurrent access failed: %v", err)
			}

			fileID++
		}
	})
}

// BenchmarkCompareLexer_IncrementalVsStandard compares incremental vs standard lexer.
func BenchmarkCompareLexer_IncrementalVsStandard(b *testing.B) {
	content := generateRealisticOrizonCode(30, 15)

	b.Run("Standard", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			lexer := NewWithFilename(content, "test.oriz")
			tokenCount := 0

			for {
				token := lexer.NextToken()
				if token.Type == TokenEOF {
					break
				}

				tokenCount++
			}
		}
	})

	b.Run("Incremental_CacheMiss", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			lexer := NewIncrementalLexer()
			filename := fmt.Sprintf("test_%d.oriz", i) // Always cache miss

			_, err := lexer.LexIncremental(filename, []byte(content), nil)
			if err != nil {
				b.Fatalf("Incremental lexer failed: %v", err)
			}
		}
	})

	b.Run("Incremental_CacheHit", func(b *testing.B) {
		lexer := NewIncrementalLexer()
		filename := "test.oriz"

		// Prime the cache.
		_, err := lexer.LexIncremental(filename, []byte(content), nil)
		if err != nil {
			b.Fatalf("Failed to prime cache: %v", err)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := lexer.LexIncremental(filename, []byte(content), nil)
			if err != nil {
				b.Fatalf("Cache hit failed: %v", err)
			}
		}
	})
}

// BenchmarkIncrementalLexer_MemoryUsage tests memory efficiency.
func BenchmarkIncrementalLexer_MemoryUsage(b *testing.B) {
	lexer := NewIncrementalLexer()

	// Test with multiple files to simulate real IDE usage.
	files := make([][]byte, 100)
	for i := 0; i < 100; i++ {
		files[i] = []byte(generateRealisticOrizonCode(5+i%20, 8+i%15))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		fileIdx := i % len(files)
		filename := fmt.Sprintf("memory_test_%d.oriz", fileIdx)

		_, err := lexer.LexIncremental(filename, files[fileIdx], nil)
		if err != nil {
			b.Fatalf("Memory test failed: %v", err)
		}

		// Periodically clear cache to test memory management.
		if i%1000 == 999 {
			lexer.ClearCache()
		}
	}
}
