//go:build ps2

package runtime

import (
	"internal/task"
	"unsafe"
)

// stackScrubMargin is left untouched below scrubDeadStack's own stack
// pointer: memzero's frame lives there.
const stackScrubMargin = 512

// getsp returns the caller's stack pointer (assembly, tinygo_getsp).
//
//go:export tinygo_getsp
func getsp() uintptr

// scrubDeadStack zeroes the unused part of the running goroutine's stack
// before it is scanned. Goroutine stacks are heap objects scanned
// conservatively in full, and on the PS2 RAM starts at address 0, so stale
// words left below the stack pointer by earlier, deeper calls look like heap
// pointers and would pin dead objects (which also stops finalizers from
// running). Suspended goroutines keep their stale area until they run deeper
// again.
func scrubDeadStack() {
	bottom := task.CurrentStackBottom() + unsafe.Sizeof(uintptr(0)) // keep the canary
	limit := getsp() - stackScrubMargin                             // below our own frame
	if bottom != 0 && limit > bottom {
		memzero(unsafe.Pointer(bottom), limit-bottom)
	}
}
