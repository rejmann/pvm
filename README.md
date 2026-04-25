# pvm — PHP Version Manager

`pvm` is a cross-platform CLI tool for installing and managing multiple PHP versions on Linux, macOS, and Windows.

## Quick start

```sh
# List available PHP versions from php.net
pvm available

# Install the current LTS branch
pvm install lts

# Install a specific branch
pvm install 8.3

# List installed versions
pvm list

# Switch to an installed version
pvm use 8.3
pvm use lts

# Remove an installed version
pvm remove 8.3
```

## Requirements

| OS | Requirement |
|----|-------------|
| Linux (Debian/Ubuntu) | `apt`, `sudo`. Older versions (e.g. 7.4) need the [ondrej/php PPA](https://launchpad.net/~ondrej/+archive/ubuntu/php) — pvm adds it automatically if needed. |
| macOS | [Homebrew](https://brew.sh) |
| Windows | No external dependency — PHP is downloaded directly from [windows.php.net](https://windows.php.net) |

## PATH setup

After the first `pvm use`, add the pvm shim directory to your PATH once:

| OS | Directory | Shell config |
|----|-----------|--------------|
| Linux | `~/.pvm/bin` | `export PATH="$HOME/.pvm/bin:$PATH"` |
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
