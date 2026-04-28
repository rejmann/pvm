package cmd

import (
	"fmt"
	"io"

	"github.com/rejmann/pvm/internal/php"
	"github.com/rejmann/pvm/internal/symlink"
	"github.com/spf13/cobra"

	phpfs "github.com/rejmann/pvm/internal/fs"
)

var ListCmd = &cobra.Command{
	Use:     "list [ls]",
	Aliases: []string{"ls"},
	Short: "List installed PHP versions",
	Args:  cobra.NoArgs,
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	return listVersions(phpfs.NewManager(baseDir()), cmd.OutOrStdout())
}

func listVersions(m *phpfs.Manager, out io.Writer) error {
	managed, err := m.InstalledVersions()
	if err != nil {
		return fmt.Errorf("read installed versions: %w", err)
	}

	current, _ := symlink.GetCurrent(m.Base)
	system := php.DetectSystem()

	managedSet := map[string]bool{}
	for _, v := range managed {
		managedSet[v] = true
	}

	printed := false

	if len(managed) > 0 {
		fmt.Fprintln(out, "pvm managed:")
		for _, v := range managed {
			if v == current {
				fmt.Fprintf(out, "  %s (current)\n", v)
			} else {
				fmt.Fprintf(out, "  %s\n", v)
			}
		}
		printed = true
	}

	var unmanaged []php.SystemInstall
	for _, s := range system {
		if !managedSet[s.Version] {
			unmanaged = append(unmanaged, s)
		}
	}

	if len(unmanaged) > 0 {
		fmt.Fprintln(out, "system:")
		for _, s := range unmanaged {
			fmt.Fprintf(out, "  %s  (%s)\n", s.Version, s.Binary)
		}
		printed = true
	}

	if !printed {
		fmt.Fprintln(out, "No PHP versions found.")
		fmt.Fprintln(out, "Run 'pvm available' to see installable versions.")
	}

	return nil
}
