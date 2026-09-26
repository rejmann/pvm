// Package selfupdate replaces the running pvm binary with a release published
// on GitHub.
package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	DefaultAPIURL            = "https://api.github.com/repos/rejmann/pvm"
	DefaultDownloadPrefixURL = "https://github.com/rejmann/pvm/releases/download"

	// maxBinarySize guards against decompressing an unexpectedly large archive.
	maxBinarySize = 200 << 20
)

// Updater fetches releases from APIURL/DownloadURL; tests point them at an
// httptest server.
type Updater struct {
	APIURL      string
	DownloadURL string
	Client      *http.Client
	GOOS        string
	GOARCH      string
}

func New() *Updater {
	return &Updater{
		APIURL:      DefaultAPIURL,
		DownloadURL: DefaultDownloadPrefixURL,
		Client:      &http.Client{Timeout: 5 * time.Minute},
		GOOS:        runtime.GOOS,
		GOARCH:      runtime.GOARCH,
	}
}

// LatestTag returns the tag of the latest published release (e.g. v1.2.0).
func (u *Updater) LatestTag(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.APIURL+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := u.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch latest release: unexpected status code: %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode latest release: %w", err)
	}
	if release.TagName == "" {
		return "", errors.New("latest release has no tag")
	}
	return release.TagName, nil
}

// AssetName is the release archive for goos/goarch, as published by the build workflow.
func AssetName(goos, goarch string) string {
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("pvm-%s-%s%s", goos, goarch, ext)
}

func binaryName(goos string) string {
	if goos == "windows" {
		return "pvm.exe"
	}
	return "pvm"
}

// Download fetches the release archive for tag and returns the pvm binary inside it.
func (u *Updater) Download(ctx context.Context, tag string) ([]byte, error) {
	asset := AssetName(u.GOOS, u.GOARCH)
	url := fmt.Sprintf("%s/%s/%s", u.DownloadURL, tag, asset)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := u.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", asset, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("release %s has no build for %s/%s (%s)", tag, u.GOOS, u.GOARCH, asset)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: unexpected status code: %d", asset, resp.StatusCode)
	}

	archive, err := io.ReadAll(io.LimitReader(resp.Body, maxBinarySize))
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", asset, err)
	}

	if u.GOOS == "windows" {
		return extractZip(archive, binaryName(u.GOOS))
	}
	return extractTarGz(archive, binaryName(u.GOOS))
}

func extractTarGz(archive []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read archive: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg && filepath.Base(hdr.Name) == name {
			return io.ReadAll(io.LimitReader(tr, maxBinarySize))
		}
	}
	return nil, fmt.Errorf("%s not found in archive", name)
}

func extractZip(archive []byte, name string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}

	for _, f := range zr.File {
		if f.FileInfo().IsDir() || filepath.Base(f.Name) != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("read archive: %w", err)
		}
		defer rc.Close()
		return io.ReadAll(io.LimitReader(rc, maxBinarySize))
	}
	return nil, fmt.Errorf("%s not found in archive", name)
}

// ErrPermission is returned by Replace when the binary's directory is not writable.
var ErrPermission = errors.New("permission denied")

// Replace swaps the executable at exe for bin. The new file is written next to
// exe and then moved into place by swap (per OS), so a failure never leaves a
// half-written binary.
func Replace(exe string, bin []byte) error {
	dir := filepath.Dir(exe)

	tmp, err := os.CreateTemp(dir, ".pvm-upgrade-*")
	if err != nil {
		return wrapPermission(err, dir)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		return fmt.Errorf("write new binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write new binary: %w", err)
	}
	if err := os.Chmod(tmpName, 0755); err != nil {
		return fmt.Errorf("write new binary: %w", err)
	}

	if err := swap(tmpName, exe); err != nil {
		return wrapPermission(err, dir)
	}
	return nil
}

func wrapPermission(err error, dir string) error {
	if errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("%w: cannot write to %s", ErrPermission, dir)
	}
	return fmt.Errorf("replace binary: %w", err)
}

// SameVersion reports whether two version strings name the same release,
// ignoring a leading "v".
func SameVersion(a, b string) bool {
	return strings.TrimPrefix(a, "v") == strings.TrimPrefix(b, "v")
}
