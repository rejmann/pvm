package installer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rejmann/pvm/internal/phpext"
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

// iniExtension parses a php.ini line that loads an extension
// (extension=php_zip.dll, zend_extension=opcache, ;extension=curl...) into
// the extension's name and whether the line is commented out.
func iniExtension(line string) (name string, commented, ok bool) {
	line = strings.TrimSpace(line)
	line, commented = strings.CutPrefix(line, ";")
	key, value, found := strings.Cut(strings.TrimSpace(line), "=")
	key = strings.TrimSpace(key)
	if !found || (key != "extension" && key != "zend_extension") {
		return "", false, false
	}
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if value == "" || strings.ContainsAny(value, " ;") {
		return "", false, false
	}
	value = filepath.Base(strings.ReplaceAll(value, `\`, "/"))
	value = strings.TrimSuffix(strings.TrimSuffix(strings.TrimPrefix(value, "php_"), ".dll"), ".so")
	return strings.ToLower(value), commented, true
}

// enableIniExtensions enables exts in installDir\php.ini (written first when
// missing): each one whose DLL ships in ext\ and is not enabled yet gets its
// commented-out line (e.g. the template's ;extension=curl) uncommented, or an
// extension= line appended. Extensions without a DLL are reported, since
// Windows builds compile many in (dom, xml...) and PECL ones are not bundled.
func enableIniExtensions(installDir string, exts []string) error {
	if err := writePHPIni(installDir); err != nil {
		return err
	}
	iniPath := filepath.Join(installDir, "php.ini")
	data, err := os.ReadFile(iniPath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	enabled := map[string]bool{}
	disabledAt := map[string]int{}
	for i, line := range lines {
		name, commented, ok := iniExtension(line)
		switch {
		case !ok:
		case !commented:
			enabled[name] = true
		case disabledAt[name] == 0:
			disabledAt[name] = i + 1
		}
	}

	var add, missing []string
	changed := false
	for _, ext := range exts {
		ext = phpext.Name(ext)
		if enabled[ext] {
			continue
		}
		if _, err := os.Stat(filepath.Join(installDir, "ext", "php_"+ext+".dll")); err != nil {
			missing = append(missing, ext)
			continue
		}
		enabled[ext] = true
		if at := disabledAt[ext]; at > 0 {
			line := lines[at-1]
			lines[at-1] = strings.Replace(line, ";", "", 1)
			changed = true
			continue
		}
		add = append(add, "extension=php_"+ext+".dll")
	}

	if changed || len(add) > 0 {
		out := strings.Join(lines, "\n")
		if len(add) > 0 {
			out = strings.TrimRight(out, "\r\n") + "\n\n" + strings.Join(add, "\n") + "\n"
		}
		if err := os.WriteFile(iniPath, []byte(out), 0644); err != nil {
			return err
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("no DLL for %s in %s", strings.Join(missing, ", "), filepath.Join(installDir, "ext"))
	}
	return nil
}

// disableIniExtensions stops installDir\php.ini from loading exts by
// commenting their lines out, so enableIniExtensions can bring them back.
// It returns the extensions php.ini has no line for, enabled or not: the
// ones compiled into PHP, or not there at all.
func disableIniExtensions(installDir string, exts []string) (missing []string, err error) {
	iniPath := filepath.Join(installDir, "php.ini")
	data, err := os.ReadFile(iniPath)
	if err != nil {
		return nil, err
	}

	want := map[string]bool{}
	for _, ext := range exts {
		want[phpext.Name(ext)] = true
	}
	found := map[string]bool{}
	changed := false
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		name, commented, ok := iniExtension(line)
		if !ok || !want[name] {
			continue
		}
		found[name] = true
		if !commented {
			lines[i] = ";" + strings.TrimLeft(line, " \t")
			changed = true
		}
	}

	for _, ext := range exts {
		if name := phpext.Name(ext); !found[name] && !slices.Contains(missing, name) {
			missing = append(missing, name)
		}
	}
	if changed {
		if err := os.WriteFile(iniPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return nil, err
		}
	}
	return missing, nil
}
