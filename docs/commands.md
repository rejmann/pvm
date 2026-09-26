# CLI Command Reference

## `pvm available` · alias `a`

Lists all PHP branches available to install from php.net.

```
pvm available [--refresh]

Flags:
  -r, --refresh   Bypass cache and fetch fresh data from php.net
```

Results are cached for **1 day** at `~/.pvm/cache/available.json` (Windows: `%LOCALAPPDATA%\pvm\cache\available.json`). Use `--refresh` to force a fresh fetch.

### Output

```
Install with: pvm install <branch>  (e.g. pvm install 8.4)
              pvm install lts       (installs newest supported branch)

18 branches listed.

  BRANCH    LATEST        STATUS
  --------  ------------  -------------
  8.4       8.4.20        supported
  8.3       8.3.30        supported
  8.2       8.2.30        supported
  7.4       7.4.33        eol
  ...
```

**STATUS** values:
- `supported` — branch is within its active support window on php.net
- `eol` — branch has reached end-of-life but patches are still listed

### How it works

On the first run (or with `--refresh`), hits `https://www.php.net/releases/index.php?json` to get the list of active majors, then launches one concurrent request per major to retrieve all patches. Retains only the highest patch per branch. The result is written to the local cache file and reused for 24 hours on subsequent calls.

---

## `pvm install <version|lts>` · alias `i`

Installs a PHP version using the appropriate backend for the current OS.

```
pvm install <version|lts>

Arguments:
  version   Branch (e.g. 8.3) or full version (e.g. 8.3.30)
  lts       Alias — resolves to the highest currently-supported branch
```

### Examples

```sh
pvm install lts       # installs the latest LTS branch
pvm install 8.3       # installs the latest 8.3.x patch
pvm install 8.3.30    # installs a specific patch version
```

### What it does

1. Resolves `lts` alias to the highest supported branch name.
2. Validates the version string format.
3. Skips installation if the version is already installed.
4. Runs the OS-appropriate installer (see below).
5. Writes the resolved binary path to `<pvm-home>/versions/<ver>/binary`.
6. Prints a PATH hint if the pvm shim directory is not yet in `$PATH`.

### OS backends

pvm detects the available package manager automatically on Linux.

