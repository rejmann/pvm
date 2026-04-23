# pvm — PHP Version Manager

`pvm` is a CLI tool for installing and managing multiple PHP versions on Linux systems that use `apt` (Debian/Ubuntu).

## Quick start

```sh
# List available PHP versions from php.net
pvm available

# Install the current LTS branch
pvm install lts

# Install a specific branch
pvm install 8.3

# Activate pvm's php in your shell
export PATH="$HOME/.pvm/bin:$PATH"
```

## Requirements

- Linux with `apt` and `sudo`
- [ondrej/php PPA](https://launchpad.net/~ondrej/+archive/ubuntu/php) for non-distro PHP versions:

```sh
sudo add-apt-repository ppa:ondrej/php && sudo apt-get update
```

## Build

```sh
go build -o pvm .
```

## Documentation

| Doc | Description |
|-----|-------------|
| [docs/architecture.md](docs/architecture.md) | Package layout and data flow |
| [docs/commands.md](docs/commands.md) | CLI command reference |
| [docs/development.md](docs/development.md) | Adding commands and installers |
