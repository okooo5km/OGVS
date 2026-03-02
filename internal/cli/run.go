// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/okooo5km/ogvs/internal/core"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
	"github.com/okooo5km/ogvs/internal/tools"
)

// run is the main CLI action dispatched from rootCmd.RunE.
func run(cmd *cobra.Command, args []string) error {
	// --show-plugins: list plugins and exit
	if flagShowPlugin {
		showPlugins()
		return nil
	}

	// Load config file
	cfg, err := loadFinalConfig(cmd)
	if err != nil {
		return err
	}

	// Route input sources
	switch {
	case flagString != "":
		return processStringInput(cfg)
	case flagFolder != "":
		return optimizeFolder(cfg)
	default:
		return processFileInputs(cfg, cmd, args)
	}
}

// loadFinalConfig loads the config file (if any) and merges CLI flags on top.
func loadFinalConfig(cmd *cobra.Command) (*core.Config, error) {
	var fileCfg *CLIConfig

	// Discover or load config file
	if flagConfig != "" {
		var err error
		fileCfg, err = loadConfig(flagConfig)
		if err != nil {
			return nil, err
		}
	} else {
		cwd, err := os.Getwd()
		if err == nil {
			if path := discoverConfig(cwd); path != "" {
				fileCfg, err = loadConfig(path)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// Start with defaults
	cfg := &core.Config{}

	// Apply config file settings
	if fileCfg != nil {
		cfg.Multipass = fileCfg.Multipass
		cfg.FloatPrecision = fileCfg.FloatPrecision

		if len(fileCfg.Plugins) > 0 {
			pluginConfigs, err := resolvePluginConfigs(fileCfg.Plugins)
			if err != nil {
				return nil, fmt.Errorf("config file: %w", err)
			}
			cfg.Plugins = pluginConfigs
		}

		if fileCfg.Js2svg != nil {
			opts := svgast.DefaultStringifyOptions()
			opts.Pretty = fileCfg.Js2svg.Pretty
			if fileCfg.Js2svg.Indent > 0 {
				opts.Indent = fileCfg.Js2svg.Indent
			}
			if fileCfg.Js2svg.EOL != "" {
				opts.EOL = fileCfg.Js2svg.EOL
			}
			opts.FinalNewline = fileCfg.Js2svg.FinalNewline
			cfg.Js2svg = opts
		}
	}

	// CLI flags override config file
	if cmd.Flags().Changed("multipass") {
		cfg.Multipass = flagMultipass
	}
	if cmd.Flags().Changed("precision") && flagPrecision >= 0 {
		cfg.FloatPrecision = &flagPrecision
	}

	// js2svg overrides from CLI flags
	if cmd.Flags().Changed("pretty") || cmd.Flags().Changed("indent") ||
		cmd.Flags().Changed("eol") || cmd.Flags().Changed("final-newline") {
		if cfg.Js2svg == nil {
			cfg.Js2svg = svgast.DefaultStringifyOptions()
		}
		if cmd.Flags().Changed("pretty") {
			cfg.Js2svg.Pretty = flagPretty
		}
		if cmd.Flags().Changed("indent") {
			cfg.Js2svg.Indent = flagIndent
		}
		if cmd.Flags().Changed("eol") {
			cfg.Js2svg.EOL = flagEOL
		}
		if cmd.Flags().Changed("final-newline") {
			cfg.Js2svg.FinalNewline = flagFinalNL
		}
	}

	return cfg, nil
}

// processStringInput handles -s/--string input.
func processStringInput(cfg *core.Config) error {
	data := flagString

	// Decode data URI if needed
	data = tools.DecodeSVGDataURI(data)

	outputPath := ""
	if len(flagOutput) > 0 {
		outputPath = flagOutput[0]
	}

	return processSVGData(cfg, data, outputPath, "")
}

// processFileInputs handles file inputs from -i flags and positional args.
func processFileInputs(cfg *core.Config, cmd *cobra.Command, args []string) error {
	// Collect all input files
	inputs := make([]string, 0)
	inputs = append(inputs, flagInput...)
	inputs = append(inputs, args...)

	// If no inputs, check if stdin is piped
	if len(inputs) == 0 {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// stdin is piped
			return processStdinInput(cfg)
		}
		// No input at all — show help
		return cmd.Help()
	}

	// Check for stdin marker
	if len(inputs) == 1 && inputs[0] == "-" {
		return processStdinInput(cfg)
	}

	// Process each file
	for i, inputPath := range inputs {
		outputPath := ""
		if i < len(flagOutput) {
			outputPath = flagOutput[i]
		}
		if err := optimizeFile(cfg, inputPath, outputPath); err != nil {
			return err
		}
	}
	return nil
}

// processStdinInput reads SVG from stdin and optimizes it.
func processStdinInput(cfg *core.Config) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	outputPath := ""
	if len(flagOutput) > 0 {
		outputPath = flagOutput[0]
	}

	return processSVGData(cfg, string(data), outputPath, "")
}

// processSVGData optimizes an SVG string and writes the result.
func processSVGData(cfg *core.Config, data string, outputPath string, inputName string) error {
	start := time.Now()

	result, err := core.Optimize(data, cfg)
	if err != nil {
		return fmt.Errorf("optimization failed: %w", err)
	}

	output := result.Data

	// Apply data URI encoding if requested
	if flagDataURI != "" {
		output = tools.EncodeSVGDataURI(output, flagDataURI)
	}

	elapsed := time.Since(start)

	// Write output
	if outputPath == "" || outputPath == "-" {
		fmt.Print(output)
	} else {
		if err := writeFile(outputPath, output); err != nil {
			return err
		}
	}

	// Print stats (only when not writing to stdout and not quiet)
	if !flagQuiet && outputPath != "" && outputPath != "-" {
		printStats(inputName, len(data), len(output), elapsed)
	}

	return nil
}

// optimizeFile reads a file, optimizes it, and writes the result.
func optimizeFile(cfg *core.Config, inputPath string, outputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", inputPath, err)
	}

	// Set the file path in config for plugin context
	fileCfg := *cfg
	fileCfg.Path = inputPath

	start := time.Now()

	result, err := core.Optimize(string(data), &fileCfg)
	if err != nil {
		return fmt.Errorf("optimization failed for %s: %w", inputPath, err)
	}

	output := result.Data

	// Apply data URI encoding if requested
	if flagDataURI != "" {
		output = tools.EncodeSVGDataURI(output, flagDataURI)
	}

	elapsed := time.Since(start)

	// Determine output path
	if outputPath == "" {
		outputPath = inputPath // overwrite in place
	}

	if outputPath == "-" {
		fmt.Print(output)
	} else {
		if err := writeFile(outputPath, output); err != nil {
			return err
		}
	}

	// Print stats
	if !flagQuiet && outputPath != "-" {
		printStats(inputPath, len(data), len(output), elapsed)
	}

	return nil
}

