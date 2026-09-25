# Development Guide

**Go never runs on your machine.** Every `make` target builds, tests or runs pvm inside Docker through [compose.yaml](../compose.yaml), so the only requirements are `make` and Docker Compose — no local Go toolchain. Do not run `go build`, `go test` or `go run` directly.

```sh
make setup        # build the Docker images (the other targets also run it, cached)
```

| Target | What it does (all in Docker) |
|--------|------------------------------|
| `make setup` | `docker compose build` — images with the current code |
| `make run <args>` | Runs `pvm <args>` in the running Ubuntu container (`pvm` service) |
| `make shell` | Opens bash in that container |
| `make up` / `make down` | Starts / removes the container (`down` resets installed PHP versions) |
| `make test` | `go test ./...` |
| `make lint` | `go vet ./...` |
| `make build` | Binary for your OS/arch → `dist/pvm` |
| `make build-all` | All release platforms → `dist/pvm-<os>-<arch>` |

## Running pvm during development

**pvm under development always runs inside a Docker container, never on your machine.** `pvm install`, `use` and `remove` call `sudo apt` and `sudo update-alternatives` on Linux, so a dev build run on the host would change your real `~/.pvm` and your system PHP.

```sh
make run install 8.5
make run use 8.5
make run list
make run -- -v        # pvm flags need `--`, otherwise make parses them
make shell            # then: php -v → PHP 8.5.x
make down             # back to a clean container
```

`make run` starts the `pvm` service in the background (`make up`) — the Dockerfile's `runtime` stage, Ubuntu 24.04 with apt and the ondrej/php PPA — and runs `pvm <args>` in it with `docker compose exec`. The container keeps running between commands, so installed PHP versions, the active version and `.php-version` files persist until `make down`. Nothing is shared with the host.

If the code changed, the next `make run` rebuilds the image and recreates the container with the new pvm, which also starts from a clean state.

No Makefile target may touch the pvm installed on your machine: `make test` and `make lint` only compile and test code (tests use `t.TempDir()`), and the `build*` targets only write to `dist/`.

## Building

```sh
make build        # dist/pvm for your OS/arch
make build-all    # dist/pvm-linux-amd64, pvm-darwin-arm64, pvm-windows-amd64.exe, ...
```

Binaries are compiled by the `go` service, which mounts `./dist` as a volume and runs as your user, so files in `dist/` are owned by you. The version shown by `pvm --version` comes from `git describe` (defaults to `dev`).

Do not run a `dist/` binary on your machine for commands that change state (`install`, `use`, `remove`, `local`) — use `make run` instead (see above).

For a one-off Go command that writes to the repo (e.g. `go mod tidy`), run the Go image with the repo mounted instead of a local toolchain:

```sh
docker run --rm -u "$(id -u):$(id -g)" -e GOCACHE=/tmp/cache -e GOMODCACHE=/tmp/mod \
  -v "$PWD":/src -w /src golang:1.27-alpine go mod tidy
```

## Running tests

```sh
make test
make lint
```

CI (`.github/workflows/ci.yml`) runs `go vet` and `go test -race` on Linux, macOS and Windows for every pull request.

Tests must not touch the real system: use `t.TempDir()` as the pvm base dir, and inject fake `InstallerFunc` / `RemoverFunc` / `version.Resolver` values. Code paths that call `symlink.SetCurrent` or `symlink.RemoveCurrent` run `sudo update-alternatives` on Linux, so they are intentionally left out of unit tests.

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
    cmd.ListCmd,
    cmd.UseCmd,
    cmd.RemoveCmd,
    cmd.CurrentCmd,
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

### Adding support for a new Linux package manager

Linux installation is handled by `internal/installer/linux.go` using a data-driven approach. Each package manager is described by a `pkgManagerDef` struct:

```go
type pkgManagerDef struct {
    bin         string                        // executable to look up in PATH
    phpPkg      func(branch string) string    // package name (e.g. "php8.3-cli")
    phpBin      func(branch string) string    // binary path after install
    installArgs func(pkg string) []string     // full arg list for the install command
    removeArgs  func(pkg string) []string     // full arg list for the remove command
    preInstall  func(branch string) error     // optional: add repo before installing
}
```

To support a new package manager, append an entry to the `packageManagers` slice in `linux.go`. `detectPackageManager()` iterates the slice in order and returns the first entry whose `bin` is found in `PATH`.

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
