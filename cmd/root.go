// Package cmd is the pvm command line: flags, arguments and messages. The
// work itself is done by the internal packages.
package cmd

import "github.com/spf13/cobra"

// Execute runs pvm with the process arguments; version is the build version.
func Execute(version string) error {
	return newRootCmd(version).Execute()
}

func newRootCmd(version string) *cobra.Command {
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
		newShimCmd(),
		newSelfUpgradeCmd(version),
		newSelfRemoveCmd(),
	)

	return root
}
