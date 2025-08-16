package collections

import "testing"

func TestVectorBasic(t *testing.T) {
	v := NewVector[int](0)
	if v.Len() != 0 || v.Cap() < 0 {
		t.Fatalf("unexpected zero state")
	}

	v.Append(1, 2)
	v.Push(3)
	if v.Len() != 3 {
		t.Fatalf("len=3 expected got %d", v.Len())
	}

	x, ok := v.Get(1)
	if !ok || x != 2 {
		t.Fatalf("Get(1)=2,true expected got %v,%v", x, ok)
	}

	if !v.Set(1, 20) {
		t.Fatalf("Set should succeed")
	}
	if y, _ := v.Get(1); y != 20 {
		t.Fatalf("Set/Get mismatch: %v", y)
	}

	if _, ok := v.Get(100); ok {
		t.Fatalf("out of range Get should be false")
	}

	z, ok := v.Pop()
	if !ok || z != 3 {
		t.Fatalf("Pop last=3 expected got %v,%v", z, ok)
	}
	if v.Len() != 2 {
		t.Fatalf("len=2 expected got %d", v.Len())
	}
}

func TestVectorIterAndAlgo(t *testing.T) {
	v := NewVector[int](0)
	v.Append(1, 2, 3, 4)

	it := v.Iter()
	sum := 0
	for {
		x, ok := it.Next()
		if !ok {
			break
		}
		sum += x
	}
	if sum != 10 {
		t.Fatalf("iter sum=10 expected got %d", sum)
	}

	m := MapVector(v, func(x int) int { return x * 2 })
	if m.Len() != 4 || m.At(0) != 2 || m.At(3) != 8 {
		t.Fatalf("MapVector failed: %v", m.ToSlice())
	}

	f := v.Filter(func(x int) bool { return x%2 == 0 })
	if f.Len() != 2 || f.At(0) != 2 || f.At(1) != 4 {
		t.Fatalf("Filter failed: %v", f.ToSlice())
	}

	r := ReduceVector(v, 0, func(acc, x int) int { return acc + x })
	if r != 10 {
		t.Fatalf("ReduceVector sum=10 expected got %d", r)
	}
}
