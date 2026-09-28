package installer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// iniExtensions are the extensions pvm enables in the php.ini it writes for
// the Windows builds: openssl so Composer can use HTTPS, zip so it can extract
// packages without 7z. Each one is enabled only when its DLL ships in ext\
// (older builds compile zip in statically).
var iniExtensions = []string{"openssl", "zip"}

const iniMarker = "; --- Added by pvm ---"

// writePHPIni creates installDir\php.ini from the bundled php.ini-production,
// with extension_dir and iniExtensions set. An existing php.ini is kept, so
// user edits survive.
func writePHPIni(installDir string) error {
	iniPath := filepath.Join(installDir, "php.ini")
	if _, err := os.Stat(iniPath); err == nil {
		return nil
	}

	template, err := os.ReadFile(filepath.Join(installDir, "php.ini-production"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read php.ini-production: %w", err)
	}

	ini := string(template) + "\n" + pvmIniBlock(installDir) + "\n"
	return os.WriteFile(iniPath, []byte(ini), 0644)
}

// pvmIniBlock is appended after the template, so its directives win over the
// template's defaults. extension_dir is absolute: a relative one would be
// resolved against the working directory, not php.exe's.
func pvmIniBlock(installDir string) string {
	extDir := filepath.Join(installDir, "ext")

	// Not %q: php.ini does not unescape backslashes, and Windows paths cannot
	// contain double quotes.
	lines := []string{iniMarker, `extension_dir = "` + extDir + `"`}
	for _, ext := range iniExtensions {
		dll := "php_" + ext + ".dll"
		if _, err := os.Stat(filepath.Join(extDir, dll)); err == nil {
			lines = append(lines, "extension="+dll)
		}
	}
	return strings.Join(lines, "\n")
}
