//go:build linux || darwin

package installer

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseScanDir(t *testing.T) {
	out := "Configuration File (php.ini) Path: /etc/php/8.3/cli\n" +
		"Scan for additional .ini files in: /etc/php/8.3/cli/conf.d\n"
	if dir, err := parseScanDir(out); err != nil || dir != "/etc/php/8.3/cli/conf.d" {
		t.Errorf("parseScanDir = (%q, %v)", dir, err)
	}
	if _, err := parseScanDir("Scan for additional .ini files in: (none)\n"); err == nil {
		t.Error("parseScanDir accepted (none)")
	}
}

func TestSetConfdEnabled(t *testing.T) {
	confd := t.TempDir()
	writeFile(t, filepath.Join(confd, "20-xdebug.ini"), "; priority=20\nzend_extension=xdebug.so\n")
	writeFile(t, filepath.Join(confd, "20-intl.ini"), "extension=intl.so\n")

	bin := filepath.Join(t.TempDir(), "php")
	writeFile(t, bin, "#!/bin/sh\necho 'Scan for additional .ini files in: "+confd+"'\n")
	if err := os.Chmod(bin, 0755); err != nil {
		t.Fatal(err)
	}

	if err := setConfdEnabled(bin, []string{"Xdebug"}, false, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(confd, "20-xdebug.ini.disabled")); err != nil {
		t.Errorf("xdebug not disabled: %v", err)
	}
	// Disabling twice is a no-op.
	if err := setConfdEnabled(bin, []string{"xdebug"}, false, io.Discard); err != nil {
		t.Fatal(err)
	}

	if err := setConfdEnabled(bin, []string{"xdebug", "intl"}, true, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"20-xdebug.ini", "20-intl.ini"} {
		if _, err := os.Stat(filepath.Join(confd, f)); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}

	err := setConfdEnabled(bin, []string{"redis"}, true, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "loads redis") {
		t.Errorf("error = %v, want redis reported", err)
	}
}
