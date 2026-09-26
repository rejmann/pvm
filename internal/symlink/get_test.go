package symlink

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGetCurrent(t *testing.T) {
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
			base := t.TempDir()
			if tt.content != nil {
				if err := os.WriteFile(filepath.Join(base, "current-version"), []byte(*tt.content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			got, err := GetCurrent(base)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetCurrent error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("GetCurrent = %q, want %q", got, tt.want)
			}
		})
	}
}

func ptr(s string) *string { return &s }
