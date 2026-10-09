package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestConfirm(t *testing.T) {
	for input, want := range map[string]bool{"y\n": true, " YES \r\n": true, "n\n": false, "\n": false, "": false} {
		if got := confirm(context.Background(), strings.NewReader(input), &bytes.Buffer{}, "? "); got != want {
			t.Errorf("confirm(%q) = %v, want %v", input, got, want)
		}
	}
}
