/*
 * PlayStation 2 (EE) linker script for TinyGo: the ps2sdk linkfile
 * (ps2sdk/ee/startup/linkfile) plus what the Go runtime needs. Linked with
 * the ps2sdk gcc (see targets/ps2.json); the libraries and crt files come
 * from the SDK.
 */
/*
# _____     ___ ____     ___ ____
#  ____|   |    ____|   |        | |____|
# |     ___|   |____ ___|    ____| |    \    PS2DEV Open Source Project.
#-----------------------------------------------------------------------
# Copyright 2001-2004, ps2dev - http://www.ps2dev.org
# Licenced under Academic Free License version 2.0
# Review ps2sdk README & LICENSE files for further details.
#
# Linkfile script for ee-ld
*/

ENTRY(__start);

PHDRS {
  text PT_LOAD;
}

SECTIONS {
	.text 0x00100000: {
		_ftext = . ;
		*(.text)
		*(.text.*)
		*(.gnu.linkonce.t*)
		KEEP(*(.init))
		KEEP(*(.fini))
		QUAD(0)
	} :text

	PROVIDE(_etext = .);
	PROVIDE(etext = .);

	/* Global/static constructors and deconstructors. */
	.ctors ALIGN(16): {
		KEEP(*crtbegin*.o(.ctors))
		KEEP(*(EXCLUDE_FILE(*crtend*.o) .ctors))
		KEEP(*(SORT(.ctors.*)))
		KEEP(*(.ctors))
	}
	.dtors ALIGN(16): {
		KEEP(*crtbegin*.o(.dtors))
		KEEP(*(EXCLUDE_FILE(*crtend*.o) .dtors))
		KEEP(*(SORT(.dtors.*)))
		KEEP(*(.dtors))
	}

	.reginfo : { *(.reginfo) }

	/* Go globals: the range the GC scans for roots. TinyGo puts every
	   mutable global that may hold a pointer in .go.data (gc-globals-section
	   in targets/ps2.json); the program has a single load segment, so the
	   zero-initialized ones are file-backed too. */
	.go.data ALIGN(16): {
		_globals_start = . ;
		*(.go.data .go.data.*)
		. = ALIGN(16);
		_globals_end = . ;
	}

	.data ALIGN(128): {
		*(.ps2go.noscan)
		_fdata = . ;
		*(.data)
		*(.data.*)
		*(.gnu.linkonce.d*)
		SORT(CONSTRUCTORS)
	}

	/* Static data.  */
	.rodata ALIGN(128): {
		*(.rodata)
		*(.rodata.*)
		*(.gnu.linkonce.r*)
	}

	.rdata ALIGN(128): { *(.rdata) }
	.gcc_except_table ALIGN(128): { *(.gcc_except_table) }

	_gp = ALIGN(128) + 0x7ff0;
	.lit4 ALIGN(128): { *(.lit4) }
	.lit8 ALIGN(128): { *(.lit8) }

	.sdata ALIGN(128): {
		*(.sdata)
		*(.sdata.*)
		*(.gnu.linkonce.s*)
	}

	_edata = .;
	PROVIDE(edata = .);

	/* Uninitialized data.  */
	.sbss ALIGN(128) : {
		_fbss = . ;
		*(.sbss)
		*(.sbss.*)
		*(.gnu.linkonce.sb*)
		*(.scommon)
	}

	.bss ALIGN(128) : {
		*(.bss)
		*(.bss.*)
		*(.gnu.linkonce.b*)
		*(COMMON)
	}
	_end_bss = .;

	_end = . ;
	PROVIDE(end = .);

	.spad 0x70000000: {
		*(.spad)
	}

	/* Symbols needed by crt0.s.  */
	PROVIDE(_stack = -1);
	PROVIDE(_stack_size = 128 * 1024);

	/* Memory map: program | libc heap | Go heap | stack. crt0 caps the libc
	   heap at _end+_heap_size (SetupHeap) and asks the kernel for a stack of
	   _stack_size bytes below the top page of RAM, which the kernel keeps;
	   the Go heap fills the gap. Override the libc heap size per program
	   with --defsym=_heap_size=N (for example from a #cgo LDFLAGS line);
	   the runtime checks all this against the kernel at boot. */
	PROVIDE(_heap_size = 4 * 1024 * 1024);
	_heap_start = (_end + _heap_size + 15) & ~15;
	_stack_top = 0x02000000 - 0x1000;
	_heap_end = _stack_top - _stack_size;
}
