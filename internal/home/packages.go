package home

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// packagesFile lists the extension packages pvm installed for a version (apt,
// dnf... packages or Homebrew formulas), one per line, so removing the
// version removes them too.
func (d *Dir) packagesFile(v string) string {
	return filepath.Join(d.VersionDir(v), "packages")
}

// Packages returns the extension packages recorded for version v.
func (d *Dir) Packages(v string) []string {
	f, err := os.Open(d.packagesFile(v))
	if err != nil {
		return nil
	}
	defer f.Close()

	var pkgs []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if p := strings.TrimSpace(sc.Text()); p != "" {
			pkgs = append(pkgs, p)
		}
	}
	return pkgs
}

// AddPackages records pkgs for version v, skipping those already listed.
func (d *Dir) AddPackages(v string, pkgs []string) error {
	seen := map[string]bool{}
	all := d.Packages(v)
	for _, p := range all {
		seen[p] = true
	}
	for _, p := range pkgs {
		if !seen[p] {
			seen[p] = true
			all = append(all, p)
		}
	}

	return d.writePackages(v, all)
}

// RemovePackages forgets pkgs for version v, once they are uninstalled.
func (d *Dir) RemovePackages(v string, pkgs []string) error {
	drop := map[string]bool{}
	for _, p := range pkgs {
		drop[p] = true
	}
	var keep []string
	for _, p := range d.Packages(v) {
		if !drop[p] {
			keep = append(keep, p)
		}
	}
	return d.writePackages(v, keep)
}

func (d *Dir) writePackages(v string, all []string) error {
	path := d.packagesFile(v)
	if len(all) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(all, "\n")+"\n"), 0644)
}
