package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rejmann/pvm/internal/pvm"
	"github.com/rejmann/pvm/internal/system"
	"github.com/spf13/cobra"
)

// ShimCmd is invoked by the php shim script; it is not meant to be run by hand.
var ShimCmd = &cobra.Command{
	Use:                "shim php [args...]",
	Short:              "Run php with the version selected for the current directory",
	Hidden:             true,
	DisableFlagParsing: true,
	RunE:               runShim,
}

func runShim(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || args[0] != "php" {
		return errors.New("usage: pvm shim php [args...]")
	}

	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	bin, err := shimTarget(newManager(cmd.Context()), dir, os.Getenv(pvm.EnvVersion), os.Getenv("PATH"))
	if err != nil {
		return err
	}
	return execBinary(bin, args[1:])
}

// shimTarget returns the php binary the shim should run. When no version is
// selected anywhere, it falls back to the first php on PATH outside pvm.
func shimTarget(m *pvm.Manager, dir, env, path string) (string, error) {
	a, err := m.Active(dir, env)
	if err == nil {
		return a.Binary, nil
	}
	if !errors.Is(err, pvm.ErrNoActiveVersion) {
		return "", err
	}

	if bin := lookPathExcluding("php", path, m.Home.ShimDir()); bin != "" {
		return bin, nil
	}
	return "", fmt.Errorf("%w and no system php found — run: pvm use <version>", err)
}

// lookPathExcluding is exec.LookPath restricted to PATH entries other than skip.
func lookPathExcluding(name, path, skip string) string {
	if runtime.GOOS == system.Windows {
		name += ".exe"
	}
	skip = filepath.Clean(skip)

	for _, dir := range filepath.SplitList(path) {
		if dir == "" || strings.EqualFold(filepath.Clean(dir), skip) {
			continue
		}
		p := filepath.Join(dir, name)
		fi, err := os.Stat(p)
		if err != nil || fi.IsDir() {
			continue
		}
		if runtime.GOOS != system.Windows && fi.Mode()&0111 == 0 {
			continue
		}
		return p
	}
	return ""
}
