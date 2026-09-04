// Runtime assembly for the PlayStation 2 EE: MIPS N32 (64-bit GPRs, 32-bit
// pointers, 16-byte stack alignment). This is the N32 version of asm_mipsx.S
// (o32: sw/lw of 32-bit registers, 8-byte aligned frames). Task switching
// lives in src/internal/task/task_stack_ps2.S.

// Do not reorder instructions to insert a branch delay slot.
// We know what we're doing, and will manually fill the branch delay slot.
.set noreorder

.section .text.tinygo_scanCurrentStack
.global  tinygo_scanCurrentStack
.type    tinygo_scanCurrentStack, %function
tinygo_scanCurrentStack:
    // Push the callee-saved registers so the conservative GC sees pointers
    // held in them (gp included: optimized N32 code uses it as an ordinary
    // register). 112 bytes keeps sp 16-byte aligned.
    addiu $sp, $sp, -112
    sd    $ra, 96($sp)
    sd    $gp, 88($sp)
    sd    $fp, 72($sp)
    sd    $s7, 64($sp)
    sd    $s6, 56($sp)
    sd    $s5, 48($sp)
    sd    $s4, 40($sp)
    sd    $s3, 32($sp)
    sd    $s2, 24($sp)
    sd    $s1, 16($sp)
    sd    $s0,  8($sp)

    // Scan the stack.
    jal   tinygo_scanstack
    move  $a0, $sp // in the branch delay slot

    ld    $ra, 96($sp)
    addiu $sp, $sp, 112
    jr    $ra
    nop

.section .text.tinygo_longjmp
.global tinygo_longjmp
.type   tinygo_longjmp, %function
tinygo_longjmp:
    // Note: the code we jump to assumes a0 is non-zero, which is already the
    // case because that's the defer frame pointer.
    lw $sp, 0($a0) // jumpSP
    lw $a1, 4($a0) // jumpPC
    jr $a1
    nop

.section .text.tinygo_getsp
.global  tinygo_getsp
.type    tinygo_getsp, %function
tinygo_getsp:
    // Return the caller's stack pointer (used by the GC stack scrub).
    jr    $ra
    move  $v0, $sp

.section .text.tinygo_getgp
.global  tinygo_getgp
.type    tinygo_getgp, %function
tinygo_getgp:
    // Return the current gp (the initial context of a new goroutine).
    jr    $ra
    move  $v0, $gp
