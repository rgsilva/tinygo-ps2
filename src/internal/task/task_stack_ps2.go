//go:build scheduler.tasks && ps2

package task

import "unsafe"

// PlayStation 2 EE: MIPS N32 ABI (32-bit pointers, 64-bit GPRs, single-float
// FPU). The saved context differs from the o32 one in task_stack_mipsx.go:
// the callee-saved GPRs are 64 bits wide and must be saved whole (sd/ld), the
// stack pointer must stay 16-byte aligned, and the single-precision FPU
// registers f20-f31 are callee-saved (LLVM saves the even ones for N32
// single-float, gcc all of them; saving all twelve covers both).

var systemStack uintptr

// calleeSavedRegs is the register context tinygo_swapTask (loader/asm_mipsx.S
// in the demos repo, task_stack_ps2.S here) pushes and pops. The layout must
// match the assembly exactly. Total size 128 bytes (a multiple of 16).
type calleeSavedRegs struct {
	s0 uint64 // 0
	s1 uint64 // 8
	s2 uint64 // 16
	s3 uint64 // 24
	s4 uint64 // 32
	s5 uint64 // 40
	s6 uint64 // 48
	s7 uint64 // 56
	s8 uint64 // 64 (fp)
	ra uint64 // 72
	// Single-precision FPU callee-saved registers.
	f20, f21, f22, f23 uint32 // 80..92
	f24, f25, f26, f27 uint32 // 96..108
	f28, f29, f30, f31 uint32 // 112..124
}

// archInit runs architecture-specific setup for the goroutine startup.
func (s *state) archInit(r *calleeSavedRegs, fn uintptr, args unsafe.Pointer) {
	// Store the initial sp for the startTask function (implemented in assembly).
	// r sits at the top of the stack allocation; the allocation is 16-byte
	// aligned and the struct is 128 bytes, so sp is 16-byte aligned as N32
	// requires.
	s.sp = uintptr(unsafe.Pointer(r))
	if s.sp&15 != 0 {
		runtimeFatal("goroutine stack is not 16-byte aligned")
	}

	// Initialize the registers. They are popped off the stack on the first
	// resume of the goroutine, which then starts at tinygo_startTask: it calls
	// the function in s0 with the argument in s1 and exits the task when that
	// returns. Values are zero-extended to 64 bits, which is correct for
	// user-space addresses (below 0x80000000) on N32.
	r.ra = uint64(uintptr(unsafe.Pointer(&startTask)))
	r.s0 = uint64(fn)
	r.s1 = uint64(uintptr(args))
}

func (s *state) resume() {
	swapTask(s.sp, &systemStack)
}

func (s *state) pause() {
	newStack := systemStack
	systemStack = 0
	swapTask(newStack, &s.sp)
}

// SystemStack returns the system stack pointer when called from a task stack.
// When called from the system stack, it returns 0.
func SystemStack() uintptr {
	return systemStack
}

// CurrentStackBottom returns the lowest address of the running goroutine's
// stack (the canary word), or 0 on the system stack. The GC uses it to wipe
// the unused part of the stack before scanning it conservatively.
func CurrentStackBottom() uintptr {
	t := Current()
	if t == nil {
		return 0
	}
	return uintptr(unsafe.Pointer(t.state.canaryPtr))
}
