package composer

import (
	"bytes"
	"regexp"
	"slices"
	"strings"
)

// missingExtRe matches Composer's platform errors, e.g.
// "symfony/framework-bundle[v8.1.0, ..., v8.1.7] require ext-xml * -> it is missing from your system."
// "Root composer.json requires PHP extension ext-intl * but it is missing from your system."
var (
	missingExtRe = regexp.MustCompile(`\bext-([A-Za-z0-9_]+)\b.*missing from your system`)
	projectRe    = regexp.MustCompile(`^Created project in (.+?)\s*$`)
	ansiRe       = regexp.MustCompile("\x1b\\[[0-9;?]*[A-Za-z]")
)

// Output watches Composer's stderr line by line for what pvm acts on.
type Output struct {
	Missing []string // extensions reported missing, in order, without repeats
	Project string   // directory create-project created, if any
	partial []byte   // unfinished last line
}

// maxLine bounds a line pvm holds on to; the ones it looks for are short.
const maxLine = 64 << 10

func (o *Output) Write(p []byte) (int, error) {
	o.partial = append(o.partial, p...)
	for {
		i := bytes.IndexByte(o.partial, '\n')
		if i < 0 {
			break
		}
		o.line(string(o.partial[:i]))
		o.partial = o.partial[i+1:]
	}
	if len(o.partial) > maxLine {
		o.partial = o.partial[len(o.partial)-maxLine:]
	}
	return len(p), nil
}

// Flush handles a last line without a newline, once the output has ended.
func (o *Output) Flush() {
	if len(o.partial) > 0 {
		o.line(string(o.partial))
		o.partial = nil
	}
}

func (o *Output) line(l string) {
	l = strings.TrimRight(ansiRe.ReplaceAllString(l, ""), "\r")
	if m := projectRe.FindStringSubmatch(l); m != nil {
		o.Project = m[1]
	}
	for _, m := range missingExtRe.FindAllStringSubmatch(l, -1) {
		ext := strings.ToLower(m[1])
		if !slices.Contains(o.Missing, ext) {
			o.Missing = append(o.Missing, ext)
		}
	}
}
