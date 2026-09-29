//go:build !windows

package selfupgrade

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/process"
	"github.com/rejmann/pvm/internal/shim"
)

// FallbackDir is where pvm installs itself when its own directory is not
// writable (an old install in e.g. /usr/local/bin): the pvm home's bin
// directory, which is the user's, so no sudo is needed.
func FallbackDir(h *home.Dir) string {
	return h.ShimDir()
}

// elevatedHint tells what to do when pvm could not write its binary at all.
func elevatedHint(string) string {
	return "pvm could not write its binary; check the permissions of ~/.pvm/bin"
}

// Moved finishes an upgrade that installed pvm at exe, in h's bin directory,
// instead of over old: the php shim is pointed at exe, and the user is told
// how to make exe the pvm that runs.
func Moved(h *home.Dir, old, exe string, out io.Writer) error {
	if _, err := os.Stat(filepath.Join(h.ShimDir(), "php")); err == nil {
		if err := shim.WriteShim(h, exe); err != nil {
			return fmt.Errorf("point the php shim at %s: %w", exe, err)
		}
	}
	fmt.Fprintf(out, "The old %s is no longer needed and can be deleted.\n", old)
	if process.LookPath("pvm", os.Getenv("PATH"), "") != exe {
		fmt.Fprintf(out, "Put %s first in your PATH so this pvm runs:\n", filepath.Dir(exe))
		fmt.Fprintf(out, "  export PATH=\"%s:$PATH\"\n", filepath.Dir(exe))
	}
	return nil
}
