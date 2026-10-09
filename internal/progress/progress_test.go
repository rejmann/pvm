package progress

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFormat(t *testing.T) {
	for _, tt := range []struct {
		n, total int64
		want     string
	}{
		{0, 10 << 20, "    0%  0.0 / 10.0 MB"},
		{5 << 20, 10 << 20, "   50%  5.0 / 10.0 MB"},
		{10 << 20, 10 << 20, "  100%  10.0 / 10.0 MB"},
		{11 << 20, 10 << 20, "  100%  11.0 / 10.0 MB"}, // more than announced
		{3 << 19, 0, "  1.5 MB"},
		{3 << 19, -1, "  1.5 MB"},
	} {
		if got := format(tt.n, tt.total); got != tt.want {
			t.Errorf("format(%d, %d) = %q, want %q", tt.n, tt.total, got, tt.want)
		}
	}
}

// lines is a writer that hands every write to the test.
type lines chan string

func (l lines) Write(p []byte) (int, error) {
	l <- string(p)
	return len(p), nil
}

func TestTrack(t *testing.T) {
	old := interval
	interval = time.Millisecond
	defer func() { interval = old }()

	const total = 10 << 20
	out := make(lines)
	r, stop := Track(out, bytes.NewReader(make([]byte, total)), total)

	waitFor := func(want string) {
		t.Helper()
		timeout := time.After(5 * time.Second)
		for {
			select {
			case line := <-out:
				if strings.Contains(line, want) {
					return
				}
			case <-timeout:
				t.Fatalf("no line with %q", want)
			}
		}
	}

	if _, err := io.ReadFull(r, make([]byte, total/2)); err != nil {
		t.Fatal(err)
	}
	waitFor("50%  5.0 / 10.0 MB")
	if _, err := io.Copy(io.Discard, r); err != nil {
		t.Fatal(err)
	}
	waitFor("100%")

	// stop erases the line and waits for the goroutine, so nothing is
	// written after it returns.
	stopped := make(chan struct{})
	go func() { stop(); close(stopped) }()
	var last string
	for done := false; !done; {
		select {
		case last = <-out:
		case <-stopped:
			done = true
		}
	}
	if strings.TrimSpace(last) != "" || !strings.HasSuffix(last, "\r") {
		t.Errorf("last write = %q, want the line erased", last)
	}
	stop() // a second call is harmless
	select {
	case line := <-out:
		t.Errorf("written after stop: %q", line)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestTrackWithoutTerminal(t *testing.T) {
	src := strings.NewReader("data")
	r, stop := Track(nil, src, 4)
	if r != io.Reader(src) {
		t.Error("a nil writer must return the reader untouched")
	}
	stop()

	if Terminal(&bytes.Buffer{}) != nil {
		t.Error("a buffer is not a terminal")
	}
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if Terminal(f) != nil {
		t.Error("a regular file is not a terminal")
	}
}
