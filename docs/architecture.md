# Architecture

## Package layout

```
pvm/
├── main.go                  # Entry point — registers Cobra commands
├── cmd/
│   ├── available.go         # `pvm available` command
│   ├── install.go           # `pvm install` command
│   ├── list.go              # `pvm list` command
│   ├── lts_resolver.go      # Bridges cobra context → php.LatestLTS
│   ├── env.go               # baseDir() — resolves ~/.pvm or $PVM_HOME
│   └── use.go               # printPathHint() — PATH guidance after install
└── internal/
    ├── php/
    │   ├── types.go         # Branch, Release, Status types
    │   ├── releases.go      # Fetches branches from php.net JSON API
    │   ├── detect.go        # DetectSystem() — finds PHP binaries outside pvm
    │   ├── util.go          # Version string parsing helpers
    │   └── http_request.go  # Generic HTTP client with JSON decoding
    ├── fs/
    │   ├── manager.go       # Manager — wraps ~/.pvm directory structure
    │   ├── binary.go        # Read/write binary path; VersionInstalled check
    │   └── versions.go      # InstalledVersions() — sorted list from versions dir
    ├── installer/
    │   └── apt.go           # APT installer (ondrej/php PPA)
    ├── symlink/
    │   ├── get.go           # GetCurrent() — reads ~/.pvm/current-version
    │   └── set.go           # SetCurrent() / RemoveCurrent() — atomic symlink swap
    └── version/
        ├── version.go       # Version struct, Parse(), and Compare()
        └── lts.go           # Resolver interface and Resolve() for aliases
```

## Runtime directory structure

```
~/.pvm/                        (or $PVM_HOME)
├── current-version            # plain text: "8.4" — written by pvm use (planned)
├── bin/
│   └── php                   # symlink → /usr/bin/php8.4 — add this dir to $PATH
└── versions/
    ├── 8.3.20/
    │   └── binary             # plain text: /usr/bin/php8.3
    └── 8.4.6/
        └── binary             # plain text: /usr/bin/php8.4
```

## Data flow — `pvm available`

```
cmd.runAvailable
  └─ php.FetchAllBranches(ctx)
       ├─ GET php.net/releases/?json          → SupportedResponse (active majors)
       └─ goroutine per major version
            └─ GET php.net/releases/?version=N → MajorResponse (all patches)
                 └─ latestPatchPerBranch()     → map[branch]latestVersion
  └─ print table to stdout
```

## Data flow — `pvm install <version|lts>`

```
cmd.runInstall
  └─ version.Resolve(arg, phpLTSResolver)
       └─ if alias "lts" → php.LatestLTS(ctx) → fetchReleases() → highest supported branch
  └─ version.Parse(concrete)       — validates format
  └─ fs.Manager.EnsureBaseDir()    — mkdir ~/.pvm/versions
  └─ fs.Manager.VersionInstalled() — checks binary file + stat
  └─ installer.Apt(base, ver)
       └─ sudo apt-get install phpX.Y-cli
       └─ write ~/.pvm/versions/<ver>/binary = /usr/bin/phpX.Y
  └─ printPathHint()               — guides user to add ~/.pvm/bin to PATH
```

## Data flow — `pvm list`

```
cmd.runList
  └─ fs.Manager.InstalledVersions()
       └─ os.ReadDir(~/.pvm/versions/)
            └─ filters: valid version name + VersionInstalled (binary file + stat)
            └─ sorted by Version.Compare()
  └─ symlink.GetCurrent(base)
       └─ reads ~/.pvm/current-version → marks matching entry as "(current)"
  └─ php.DetectSystem()
       └─ glob: /usr/bin/php[0-9]*, /usr/local/bin/php[0-9]*, Homebrew paths
       └─ also: exec.LookPath("php") for plain php in $PATH
       └─ each binary → php -r "echo PHP_MAJOR_VERSION.'.'.PHP_MINOR_VERSION;"
       └─ deduplicates by version string, sorted by Version.Compare()
  └─ prints pvm-managed group, then system group (skipping overlap)
```

## Key design decisions

- **`internal/` boundary** — `php`, `fs`, `installer`, `version` are independent packages with no circular imports. Commands wire them together.
- **`InstallerFunc` type** — `install.go` accepts `func(base, ver string) error`, making it easy to swap `apt` for another backend (e.g. source compilation) without changing the command.
- **`version.Resolver` interface** — decouples alias resolution from the php.net API, enabling unit-testing without network calls.
- **`binary` file** — stores only the resolved binary path. This keeps version detection O(1) (a single file read + stat) and lets the installer be the sole authority on where the binary lives.
- **Concurrent branch fetching** — `FetchAllBranches` launches one goroutine per active major version and fans in the results, reducing latency when php.net is slow.
- **`current-version` file** — a plain-text file holding the active version name; `symlink.GetCurrent` reads it, `symlink.SetCurrent` writes it atomically alongside the `bin/php` symlink swap.
- **Atomic symlink swap** — `SetCurrent` writes to `bin/php.tmp` then `os.Rename`s it over `bin/php`, which is atomic on Linux and avoids a window where `php` is absent.
- **System PHP detection in `pvm list`** — `DetectSystem` runs each candidate binary with `php -r "…"` rather than parsing binary names, so it works regardless of naming conventions (e.g. Homebrew's `opt/php@8.3/bin/php`). Versions already managed by pvm are filtered out of the system group to avoid duplicates.
