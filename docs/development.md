# Development Guide

## Building

```sh
go build -o pvm .
./pvm available
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

`installer.Apt` satisfies the `InstallerFunc` type defined in `cmd/install.go`:

```go
type InstallerFunc func(base, ver string) error
```

To add a new backend (e.g. compile from source):

1. Create `internal/installer/source.go` and implement `func Source(base, ver string) error`.
2. In `cmd/install.go`, select the backend based on a flag or runtime detection and pass it to `installVersion`.

The backend is responsible for:
- Installing the PHP binary by any means.
- Writing the resolved binary path to `<base>/versions/<ver>/binary`.

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

## Directory layout recap

| Path | Purpose |
|------|---------|
| `~/.pvm/versions/<ver>/binary` | Path to the installed PHP binary for version `<ver>` |
| `~/.pvm/bin/` | (planned) Symlinks — `php` → active version |
| `$PVM_HOME` | Overrides `~/.pvm` when set |

## php.net API

`internal/php/releases.go` uses two endpoints:

| Endpoint | Returns |
|----------|---------|
| `https://www.php.net/releases/index.php?json` | Map of active major versions and their supported branches |
| `https://www.php.net/releases/index.php?json&max=500&version=<N>` | All patch releases for a major version |

Both return JSON. The HTTP client (`internal/php/http_request.go`) is generic over the response type using Go generics (`httpRequest[T any]`).
