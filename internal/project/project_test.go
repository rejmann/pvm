package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFind(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}

	if _, _, err := Find(child); !errors.Is(err, ErrNotFound) {
		t.Fatalf("no file: error = %v, want ErrNotFound", err)
	}

	rootFile := write(t, root, "8.2\n")
	v, p, err := Find(child)
	if err != nil || v != "8.2" || p != rootFile {
		t.Fatalf("Find = (%q, %q, %v), want (8.2, %q, nil)", v, p, err, rootFile)
	}

	nearest := write(t, filepath.Join(root, "a"), "8.3")
	v, p, err = Find(child)
	if err != nil || v != "8.3" || p != nearest {
		t.Fatalf("nearest file should win: (%q, %q, %v)", v, p, err)
	}
}

func TestRead(t *testing.T) {
	tests := []struct {
		content string
		want    string
		wantErr bool
	}{
		{content: "8.3", want: "8.3"},
		{content: "  8.3.30  \r\n", want: "8.3.30"},
		{content: "# pinned for prod\n\n8.1\n8.2\n", want: "8.1"},
		{content: "", wantErr: true},
		{content: "\n# only a comment\n", wantErr: true},
	}

	for _, tt := range tests {
		p := write(t, t.TempDir(), tt.content)
		got, err := Read(p)
		if tt.wantErr {
			if err == nil {
				t.Errorf("Read(%q) expected error, got %q", tt.content, got)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("Read(%q) = (%q, %v), want %q", tt.content, got, err, tt.want)
		}
	}
}

func TestWriteAndRemove(t *testing.T) {
	dir := t.TempDir()

	p, err := Write(dir, "8.4")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := Read(p); err != nil || got != "8.4" {
		t.Fatalf("Read after Write = (%q, %v)", got, err)
	}

	if _, err := Remove(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("file should be gone, stat err = %v", err)
	}
	if _, err := Remove(dir); err != nil {
		t.Errorf("Remove on missing file = %v, want nil", err)
	}
}

func write(t *testing.T, dir, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, FileName)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}
