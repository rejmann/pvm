package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rejmann/pvm/internal/selfupdate"
)

// newReleaseServer publishes latest as the latest release and a linux/amd64
// archive (containing "pvm <tag>") for each tag in tags.
func newReleaseServer(t *testing.T, latest string, tags ...string) *selfupdate.Updater {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"` + latest + `"}`))
	})
	for _, tag := range tags {
		archive := linuxArchive(t, []byte("pvm "+tag))
		mux.HandleFunc("/download/"+tag+"/pvm-linux-amd64.tar.gz", func(w http.ResponseWriter, r *http.Request) {
			w.Write(archive)
		})
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &selfupdate.Updater{
		APIURL:      srv.URL,
		DownloadURL: srv.URL + "/download",
		Client:      srv.Client(),
		GOOS:        "linux",
		GOARCH:      "amd64",
	}
}

func linuxArchive(t *testing.T, bin []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "pvm", Mode: 0755, Size: int64(len(bin)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	tw.Write(bin)
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func fakeExe(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "pvm")
	if err := os.WriteFile(exe, []byte("pvm current"), 0755); err != nil {
		t.Fatal(err)
	}
	return exe
}

func TestSelfUpgrade(t *testing.T) {
	tests := []struct {
		name    string
		current string
		tag     string
		check   bool
		wantBin string
		wantOut string
	}{
		{name: "upgrades to latest", current: "v1.0.0", wantBin: "pvm v1.2.0", wantOut: "pvm upgraded from v1.0.0 to v1.2.0"},
		{name: "dev build upgrades", current: "dev", wantBin: "pvm v1.2.0", wantOut: "to v1.2.0"},
		{name: "already latest", current: "v1.2.0", wantBin: "pvm current", wantOut: "pvm is already at v1.2.0."},
		{name: "check only", current: "v1.0.0", check: true, wantBin: "pvm current", wantOut: "pvm v1.2.0 is available (current: v1.0.0)"},
		{name: "explicit tag", current: "v1.2.0", tag: "v1.1.0", wantBin: "pvm v1.1.0", wantOut: "to v1.1.0"},
		{name: "tag without v", current: "v1.2.0", tag: "1.1.0", wantBin: "pvm v1.1.0", wantOut: "to v1.1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := newReleaseServer(t, "v1.2.0", "v1.1.0", "v1.2.0")
			exe := fakeExe(t)
			var out bytes.Buffer

			if err := selfUpgrade(context.Background(), u, exe, tt.current, tt.tag, tt.check, &out); err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(out.String(), tt.wantOut) {
				t.Errorf("output = %q, want it to contain %q", out.String(), tt.wantOut)
			}
			got, _ := os.ReadFile(exe)
			if string(got) != tt.wantBin {
				t.Errorf("binary = %q, want %q", got, tt.wantBin)
			}
		})
	}
}

func TestSelfUpgradeUnknownTag(t *testing.T) {
	u := newReleaseServer(t, "v1.2.0", "v1.2.0")
	exe := fakeExe(t)

	err := selfUpgrade(context.Background(), u, exe, "v1.2.0", "v9.9.9", false, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "has no build for linux/amd64") {
		t.Fatalf("error = %v, want missing build", err)
	}
	got, _ := os.ReadFile(exe)
	if string(got) != "pvm current" {
		t.Errorf("binary changed to %q after a failed upgrade", got)
	}
}
