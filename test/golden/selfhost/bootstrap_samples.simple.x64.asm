; module main
add:
  push rbp
  mov rbp, rsp
  sub rsp, 48
entry:
; alloca a -> %a.addr
  mov rax, qword ptr [a]
  mov qword ptr [rbp-8], rax
; alloca b -> %b.addr
  mov rax, qword ptr [b]
  mov qword ptr [rbp-16], rax
  mov rax, qword ptr [rbp-8]
  mov qword ptr [rbp-24], rax
  mov rax, qword ptr [rbp-16]
  mov qword ptr [rbp-32], rax
  mov rax, qword ptr [rbp-24]
  mov r10, qword ptr [rbp-32]
  add rax, r10
  mov qword ptr [rbp-40], rax
  mov rax, qword ptr [rbp-40]
  mov rsp, rbp
  pop rbp
  ret
cont_t3:
  mov rsp, rbp
  pop rbp
  ret
main:
  push rbp
  mov rbp, rsp
entry:
  mov rsp, rbp
  pop rbp
  ret
cont_t0:
  mov rsp, rbp
  pop rbp
  ret
