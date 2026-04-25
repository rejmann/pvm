# Architecture

## Package layout

```
pvm/
├── main.go                      # Entry point — registers Cobra commands
├── system/
│   └── system.go                # OS constants (Linux, Darwin, Windows)
├── cmd/
│   ├── available.go             # `pvm available` command
│   ├── install.go               # `pvm install` command
│   ├── list.go                  # `pvm list` command
│   ├── use.go                   # `pvm use` command
│   ├── remove.go                # `pvm remove` command
│   ├── lts_resolver.go          # Bridges cobra context → php.LatestLTS
│   ├── env.go                   # baseDir() for Linux/macOS → ~/.pvm
│   └── env_windows.go           # baseDir() for Windows → %LOCALAPPDATA%\pvm
└── internal/
    ├── php/
    │   ├── types.go             # Branch, Release, Status types
    │   ├── releases.go          # Fetches branches + LatestPatch from php.net API
    │   ├── detect.go            # DetectSystem() — finds PHP binaries outside pvm
    │   ├── util.go              # Version string parsing helpers
    │   └── http_request.go      # Generic HTTP client with JSON decoding
    ├── fs/
    │   ├── manager.go           # Manager — wraps pvm home directory structure
    │   ├── binary.go            # Read/write binary path; VersionInstalled check
    │   └── versions.go          # InstalledVersions() — sorted list from versions dir
    ├── installer/
    │   ├── select.go            # Dispatches Install/Remove by runtime.GOOS
    │   ├── apt.go               # Linux: apt-get + ondrej/php PPA auto-add
    │   ├── brew.go              # macOS: Homebrew
    │   ├── windows.go           # Windows: download from windows.php.net
    │   └── windows_download.go  # HTTP download + zip extraction helpers
    ├── symlink/
    │   ├── get.go               # GetCurrent() — reads current-version file
    │   └── set.go               # SetCurrent() / RemoveCurrent() — per-OS switching
    └── version/
        ├── version.go           # Version struct, Parse(), Compare()
        └── lts.go               # Resolver interface and Resolve() for aliases
```

## Runtime directory structure

### Linux / macOS

```
~/.pvm/                        (or $PVM_HOME)
├── current-version            # plain text: "8.3" — written by pvm use
├── bin/
│   └── php                    # symlink → active PHP binary (Linux)
├── shims/
│   └── php                    # symlink → active PHP binary (macOS)
└── versions/
    ├── 8.3/
    │   └── binary             # plain text: /usr/bin/php8.3
    └── 8.4/
        └── binary             # plain text: /usr/bin/php8.4
```

### Windows

```
%LOCALAPPDATA%\pvm\            (or %PVM_HOME%)
├── current-version            # plain text: "8.3"
├── shims\
│   └── php.bat                # batch shim → active php.exe
├── php\
│   ├── 8.3\                   # extracted from windows.php.net zip
│   │   ├── php.exe
│   │   └── ...
│   └── 8.4\
│       ├── php.exe
│       └── ...
└── versions\
    ├── 8.3\
    │   └── binary             # plain text: C:\Users\...\AppData\Local\pvm\php\8.3\php.exe
    └── 8.4\
        └── binary
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
       └─ if alias "lts" → php.LatestLTS(ctx)
  └─ version.Parse(concrete)
  └─ fs.Manager.EnsurebaseDir()
  └─ fs.Manager.VersionInstalled()
  └─ installer.Install(base, ver)            ← dispatches by runtime.GOOS
       ├─ Linux:   AptInstall()
       │    └─ apt-get install phpX.Y-cli
       │    └─ auto-adds ondrej/php PPA on failure, retries
       ├─ macOS:   BrewInstall()
       │    └─ brew install php@X.Y
       └─ Windows: WindowsInstall()
            └─ php.LatestPatch(ctx, branch)  ← resolves "8.3" → "8.3.30" if needed
            └─ downloadAndExtractPHP()
                 └─ tries windows.php.net/releases/ then /archives/
                 └─ extracts zip to %LOCALAPPDATA%\pvm\php\<branch>\
            └─ write versions/<ver>/binary = <installDir>\php.exe
```

## Data flow — `pvm use`

```
cmd.runUse
  └─ version.Resolve(arg, phpLTSResolver)
  └─ fs.Manager.VersionInstalled()
  └─ fs.Manager.GetVersionBinary()
  └─ symlink.SetCurrent(base, ver, binPath)  ← dispatches by runtime.GOOS
       ├─ Linux:   update-alternatives --set php <binary>
       │           + ~/.pvm/bin/php symlink
       ├─ macOS:   ~/.pvm/shims/php symlink
       └─ Windows: resolves php.exe from %LOCALAPPDATA%\pvm\php\<branch>\
                   writes %LOCALAPPDATA%\pvm\shims\php.bat
  └─ writeCurrentVersion(base, ver)
  └─ printPathHint() if shim dir not in PATH
```

## Data flow — `pvm remove`

```
cmd.runRemove
  └─ version.Parse(arg)
  └─ fs.Manager.VersionInstalled()
  └─ symlink.GetCurrent()
  └─ installer.Remove(base, ver)             ← dispatches by runtime.GOOS
       ├─ Linux:   apt-get remove phpX.Y-cli
       ├─ macOS:   brew uninstall php@X.Y
       └─ Windows: os.RemoveAll(%LOCALAPPDATA%\pvm\php\<branch>\)
  └─ fs.Manager.RemoveVersionDir()
  └─ if was active → symlink.RemoveCurrent()
```

## Key design decisions

- **Platform-specific `baseDir()`** — `cmd/env.go` (`~/.pvm`) and `cmd/env_windows.go` (`%LOCALAPPDATA%\pvm`) use build tags so each OS follows its own convention.
- **Windows: direct download instead of a package manager** — winget treats all PHP versions as the same product (shared Windows Installer GUID), making side-by-side installs impossible. Downloading zips from `windows.php.net` lets pvm own every version in its own isolated directory.
- **`InstallerFunc` type** — `install.go` accepts `func(base, ver string) error`, making it easy to swap backends without changing the command layer.
- **`version.Resolver` interface** — decouples alias resolution from the php.net API, enabling unit-testing without network calls.
- **`binary` file** — stores only the resolved binary path, keeping version detection O(1) (one file read + stat).
- **Concurrent branch fetching** — `FetchAllBranches` fans out one goroutine per active major version, reducing latency when php.net is slow.
- **`current-version` file** — plain-text file tracking the active version; used by `pvm list`. On Linux it is complementary to `update-alternatives`; on Windows/macOS it is the sole source of truth for display.
