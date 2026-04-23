package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "pvm",
	Short:         "PVM is a tool for managing multiple versions of PHP",
	Long:          `PVM is a tool for managing multiple versions of PHP, allowing you to easily switch between different versions for different projects.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	cmds := []*cobra.Command{
		availableCmd,
	}
	rootCmd.AddCommand(cmds...)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
