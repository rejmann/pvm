# Development Guide

## Building

```sh
go build -o pvm .
./pvm available
```

Cross-compile for Windows from Linux/macOS:

```sh
GOOS=windows GOARCH=amd64 go build -o pvm.exe .
```

## Running tests

```sh
go test ./...
```

## Project conventions

- Commands live in `cmd/` and only wire packages together — no business logic.
- All business logic goes in `internal/` packages.
- `internal/` packages must not import `cmd/`.
- Accept `io.Writer` for output in every function that prints, to keep things testable.
- Accept `context.Context` as the first argument in every function that does I/O.
- Use build tags or `_<os>.go` filename suffixes for platform-specific code.

## Adding a new command

1. Create `cmd/<name>.go` with a `var <Name>Cmd = &cobra.Command{...}`.
2. Register it in `main.go`:

```go
cmds := []*cobra.Command{
    cmd.AvailableCmd,
    cmd.InstallCmd,
    cmd.YourNewCmd,   // add here
}
```

3. Put shared helpers (e.g. `baseDir()`) in `cmd/env.go`.

## Adding a new installer backend

`installer.Install` dispatches by `runtime.GOOS` in `internal/installer/select.go`. Each backend satisfies the `InstallerFunc` type defined in `cmd/install.go`:

```go
type InstallerFunc func(base, ver string) error
```

The backend is responsible for:
- Installing the PHP binary by any means.
- Writing the resolved binary path to `<base>/versions/<ver>/binary`.

## Platform-specific base directory

`cmd/env.go` (Linux/macOS) and `cmd/env_windows.go` (Windows) both define `baseDir()` using build tags. The `PVM_HOME` environment variable overrides the default on all platforms.

| OS | Default |
|----|---------|
| Linux / macOS | `~/.pvm` |
| Windows | `%LOCALAPPDATA%\pvm` |

## Resolving version aliases

`internal/version/lts.go` defines the `Resolver` interface:

```go
type Resolver interface {
    ResolveLTS() (string, error)
}
```

`cmd/lts_resolver.go` provides the production implementation via `php.LatestLTS`. In tests, pass a stub:

```go
type stubResolver struct{ v string }
func (s stubResolver) ResolveLTS() (string, error) { return s.v, nil }
```

## Resolving branch → full version

`php.LatestPatch(ctx, branch)` returns the latest full version for a branch (e.g. `"8.3"` → `"8.3.30"`). Used by the Windows installer to construct the download URL.

## Managing the active version

`internal/symlink` owns the shim/symlink and `current-version` file:

```go
err := symlink.SetCurrent(m.Base, "8.3", "/usr/bin/php8.3")
err := symlink.RemoveCurrent(m.Base)
ver, err := symlink.GetCurrent(m.Base)
```

## Windows: download URL structure

`windows_download.go` tries URLs in this order per version:

```
https://windows.php.net/downloads/releases/php-<ver>-nts-Win32-<vc>-x64.zip
https://windows.php.net/downloads/releases/php-<ver>-Win32-<vc>-x64.zip
https://windows.php.net/downloads/releases/archives/php-<ver>-nts-Win32-<vc>-x64.zip
https://windows.php.net/downloads/releases/archives/php-<ver>-Win32-<vc>-x64.zip
```

VC version mapping:

| PHP | VC |
|-----|----|
| 8.x | vs16 |
| 7.2 – 7.4 | vc15 |
| 7.0 – 7.1 | vc14 |
| 5.x | vc11 |

## php.net API

`internal/php/releases.go` uses two endpoints:

| Endpoint | Returns |
|----------|---------|
| `https://www.php.net/releases/index.php?json` | Map of active major versions and their supported branches |
| `https://www.php.net/releases/index.php?json&max=500&version=<N>` | All patch releases for a major version |

Both return JSON. The HTTP client (`internal/php/http_request.go`) is generic over the response type using Go generics (`httpRequest[T any]`).

## Directory layout recap

| Path (Linux/macOS) | Path (Windows) | Purpose |
|---|---|---|
| `~/.pvm/versions/<ver>/binary` | `%LOCALAPPDATA%\pvm\versions\<ver>\binary` | Path to the PHP binary for `<ver>` |
| `~/.pvm/bin/php` | — | Symlink to active binary (Linux) |
| `~/.pvm/shims/php` | `%LOCALAPPDATA%\pvm\shims\php.bat` | Shim to active binary (macOS/Windows) |
| `~/.pvm/current-version` | `%LOCALAPPDATA%\pvm\current-version` | Active version name |
| `~/.pvm/php/<branch>/` | `%LOCALAPPDATA%\pvm\php\<branch>\` | Extracted PHP install (Windows only) |
