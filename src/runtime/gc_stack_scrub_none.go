//go:build !ps2

package runtime

// scrubDeadStack is called from scanstack before the running goroutine's
// stack is scanned. Most targets do not need to do anything here.
func scrubDeadStack() {}
