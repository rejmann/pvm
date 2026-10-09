package installer

import (
	"archive/zip"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFirstAvailable(t *testing.T) {
	// /slow answers only after release is closed, or when its request is
	// cancelled.
	release := make(chan struct{})
	var cancelled atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("%s %s: want HEAD", r.Method, r.URL.Path)
		}
		switch r.URL.Path {
		case "/ok", "/ok2":
		case "/slow":
			select {
			case <-release:
			case <-r.Context().Done():
				cancelled.Store(true)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	urls := func(paths ...string) []string {
		for i, p := range paths {
			paths[i] = srv.URL + p
		}
		return paths
	}

	t.Run("the most wanted wins, without waiting for the rest", func(t *testing.T) {
		got, err := firstAvailable(context.Background(), srv.Client(), urls("/missing", "/ok", "/slow", "/ok2"))
		if err != nil || got != srv.URL+"/ok" {
			t.Fatalf("firstAvailable = (%q, %v), want /ok", got, err)
		}
	})

	t.Run("a slower URL still wins over the ones after it", func(t *testing.T) {
		done := make(chan string, 1)
		go func() {
			got, _ := firstAvailable(context.Background(), srv.Client(), urls("/slow", "/ok"))
			done <- got
		}()
		close(release)
		if got := <-done; got != srv.URL+"/slow" {
			t.Errorf("firstAvailable = %q, want /slow", got)
		}
	})

	t.Run("none exists", func(t *testing.T) {
		_, err := firstAvailable(context.Background(), srv.Client(), urls("/a", "/b"))
		if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
			t.Errorf("error = %v, want HTTP 404", err)
		}
		if _, err := firstAvailable(context.Background(), srv.Client(), nil); err == nil {
			t.Error("no URLs: want an error")
		}
	})

	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := firstAvailable(ctx, srv.Client(), urls("/ok")); err == nil {
			t.Error("want an error from a cancelled context")
		}
	})
}

func writeZip(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "php.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractZip(t *testing.T) {
	files := map[string]string{"php.exe": "binary", "empty/": ""}
	for i := range 200 {
		files[fmt.Sprintf("ext/dir%d/php_%d.dll", i%7, i)] = strings.Repeat("x", i)
	}

	t.Run("extracts every entry", func(t *testing.T) {
		dest := t.TempDir()
		if err := extractZip(context.Background(), writeZip(t, files), dest); err != nil {
			t.Fatal(err)
		}
		for name, want := range files {
			path := filepath.Join(dest, filepath.FromSlash(name))
			if strings.HasSuffix(name, "/") {
				if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
					t.Errorf("%s: not a directory (%v)", name, err)
				}
				continue
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != want {
				t.Errorf("%s: got %d bytes (%v), want %d", name, len(got), err, len(want))
			}
		}
	})

	t.Run("refuses paths outside the destination", func(t *testing.T) {
		parent := t.TempDir()
		dest := filepath.Join(parent, "php")
		err := extractZip(context.Background(), writeZip(t, map[string]string{"../evil.txt": "x"}), dest)
		if err == nil || !strings.Contains(err.Error(), "invalid path in zip") {
			t.Errorf("error = %v, want invalid path", err)
		}
		if _, err := os.Stat(filepath.Join(parent, "evil.txt")); err == nil {
			t.Error("evil.txt was written outside the destination")
		}
	})

	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := extractZip(ctx, writeZip(t, files), t.TempDir()); err == nil {
			t.Error("want an error from a cancelled context")
		}
	})
}
