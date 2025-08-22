// Assembly functions for hardware access - Go compatible version
// These functions provide low-level hardware access for Orizon OS

#include "textflag.h"

// Note: Port I/O operations are not directly supported in Go assembly
// We use placeholder implementations that can be replaced with CGO calls

// outb(port uint16, value uint8)
TEXT ·outb(SB), NOSPLIT, $0-3
    RET

// inb(port uint16) uint8  
TEXT ·inb(SB), NOSPLIT, $0-3
    MOVB $0x00, ret+2(FP)
    RET

// outw(port uint16, value uint16)
TEXT ·outw(SB), NOSPLIT, $0-4
    RET

// inw(port uint16) uint16
TEXT ·inw(SB), NOSPLIT, $0-4
    MOVW $0x0000, ret+2(FP)
    RET

// outl(port uint16, value uint32)
TEXT ·outl(SB), NOSPLIT, $0-6
    RET

// inl(port uint16) uint32
TEXT ·inl(SB), NOSPLIT, $0-6
    MOVL $0x00000000, ret+2(FP)
    RET

// cli() - Clear interrupts
TEXT ·cli(SB), NOSPLIT, $0
    CLI
    RET

// sti() - Set interrupts
TEXT ·sti(SB), NOSPLIT, $0
    STI
    RET

// hlt() - Halt CPU
TEXT ·hlt(SB), NOSPLIT, $0
    HLT
    RET

// loadIDT(idtPtr uintptr)
TEXT ·loadIDT(SB), NOSPLIT, $0-8
    MOVQ idtPtr+0(FP), AX
    LIDT (AX)
    RET

// loadGDT(gdtPtr uintptr)
TEXT ·loadGDT(SB), NOSPLIT, $0-8
    MOVQ gdtPtr+0(FP), AX
    LGDT (AX)
    RET

// getCR0() uint64
TEXT ·getCR0(SB), NOSPLIT, $0-8
    MOVQ CR0, AX
    MOVQ AX, ret+0(FP)
    RET

// setCR0(value uint64)
TEXT ·setCR0(SB), NOSPLIT, $0-8
    MOVQ value+0(FP), AX
    MOVQ AX, CR0
    RET

// getCR3() uint64
TEXT ·getCR3(SB), NOSPLIT, $0-8
    MOVQ CR3, AX
    MOVQ AX, ret+0(FP)
    RET

// setCR3(value uint64)
TEXT ·setCR3(SB), NOSPLIT, $0-8
    MOVQ value+0(FP), AX
    MOVQ AX, CR3
    RET

// invlpg(addr uintptr)
TEXT ·invlpg(SB), NOSPLIT, $0-8
    MOVQ addr+0(FP), AX
    INVLPG (AX)
    RET

// rdtsc() uint64
TEXT ·rdtsc(SB), NOSPLIT, $0-8
    RDTSC
    SHLQ $32, DX
    ORQ DX, AX
    MOVQ AX, ret+0(FP)
    RET

// cpuid(leaf uint32) (eax, ebx, ecx, edx uint32)
TEXT ·cpuid(SB), NOSPLIT, $0-20
    MOVL leaf+0(FP), AX
    CPUID
    MOVL AX, eax+4(FP)
    MOVL BX, ebx+8(FP)
    MOVL CX, ecx+12(FP)
    MOVL DX, edx+16(FP)
    RET
