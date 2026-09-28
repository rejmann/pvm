package php

import (
	"fmt"
	"os/exec"
	"strings"
)

// Info is what Probe learns about a php binary.
type Info struct {
	Version string // exact version, e.g. "8.3.12"
	Zip     bool   // zip extension loaded
}

// probeScript prints the version and whether zip is loaded on the last two
// lines, so startup warnings printed before them are ignored.
const probeScript = `echo "\n", PHP_MAJOR_VERSION, ".", PHP_MINOR_VERSION, ".", PHP_RELEASE_VERSION, "\n", extension_loaded("zip") ? 1 : 0;`

// Probe asks the php binary for its exact version, since pvm may only know
// the branch (8.3) and Composer's minimum PHP is a patch (7.2.5), and whether
// it can extract zip archives. php.ini is loaded, as Composer will load it.
func Probe(bin string) (Info, error) {
	out, err := exec.Command(bin, "-r", probeScript).Output()
	if err != nil {
		return Info{}, fmt.Errorf("run %s: %w", bin, err)
	}
	return parseProbe(string(out))
}

func parseProbe(out string) (Info, error) {
	lines := strings.Split(strings.TrimRight(out, "\r\n"), "\n")
	if len(lines) < 2 {
		return Info{}, fmt.Errorf("unexpected php output: %q", out)
	}
	return Info{
		Version: strings.TrimSpace(lines[len(lines)-2]),
		Zip:     strings.TrimSpace(lines[len(lines)-1]) == "1",
	}, nil
}
