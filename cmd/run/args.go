// Package run holds the argument handling of pvm run.
package run

import (
	"fmt"
	"strings"

	"github.com/rejmann/pvm/internal/version"
)

// SplitArgs separates the optional version from the php arguments. The
// version comes from -v/--version anywhere before a "--", or positionally from
// a first argument that parses as a version or alias ("lts"). Everything after
// "--" goes to the script untouched.
func SplitArgs(args []string) (versionArg string, rest []string, err error) {
	set := func(v string) error {
		if versionArg != "" {
			return fmt.Errorf("version given twice (%s and %s)", versionArg, v)
		}
		versionArg = v
		return nil
	}

	var passthrough []string
scan:
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			passthrough = args[i+1:]
			break scan
		case a == "-v" || a == "--version":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("flag %s needs a version", a)
			}
			if err := set(args[i+1]); err != nil {
				return "", nil, err
			}
			i++
		case strings.HasPrefix(a, "--version="):
			if err := set(strings.TrimPrefix(a, "--version=")); err != nil {
				return "", nil, err
			}
		default:
			rest = append(rest, a)
		}
	}

	if len(rest) > 0 && looksLikeVersion(rest[0]) {
		if err := set(rest[0]); err != nil {
			return "", nil, err
		}
		rest = rest[1:]
	}
	return versionArg, append(rest, passthrough...), nil
}

func looksLikeVersion(s string) bool {
	_, err := version.Parse(s)
	return err == nil || version.IsAlias(s)
}
