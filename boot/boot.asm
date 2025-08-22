; Orizon OS Bootloader - Stage 1
; This bootloader loads and starts the Orizon OS kernel
; Target: x86_64 architecture with UEFI support

BITS 16
ORG 0x7C00

; Boot sector magic
jmp start
nop

; BIOS Parameter Block (BPB) for compatibility
times 3-($-$$) db 0
oem_identifier     db 'ORIZONOS'
bytes_per_sector   dw 512
sectors_per_cluster db 1
reserved_sectors   dw 1
number_of_fats     db 2
root_entries       dw 224
total_sectors      dw 2880
media_type         db 0xF0
sectors_per_fat    dw 9
sectors_per_track  dw 18
number_of_heads    dw 2
hidden_sectors     dd 0
total_sectors_big  dd 0

; Extended boot record
drive_number       db 0
nt_flags          db 0
signature         db 0x29
volume_id         dd 0xD105
volume_label      db 'ORIZON OS  '
file_system       db 'FAT12   '

start:
    ; Clear interrupts and set up segments
    cli
    xor ax, ax
    mov ds, ax
    mov es, ax
    mov ss, ax
    mov sp, 0x7C00
    
    ; Store boot drive
    mov [drive_number], dl
    
    ; Print boot message
    mov si, boot_msg
    call print_string
    
    ; Enable A20 line (required for accessing memory above 1MB)
    call enable_a20
    
    ; Load Global Descriptor Table
    lgdt [gdt_descriptor]
    
    ; Enter protected mode
    mov eax, cr0
    or eax, 1
    mov cr0, eax
    
    ; Far jump to 32-bit code
    jmp CODE_SEG:protected_mode_start

; 16-bit functions
print_string:
    lodsb
    or al, al
    jz .done
    mov ah, 0x0E
    int 0x10
    jmp print_string
.done:
    ret

enable_a20:
    ; Fast A20 gate method
    in al, 0x92
    or al, 2
    out 0x92, al
    ret

; Global Descriptor Table
gdt_start:
    ; Null descriptor
    dq 0
    
gdt_code:
    ; Code segment: base=0, limit=0xfffff, access=9Ah, flags=CFh
    dw 0xFFFF    ; limit low
    dw 0x0000    ; base low
    db 0x00      ; base middle
    db 10011010b ; access byte
    db 11001111b ; flags + limit high
    db 0x00      ; base high
    
gdt_data:
    ; Data segment: base=0, limit=0xfffff, access=92h, flags=CFh
    dw 0xFFFF    ; limit low
    dw 0x0000    ; base low
    db 0x00      ; base middle
    db 10010010b ; access byte
    db 11001111b ; flags + limit high
    db 0x00      ; base high
    
gdt_end:

gdt_descriptor:
    dw gdt_end - gdt_start - 1  ; size
    dd gdt_start                ; offset

; Constants
CODE_SEG equ gdt_code - gdt_start
DATA_SEG equ gdt_data - gdt_start

; 32-bit protected mode code
BITS 32
protected_mode_start:
    ; Set up segment registers
    mov ax, DATA_SEG
    mov ds, ax
    mov ss, ax
    mov es, ax
    mov fs, ax
    mov gs, ax
    
    ; Set up stack
    mov esp, 0x90000
    
    ; Clear screen and print message
    call clear_screen
    mov esi, protected_msg
    call print_string_pm
    
    ; Load kernel from disk
    call load_kernel
    
    ; Check if kernel loaded successfully
    cmp eax, 0
    jne kernel_error
    
    ; Prepare for long mode (64-bit)
    call setup_long_mode
    
    ; Jump to kernel entry point
    mov esi, kernel_loaded_msg
    call print_string_pm
    
    ; Jump to Orizon kernel
    jmp KERNEL_OFFSET

; 32-bit functions
clear_screen:
    pusha
    mov edi, 0xB8000
    mov ecx, 80 * 25
    mov ax, 0x0F20  ; White on black space
    rep stosw
    popa
    ret

print_string_pm:
    pusha
    mov ebx, 0xB8000
    
.loop:
    lodsb
    or al, al
    jz .done
    
    mov [ebx], al
    mov byte [ebx + 1], 0x0F  ; White on black
    add ebx, 2
    jmp .loop
    
.done:
    popa
    ret

load_kernel:
    ; Load kernel sectors from disk
    ; This is a simplified version - real implementation would use filesystem
    mov esi, loading_msg
    call print_string_pm
    
    ; Reset disk
    mov ah, 0x00
    mov dl, [drive_number]
    int 0x13
    jc disk_error
    
    ; Read kernel sectors (starting from sector 2)
    mov ah, 0x02        ; Read sectors
    mov al, 18          ; Number of sectors to read (reduced from 63)
    mov ch, 0           ; Cylinder
    mov cl, 2           ; Starting sector
    mov dh, 0           ; Head
    mov dl, [drive_number]
    mov bx, KERNEL_OFFSET
    int 0x13
    jc disk_error
    
    xor eax, eax        ; Success
    ret

disk_error:
    mov esi, disk_error_msg
    call print_string_pm
    mov eax, 1
    ret

kernel_error:
    mov esi, kernel_error_msg
    call print_string_pm
    jmp $

setup_long_mode:
    ; Set up paging for long mode
    ; Create page tables at 0x1000
    mov edi, 0x1000
    mov cr3, edi
    xor eax, eax
    mov ecx, 4096
    rep stosd
    mov edi, cr3
    
    ; Set up page directory pointer table
    mov dword [edi], 0x2003      ; PDP at 0x2000
    mov dword [edi + 0x1000], 0x3003  ; PD at 0x3000
    mov dword [edi + 0x2000], 0x4003  ; PT at 0x4000
    
    ; Set up page table (identity map first 2MB)
    mov edi, 0x4000
    mov eax, 3
    mov ecx, 512
    
.set_entry:
    mov [edi], eax
    add eax, 0x1000
    add edi, 8
    loop .set_entry
    
    ; Enable PAE
    mov eax, cr4
    or eax, 1 << 5
    mov cr4, eax
    
    ; Set long mode bit in EFER MSR
    mov ecx, 0xC0000080
    rdmsr
    or eax, 1 << 8
    wrmsr
    
    ; Enable paging
    mov eax, cr0
    or eax, 1 << 31
    mov cr0, eax
    
    ret

; Constants
KERNEL_OFFSET equ 0x10000

; String messages
boot_msg          db 'Orizon OS Bootloader v1.0', 13, 10, 0
protected_msg     db 'Entering protected mode...', 0
loading_msg       db 'Loading Orizon kernel...', 0
kernel_loaded_msg db 'Kernel loaded, starting Orizon OS...', 0
disk_error_msg    db 'Disk read error!', 0
kernel_error_msg  db 'Kernel load failed!', 0

; Pad to 510 bytes and add boot signature  
times (510-($-$$)) db 0
dw 0xAA55
