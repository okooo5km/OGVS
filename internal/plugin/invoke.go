// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package plugin

import (
	"fmt"

	"github.com/okooo5km/ogvs/internal/svgast"
)

// InvokePlugins resolves plugin configs and invokes them on the AST.
// This is the main entry point called from the optimize pipeline.
//
// It matches SVGO's invokePlugins flow:
//  1. For each plugin config, resolve it (find the builtin fn, merge params)
//  2. Call plugin.fn(ast, params, info) to get a Visitor
//  3. If visitor is non-nil, call Visit(ast, visitor)
func InvokePlugins(
	root *svgast.Root,
	info *PluginInfo,
	pluginConfigs []PluginConfig,
	globalOverrides map[string]any,
) error {
	for _, cfg := range pluginConfigs {
		resolved, err := resolvePluginConfig(cfg)
		if err != nil {
			return err
		}
		if resolved == nil {
			continue
		}

		// Merge params: resolved.Params + globalOverrides
		params := make(map[string]any)
		for k, v := range resolved.Params {
			params[k] = v
		}
		for k, v := range globalOverrides {
			params[k] = v
		}

		visitor := resolved.Fn(root, params, info)
		if visitor != nil {
			svgast.Visit(root, visitor, nil)
		}
	}
	return nil
}

// InvokeResolved invokes already-resolved plugins on the AST.
// Used by presets that have already resolved their sub-plugins.
func InvokeResolved(root *svgast.Root, info *PluginInfo, plugins []*ResolvedPlugin) {
	for _, p := range plugins {
		visitor := p.Fn(root, p.Params, info)
		if visitor != nil {
			svgast.Visit(root, visitor, nil)
		}
	}
}

// PluginConfig represents a user-specified plugin configuration.
// This is the input format — it gets resolved against the registry.
type PluginConfig struct {
	// Name is the plugin name (required).
	Name string

	// Params is plugin-specific parameters (optional).
	Params map[string]any

	// Fn is a custom plugin function (optional).
	// If nil, the builtin plugin is looked up by Name.
	Fn PluginFunc
}

// resolvePluginConfig resolves a PluginConfig into a ResolvedPlugin.
// Matches SVGO's resolvePluginConfig function.
func resolvePluginConfig(cfg PluginConfig) (*ResolvedPlugin, error) {
	fn := cfg.Fn
	if fn == nil {
		// Look up builtin plugin
		p := Get(cfg.Name)
		if p == nil {
			return nil, fmt.Errorf("unknown builtin plugin %q", cfg.Name)
		}
		fn = p.Fn
	}

	params := cfg.Params
	if params == nil {
		params = make(map[string]any)
	}

	return &ResolvedPlugin{
		Name:   cfg.Name,
		Params: params,
		Fn:     fn,
	}, nil
}
