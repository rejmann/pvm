# Architecture

## Package layout

```
pvm/
├── main.go                  # Entry point — registers Cobra commands
├── cmd/
│   ├── available.go         # `pvm available` command
│   ├── install.go           # `pvm install` command
│   ├── lts_resolver.go      # Bridges cobra context → php.LatestLTS
│   ├── env.go               # baseDir() — resolves ~/.pvm or $PVM_HOME
│   └── use.go               # printPathHint() — PATH guidance after install
└── internal/
    ├── php/
    │   ├── types.go         # Branch, Release, Status types
    │   ├── releases.go      # Fetches branches from php.net JSON API
    │   ├── util.go          # Version string parsing helpers
    │   └── http_request.go  # Generic HTTP client with JSON decoding
    ├── fs/
    │   ├── manager.go       # Manager — wraps ~/.pvm directory structure
    │   └── binary.go        # Read/write binary path; VersionInstalled check
    ├── installer/
    │   └── apt.go           # APT installer (ondrej/php PPA)
    └── version/
        ├── version.go       # Version struct and Parse()
        └── lts.go           # Resolver interface and Resolve() for aliases
```

## Runtime directory structure

```
~/.pvm/              (or $PVM_HOME)
└── versions/
    ├── 8.3.20/
    │   └── binary   # plain text file: /usr/bin/php8.3
    └── 8.4.6/
        └── binary   # plain text file: /usr/bin/php8.4
```

> The `bin/` directory (for symlinks) is not yet created — it will hold `php` pointing to the active version.

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

## Key design decisions

- **`internal/` boundary** — `php`, `fs`, `installer`, `version` are independent packages with no circular imports. Commands wire them together.
- **`InstallerFunc` type** — `install.go` accepts `func(base, ver string) error`, making it easy to swap `apt` for another backend (e.g. source compilation) without changing the command.
- **`version.Resolver` interface** — decouples alias resolution from the php.net API, enabling unit-testing without network calls.
- **`binary` file** — stores only the resolved binary path. This keeps version detection O(1) (a single file read + stat) and lets the installer be the sole authority on where the binary lives.
- **Concurrent branch fetching** — `FetchAllBranches` launches one goroutine per active major version and fans in the results, reducing latency when php.net is slow.
