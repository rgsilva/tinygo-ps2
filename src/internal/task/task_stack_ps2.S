// Task switching for the PlayStation 2 EE (MIPS N32, R5900, single-float).
//
// Differences from task_stack_mipsx.S (o32):
//   - callee-saved GPRs s0-s7, fp, ra are 64 bits wide: sd/ld, 8-byte slots;
//   - the stack pointer must stay 16-byte aligned (frame = 128 bytes);
//   - the single-precision FPU registers f20-f31 are callee-saved and saved
//     with swc1/lwc1 (LLVM's N32 single-float CSR list has the even ones,
//     gcc's all twelve; save all twelve);
//   - gp is not touched: the program is non-PIC and gp is a constant set by
//     crt0, so leaving it alone preserves it (saving it would load 0 from
//     the initial frame of a new goroutine).
// The frame layout must match calleeSavedRegs in task_stack_ps2.go.

.set noreorder

// tinygo_task_exit only exists with a scheduler; weak so that a
// -scheduler=none build (which never runs this code) still links.
.weak tinygo_task_exit

.section .text.tinygo_startTask
.global  tinygo_startTask
.type    tinygo_startTask, %function
tinygo_startTask:
    // Runs on the new stack with the callee-saved registers just popped:
    // s0 = function to call (goroutine start wrapper), s1 = its argument.
    move  $a0, $s1
    jalr  $s0
    nop

    // The goroutine returned: exit it. Tail call.
    j     tinygo_task_exit
    nop

.section .text.tinygo_swapTask
.global  tinygo_swapTask
.type    tinygo_swapTask, %function
tinygo_swapTask:
    // a0 = newStack uintptr
    // a1 = oldStack *uintptr
    addiu $sp, $sp, -128
    sd    $s0,   0($sp)
    sd    $s1,   8($sp)
    sd    $s2,  16($sp)
    sd    $s3,  24($sp)
    sd    $s4,  32($sp)
    sd    $s5,  40($sp)
    sd    $s6,  48($sp)
    sd    $s7,  56($sp)
    sd    $fp,  64($sp)
    sd    $ra,  72($sp)
    swc1  $f20, 80($sp)
    swc1  $f21, 84($sp)
    swc1  $f22, 88($sp)
    swc1  $f23, 92($sp)
    swc1  $f24, 96($sp)
    swc1  $f25, 100($sp)
    swc1  $f26, 104($sp)
    swc1  $f27, 108($sp)
    swc1  $f28, 112($sp)
    swc1  $f29, 116($sp)
    swc1  $f30, 120($sp)
    swc1  $f31, 124($sp)

    // Save the current stack pointer in *oldStack (32-bit pointer).
    sw    $sp, 0($a1)

    // Switch to the new stack.
    move  $sp, $a0

    ld    $s0,   0($sp)
    ld    $s1,   8($sp)
    ld    $s2,  16($sp)
    ld    $s3,  24($sp)
    ld    $s4,  32($sp)
    ld    $s5,  40($sp)
    ld    $s6,  48($sp)
    ld    $s7,  56($sp)
    ld    $fp,  64($sp)
    ld    $ra,  72($sp)
    lwc1  $f20, 80($sp)
    lwc1  $f21, 84($sp)
    lwc1  $f22, 88($sp)
    lwc1  $f23, 92($sp)
    lwc1  $f24, 96($sp)
    lwc1  $f25, 100($sp)
    lwc1  $f26, 104($sp)
    lwc1  $f27, 108($sp)
    lwc1  $f28, 112($sp)
    lwc1  $f29, 116($sp)
    lwc1  $f30, 120($sp)
    lwc1  $f31, 124($sp)
    addiu $sp, $sp, 128

    // Return into the task (tinygo_startTask on the first switch).
    jr    $ra
    nop

