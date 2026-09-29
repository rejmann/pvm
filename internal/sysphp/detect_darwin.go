package sysphp

const phpExe = "php"

func platformGlobs() []string {
	return []string{
		"/opt/homebrew/opt/php*/bin/php", // Apple Silicon Homebrew
		"/usr/local/opt/php*/bin/php",    // Intel Homebrew
		"/opt/homebrew/bin/php[0-9]*",
		"/usr/local/bin/php[0-9]*",
		"/opt/local/bin/php[0-9]*", // MacPorts
	}
}
