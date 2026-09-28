# Architecture

For the end-to-end flow across commands — lifecycle, state files and platform differences — see [flow.md](flow.md). This document covers packages and code-level data flow.

## Package layout

```
pvm/
├── main.go                      # Entry point — cmd.Execute(version)
├── cmd/                         # Cobra commands only: flags, arguments, messages
│   ├── root.go                  # Execute() / newRootCmd() — registers every command
│   ├── helpers.go               # ltsResolver, streams, versionLabel, confirm, isTerminal
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
│   └── self_remove.go           # `pvm self-remove` — uninstalls pvm
└── internal/
    ├── home/                    # the only package that knows the pvm home layout
    │   ├── home.go              # Dir, Default(), ShimDir, PHPDir, ComposerDir, CacheDir
    │   ├── versions.go          # RegisterVersion, VersionBinary, InstalledVersions, MatchInstalled
    │   ├── current.go           # Current / SetCurrent / ClearCurrent (current-version)
    │   └── default_*.go         # default base dir: ~/.pvm or %LOCALAPPDATA%\pvm
    ├── resolve/
    │   └── resolve.go           # Active(): PVM_VERSION → .php-version → global; Version(): explicit/lts
    ├── activate/                # connects the global version to the system
    │   ├── linux.go             # Use/Clear: update-alternatives + shim + ~/.local/bin/php
    │   ├── darwin.go            # Use/Clear: shim only
    │   ├── windows.go           # Use: php.bat shim, user PATH, PowerShell profile wrapper
    │   ├── unix.go              # EnsureShim(): writes the php shim script
    │   └── pswrapper.go         # Builds/replaces/removes the PowerShell profile wrapper block
    ├── installer/               # one backend per OS, same functions in each
    │   ├── linux.go             # detects package manager + per-PM install logic
    │   ├── darwin.go            # Homebrew
    │   ├── windows.go           # download from windows.php.net
    │   ├── windows_download.go  # download (httpx) + zip extraction helpers
    │   └── phpini.go            # writePHPIni(): php.ini with openssl + zip for Windows builds
    ├── catalog/
    │   ├── releases.go          # php.net branches, LatestPatch, LatestLTS
    │   ├── cache.go             # 24-hour cache of available branches
    │   └── types.go             # Branch, Status
    ├── php/
    │   ├── detect*.go           # DetectSystem(): PHP binaries outside pvm
    │   └── probe.go             # Probe(): exact version + zip extension of a binary
    ├── composer/
    │   ├── composer.go          # per-PHP Composer: release choice (/versions), download, signature check, Env()
    │   └── keys/                # Composer's public keys (embedded): release signatures, Composer home
    ├── proc/
    │   ├── proc.go              # Streams (output of programs pvm runs), LookPathExcluding
    │   └── exec_*.go            # Exec(): syscall.Exec on Unix, child process + exit code on Windows
    ├── httpx/
    │   └── httpx.go             # shared HTTP client: context, timeout, size limit, User-Agent
    ├── project/
    │   └── project.go           # Find/Read .php-version
    ├── selfupdate/
    │   ├── selfupdate.go        # Latest release tag, archive download/extraction, binary swap
    │   └── remove_*.go          # RemoveBinary(): deletes the running pvm binary (per OS)
    └── version/
        ├── version.go           # Version, Parse(), Compare(), Branch()
        └── lts.go               # Resolver interface and Resolve() for aliases
```

Dependencies point one way: `cmd` → `resolve` / `activate` / `installer` / `composer` / `catalog` → `home` / `proc` / `httpx` / `version`. No internal package imports `cmd`, and none but `home` builds paths inside the pvm home.

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
│   │   ├── php.ini            # written by pvm: extension_dir + openssl, zip
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
  └─ catalog.FetchAllBranchesCached(ctx, h.CacheDir())   ← cache/available.json, 24 h
  └─ catalog.FetchAllBranches(ctx)
       ├─ GET php.net/releases/?json          → SupportedResponse (active majors)
       └─ goroutine per major version
            └─ GET php.net/releases/?version=N → MajorResponse (all patches)
                 └─ latestPatchPerBranch()     → map[branch]latestVersion
  └─ print table to stdout
