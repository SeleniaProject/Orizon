package main

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/orizon-lang/orizon/internal/allocator"
	"github.com/orizon-lang/orizon/internal/lexer"
	"github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/types"
)

// メモリ使用量を測定するヘルパー関数
func getMemoryUsage() (alloc, sys uint64) {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	return m.Alloc, m.Sys
}

// 型システムのパフォーマンスベンチマーク
func BenchmarkTypeSystem(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 複雑な型を作成して比較
		intType := types.TypeInt64
		ptrType := types.NewPointerType(intType, false)
		arrayType := types.NewArrayType(ptrType, 100)

		// 型比較のベンチマーク
		_ = arrayType.Equals(arrayType)
		_ = arrayType.CanConvertTo(ptrType)

		// 型レジストリの操作
		registry := types.NewTypeRegistry()
		registry.RegisterType("test_type", arrayType)
	}
}

// パーサーのパフォーマンスベンチマーク
func BenchmarkParser(b *testing.B) {
	source := `
	fn fibonacci(n: int64) -> int64 {
		if n <= 1 {
			return n
		}
		return fibonacci(n-1) + fibonacci(n-2)
	}

	fn main() {
		let result = fibonacci(35)
		print(result)
	}
	`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		l := lexer.NewWithFilename(source, "benchmark.oriz")
		p := parser.NewParser(l, "benchmark.oriz")

		_, errors := p.Parse()
		if len(errors) > 0 {
			b.Fatal("Parse errors:", errors)
		}
	}
}

// メモリリークテスト
func TestMemoryLeak(t *testing.T) {
	initialAlloc, initialSys := getMemoryUsage()

	// 多数の型を作成
	for i := 0; i < 1000; i++ { // 数を減らしてより正確な測定
		registry := types.NewTypeRegistry()
		intType := types.TypeInt64
		for j := 0; j < 10; j++ { // 数を減らしてより正確な測定
			ptrType := types.NewPointerType(intType, false)
			registry.RegisterType("test_type_"+string(rune(j)), ptrType)
		}
		runtime.GC()
	}

	finalAlloc, finalSys := getMemoryUsage()

	allocGrowth := float64(finalAlloc-initialAlloc) / float64(initialAlloc)
	sysGrowth := float64(finalSys-initialSys) / float64(initialSys)

	t.Logf("Memory usage - Initial: Alloc=%d, Sys=%d", initialAlloc, initialSys)
	t.Logf("Memory usage - Final: Alloc=%d, Sys=%d", finalAlloc, finalSys)
	t.Logf("Memory growth: Alloc=%.2f%%, Sys=%.2f%%", allocGrowth*100, sysGrowth*100)

	if allocGrowth > 0.5 { // 50%以上の増加はリークの兆候
		t.Errorf("Memory leak detected: Alloc growth: %.2f%%, Sys growth: %.2f%%", allocGrowth*100, sysGrowth*100)
	}
}

// 字句解析器のパフォーマンスベンチマーク
func BenchmarkLexer(b *testing.B) {
	// 単純なソースコードでテスト
	source := "fn main() { let x = 1 + 2 }"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		l := lexer.New(source)

		tokenCount := 0
		for {
			token := l.NextToken()
			tokenCount++
			if token.Type == lexer.TokenEOF {
				break
			}
			if token.Type == lexer.TokenError {
				b.Fatal("Lexer error:", token.Literal)
			}
		}

		// ベンチマーク結果にトークン数を報告
		if i == 0 {
			b.ReportMetric(float64(tokenCount), "tokens/op")
		}
	}
}

// 字句解析器のメモリ使用量ベンチマーク
func BenchmarkLexerMemory(b *testing.B) {
	source := `
	fn fibonacci(n: int64) -> int64 {
		if n <= 1 {
			return n
		}
		return fibonacci(n-1) + fibonacci(n-2)
	}

	fn main() {
		let result = fibonacci(35)
		print(result)
	}
	`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		l := lexer.NewWithFilename(source, "benchmark.oriz")

		for {
			token := l.NextToken()
			if token.Type == lexer.TokenEOF {
				break
			}
		}
	}
}

