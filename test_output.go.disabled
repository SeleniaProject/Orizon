package main

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/exception"
)

func main() {
	integration := exception.NewSimplifiedMIRIntegration()
	mirCode := integration.GenerateBoundsCheckMIR("index_var", "length_var", "myArray")
	fmt.Printf("Generated MIR:\n%s\n", mirCode)

	emitter := exception.NewX64ExceptionEmitter()
	x64Code := emitter.EmitBoundsCheckX64("rax", "rbx", "testArray")
	fmt.Printf("Generated x64:\n%s\n", x64Code)
}