// optimizeFolder processes all *.svg files in a folder.
func optimizeFolder(cfg *core.Config) error {
	folder := flagFolder
	outputFolder := ""
	if len(flagOutput) > 0 {
		outputFolder = flagOutput[0]
	}

	// Compile exclude patterns
	var excludePatterns []*regexp.Regexp
	for _, pat := range flagExclude {
		re, err := regexp.Compile(pat)
		if err != nil {
			return fmt.Errorf("invalid exclude pattern %q: %w", pat, err)
		}
		excludePatterns = append(excludePatterns, re)
	}

	// Walk the folder
	return filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories (but recurse if needed)
		if info.IsDir() {
			if path != folder && !flagRecursive {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .svg files
		if strings.ToLower(filepath.Ext(path)) != ".svg" {
			return nil
		}

		// Check exclude patterns
		relPath, _ := filepath.Rel(folder, path)
		for _, re := range excludePatterns {
			if re.MatchString(relPath) || re.MatchString(filepath.Base(path)) {
				return nil
			}
		}

		// Determine output path
		outputPath := ""
		if outputFolder != "" {
			outputPath = filepath.Join(outputFolder, relPath)
		}

		return optimizeFile(cfg, path, outputPath)
	})
}

// writeFile writes data to a file, creating parent directories as needed.
func writeFile(path string, data string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

// printStats prints optimization statistics for a file.
// Format matches SVGO:
//
//	input.svg:
//	Done in 12 ms!
//	1.234 KiB - 45.2% = 0.677 KiB
func printStats(name string, inSize, outSize int, elapsed time.Duration) {
	if name != "" {
		fmt.Fprintf(os.Stderr, "\n%s:\n", name)
	}
	fmt.Fprintf(os.Stderr, "Done in %d ms!\n", elapsed.Milliseconds())

	inKiB := math.Round(float64(inSize)/1024*1000) / 1000
	outKiB := math.Round(float64(outSize)/1024*1000) / 1000

	if inSize == 0 {
		fmt.Fprintf(os.Stderr, "%.3g KiB - 0%% = %.3g KiB\n", inKiB, outKiB)
		return
	}

	profit := 100 - float64(outSize)*100/float64(inSize)
	profitAbs := math.Abs(math.Round(profit*10) / 10)

	sign := "-"
	if profit < 0 {
		sign = "+"
	}

	fmt.Fprintf(os.Stderr, "%.3g KiB %s %.1f%% = %.3g KiB\n", inKiB, sign, profitAbs, outKiB)
}

// showPlugins lists all available plugins with descriptions.
func showPlugins() {
	names := plugin.Names()
	sort.Strings(names)

	// Build preset membership map
	presetMap := make(map[string][]string)
	for _, name := range names {
		p := plugin.Get(name)
		if p != nil && p.IsPreset {
			for _, sub := range p.Plugins {
				presetMap[sub.Name] = append(presetMap[sub.Name], name)
			}
		}
	}

	fmt.Println("Currently available plugins:")
	fmt.Println()
	for _, name := range names {
		p := plugin.Get(name)
		if p == nil || p.IsPreset {
			continue
		}

		desc := p.Description
		if desc == "" {
			desc = "(no description)"
		}

		presets := presetMap[name]
		presetInfo := ""
		if len(presets) > 0 {
			presetInfo = " (" + strings.Join(presets, ", ") + ")"
		}

		fmt.Printf("  [ %s ] %s%s\n", name, desc, presetInfo)
	}
}
