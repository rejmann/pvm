package cmd

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
)

// recordingOps returns selfRemoveOps that record their calls instead of
// touching the system; binErr is returned by removeBinary.
func recordingOps(calls *[]string, binErr error) selfRemoveOps {
	return selfRemoveOps{
		removeVersion: func(_, v string) error {
			*calls = append(*calls, "php "+v)
			return nil
		},
		removeIntegration: func(_, _ string) error {
			*calls = append(*calls, "integration")
			return nil
		},
		removeBinary: func(string) error {
			*calls = append(*calls, "binary")
			return binErr
		},
	}
}

func TestSelfRemove(t *testing.T) {
	t.Run("aborts without confirmation", func(t *testing.T) {
		m := newManager(t)
		var calls []string
		var out bytes.Buffer

		err := selfRemove(m, fakeExe(t), false, false, recordingOps(&calls, nil), strings.NewReader("n\n"), &out, &bytes.Buffer{})
		if err != nil {
			t.Fatal(err)
		}
		if len(calls) != 0 {
			t.Errorf("nothing should be removed, got calls %v", calls)
		}
		if _, err := os.Stat(m.Base); err != nil {
			t.Errorf("data dir should be kept: %v", err)
		}
		if !strings.Contains(out.String(), "Aborted.") {
			t.Errorf("unexpected output: %q", out.String())
		}
	})

	t.Run("confirmed removes data, integration and binary", func(t *testing.T) {
		m := newManager(t)
		fakeInstall(t, m, "8.3")
		var calls []string
		var out bytes.Buffer

		err := selfRemove(m, fakeExe(t), false, false, recordingOps(&calls, nil), strings.NewReader("y\n"), &out, &bytes.Buffer{})
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"integration", "binary"}; !reflect.DeepEqual(calls, want) {
			t.Errorf("calls = %v, want %v", calls, want)
		}
		if _, err := os.Stat(m.Base); !os.IsNotExist(err) {
			t.Errorf("data dir should be gone, stat err = %v", err)
		}
		if !strings.Contains(out.String(), "pvm removed.") {
			t.Errorf("unexpected output: %q", out.String())
		}
	})

	t.Run("--php removes installed versions first", func(t *testing.T) {
		m := newManager(t)
		fakeInstall(t, m, "8.2")
		fakeInstall(t, m, "8.3")
		var calls []string

		err := selfRemove(m, fakeExe(t), true, true, recordingOps(&calls, nil), strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"php 8.2", "php 8.3", "integration", "binary"}; !reflect.DeepEqual(calls, want) {
			t.Errorf("calls = %v, want %v", calls, want)
		}
	})

	t.Run("stops when a PHP version cannot be removed", func(t *testing.T) {
		m := newManager(t)
		fakeInstall(t, m, "8.3")
		var calls []string
		ops := recordingOps(&calls, nil)
		ops.removeVersion = func(_, _ string) error { return errors.New("apt failed") }

		err := selfRemove(m, fakeExe(t), true, true, ops, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "remove PHP 8.3") {
			t.Fatalf("error = %v, want remove PHP 8.3 failure", err)
		}
		if len(calls) != 0 {
			t.Errorf("nothing else should run, got calls %v", calls)
		}
		if _, err := os.Stat(m.Base); err != nil {
			t.Errorf("data dir should be kept: %v", err)
		}
	})

	t.Run("integration failure is only a warning", func(t *testing.T) {
		m := newManager(t)
		var calls []string
		ops := recordingOps(&calls, nil)
		ops.removeIntegration = func(_, _ string) error { return errors.New("no powershell") }
		var errOut bytes.Buffer

		err := selfRemove(m, fakeExe(t), false, true, ops, strings.NewReader(""), &bytes.Buffer{}, &errOut)
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
		m := newManager(t)
		var calls []string
		exe := fakeExe(t)
		binErr := fmt.Errorf("%w: cannot write to %s", selfupdate.ErrPermission, filepath.Dir(exe))

		err := selfRemove(m, exe, false, true, recordingOps(&calls, binErr), strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
		if !errors.Is(err, selfupdate.ErrPermission) || !strings.Contains(err.Error(), exe) {
			t.Fatalf("error = %v, want permission error naming %s", err, exe)
		}
		if _, err := os.Stat(m.Base); !os.IsNotExist(err) {
			t.Errorf("data dir should be gone before the binary, stat err = %v", err)
		}
	})

	t.Run("refuses to remove the home directory", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("no home directory")
		}
		m := newManager(t)
		m.Base = home
		var calls []string

		err = selfRemove(m, fakeExe(t), false, true, recordingOps(&calls, nil), strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "refusing to remove") {
			t.Fatalf("error = %v, want refusal", err)
		}
		if len(calls) != 0 {
			t.Errorf("nothing should run, got calls %v", calls)
		}
	})
}

func TestConfirm(t *testing.T) {
	for input, want := range map[string]bool{"y\n": true, " YES \r\n": true, "n\n": false, "\n": false, "": false} {
		if got := confirm(strings.NewReader(input), &bytes.Buffer{}, "? "); got != want {
			t.Errorf("confirm(%q) = %v, want %v", input, got, want)
		}
	}
}
