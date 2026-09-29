// Package use holds the helpers of pvm use.
package use

import (
	"fmt"
	"io"

	"github.com/rejmann/pvm/internal/pvm"
)

// Arg returns the version to switch to: the explicit argument, or the one
// declared in the nearest .php-version when none is given.
func Arg(args []string, dir string, out io.Writer) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	v, path, err := pvm.ProjectVersion(dir)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(out, "Found %s with version %s.\n", path, v)
	return v, nil
}
