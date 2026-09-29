package selfremove

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rejmann/pvm/internal/selfupdate"

	"github.com/rejmann/pvm/internal/home"
)

// recordingOps returns Ops that record their calls instead of
// touching the system; binErr is returned by removeBinary.
func recordingOps(calls *[]string, binErr error) Ops {
	return Ops{
		RemoveVersion: func(_ *home.Dir, v string) error {
			*calls = append(*calls, "php "+v)
			return nil
		},
		RemoveIntegration: func(_ *home.Dir, _ string) error {
			*calls = append(*calls, "integration")
			return nil
		},
		RemoveBinary: func(string) error {
			*calls = append(*calls, "binary")
			return binErr
		},
		Confirm: func(string) bool { return false },
	}
}

func TestRun(t *testing.T) {
	t.Run("aborts without confirmation", func(t *testing.T) {
		h := newHome(t)
		var calls []string
		var out bytes.Buffer

		err := Run(h, fakeExe(t), false, false, answer(recordingOps(&calls, nil), false), &out, &bytes.Buffer{})
		if err != nil {
			t.Fatal(err)
		}
		if len(calls) != 0 {
			t.Errorf("nothing should be removed, got calls %v", calls)
		}
		if _, err := os.Stat(h.Path); err != nil {
			t.Errorf("data dir should be kept: %v", err)
		}
		if !strings.Contains(out.String(), "Aborted.") {
			t.Errorf("unexpected output: %q", out.String())
		}
	})

	t.Run("confirmed removes data, integration and binary", func(t *testing.T) {
		h := newHome(t)
		fakeInstall(t, h, "8.3")
		var calls []string
		var out bytes.Buffer

		err := Run(h, fakeExe(t), false, false, answer(recordingOps(&calls, nil), true), &out, &bytes.Buffer{})
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"integration", "binary"}; !reflect.DeepEqual(calls, want) {
			t.Errorf("calls = %v, want %v", calls, want)
		}
		if _, err := os.Stat(h.Path); !os.IsNotExist(err) {
			t.Errorf("data dir should be gone, stat err = %v", err)
		}
		if !strings.Contains(out.String(), "pvm removed.") {
			t.Errorf("unexpected output: %q", out.String())
		}
	})

	t.Run("--php removes installed versions first", func(t *testing.T) {
		h := newHome(t)
		fakeInstall(t, h, "8.2")
		fakeInstall(t, h, "8.3")
		var calls []string

		err := Run(h, fakeExe(t), true, true, recordingOps(&calls, nil), &bytes.Buffer{}, &bytes.Buffer{})
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"php 8.2", "php 8.3", "integration", "binary"}; !reflect.DeepEqual(calls, want) {
			t.Errorf("calls = %v, want %v", calls, want)
		}
	})

	t.Run("stops when a PHP version cannot be removed", func(t *testing.T) {
		h := newHome(t)
		fakeInstall(t, h, "8.3")
		var calls []string
		ops := recordingOps(&calls, nil)
		ops.RemoveVersion = func(_ *home.Dir, _ string) error { return errors.New("apt failed") }

		err := Run(h, fakeExe(t), true, true, ops, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "remove PHP 8.3") {
			t.Fatalf("error = %v, want remove PHP 8.3 failure", err)
		}
		if len(calls) != 0 {
			t.Errorf("nothing else should run, got calls %v", calls)
		}
		if _, err := os.Stat(h.Path); err != nil {
			t.Errorf("data dir should be kept: %v", err)
		}
	})

	t.Run("integration failure is only a warning", func(t *testing.T) {
		h := newHome(t)
		var calls []string
		ops := recordingOps(&calls, nil)
		ops.RemoveIntegration = func(_ *home.Dir, _ string) error { return errors.New("no powershell") }
		var errOut bytes.Buffer

		err := Run(h, fakeExe(t), false, true, ops, &bytes.Buffer{}, &errOut)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(errOut.String(), "Warning: no powershell") {
			t.Errorf("unexpected stderr: %q", errOut.String())
		}
		if want := []string{"binary"}; !reflect.DeepEqual(calls, want) {
			t.Errorf("calls = %v, want %v", calls, want)
		}
	})

	t.Run("binary permission error explains how to finish", func(t *testing.T) {
		h := newHome(t)
		var calls []string
		exe := fakeExe(t)
		binErr := fmt.Errorf("%w: cannot write to %s", selfupdate.ErrPermission, filepath.Dir(exe))

		err := Run(h, exe, false, true, recordingOps(&calls, binErr), &bytes.Buffer{}, &bytes.Buffer{})
		if !errors.Is(err, selfupdate.ErrPermission) || !strings.Contains(err.Error(), exe) {
			t.Fatalf("error = %v, want permission error naming %s", err, exe)
		}
		if _, err := os.Stat(h.Path); !os.IsNotExist(err) {
			t.Errorf("data dir should be gone before the binary, stat err = %v", err)
		}
	})

	t.Run("refuses to remove the home directory", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("no home directory")
		}
		h := newHome(t)
		h.Path = home
		var calls []string

		err = Run(h, fakeExe(t), false, true, recordingOps(&calls, nil), &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "refusing to remove") {
			t.Fatalf("error = %v, want refusal", err)
		}
		if len(calls) != 0 {
			t.Errorf("nothing should run, got calls %v", calls)
		}
	})
}

// answer makes ops.Confirm answer yes or no.
func answer(ops Ops, yes bool) Ops {
	ops.Confirm = func(string) bool { return yes }
	return ops
}

func newHome(t *testing.T) *home.Dir {
	t.Helper()
	h := home.New(t.TempDir())
	if err := h.Init(); err != nil {
		t.Fatal(err)
	}
	return h
}

func fakeInstall(t *testing.T, h *home.Dir, v string) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "php"+v)
	if err := os.WriteFile(bin, nil, 0755); err != nil {
		t.Fatal(err)
	}
	if err := h.SetBinary(v, bin); err != nil {
		t.Fatal(err)
	}
}

func fakeExe(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "pvm")
	if err := os.WriteFile(exe, []byte("pvm current"), 0755); err != nil {
		t.Fatal(err)
	}
	return exe
}
