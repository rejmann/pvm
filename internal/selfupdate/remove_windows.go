//go:build windows

package selfupdate

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

const createNoWindow = 0x08000000

// RemoveBinary deletes exe. Windows cannot delete a running executable, so it
// is moved aside (which also proves the directory is writable) and a detached
// cmd.exe deletes it — and its directory, if left empty — once pvm has exited.
func RemoveBinary(exe string) error {
	RemoveOld(exe)

	dir := filepath.Dir(exe)
	old := fmt.Sprintf("%s.old-%d", exe, time.Now().UnixNano())
	if err := os.Rename(exe, old); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return wrapRemove(err, dir)
	}

	script := fmt.Sprintf(`ping -n 3 127.0.0.1 >nul & del /f /q "%s" & rmdir "%s" 2>nul`, old, dir)
	cmd := exec.Command("cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// raw command line: Go's argument quoting is not what cmd.exe expects
		CmdLine:       `cmd.exe /d /s /c "` + script + `"`,
		CreationFlags: createNoWindow | syscall.CREATE_NEW_PROCESS_GROUP,
	}
	if err := cmd.Start(); err != nil {
		_ = os.Rename(old, exe)
		return fmt.Errorf("schedule binary removal: %w", err)
	}
	return cmd.Process.Release()
}
