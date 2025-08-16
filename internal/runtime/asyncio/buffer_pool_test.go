package asyncio

import "testing"

func TestBytePool_GetPut(t *testing.T) {
	bp := DefaultBytePool()
	buf := bp.Get(1500) // should map to 2048 bucket
	if cap(buf) < 1500 {
		t.Fatalf("cap too small: %d", cap(buf))
	}
	sizes, inuse := bp.Stats()
	_ = sizes
	var sum int64
	for _, v := range inuse {
		sum += v
	}
	if sum == 0 {
		t.Fatal("expected inuse > 0")
	}
	bp.Put(buf)
}

func TestBytePool_Oversize(t *testing.T) {
	bp := DefaultBytePool()
	buf := bp.Get(1 << 20) // 1MB oversize
	if cap(buf) < (1 << 20) {
		t.Fatalf("cap too small: %d", cap(buf))
	}
	// Put should drop oversize without panic
	bp.Put(buf)
}
