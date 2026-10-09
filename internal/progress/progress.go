// Package progress shows how far a download is, on a terminal.
package progress

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/term"
)

// interval is how often the line is redrawn.
var interval = 200 * time.Millisecond

// Terminal returns w when it is a terminal and nil otherwise, so a pipe or a
// file never receives the progress line.
func Terminal(w io.Writer) io.Writer {
	if f, ok := w.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		return w
	}
	return nil
}

// Track returns r wrapped so that, while it is read, a goroutine redraws one
// line on w with the bytes read so far out of total (unknown when <= 0). The
// returned function stops it and erases the line; call it when the download
// ends. A nil w (see Terminal) shows nothing.
func Track(w io.Writer, r io.Reader, total int64) (io.Reader, func()) {
	if w == nil {
		return r, func() {}
	}

	c := &counter{r: r}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		tick := time.NewTicker(interval)
		defer tick.Stop()

		width := 0 // of the longest line drawn, to overwrite what is left of it
		for {
			select {
			case <-tick.C:
				line := format(c.n.Load(), total)
				width = max(width, len(line))
				fmt.Fprintf(w, "\r%-*s", width, line)
			case <-stop:
				if width > 0 {
					fmt.Fprintf(w, "\r%*s\r", width, "")
				}
				return
			}
		}
	}()

	return c, sync.OnceFunc(func() {
		close(stop)
		<-done
	})
}

// counter counts the bytes read through it; the download writes the count
// and the drawing goroutine reads it.
type counter struct {
	r io.Reader
	n atomic.Int64
}

func (c *counter) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n.Add(int64(n))
	return n, err
}

func format(n, total int64) string {
	const mb = 1 << 20
	if total <= 0 {
		return fmt.Sprintf("  %.1f MB", float64(n)/mb)
	}
	return fmt.Sprintf("  %3d%%  %.1f / %.1f MB", min(n*100/total, 100), float64(n)/mb, float64(total)/mb)
}
