# Development Guide

**Go never runs on your machine.** Every `make` target builds, tests or runs pvm inside Docker through [compose.yaml](../compose.yaml), so the only requirements are `make` and Docker Compose — no local Go toolchain, git or Unix tools. Do not run `go build`, `go test` or `go run` directly.

The Makefile only calls `docker compose` (and `docker version`, to learn the host OS/arch), with no shell syntax, so it works the same from bash, zsh, PowerShell or `cmd.exe` — on Windows, install Docker Desktop and GNU make (e.g. `winget install GnuWin32.Make` or `choco install make`). Anything that needs a shell lives in [cli/](../cli) and runs inside the containers:

| Script | Runs in | Does |
|--------|---------|------|
| `cli/go-entrypoint` | `go` service | creates `dist/` and `.local/`, then runs the command as the owner of the project directory (so outputs are yours, without `id -u` on the host) |
| `cli/build` | `go` service | `go build` with the version from `git describe` |
| `cli/pvm` | `app-pvm` container | refreshes `/usr/local/bin/pvm` from the build, then runs `pvm` or `bash` |
| `cli/help.awk` | `go` service | `make help`: lists every `target: ## description` line of the Makefile |

To document a new target, end its rule with `## description` — `make help` picks it up. `.gitattributes` keeps these scripts with LF line endings on Windows checkouts.

```sh
make setup        # build the Docker images (the other targets also run it, cached)
```

| Target | What it does (all in Docker) |
|--------|------------------------------|
| `make setup` | `docker compose build` — the Go toolchain and Ubuntu images (code is mounted, not baked in) |
| `make pvm <args>` | Runs `pvm <args>` in the running Ubuntu container (`pvm` service) |
| `make shell` | Opens bash in that container |
| `make up` / `make down` | Starts / removes the container (`down` resets installed PHP versions) |
| `make test` | `go test ./...` |
| `make lint` | `go vet ./...` |
| `make build` | Binary for your OS/arch → `dist/pvm` |
| `make build-all` | All release platforms → `dist/pvm-<os>-<arch>` |

## Running pvm during development

**pvm under development always runs inside a Docker container, never on your machine.** `pvm install`, `use` and `remove` call `sudo apt` and `sudo update-alternatives` on Linux, so a dev build run on the host would change your real `~/.pvm` and your system PHP.

```sh
make pvm install 8.5
make pvm use 8.5
make pvm list
make pvm -- -v        # pvm flags need `--`, otherwise make parses them
make pvm run 8.5 teste.php
make pvm -- run teste.php -v 8.2   # flags after `pvm` need `--`
make shell            # then: php -v → PHP 8.5.x
make down             # back to a clean container
```

`make pvm` starts the `pvm` service in the background (`make up`) — the Dockerfile's `runtime` stage, Ubuntu 24.04 with apt and the ondrej/php PPA — and runs `pvm <args>` in it with `docker compose exec`. The container keeps running between commands, so installed PHP versions and the active version persist until `make down`. The project is mounted read-only at `/app` (the working directory), so PHP can run files from the repo — `make pvm run 8.5 teste.php` — but nothing in the container can change them (writing a `.php-version` there fails with "read-only file system").

The container runs a copy of the build at `/usr/local/bin/pvm`, which `make pvm` / `make shell` refresh with `cp -u` — only when `.local/bin/pvm` is newer. That lets `make pvm self-upgrade` replace it with a real release: the upgraded binary stays in use until you change the code (the rebuild is newer, so it is copied back) or run `make down`. Flags after `pvm` need `--` or `ARGS`, since make would parse them itself: `make pvm -- run 8.5 --file teste.php`.

Downloaded PHP packages are cached in `.local/apt/archives` (gitignored, mounted into the container), so after `make down` the next `pvm install` of the same version doesn't download anything again. The files there are created by root inside the container; to clear the cache without `sudo`, run `docker compose run --rm --entrypoint rm go -rf .local/apt` (skipping the entrypoint keeps the container as root).

