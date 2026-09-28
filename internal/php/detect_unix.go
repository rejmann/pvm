//go:build linux || darwin

package php

import "runtime"

func platformGlobs() []string {
	switch runtime.GOOS {
	case "darwin":
		return darwinGlobs()
	default:
		return linuxGlobs()
	}
}

func linuxGlobs() []string {
	return []string{
		"/usr/bin/php[0-9]*",
		"/usr/local/bin/php[0-9]*",
	}
}

func darwinGlobs() []string {
	return []string{
		"/opt/homebrew/opt/php*/bin/php", // Apple Silicon Homebrew
		"/usr/local/opt/php*/bin/php",    // Intel Homebrew
		"/opt/homebrew/bin/php[0-9]*",
		"/usr/local/bin/php[0-9]*",
		"/opt/local/bin/php[0-9]*", // MacPorts
	}
}
