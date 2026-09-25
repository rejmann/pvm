package installer

import "testing"

func TestMajorMinor(t *testing.T) {
	tests := map[string]string{
		"8.3.30": "8.3",
		"8.3":    "8.3",
		"8":      "8",
	}
	for in, want := range tests {
		if got := majorMinor(in); got != want {
			t.Errorf("majorMinor(%q) = %q, want %q", in, got, want)
		}
	}
}
