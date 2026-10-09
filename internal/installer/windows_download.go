//go:build windows

package installer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/rejmann/pvm/internal/progress"
)

// probeTimeout bounds the search for the build among the candidate URLs.
const probeTimeout = 30 * time.Second

// downloadClient has no timeout of its own: the download lasts as long as
// the connection allows and stops when the context is cancelled.
var downloadClient = &http.Client{}

func downloadAndExtractPHP(ctx context.Context, ver, destDir string, out, show io.Writer) error {
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	url, err := firstAvailable(probeCtx, downloadClient, candidateURLs(ver))
	cancel()
	if err != nil {
		return fmt.Errorf("could not find PHP %s for Windows: %w", ver, err)
	}

	fmt.Fprintf(out, "Downloading %s\n", url)
	if err := downloadExtract(ctx, url, destDir, show); err != nil {
		return fmt.Errorf("could not download PHP %s for Windows: %w", ver, err)
	}
	return nil
}

// vcVersions lists all known VC strings newest-first.
// We try them all so no hardcoded mapping per PHP version is needed.
var vcVersions = []string{"vs17", "vs16", "vc15", "vc14", "vc11"}

// candidateURLs lists where the build of ver may be, most wanted first.
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

// downloadExtract downloads the zip at url, showing how far it is on show
// (if not nil), and extracts it into destDir.
func downloadExtract(ctx context.Context, url, destDir string, show io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := downloadClient.Do(req)
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

	body, stop := progress.Track(show, resp.Body, resp.ContentLength)
	_, err = io.Copy(tmp, body)
	stop()
	if err != nil {
		tmp.Close()
		return fmt.Errorf("download: %w", err)
	}
	tmp.Close()

	return extractZip(ctx, tmpName, destDir)
}
