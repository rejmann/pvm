# Development Guide

**Go never runs on your machine.** Every `make` target builds, tests or runs pvm inside Docker through [compose.yaml](../compose.yaml), so the only requirements are `make` and Docker Compose — no local Go toolchain, git or Unix tools. Do not run `go build`, `go test` or `go run` directly.

The Makefile only calls `docker compose` (and `docker version`, to learn the host OS/arch), with no shell syntax, so it works the same from bash, zsh, PowerShell or `cmd.exe` — on Windows, install Docker Desktop and GNU make (e.g. `winget install GnuWin32.Make` or `choco install make`). Anything that needs a shell lives in [cli/](../cli) and runs inside the containers:

| Script | Runs in | Does |
|--------|---------|------|
| `cli/go-entrypoint` | `go` service | creates `dist/` and `.local/`, then runs the command as the owner of the project directory (so outputs are yours, without `id -u` on the host) |
| `cli/build` | `go` service | `go build` with the version from `git describe` |
| `cli/pvm` | `app-pvm` container | refreshes `~/.pvm/bin/pvm` (where the README installs it) from the build, then runs `pvm` or `bash` |
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

**pvm under development always runs inside a Docker container, never on your machine.** `pvm install`, `remove` and `ext` call `sudo apt` on Linux, and every command uses `~/.pvm`, so a dev build run on the host would change your real pvm and your system PHP.

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

The container runs a copy of the build at `~/.pvm/bin/pvm`, which `make pvm` / `make shell` refresh with `cp -u` — only when `.local/bin/pvm` is newer. That lets `make pvm self-upgrade` replace it with a real release: the upgraded binary stays in use until you change the code (the rebuild is newer, so it is copied back) or run `make down`. Flags after `pvm` need `--` or `ARGS`, since make would parse them itself: `make pvm -- run 8.5 --file teste.php`.

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

Tests must not touch the real system: use `home.New(t.TempDir())` as the pvm home and give `pvm.Manager` fakes of its interfaces (`Installer`, `Activator`, `version.Resolver`; for `pvm.Composer` also `ExtensionInstaller`, `ComposerSource` and the `Probe`/`Exec` functions). The real `installer.System` runs `sudo` on Linux and `shim.Activator` writes to the pvm home, so they only run in the container (`make pvm`).

## Project conventions

The layout follows common Go conventions — see [architecture.md](architecture.md#principles).

- Commands live in `cmd/`: they parse arguments, call `internal/pvm` and print. No rules there.
- Use cases live in `internal/pvm`; what they need from the system comes in through small interfaces declared in that package.
- Only `internal/home` builds paths inside the pvm home.
- `internal/` packages must not import `cmd/`, and `internal/pvm` must not import `installer` or `shim`.
- Accept `io.Writer` for output in every function that prints, to keep things testable.
- Accept `context.Context` as the first argument in every function that does network I/O.
- Platform-specific code goes in `_linux.go` / `_darwin.go` / `_windows.go` files (or `//go:build !windows`), never in a `switch runtime.GOOS`.

## Adding a new command

1. Create `cmd/<name>.go` with a constructor:

```go
func newYourCmd() *cobra.Command {
    return &cobra.Command{
        Use:  "your-command",
        RunE: func(cmd *cobra.Command, args []string) error {
            m := newManager(cmd) // pvm.Manager wired to this system
            // call m, print to cmd.OutOrStdout()
        },
    }
}
```

2. Register it in `NewRootCmd` in `cmd/root.go`.
3. Put the rule itself in `internal/pvm` (with a test using the fakes) and keep only argument parsing and output in the command.
4. Helpers used only by this command go in `cmd/<name>/` (package `<name>`, e.g. `cmd/run/` for `run.go`), with their tests; `cmd/` itself keeps only command files. If a helper is needed by several commands, discuss where it belongs first.

## Adding a new installer backend

`internal/installer` has one `System` type per OS (`linux.go`, `brew.go` for macOS, `windows.go`), selected by build tags, each built with `installer.New(stdout, stderr)`. It satisfies `pvm.Installer` and `pvm.ExtensionInstaller`:

```go
Install(h *home.Dir, ver string) error
Remove(h *home.Dir, ver string) error
AddExtensions(h *home.Dir, ver string, exts []string) error
```

`Install` installs PHP by any means and records the binary with `h.SetBinary(ver, bin)`; extension packages it installs are recorded with `h.AddPackages`. Output goes to the `Stdout`/`Stderr` writers, never straight to `os.Stdout`.

### Adding support for a new Linux package manager

Linux installation is data-driven in `internal/installer/linux.go`. Each package manager is described by a `pkgManagerDef`:

```go
type pkgManagerDef struct {
    bin         string                                          // executable to look up in PATH
    phpPkg      func(branch string) string                      // package name (e.g. "php8.3-cli")
    phpBin      func(branch string) string                      // binary path after install
    installArgs func(pkg string) []string                       // full arg list for the install command
    removeArgs  func(pkg string) []string                       // full arg list for the remove command
    preInstall  func(branch string, stdout, stderr io.Writer) error // optional: add repo before installing
    extPkg      func(branch, ext string) string                 // package of an extension; nil = unmanaged
}
```

To support a new package manager, append an entry to the `packageManagers` slice. `detectPackageManager()` iterates the slice in order and returns the first entry whose `bin` is found in `PATH`.

## Platform-specific base directory

`internal/home` decides the default pvm home (`default_unix.go`, `default_windows.go`). The `PVM_HOME` environment variable overrides it on all platforms; `home.Default()` applies both.

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

`cmd/lts_resolver.go` provides the production implementation via `phpnet.LatestLTS`. In tests, pass a stub:

```go
type stubResolver struct{ v string }
func (s stubResolver) ResolveLTS() (string, error) { return s.v, nil }
```

## Resolving branch → full version

`phpnet.LatestPatch(ctx, branch)` returns the latest full version for a branch (e.g. `"8.3"` → `"8.3.30"`). Used by the Windows installer to construct the download URL.

## Managing the active version

`internal/home` stores the global version; `internal/shim` switches it on the system through one `Activator` per OS (`shim.New(stdout, stderr)`), which satisfies `pvm.Activator`:

```go
ver, err := h.Current()                        // home: read current-version
err := activator.Activate(h, "8.3", "/usr/bin/php8.3") // shim + current-version
err := activator.Deactivate(h)
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

`internal/phpnet/releases.go` uses two endpoints:

| Endpoint | Returns |
|----------|---------|
| `https://www.php.net/releases/index.php?json` | Map of active major versions and their supported branches |
| `https://www.php.net/releases/index.php?json&max=500&version=<N>` | All patch releases for a major version |

Both return JSON, decoded by the generic `getJSON[T any]` in `internal/phpnet/http.go`.

## Directory layout recap

| Path (Linux/macOS) | Path (Windows) | Purpose |
|---|---|---|
| `~/.pvm/versions/<ver>/binary` | `%LOCALAPPDATA%\pvm\versions\<ver>\binary` | Path to the PHP binary for `<ver>` |
| `~/.pvm/bin/php` | `%LOCALAPPDATA%\pvm\shims\php.bat` | Shim |
| `~/.pvm/bin/pvm` | `%LOCALAPPDATA%\Programs\pvm\pvm.exe` | pvm itself |
| `~/.pvm/current-version` | `%LOCALAPPDATA%\pvm\current-version` | Active version name |
| `~/.pvm/php/<branch>/` | `%LOCALAPPDATA%\pvm\php\<branch>\` | Extracted PHP install (Windows only) |
