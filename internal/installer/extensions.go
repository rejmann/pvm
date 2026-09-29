package installer

import "strings"

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
