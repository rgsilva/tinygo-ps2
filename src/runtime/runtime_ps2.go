//go:build ps2

package runtime

/*
#define _EE
#include <kernel.h>

// ps2sdk system timer: 64-bit count of EE bus clock cycles (147.456 MHz),
// kept by libkernel with the T2 overflow interrupt. Referencing it makes
// crt0's _InitSys start the timer (weak _ps2sdk_init_timer hook).
extern unsigned long long GetTimerSystemTime(void);

extern void _exit(int status);
extern int SleepThread(void);
extern int iWakeupThread(int thread_id);
extern int SetTimerAlarm(unsigned long long clock_cycles, unsigned long long (*handler)(int, unsigned long long, unsigned long long, void *, void *), void *arg);
extern int ReleaseTimerAlarm(int id);

// Timer alarm handler used by sleepTicks. It runs in interrupt context, so
// it does nothing but wake the thread that armed it (no Go code here).
static unsigned long long ps2go_wake(int id, unsigned long long scheduled, unsigned long long actual, void *arg, void *pc) {
	iWakeupThread((int)(long)arg);
	return 0;
}

// Sleep the current EE thread for the given number of bus clock cycles using
// a one-shot timer alarm. Returns non-zero if no alarm could be armed.
static int ps2go_sleep_alarm(unsigned long long cycles) {
	int id = SetTimerAlarm(cycles, ps2go_wake, (void *)(long)GetThreadId());
	if (id < 0) {
		return id;
	}
	SleepThread();
	ReleaseTimerAlarm(id);
	return 0;
}
extern void *EndOfHeap(void);
extern int GetThreadId(void);
extern int ReferThreadStatus(int thread_id, ee_thread_status_t *info);

// The EE thread's stack as the kernel set it up for crt0, as base and end
// addresses (0 on failure). Returned by value: preinit runs before the Go
// heap exists, and a Go local whose address is passed to C would be
// heap-allocated.
static unsigned int ps2go_stack_base(void) {
	ee_thread_status_t st;
	if (ReferThreadStatus(GetThreadId(), &st) < 0) {
		return 0;
	}
	return (unsigned int)st.stack;
}
static unsigned int ps2go_stack_end(void) {
	ee_thread_status_t st;
	if (ReferThreadStatus(GetThreadId(), &st) < 0) {
		return 0;
	}
	return (unsigned int)st.stack + (unsigned int)st.stack_size;
}

// Fatal messages also go to the TV when the program links ps2sdk's libdebug
// (weak: nothing is pulled in otherwise; link with -Wl,-u,scr_printf to make
// sure it is).
extern void init_scr(void) __attribute__((weak));
extern void scr_printf(const char *fmt, ...) __attribute__((weak));
static void ps2go_screen(const char *fmt, unsigned int a, unsigned int b, unsigned int c) {
	static int inited;
	if (!init_scr || !scr_printf) {
		return;
	}
	if (!inited) {
		init_scr();
		inited = 1;
	}
	scr_printf(fmt, a, b, c);
}
extern void sio_init(unsigned int baudrate, unsigned char lcr_ueps, unsigned char lcr_upen, unsigned char lcr_usbl, unsigned char lcr_umode);
extern int sio_putc(int c);

*/
import "C"
import "unsafe"

func initUART() {
	// The EE serial port (SIO). Emulators log it; on hardware it needs the
	// debug connector. It works before the SIF/IOP is up, unlike printf.
	C.sio_init(38400, 0, 0, 0, 0)
}

func putchar(c byte) {
	C.sio_putc(C.int(c))
}

func getchar() byte {
	// UART is not supported.
	return 0
}

func buffered() int {
	// UART is not supported.
	return 0
}

func sleepWDT(period uint8) {
	// TODO
}

func exit(code int) {
	C._exit(C.int(code))
}

// abort ends the program after a fatal error (the message has already been
// printed): say so on the serial port and park the thread, keeping memory
// intact for a debugger or the test harness.
func abort() {
	printstring("runtime: abort\n")
	for {
		C.SleepThread()
	}
}

// timeUnit is EE bus clock cycles, as counted by GetTimerSystemTime.
const busClockHz = 147456000

// The conversions split the value so that the multiplication never overflows
// int64 (ticks*1e9 would after about a minute).
func ticksToNanoseconds(ticks timeUnit) int64 {
	t := int64(ticks)
	return t/busClockHz*1e9 + (t%busClockHz)*1e9/busClockHz
}

func nanosecondsToTicks(ns int64) timeUnit {
	return timeUnit(ns/1e9*busClockHz + (ns%1e9)*busClockHz/1e9)
}

