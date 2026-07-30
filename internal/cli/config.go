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
	Multipass      bool `yaml:"multipass"`
	FloatPrecision *int `yaml:"floatPrecision"`
	// DataURI encodes the output as a data URI (base64/enc/unenc), and is
	// overridden by --datauri. SVGO applies the encoding twice when it comes
	// from a config file (once in optimize(), again in its CLI layer); OGVS
	// encodes exactly once.
	DataURI string        `yaml:"datauri"`
	Plugins []any         `yaml:"plugins"` // string or map
	Js2svg  *Js2svgConfig `yaml:"js2svg"`
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

// configInDir returns the path of the first config file present in dir, or ""
// if none is. Entries that are not regular files — most notably a directory
// named ogvs.config.yaml — are skipped rather than returned, mirroring the
// stats.isFile() guard in SVGO's lib/svgo-node.js. Returning such an entry
// would make loadConfig fail on read and abort an otherwise valid run.
func configInDir(dir string) string {
	for _, name := range configFileNames {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}

// findProjectRoot returns the nearest ancestor of dir (dir itself included)
// that holds a .git entry. It is a file in a worktree or submodule checkout
// and a directory in a normal clone, so presence — not type — is what counts.
//
// go.mod and package.json are deliberately not markers: both legitimately
// appear in subdirectories of a monorepo, so they would stop the search short
// of a repository-root config.
func findProjectRoot(dir string) (string, bool) {
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// discoverConfig searches for a config file starting at cwd.
//
// SVGO (loadConfig in lib/svgo-node.js) walks every ancestor up to the
// filesystem root, so any ogvs.config.yaml above the working directory — in
// /tmp, in a shared checkout, in $HOME — would silently redefine the plugin
// set for every run beneath it. OGVS bounds the search instead: cwd is always
// consulted, and ancestors only up to the enclosing project root. With no
// enclosing project, only cwd is consulted. Running from a subdirectory of a
// repository that keeps its config at the root keeps working, which is the
// case the upward search exists to serve.
//
// Returns the path if found, or empty string if not found.
func discoverConfig(cwd string) string {
	dir, err := filepath.Abs(cwd)
	if err != nil {
		return ""
	}

	// cwd is always in scope, project or not.
	if path := configInDir(dir); path != "" {
		return path
	}

	root, ok := findProjectRoot(dir)
	if !ok {
		return ""
	}

	// Ascend to the project root inclusive, nearest config wins.
	for dir != root {
		parent := filepath.Dir(dir)
		if parent == dir {
			return "" // defensive: root was not an ancestor after all
		}
		dir = parent

		if path := configInDir(dir); path != "" {
			return path
		}
	}
	return ""
}

// loadConfig reads and parses a YAML config file.
func loadConfig(path string) (*CLIConfig, error) {
	// A --config pointing at a directory otherwise surfaces as a bare EISDIR
	// from ReadFile; say what is actually wrong. Only directories are rejected:
	// process substitution and pipes hand over character devices and FIFOs,
	// which read back just fine.
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return nil, fmt.Errorf("config file %s is a directory, not a file", path)
	}

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
// normalizeParamNumbers converts integer values to float64 throughout a params
// tree. yaml.v3 decodes `floatPrecision: 3` as int, but plugins type-assert
// float64 (the shape JSON produces), so integer literals would be silently
// ignored and the default used instead.
func normalizeParamNumbers(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, item := range val {
			out[k] = normalizeParamNumbers(item)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = normalizeParamNumbers(item)
		}
		return out
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case uint64:
		return float64(val)
	default:
		return v
	}
}

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
					params = normalizeParamNumbers(pMap).(map[string]any)
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
