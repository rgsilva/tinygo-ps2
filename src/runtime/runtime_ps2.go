//go:build ps2

package runtime

/*
// ps2sdk system timer: 64-bit count of EE bus clock cycles (147.456 MHz),
// kept by libkernel with the T2 overflow interrupt. Referencing it makes
// crt0's _InitSys start the timer (weak _ps2sdk_init_timer hook).
extern unsigned long long GetTimerSystemTime(void);

extern void _exit(int status);
extern int SleepThread(void);
extern int GetThreadId(void);
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
extern void* malloc(unsigned int size);
extern void free(void *ptr);
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
	preexit()
	exit(0)
}

const (
	memSize = uint(24 * 1024 * 1024)
)

// Address handed out for zero-size allocations; must never be dereferenced.
// Low memory belongs to the kernel (exception vectors), which is never a Go
// object.
const zeroSizeAllocPtr uintptr = 16

var (
	goMemoryAddr uintptr
)

func preinit() {
	// NOTE: no need to clear .bss and other memory areas as crt0 is already doing that in __start.

	// Since we're loading into whatever ps2dev kernel thingy that exists, it's safer for us to do
	// a proper malloc before proceeding. This guarantees that the heap location is ours. We will
	// need to free it later on though.

	goMemoryAddr = uintptr(unsafe.Pointer(C.malloc(C.uint(memSize))))
	heapStart = goMemoryAddr
	heapEnd = goMemoryAddr + uintptr(memSize)
	HeapStart = heapStart
	HeapEnd = heapEnd
	initUART()
}

func preexit() {
	C.free(unsafe.Pointer(heapStart))
}
