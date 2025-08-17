; module main
main:
  push rbp
  mov rbp, rsp
entry:
  mov rax, 0
  mov rsp, rbp
  pop rbp
  ret
cont_t0:
  mov rsp, rbp
  pop rbp
  ret
