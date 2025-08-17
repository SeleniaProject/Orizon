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
demo_advanced_macros:
  push rbp
  mov rbp, rsp
entry:
  mov rsp, rbp
  pop rbp
  ret
