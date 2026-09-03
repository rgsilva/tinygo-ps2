//go:build ps2

// What baremetal.go provides for other targets, minus the malloc/calloc/free
// exports: on the PS2 the C heap belongs to the ps2sdk libc (newlib), whose
// allocator cannot be replaced without symbol clashes.

package runtime

import "C"
import (
	"sync/atomic"
)

// The heap and stack bounds come from baremetal_memory.go (linker symbols
// _heap_start, _heap_end, _globals_start, _globals_end, _stack_top, provided
// by the ps2 link flags). preinit replaces the heap bounds with a block from
// the ps2sdk libc heap; these exported copies let tools report them.
var (
	HeapStart uintptr
	HeapEnd   uintptr
)

//export runtime_putchar
func runtime_putchar(c byte) {
	putchar(c)
}

//go:linkname syscall_Exit syscall.Exit
func syscall_Exit(code int) {
	exit(code)
}

const baremetal = true

// timeOffset is how long the monotonic clock started after the Unix epoch. It
// should be a positive integer under normal operation or zero when it has not
// been set.
var timeOffset atomic.Int64

//go:linkname now time.now
func now() (sec int64, nsec int32, mono int64) {
	mono = nanotime()
	to := timeOffset.Load()
	sec = (mono + to) / (1000 * 1000 * 1000)
	nsec = int32((mono + to) - sec*(1000*1000*1000))
	return
}

// AdjustTimeOffset adds the given offset to the built-in time offset. A
// positive value adds to the time (skipping some time), a negative value moves
// the clock into the past.
func AdjustTimeOffset(offset int64) {
	timeOffset.Add(offset)
}

// Picolibc is not configured to define its own errno value, instead it calls
// __errno_location.
// TODO: a global works well enough for now (same as errno on Linux with
// -scheduler=tasks), but this should ideally be a thread-local variable stored
// in task.Task.
// Especially when we add multicore support for microcontrollers.
var errno int32

//export __errno_location
func libc_errno_location() *int32 {
	return &errno
}
