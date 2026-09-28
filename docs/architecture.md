# Architecture

For the end-to-end flow across commands — lifecycle, state files and platform differences — see [flow.md](flow.md). This document covers packages and code-level data flow.

## Package layout

```
pvm/
├── main.go                      # Entry point — registers Cobra commands
├── cmd/
│   ├── available.go             # `pvm available` command
│   ├── current.go               # `pvm current` command
│   ├── install.go               # `pvm install` command
│   ├── list.go                  # `pvm list` command
│   ├── use.go                   # `pvm use` command
│   ├── remove.go                # `pvm remove` command
│   ├── which.go                 # `pvm which` command
│   ├── run.go                   # `pvm run` — run a PHP file with a specific version
│   ├── composer.go              # `pvm composer` — run composer.phar with the version in use
│   ├── shim.go                  # hidden `pvm shim php` — entry point of the php shim
│   ├── self_upgrade.go          # `pvm self-upgrade` — replaces pvm with a GitHub release
│   ├── self_remove.go           # `pvm self-remove` — uninstalls pvm
│   ├── active.go                # resolveActive(): PVM_VERSION → .php-version → global
│   ├── exec_unix.go             # execBinary() via syscall.Exec
│   ├── exec_windows.go          # execBinary() via child process + exit code
│   ├── lts_resolver.go          # Bridges cobra context → php.LatestLTS
│   ├── env.go                   # baseDir() for Linux/macOS → ~/.pvm
│   └── env_windows.go           # baseDir() for Windows → %LOCALAPPDATA%\pvm
└── internal/
    ├── php/
    │   ├── types.go             # Branch, Release, Status, SystemInstall types
    │   ├── releases.go          # Fetches branches + LatestPatch from php.net API
    │   ├── cache.go             # 24-hour cache for available versions
    │   ├── detect.go            # DetectSystem() dispatcher
    │   ├── detect_unix.go       # Unix: find PHP in common paths
    │   ├── detect_windows.go    # Windows: scan registry for PHP
    │   ├── util.go              # Version string parsing helpers
    │   └── http_request.go      # Generic HTTP client with JSON decoding
    ├── composer/
    │   ├── composer.go          # per-PHP Composer: release choice (/versions), download, signature check, Env()
    │   └── keys/                # Composer's public keys (embedded): release signatures, Composer home
    ├── fs/
    │   ├── manager.go           # Manager — wraps pvm home directory structure
    │   ├── binary.go            # Read/write binary path; VersionInstalled check
    │   └── versions.go          # InstalledVersions() — sorted list from versions dir
    ├── installer/
    │   ├── select_unix.go       # Unix: dispatches Install/Remove by runtime.GOOS
    │   ├── select_windows.go    # Windows: dispatches Install/Remove
    │   ├── linux.go             # Linux: detects package manager + per-PM install logic
    │   ├── brew.go              # macOS: Homebrew
    │   ├── windows.go           # Windows: download from windows.php.net
    │   ├── windows_download.go  # HTTP download + zip extraction helpers
    │   ├── phpini.go            # writePHPIni(): php.ini with openssl + Extensions for Windows builds
    │   └── util.go              # majorMinor() version helper
    ├── symlink/
    │   ├── get.go               # GetCurrent() — reads current-version file
    │   ├── set.go               # SetCurrent() / RemoveCurrent() dispatcher
    │   ├── set_unix.go          # Unix: symlinks + update-alternatives (Linux)
    │   ├── set_windows.go       # Windows: batch shim, user PATH, PowerShell profile wrapper
    │   ├── pswrapper.go         # Builds/replaces/removes the PowerShell profile wrapper block
    │   ├── shim.go              # ShimDir(): ~/.pvm/bin (Linux) or shims/ (others)
    │   └── shim_unix.go         # EnsureShim(): writes the php shim script
    ├── project/
    │   └── project.go           # Find/Read .php-version
    ├── selfupdate/
    │   ├── selfupdate.go        # Latest release tag, archive download/extraction, binary swap
    │   └── remove_*.go          # RemoveBinary(): deletes the running pvm binary (per OS)
    ├── system/
    │   └── system.go            # OS constants (Linux, Darwin, Windows)
    └── version/
        ├── version.go           # Version struct, Parse(), Compare()
        └── lts.go               # Resolver interface and Resolve() for aliases
```

