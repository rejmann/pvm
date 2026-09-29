package home

import (
	"errors"
	"os"
	"testing"
)

func TestCurrent(t *testing.T) {
	tests := []struct {
		name    string
		content *string
		want    string
		wantErr error
	}{
		{name: "no file", content: nil, wantErr: ErrNoCurrentVersion},
		{name: "empty file", content: ptr("  \n"), wantErr: ErrNoCurrentVersion},
		{name: "version with whitespace", content: ptr("8.3\n"), want: "8.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New(t.TempDir())
			if tt.content != nil {
				if err := os.WriteFile(d.CurrentFile(), []byte(*tt.content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			got, err := d.Current()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Current error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Current = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCurrentRoundTrip(t *testing.T) {
	d := New(t.TempDir())
	if err := d.SetCurrent("8.2"); err != nil {
		t.Fatal(err)
	}
	got, err := d.Current()
	if err != nil || got != "8.2" {
		t.Fatalf("Current = (%q, %v), want (\"8.2\", nil)", got, err)
	}

	if err := d.ClearCurrent(); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Current(); !errors.Is(err, ErrNoCurrentVersion) {
		t.Errorf("after clear, Current error = %v, want ErrNoCurrentVersion", err)
	}
	// Clearing again is a no-op.
	if err := d.ClearCurrent(); err != nil {
		t.Errorf("second ClearCurrent = %v, want nil", err)
	}
}

func ptr(s string) *string { return &s }
