package use

import (
	"fmt"
	"io"
)

// printPathSetup tells how to put the shim directory on PATH.
func printPathSetup(out io.Writer, _ string) {
	fmt.Fprintf(out, "\nOne-time setup: reload your PowerShell profile to activate version switching:\n")
	fmt.Fprintf(out, "  . $PROFILE\n")
	fmt.Fprintf(out, "\nAfter that, pvm use will switch versions instantly in any new terminal.\n")
}
