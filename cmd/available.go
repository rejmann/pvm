package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/rejmann/pvm/internal/catalog"
	"github.com/rejmann/pvm/internal/home"
	"github.com/spf13/cobra"
)

func newAvailableCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "available [a]",
		Aliases: []string{"a"},
		Short:   "List all PHP versions available to install from php.net",
		Long:    "List all PHP versions available to install from php.net.\nResults are cached for 1 day. Use --refresh to force a fresh fetch.",
		Args:    cobra.NoArgs,
		RunE:    runAvailable,
	}
	c.Flags().BoolP("refresh", "r", false, "Refresh the list of available PHP versions by fetching from php.net")
	return c
}

func runAvailable(cmd *cobra.Command, args []string) error {
	refresh, _ := cmd.Flags().GetBool("refresh")
	return printAvailable(cmd.Context(), home.Default(), refresh, cmd.OutOrStdout())
}

func printAvailable(ctx context.Context, h *home.Dir, forceRefresh bool, out io.Writer) error {
	branches, err := catalog.FetchAllBranchesCached(ctx, h.CacheDir(), forceRefresh)
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
