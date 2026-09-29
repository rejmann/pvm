package selfremove

import (
	"fmt"
	"io"
)

// phpInHome is true when the PHP builds live in the pvm home and so go with it.
const phpInHome = true

// removeBinaryHint tells how to delete a binary pvm could not remove.
func removeBinaryHint(exe string) string {
	return "delete " + exe + " from a terminal opened as Administrator"
}

// printAfterRemoval is what is left for the user after pvm self-remove.
func printAfterRemoval(out io.Writer, _ string) {
	fmt.Fprintln(out, "Open a new terminal to pick up the updated PATH.")
}
