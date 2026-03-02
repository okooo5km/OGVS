// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package plugin

import (
	"fmt"

	"github.com/okooo5km/ogvs/internal/svgast"
)

// CreatePreset creates a preset plugin that groups multiple plugins.
// Matches SVGO's createPreset() function in plugins.js.
//
// A preset is itself a plugin whose fn calls invokePlugins on its sub-plugins.
// Preset params support:
//   - "overrides": map[string]any — per-plugin param overrides (or false to disable)
//   - "floatPrecision": float64 — global float precision for all sub-plugins
func CreatePreset(name string, plugins []*Plugin) *Plugin {
	return &Plugin{
		Name:     name,
		IsPreset: true,
		Plugins:  plugins,
		Fn: func(root *svgast.Root, params map[string]any, info *PluginInfo) *svgast.Visitor {
			// Extract preset-level params
			var overrides map[string]any
			if ov, ok := params["overrides"]; ok {
				if ovMap, ok := ov.(map[string]any); ok {
					overrides = ovMap
				}
			}

			globalOverrides := make(map[string]any)
			if fp, ok := params["floatPrecision"]; ok {
				globalOverrides["floatPrecision"] = fp
			}

			// Validate overrides reference valid plugin names
			if overrides != nil {
				pluginNames := make(map[string]bool, len(plugins))
				for _, p := range plugins {
					pluginNames[p.Name] = true
				}
				for pName := range overrides {
					if !pluginNames[pName] {
						fmt.Printf("Warning: plugin %q is not part of %s\n", pName, name)
					}
				}
			}

			// Resolve sub-plugins and invoke them
			resolved := resolvePresetPlugins(plugins, overrides, globalOverrides)
			InvokeResolved(root, info, resolved)

			// Preset itself returns nil (it already invoked sub-plugins directly)
			return nil
		},
	}
}

// resolvePresetPlugins resolves preset sub-plugins with overrides applied.
func resolvePresetPlugins(plugins []*Plugin, overrides, globalOverrides map[string]any) []*ResolvedPlugin {
	resolved := make([]*ResolvedPlugin, 0, len(plugins))
	for _, p := range plugins {
		// Check if plugin is disabled via overrides
		if overrides != nil {
			if ov, ok := overrides[p.Name]; ok {
				if ov == false || ov == nil {
					continue // skip disabled plugin
				}
			}
		}

		// Merge params: plugin.params (none for sub-plugins) + globalOverrides + override
		params := make(map[string]any)
		for k, v := range globalOverrides {
			params[k] = v
		}
		if overrides != nil {
			if ov, ok := overrides[p.Name]; ok {
				if ovMap, ok := ov.(map[string]any); ok {
					for k, v := range ovMap {
						params[k] = v
					}
				}
			}
		}

		resolved = append(resolved, &ResolvedPlugin{
			Name:   p.Name,
			Params: params,
			Fn:     p.Fn,
		})
	}
	return resolved
}

// PresetDefaultPluginNames lists the 34 plugins in preset-default order.
// Matches SVGO's plugins/preset-default.js exactly.
var PresetDefaultPluginNames = []string{
	"removeDoctype",
	"removeXMLProcInst",
	"removeComments",
	"removeDeprecatedAttrs",
	"removeMetadata",
	"removeEditorsNSData",
	"cleanupAttrs",
	"mergeStyles",
	"inlineStyles",
	"minifyStyles",
	"cleanupIds",
	"removeUselessDefs",
	"cleanupNumericValues",
	"convertColors",
	"removeUnknownsAndDefaults",
	"removeNonInheritableGroupAttrs",
	"removeUselessStrokeAndFill",
	"cleanupEnableBackground",
	"removeHiddenElems",
	"removeEmptyText",
	"convertShapeToPath",
	"convertEllipseToCircle",
	"moveElemsAttrsToGroup",
	"moveGroupAttrsToElems",
	"collapseGroups",
	"convertPathData",
	"convertTransform",
	"removeEmptyAttrs",
	"removeEmptyContainers",
	"mergePaths",
	"removeUnusedNS",
	"sortAttrs",
	"sortDefsChildren",
	"removeDesc",
}

// RegisterPresetDefault creates and registers the preset-default plugin.
// This should be called after all individual plugins are registered.
// Returns the created preset plugin.
func RegisterPresetDefault() *Plugin {
	plugins := make([]*Plugin, 0, len(PresetDefaultPluginNames))
	for _, name := range PresetDefaultPluginNames {
		p := Get(name)
		if p != nil {
			plugins = append(plugins, p)
		}
		// Skip plugins not yet registered (will be added in later phases)
	}
	preset := CreatePreset("preset-default", plugins)
	Register(preset)
	return preset
}
