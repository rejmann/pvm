package home

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

func (d *Dir) systemFile(v string) string {
	return filepath.Join(d.VersionDir(v), "system")
}

func (d *Dir) systemCheckedFile() string {
	return filepath.Join(d.Path, "system-checked")
}

// System reports whether installed version v was installed outside pvm and
// only adopted by it, so pvm must not uninstall it.
func (d *Dir) System(v string) bool {
	_, err := os.Stat(d.systemFile(v))
	return err == nil
}

// SetSystem records bin as the php binary of version v, installed outside pvm.
func (d *Dir) SetSystem(v, bin string) error {
	if err := d.SetBinary(v, bin); err != nil {
		return err
	}
	return os.WriteFile(d.systemFile(v), nil, 0644)
}

// SystemChecked reports whether pvm already looked for a PHP installed
// before it; it only does so once.
func (d *Dir) SystemChecked() bool {
	_, err := os.Stat(d.systemCheckedFile())
	return !errors.Is(err, fs.ErrNotExist)
}

// SetSystemChecked records that pvm looked for a PHP installed before it.
func (d *Dir) SetSystemChecked() error {
	if err := os.MkdirAll(d.Path, 0755); err != nil {
		return err
	}
	return os.WriteFile(d.systemCheckedFile(), nil, 0644)
}
