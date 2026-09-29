package cmd

import (
	"fmt"
	"io"

	"github.com/rejmann/pvm/internal/pvm"
	"github.com/spf13/cobra"
)

var InstallCmd = &cobra.Command{
	Use:     "install [i] <version|lts>",
	Aliases: []string{"i"},
	Short:   "Install a PHP version",
	Args:    cobra.ExactArgs(1),
	RunE:    runInstall,
}

func runInstall(cmd *cobra.Command, args []string) error {
	return installVersion(newManager(cmd), args[0], cmd.OutOrStdout())
}

func installVersion(m *pvm.Manager, arg string, out io.Writer) error {
	t, err := m.Target(arg)
	if err != nil {
		return err
	}
	err = m.Install(t, func() { fmt.Fprintf(out, "Installing PHP %s...\n", t) })
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "PHP %s installed successfully.\n", t)
	return nil
}
