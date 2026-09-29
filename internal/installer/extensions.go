package installer

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// BaseExtensions are installed with every PHP version: what most Composer
// projects need and the minimal packages (e.g. apt's -cli) leave out.
var BaseExtensions = []string{"zip", "xml", "mbstring", "curl"}

// extensionAliases maps extensions that ship inside another one's package
// (Composer names them ext-dom, ext-pdo_mysql...) to that extension.
var extensionAliases = map[string]string{
	"dom":        "xml",
	"simplexml":  "xml",
	"xmlreader":  "xml",
	"xmlwriter":  "xml",
	"xsl":        "xml",
	"mysqli":     "mysql",
	"mysqlnd":    "mysql",
	"pdo_mysql":  "mysql",
	"pdo_pgsql":  "pgsql",
	"pdo_sqlite": "sqlite3",
}

// normalizeExtension lowercases ext, drops Composer's "ext-" prefix and
// resolves extensionAliases.
func normalizeExtension(ext string) string {
	ext = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ext)), "ext-")
	if alias, ok := extensionAliases[ext]; ok {
		return alias
	}
	return ext
}

// packagesFile lists the extension packages pvm installed for a version, one
// per line, so removing the version removes them too.
func packagesFile(base, ver string) string {
	return filepath.Join(base, "versions", ver, "packages")
}

func readPackages(base, ver string) []string {
	f, err := os.Open(packagesFile(base, ver))
	if err != nil {
		return nil
	}
	defer f.Close()

	var pkgs []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if p := strings.TrimSpace(sc.Text()); p != "" {
			pkgs = append(pkgs, p)
		}
	}
	return pkgs
}

// recordPackages adds pkgs to the version's packages file, skipping those
// already listed.
func recordPackages(base, ver string, pkgs []string) error {
	seen := map[string]bool{}
	all := readPackages(base, ver)
	for _, p := range all {
		seen[p] = true
	}
	for _, p := range pkgs {
		if !seen[p] {
			seen[p] = true
			all = append(all, p)
		}
	}

	path := packagesFile(base, ver)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(all, "\n")+"\n"), 0644)
}
