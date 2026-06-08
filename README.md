# pvm — PHP Version Manager

`pvm` is a cross-platform CLI tool for installing and managing multiple PHP versions on Linux, macOS, and Windows.

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
```

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

```sh
go build -o pvm .
```

Cross-compile for Windows from Linux/macOS:

```sh
GOOS=windows GOARCH=amd64 go build -o pvm.exe .
```

## Documentation

| Doc | Description |
|-----|-------------|
| [docs/architecture.md](docs/architecture.md) | Package layout and data flow |
| [docs/commands.md](docs/commands.md) | CLI command reference |
| [docs/development.md](docs/development.md) | Adding commands and installers |
