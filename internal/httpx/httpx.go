// Package httpx is the HTTP client shared by everything pvm downloads: every
// request has a context, a timeout and a size limit, and identifies pvm.
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// APITimeout bounds small JSON requests (php.net, GitHub API).
	APITimeout = 15 * time.Second
	// DownloadTimeout bounds whole downloads (PHP builds, Composer, pvm).
	DownloadTimeout = 10 * time.Minute

	userAgent = "pvm (+https://github.com/rejmann/pvm)"
)

// ErrTooLarge is returned when a response exceeds the caller's limit.
var ErrTooLarge = errors.New("response too large")

// StatusError is returned for any response other than 200 OK.
type StatusError struct {
	Code int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("unexpected status code: %d", e.Code)
}

// IsNotFound reports whether err is a 404 response.
func IsNotFound(err error) bool {
	var se *StatusError
	return errors.As(err, &se) && se.Code == http.StatusNotFound
}

func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

// Open sends a GET for url and returns the body of a 200 response; the
// caller closes it. header, if not nil, is added to the request.
func Open(ctx context.Context, c *http.Client, url string, header http.Header) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, &StatusError{Code: resp.StatusCode}
	}
	return resp.Body, nil
}

// Get returns the body of url, failing with ErrTooLarge past limit bytes.
func Get(ctx context.Context, c *http.Client, url string, limit int64, header http.Header) ([]byte, error) {
	body, err := Open(ctx, c, url, header)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrTooLarge
	}
	return data, nil
}

// GetJSON decodes the JSON body of url into a T.
func GetJSON[T any](ctx context.Context, c *http.Client, url string, limit int64, header http.Header) (T, error) {
	var v T
	data, err := Get(ctx, c, url, limit, header)
	if err != nil {
		return v, err
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return v, fmt.Errorf("decode response: %w", err)
	}
	return v, nil
}

// Download streams url into w, failing with ErrTooLarge past limit bytes.
func Download(ctx context.Context, c *http.Client, url string, w io.Writer, limit int64) error {
	body, err := Open(ctx, c, url, nil)
	if err != nil {
		return err
	}
	defer body.Close()

	n, err := io.Copy(w, io.LimitReader(body, limit+1))
	if err != nil {
		return err
	}
	if n > limit {
		return ErrTooLarge
	}
	return nil
}
