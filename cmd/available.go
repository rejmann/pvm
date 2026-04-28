package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/rejmann/pvm/internal/php"
	"github.com/spf13/cobra"

	phpfs "github.com/rejmann/pvm/internal/fs"
)

var AvailableCmd = &cobra.Command{
	Use:     "available [a]",
	Aliases: []string{"a"},
	Short: "List all PHP versions available to install from php.net",
	Long:  "List all PHP versions available to install from php.net.\nResults are cached for 1 day. Use --refresh to force a fresh fetch.",
	Args:  cobra.NoArgs,
	RunE:  runAvailable,
}

func init() {
	AvailableCmd.Flags().BoolP("refresh", "r", false, "Refresh the list of available PHP versions by fetching from php.net")
}

func runAvailable(cmd *cobra.Command, args []string) error {
	refresh, _ := cmd.Flags().GetBool("refresh")
	return printAvailable(cmd.Context(), phpfs.NewManager(baseDir()), refresh, cmd.OutOrStdout())
}

func printAvailable(ctx context.Context, m *phpfs.Manager, forceRefresh bool, out io.Writer) error {
	branches, err := php.FetchAllBranchesCached(ctx, m.Base, forceRefresh)
	if err != nil {
		return err
	}

	fmt.Fprintln(out, "Install with: pvm install <branch>  (e.g. pvm install 8.4)")
	fmt.Fprintln(out, "              pvm install lts       (installs newest supported branch)")
	fmt.Fprintf(out, "\n%d branches listed.\n", len(branches))

	fmt.Fprintf(out, "\n  %-8s  %-12s  %-13s\n", "BRANCH", "LATEST", "STATUS")
	fmt.Fprintln(out, "  --------  ------------  -------------")

	for _, b := range branches {
		fmt.Fprintf(out, "  %-8s  %-12s  %-13s\n", b.Name, b.Latest, b.Status)
	}

	return nil
}
