package installer

import "strings"

func majorMinor(ver string) string {
	parts := strings.SplitN(ver, ".", 3)
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return ver
}

// Extensions are the PHP extensions pvm makes sure every version has, so
// Composer and the usual frameworks (Symfony, Laravel) work out of the box:
// zip extracts packages without unzip/7z, curl speeds up downloads, and xml
// (dom, simplexml, xmlwriter…) and mbstring are required by most packages.
// Package managers whose PHP package leaves them out get them as extra
// packages; the Windows php.ini enables them.
var Extensions = []string{"curl", "mbstring", "xml", "zip"}
