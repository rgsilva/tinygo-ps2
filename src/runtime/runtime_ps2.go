//go:build ps2

package runtime

/*
extern unsigned int GetTimerCounter(int);

extern void _exit(int status);
extern void* malloc(unsigned int size);
extern void free(void *ptr);
extern void sio_init(unsigned int baudrate, unsigned char lcr_ueps, unsigned char lcr_upen, unsigned char lcr_usbl, unsigned char lcr_umode);
extern int sio_putc(int c);

extern long __muldi3(long a, long b);

extern long __divdi3(long a, long b);
extern unsigned long __udivdi3 (unsigned long a, unsigned long b);
extern long __moddi3(long a, long b);
extern unsigned long __umoddi3(unsigned long a, unsigned long b);

extern double __adddf3 (double a, double b); // a + b
extern double __subdf3 (double a, double b); // a - b
extern double __muldf3 (double a, double b); // a * b
extern double __divdf3 (double a, double b); // a / b
extern int __gtdf2 (double a, double b); // a > b
extern int __gedf2 (double a, double b); // a >= b
extern int __ltdf2 (double a, double b); // a < b
extern int __ledf2 (double a, double b); // a <= b
extern int __eqdf2 (double a, double b); // a == b
extern int __nedf2 (double a, double b); // a != b
extern float __truncdfsf2(double a); // float64 -> float32
extern double __extendsfdf2(float a); // float32 -> float64
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

func abort() {
	// TODO.
	for {
	}
}

func ticksToNanoseconds(ticks timeUnit) int64 {
	return int64(ticks)
}

func nanosecondsToTicks(ns int64) timeUnit {
	return timeUnit(ns)
}

// Sleep this number of ticks of nanoseconds.
func sleepTicks(d timeUnit) {
	// TODO
}

func ticks() (ticksReturn timeUnit) {
	return 0
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

func int64mul(a, b int64) int64 {
	return int64(C.__muldi3(C.long(a), C.long(b)))
}

func int64div(a, b int64) int64 {
	return int64(C.__divdi3(C.long(a), C.long(b)))
}

func uint64div(a, b uint64) uint64 {
	return uint64(C.__udivdi3(C.ulong(a), C.ulong(b)))
}

func int64mod(a, b int64) int64 {
	return int64(C.__moddi3(C.long(a), C.long(b)))
}

func uint64mod(a, b uint64) uint64 {
	return uint64(C.__umoddi3(C.ulong(a), C.ulong(b)))
}

func float64add(a, b float64) float64 {
	return float64(C.__adddf3(C.double(a), C.double(b)))
}

func float64sub(a, b float64) float64 {
	return float64(C.__subdf3(C.double(a), C.double(b)))
}

func float64mul(a, b float64) float64 {
	return float64(C.__muldf3(C.double(a), C.double(b)))
}

func float64div(a, b float64) float64 {
	return float64(C.__divdf3(C.double(a), C.double(b)))
}
func float64gt(a, b float64) bool {
	return int(C.__gtdf2(C.double(a), C.double(b))) > 0
}

func float64ge(a, b float64) bool {
	return int(C.__gedf2(C.double(a), C.double(b))) >= 0
}

func float64lt(a, b float64) bool {
	return int(C.__ltdf2(C.double(a), C.double(b))) < 0
}

func float64le(a, b float64) bool {
	return int(C.__ledf2(C.double(a), C.double(b))) <= 0
}

func float64eq(a, b float64) bool {
	return int(C.__eqdf2(C.double(a), C.double(b))) == 0
}

func float64ne(a, b float64) bool {
	return int(C.__nedf2(C.double(a), C.double(b))) != 0
}

func float64to32(a float64) float32 {
	return float32(C.__truncdfsf2(C.double(a)))
}

func float32to64(a float32) float64 {
	return float64(C.__extendsfdf2(C.float(a)))
}
