//go:build !windows

package selfupgrade

// elevatedHint tells how to re-run command with the rights it lacked.
func elevatedHint(command string) string {
	return "re-run with: sudo " + command
}
