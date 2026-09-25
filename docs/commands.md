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

## `pvm use <version|lts>` · alias `u`

Switches the active PHP version.

```
pvm use <version|lts>

Arguments:
  version   Full version or branch as recorded by pvm install
  lts       Alias — resolves to the highest currently-supported branch
```

### Examples

```sh
pvm use 8.3       # activates 8.3.x (whichever patch was installed)
pvm use lts       # activates e.g. 8.4
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
| Linux | `sudo update-alternatives --set php <binary>` + `~/.pvm/bin/php` symlink |
| macOS | Symlink at `~/.pvm/shims/php` |
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

## `pvm current` · alias `cur`

Shows the currently active PHP version as tracked by pvm.

```
pvm current
```

### Output

```
Current PHP version: 8.3
```

If no version is active:

```
No PHP version is currently active.
```

### What it does

Reads `<pvm-home>/current-version` and prints its contents. The file is written by `pvm use` and cleared by `pvm remove` when the removed version was active.
