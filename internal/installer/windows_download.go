//go:build windows

package installer

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rejmann/pvm/internal/httpx"
	"github.com/rejmann/pvm/internal/proc"
)

// maxZipSize guards against an unexpectedly large download.
const maxZipSize = 500 << 20

var downloadClient = httpx.NewClient(httpx.DownloadTimeout)

func downloadAndExtractPHP(ctx context.Context, ver, destDir string, s proc.Streams) error {
	var lastErr error
	for _, url := range candidateURLs(ver) {
		fmt.Fprintf(s.Out, "Trying %s\n", url)
		if err := downloadExtract(ctx, url, destDir); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("could not download PHP %s for Windows: %w", ver, lastErr)
}

// vcVersions lists all known VC strings newest-first.
// We try them all so no hardcoded mapping per PHP version is needed.
var vcVersions = []string{"vs17", "vs16", "vc15", "vc14", "vc11"}

func candidateURLs(ver string) []string {
	releases := "https://windows.php.net/downloads/releases"
	archives := releases + "/archives"

	var urls []string
	for _, base := range []string{releases, archives} {
		for _, vc := range vcVersions {
			urls = append(urls,
				fmt.Sprintf("%s/php-%s-nts-Win32-%s-x64.zip", base, ver, vc),
			)
		}
	}
	return urls
}

func downloadExtract(ctx context.Context, url, destDir string) error {
	tmp, err := os.CreateTemp("", "pvm-php-*.zip")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := httpx.Download(ctx, downloadClient, url, tmp, maxZipSize); err != nil {
		tmp.Close()
		return fmt.Errorf("download %s: %w", url, err)
	}
	tmp.Close()

	return extractZip(tmpName, destDir)
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if err := extractZipEntry(f, destDir); err != nil {
			return err
		}
	}
	return nil
}

func extractZipEntry(f *zip.File, destDir string) error {
	target := filepath.Join(destDir, f.Name)

	// prevent zip slip
	if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid path in zip: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0755)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}

	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = io.Copy(out, rc) //nolint:gosec
	return err
}
