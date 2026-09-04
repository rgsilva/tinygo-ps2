//go:build gc.conservative || gc.precise

package runtime

import "unsafe"

// A faster sweep and findHead for the block allocator (gc_blocks.go),
// enabled with gcIndexedSweep, for targets with big heaps:
//
//   - the sweep handles a whole 32-bit word of block states at a time and
//     starts below the highest block allocated since the last sweep, so an
//     idle program's collection costs almost nothing;
//   - a head index (the lowest head in every group of blocksPerHeadGroup
//     blocks) and a one-entry cache make a pointer into the middle of a big
//     object cheap to resolve, instead of walking its tail blocks one by one.
//
// With gcIndexedSweep false none of this is compiled in.

// The sweep looks at a whole state word at a time where it can.
const (
	stateWordBlocks   = blocksPerStateByte * unsafe.Sizeof(uint32(0)) // blocks per state word
	stateWordLowMask  = uint32(blockStateEach) * 0x01010101           // the low bit of every block
	stateWordAllTails = uint32(blockStateByteAllTails) * 0x01010101
)

// The head index: for every group of blocksPerHeadGroup blocks, the offset of
// the lowest head block in the group plus one (0: no head in the group). It
// is kept exact: alloc records new heads, sweep rebuilds it from the
// surviving ones, and blocks are only ever freed by sweep.
const blocksPerHeadGroup = 512

var (
	headIndexStart unsafe.Pointer // right after the block states
	gcHighWater    gcBlock        // the block just past the highest one allocated since the last sweep

	// findHead also remembers the last head it found and the lowest block
	// it has seen belonging to that object (blocks between are its tails):
	// a program holding many pointers into one big object hits this with a
	// compare. Reset by sweep, the only place that frees blocks.
	headCacheHead gcBlock
	headCacheLow  gcBlock
	headCacheOK   bool
)

// headIndexSize is the metadata to reserve for the index, for a heap of
// totalSize bytes.
func headIndexSize(totalSize uintptr) uintptr {
	return totalSize/(bytesPerBlock*blocksPerHeadGroup)*2 + 4*unsafe.Sizeof(uint64(0))
}

// placeHeadIndex puts the index right after the block states (8-byte
// aligned: findHead reads four entries at a time).
func placeHeadIndex(numBlocks uintptr) {
	headIndexStart = unsafe.Add(metadataStart, ((numBlocks+blocksPerStateByte-1)/blocksPerStateByte+7)&^7)
	if gcAsserts && uintptr(headIndexStart)+((numBlocks+blocksPerHeadGroup-1)/blocksPerHeadGroup*2+7)&^7 > heapEnd {
		runtimeFatal("gc: head index does not fit")
	}
}

// stateWordBelow returns the state word covering the stateWordBlocks blocks
// below b. b must be a multiple of stateWordBlocks.
func (b gcBlock) stateWordBelow() *uint32 {
	return (*uint32)(unsafe.Add(metadataStart, (b-gcBlock(stateWordBlocks))/blocksPerStateByte))
}

func headIndexEntry(g gcBlock) *uint16 {
	return (*uint16)(unsafe.Add(headIndexStart, g*2))
}

// recordHead notes that b is a head block.
func (b gcBlock) recordHead() {
	e := headIndexEntry(b / blocksPerHeadGroup)
	if off := uint16(b%blocksPerHeadGroup) + 1; *e == 0 || off < *e {
		*e = off
	}
}

// noteAllocatedHead is called by alloc for the head block of a new object.
func noteAllocatedHead(b gcBlock) {
	if !gcIndexedSweep {
		return
	}
	if b+1 > gcHighWater {
		gcHighWater = b + 1
	}
	b.recordHead()
}

// clearHeadIndex empties the head index (sweep rebuilds it).
func clearHeadIndex() {
	memzero(headIndexStart, (uintptr((endBlock+blocksPerHeadGroup-1)/blocksPerHeadGroup)*2+7)&^7)
}

// rebuildHeadIndex recomputes the head index from the block states.
func rebuildHeadIndex() {
	clearHeadIndex()
	for b := gcBlock(0); b < endBlock; b++ {
		if state := b.state(); state == blockStateHead || state == blockStateMark {
			b.recordHead()
		}
	}
}

