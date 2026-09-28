package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rejmann/pvm/internal/catalog"
	"github.com/rejmann/pvm/internal/proc"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// ltsResolver resolves the "lts" alias against php.net.
type ltsResolver struct {
	ctx context.Context
}

func (r ltsResolver) ResolveLTS() (string, error) {
	return catalog.LatestLTS(r.ctx)
}

// streams sends the output of programs pvm runs to the command's output.
func streams(cmd *cobra.Command) proc.Streams {
	return proc.Streams{Out: cmd.OutOrStdout(), Err: cmd.ErrOrStderr()}
}

// versionLabel names a version in messages, marking one that came from the
// "lts" alias.
func versionLabel(v string, wasAlias bool) string {
	if wasAlias {
		return v + " (lts)"
	}
	return v
}

func confirm(in io.Reader, out io.Writer, prompt string) bool {
	fmt.Fprint(out, prompt)
	line, _ := bufio.NewReader(in).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	}
	return false
}

// isTerminal is false for pipes and redirects, including </dev/null (a char
// device, so a mode check alone is not enough).
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
