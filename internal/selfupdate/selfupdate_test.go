package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func tarGz(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0755, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func zipArchive(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// newTestUpdater serves /releases/latest with tag and every archive in assets
// under /download/<tag>/<asset>.
func newTestUpdater(t *testing.T, goos, tag string, assets map[string][]byte) *Updater {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"` + tag + `"}`))
	})
	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		data, ok := assets[strings.TrimPrefix(r.URL.Path, "/download/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &Updater{
		APIURL:      srv.URL,
		DownloadURL: srv.URL + "/download",
		Client:      srv.Client(),
		GOOS:        goos,
		GOARCH:      "amd64",
	}
}

func TestLatestTag(t *testing.T) {
	u := newTestUpdater(t, "linux", "v1.2.0", nil)
	got, err := u.LatestTag(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != "v1.2.0" {
		t.Errorf("LatestTag = %q, want v1.2.0", got)
	}
}

func TestAssetName(t *testing.T) {
	tests := map[[2]string]string{
		{"linux", "amd64"}:   "pvm-linux-amd64.tar.gz",
		{"darwin", "arm64"}:  "pvm-darwin-arm64.tar.gz",
		{"windows", "amd64"}: "pvm-windows-amd64.zip",
	}
	for in, want := range tests {
		if got := AssetName(in[0], in[1]); got != want {
			t.Errorf("AssetName(%s, %s) = %q, want %q", in[0], in[1], got, want)
		}
	}
}

func TestDownload(t *testing.T) {
	bin := []byte("new pvm binary")

	t.Run("tar.gz", func(t *testing.T) {
		u := newTestUpdater(t, "linux", "v1.2.0", map[string][]byte{
			"v1.2.0/pvm-linux-amd64.tar.gz": tarGz(t, "pvm", bin),
		})
		got, err := u.Download(context.Background(), "v1.2.0")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, bin) {
			t.Errorf("Download = %q, want %q", got, bin)
		}
	})

	t.Run("zip", func(t *testing.T) {
		u := newTestUpdater(t, "windows", "v1.2.0", map[string][]byte{
			"v1.2.0/pvm-windows-amd64.zip": zipArchive(t, "pvm.exe", bin),
		})
		got, err := u.Download(context.Background(), "v1.2.0")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, bin) {
			t.Errorf("Download = %q, want %q", got, bin)
		}
	})

	t.Run("missing build", func(t *testing.T) {
		u := newTestUpdater(t, "linux", "v1.2.0", nil)
		_, err := u.Download(context.Background(), "v1.2.0")
		if err == nil || !strings.Contains(err.Error(), "has no build for linux/amd64") {
			t.Errorf("error = %v, want missing build", err)
		}
	})

	t.Run("binary not in archive", func(t *testing.T) {
		u := newTestUpdater(t, "linux", "v1.2.0", map[string][]byte{
			"v1.2.0/pvm-linux-amd64.tar.gz": tarGz(t, "README.md", bin),
		})
		_, err := u.Download(context.Background(), "v1.2.0")
		if err == nil || !strings.Contains(err.Error(), "pvm not found in archive") {
			t.Errorf("error = %v, want pvm not found", err)
		}
	})
}

func TestReplace(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "pvm")
	if err := os.WriteFile(exe, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := Replace(exe, []byte("new")); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Errorf("binary = %q, want new", got)
	}
	fi, err := os.Stat(exe)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0755 {
		t.Errorf("mode = %v, want 0755", fi.Mode().Perm())
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".pvm-upgrade-") {
			t.Errorf("temp file %s left behind", e.Name())
		}
	}

	RemoveOld(exe)
	entries, _ = os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("dir has %d entries after RemoveOld, want only the binary", len(entries))
	}
}

func TestReplacePermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a non-root Unix user")
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "pvm")
	if err := os.WriteFile(exe, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0755) })

	err := Replace(exe, []byte("new"))
	if !errors.Is(err, ErrPermission) {
		t.Errorf("error = %v, want ErrPermission", err)
	}
}

func TestSameVersion(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"v1.2.0", "v1.2.0", true},
		{"1.2.0", "v1.2.0", true},
		{"v1.2.0", "v1.3.0", false},
		{"dev", "v1.2.0", false},
		{"v1.2.0-3-gabc1234", "v1.2.0", false},
	}
	for _, tt := range tests {
		if got := SameVersion(tt.a, tt.b); got != tt.want {
			t.Errorf("SameVersion(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
