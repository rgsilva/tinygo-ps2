//go:build ps2

package runtime

/*
// The CPU's COP0 Count register, running since reset.
static unsigned int ps2go_cop0_count(void) {
	unsigned int c;
	__asm__ __volatile__("mfc0 %0, $9" : "=r"(c));
	return c;
}
*/
import "C"

// The PS2 has no hardware random number generator. hardwareRand instead
// stirs what varies into a splitmix64 state: the bus clock timer, the COP0
// Count register and the wall-clock offset once the RTC has been read. That
// seeds the maps and math/rand differently on every boot of a console (and
// on every run of an emulator once the clock is synced); it is not a source
// of cryptographic entropy, so crypto/rand stays without a Reader.
var randState uint64

func hardwareRand() (n uint64, ok bool) {
	s := randState
	s ^= uint64(ticks())
	s ^= uint64(C.ps2go_cop0_count()) << 32
	s ^= uint64(timeOffset.Load())
	s += 0x9e3779b97f4a7c15
	randState = s
	z := s
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31), true
}
