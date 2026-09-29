// Package shim switches the PHP version users get when they type php: it
// writes the php shim into the pvm home and records the global version. Each
// OS has its own Activator, selected by build tags:
//
//   - Linux and macOS: the shim, in the pvm home's bin directory — no root
//     needed, /usr/bin/php is left to the system;
//   - Windows: a php.bat shim, the user PATH and a PowerShell profile wrapper.
package shim
