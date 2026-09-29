package home

import (
	"strings"
	"testing"
)

func TestPackages(t *testing.T) {
	d := New(t.TempDir())
	if got := d.Packages("8.4"); got != nil {
		t.Fatalf("Packages without file = %v", got)
	}
	if err := d.AddPackages("8.4", []string{"php8.4-zip", "php8.4-xml"}); err != nil {
		t.Fatal(err)
	}
	if err := d.AddPackages("8.4", []string{"php8.4-xml", "php8.4-intl"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(d.Packages("8.4"), ","); got != "php8.4-zip,php8.4-xml,php8.4-intl" {
		t.Errorf("packages = %s", got)
	}
	if got := d.Packages("8.3"); got != nil {
		t.Errorf("other version has packages %v", got)
	}
}

func TestSetBinary(t *testing.T) {
	d := New(t.TempDir())
	bin := install(t, d, "8.4") // real file to point at
	if err := d.RemoveVersion("8.4"); err != nil {
		t.Fatal(err)
	}

	if err := d.SetBinary("8.4", bin); err != nil {
		t.Fatal(err)
	}
	if got, err := d.Binary("8.4"); err != nil || got != bin || !d.Installed("8.4") {
		t.Errorf("Binary = (%q, %v), installed = %v", got, err, d.Installed("8.4"))
	}
}
