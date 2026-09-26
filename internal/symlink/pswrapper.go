package symlink

import (
	"fmt"
	"strings"
)

const (
	psWrapperStart = "# pvm-wrapper"
	psWrapperEnd   = "# end pvm-wrapper"
)

// powerShellWrapper returns the profile block that makes `pvm use` update
// $env:PATH in the current session. It must parse on Windows PowerShell 5.1,
// so PowerShell 7-only syntax (?., ??, Join-String) is off limits.
func powerShellWrapper(pvmExe, shimDir string) string {
	return fmt.Sprintf(`%s — managed by pvm, do not edit this block manually
function Invoke-PVM {
    $exe = '%s'
    if (-not (Test-Path $exe)) {
        $cmd = Get-Command pvm.exe -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
        if (-not $cmd) { Write-Error "pvm.exe not found"; return }
        $exe = $cmd.Source
    }
    & $exe @args
    if ($LASTEXITCODE -eq 0 -and $args.Count -gt 0 -and $args[0] -eq 'use') {
        $shimDir = '%s'
        $rest = @($env:PATH -split ';' | Where-Object { $_ -and $_ -ne $shimDir })
        $env:PATH = (@($shimDir) + $rest) -join ';'
    }
}
Set-Alias -Name pvm -Value Invoke-PVM -Force
%s
`, psWrapperStart, psSingleQuote(pvmExe), psSingleQuote(shimDir), psWrapperEnd)
}

// upsertPowerShellWrapper returns profile with the pvm wrapper block set to
// wrapper, replacing an existing block (e.g. one written by an older pvm) or
// appending a new one. changed is false when the profile is already current.
func upsertPowerShellWrapper(profile, wrapper string) (updated string, changed bool) {
	start := strings.Index(profile, psWrapperStart)
	if start == -1 {
		return profile + "\n" + wrapper, true
	}

	end := strings.Index(profile[start:], psWrapperEnd)
	if end == -1 {
		// unterminated block: drop everything from the start marker on
		return profile[:start] + wrapper, true
	}
	end += start + len(psWrapperEnd)
	if end < len(profile) && profile[end] == '\r' {
		end++
	}
	if end < len(profile) && profile[end] == '\n' {
		end++
	}

	updated = profile[:start] + wrapper + profile[end:]
	return updated, updated != profile
}

// psSingleQuote escapes s for use inside a single-quoted PowerShell string.
func psSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
