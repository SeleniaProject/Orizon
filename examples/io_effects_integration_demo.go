// Phase 2.5.3 Effect Type System Integration Demo
// 副作用・例外・I/O効果の統合デモ
package main

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/types"
)

// Updated demo to match the current unified effects API.
func integrationDemo() {
	fmt.Println("=== 統合効果システムデモ ===")

	// メモリ読み取り副作用（I/O効果として扱う）
	memRead := types.NewIOEffect(types.IOEffectSharedMemoryRead, types.IOPermissionRead)
	memRead.Description = "メモリから値を読む"

	// ファイル書き込みI/O効果
	fileWrite := types.NewIOEffect(types.IOEffectFileWrite, types.IOPermissionWrite)
	fileWrite.Description = "ファイルへ書き込む"

	// 例外効果（ExceptionSpecを使う）.
	excSpec := &types.ExceptionSpec{
		Kind:     types.ExceptionIOError,
		Severity: types.ExceptionSeverityCritical,
		Message:  "I/O例外",
	}

	// 例外効果の作成.
	excEffect := types.NewExceptionEffect(types.ExceptionEffectIOError, 1)
	excEffect.Spec = excSpec

	// 統合効果へ変換.
	conv := types.NewUnifiedEffectConverter()
	ue1 := conv.FromIOEffect(memRead)
	ue2 := conv.FromIOEffect(fileWrite)
	ue3 := conv.FromExceptionEffect(excEffect)

	// 統合セット.
	set := types.NewUnifiedEffectSet()
	set.Add(ue1)
	set.Add(ue2)
	set.Add(ue3)

	fmt.Printf("統合効果セット: %d個\n", set.Size())
	for _, e := range set.ToSlice() {
		fmt.Printf("- %s: %s\n", e.Kind.String(), e.Description)
	}

	// 統合シグネチャ.
	sig := types.NewUnifiedEffectSignature("complexFunc")
	sig.AddEffect(ue1)
	sig.AddEffect(ue2)
	sig.AddEffect(ue3)
	fmt.Printf("complexFunc pure? %v\n", sig.Pure)

	// 制約テスト.
	kindConstraint := types.NewUnifiedEffectKindConstraint()
	kindConstraint.Allow(types.UnifiedEffectIOWrite)
	kindConstraint.Deny(types.UnifiedEffectThrowsException)
	sig.AddConstraint(kindConstraint)
	violations := sig.CheckConstraints(ue3)
	fmt.Printf("例外効果の制約違反数: %d\n", len(violations))
}
