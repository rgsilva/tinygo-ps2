//go:build ps2

package runtime

// The PS2 has a big heap (tens of MB) and RAM at address 0: the sweep and
// findHead of gc_blocks_indexed.go keep collections proportional to what the
// program uses rather than to the heap size. Other targets keep upstream's
// code (and code size).
const gcIndexedSweep = true