## Runtime directory structure

### Linux / macOS

```
~/.pvm/                        (or $PVM_HOME)
├── current-version            # plain text: "8.3" — written by pvm use
├── composer/                  # created by `pvm composer`
│   ├── cache/                 # COMPOSER_CACHE_DIR, shared
│   └── php/
│       └── 8.3/               # one per PHP version; removed by pvm remove
│           ├── composer.phar  # newest Composer supporting this PHP
│           └── home/          # COMPOSER_HOME: config, auth.json, global packages, backups
├── bin/
│   └── php                    # shim script → `pvm shim php` (Linux)
├── shims/
│   └── php                    # shim script → `pvm shim php` (macOS)
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
├── composer\                  # same layout as Linux / macOS
├── shims\
│   └── php.bat                # batch shim → active php.exe
├── php\
│   ├── 8.3\                   # extracted from windows.php.net zip
│   │   ├── php.exe
│   │   ├── php.ini            # written by pvm: extension_dir + openssl, curl, mbstring, zip
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
       ├─ Linux:   LinuxInstall()
       │    └─ detectPackageManager()        ← scans PATH for apt-get/dnf/yum/pacman/zypper
       │    └─ preInstall() if needed:
       │         ├─ apt-get → adds ondrej/php PPA
       │         └─ dnf/yum → adds Remi repo
       │    └─ sudo <pm> install <php-pkg>   ← package name varies by distro
       │    └─ installExtras()               ← apt/dnf/yum: installer.Extensions packages, warning on failure
       ├─ macOS:   BrewInstall()
       │    └─ brew install php@X.Y
       └─ Windows: WindowsInstall()
            └─ php.LatestPatch(ctx, branch)  ← resolves "8.3" → "8.3.30" if needed
            └─ downloadAndExtractPHP()
                 └─ tries windows.php.net/releases/ then /archives/
                 └─ extracts zip to %LOCALAPPDATA%\pvm\php\<branch>\
            └─ writePHPIni()                 ← php.ini-production + extension_dir, openssl + Extensions
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
       │           + EnsureShim() → ~/.pvm/bin/php
       ├─ macOS:   EnsureShim() → ~/.pvm/shims/php
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
       ├─ Linux:   detectPackageManager() → sudo <pm> remove <php-pkg>
       ├─ macOS:   brew uninstall php@X.Y
       └─ Windows: os.RemoveAll(%LOCALAPPDATA%\pvm\php\<branch>\)
  └─ fs.Manager.RemoveVersionDir()
  └─ if was active → symlink.RemoveCurrent()
```

## Data flow — `pvm list`

```
cmd.runList
  └─ symlink.GetCurrent()                    ← reads current-version file
  └─ fs.Manager.InstalledVersions()          ← sorted list from versions dir
  └─ php.DetectSystem()                      ← finds PHP binaries outside pvm
  └─ print grouped table to stdout
```

## Data flow — `php` (via shim)

```
~/.pvm/bin/php "$@"                          ← #!/bin/sh script written by EnsureShim
  └─ exec pvm shim php "$@"
       └─ cmd.shimTarget
            ├─ cmd.resolveActive(base, cwd, $PVM_VERSION)
            │    ├─ $PVM_VERSION
            │    ├─ project.Find(cwd)        ← nearest .php-version walking up
            │    ├─ symlink.GetCurrent(base) ← global current-version
            │    └─ fs.Manager.MatchInstalled() → versions/<ver>/binary
            └─ ErrNoActiveVersion → first php on PATH outside ShimDir
       └─ execBinary()                       ← syscall.Exec, so pvm is replaced by php
```

## Data flow — `pvm composer`

```
cmd.runComposer
  └─ cmd.resolveActive(base, cwd, $PVM_VERSION)      ← pvm-managed only, no system php
  └─ probePHP() → php -r '…'                          ← exact version + installer.Extensions not loaded
  └─ neededExtensions(): missing, minus zip if unzip/7z on PATH
       └─ offerExtensions() → terminal: confirm → installer.EnsureExtensions()
  └─ composer.Downloader.Ensure(base, installed, exact)
       └─ composer/php/<installed>/composer.phar missing?
            └─ GET getcomposer.org/versions → SelectRelease(): newest with min-php ≤ exact
            └─ GET /download/<ver>/composer.phar + .sig → RSA-SHA384 with embedded key
            └─ writeKeys(home) + temp file + rename into composer/php/<installed>/
  └─ composer.Env() → COMPOSER_HOME=composer/php/<installed>/home, COMPOSER_CACHE_DIR=composer/cache (unless set)
  └─ execBinary(php, [composer.phar, args...])       ← PVM_VERSION set for child processes
```

