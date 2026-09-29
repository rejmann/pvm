// Package installer installs and removes PHP versions the way each OS
// supports it: the distribution's package manager on Linux, Homebrew on
// macOS, and the builds from windows.php.net on Windows. Each OS has its own
// System type, selected by build tags.
package installer
