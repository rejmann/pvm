//go:build windows

package selfupdate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// swap moves exe aside to exe+".old" and puts the new binary in its place:
// Windows can rename a running executable but not overwrite it. If a previous
// .old is still locked (another pvm process is running), a unique suffix is
// used instead. RemoveOld cleans them up on the next upgrade.
func swap(newBin, exe string) error {
	old := exe + ".old"
	if err := os.Remove(old); err != nil && !os.IsNotExist(err) {
		old = fmt.Sprintf("%s.old-%d", exe, time.Now().UnixNano())
	}

	if err := os.Rename(exe, old); err != nil {
		return err
	}
	if err := os.Rename(newBin, exe); err != nil {
		_ = os.Rename(old, exe)
		return err
	}
	return nil
}

// RemoveOld deletes the binaries moved aside by previous upgrades; the ones
// still in use are left for the next time.
func RemoveOld(exe string) {
	dir, base := filepath.Split(exe)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), base+".old") {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}
