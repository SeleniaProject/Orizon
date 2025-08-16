package debug

import "testing"

func TestPCMap_AddrToLine(t *testing.T) {
	p := buildMinimalHIR()
	em := NewEmitter()
	dbg, err := em.Emit(p)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	m := BuildPCMap(dbg)
	if len(m.Ranges) == 0 {
		t.Fatalf("no ranges")
	}
	r := m.Ranges[0]
	// head
	if file, line, ok := m.AddrToLine(r.Low); !ok || file == "" || line == 0 {
		t.Fatalf("unexpected head resolve: %v %v %v", file, line, ok)
	}
	// tail-1
	if file, line, ok := m.AddrToLine(r.High - 1); !ok || file == "" || line == 0 {
		t.Fatalf("unexpected tail resolve: %v %v %v", file, line, ok)
	}
	// out of range
	if _, _, ok := m.AddrToLine(r.High + 1024); ok {
		t.Fatalf("expected miss for out-of-range address")
	}
}
