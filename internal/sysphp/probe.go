package sysphp

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/rejmann/pvm/internal/phpext"
)

// Info is what Probe learns from a php binary.
type Info struct {
	Version    string   // exact version, e.g. "8.3.12"
	Zip        bool     // zip extension loaded
	Extensions []string // loaded extensions and Zend extensions, named as by phpext.Name, sorted
}

// Has reports whether extension ext (any spelling phpext.Name accepts) is loaded.
func (i Info) Has(ext string) bool {
	_, found := slices.BinarySearch(i.Extensions, phpext.Name(ext))
	return found
}

// probeScript prints the version and the loaded extensions on the last two
// lines, so startup warnings printed before them are ignored.
const probeScript = `echo "\n", PHP_MAJOR_VERSION, ".", PHP_MINOR_VERSION, ".", PHP_RELEASE_VERSION, "\n", implode(",", array_merge(get_loaded_extensions(), get_loaded_extensions(true)));`

// Probe asks the php binary for its exact version, since pvm may only know
// the branch (8.3) and Composer's minimum PHP is a patch (7.2.5), and for its
// loaded extensions (e.g. whether it can extract zip archives). php.ini is
// loaded, as Composer will load it.
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
	var exts []string
	for _, e := range strings.Split(lines[len(lines)-1], ",") {
		if e = phpext.Name(e); e != "" {
			exts = append(exts, e)
		}
	}
	slices.Sort(exts)
	exts = slices.Compact(exts)

	info := Info{Version: strings.TrimSpace(lines[len(lines)-2]), Extensions: exts}
	info.Zip = info.Has("zip")
	return info, nil
}
