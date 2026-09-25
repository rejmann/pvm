package main

import (
	"fmt"
	"os"

	"github.com/rejmann/pvm/cmd"
	"github.com/spf13/cobra"
)

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
	}
	rootCmd.AddCommand(cmds...)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
