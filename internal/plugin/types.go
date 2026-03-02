// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package plugin provides the plugin interface, registry, preset system,
// and execution engine for the SVG optimization pipeline.
//
// This mirrors SVGO's plugin architecture:
//   - Plugins are functions that receive the AST + params and return a Visitor
//   - Plugins are registered in a global registry by name
//   - Presets group multiple plugins into an ordered execution list
//   - invokePlugins iterates plugins sequentially, calling each fn then visiting the AST
package plugin

import "github.com/okooo5km/ogvs/internal/svgast"

// PluginInfo provides contextual information to plugin functions.
// Matches SVGO's info object passed to plugin fn.
type PluginInfo struct {
	// Path is the file path of the SVG being optimized.
	Path string

	// MultipassCount is the current pass number (0-based) during multipass optimization.
	MultipassCount int
}

// PluginFunc is the function signature for plugin implementations.
//
// It receives:
//   - root: the full AST tree
//   - params: merged parameters (plugin defaults + global overrides + user overrides)
//   - info: contextual information (file path, multipass count)
//
// It returns a Visitor to traverse the AST, or nil to skip this plugin.
// Matches SVGO's: (ast, params, info) => visitor | null
type PluginFunc func(root *svgast.Root, params map[string]any, info *PluginInfo) *svgast.Visitor

// Plugin represents a registered plugin with metadata.
type Plugin struct {
	// Name is the unique plugin identifier (e.g. "removeComments").
	Name string

	// Description is a human-readable description of what the plugin does.
	Description string

	// Fn is the plugin implementation function.
	Fn PluginFunc

	// IsPreset indicates this is a preset (contains sub-plugins).
	IsPreset bool

	// Plugins holds the sub-plugins for presets.
	// Only used when IsPreset is true.
	Plugins []*Plugin
}

// ResolvedPlugin is a plugin ready for execution with its merged params.
// This is the result of resolving a PluginConfig against the registry.
type ResolvedPlugin struct {
	Name   string
	Params map[string]any
	Fn     PluginFunc
}
