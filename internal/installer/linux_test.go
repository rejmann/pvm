//go:build linux

package installer

import (
	"strings"
	"testing"
)

func TestRemiExtPkg(t *testing.T) {
	for ext, want := range map[string]string{
		"xml":     "php8.4-php-xml",
		"zip":     "php8.4-php-pecl-zip",
		"mysql":   "php8.4-php-mysqlnd",
		"sqlite3": "php8.4-php-pdo",
	} {
		if got := remiExtPkg("8.4", ext); got != want {
			t.Errorf("remiExtPkg(%q) = %q, want %q", ext, got, want)
		}
	}
}

func TestParseOSRelease(t *testing.T) {
	for name, tc := range map[string]struct {
		in   string
		want osRelease
	}{
		"debian": {
			in:   "PRETTY_NAME=\"Debian GNU/Linux 13 (trixie)\"\nID=debian\nVERSION_CODENAME=trixie\n",
			want: osRelease{ID: "debian", Codename: "trixie"},
		},
		"mint": {
			in:   "ID=linuxmint\nID_LIKE=\"ubuntu debian\"\nVERSION_CODENAME=wilma\n",
			want: osRelease{ID: "linuxmint", IDLike: []string{"ubuntu", "debian"}, Codename: "wilma"},
		},
		"lmde": {
			in:   "ID=linuxmint\nID_LIKE=debian\nDEBIAN_CODENAME=bookworm\n",
			want: osRelease{ID: "linuxmint", IDLike: []string{"debian"}, Codename: "bookworm"},
		},
	} {
		got := parseOSRelease(strings.NewReader(tc.in))
		if got.ID != tc.want.ID || got.Codename != tc.want.Codename || strings.Join(got.IDLike, " ") != strings.Join(tc.want.IDLike, " ") {
			t.Errorf("%s: parseOSRelease = %+v, want %+v", name, got, tc.want)
		}
	}
}

func TestOSReleaseIs(t *testing.T) {
	mint := osRelease{ID: "linuxmint", IDLike: []string{"ubuntu", "debian"}}
	if !mint.is("ubuntu") {
		t.Error("Mint should count as Ubuntu, so it gets the PPA")
	}
	if (osRelease{ID: "debian"}).is("ubuntu") {
		t.Error("Debian is not Ubuntu")
	}
}

func TestSuryListLine(t *testing.T) {
	want := "deb [signed-by=/usr/share/keyrings/debsuryorg-archive-keyring.gpg] https://packages.sury.org/php/ trixie main\n"
	if got := suryListLine("trixie"); got != want {
		t.Errorf("suryListLine = %q, want %q", got, want)
	}
}
