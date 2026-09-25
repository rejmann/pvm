package main

import (
	"fmt"
	"os"

	"github.com/rejmann/pvm/cmd"
	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags "-X main.version=<tag>".
var version = "dev"

var rootCmd = &cobra.Command{
	Use:           "pvm",
	Short:         "PVM is a tool for managing multiple versions of PHP.",
	Long:          `PVM is a tool for managing multiple versions of PHP, allowing you to easily switch between different versions for different projects.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func main() {
	cmds := []*cobra.Command{
		cmd.AvailableCmd,
		cmd.InstallCmd,
		cmd.ListCmd,
		cmd.UseCmd,
		cmd.RemoveCmd,
		cmd.CurrentCmd,
		cmd.LocalCmd,
		cmd.WhichCmd,
		cmd.ShimCmd,
	}
	rootCmd.AddCommand(cmds...)
	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
