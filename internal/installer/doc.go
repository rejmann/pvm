// Package installer installs and removes PHP versions, one backend per OS
// selected by build tags: the system package manager on Linux, Homebrew on
// macOS, and builds from windows.php.net on Windows. Every backend has the
// same functions:
//
//	Install(ctx, h, ver, s) (bin string, err error)
//	Remove(h, ver, s) error
//	EnsureExtensions(h, ver, s) error
//
// Install returns the php binary; registering it in the pvm home is up to the
// caller.
package installer
