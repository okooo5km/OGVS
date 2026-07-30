// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package core

import (
	"math"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

// Optimize runs the SVG optimization pipeline.
//
// This matches SVGO's optimize() function:
// 1. Parse SVG string to XAST
// 2. Invoke plugins to transform the AST
// 3. Stringify XAST back to SVG
// 4. If multipass enabled, repeat until output size stops decreasing (max 10 passes)
func Optimize(input string, config *Config) (*Output, error) {
	if config == nil {
		config = &Config{}
	}

	maxPasses := 1
	if config.Multipass {
		maxPasses = 10
	}

	// Resolve stringify options
	js2svg := config.Js2svg
	if js2svg == nil {
		js2svg = svgast.DefaultStringifyOptions()
	}

	// Resolve plugin configs (default to preset-default)
	// Resolve plugin configs (default to preset-default when nil, like SVGO)
	pluginConfigs := config.Plugins
	if pluginConfigs == nil {
		pluginConfigs = []plugin.PluginConfig{
			{Name: "preset-default"},
		}
	}

	// Build global overrides
	// Plugins read numeric params as float64 (the shape JSON decoding
	// produces), so the override has to be widened before it is handed over.
	globalOverrides := make(map[string]any)
	if config.FloatPrecision != nil {
		globalOverrides["floatPrecision"] = float64(*config.FloatPrecision)
	}

	currentInput := input
	output := input
	prevSize := math.MaxInt // ensure first pass always runs (matches SVGO's Number.MAX_SAFE_INTEGER)

	for i := range maxPasses {
		// Parse
		root, err := svgast.ParseSvg(currentInput, config.Path)
		if err != nil {
			return nil, err
		}

		// Invoke plugins
		info := &plugin.PluginInfo{
			Path:           config.Path,
			MultipassCount: i,
		}
		if err := plugin.InvokePlugins(root, info, pluginConfigs, globalOverrides); err != nil {
			return nil, err
		}

		// Stringify
		output = svgast.StringifySvg(root, js2svg)

		// Check convergence. SVGO keeps the bytes of the pass it just ran even
		// when that pass did not shrink the document, so the result of the
		// final, non-improving pass is what gets returned.
		if len(output) >= prevSize {
			break
		}

		prevSize = len(output)
		currentInput = output
	}

	return &Output{Data: output}, nil
}
