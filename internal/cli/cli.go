// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package cli implements the ogvs command-line interface.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/okooo5km/ogvs/internal/version"
)

// CLI flags
var (
	flagInput      []string
	flagString     string
	flagFolder     string
	flagOutput     []string
	flagPrecision  int
	flagConfig     string
	flagDataURI    string
	flagMultipass  bool
	flagPretty     bool
	flagIndent     int
	flagEOL        string
	flagFinalNL    bool
	flagRecursive  bool
	flagExclude    []string
	flagQuiet      bool
	flagShowPlugin bool
)

// rootCmd is the base command for ogvs.
var rootCmd = &cobra.Command{
	Use:   "ogvs [flags] [INPUT...]",
	Short: "ogvs — SVG Optimizer (Go implementation of SVGO)",
	Long: `ogvs is a Go implementation of SVGO, the popular SVG optimizer.
It removes unnecessary data from SVG files, reducing their size
without affecting rendering.`,
	Version:       version.Version,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Validate precision range
		if cmd.Flags().Changed("precision") {
			if flagPrecision < 0 || flagPrecision > 20 {
				return fmt.Errorf("precision must be between 0 and 20, got %d", flagPrecision)
			}
		}

		// Validate eol value
		if cmd.Flags().Changed("eol") {
			if flagEOL != "lf" && flagEOL != "crlf" {
				return fmt.Errorf("eol must be 'lf' or 'crlf', got %q", flagEOL)
			}
		}

		// Validate datauri value
		if cmd.Flags().Changed("datauri") {
			if flagDataURI != "base64" && flagDataURI != "enc" && flagDataURI != "unenc" {
				return fmt.Errorf("datauri must be 'base64', 'enc', or 'unenc', got %q", flagDataURI)
			}
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, args)
	},
}

func init() {
	flags := rootCmd.Flags()

	flags.StringSliceVarP(&flagInput, "input", "i", nil, "input file(s), '-' for STDIN")
	flags.StringVarP(&flagString, "string", "s", "", "input SVG string")
	flags.StringVarP(&flagFolder, "folder", "f", "", "input folder")
	flags.StringSliceVarP(&flagOutput, "output", "o", nil, "output file(s)/folder, '-' for STDOUT")

	flags.IntVarP(&flagPrecision, "precision", "p", -1, "float precision (0-20), -1 to use defaults")
	flags.StringVar(&flagConfig, "config", "", "config file path")
	flags.StringVar(&flagDataURI, "datauri", "", "output as data URI (base64/enc/unenc)")
	flags.BoolVar(&flagMultipass, "multipass", false, "enable multipass optimization")
	flags.BoolVar(&flagPretty, "pretty", false, "pretty-print output")
	flags.IntVar(&flagIndent, "indent", 4, "indent size for pretty-print")
	flags.StringVar(&flagEOL, "eol", "lf", "line ending (lf/crlf)")
	flags.BoolVar(&flagFinalNL, "final-newline", false, "add final newline")

	flags.BoolVarP(&flagRecursive, "recursive", "r", false, "recurse into subfolders")
	flags.StringSliceVar(&flagExclude, "exclude", nil, "exclude file pattern(s) (regex)")
	flags.BoolVarP(&flagQuiet, "quiet", "q", false, "suppress output")
	flags.BoolVar(&flagShowPlugin, "show-plugins", false, "list available plugins")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
