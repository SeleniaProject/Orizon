package collections

import (
	"fmt"
	"testing"
)

func TestBasicVector(t *testing.T) {
	vec := NewVector[int]()

	// Test basic operations
	vec.Push(1)
	vec.Push(2)
	vec.Push(3)

	if vec.Len() != 3 {
		t.Errorf("Expected length 3, got %d", vec.Len())
	}

	val, ok := vec.Get(1)
	if !ok || val != 2 {
		t.Errorf("Expected 2 at index 1, got %v", val)
	}

	popped, ok := vec.Pop()
	if !ok || popped != 3 {
		t.Errorf("Expected to pop 3, got %v", popped)
	}

	if vec.Len() != 2 {
		t.Errorf("Expected length 2 after pop, got %d", vec.Len())
	}

	fmt.Printf("Vector test completed successfully\n")
}

func TestBasicHashMap(t *testing.T) {
	hashMap := NewHashMap[string, int]()

	// Test basic operations
	hashMap.Insert("key1", 1)
	hashMap.Insert("key2", 2)
	hashMap.Insert("key3", 3)

	if hashMap.Len() != 3 {
		t.Errorf("Expected size 3, got %d", hashMap.Len())
	}

	val, ok := hashMap.Get("key2")
	if !ok || val != 2 {
		t.Errorf("Expected 2 for key2, got %v", val)
	}

	if hashMap.Remove("key2") != true {
		t.Error("Expected to successfully remove key2")
	}

	if hashMap.Len() != 2 {
		t.Errorf("Expected size 2 after removal, got %d", hashMap.Len())
	}

	_, ok = hashMap.Get("key2")
	if ok {
		t.Error("key2 should not exist after removal")
	}

	fmt.Printf("HashMap test completed successfully\n")
}

func TestBasicAtomicCounter(t *testing.T) {
	counter := NewAtomicCounter()

	// Test increment
	counter.Inc()
	counter.Inc()
	counter.Add(5)

	if counter.Get() != 7 {
		t.Errorf("Expected counter value 7, got %d", counter.Get())
	}

	// Test decrement
	counter.Dec()
	if counter.Get() != 6 {
		t.Errorf("Expected counter value 6, got %d", counter.Get())
	}

	fmt.Printf("AtomicCounter test completed successfully\n")
}

func TestBasicCompilation(t *testing.T) {
	// Just test that all types can be instantiated
	vec := NewVector[string]()
	vec.Push("test")

	hashMap := NewHashMap[int, string]()
	hashMap.Insert(1, "one")

	hashSet := NewHashSet[string]()
	hashSet.Insert("item")

	stack := NewLockFreeStack[int]()
	stack.Push(42)

	counter := NewAtomicCounter()
	counter.Inc()

	fmt.Printf("All basic data structures compile and instantiate correctly\n")
}
