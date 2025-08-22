// Package errors provides standardized error messaging for Orizon
package errors

import (
	"fmt"
	"runtime"
)

// ErrorCategory represents different categories of errors
type ErrorCategory string

const (
	CategoryMemory     ErrorCategory = "MEMORY"
	CategorySecurity   ErrorCategory = "SECURITY"
	CategoryBounds     ErrorCategory = "BOUNDS"
	CategoryOverflow   ErrorCategory = "OVERFLOW"
	CategoryValidation ErrorCategory = "VALIDATION"
	CategorySystem     ErrorCategory = "SYSTEM"
)

// StandardError provides a consistent error format
type StandardError struct {
	Category ErrorCategory
	Code     string
	Message  string
	Context  map[string]interface{}
	Caller   string
}

// Error implements the error interface
func (e *StandardError) Error() string {
	return fmt.Sprintf("[%s:%s] %s (caller: %s)", e.Category, e.Code, e.Message, e.Caller)
}

// NewStandardError creates a new standardized error
func NewStandardError(category ErrorCategory, code, message string, context map[string]interface{}) *StandardError {
	pc, _, _, ok := runtime.Caller(1)
	caller := "unknown"
	if ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			caller = fn.Name()
		}
	}

	return &StandardError{
		Category: category,
		Code:     code,
		Message:  message,
		Context:  context,
		Caller:   caller,
	}
}

// Common error constructors
func IndexOutOfBounds(index, length uintptr) *StandardError {
	return NewStandardError(CategoryBounds, "INDEX_OUT_OF_BOUNDS",
		fmt.Sprintf("Index %d out of bounds for length %d", index, length),
		map[string]interface{}{"index": index, "length": length})
}

func IntegerOverflow(operation string, values ...interface{}) *StandardError {
	return NewStandardError(CategoryOverflow, "INTEGER_OVERFLOW",
		fmt.Sprintf("Integer overflow in %s operation", operation),
		map[string]interface{}{"operation": operation, "values": values})
}

func NullPointer(operation string) *StandardError {
	return NewStandardError(CategoryMemory, "NULL_POINTER",
		fmt.Sprintf("Null pointer dereference in %s", operation),
		map[string]interface{}{"operation": operation})
}

func InvalidSize(size uintptr, context string) *StandardError {
	return NewStandardError(CategoryValidation, "INVALID_SIZE",
		fmt.Sprintf("Invalid size %d in %s", size, context),
		map[string]interface{}{"size": size, "context": context})
}

func PointerArithmetic(details string) *StandardError {
	return NewStandardError(CategorySecurity, "POINTER_ARITHMETIC",
		fmt.Sprintf("Unsafe pointer arithmetic: %s", details),
		map[string]interface{}{"details": details})
}
