package cmd

import (
	"fmt"
	"io"
)

// phpInHome is true when the PHP builds live in the pvm home and so go with it.
const phpInHome = true

// printPathSetup tells how to put the shim directory on PATH.
func printPathSetup(out io.Writer, _ string) {
	fmt.Fprintf(out, "\nOne-time setup: reload your PowerShell profile to activate version switching:\n")
	fmt.Fprintf(out, "  . $PROFILE\n")
	fmt.Fprintf(out, "\nAfter that, pvm use will switch versions instantly in any new terminal.\n")
}

// elevatedHint tells how to re-run command with the rights it lacked.
func elevatedHint(command string) string {
	return "re-run " + command + " from a terminal opened as Administrator"
}

// removeBinaryHint tells how to delete a binary pvm could not remove.
func removeBinaryHint(exe string) string {
	return "delete " + exe + " from a terminal opened as Administrator"
}

// printAfterRemoval is what is left for the user after pvm self-remove.
func printAfterRemoval(out io.Writer, _ string) {
	fmt.Fprintln(out, "Open a new terminal to pick up the updated PATH.")
}
