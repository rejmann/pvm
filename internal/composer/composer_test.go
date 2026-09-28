package composer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

func TestChannel(t *testing.T) {
	tests := []struct {
		php, want, wantErr string
	}{
		{php: "8.5.1", want: ChannelStable},
		{php: "7.2.5", want: ChannelStable},
		{php: "7.3", want: ChannelStable},
		{php: "7.2.4", want: ChannelLTS},
		{php: "7.1.33", want: ChannelLTS},
		{php: "5.3.2", want: ChannelLTS},
		{php: "5.3.1", wantErr: "requires PHP 5.3.2"},
		{php: "8.x", wantErr: "invalid version"},
	}
	for _, tt := range tests {
		got, err := Channel(tt.php)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Channel(%q) error = %v, want %q", tt.php, err, tt.wantErr)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("Channel(%q) = (%q, %v), want %q", tt.php, got, err, tt.want)
		}
	}
}

// pharServer serves phar and checksum under /<channel>/composer.phar[.sha256]
// and counts phar downloads.
func pharServer(t *testing.T, phar, checksum string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + ChannelStable + "/" + PharName:
			hits.Add(1)
			w.Write([]byte(phar))
		case "/" + ChannelStable + "/" + PharName + ".sha256":
			w.Write([]byte(checksum))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func sha(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestEnsureDownloadsOnce(t *testing.T) {
	const phar = "<?php // fake composer"
	srv, hits := pharServer(t, phar, sha(phar)+"  composer.phar\n")
	d := &Downloader{DownloadURL: srv.URL, Client: srv.Client()}
	base := t.TempDir()

	var announced int
	for i := 0; i < 2; i++ {
		path, err := d.Ensure(context.Background(), base, ChannelStable, func() { announced++ })
		if err != nil {
			t.Fatal(err)
		}
		if path != PharPath(base, ChannelStable) {
			t.Fatalf("path = %q", path)
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != phar {
			t.Fatalf("phar = %q, %v", data, err)
		}
	}
	if hits.Load() != 1 || announced != 1 {
		t.Errorf("downloads = %d, announced = %d, want 1 each", hits.Load(), announced)
	}
}

func TestEnsureErrors(t *testing.T) {
	t.Run("checksum mismatch", func(t *testing.T) {
		srv, _ := pharServer(t, "phar", sha("other"))
		d := &Downloader{DownloadURL: srv.URL, Client: srv.Client()}
		base := t.TempDir()

		_, err := d.Ensure(context.Background(), base, ChannelStable, nil)
		if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
			t.Fatalf("error = %v", err)
		}
		if _, err := os.Stat(PharPath(base, ChannelStable)); !os.IsNotExist(err) {
			t.Errorf("phar left behind after a bad download: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		srv, _ := pharServer(t, "phar", sha("phar"))
		d := &Downloader{DownloadURL: srv.URL, Client: srv.Client()}

		_, err := d.Ensure(context.Background(), t.TempDir(), ChannelLTS, nil)
		if err == nil || !strings.Contains(err.Error(), "404") {
			t.Fatalf("error = %v", err)
		}
	})
}