## Data flow — `pvm current` / `pvm which`

```
cmd.runCurrent / cmd.runWhich
  └─ cmd.resolveActive(base, cwd, $PVM_VERSION)
       ├─ found → print version (+ source) or binary path
       └─ ErrNoActiveVersion → "No PHP version is currently active."
```

## Data flow — `pvm self-upgrade [tag]`

```
cmd.runSelfUpgrade
  ├─ os.Executable + EvalSymlinks → path of the running binary
  ├─ selfupdate.LatestTag (GitHub API /releases/latest) — unless a tag was given
  ├─ same as the build version (main.version) → "already at <tag>"
  ├─ selfupdate.Download → pvm-<os>-<arch>.tar.gz|.zip → pvm binary bytes
  └─ selfupdate.Replace → temp file in the same dir → rename over the binary
```

## Data flow — `pvm self-remove`

```
cmd.runSelfRemove
  ├─ checkRemovableBase → refuse home/root as data dir
  ├─ list binary, data dir, installed versions → confirm (unless --yes)
  ├─ --php → installer.Remove for each installed version
  ├─ symlink.RemoveIntegration → Windows: user PATH + PowerShell profile (no-op on Unix)
  ├─ os.RemoveAll(data dir)
  └─ selfupdate.RemoveBinary → Unix: os.Remove · Windows: rename + detached cmd.exe deletes it after exit
```

## Key design decisions

- **Platform-specific `baseDir()`** — `cmd/env.go` (`~/.pvm`) and `cmd/env_windows.go` (`%LOCALAPPDATA%\pvm`) use build tags so each OS follows its own convention.
- **Windows: direct download instead of a package manager** — winget treats all PHP versions as the same product (shared Windows Installer GUID), making side-by-side installs impossible. Downloading zips from `windows.php.net` lets pvm own every version in its own isolated directory.
- **`InstallerFunc` type** — `install.go` accepts `func(base, ver string) error`, making it easy to swap backends without changing the command layer.
- **`version.Resolver` interface** — decouples alias resolution from the php.net API, enabling unit-testing without network calls.
- **`binary` file** — stores only the resolved binary path, keeping version detection O(1) (one file read + stat).
- **Concurrent branch fetching** — `FetchAllBranches` fans out one goroutine per active major version, reducing latency when php.net is slow.
- **`current-version` file** — plain-text file tracking the global version; used by `pvm list` and as the shim's fallback. On Linux it is complementary to `update-alternatives`, which keeps `/usr/bin/php` pointing at the global version for services that do not use the shim.
- **Composer downloaded on demand, not installed globally** — `pvm composer` fetches `composer.phar` into the pvm home on first use instead of at pvm install time, so users who never run Composer never download it, and no `composer` binary competes with one already on `PATH`. Composer's config and cache go under the pvm home too, so everything pvm brings in is removed by `pvm remove` / `pvm self-remove`, and `pvm composer` only runs PHP versions pvm manages.
- **One Composer per PHP version** — the phar, `self-update` backups and global packages all depend on the PHP running Composer, so sharing them lets one version break another (a `self-update` under 8.5 to a Composer that 7.4 cannot run, a `--rollback` restoring another version's backup, global tools resolved for the wrong PHP). Each version gets its own phar and `COMPOSER_HOME`; only the PHP-independent download cache is shared. The release is chosen from getcomposer.org/versions by `min-php`, as `self-update` does, instead of a hard-coded PHP → Composer table, and verified with Composer's signing key.
- **Dynamic shim instead of a symlink** — the Unix shim is a tiny `sh` script that calls `pvm shim php`, so the version is picked per call from `PVM_VERSION`/`.php-version`/global. pvm then `exec`s the real binary, adding ~2 ms and keeping signals, stdin and the exit code intact. The shim embeds pvm's absolute path and is regenerated by `pvm use`.
