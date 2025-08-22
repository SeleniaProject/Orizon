; Self-contained Orizon OS Bootloader
; Loads and executes Orizon bytecode directly

[BITS 16]
[ORG 0x7C00]

start:
    ; Clear screen and setup
    call clear_screen
    call setup_segments
    
    ; Print welcome message
    mov si, welcome_msg
    call print_string
    
    ; Switch to 32-bit protected mode
    call enable_a20
    call setup_gdt
    call load_kernel_from_disk
    call enter_protected_mode
    
    jmp $  ; Halt here for now

clear_screen:
    mov ah, 0x00
    mov al, 0x03
    int 0x10
    ret

setup_segments:
    xor ax, ax
    mov ds, ax
    mov es, ax
    mov ss, ax
    mov sp, 0x7C00
    ret

enable_a20:
    ; Enable A20 line for 32-bit addressing
    in al, 0x92
    or al, 2
    out 0x92, al
    ret

setup_gdt:
    lgdt [gdt_descriptor]
    ret

load_kernel_from_disk:
    ; Load Orizon kernel from sector 2 to 0x1000
    mov ah, 0x02    ; Read sectors
    mov al, 18      ; Number of sectors
    mov ch, 0       ; Cylinder
    mov cl, 2       ; Starting sector
    mov dh, 0       ; Head
    mov dl, 0x80    ; Drive (first hard disk)
    mov bx, 0x1000  ; Load address
    int 0x13
    jc disk_error
    ret

disk_error:
    mov si, disk_error_msg
    call print_string
    jmp $

enter_protected_mode:
    mov eax, cr0
    or eax, 1
    mov cr0, eax
    
    jmp 0x08:protected_mode_main

[BITS 32]
protected_mode_main:
    ; Setup 32-bit segments
    mov ax, 0x10
    mov ds, ax
    mov es, ax
    mov fs, ax
    mov gs, ax
    mov ss, ax
    
    ; Print 32-bit message
    mov esi, pm_msg
    call print_string_pm
    
    ; Jump to Orizon kernel (embedded bytecode)
    jmp orizon_kernel_start

print_string_pm:
    mov ebx, 0xB8000  ; VGA text buffer
    mov ecx, 0
.loop:
    lodsb
    cmp al, 0
    je .done
    mov [ebx + ecx * 2], al
    mov byte [ebx + ecx * 2 + 1], 0x07  ; White on black
    inc ecx
    jmp .loop
.done:
    ret

orizon_kernel_start:
    ; Load and execute Orizon kernel from disk
    ; The kernel was loaded at 0x1000 by the bootloader
    
    ; Display Orizon startup message
    mov esi, orizon_msg
    call print_string_pm
    
    ; Jump to loaded Orizon kernel at 0x1000
    call 0x1000
    
    ; If kernel returns, show success
    mov esi, success_msg
    call print_string_pm
    
    ; Infinite loop
    jmp $

[BITS 16]
print_string:
    lodsb
    cmp al, 0
    je .done
    mov ah, 0x0E
    int 0x10
    jmp print_string
.done:
    ret

; Data section
welcome_msg db 'Pure Orizon OS Self-Contained Edition', 13, 10, 'Loading kernel...', 13, 10, 0
disk_error_msg db 'Disk read error!', 13, 10, 0
pm_msg db 'Protected mode active! Orizon kernel loaded.', 0
orizon_msg db 'Starting Pure Orizon Kernel...', 0
success_msg db 'Pure Orizon OS Running Successfully!', 0

; GDT (Global Descriptor Table)
gdt_start:
    ; Null descriptor
    dd 0x0
    dd 0x0
    
    ; Code segment descriptor
    dw 0xFFFF    ; Limit low
    dw 0x0000    ; Base low
    db 0x00      ; Base middle
    db 10011010b ; Access byte
    db 11001111b ; Granularity
    db 0x00      ; Base high
    
    ; Data segment descriptor
    dw 0xFFFF    ; Limit low
    dw 0x0000    ; Base low
    db 0x00      ; Base middle
    db 10010010b ; Access byte
    db 11001111b ; Granularity
    db 0x00      ; Base high

gdt_descriptor:
    dw gdt_descriptor - gdt_start - 1  ; GDT size
    dd gdt_start                       ; GDT address

; Boot signature
times 510-($-$$) db 0
dw 0xAA55
