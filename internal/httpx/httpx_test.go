package httpx

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			w.Write([]byte(`{"name":"pvm"}`))
		case "/agent":
			w.Write([]byte(r.Header.Get("User-Agent") + "|" + r.Header.Get("Accept")))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestGet(t *testing.T) {
	srv := newServer(t)
	ctx := context.Background()

	got, err := Get(ctx, srv.Client(), srv.URL+"/agent", 1<<10, http.Header{"Accept": {"text/plain"}})
	if err != nil {
		t.Fatal(err)
	}
	if want := userAgent + "|text/plain"; string(got) != want {
		t.Errorf("Get = %q, want %q", got, want)
	}

	if _, err := Get(ctx, srv.Client(), srv.URL+"/json", 4, nil); !errors.Is(err, ErrTooLarge) {
		t.Errorf("over the limit: error = %v, want ErrTooLarge", err)
	}

	_, err = Get(ctx, srv.Client(), srv.URL+"/missing", 1<<10, nil)
	if !IsNotFound(err) || err.Error() != "unexpected status code: 404" {
		t.Errorf("missing: error = %v, want 404 StatusError", err)
	}
}

func TestGetJSON(t *testing.T) {
	srv := newServer(t)

	got, err := GetJSON[struct{ Name string }](context.Background(), srv.Client(), srv.URL+"/json", 1<<10, nil)
	if err != nil || got.Name != "pvm" {
		t.Fatalf("GetJSON = (%+v, %v)", got, err)
	}
}

func TestDownload(t *testing.T) {
	srv := newServer(t)
	ctx := context.Background()

	var buf bytes.Buffer
	if err := Download(ctx, srv.Client(), srv.URL+"/json", &buf, 1<<10); err != nil || buf.String() != `{"name":"pvm"}` {
		t.Fatalf("Download = (%q, %v)", buf.String(), err)
	}
	if err := Download(ctx, srv.Client(), srv.URL+"/json", &bytes.Buffer{}, 4); !errors.Is(err, ErrTooLarge) {
		t.Errorf("over the limit: error = %v, want ErrTooLarge", err)
	}
}
