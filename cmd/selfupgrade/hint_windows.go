package selfupgrade

// elevatedHint tells how to re-run command with the rights it lacked.
func elevatedHint(command string) string {
	return "re-run " + command + " from a terminal opened as Administrator"
}
