package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirm(t *testing.T) {
	for input, want := range map[string]bool{"y\n": true, " YES \r\n": true, "n\n": false, "\n": false, "": false} {
		if got := confirm(strings.NewReader(input), &bytes.Buffer{}, "? "); got != want {
			t.Errorf("confirm(%q) = %v, want %v", input, got, want)
		}
	}
}
