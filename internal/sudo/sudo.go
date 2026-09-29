// Package sudo runs the commands pvm needs root for (package managers,
// update-alternatives) the way a user expects from a terminal: the password
// is asked once, up front, and every command after it reuses sudo's cached
// credentials. As root, commands run directly and sudo is not needed.
package sudo

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// root reports whether pvm already runs as root (e.g. in a container).
func root() bool {
	return os.Geteuid() == 0
}

// Authenticate makes sure the next Command calls will not stop for a
// password: when sudo has no cached credentials, it says why pvm needs them
// on w and runs sudo -v, which asks for the password on the terminal.
func Authenticate(w io.Writer, why string) error {
	if root() {
		return nil
	}
	if _, err := exec.LookPath("sudo"); err != nil {
		return fmt.Errorf("pvm needs root to %s, but sudo was not found — run pvm as root", why)
	}
	if exec.Command("sudo", "-n", "true").Run() == nil {
		return nil
	}

	// sudo reads the password from the controlling terminal, not stdin.
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return fmt.Errorf("pvm needs administrator rights (sudo) to %s: run it from a terminal, so sudo can ask for your password, or as root", why)
	}
	tty.Close()

	fmt.Fprintf(w, "pvm needs administrator rights (sudo) to %s.\n", why)
	v := exec.Command("sudo", "-v")
	v.Stdin, v.Stdout, v.Stderr = os.Stdin, w, w
	if err := v.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return errors.New("sudo authentication failed")
		}
		return fmt.Errorf("sudo -v: %w", err)
	}
	return nil
}

// Command returns the command that runs args as root: through sudo, or
// directly when pvm is root already. Its stdin is pvm's, so sudo can still
// ask for the password if the cached credentials expired.
func Command(args ...string) *exec.Cmd {
	if !root() {
		args = append([]string{"sudo"}, args...)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	return cmd
}
