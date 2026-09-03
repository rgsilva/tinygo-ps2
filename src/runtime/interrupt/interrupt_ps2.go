//go:build ps2

package interrupt

/*
extern int DIntr(void);
extern int EIntr(void);
*/
import "C"

// State represents the previous global interrupt state.
type State uintptr

// Disable disables all interrupts and returns the previous interrupt state. It
// can be used in a critical section like this:
//
//	state := interrupt.Disable()
//	// critical section
//	interrupt.Restore(state)
//
// Critical sections can be nested. Make sure to call Restore in the same order
// as you called Disable (this happens naturally with the pattern above).
func Disable() (state State) {
	// DIntr returns non-zero if interrupts were enabled (and are now off),
	// zero if they were already disabled.
	if C.DIntr() != 0 {
		return 1
	}
	return 0
}

// Restore restores interrupts to what they were before. Give the previous state
// returned by Disable as a parameter. If interrupts were disabled before
// calling Disable, this will not re-enable interrupts, allowing for nested
// critical sections.
func Restore(state State) {
	if state != 0 {
		C.EIntr()
	}
}

// In returns whether the system is currently in an interrupt.
func In() bool {
	// Go code never runs in an interrupt handler on this target.
	return false
}