| OS / Distro | Backend | Notes |
|-------------|---------|-------|
| Linux (Debian/Ubuntu) | `apt-get install php<X.Y>-cli` | Adds [ondrej/php PPA](https://launchpad.net/~ondrej/+archive/ubuntu/php) automatically if the package is not found |
| Linux (Fedora) | `dnf install php<X.Y>-php-cli` | Adds [Remi repo](https://rpms.remirepo.net) automatically if the package is not found |
| Linux (RHEL/CentOS) | `yum install php<X.Y>-php-cli` | Adds [Remi repo](https://rpms.remirepo.net) automatically if the package is not found |
| Linux (Arch) | `pacman -S php` | Only the version in the official repos; no extra repo added |
| Linux (openSUSE) | `zypper install php<X.Y>` | — |
| macOS | `brew install php@<X.Y>` | Requires [Homebrew](https://brew.sh) |
| Windows | Downloads zip from `windows.php.net` and extracts to `%LOCALAPPDATA%\pvm\php\<branch>\` | No external dependency |

### Windows install directory

Each branch is isolated under the pvm home:

```
%LOCALAPPDATA%\pvm\php\8.3\php.exe
%LOCALAPPDATA%\pvm\php\8.5\php.exe
```

> `PVM_HOME` environment variable overrides the default pvm home directory on all platforms.

---

## `pvm use [version|lts]` · alias `u`

Switches the global PHP version.

```
pvm use [version|lts]

Arguments:
  version   Full version or branch as recorded by pvm install
  lts       Alias — resolves to the highest currently-supported branch
  (none)    Uses the version from the nearest .php-version file
```

### Examples

```sh
pvm use 8.3       # activates 8.3.x (whichever patch was installed)
pvm use lts       # activates e.g. 8.4
pvm use           # activates the version in ./.php-version (or a parent directory)
```

### What it does

1. Resolves `lts` alias if needed.
2. Fails with a helpful error if the version is not installed.
3. Reads the binary path from `<pvm-home>/versions/<ver>/binary`.
4. Updates the active version via the OS-appropriate mechanism (see below).
5. Writes the active version to `<pvm-home>/current-version`.
6. Prints a PATH hint if the shim directory is not in `$PATH`.

### OS switching mechanism

| OS | Mechanism |
|----|-----------|
| Linux | `sudo update-alternatives --set php <binary>` + `~/.pvm/bin/php` shim script |
| macOS | Shim script at `~/.pvm/shims/php` |
| Windows | Batch shim at `%LOCALAPPDATA%\pvm\shims\php.bat` pointing to the installed `php.exe` |

### Typical workflow

```sh
pvm install 8.3
pvm install 8.4
pvm use 8.4       # → Now using PHP 8.4.20.
pvm use 8.3       # → Now using PHP 8.3.30.
```

---

## `pvm list` · alias `ls`

Lists all PHP versions found on the machine.

```
pvm list
```

### Output

```
pvm managed:
  8.3  (current)
  8.4
system:
  8.1  (/usr/bin/php8.1)
```

**Groups:**
- `pvm managed` — versions installed via `pvm install`. The active version is marked `(current)`.
- `system` — PHP binaries found outside pvm. Versions already tracked by pvm are excluded.

---

## `pvm remove <version>` · alias `rm`

Removes a pvm-managed PHP version.

```
pvm remove <version>

Arguments:
  version   Exact version string as installed (e.g. 8.3)
```

### What it does

1. Validates the version string format.
2. Fails if the version is not installed.
3. Removes the version files/directory.
4. If the removed version was active, clears `current-version` and prints a warning:
   ```
   Warning: PHP 8.3 was the active version. No version is now active.
   ```

> On Windows, `pvm remove` deletes `%LOCALAPPDATA%\pvm\php\<branch>\` entirely.

---

## Per-project versions (`.php-version`)

If a project has a `.php-version` file, `php` switches automatically to that version inside the project directory and its subdirectories. pvm only reads this file; it never creates it.

```sh
cd ~/code/legacy-app  # contains .php-version with 7.4
php -v            # → PHP 7.4.33 — also in any subdirectory
cd ~/code/new-app
php -v            # → global version again
```

### How the version is chosen

Every `php` call goes through the pvm shim, which picks the first match:

1. `PVM_VERSION` environment variable (e.g. `PVM_VERSION=8.2 php -v`)
2. The nearest `.php-version`, searching from the current directory up to `/`
3. The global version set by `pvm use`
4. The first `php` on `PATH` outside pvm (system PHP)

A version from steps 1–3 that is not installed is an error; pvm never silently falls back to another version.

`.php-version` holds a single version (`8.3` or `8.3.30`). Blank lines and lines starting with `#` are ignored. A branch such as `8.3` matches the highest installed `8.3.x`. The format is the same one used by other PHP version managers, so the file can be committed.

> Per-directory switching works on Linux and macOS. On Windows, `pvm current` / `pvm which` honour `.php-version`, but the `php.bat` shim still uses the global version.

---

## `pvm which`

Prints the path of the PHP binary that `php` would run in the current directory, following the rules in [How the version is chosen](#how-the-version-is-chosen).

```sh
pvm which         # → /usr/bin/php8.3
```

---

## `pvm run [-v version | version] <file> [args...]`

Runs a PHP file with a specific installed version, without changing the global or project version.

```
pvm run [version|lts] <file> [args...]
pvm run [version|lts] -f|--file <file> [args...]
pvm run -v|--version <version|lts> <file> [args...]

Arguments:
  version    Installed version or branch (e.g. 8.2 matches the highest installed 8.2.x),
             given first or with -v/--version (also --version=8.2); it must be installed
  lts        Alias — resolves to the highest currently-supported branch
  (none)     Uses the version in use: PVM_VERSION → nearest .php-version → global
  file       PHP file to run — required; it must exist
  args       Passed to the script unchanged
```

A file is mandatory: `pvm run`, `pvm run 8.5`, `pvm run -v` or `pvm run 8.5 -r '...'` fail with `no PHP file given`, and a missing file or a directory fails before php starts. The first argument is taken as the version only if it looks like one (`8.5`, `8.2.30`, `lts`). `-v`/`--version` works anywhere (`pvm run script.php -v 8.2`); giving the version twice is an error. Arguments after `--` go to the script untouched, so use it when the script has its own `-v`: `pvm run script.php -- -v 8.2`.

### Examples

```sh
pvm run 8.5 script.php --input data.txt
pvm run 8.2 --file script.php
pvm run --version 8.2 script.php
pvm run -v lts script.php
pvm run script.php -v 8.2
pvm run script.php -- -v   # -v goes to the script
pvm run 8.2 vendor/bin/phpunit
pvm run script.php       # version in use in this directory
```

### What it does

1. Checks that a PHP file was given and exists.
2. Resolves the version like `pvm use` (alias, branch → installed patch), or — without one — like the `php` shim; fails if it is not installed.
3. Sets `PVM_VERSION=<version>` for the new process, so anything it starts that calls `php` through the pvm shim (Composer, `#!/usr/bin/env php` scripts, `shell_exec("php ...")`) uses the same version.
4. Replaces itself with the php binary (Windows: runs it as a child), so stdin, stdout, signals and the exit code are php's own.

---

## `pvm current` · alias `cur`

Shows the PHP version active in the current directory and, when it is not the global one, where it was set.

```
pvm current
```

### Output

```
Current PHP version: 8.3
Current PHP version: 7.4 (set by /home/me/code/legacy-app/.php-version)
```

If no version is active:

```
No PHP version is currently active.
```

### What it does

Resolves the version exactly like the shim does (`PVM_VERSION` → `.php-version` → `<pvm-home>/current-version`). The global file is written by `pvm use` and cleared by `pvm remove` when the removed version was active.

---

## `pvm self-upgrade [tag]`

Upgrades pvm itself by downloading a release from [GitHub Releases](https://github.com/rejmann/pvm/releases) and replacing the running binary.

```
pvm self-upgrade [tag] [--check]

Arguments:
  tag         Release to install (e.g. v1.2.0 or 1.2.0) — also allows downgrading
  (none)      Installs the latest release

Flags:
  -c, --check   Only report whether a newer release is available
```

### Examples

```sh
pvm self-upgrade           # latest release
pvm self-upgrade --check   # pvm v1.2.0 is available (current: v1.1.0). Run: pvm self-upgrade
pvm self-upgrade v1.1.0    # a specific release
sudo pvm self-upgrade      # when pvm lives in a root-owned directory such as /usr/local/bin
```

### What it does

1. Resolves the target tag — the given one, or the latest release from the GitHub API.
2. Stops if the running version already is that tag.
3. Downloads `pvm-<os>-<arch>.tar.gz` (Windows: `.zip`) for that tag and extracts the `pvm` binary.
4. Writes it next to the running binary (symlinks resolved) and renames it over the old one, so a failed upgrade never leaves a broken binary. On Windows the running `pvm.exe` is first moved to `pvm.exe.old`, which the next upgrade removes.

If the directory is not writable, it fails with `permission denied` and suggests re-running with `sudo`. On Windows the hint is to run it from a terminal opened as Administrator. Every published platform can self-upgrade: Linux (amd64, arm64), macOS (amd64, arm64) and Windows amd64; releases published before a platform was added fail with `has no build for <os>/<arch>`.
