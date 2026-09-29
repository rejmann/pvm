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
// the Windows builds: openssl so Composer can use HTTPS, then phpext.Base
// (xml is compiled in). Each one is enabled only when its DLL ships in ext\
// (older builds compile zip in statically).
var iniExtensions = []string{"openssl", "zip", "mbstring", "curl"}

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

// enableIniExtensions enables exts in installDir\php.ini (written first when
// missing): each one whose DLL ships in ext\ and is not enabled yet gets an
// extension= line. Extensions without a DLL are reported, since Windows
// builds compile many in (dom, xml...) and PECL ones are not bundled.
func enableIniExtensions(installDir string, exts []string) error {
	if err := writePHPIni(installDir); err != nil {
		return err
	}
	iniPath := filepath.Join(installDir, "php.ini")
	data, err := os.ReadFile(iniPath)
	if err != nil {
		return err
	}
	enabled := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		name, ok := strings.CutPrefix(strings.TrimSpace(line), "extension=")
		if !ok {
			continue
		}
		name = strings.TrimSuffix(strings.TrimPrefix(strings.Trim(name, `"`), "php_"), ".dll")
		enabled[strings.ToLower(name)] = true
	}

	var add, missing []string
	for _, ext := range exts {
		ext = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ext)), "ext-")
		if enabled[ext] {
			continue
		}
		if _, err := os.Stat(filepath.Join(installDir, "ext", "php_"+ext+".dll")); err != nil {
			missing = append(missing, ext)
			continue
		}
		enabled[ext] = true
		add = append(add, "extension=php_"+ext+".dll")
	}

	if len(add) > 0 {
		f, err := os.OpenFile(iniPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		_, err = f.WriteString("\n" + strings.Join(add, "\n") + "\n")
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return err
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("no DLL for %s in %s", strings.Join(missing, ", "), filepath.Join(installDir, "ext"))
	}
	return nil
}
