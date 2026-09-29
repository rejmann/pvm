package composer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rejmann/pvm/internal/phpext"
)

// Requirements says which extensions a Composer command will check, so pvm
// can offer them before Composer runs. The zero value checks nothing.
type Requirements struct {
	Lock bool // also the ones composer.lock's packages require (install)
	Dev  bool // include require-dev and packages-dev
}

// RequirementsFor tells what Composer checks for args: install and update
// check the project's platform requirements, unless the command works on
// another directory or ignores them. Other commands (require, which adds
// packages pvm cannot know yet, or no command at all) check nothing.
func RequirementsFor(args []string) (Requirements, bool) {
	command := ""
	noDev := false
	for _, a := range args {
		switch {
		case a == "--":
			return Requirements{}, false
		case a == "-d" || strings.HasPrefix(a, "--working-dir") || strings.HasPrefix(a, "--ignore-platform-req"):
			return Requirements{}, false
		case a == "--no-dev" || a == "--update-no-dev":
			noDev = true
		case command == "" && !strings.HasPrefix(a, "-"):
			command = a
		}
	}
	switch command {
	case "install", "i", "reinstall":
		return Requirements{Lock: true, Dev: !noDev}, true
	case "update", "u", "upgrade":
		return Requirements{Dev: !noDev}, true
	}
	return Requirements{}, false
}

// manifest is the part of composer.json pvm reads.
type manifest struct {
	Require    map[string]string `json:"require"`
	RequireDev map[string]string `json:"require-dev"`
	Config     struct {
		Platform map[string]any `json:"platform"`
	} `json:"config"`
}

type lockFile struct {
	Packages    []lockPackage `json:"packages"`
	PackagesDev []lockPackage `json:"packages-dev"`
}

type lockPackage struct {
	Require map[string]string `json:"require"`
}

// RequiredExtensions lists the extensions (named as by phpext.Name) that the
// project in dir requires: composer.json's, and with r.Lock its locked
// packages'. Extensions config.platform pretends are present are left out.
// $COMPOSER, if set, names the composer.json. A missing or unreadable file
// requires nothing: Composer reports those itself.
func RequiredExtensions(dir string, r Requirements, getenv func(string) string) []string {
	name := "composer.json"
	if v := getenv("COMPOSER"); v != "" {
		name = v
	}
	jsonPath := name
	if !filepath.IsAbs(jsonPath) {
		jsonPath = filepath.Join(dir, name)
	}

	var m manifest
	if !readJSON(jsonPath, &m) {
		return nil
	}
	reqs := []map[string]string{m.Require}
	if r.Dev {
		reqs = append(reqs, m.RequireDev)
	}
	if r.Lock {
		var l lockFile
		if readJSON(strings.TrimSuffix(jsonPath, ".json")+".lock", &l) {
			pkgs := l.Packages
			if r.Dev {
				pkgs = append(pkgs, l.PackagesDev...)
			}
			for _, p := range pkgs {
				reqs = append(reqs, p.Require)
			}
		}
	}

	faked := map[string]bool{}
	for k, v := range m.Config.Platform {
		if v != false {
			faked[phpext.Name(k)] = true
		}
	}

	var exts []string
	for _, req := range reqs {
		for k := range req {
			if !strings.HasPrefix(strings.ToLower(k), "ext-") {
				continue
			}
			if ext := phpext.Name(k); !faked[ext] {
				exts = append(exts, ext)
			}
		}
	}
	slices.Sort(exts)
	return slices.Compact(exts)
}

func readJSON(path string, v any) bool {
	data, err := os.ReadFile(path)
	return err == nil && json.Unmarshal(data, v) == nil
}
