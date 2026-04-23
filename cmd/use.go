package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func printPathHint(base string, out io.Writer) {
	pvmBin := filepath.Join(base, "bin")
	entries := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))

	inPath := false
	isFirst := false
	for i, e := range entries {
		if e == pvmBin {
			inPath = true
			isFirst = (i == 0)
			break
		}
	}

	if !inPath {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "⚠  pvm is not in your PATH — 'php' still points to the system binary.")
		fmt.Fprintln(out, "   Run this now to activate it in your current shell:")
		fmt.Fprintf(out, "     export PATH=\"%s:$PATH\"\n", pvmBin)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "   To make it permanent, add to ~/.zshrc or ~/.bashrc:")
		fmt.Fprintf(out, "     export PATH=\"%s:$PATH\"\n", pvmBin)
		fmt.Fprintln(out, "   Then restart your shell or run: source ~/.zshrc")
	} else if !isFirst {
		fmt.Fprintf(out, "\n⚠  %s is in PATH but not first — system PHP may shadow it.\n", pvmBin)
	}
}
