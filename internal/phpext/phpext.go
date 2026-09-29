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

// Normalize lowercases ext, drops Composer's "ext-" prefix and resolves
// extensions that ship inside another one's package (dom → xml).
func Normalize(ext string) string {
	ext = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ext)), "ext-")
	if alias, ok := aliases[ext]; ok {
		return alias
	}
	return ext
}