`make pvm` receives its arguments as make goals, so quotes and spaces are lost. For arguments like that, pass them in `ARGS` (on Windows `cmd.exe`, use double quotes inside: single quotes do not group there):

```sh
make pvm ARGS="run 8.5 teste.php 'um argumento com espaços'"
```

pvm itself is not baked into the image: `make pvm` compiles it into `.local/bin/pvm` (gitignored), which the container sees through the project mount. The binary is only rebuilt when a `.go` file, `go.mod` or `go.sum` changes (found with make's own `wildcard`, not `find`), and Go modules and the build cache live in `.local/go` and `.local/go-cache`, so an unchanged `make pvm` takes well under a second. It is built for the architecture of Docker's VM (`docker version`), which is what the container runs. Changing Go code therefore keeps the same container — and the PHP versions installed in it. The container is only recreated when the Dockerfile or `compose.yaml` change, or after `make down`; reinstalling then doesn't download anything thanks to the apt cache.

No Makefile target may touch the pvm installed on your machine: `make test` and `make lint` only compile and test code (tests use `t.TempDir()`), and the `build*` targets only write to `dist/`.

## Building

```sh
make build        # dist/pvm for your OS/arch
make build-all    # dist/pvm-linux-amd64, pvm-darwin-arm64, pvm-windows-amd64.exe, ...
```

Binaries are compiled by the `go` service, which mounts the project at `/src` and runs as the owner of the project directory, so files in `dist/` are owned by you. The version shown by `pvm --version` comes from `git describe`, run inside the container (defaults to `dev`). `make build` targets the OS/arch of your Docker client (`dist/pvm`, or `dist/pvm.exe` on Windows).

Do not run a `dist/` binary on your machine for commands that change state (`install`, `use`, `remove`) — use `make pvm` instead (see above).

For a one-off Go command that writes to the repo (e.g. `go mod tidy`, `go get`), use the `go` service — the repo is mounted and files come out owned by you:

```sh
docker compose run --rm go go mod tidy
docker compose run --rm go gofmt -l .
```

## Running tests

```sh
make test
make lint
```

CI (`.github/workflows/ci.yml`) runs `go vet` and `go test -race` on Linux, macOS and Windows for every pull request.

Tests must not touch the real system: use `home.New(t.TempDir())` as the pvm home, and inject fake `installFunc` / `removeFunc` / `version.Resolver` values. Code paths that call `activate.Use` or `activate.Clear` run `sudo update-alternatives` on Linux, so they are intentionally left out of unit tests.

## Project conventions

- Commands live in `cmd/` and only wire packages together — no business logic.
- All business logic goes in `internal/` packages.
- `internal/` packages must not import `cmd/`.
- Only `internal/home` knows the layout of the pvm home: other packages ask it for paths (`h.ShimDir()`, `h.ComposerDir()`, …) instead of joining `"versions"`, `"current-version"` etc. themselves.
- Accept `io.Writer` (or `proc.Streams`, for functions that run other programs) for output in every function that prints, to keep things testable. Internal packages never write to `os.Stdout` / `os.Stderr` directly.
- HTTP goes through `internal/httpx`, so every request has a context, a timeout and a size limit.
- Accept `context.Context` as the first argument in every function that does I/O.
- Use build tags or `_<os>.go` filename suffixes for platform-specific code.

## Adding a new command

1. Create `cmd/<name>.go` with a constructor `func new<Name>Cmd() *cobra.Command` that declares its flags.
2. Register it in `newRootCmd` in `cmd/root.go`, and add its name to `TestRootCmd`:

```go
root.AddCommand(
    newAvailableCmd(),
    newInstallCmd(),
    // ...
    newYourCmd(), // add here
)
```

3. Get the pvm home with `home.Default()`, and put shared helpers (`streams`, `versionLabel`, `confirm`, …) in `cmd/helpers.go`.

## Adding a new installer backend

Each OS has one backend file, selected by build tags: `linux.go`, `darwin.go` and `windows.go` in `internal/installer`. Every backend defines the same functions:

```go
func Install(ctx context.Context, h *home.Dir, ver string, s proc.Streams) (bin string, err error)
func Remove(h *home.Dir, ver string, s proc.Streams) error
func EnsureExtensions(h *home.Dir, ver string, s proc.Streams) error
```

The backend is responsible for installing the PHP binary by any means and returning its path. `pvm install` then registers it with `h.RegisterVersion(ver, bin)`, which writes `versions/<ver>/binary`.

### Adding support for a new Linux package manager

Linux installation is handled by `internal/installer/linux.go` using a data-driven approach. Each package manager is described by a `pkgManagerDef` struct:

```go
type pkgManagerDef struct {
    bin         string                        // executable to look up in PATH
    phpPkg      func(branch string) string    // package name (e.g. "php8.3-cli")
    phpBin      func(branch string) string    // binary path after install
    installArgs func(pkg string) []string     // full arg list for the install command
    removeArgs  func(pkg string) []string     // full arg list for the remove command
    preInstall  func(branch string, s proc.Streams) error // optional: add repo before installing
}
```

To support a new package manager, append an entry to the `packageManagers` slice in `linux.go`. `detectPackageManager()` iterates the slice in order and returns the first entry whose `bin` is found in `PATH`.

## Platform-specific base directory

`home.Default()` returns the pvm home. The default comes from `internal/home/default_unix.go` (Linux/macOS) and `default_windows.go` (Windows), selected by build tags; the `PVM_HOME` environment variable overrides it on all platforms.

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

`ltsResolver` in `cmd/helpers.go` provides the production implementation via `catalog.LatestLTS`. In tests, pass a stub:

```go
type stubResolver struct{ v string }
func (s stubResolver) ResolveLTS() (string, error) { return s.v, nil }
```

## Resolving branch → full version

`catalog.LatestPatch(ctx, branch)` returns the latest full version for a branch (e.g. `"8.3"` → `"8.3.30"`). Used by the Windows installer to construct the download URL.

## Managing the active version

The global version is state in the pvm home; making it the system's `php` is `internal/activate`'s job:

```go
ver, err := h.Current()                                     // read current-version
err := activate.Use(h, "8.3", "/usr/bin/php8.3", streams)   // shim, update-alternatives / PATH, then h.SetCurrent
err := activate.Clear(h, streams)                           // undo, then h.ClearCurrent
```

Which PHP runs in a directory ($PVM_VERSION → `.php-version` → global) is decided by `resolve.Active(h, dir, env)`; `resolve.Version(h, arg, resolver)` resolves an explicit version or `lts`.

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

`internal/catalog/releases.go` uses two endpoints:

| Endpoint | Returns |
|----------|---------|
| `https://www.php.net/releases/index.php?json` | Map of active major versions and their supported branches |
| `https://www.php.net/releases/index.php?json&max=500&version=<N>` | All patch releases for a major version |

Both return JSON, decoded with `httpx.GetJSON[T]`.

## Directory layout recap

| Path (Linux/macOS) | Path (Windows) | Purpose |
|---|---|---|
| `~/.pvm/versions/<ver>/binary` | `%LOCALAPPDATA%\pvm\versions\<ver>\binary` | Path to the PHP binary for `<ver>` |
| `~/.pvm/bin/php` | — | Symlink to active binary (Linux) |
| `~/.pvm/shims/php` | `%LOCALAPPDATA%\pvm\shims\php.bat` | Shim to active binary (macOS/Windows) |
| `~/.pvm/current-version` | `%LOCALAPPDATA%\pvm\current-version` | Active version name |
| `~/.pvm/php/<branch>/` | `%LOCALAPPDATA%\pvm\php\<branch>\` | Extracted PHP install (Windows only) |
