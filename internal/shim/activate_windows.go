package shim

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rejmann/pvm/internal/home"
	pvmversion "github.com/rejmann/pvm/internal/version"
)

// Activator switches the global version with a php.bat shim that reads
// current-version, and puts the shim directory on the user PATH.
type Activator struct {
	Stdout, Stderr io.Writer
}

// New returns the activator for this system.
func New(stdout, stderr io.Writer) *Activator {
	return &Activator{Stdout: stdout, Stderr: stderr}
}

// Deactivate is not supported on Windows yet.
func (a *Activator) Deactivate(*home.Dir) error {
	return fmt.Errorf("to be implemented: remove current version on Windows (manual PATH cleanup required)")
}

// Activate makes version the global one.
func (a *Activator) Activate(h *home.Dir, version, binaryPath string) error {
	// prefer the deterministic install dir over whatever is stored in the binary file
	if p := windowsInstalledBinary(h, version); p != "" {
		binaryPath = p
	}

	if binaryPath == "" {
		return fmt.Errorf("PHP %s binary not found — run: pvm install %s", version, version)
	}

	shimDir := h.ShimDir()
	if err := os.MkdirAll(shimDir, 0755); err != nil {
		return fmt.Errorf("create shims directory: %w", err)
	}

	// dynamic shim: reads current-version at runtime so switching versions
	// only requires updating the current-version file — no PATH changes needed
	content := fmt.Sprintf(
		"@echo off\r\n"+
			"set /p PHP_VER=<\"%s\"\r\n"+
			"\"%s\\%%PHP_VER%%\\php.exe\" %%*\r\n",
		h.CurrentFile(),
		h.PHPRoot(),
	)
	shimPath := filepath.Join(shimDir, "php.bat")
	if err := os.WriteFile(shimPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write php shim: %w", err)
	}

	prependToUserPath(shimDir)
	installPowerShellWrapper(shimDir)

	return h.SetCurrent(version)
}

// installPowerShellWrapper writes a pvm wrapper function to the user's
// PowerShell profile so that `pvm use` updates $env:PATH in the current
// session automatically. A block left by an older pvm is replaced.
// Best-effort — silently ignored on failure.
func installPowerShellWrapper(shimDir string) {
	pvmExe, err := os.Executable()
	if err != nil {
		return
	}

	profilePath, err := powerShellProfilePath()
	if err != nil {
		return
	}

	existing, _ := os.ReadFile(profilePath)
	updated, changed := upsertPowerShellWrapper(string(existing), powerShellWrapper(pvmExe, shimDir))
	if !changed {
		return
	}

	_ = os.MkdirAll(filepath.Dir(profilePath), 0755)
	_ = os.WriteFile(profilePath, []byte(updated), 0644)
}

// RemoveIntegration undoes what pvm set up outside its data directory: the
// shim directory and binDir entries in the user PATH, and the wrapper in the
// PowerShell profile.
func (a *Activator) RemoveIntegration(h *home.Dir, binDir string) error {
	if err := removeFromUserPath(h.ShimDir(), binDir); err != nil {
		return fmt.Errorf("update user PATH: %w", err)
	}

	profilePath, err := powerShellProfilePath()
	if err != nil {
		return fmt.Errorf("locate PowerShell profile: %w", err)
	}
	existing, err := os.ReadFile(profilePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read PowerShell profile: %w", err)
	}
	updated, changed := removePowerShellWrapper(string(existing))
	if !changed {
		return nil
	}
	if err := os.WriteFile(profilePath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("write PowerShell profile: %w", err)
	}
	return nil
}

// powerShellProfilePath returns $PROFILE of Windows PowerShell.
func powerShellProfilePath() (string, error) {
	out, err := exec.Command(
		"powershell", "-NoProfile", "-NonInteractive", "-Command", "$PROFILE",
	).Output()
	if err != nil {
		return "", err
	}
	p := strings.TrimSpace(string(out))
	if p == "" {
		return "", errors.New("$PROFILE is empty")
	}
	return p, nil
}

// removeFromUserPath drops dirs from the current user PATH in the Windows
// registry.
func removeFromUserPath(dirs ...string) error {
	quoted := make([]string, len(dirs))
	for i, d := range dirs {
		quoted[i] = "'" + psSingleQuote(d) + "'"
	}
	script := fmt.Sprintf(`
$dirs = @(%s)
$current = [System.Environment]::GetEnvironmentVariable('Path', 'User')
$parts = @($current -split ';' | Where-Object { $_ -and $dirs -notcontains $_.TrimEnd('\') })
[System.Environment]::SetEnvironmentVariable('Path', ($parts -join ';'), 'User')
`, strings.Join(quoted, ", "))

	return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Run()
}

// prependToUserPath adds dir to the front of the current user PATH in the
// Windows registry. It is a best-effort call — errors are silently ignored
// because the shim already works if the user manually adds the directory.
func prependToUserPath(dir string) {
	script := fmt.Sprintf(`
$dir = '%s'
$current = [System.Environment]::GetEnvironmentVariable('Path', 'User')
$parts = $current -split ';' | Where-Object { $_ -ne $dir -and $_ -ne '' }
$newPath = ($dir + ';' + ($parts -join ';')).TrimEnd(';')
[System.Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
`, dir)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	_ = cmd.Run()
}

// windowsInstalledBinary returns the php.exe path from the pvm-managed install
// directory (php/<major>.<minor>/php.exe) if it exists.
func windowsInstalledBinary(h *home.Dir, version string) string {
	p := filepath.Join(h.PHPDir(pvmversion.Branch(version)), "php.exe")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}
