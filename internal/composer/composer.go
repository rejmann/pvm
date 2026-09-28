// Package composer downloads composer.phar on demand and picks the Composer
// release line that supports a given PHP version.
package composer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rejmann/pvm/internal/version"
)

const (
	DefaultDownloadURL = "https://getcomposer.org/download"

	// ChannelStable is the latest Composer release; it requires PHP 7.2.5+.
	ChannelStable = "latest-stable"
	// ChannelLTS is the Composer 2.2 long-term-support line, for PHP 5.3.2 up to 7.2.4.
	ChannelLTS = "latest-2.2.x"

	PharName = "composer.phar"

	// maxPharSize guards against an unexpectedly large download.
	maxPharSize = 100 << 20
)

var (
	minStable = version.Version{Major: 7, Minor: 2, Patch: 5}
	minLTS    = version.Version{Major: 5, Minor: 3, Patch: 2}
)

// Channel returns the Composer release line that runs on phpVersion (e.g. "8.3.12").
func Channel(phpVersion string) (string, error) {
	v, err := version.Parse(phpVersion)
	if err != nil {
		return "", err
	}
	switch {
	case v.Compare(minStable) >= 0:
		return ChannelStable, nil
	case v.Compare(minLTS) >= 0:
		return ChannelLTS, nil
	default:
		return "", fmt.Errorf("Composer requires PHP 5.3.2 or newer (found %s)", phpVersion)
	}
}

func dir(base string) string {
	return filepath.Join(base, "composer")
}

// PharPath is where composer.phar for channel lives under the pvm home.
func PharPath(base, channel string) string {
	return filepath.Join(dir(base), channel, PharName)
}

// Env returns the variables that keep Composer's config, auth and cache under
// the pvm home instead of the user's global directories. A variable the user
// already set (getenv returns non-empty) is left alone.
func Env(base string, getenv func(string) string) map[string]string {
	env := map[string]string{}
	for name, path := range map[string]string{
		"COMPOSER_HOME":      filepath.Join(dir(base), "home"),
		"COMPOSER_CACHE_DIR": filepath.Join(dir(base), "cache"),
	} {
		if getenv(name) == "" {
			env[name] = path
		}
	}
	return env
}

// Downloader fetches composer.phar from DownloadURL; tests point it at an
// httptest server.
type Downloader struct {
	DownloadURL string
	Client      *http.Client
}

func New() *Downloader {
	return &Downloader{
		DownloadURL: DefaultDownloadURL,
		Client:      &http.Client{Timeout: 5 * time.Minute},
	}
}

// Ensure returns the path of composer.phar for channel, downloading it first
// when it is missing. onDownload, if not nil, is called right before a download.
func (d *Downloader) Ensure(ctx context.Context, base, channel string, onDownload func()) (string, error) {
	path := PharPath(base, channel)
	if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
		return path, nil
	}

	if onDownload != nil {
		onDownload()
	}
	phar, err := d.download(ctx, channel)
	if err != nil {
		return "", err
	}
	if err := writeAtomic(path, phar); err != nil {
		return "", fmt.Errorf("save %s: %w", PharName, err)
	}
	return path, nil
}

// download fetches composer.phar for channel and checks it against the
// published SHA-256.
func (d *Downloader) download(ctx context.Context, channel string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s/%s", d.DownloadURL, channel, PharName)

	phar, err := d.get(ctx, url, maxPharSize)
	if err != nil {
		return nil, fmt.Errorf("download Composer (%s): %w", channel, err)
	}
	sum, err := d.get(ctx, url+".sha256", 1<<10)
	if err != nil {
		return nil, fmt.Errorf("download Composer checksum (%s): %w", channel, err)
	}

	fields := strings.Fields(string(sum))
	if len(fields) == 0 {
		return nil, fmt.Errorf("empty Composer checksum (%s)", channel)
	}
	got := sha256.Sum256(phar)
	if !strings.EqualFold(hex.EncodeToString(got[:]), fields[0]) {
		return nil, fmt.Errorf("Composer (%s) checksum mismatch — download corrupted, try again", channel)
	}
	return phar, nil
}

func (d *Downloader) get(ctx context.Context, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := d.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, errors.New("response too large")
	}
	return body, nil
}

// writeAtomic writes data next to path and renames it into place, so an
// interrupted download never leaves a truncated composer.phar behind.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".composer-*.phar")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := io.Copy(tmp, bytes.NewReader(data)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0755); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
