# CLI Command Reference

## `pvm available`

Lists all PHP branches available to install from php.net.

```
pvm available [flags]

Flags:
  -r, --refresh   Force a fresh fetch (cache is not yet implemented — flag reserved)
```

### Output

```
Install with: pvm install <branch>  (e.g. pvm install 8.4)
              pvm install lts       (installs newest supported branch)

12 branches listed.

  BRANCH    LATEST        STATUS
  --------  ------------  -------------
  8.4       8.4.6         supported
  8.3       8.3.20        supported
  8.2       8.2.27        supported
  8.1       8.1.31        eol
  ...
```

**STATUS** values:
- `supported` — branch is within its active support window on php.net
- `eol` — branch has reached end-of-life but patches are still listed

### How it works

Hits `https://www.php.net/releases/index.php?json` to get the list of active majors, then launches one concurrent request per major to retrieve all patches. It retains only the highest patch per branch.

---

## `pvm list`

Lists all PHP versions found on the machine, split into two groups.

```
pvm list
```

### Output

```
pvm managed:
  8.3.20
  8.4.6  (current)
system:
  8.1  (/usr/bin/php8.1)
```

**Groups:**
- `pvm managed` — versions installed via `pvm install`, stored under `~/.pvm/versions/`. The version set as active via `pvm use` is marked `(current)`.
- `system` — PHP binaries found on the host **outside** of pvm (e.g. distro package, Homebrew). Versions already tracked by pvm are excluded from this group to avoid duplicates.

If no PHP versions are found at all:

```
No PHP versions found.
Run 'pvm available' to see installable versions.
```

### How it works

1. Reads `~/.pvm/versions/` to build the managed list (sorted ascending by version).
2. Reads `~/.pvm/current-version` to mark the active version.
3. Runs `DetectSystem()` which globs well-known binary locations (`/usr/bin/php[0-9]*`, `/usr/local/bin/php[0-9]*`, Homebrew paths) and also checks `php` in `$PATH`. For each candidate it executes `php -r "echo PHP_MAJOR_VERSION.'.'.PHP_MINOR_VERSION;"` to confirm the version.

---

## `pvm install <version|lts>`

Installs a PHP version using the system `apt` package manager.

```
pvm install <version|lts>

Arguments:
  version   Branch (e.g. 8.3) or full version (e.g. 8.3.20)
  lts       Alias — resolves to the highest currently-supported branch
```

### Examples

```sh
pvm install lts       # installs e.g. 8.4
pvm install 8.3       # installs php8.3-cli via apt
pvm install 8.3.20    # same package — apt resolves the branch
```

### What it does

1. Resolves `lts` alias to the highest supported branch name (e.g. `8.4`).
2. Validates the version string format.
3. Creates `~/.pvm/versions/` if it does not exist.
4. Skips installation if the version is already installed.
5. Runs `sudo apt-get install -y php<X.Y>-cli`.
6. Writes the resolved binary path to `~/.pvm/versions/<ver>/binary`.
7. Prints a PATH hint if `~/.pvm/bin` is not first in `$PATH`.

### Prerequisites

The [ondrej/php PPA](https://launchpad.net/~ondrej/+archive/ubuntu/php) must be added before installing non-distro PHP versions:

```sh
sudo add-apt-repository ppa:ondrej/php
sudo apt-get update
```

### PATH setup

After installing, add `~/.pvm/bin` to your shell profile to make `php` point to the pvm-managed version:

```sh
# ~/.zshrc or ~/.bashrc
export PATH="$HOME/.pvm/bin:$PATH"
```

Then restart your shell or run `source ~/.zshrc`.

> `PVM_HOME` environment variable overrides the default `~/.pvm` directory.