```

## Data flow — `pvm install <version|lts>`

```
cmd.runInstall
  └─ version.Resolve(arg, ltsResolver)
       └─ if alias "lts" → catalog.LatestLTS(ctx)
  └─ version.Parse(concrete)
  └─ home.Dir.EnsureBaseDir()
  └─ home.Dir.VersionInstalled()
  └─ installer.Install(ctx, h, ver, streams) ← backend chosen by build tag, returns the php binary
       ├─ Linux:   linux.go
       │    └─ detectPackageManager()        ← scans PATH for apt-get/dnf/yum/pacman/zypper
       │    └─ preInstall() if needed:
       │         ├─ apt-get → adds ondrej/php PPA
       │         └─ dnf/yum → adds Remi repo
       │    └─ sudo <pm> install <php-pkg>   ← package name varies by distro
       │    └─ installExtras()               ← apt/dnf/yum: zip extension, warning on failure
       ├─ macOS:   darwin.go
       │    └─ brew install php@X.Y
       └─ Windows: windows.go
            └─ catalog.LatestPatch(ctx, branch) ← resolves "8.3" → "8.3.30" if needed
            └─ downloadAndExtractPHP()      ← httpx: context, timeout, size limit
                 └─ tries windows.php.net/releases/ then /archives/
                 └─ extracts zip to h.PHPDir(branch)
            └─ writePHPIni()                 ← php.ini-production + extension_dir, openssl, zip
  └─ home.Dir.RegisterVersion(ver, bin)      ← versions/<ver>/binary
```

## Data flow — `pvm use`

```
cmd.runUse
  └─ version.Resolve(arg, ltsResolver)
  └─ home.Dir.VersionInstalled()
  └─ home.Dir.VersionBinary()
  └─ activate.Use(h, ver, bin, streams)      ← backend chosen by build tag
       ├─ Linux:   update-alternatives --set php <binary>
       │           + EnsureShim() → ~/.pvm/bin/php
       ├─ macOS:   EnsureShim() → ~/.pvm/shims/php
       └─ Windows: resolves php.exe from h.PHPDir(branch)
                   writes %LOCALAPPDATA%\pvm\shims\php.bat
       └─ home.Dir.SetCurrent(ver)
  └─ printPathHint() if shim dir not in PATH
```

## Data flow — `pvm remove`

```
cmd.runRemove
  └─ version.Parse(arg)
  └─ home.Dir.VersionInstalled()
  └─ home.Dir.Current()
  └─ installer.Remove(h, ver, streams)       ← backend chosen by build tag
       ├─ Linux:   detectPackageManager() → sudo <pm> remove <php-pkg>
       ├─ macOS:   brew uninstall php@X.Y
       └─ Windows: os.RemoveAll(h.PHPDir(branch))
  └─ home.Dir.RemoveVersionDir()
  └─ composer.Remove(h.ComposerDir(), ver)
  └─ if was active → activate.Clear(h, streams)
```

## Data flow — `pvm list`

```
cmd.runList
  └─ home.Dir.Current()                      ← reads current-version file
  └─ home.Dir.InstalledVersions()            ← sorted list from versions dir
  └─ php.DetectSystem()                      ← finds PHP binaries outside pvm
  └─ print grouped table to stdout
```

## Data flow — `php` (via shim)

```
~/.pvm/bin/php "$@"                          ← #!/bin/sh script written by EnsureShim
  └─ exec pvm shim php "$@"
       └─ cmd.shimTarget
            ├─ resolve.Active(h, cwd, $PVM_VERSION)
            │    ├─ $PVM_VERSION
            │    ├─ project.Find(cwd)        ← nearest .php-version walking up
            │    ├─ home.Dir.Current()       ← global current-version
            │    └─ home.Dir.MatchInstalled() → versions/<ver>/binary
            └─ ErrNoActiveVersion → proc.LookPathExcluding("php", PATH, h.ShimDir())
       └─ proc.Exec()                        ← syscall.Exec, so pvm is replaced by php
```

## Data flow — `pvm composer`

```
cmd.runComposer
  └─ resolve.Active(h, cwd, $PVM_VERSION)             ← pvm-managed only, no system php
  └─ php.Probe() → php -r '…'                         ← exact version + extension_loaded("zip")
  └─ no zip and no unzip/7z on PATH?
       └─ offerZipExtension() → terminal: confirm → installer.EnsureExtensions()
  └─ composer.Downloader.Ensure(h.ComposerDir(), installed, exact)
       └─ composer/php/<installed>/composer.phar missing?
            └─ GET getcomposer.org/versions → SelectRelease(): newest with min-php ≤ exact
            └─ GET /download/<ver>/composer.phar + .sig → RSA-SHA384 with embedded key
            └─ writeKeys(home) + temp file + rename into composer/php/<installed>/
  └─ composer.Env() → COMPOSER_HOME=composer/php/<installed>/home, COMPOSER_CACHE_DIR=composer/cache (unless set)
  └─ proc.Exec(php, [composer.phar, args...])        ← PVM_VERSION set for child processes
