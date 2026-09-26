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
	start, end, ok := findPowerShellWrapper(profile)
	if !ok {
		return profile + "\n" + wrapper, true
	}

	updated = profile[:start] + wrapper + profile[end:]
	return updated, updated != profile
}

// removePowerShellWrapper returns profile without the pvm wrapper block and
// the blank line written before it. changed is false when there is no block.
func removePowerShellWrapper(profile string) (updated string, changed bool) {
	start, end, ok := findPowerShellWrapper(profile)
	if !ok {
		return profile, false
	}
	if start > 0 && profile[start-1] == '\n' {
		start--
	}
	return profile[:start] + profile[end:], true
}

// findPowerShellWrapper locates the wrapper block in profile, including the
// line break after its end marker. An unterminated block runs to the end.
func findPowerShellWrapper(profile string) (start, end int, ok bool) {
	start = strings.Index(profile, psWrapperStart)
	if start == -1 {
		return 0, 0, false
	}

	end = strings.Index(profile[start:], psWrapperEnd)
	if end == -1 {
		return start, len(profile), true
	}
	end += start + len(psWrapperEnd)
	if end < len(profile) && profile[end] == '\r' {
		end++
	}
	if end < len(profile) && profile[end] == '\n' {
		end++
	}
	return start, end, true
}

// psSingleQuote escapes s for use inside a single-quoted PowerShell string.
func psSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