// findHeadIndexed is findHead with the head index and the cache.
func (b gcBlock) findHeadIndexed() gcBlock {
	if headCacheOK && b <= headCacheHead && b >= headCacheLow {
		return headCacheHead
	}
	start := b

	// The lowest head in b's group is the one we want when it is at or above
	// b: everything between is a tail.
	g := b / blocksPerHeadGroup
	if e := *headIndexEntry(g); e != 0 {
		if h := g*blocksPerHeadGroup + gcBlock(e-1); h >= b {
			b = h
			goto found
		}
	}

	// Otherwise walk the rest of the group, then hop to the first group
	// above that has a head.
	{
		end := (g + 1) * blocksPerHeadGroup
		if end > endBlock {
			end = endBlock
		}
		for b < end {
			// A state word (or byte) holding only tail blocks is skipped at
			// once.
			if b%gcBlock(stateWordBlocks) == 0 && b+gcBlock(stateWordBlocks) <= end &&
				*(b + gcBlock(stateWordBlocks)).stateWordBelow() == stateWordAllTails {
				b += gcBlock(stateWordBlocks)
				continue
			}
			stateByte := b.stateByte()
			if stateByte == blockStateByteAllTails {
				b += blocksPerStateByte - (b % blocksPerStateByte)
				continue
			}
			if b.stateFromByte(stateByte) != blockStateTail {
				goto found
			}
			b++
		}
		for g++; ; g++ {
			if gcAsserts && g*blocksPerHeadGroup >= endBlock {
				runtimeFatal("gc: found tail without head")
			}
			// Four empty groups at a time (the index is 8-byte aligned).
			if g%4 == 0 && *(*uint64)(unsafe.Add(headIndexStart, g*2)) == 0 {
				g += 3
				continue
			}
			if e := *headIndexEntry(g); e != 0 {
				b = g*blocksPerHeadGroup + gcBlock(e-1)
				break
			}
		}
	}

found:
	if headCacheOK && b == headCacheHead {
		if start < headCacheLow {
			headCacheLow = start
		}
	} else {
		headCacheHead, headCacheLow, headCacheOK = b, start, true
	}
	if gcAsserts {
		if b.state() != blockStateHead && b.state() != blockStateMark {
			runtimeFatal("gc: found tail without head")
		}
	}
	return b
}

// sweepIndexed is sweep with whole state words and the high-water mark.
func sweepIndexed() uintptr {
	// Discard the old free ranges list.
	freeRanges = nil

	// Scan backwards through the block metadata, starting below the highest
	// block allocated since the last sweep: nothing above it has been touched
	// (free ranges are best-fit and carved from their low end, so the big
	// range at the top is used last), and that part joins the first free
	// range without looking at its metadata.
	block := gcHighWater
	freeEnd := endBlock
	var freeBlocks uintptr
	clearHeadIndex() // rebuilt from the surviving heads below
	headCacheOK = false
	for {
		// Scan backwards until we find a marked head.
		// Free the blocks as we go.
		for block > 0 {
			if block%gcBlock(stateWordBlocks) == 0 && block >= gcBlock(stateWordBlocks) {
				// A marked head has both of its bits set. If no block in this
				// word has, all of them are freed (or already free) at once.
				w := block.stateWordBelow()
				if v := *w; v&(v>>blocksPerStateByte)&stateWordLowMask == 0 {
					if v != 0 {
						*w = 0
					}
					block -= gcBlock(stateWordBlocks)
					continue
				}
			}
			if (block - 1).state() == blockStateMark {
				break
			}
			block--
			block.free()
		}

		if freeLen := uintptr(freeEnd - block); freeLen > 0 {
			// Insert the freed blocks.
			freeBlocks += freeLen
			insertFreeRange(block.pointer(), freeLen)
		}
		if freeEnd == endBlock {
			// The first free range found from the top: everything above the
			// highest live object is free.
			gcHighWater = block
		}

		if block == 0 {
			// There are no more blocks to sweep.
			break
		}

		// Unmark the next head.
		block--
		block.unmark()
		block.recordHead()

		// Skip the tail, a word at a time inside big objects.
		for block > 0 {
			if block%gcBlock(stateWordBlocks) == 0 && block >= gcBlock(stateWordBlocks) &&
				*block.stateWordBelow() == stateWordAllTails {
				block -= gcBlock(stateWordBlocks)
				continue
			}
			if (block - 1).state() != blockStateTail {
				break
			}
			block--
		}
		freeEnd = block
	}

	if gcDebug {
		println("free ranges after sweep:")
		dumpFreeRangeCounts()
	}

	return freeBlocks * bytesPerBlock
}
