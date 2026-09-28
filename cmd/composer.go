package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rejmann/pvm/internal/composer"
	phpfs "github.com/rejmann/pvm/internal/fs"
	"github.com/spf13/cobra"
)

var ComposerCmd = &cobra.Command{
	Use:   "composer [args...]",
	Short: "Run Composer with the PHP version in use (downloads composer.phar on first use)",
	Long: `Run Composer with the PHP version in use in the current directory
($PVM_VERSION, the nearest .php-version, then the global version). Every
argument is passed to Composer unchanged, including -h, -V and --.

composer.phar is not installed globally: pvm downloads it from getcomposer.org
the first time it is needed and keeps it in its home directory. PHP 7.2.5 and
newer use the latest Composer; PHP 5.3.2 to 7.2.4 use the Composer 2.2 LTS.

PVM_VERSION is set for the process, so scripts Composer runs that call php
use the same version.`,
	Example: `  pvm composer install
  pvm composer require monolog/monolog
  pvm composer -V
  PVM_VERSION=8.2 pvm composer update`,
	DisableFlagParsing: true,
	RunE:               runComposer,
}

func runComposer(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	base := baseDir()

	installed, bin, err := phpTarget(phpfs.NewManager(base), dir, os.Getenv(envVersion), os.Getenv("PATH"))
	if err != nil {
		return err
	}

	phpVersion, err := binaryPHPVersion(bin)
	if err != nil {
		return err
	}
	channel, err := composer.Channel(phpVersion)
	if err != nil {
		return err
	}

	phar, err := composer.New().Ensure(cmd.Context(), base, channel, func() {
		fmt.Fprintf(os.Stderr, "Downloading Composer (%s) for PHP %s...\n", channel, phpVersion)
	})
	if err != nil {
		return err
	}

	if installed != "" {
		if err := os.Setenv(envVersion, installed); err != nil {
			return err
		}
	}
	return execBinary(bin, append([]string{phar}, args...))
}

// binaryPHPVersion asks the php binary for its exact version, since pvm may
// only know the branch (8.3) and Composer's requirement is a patch (7.2.5).
func binaryPHPVersion(bin string) (string, error) {
	out, err := exec.Command(bin, "-n", "-r", "echo PHP_MAJOR_VERSION.'.'.PHP_MINOR_VERSION.'.'.PHP_RELEASE_VERSION;").Output()
	if err != nil {
		return "", fmt.Errorf("read version of %s: %w", bin, err)
	}
	return strings.TrimSpace(string(out)), nil
}
