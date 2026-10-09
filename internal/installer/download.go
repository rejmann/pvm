package installer

// The helpers here serve the Windows installer, the only one that downloads
// PHP itself; they carry no build tag so their tests run on every system.

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sync/errgroup"
)

// firstAvailable returns the first URL of urls (ordered by preference) that
// exists. All of them are asked at once with HEAD, and it answers as soon as
// every URL before the chosen one is known to be missing, cancelling the
// requests still running.
func firstAvailable(ctx context.Context, client *http.Client, urls []string) (string, error) {
	if len(urls) == 0 {
		return "", errors.New("no URL to try")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type probe struct {
		i   int
		err error
	}
	// Buffered, so the probes still running when an answer is found finish
	// without a receiver.
	probes := make(chan probe, len(urls))
	for i, url := range urls {
		go func() { probes <- probe{i, urlExists(ctx, client, url)} }()
	}

	done := make([]bool, len(urls))
	errs := make([]error, len(urls))
	next := 0 // the most wanted URL not yet known to be missing
	for range urls {
		p := <-probes
		done[p.i], errs[p.i] = true, p.err
		for ; next < len(urls) && done[next]; next++ {
			if errs[next] == nil {
				return urls[next], nil
			}
		}
	}
	return "", errs[len(errs)-1]
}

func urlExists(ctx context.Context, client *http.Client, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}
	return nil
}

// extractZip extracts the archive at zipPath into destDir, several entries at
// a time: inflating and writing each file is independent of the others. The
// first error, or ctx being cancelled, stops it.
func extractZip(ctx context.Context, zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())
	for _, f := range r.File {
		if gctx.Err() != nil {
			break
		}
		g.Go(func() error { return extractZipEntry(f, destDir) })
	}
	if err := g.Wait(); err != nil {
		return err
	}
	// Entries left out because the caller gave up.
	return ctx.Err()
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
