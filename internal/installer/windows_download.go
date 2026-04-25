package installer

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func downloadAndExtractPHP(ver, branch, destDir string) error {
	vc := vcForBranch(branch)

	var lastErr error
	for _, url := range candidateURLs(ver, vc) {
		fmt.Printf("Trying %s\n", url)
		if err := downloadExtract(url, destDir); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("could not download PHP %s for Windows: %w", ver, lastErr)
}

func vcForBranch(branch string) string {
	parts := strings.SplitN(branch, ".", 2)
	if len(parts) < 2 {
		return "vs16"
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	switch {
	case major >= 8:
		return "vs16"
	case major == 7 && minor >= 2:
		return "vc15"
	case major == 7:
		return "vc14"
	default:
		return "vc11"
	}
}

func candidateURLs(ver, vc string) []string {
	releases := "https://windows.php.net/downloads/releases"
	archives := releases + "/archives"

	var urls []string
	for _, base := range []string{releases, archives} {
		urls = append(urls,
			fmt.Sprintf("%s/php-%s-nts-Win32-%s-x64.zip", base, ver, vc),
			fmt.Sprintf("%s/php-%s-Win32-%s-x64.zip", base, ver, vc),
		)
	}
	return urls
}

func downloadExtract(url, destDir string) error {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	tmp, err := os.CreateTemp("", "pvm-php-*.zip")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("download: %w", err)
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
