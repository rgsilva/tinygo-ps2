//go:build ps2 && (gc.conservative || gc.precise)

package runtime

import (
	"internal/task"
	"sync/atomic"
	"unsafe"
)

// Stack scanning on the PS2. Same as gc_stack_raw.go, except that the unused
// part of the running goroutine's stack is zeroed before the scan: goroutine
// stacks are heap objects scanned conservatively in full, and stale pointers
// left below the stack pointer by earlier, deeper calls would otherwise pin
// dead objects (which also stops finalizers from running). Suspended
// goroutines keep their stale area until they run deeper again.

// Unused.
var gcScanState atomic.Uint32

// stackScrubMargin is left untouched below scanstack's own stack pointer:
// memzero's frame lives there.
const stackScrubMargin = 512

func gcMarkReachable() {
	markStack()
	findGlobals(markRoots)
}

// markStack marks all root pointers found on the stack.
func markStack() {
	// Scan the current stack, and all current registers.
	scanCurrentStack()

	if !task.OnSystemStack() {
		// Mark system stack.
		markRoots(task.SystemStack(), stackTop)
	}
}

//go:export tinygo_scanCurrentStack
func scanCurrentStack()

// getsp returns the caller's stack pointer (assembly, tinygo_getsp).
//
//go:export tinygo_getsp
func getsp() uintptr

//go:export tinygo_scanstack
func scanstack(sp uintptr) {
	// Mark current stack.
	// This function is called by scanCurrentStack, after pushing all registers onto the stack.
	// Callee-saved registers have been pushed onto stack by tinygo_localscan, so this will scan them too.
	if task.OnSystemStack() {
		// This is the system stack.
		// Scan all words on the stack.
		markRoots(sp, stackTop)
	} else {
		// This is a goroutine stack: wipe what is below the stack pointer,
		// then mark the whole stack object (scanned conservatively).
		bottom := task.CurrentStackBottom() + unsafe.Sizeof(uintptr(0)) // keep the canary
		limit := getsp() - stackScrubMargin                             // below our own frame
		if bottom != 0 && limit > bottom {
			memzero(unsafe.Pointer(bottom), limit-bottom)
		}
		markCurrentGoroutineStack(sp)
	}
}

func gcResumeWorld() {
	// Nothing to do here (single threaded).
}
