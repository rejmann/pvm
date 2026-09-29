// Package phpext names PHP extensions the way pvm installs them.
package phpext

import "strings"

// Base are installed with every PHP version: what most Composer
// projects need and the minimal packages (e.g. apt's -cli) leave out.
var Base = []string{"zip", "xml", "mbstring", "curl"}

// aliases maps extensions that ship inside another one's package
// (Composer names them ext-dom, ext-pdo_mysql...) to that extension.
var aliases = map[string]string{
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

// Name is the extension as PHP loads it: lowercased and without Composer's
// "ext-" prefix (ext-PDO_MySQL → pdo_mysql). "Zend OPcache", as PHP lists it,
// is opcache.
func Name(ext string) string {
	ext = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ext)), "ext-")
	if ext == "zend opcache" || ext == "zend-opcache" {
		return "opcache"
	}
	return ext
}

// Normalize is Name with extensions that ship inside another one's package
// resolved to that package's extension (dom → xml).
func Normalize(ext string) string {
	ext = Name(ext)
	if alias, ok := aliases[ext]; ok {
		return alias
	}
	return ext
}

// Removal is what removing extensions did with each one.
type Removal struct {
	Uninstalled []string // pvm had installed them: uninstalled
	Disabled    []string // ship with PHP (or pvm did not install them): turned off instead
	Stuck       []string // no package or .ini to act on: compiled into PHP, or not there at all
}
