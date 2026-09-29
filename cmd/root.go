// Package cmd is pvm's command line: each command parses its arguments,
// calls the use cases in internal/pvm and prints the result.
package cmd

import "github.com/spf13/cobra"

// NewRootCmd returns the pvm command with all its subcommands; version is
// the running pvm version, set at build time.
func NewRootCmd(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "pvm",
		Short:         "PVM is a tool for managing multiple versions of PHP.",
		Long:          `PVM is a tool for managing multiple versions of PHP, allowing you to easily switch between different versions for different projects.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newAvailableCmd(),
		newInstallCmd(),
		newListCmd(),
		newUseCmd(),
		newRemoveCmd(),
		newCurrentCmd(),
		newWhichCmd(),
		newRunCmd(),
		newComposerCmd(),
		newExtCmd(),
		newShimCmd(),
		newSelfUpgradeCmd(version),
		newSelfRemoveCmd(),
	)
	return root
}
