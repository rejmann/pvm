package selfupgrade

import (
	"io"

	"github.com/rejmann/pvm/internal/home"
)

// FallbackDir is empty on Windows: pvm is installed per user
// (%LOCALAPPDATA%\Programs\pvm), so its directory is always writable.
func FallbackDir(*home.Dir) string {
	return ""
}

// elevatedHint tells how to re-run command with the rights it lacked.
func elevatedHint(command string) string {
	return "re-run " + command + " from a terminal opened as Administrator"
}

// Moved is never called on Windows, which has no FallbackDir.
func Moved(*home.Dir, string, string, io.Writer) error {
	return nil
}
