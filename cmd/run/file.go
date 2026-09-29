package run

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoFile is returned when the php arguments do not start with a file.
var ErrNoFile = errors.New("no PHP file given — usage: pvm run [-v version | version] <file> [args...]")

// CheckFile makes sure the php arguments start with an existing file,
// given directly or with -f/--file, so run only ever runs a script.
func CheckFile(rest []string, dir string) error {
	if len(rest) == 0 {
		return ErrNoFile
	}

	file := rest[0]
	if file == "-f" || file == "--file" {
		if len(rest) < 2 {
			return ErrNoFile
		}
		file = rest[1]
	} else if strings.HasPrefix(file, "-") {
		return ErrNoFile
	}

	path := file
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("PHP file %s not found", file)
	}
	if fi.IsDir() {
		return fmt.Errorf("%s is a directory, not a PHP file", file)
	}
	return nil
}
