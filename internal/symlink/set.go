package symlink

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rejmann/pvm/system"
)

func SetCurrent(base, version, binaryPath string) error {
	switch runtime.GOOS {
	case system.Linux:
		return setCurrentLinux(base, version, binaryPath)
	case system.Windows:
		return setCurrentWindows(base, version, binaryPath)
	default:
		return setCurrentShim(base, version, binaryPath)
	}
}

func RemoveCurrent(base string) error {
	switch runtime.GOOS {
	case system.Linux:
		return removeCurrentLinux(base)
	default:
		return removeCurrentShim(base)
	}
}

func setCurrentLinux(base, version, binaryPath string) error {
	cmd := exec.Command("sudo", "update-alternatives", "--set", "php", binaryPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("update-alternatives --set php %s: %w", binaryPath, err)
	}

	if err := updateSymlink(filepath.Join(base, "bin", "php"), binaryPath); err != nil {
		return fmt.Errorf("update pvm bin symlink: %w", err)
	}

	localBin := filepath.Join(filepath.Dir(base), ".local", "bin", "php")
	if fi, err := os.Lstat(localBin); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if err := updateSymlink(localBin, binaryPath); err != nil {
			return fmt.Errorf("update ~/.local/bin/php symlink: %w", err)
		}
	}

	return writeCurrentVersion(base, version)
}

func removeCurrentLinux(base string) error {
	cmd := exec.Command("sudo", "update-alternatives", "--auto", "php")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("update-alternatives --auto php: %w", err)
	}
	return removeCurrentVersion(base)
}

func updateSymlink(link, target string) error {
	if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
		return err
	}
	tmp := link + ".tmp"
	os.Remove(tmp)
	if err := os.Symlink(target, tmp); err != nil {
		return err
	}
	return os.Rename(tmp, link)
}

func setCurrentShim(base, version, binaryPath string) error {
	shimDir := filepath.Join(base, "shims")
	if err := os.MkdirAll(shimDir, 0755); err != nil {
		return fmt.Errorf("create shims directory: %w", err)
	}

	shimPath := filepath.Join(shimDir, "php")
	_ = os.Remove(shimPath)
	if err := os.Symlink(binaryPath, shimPath); err != nil {
		return fmt.Errorf("create php shim: %w", err)
	}

	return writeCurrentVersion(base, version)
}

func removeCurrentShim(base string) error {
	shimPath := filepath.Join(base, "shims", "php")
	if err := os.Remove(shimPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove php shim: %w", err)
	}
	return removeCurrentVersion(base)
}

func setCurrentWindows(base, version, binaryPath string) error {
	// prefer the deterministic install dir over whatever is stored in the binary file
	if p := windowsInstalledBinary(base, version); p != "" {
		binaryPath = p
	}

	if binaryPath == "" {
		return fmt.Errorf("PHP %s binary not found — run: pvm install %s", version, version)
	}

	shimDir := filepath.Join(base, "shims")
	if err := os.MkdirAll(shimDir, 0755); err != nil {
		return fmt.Errorf("create shims directory: %w", err)
	}

	// dynamic shim: reads current-version at runtime so switching versions
	// only requires updating the current-version file — no PATH changes needed
	content := fmt.Sprintf(
		"@echo off\r\n"+
			"set /p PHP_VER=<\"%s\"\r\n"+
			"\"%s\\%%PHP_VER%%\\php.exe\" %%*\r\n",
		filepath.Join(base, "current-version"),
		filepath.Join(base, "php"),
	)
	shimPath := filepath.Join(shimDir, "php.bat")
	if err := os.WriteFile(shimPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write php shim: %w", err)
	}

	prependToUserPath(shimDir)
	installPowerShellWrapper(base, shimDir)

	return writeCurrentVersion(base, version)
}

// installPowerShellWrapper appends a pvm wrapper function to the user's
// PowerShell profile so that `pvm use` updates $env:PATH in the current
// session automatically. Best-effort — silently ignored on failure.
func installPowerShellWrapper(base, shimDir string) {
	profileOut, err := exec.Command(
		"powershell", "-NoProfile", "-NonInteractive", "-Command", "$PROFILE",
	).Output()
	if err != nil {
		return
	}
	profilePath := strings.TrimSpace(string(profileOut))
	if profilePath == "" {
		return
	}

	existing, _ := os.ReadFile(profilePath)
	if strings.Contains(string(existing), "# pvm-wrapper") {
		return
	}

	wrapper := fmt.Sprintf(`
# pvm-wrapper — managed by pvm, do not edit this block manually
function Invoke-PVM {
    $exe = "%s\pvm.exe"
    if (-not (Test-Path $exe)) { $exe = (Get-Command pvm.exe -ErrorAction SilentlyContinue)?.Source }
    if (-not $exe) { Write-Error "pvm.exe not found"; return }
    & $exe @args
    if ($LASTEXITCODE -eq 0 -and $args.Count -gt 0 -and $args[0] -eq 'use') {
        $shimDir = "%s"
        $env:PATH = $shimDir + ';' + (($env:PATH -split ';') | Where-Object { $_ -ne $shimDir -and $_ } | Join-String -Separator ';')
    }
}
Set-Alias -Name pvm -Value Invoke-PVM -Force
# end pvm-wrapper
`, base, shimDir)

	_ = os.MkdirAll(filepath.Dir(profilePath), 0755)
	f, err := os.OpenFile(profilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(wrapper)
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
// directory (base/php/<major>.<minor>/php.exe) if it exists.
func windowsInstalledBinary(base, version string) string {
	branch := versionBranch(version)
	p := filepath.Join(base, "php", branch, "php.exe")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

func versionBranch(version string) string {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return version
}

func writeCurrentVersion(base, version string) error {
	if err := os.WriteFile(filepath.Join(base, "current-version"), []byte(version), 0644); err != nil {
		return fmt.Errorf("write current-version: %w", err)
	}
	return nil
}

func removeCurrentVersion(base string) error {
	vf := filepath.Join(base, "current-version")
	if err := os.Remove(vf); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove current-version: %w", err)
	}
	return nil
}
