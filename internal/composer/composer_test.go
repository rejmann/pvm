package composer

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// Mirrors getcomposer.org/versions: newest first, 2.2 LTS for old PHP.
var testReleases = []Release{
	{Path: "/download/2.10.3/composer.phar", Version: "2.10.3", MinPHP: 70205},
	{Path: "/download/2.2.30/composer.phar", Version: "2.2.30", MinPHP: 50300},
}

func TestSelectRelease(t *testing.T) {
	tests := []struct {
		php, want, wantErr string
	}{
		{php: "8.5.1", want: "2.10.3"},
		{php: "7.2.5", want: "2.10.3"},
		{php: "7.2.4", want: "2.2.30"},
		{php: "5.6.40", want: "2.2.30"},
		{php: "5.3.0", want: "2.2.30"},
		{php: "5.2.17", wantErr: "no Composer release supports PHP 5.2.17"},
		{php: "8.x", wantErr: "invalid version"},
	}
	for _, tt := range tests {
		got, err := SelectRelease(testReleases, tt.php)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("SelectRelease(%q) error = %v, want %q", tt.php, err, tt.wantErr)
			}
			continue
		}
		if err != nil || got.Version != tt.want {
			t.Errorf("SelectRelease(%q) = (%q, %v), want %q", tt.php, got.Version, err, tt.want)
		}
	}

	t.Run("order of the list does not matter", func(t *testing.T) {
		reversed := []Release{testReleases[1], testReleases[0]}
		if got, _ := SelectRelease(reversed, "8.3.0"); got.Version != "2.10.3" {
			t.Errorf("got %q, want 2.10.3", got.Version)
		}
	})
}

type fakeComposer struct {
	*httptest.Server
	pubPEM    []byte
	downloads atomic.Int32
}

// newFakeComposer serves /versions and every test release, signed with a
// fresh key; tamper corrupts the phars after signing.
func newFakeComposer(t *testing.T, tamper bool) *fakeComposer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeComposer{pubPEM: pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})}

	files := map[string][]byte{}
	for _, r := range testReleases {
		phar := []byte("<?php // composer " + r.Version)
		sum := sha512.Sum384(phar)
		sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA384, sum[:])
		if err != nil {
			t.Fatal(err)
		}
		sigJSON, _ := json.Marshal(map[string]string{"sha384": base64.StdEncoding.EncodeToString(sig)})
		if tamper {
			phar = append(phar, '!')
		}
		files[r.Path] = phar
		files[r.Path+".sig"] = sigJSON
	}
	versions, _ := json.Marshal(map[string][]Release{"stable": testReleases})
	files["/versions"] = versions

	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, PharName) {
			f.downloads.Add(1)
		}
		w.Write(data)
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeComposer) downloader() *Downloader {
	return &Downloader{BaseURL: f.URL, PublicKey: f.pubPEM, Client: f.Client()}
}

func TestEnsurePerPHPVersion(t *testing.T) {
	f := newFakeComposer(t, false)
	d := f.downloader()
	root := t.TempDir()

	var announced []string
	onDownload := func(r Release) { announced = append(announced, r.Version) }

	for _, c := range []struct{ installed, exact, want string }{
		{"8.3", "8.3.12", "2.10.3"},
		{"5.6", "5.6.40", "2.2.30"},
		{"8.3", "8.3.12", "2.10.3"}, // already there: no second download
	} {
		path, err := d.Ensure(context.Background(), root, c.installed, c.exact, onDownload)
		if err != nil {
			t.Fatal(err)
		}
		if path != PharPath(root, c.installed) {
			t.Fatalf("path = %q, want %q", path, PharPath(root, c.installed))
		}
		if data, _ := os.ReadFile(path); !strings.Contains(string(data), c.want) {
			t.Errorf("PHP %s got phar %q, want Composer %s", c.installed, data, c.want)
		}
	}

	if f.downloads.Load() != 2 || strings.Join(announced, ",") != "2.10.3,2.2.30" {
		t.Errorf("downloads = %d, announced = %v; want 2 downloads of 2.10.3 and 2.2.30", f.downloads.Load(), announced)
	}

	home := Env(root, "8.3", func(string) string { return "" })["COMPOSER_HOME"]
	for _, key := range []string{"keys.tags.pub", "keys.dev.pub"} {
		if _, err := os.Stat(filepath.Join(home, key)); err != nil {
			t.Errorf("%s not written to the Composer home: %v", key, err)
		}
	}
}

func TestEnsureRejectsBadSignature(t *testing.T) {
	d := newFakeComposer(t, true).downloader()
	root := t.TempDir()

	_, err := d.Ensure(context.Background(), root, "8.3", "8.3.12", nil)
	if err == nil || !strings.Contains(err.Error(), "signature verification failed") {
		t.Fatalf("error = %v", err)
	}
	if _, err := os.Stat(VersionDir(root, "8.3")); !os.IsNotExist(err) {
		t.Errorf("files left behind after a bad download: %v", err)
	}
}

func TestEnsureWithRealKeyRejectsOtherSigner(t *testing.T) {
	f := newFakeComposer(t, false)
	d := &Downloader{BaseURL: f.URL, PublicKey: tagsKey, Client: f.Client()}

	_, err := d.Ensure(context.Background(), t.TempDir(), "8.3", "8.3.12", nil)
	if err == nil || !strings.Contains(err.Error(), "signature verification failed") {
		t.Fatalf("error = %v, want Composer's key to reject a phar it did not sign", err)
	}
}

func TestEnvAndRemove(t *testing.T) {
	root := t.TempDir()

	a := Env(root, "8.3", func(string) string { return "" })
	b := Env(root, "7.4", func(string) string { return "" })
	if a["COMPOSER_HOME"] == b["COMPOSER_HOME"] {
		t.Errorf("PHP versions share COMPOSER_HOME %q", a["COMPOSER_HOME"])
	}
	if a["COMPOSER_CACHE_DIR"] != b["COMPOSER_CACHE_DIR"] {
		t.Errorf("cache not shared: %q vs %q", a["COMPOSER_CACHE_DIR"], b["COMPOSER_CACHE_DIR"])
	}

	custom := Env(root, "8.3", func(name string) string {
		if name == "COMPOSER_HOME" {
			return "/custom"
		}
		return ""
	})
	if _, ok := custom["COMPOSER_HOME"]; ok || len(custom) != 1 {
		t.Errorf("Env overrides a variable the user set: %v", custom)
	}

	if err := os.MkdirAll(filepath.Join(VersionDir(root, "8.3"), "home"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := Remove(root, "8.3"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(VersionDir(root, "8.3")); !os.IsNotExist(err) {
		t.Errorf("Remove left %s", VersionDir(root, "8.3"))
	}
	if err := Remove(root, "9.9"); err != nil {
		t.Errorf("Remove of a version without Composer: %v", err)
	}
}
