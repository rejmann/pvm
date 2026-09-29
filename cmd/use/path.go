package use

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rejmann/pvm/internal/home"
)

// PrintPathHint tells how to put the shim directory on PATH, unless it
// already is.
func PrintPathHint(out io.Writer, h *home.Dir) {
	managed := h.ShimDir()

	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if strings.EqualFold(p, managed) {
			return
		}
	}

	printPathSetup(out, managed)
}
