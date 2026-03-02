// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package core provides the SVG optimization pipeline.
package core

import (
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

// Config defines the optimization configuration.
type Config struct {
	// Path is the file path of the SVG being optimized (for plugin context).
	Path string

	// Multipass enables running the plugin chain up to 10 times
	// until the output size stops decreasing.
	Multipass bool

	// FloatPrecision sets the global float precision override.
	// nil means use plugin defaults.
	FloatPrecision *int

	// Plugins is the list of plugin configurations.
	// Default: ["preset-default"]
	Plugins []plugin.PluginConfig

	// Js2svg configures the stringifier output.
	Js2svg *svgast.StringifyOptions
}

// Output is the result of an optimization run.
type Output struct {
	Data string
}
