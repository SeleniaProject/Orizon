; Minimal Orizon OS Bootloader
; Fits in 512 bytes and loads kernel

[BITS 16]
[ORG 0x7C00]

start:
    ; Clear interrupts and set up stack
    cli
    xor ax, ax
    mov ds, ax
    mov es, ax
    mov ss, ax
    mov sp, 0x7C00

    ; Print boot message
    mov si, boot_msg
    call print_string

    ; Load kernel from disk
    mov ah, 0x02        ; Read sectors function
    mov al, 18          ; Number of sectors to read
    mov ch, 0           ; Cylinder
    mov cl, 2           ; Starting sector (sector after bootloader)
    mov dh, 0           ; Head
    mov dl, 0x80        ; Drive number (first hard disk)
    mov bx, 0x1000      ; Load address
    int 0x13
    jc disk_error

    ; Print success message
    mov si, success_msg
    call print_string

    ; Simple kernel simulation (infinite loop with message)
    mov si, kernel_msg
    call print_string
    
    ; Infinite loop (kernel would take over here)
    jmp infinite_loop

disk_error:
    mov si, error_msg
    call print_string
    jmp infinite_loop

infinite_loop:
    hlt
    jmp infinite_loop

print_string:
    lodsb
    or al, al
    jz .done
    mov ah, 0x0E
    int 0x10
    jmp print_string
.done:
    ret

boot_msg db 'Orizon OS Loading...', 13, 10, 0
success_msg db 'Kernel loaded successfully!', 13, 10, 0
kernel_msg db 'Orizon Kernel Running!', 13, 10, 0
error_msg db 'Boot Error!', 13, 10, 0

; Pad to 510 bytes and add boot signature
times 510-($-$$) db 0
dw 0xAA55