// アロケータのパフォーマンスベンチマーク
func BenchmarkOptimizedAllocator(b *testing.B) {
	config := &allocator.Config{
		EnableTracking: false, // Disable tracking for performance
		AlignmentSize:  8,
		MemoryLimit:    0,
	}

	alloc := allocator.NewOptimizedAllocator(config)
	defer alloc.Reset()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Test different allocation sizes
		sizes := []uintptr{32, 64, 128, 256, 512}

		for _, size := range sizes {
			ptr := alloc.Alloc(size)
			if ptr != nil {
				alloc.Free(ptr)
			}
		}
	}
}

// システムアロケータとの比較ベンチマーク
func BenchmarkSystemAllocator(b *testing.B) {
	config := &allocator.Config{
		EnableTracking: false,
		AlignmentSize:  8,
		MemoryLimit:    0,
	}

	alloc := allocator.NewSystemAllocator(config)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sizes := []uintptr{32, 64, 128, 256, 512}

		for _, size := range sizes {
			ptr := alloc.Alloc(size)
			if ptr != nil {
				alloc.Free(ptr)
			}
		}
	}
}

// CPUプロファイリング用のベンチマーク
func BenchmarkCPUIntensive(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// 型推論のシミュレーション
		registry := types.NewTypeRegistry()

		// 大規模な構造体の作成
		fields := make([]types.StructField, 100)
		for j := range fields {
			fields[j] = types.StructField{
				Name: "field" + string(rune(j)),
				Type: types.TypeInt64,
			}
		}

		structType := types.NewStructType("LargeStruct", fields)

		// 複雑な型比較
		for k := 0; k < 10; k++ {
			_ = structType.Equals(structType)
		}

		// レジストリ操作
		registry.RegisterType("large_struct", structType)
	}
}

// 包括的なシステムベンチマーク
func BenchmarkSystemPerformance(b *testing.B) {
	// コンパイラ全体のパフォーマンス
	source := `
	fn fibonacci(n: int64) -> int64 {
		if n <= 1 {
			return n
		}
		return fibonacci(n-1) + fibonacci(n-2)
	}

	fn main() {
		let result = fibonacci(30)
		print(result)
	}
	`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 字句解析
		l := lexer.New(source)
		tokens := []lexer.Token{}
		for {
			token := l.NextToken()
			tokens = append(tokens, token)
			if token.Type == lexer.TokenEOF {
				break
			}
		}

		// パーサー処理
		l2 := lexer.New(source)
		p := parser.NewParser(l2, "benchmark.oriz")
		_, errors := p.Parse()
		if len(errors) > 0 {
			b.Fatal("Parse errors:", errors)
		}
	}
}

// メモリ使用量の包括的ベンチマーク
func BenchmarkMemoryEfficiency(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 複数のメモリ割り当てパターンをテスト
		sources := []string{
			"fn test1() { let x = 1 }",
			"fn test2() { let x = 1 + 2 * 3 }",
			"fn test3(a: int64, b: int64) { return a + b }",
			"struct Test { field1: int64, field2: int64 }",
		}

		for _, source := range sources {
			l := lexer.New(source)
			for {
				token := l.NextToken()
				if token.Type == lexer.TokenEOF {
					break
				}
			}
		}
	}
}

// 並行処理ベンチマーク
func BenchmarkConcurrentPerformance(b *testing.B) {
	b.ResetTimer()

	// 並行字句解析のテスト
	sources := make([]string, 100)
	for i := range sources {
		sources[i] = fmt.Sprintf("fn func%d() { let x = %d }", i, i)
	}

	b.RunParallel(func(pb *testing.PB) {
		localSources := sources
		i := 0
		for pb.Next() {
			source := localSources[i%len(localSources)]
			l := lexer.New(source)
			for {
				token := l.NextToken()
				if token.Type == lexer.TokenEOF {
					break
				}
			}
			i++
		}
	})
}
