//go:build !windows

package use

import (
	"fmt"
	"io"
)

// printPathSetup tells how to put the shim directory on PATH.
func printPathSetup(out io.Writer, shimDir string) {
	fmt.Fprintf(out, "\nHint: add %s to your PATH to use this version:\n", shimDir)
	fmt.Fprintf(out, "  export PATH=\"%s:$PATH\"\n", shimDir)
}
