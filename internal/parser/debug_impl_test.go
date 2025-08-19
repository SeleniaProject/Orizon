package parser

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/lexer"
)

func TestSimpleImplWhere(t *testing.T) {
	// Test case 1: Working case (with generics)
	src1 := "impl<T> S<T> where T: Eq { }"
	l1 := lexer.New(src1)
	p1 := NewParser(l1, "test1.oriz")
	prog1, errs1 := p1.Parse()
	if len(errs1) != 0 {
		t.Errorf("Test 1 failed: %v", errs1)
	} else {
		t.Log("Test 1 passed")
	}

	// Test case 2: Failing case (without generics on type)
	src2 := "impl<T> S where T: Eq { }"
	l2 := lexer.New(src2)
	p2 := NewParser(l2, "test2.oriz")
	prog2, errs2 := p2.Parse()
	if len(errs2) != 0 {
		t.Errorf("Test 2 failed: %v", errs2)
	} else {
		t.Log("Test 2 passed")
	}

	_ = prog1
	_ = prog2
}
