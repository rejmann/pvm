# pvm — PHP Version Manager

`pvm` is a cross-platform CLI tool for installing and managing multiple PHP versions on Linux, macOS, and Windows.

## Installation

The commands below always fetch the **latest** release from [GitHub Releases](https://github.com/rejmann/pvm/releases/latest) — no version number to update. Once installed, upgrade with `pvm self-upgrade` (prefix with `sudo` if pvm is in a root-owned directory such as `/usr/local/bin`).

### Linux (x86_64)

```sh
curl -fsSL https://github.com/rejmann/pvm/releases/latest/download/pvm-linux-amd64.tar.gz \
  | sudo tar -xz --no-same-owner -C /usr/local/bin pvm
```

### macOS (Apple Silicon)

```sh
sudo mkdir -p /usr/local/bin
curl -fsSL https://github.com/rejmann/pvm/releases/latest/download/pvm-darwin-arm64.tar.gz \
  | sudo tar -xz --no-same-owner -C /usr/local/bin pvm
```

### Windows (PowerShell)

Installs `pvm.exe` to `%LOCALAPPDATA%\Programs\pvm` and adds it to your user `PATH` (no admin rights needed):

```powershell
$dir = "$env:LOCALAPPDATA\Programs\pvm"
$zip = "$env:TEMP\pvm.zip"
New-Item -ItemType Directory -Force -Path $dir | Out-Null
Invoke-WebRequest -Uri "https://github.com/rejmann/pvm/releases/latest/download/pvm-windows-amd64.zip" -OutFile $zip
Expand-Archive -Path $zip -DestinationPath $dir -Force
Remove-Item $zip
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$dir*") {
  [Environment]::SetEnvironmentVariable("Path", "$dir;$userPath", "User")
}
$env:Path = "$dir;$env:Path"
```

The last line makes `pvm` available in the current session; other terminals that were already open need to be restarted to pick up the updated `PATH`.

### Installing a specific version

Replace `latest/download` with `download/<tag>` in any of the commands above. For example, on Linux:

```sh
VERSION=v1.0.1
curl -fsSL "https://github.com/rejmann/pvm/releases/download/${VERSION}/pvm-linux-amd64.tar.gz" \
  | sudo tar -xz --no-same-owner -C /usr/local/bin pvm
```

On Windows, set the URL in the PowerShell snippet to:

```powershell
$version = "v1.0.1"
Invoke-WebRequest -Uri "https://github.com/rejmann/pvm/releases/download/$version/pvm-windows-amd64.zip" -OutFile $zip
```

All available tags are listed on the [releases page](https://github.com/rejmann/pvm/releases).

### Verify

```sh
pvm --help
```

> Prebuilt binaries are published for Linux (x86_64, arm64), macOS (Apple Silicon, Intel) and Windows x86_64 — on Linux arm64 or an Intel Mac, swap `amd64`/`arm64` in the archive name (`pvm-linux-arm64.tar.gz`, `pvm-darwin-amd64.tar.gz`). On other platforms, see [Build](#build).

## Quick start

```sh
# List available PHP versions from php.net
pvm available          # alias: a

# Install the current LTS branch
pvm install lts        # alias: i
pvm install 8.3

# List installed versions
pvm list               # alias: ls

# Show the currently active version
pvm current            # alias: cur

# Switch to an installed version
pvm use 8.3            # alias: u
pvm use lts

# Remove an installed version
pvm remove 8.3         # alias: rm

# Upgrade pvm itself to the latest release
pvm self-upgrade
```

## Per-project versions

If a project has a `.php-version` file, `php` switches automatically to that version inside the project directory and its subdirectories:

```sh
cd ~/code/legacy-app   # contains .php-version with 7.4
php -v                 # PHP 7.4.x
pvm current            # Current PHP version: 7.4 (set by ~/code/legacy-app/.php-version)
pvm which              # /usr/bin/php7.4

PVM_VERSION=8.3 php -v # one-off override
pvm run 8.2 script.php # run a script with a specific version
```

`pvm use` without arguments activates the version from `.php-version` globally. Automatic per-directory switching currently works on Linux and macOS. It requires the shim directory in your `PATH` (see below).

## Requirements

| OS | Requirement |
|----|-------------|
| Linux (Debian/Ubuntu) | `apt-get`, `sudo`. Extra PHP branches (e.g. 7.4) need the [ondrej/php PPA](https://launchpad.net/~ondrej/+archive/ubuntu/php) — pvm adds it automatically. |
| Linux (Fedora) | `dnf`, `sudo`. Extra branches need the [Remi repo](https://rpms.remirepo.net) — pvm adds it automatically. |
| Linux (RHEL/CentOS) | `yum`, `sudo`. Extra branches need the [Remi repo](https://rpms.remirepo.net) — pvm adds it automatically. |
| Linux (Arch) | `pacman`, `sudo`. Only the version available in the official repos can be installed. |
| Linux (openSUSE) | `zypper`, `sudo`. |
| macOS | [Homebrew](https://brew.sh) |
| Windows | No external dependency — PHP is downloaded directly from [windows.php.net](https://windows.php.net) |

pvm detects the package manager automatically on Linux — no configuration needed.

## PATH setup

After the first `pvm use`, add the pvm shim directory to your PATH once:

| OS | Directory | Shell config |
|----|-----------|--------------|
| Linux | `~/.pvm/bin` | `export PATH="$HOME/.pvm/bin:$PATH"` in `~/.bashrc` / `~/.zshrc` |
| macOS | `~/.pvm/shims` | `export PATH="$HOME/.pvm/shims:$PATH"` |
| Windows | `%LOCALAPPDATA%\pvm\shims` | `setx PATH "%LOCALAPPDATA%\pvm\shims;%PATH%"` |

> `pvm use` prints the exact command if the directory is not yet in your PATH.

## Build

Requires only `make` and Docker Compose — Go runs inside containers, never on your machine:

```sh
make setup        # build the Docker images
make build        # dist/pvm for your OS/arch
make build-all    # all platforms, e.g. dist/pvm-windows-amd64.exe
make test
```

> **Developing pvm?** Use `make pvm <command>` — it runs pvm inside a Docker container, never on your machine, so your real pvm and system PHP stay untouched. See [docs/development.md](docs/development.md).

## Documentation

| Doc | Description |
|-----|-------------|
| [docs/architecture.md](docs/architecture.md) | Package layout and data flow |
| [docs/commands.md](docs/commands.md) | CLI command reference |
| [docs/development.md](docs/development.md) | Adding commands and installers |
