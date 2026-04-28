//go:build windows

package symlink

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func SetCurrent(base, version, binaryPath string) error {
	return setCurrentWindows(base, version, binaryPath)
}

func RemoveCurrent(base string) error {
	return fmt.Errorf("to be implemented: remove current version on Windows (manual PATH cleanup required)")
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
