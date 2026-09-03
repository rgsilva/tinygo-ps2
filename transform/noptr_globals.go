package transform

import (
	"tinygo.org/x/go-llvm"
)

// MoveNoPointerGlobals moves every mutable global whose type cannot contain a
// pointer into the given section, and returns how many it moved. Constants
// are left alone (they end up in read-only sections anyway), as are globals
// that already have a section and declarations.
//
// This is meant for targets whose conservative GC scans the globals range
// linearly and whose heap sits at small addresses (the PS2: RAM starts at
// address 0), where an integer such as an allocation counter or a table of
// float constants looks exactly like a heap pointer and pins an object for
// the life of the program. The target's linker script places the section
// outside the scanned range. A uintptr global then no longer keeps an object
// alive, which matches Go's rules.
func MoveNoPointerGlobals(mod llvm.Module, section string) int {
	moved := 0
	for global := mod.FirstGlobal(); !global.IsNil(); global = llvm.NextGlobal(global) {
		if global.IsDeclaration() || global.IsGlobalConstant() || global.Section() != "" {
			continue
		}
		if global.Linkage() != llvm.InternalLinkage && global.Linkage() != llvm.PrivateLinkage {
			// Externally visible globals may be C variables with layouts we
			// should not second-guess (and LLVM's own metadata lists).
			continue
		}
		if typeMayContainPointer(global.GlobalValueType()) {
			continue
		}
		global.SetSection(section)
		moved++
	}
	return moved
}

// typeMayContainPointer reports whether a value of this LLVM type can hold a
// pointer anywhere inside it.
func typeMayContainPointer(t llvm.Type) bool {
	switch t.TypeKind() {
	case llvm.PointerTypeKind:
		return true
	case llvm.StructTypeKind:
		for _, elem := range t.StructElementTypes() {
			if typeMayContainPointer(elem) {
				return true
			}
		}
		return false
	case llvm.ArrayTypeKind, llvm.VectorTypeKind:
		return typeMayContainPointer(t.ElementType())
	default:
		return false
	}
}
