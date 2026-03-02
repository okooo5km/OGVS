// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/okooo5km/ogvs/internal/plugin"
)

// CLIConfig represents the YAML configuration file structure.
type CLIConfig struct {
	Multipass      bool          `yaml:"multipass"`
	FloatPrecision *int          `yaml:"floatPrecision"`
	DataURI        string        `yaml:"datauri"`
	Plugins        []any         `yaml:"plugins"` // string or map
	Js2svg         *Js2svgConfig `yaml:"js2svg"`
}

// Js2svgConfig represents js2svg output configuration.
type Js2svgConfig struct {
	Pretty       bool   `yaml:"pretty"`
	Indent       int    `yaml:"indent"`
	EOL          string `yaml:"eol"`
	FinalNewline bool   `yaml:"finalNewline"`
}

// configFileNames are the config file names to search for.
var configFileNames = []string{
	"ogvs.config.yaml",
	"ogvs.config.yml",
}

// discoverConfig searches upward from cwd for a config file.
// Returns the path if found, or empty string if not found.
func discoverConfig(cwd string) string {
	dir, err := filepath.Abs(cwd)
	if err != nil {
		return ""
	}

	for {
		for _, name := range configFileNames {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}

	return ""
}

// loadConfig reads and parses a YAML config file.
func loadConfig(path string) (*CLIConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg CLIConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	return &cfg, nil
}

// resolvePluginConfigs converts the mixed-type plugins list from YAML
// (strings and maps) into a typed []plugin.PluginConfig slice.
func resolvePluginConfigs(raw []any) ([]plugin.PluginConfig, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	configs := make([]plugin.PluginConfig, 0, len(raw))
	for i, item := range raw {
		switch v := item.(type) {
		case string:
			// Simple plugin name: "preset-default" or "removeComments"
			configs = append(configs, plugin.PluginConfig{Name: v})
		case map[string]any:
			// Object form: {name: "preset-default", params: {...}}
			name, ok := v["name"].(string)
			if !ok {
				return nil, fmt.Errorf("plugin entry %d: missing or invalid 'name' field", i)
			}
			var params map[string]any
			if p, ok := v["params"]; ok {
				if pMap, ok := p.(map[string]any); ok {
					params = pMap
				}
			}
			configs = append(configs, plugin.PluginConfig{
				Name:   name,
				Params: params,
			})
		default:
			return nil, fmt.Errorf("plugin entry %d: unsupported type %T", i, item)
		}
	}
	return configs, nil
}
