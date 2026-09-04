package transform_test

import (
	"testing"

	"github.com/tinygo-org/tinygo/transform"
	"tinygo.org/x/go-llvm"
)

func TestMoveNoPointerGlobals(t *testing.T) {
	t.Parallel()
	testTransform(t, "testdata/noptr-globals", func(mod llvm.Module) {
		if n := transform.MoveNoPointerGlobals(mod, ".noptr"); n != 3 {
			t.Errorf("moved %d globals, want 3", n)
		}
	})
}

func TestMoveScannedGlobals(t *testing.T) {
	t.Parallel()
	testTransform(t, "testdata/scanned-globals", func(mod llvm.Module) {
		transform.MoveNoPointerGlobals(mod, ".noptr")
		if n := transform.MoveScannedGlobals(mod, ".scan"); n != 3 {
			t.Errorf("moved %d globals, want 3", n)
		}
	})
}
