// Package composer gives each pvm-managed PHP version its own composer.phar
// and Composer home, downloaded on demand from getcomposer.org.
//
// Layout under <pvm-home>/composer:
//
//	cache/                       COMPOSER_CACHE_DIR, shared: downloads don't depend on PHP
//	php/<version>/composer.phar  the Composer this PHP version runs
//	php/<version>/home/          COMPOSER_HOME: config, auth.json, global packages,
//	                             self-update backups and the verification keys
//
// Nothing is shared between PHP versions but the cache, so `self-update`,
// `--rollback` or `global require` run for one version never affect another.
package composer

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rejmann/pvm/internal/version"
)

const (
	DefaultBaseURL = "https://getcomposer.org"

	PharName = "composer.phar"

	// maxPharSize guards against an unexpectedly large download.
	maxPharSize = 100 << 20
)

// Composer's public keys (https://composer.github.io/pubkeys.html): tags signs
// releases, dev signs snapshots. They are also written to every Composer home,
// where `self-update` and `diagnose` look for them.
var (
	//go:embed keys/tags.pub
	tagsKey []byte
	//go:embed keys/dev.pub
	devKey []byte
)

func dir(base string) string {
	return filepath.Join(base, "composer")
}

// VersionDir holds everything Composer keeps for one pvm-managed PHP version.
func VersionDir(base, phpVersion string) string {
	return filepath.Join(dir(base), "php", phpVersion)
}

// PharPath is the composer.phar that PHP phpVersion (as installed by pvm, e.g. "8.3") runs.
func PharPath(base, phpVersion string) string {
	return filepath.Join(VersionDir(base, phpVersion), PharName)
}

func homeDir(base, phpVersion string) string {
	return filepath.Join(VersionDir(base, phpVersion), "home")
}

// Remove deletes the Composer of a PHP version; called when pvm removes it.
func Remove(base, phpVersion string) error {
	return os.RemoveAll(VersionDir(base, phpVersion))
}

// Env returns the variables that keep Composer's home (per PHP version) and
// cache under the pvm home instead of the user's global directories. A
// variable the user already set (getenv returns non-empty) is left alone.
func Env(base, phpVersion string, getenv func(string) string) map[string]string {
	env := map[string]string{}
	for name, path := range map[string]string{
		"COMPOSER_HOME":      homeDir(base, phpVersion),
		"COMPOSER_CACHE_DIR": filepath.Join(dir(base), "cache"),
	} {
		if getenv(name) == "" {
			env[name] = path
		}
	}
	return env
}

// Release is one entry of getcomposer.org/versions.
type Release struct {
	Path    string `json:"path"`    // e.g. /download/2.10.3/composer.phar
	Version string `json:"version"` // e.g. 2.10.3
	MinPHP  int    `json:"min-php"` // PHP_VERSION_ID, e.g. 70205
}

// SelectRelease picks the newest release that runs on phpVersion (e.g.
// "7.4.33") — the same rule `composer self-update` follows.
func SelectRelease(releases []Release, phpVersion string) (Release, error) {
	php, err := version.Parse(phpVersion)
	if err != nil {
		return Release{}, err
	}
	id := php.Major*10000 + php.Minor*100 + php.Patch

	var best Release
	var bestVer version.Version
	for _, r := range releases {
		v, err := version.Parse(r.Version)
		if err != nil || r.MinPHP > id {
			continue
		}
		if best.Version == "" || v.Compare(bestVer) > 0 {
			best, bestVer = r, v
		}
	}
	if best.Version == "" {
		return Release{}, fmt.Errorf("no Composer release supports PHP %s", phpVersion)
	}
	return best, nil
}

// Downloader fetches Composer from BaseURL and checks it against PublicKey;
// tests point both at their own server and key.
type Downloader struct {
	BaseURL   string
	PublicKey []byte // PEM
	Client    *http.Client
}

func New() *Downloader {
	return &Downloader{
		BaseURL:   DefaultBaseURL,
		PublicKey: tagsKey,
		Client:    &http.Client{Timeout: 5 * time.Minute},
	}
}

// Ensure returns the composer.phar of pvm-managed PHP phpVersion, whose exact
// version is phpExact (e.g. "8.3.12"). When it is missing, the newest Composer
// for phpExact is downloaded, its signature verified, and the Composer home
// prepared; onDownload, if not nil, is told which release before the download.
// An existing phar is never replaced: updating it is `composer self-update`'s job.
func (d *Downloader) Ensure(ctx context.Context, base, phpVersion, phpExact string, onDownload func(Release)) (string, error) {
	path := PharPath(base, phpVersion)
	if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
		return path, nil
	}

	var releases struct {
		Stable []Release `json:"stable"`
	}
	body, err := d.get(ctx, d.BaseURL+"/versions", 1<<20)
	if err != nil {
		return "", fmt.Errorf("list Composer releases: %w", err)
	}
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", fmt.Errorf("list Composer releases: %w", err)
	}
	r, err := SelectRelease(releases.Stable, phpExact)
	if err != nil {
		return "", err
	}

	if onDownload != nil {
		onDownload(r)
	}
	phar, err := d.download(ctx, r)
	if err != nil {
		return "", err
	}

	if err := writeKeys(homeDir(base, phpVersion)); err != nil {
		return "", fmt.Errorf("prepare Composer home: %w", err)
	}
	if err := writeAtomic(path, phar, 0755); err != nil {
		return "", fmt.Errorf("save %s: %w", PharName, err)
	}
	return path, nil
}

// download fetches release r and verifies its RSA-SHA384 signature, as
// `composer self-update` does.
func (d *Downloader) download(ctx context.Context, r Release) ([]byte, error) {
	url := d.BaseURL + r.Path

	phar, err := d.get(ctx, url, maxPharSize)
	if err != nil {
		return nil, fmt.Errorf("download Composer %s: %w", r.Version, err)
	}
	sigJSON, err := d.get(ctx, url+".sig", 1<<12)
	if err != nil {
		return nil, fmt.Errorf("download Composer %s signature: %w", r.Version, err)
	}
	if err := verify(phar, sigJSON, d.PublicKey); err != nil {
		return nil, fmt.Errorf("Composer %s: %w", r.Version, err)
	}
	return phar, nil
}

func verify(phar, sigJSON, pubPEM []byte) error {
	var sig struct {
		SHA384 string `json:"sha384"`
	}
	if err := json.Unmarshal(sigJSON, &sig); err != nil || sig.SHA384 == "" {
		return errors.New("malformed signature file")
	}
	raw, err := base64.StdEncoding.DecodeString(sig.SHA384)
	if err != nil {
		return errors.New("malformed signature file")
	}

	block, _ := pem.Decode(pubPEM)
	if block == nil {
		return errors.New("invalid public key")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return errors.New("invalid public key: not RSA")
	}

	sum := sha512.Sum384(phar)
	if err := rsa.VerifyPKCS1v15(rsaKey, crypto.SHA384, sum[:], raw); err != nil {
		return errors.New("signature verification failed — download corrupted or tampered with")
	}
	return nil
}

// writeKeys puts Composer's public keys in its home, where self-update
// verifies new releases and diagnose checks for them. Existing keys are kept.
func writeKeys(home string) error {
	for name, key := range map[string][]byte{"keys.tags.pub": tagsKey, "keys.dev.pub": devKey} {
		path := filepath.Join(home, name)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := writeAtomic(path, key, 0644); err != nil {
			return err
		}
	}
	return nil
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
// interrupted download never leaves a truncated file behind.
func writeAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".pvm-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
