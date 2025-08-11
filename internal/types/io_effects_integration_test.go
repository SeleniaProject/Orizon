// Phase 2.5.3 Effect Type System Integration Test
// 統合効果システム（副作用・例外・I/O効果）の完全テスト
package types

import (
	"testing"
)

func TestUnifiedEffectIntegration(t *testing.T) {
	// 1. Side effect + I/O effect + Exception effect の複合
	memRead := NewIOEffect(IOEffectSharedMemoryRead, IOPermissionRead)
	fileWrite := NewIOEffect(IOEffectFileWrite, IOPermissionWrite)
	exc := NewExceptionEffect(ExceptionEffectThrows, EffectLevelCritical)

	// 統合効果へ変換
	conv := NewUnifiedEffectConverter()
	ue1 := conv.FromIOEffect(memRead)
	ue2 := conv.FromIOEffect(fileWrite)
	ue3 := conv.FromExceptionEffect(exc)

	// 統合セット
	set := NewUnifiedEffectSet()
	set.Add(ue1)
	set.Add(ue2)
	set.Add(ue3)

	if set.Size() != 3 {
		t.Errorf("統合効果セットのサイズが不正: got %d, want 3", set.Size())
	}
	if set.IsPure() {
		t.Error("副作用・例外・I/O効果を含むセットはpureであってはならない")
	}

	// 統合シグネチャ
	sig := NewUnifiedEffectSignature("complexFunc")
	sig.AddEffect(ue1)
	sig.AddEffect(ue2)
	sig.AddEffect(ue3)
	if sig.Pure {
		t.Error("複合効果関数はpureであってはならない")
	}
	if !sig.Effects.Contains(ue2) {
		t.Error("fileWrite効果が統合シグネチャに含まれていない")
	}

	// 制約テスト
	kindConstraint := NewUnifiedEffectKindConstraint()
	kindConstraint.Allow(UnifiedEffectIOWrite)
	kindConstraint.Deny(UnifiedEffectThrowsException)
	sig.AddConstraint(kindConstraint)
	violations := sig.CheckConstraints(ue3)
	if len(violations) == 0 {
		t.Error("例外効果は制約違反になるべき")
	}
}
