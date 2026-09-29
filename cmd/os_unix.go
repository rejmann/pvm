//go:build !windows

package cmd

import (
	"fmt"
	"io"
)

// phpInHome is true when the PHP builds live in the pvm home and so go with it.
const phpInHome = false

// printPathSetup tells how to put the shim directory on PATH.
func printPathSetup(out io.Writer, shimDir string) {
	fmt.Fprintf(out, "\nHint: add %s to your PATH to use this version:\n", shimDir)
	fmt.Fprintf(out, "  export PATH=\"%s:$PATH\"\n", shimDir)
}

// elevatedHint tells how to re-run command with the rights it lacked.
func elevatedHint(command string) string {
	return "re-run with: sudo " + command
}

// removeBinaryHint tells how to delete a binary pvm could not remove.
func removeBinaryHint(exe string) string {
	return "finish with: sudo rm " + exe
}

// printAfterRemoval is what is left for the user after pvm self-remove.
func printAfterRemoval(out io.Writer, shimDir string) {
	fmt.Fprintf(out, "If your shell config adds %s to PATH, remove that line.\n", shimDir)
}
