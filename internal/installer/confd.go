//go:build linux || darwin

package installer

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rejmann/pvm/internal/phpext"
	"github.com/rejmann/pvm/internal/sudo"
)

// disabledSuffix is added to a conf.d file to turn its extension off: PHP
// only loads the *.ini files of its scan directory.
const disabledSuffix = ".disabled"

const scanDirPrefix = "Scan for additional .ini files in:"

// iniScanDir asks bin for the directory of additional .ini files it loads
// (conf.d, php.d...), where packages drop one file per extension.
func iniScanDir(bin string) (string, error) {
	out, err := exec.Command(bin, "--ini").Output()
	if err != nil {
		return "", fmt.Errorf("%s --ini: %w", bin, err)
	}
	return parseScanDir(string(out))
}

func parseScanDir(out string) (string, error) {
	for _, line := range strings.Split(out, "\n") {
		dir, ok := strings.CutPrefix(strings.TrimSpace(line), scanDirPrefix)
		if !ok {
			continue
		}
		if dir = strings.TrimSpace(dir); dir == "" || dir == "(none)" {
			break
		}
		return dir, nil
	}
	return "", errors.New("this PHP has no directory of additional .ini files")
}

// confdFile finds the file in dir that loads ext, and whether it is enabled
// (*.ini) or was disabled by pvm (*.ini.disabled).
func confdFile(dir, ext string) (path string, enabled bool, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false, err
	}
	for _, e := range entries {
		name := e.Name()
		on := strings.HasSuffix(name, ".ini")
		if e.IsDir() || !on && !strings.HasSuffix(name, ".ini"+disabledSuffix) {
			continue
		}
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if n, commented, ok := iniExtension(line); ok && !commented && n == ext {
				return path, on, nil
			}
		}
	}
	return "", false, fs.ErrNotExist
}

// confdLoads reports whether a file of the scan directory of the PHP at bin
// loads ext, enabled or disabled by pvm.
func confdLoads(bin, ext string) bool {
	dir, err := iniScanDir(bin)
	if err != nil {
		return false
	}
	_, _, err = confdFile(dir, phpext.Name(ext))
	return err == nil
}

// setConfdEnabled turns exts on or off for the PHP at bin by renaming the
// file of its scan directory that loads each one, through sudo when the
// directory belongs to root.
func setConfdEnabled(bin string, exts []string, enabled bool, w io.Writer) error {
	dir, err := iniScanDir(bin)
	if err != nil {
		return err
	}

	type rename struct{ from, to string }
	var renames []rename
	var missing []string
	for _, ext := range exts {
		ext = phpext.Name(ext)
		path, on, err := confdFile(dir, ext)
		if err != nil {
			missing = append(missing, ext)
			continue
		}
		switch {
		case on == enabled:
		case enabled:
			renames = append(renames, rename{path, strings.TrimSuffix(path, disabledSuffix)})
		default:
			renames = append(renames, rename{path, path + disabledSuffix})
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("no .ini file in %s loads %s", dir, strings.Join(missing, ", "))
	}

	for _, r := range renames {
		err := os.Rename(r.from, r.to)
		if err == nil {
			continue
		}
		if !errors.Is(err, fs.ErrPermission) {
			return err
		}
		if err := sudo.Authenticate(w, "change "+dir); err != nil {
			return err
		}
		mv := sudo.Command("mv", r.from, r.to)
		mv.Stdout, mv.Stderr = w, w
		if err := mv.Run(); err != nil {
			return fmt.Errorf("mv %s %s: %w", r.from, r.to, err)
		}
	}
	return nil
}
