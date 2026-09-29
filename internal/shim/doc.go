// Package shim switches the PHP version users get when they type php: it
// writes the php shim into the pvm home and records the global version. Each
// OS has its own Activator, selected by build tags:
//
//   - Linux: the shim, plus update-alternatives so /usr/bin/php follows the
//     global version for programs that do not use the shim;
//   - macOS: the shim;
//   - Windows: a php.bat shim, the user PATH and a PowerShell profile wrapper.
package shim
