//go:build !windows

package selfremove

import (
	"fmt"
	"io"
)

// phpInHome is true when the PHP builds live in the pvm home and so go with it.
const phpInHome = false

// removeBinaryHint tells how to delete a binary pvm could not remove.
func removeBinaryHint(exe string) string {
	return "finish with: sudo rm " + exe
}

// printAfterRemoval is what is left for the user after pvm self-remove.
func printAfterRemoval(out io.Writer, shimDir string) {
	fmt.Fprintf(out, "If your shell config adds %s to PATH, remove that line.\n", shimDir)
}
