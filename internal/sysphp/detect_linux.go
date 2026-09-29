package sysphp

const phpExe = "php"

func platformGlobs() []string {
	return []string{
		"/usr/bin/php[0-9]*",
		"/usr/local/bin/php[0-9]*",
	}
}