```

## Data flow — `pvm current` / `pvm which`

```
cmd.runCurrent / cmd.runWhich
  └─ resolve.Active(h, cwd, $PVM_VERSION)
       ├─ found → print version (+ source) or binary path
       └─ ErrNoActiveVersion → "No PHP version is currently active."
```

## Data flow — `pvm self-upgrade [tag]`

```
cmd.runSelfUpgrade
  ├─ os.Executable + EvalSymlinks → path of the running binary
  ├─ selfupdate.LatestTag (GitHub API /releases/latest) — unless a tag was given
  ├─ same as the build version (main.version, passed to cmd.Execute) → "already at <tag>"
  ├─ selfupdate.Download → pvm-<os>-<arch>.tar.gz|.zip → pvm binary bytes
  └─ selfupdate.Replace → temp file in the same dir → rename over the binary
```

## Data flow — `pvm self-remove`

```
cmd.runSelfRemove
  ├─ checkRemovableBase → refuse home/root as data dir
  ├─ list binary, data dir, installed versions → confirm (unless --yes)
  ├─ --php → installer.Remove for each installed version
  ├─ activate.RemoveIntegration → Windows: user PATH + PowerShell profile (no-op on Unix)
  ├─ os.RemoveAll(data dir)
  └─ selfupdate.RemoveBinary → Unix: os.Remove · Windows: rename + detached cmd.exe deletes it after exit
```

## Key design decisions

- **One owner for the pvm home** — `internal/home` builds every path under the pvm home, so the on-disk layout is defined in one place. Its default base dir comes from build-tagged files (`~/.pvm` on Unix, `%LOCALAPPDATA%\pvm` on Windows).
- **Build tags instead of `runtime.GOOS` switches** — `installer` and `activate` have one file per OS defining the same functions, so each binary only contains its own platform's code and there is no dispatch layer.
- **Windows: direct download instead of a package manager** — winget treats all PHP versions as the same product (shared Windows Installer GUID), making side-by-side installs impossible. Downloading zips from `windows.php.net` lets pvm own every version in its own isolated directory.
- **Installers return the binary, `home` registers it** — `installer.Install` returns the php binary path and `pvm install` records it with `home.Dir.RegisterVersion`, so no backend writes pvm's metadata. `cmd/install.go` takes an `installFunc`, which tests replace.
- **`version.Resolver` interface** — decouples alias resolution from the php.net API, enabling unit-testing without network calls.
- **`binary` file** — stores only the resolved binary path, keeping version detection O(1) (one file read + stat).
- **Shared HTTP client** — `internal/httpx` gives every download a context, a timeout and a size limit, and turns non-200 responses into a `StatusError`.
- **Concurrent branch fetching** — `FetchAllBranches` fans out one goroutine per active major version, reducing latency when php.net is slow.
- **`current-version` file** — plain-text file tracking the global version; used by `pvm list` and as the shim's fallback. On Linux it is complementary to `update-alternatives`, which keeps `/usr/bin/php` pointing at the global version for services that do not use the shim.
- **Composer downloaded on demand, not installed globally** — `pvm composer` fetches `composer.phar` into the pvm home on first use instead of at pvm install time, so users who never run Composer never download it, and no `composer` binary competes with one already on `PATH`. Composer's config and cache go under the pvm home too, so everything pvm brings in is removed by `pvm remove` / `pvm self-remove`, and `pvm composer` only runs PHP versions pvm manages.
- **One Composer per PHP version** — the phar, `self-update` backups and global packages all depend on the PHP running Composer, so sharing them lets one version break another (a `self-update` under 8.5 to a Composer that 7.4 cannot run, a `--rollback` restoring another version's backup, global tools resolved for the wrong PHP). Each version gets its own phar and `COMPOSER_HOME`; only the PHP-independent download cache is shared. The release is chosen from getcomposer.org/versions by `min-php`, as `self-update` does, instead of a hard-coded PHP → Composer table, and verified with Composer's signing key.
- **Dynamic shim instead of a symlink** — the Unix shim is a tiny `sh` script that calls `pvm shim php`, so the version is picked per call from `PVM_VERSION`/`.php-version`/global. pvm then `exec`s the real binary, adding ~2 ms and keeping signals, stdin and the exit code intact. The shim embeds pvm's absolute path and is regenerated by `pvm use`.