// sleepTicks waits for d bus clock cycles. The scheduler calls it on the
// system stack when no goroutine is runnable. Short waits spin; longer ones
// arm a timer alarm that wakes this EE thread and sleep the thread, so the
// CPU idles instead of polling. The loop guards against an early wakeup.
func sleepTicks(d timeUnit) {
	const spinThreshold = 2000 // ~14 us
	end := ticks() + d
	for {
		now := ticks()
		if now >= end {
			return
		}
		remaining := end - now
		if remaining < spinThreshold {
			continue
		}
		if C.ps2go_sleep_alarm(C.ulonglong(remaining)) != 0 {
			// No alarm available: fall back to polling.
			continue
		}
	}
}

func ticks() timeUnit {
	return timeUnit(C.GetTimerSystemTime())
}

//export main
func main() {
	preinit()
	run()
	exit(0)
}

// Address handed out for zero-size allocations; must never be dereferenced.
// Low memory belongs to the kernel (exception vectors), which is never a Go
// object.
const zeroSizeAllocPtr uintptr = 16

// The Go heap bounds (baremetal_memory.go linker symbols), exported so that
// tools and tests can report them.
var (
	HeapStart uintptr
	HeapEnd   uintptr
)

// The EE memory map is decided at link time (see the --defsym flags in the
// demos Makefile), from low to high addresses:
//
//	kernel | program (.text .. _end) | libc heap | Go heap | stack
//
// crt0 caps the libc heap at _end + _heap_size (SetupHeap; sbrk refuses to
// grow past EndOfHeap), the Go heap is [_heap_start, _heap_end), and the
// kernel puts the stack of _stack_size bytes below the top page of RAM, which
// it keeps for itself; _stack_top must be that stack's top since the GC scans
// up to it. preinit only checks that the link agrees with the kernel, so a
// wrong split fails loudly instead of corrupting memory.
func preinit() {
	// crt0 has already cleared .bss.
	initUART()
	// No allocation here: the Go heap does not exist yet.
	libcEnd := uintptr(C.EndOfHeap())
	stackBase, stackEnd := uintptr(C.ps2go_stack_base()), uintptr(C.ps2go_stack_end())
	switch {
	case stackBase == 0:
		layoutFatal("cannot read the EE thread's stack from the kernel\x00", 0, 0, 0)
	case heapStart%16 != 0:
		layoutFatal("Go heap start 0x%08x is not 16-byte aligned\x00", heapStart, 0, 0)
	case heapStart < libcEnd:
		layoutFatal("libc heap ends at 0x%08x, above the Go heap start 0x%08x (LIBC_HEAP too big or unlimited)\x00", libcEnd, heapStart, 0)
	case heapEnd <= heapStart:
		layoutFatal("Go heap end 0x%08x is not above its start 0x%08x (LIBC_HEAP too big)\x00", heapEnd, heapStart, 0)
	case heapEnd > stackBase:
		layoutFatal("Go heap end 0x%08x overlaps the stack 0x%08x-0x%08x\x00", heapEnd, stackBase, stackEnd)
	case stackTop != stackEnd:
		layoutFatal("linked stack top 0x%08x differs from the kernel's 0x%08x\x00", stackTop, stackEnd, 0)
	}
	HeapStart = heapStart
	HeapEnd = heapEnd
}

// layoutFatal reports a memory map mismatch on the serial port and on the TV
// (libdebug, when linked), with the whole map, and stops. msg is a printf
// format with up to three 0x%08x fields and a trailing NUL.
func layoutFatal(msg string, a, b, c uintptr) {
	const summary = "program end .. libc heap end 0x%08x | Go heap 0x%08x-0x%08x | stack\x00"
	libcEnd := uintptr(C.EndOfHeap())
	for _, line := range []struct {
		fmt     string
		a, b, c uintptr
	}{
		{"runtime: bad memory layout (see the LIBC_HEAP split in the Makefile):\n\x00", 0, 0, 0},
		{msg, a, b, c},
		{"\n\x00", 0, 0, 0},
		{summary, libcEnd, heapStart, heapEnd},
		{"\n\x00", 0, 0, 0},
	} {
		C.ps2go_screen((*C.char)(unsafe.Pointer(unsafe.StringData(line.fmt))), C.uint(line.a), C.uint(line.b), C.uint(line.c))
		printfmt(line.fmt[:len(line.fmt)-1], line.a, line.b, line.c)
	}
	abort()
}

// printfmt prints fmt on the serial port, substituting up to three 0x%08x
// fields (the only directive layoutFatal uses).
func printfmt(fmt string, a, b, c uintptr) {
	args := [3]uintptr{a, b, c}
	n := 0
	for i := 0; i < len(fmt); i++ {
		if fmt[i] == '%' && i+4 < len(fmt) && fmt[i+1] == '0' && fmt[i+2] == '8' && fmt[i+3] == 'x' {
			if n < 3 {
				for shift := 28; shift >= 0; shift -= 4 {
					putchar("0123456789abcdef"[(args[n]>>uint(shift))&0xf])
				}
				n++
			}
			i += 3
			continue
		}
		putchar(fmt[i])
	}
}
