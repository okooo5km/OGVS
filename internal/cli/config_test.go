// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/okooo5km/ogvs/internal/plugin"
)

// ---------- discoverConfig ----------

func TestDiscoverConfig_FoundYaml(t *testing.T) {
	// Create temp dir structure: base/sub1/sub2/
	base := t.TempDir()
	sub1 := filepath.Join(base, "sub1")
	sub2 := filepath.Join(sub1, "sub2")
	if err := os.MkdirAll(sub2, 0o755); err != nil {
		t.Fatal(err)
	}

	// Put ogvs.config.yaml in base/
	cfgPath := filepath.Join(base, "ogvs.config.yaml")
	if err := os.WriteFile(cfgPath, []byte("multipass: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Call discoverConfig from base/sub1/sub2/
	got := discoverConfig(sub2)
	if got != cfgPath {
		t.Errorf("discoverConfig(%q) = %q, want %q", sub2, got, cfgPath)
	}
}

func TestDiscoverConfig_FoundInCwd(t *testing.T) {
	// Config file is in the same directory as cwd
	base := t.TempDir()
	cfgPath := filepath.Join(base, "ogvs.config.yaml")
	if err := os.WriteFile(cfgPath, []byte("multipass: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := discoverConfig(base)
	if got != cfgPath {
		t.Errorf("discoverConfig(%q) = %q, want %q", base, got, cfgPath)
	}
}

func TestDiscoverConfig_YmlVariant(t *testing.T) {
	base := t.TempDir()
	cfgPath := filepath.Join(base, "ogvs.config.yml")
	if err := os.WriteFile(cfgPath, []byte("multipass: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := discoverConfig(base)
	if got != cfgPath {
		t.Errorf("discoverConfig(%q) = %q, want %q", base, got, cfgPath)
	}
}

func TestDiscoverConfig_YamlTakesPriorityOverYml(t *testing.T) {
	// When both .yaml and .yml exist in the same directory, .yaml should be found first
	// because configFileNames lists .yaml before .yml.
	base := t.TempDir()
	yamlPath := filepath.Join(base, "ogvs.config.yaml")
	ymlPath := filepath.Join(base, "ogvs.config.yml")
	if err := os.WriteFile(yamlPath, []byte("multipass: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ymlPath, []byte("multipass: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := discoverConfig(base)
	if got != yamlPath {
		t.Errorf("discoverConfig(%q) = %q, want %q (yaml should take priority over yml)", base, got, yamlPath)
	}
}

func TestDiscoverConfig_NotFound(t *testing.T) {
	// Create temp dir with no config files
	base := t.TempDir()
	sub := filepath.Join(base, "a", "b", "c")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got := discoverConfig(sub)
	// The search goes all the way up to root, so it might find a real config.
	// We verify that if it found something, it's not inside our temp dir (which has no config).
	// In practice, there should be no ogvs.config.yaml at / or other parents of /tmp.
	if got != "" {
		// Check it's not inside our temp dir (it shouldn't be, since we didn't create one)
		rel, err := filepath.Rel(base, got)
		if err == nil && !filepath.IsAbs(rel) && rel[0] != '.' {
			t.Errorf("discoverConfig(%q) found config inside temp dir: %q, but we didn't create one", sub, got)
		}
		// If it found something outside our temp dir, that's the real filesystem — acceptable.
		// But for a clean test, we just note it.
		t.Logf("note: discoverConfig found config outside temp dir at %q (real filesystem)", got)
	}
}

func TestDiscoverConfig_NestedFindsClosest(t *testing.T) {
	// base/ogvs.config.yaml
	// base/inner/ogvs.config.yaml
	// Call from base/inner/deep/ — should find base/inner/ogvs.config.yaml
	base := t.TempDir()
	inner := filepath.Join(base, "inner")
	deep := filepath.Join(inner, "deep")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	baseCfg := filepath.Join(base, "ogvs.config.yaml")
	innerCfg := filepath.Join(inner, "ogvs.config.yaml")
	if err := os.WriteFile(baseCfg, []byte("multipass: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(innerCfg, []byte("multipass: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := discoverConfig(deep)
	if got != innerCfg {
		t.Errorf("discoverConfig(%q) = %q, want %q (should find closest config)", deep, got, innerCfg)
	}
}

// ---------- loadConfig ----------

func TestLoadConfig_FullYAML(t *testing.T) {
	yamlContent := `multipass: true
floatPrecision: 3
datauri: base64
plugins:
  - preset-default
js2svg:
  pretty: true
  indent: 4
  eol: crlf
  finalNewline: true
`
	path := filepath.Join(t.TempDir(), "ogvs.config.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig(%q) error: %v", path, err)
	}

	if !cfg.Multipass {
		t.Error("expected Multipass = true")
	}
	if cfg.FloatPrecision == nil {
		t.Fatal("expected FloatPrecision to be non-nil")
	}
	if *cfg.FloatPrecision != 3 {
		t.Errorf("expected FloatPrecision = 3, got %d", *cfg.FloatPrecision)
	}
	if cfg.DataURI != "base64" {
		t.Errorf("expected DataURI = %q, got %q", "base64", cfg.DataURI)
	}
	if len(cfg.Plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(cfg.Plugins))
	}
	if cfg.Plugins[0] != "preset-default" {
		t.Errorf("expected plugin = %q, got %v", "preset-default", cfg.Plugins[0])
	}
	if cfg.Js2svg == nil {
		t.Fatal("expected Js2svg to be non-nil")
	}
	if !cfg.Js2svg.Pretty {
		t.Error("expected Js2svg.Pretty = true")
	}
	if cfg.Js2svg.Indent != 4 {
		t.Errorf("expected Js2svg.Indent = 4, got %d", cfg.Js2svg.Indent)
	}
	if cfg.Js2svg.EOL != "crlf" {
		t.Errorf("expected Js2svg.EOL = %q, got %q", "crlf", cfg.Js2svg.EOL)
	}
	if !cfg.Js2svg.FinalNewline {
		t.Error("expected Js2svg.FinalNewline = true")
	}
}

func TestLoadConfig_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ogvs.config.yaml")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig(%q) error: %v", path, err)
	}

	// Zero-value CLIConfig
	if cfg.Multipass {
		t.Error("expected Multipass = false for empty file")
	}
	if cfg.FloatPrecision != nil {
		t.Error("expected FloatPrecision = nil for empty file")
	}
	if cfg.DataURI != "" {
		t.Errorf("expected DataURI = %q for empty file, got %q", "", cfg.DataURI)
	}
	if len(cfg.Plugins) != 0 {
		t.Errorf("expected 0 plugins for empty file, got %d", len(cfg.Plugins))
	}
	if cfg.Js2svg != nil {
		t.Error("expected Js2svg = nil for empty file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ogvs.config.yaml")
	if err := os.WriteFile(path, []byte("{{{{invalid yaml!!!!"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := loadConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.yaml")
	_, err := loadConfig(path)
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestLoadConfig_FloatPrecisionZero(t *testing.T) {
	// floatPrecision: 0 is valid and should be *int pointing to 0, not nil
	yamlContent := `floatPrecision: 0
`
	path := filepath.Join(t.TempDir(), "ogvs.config.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	if cfg.FloatPrecision == nil {
		t.Fatal("expected FloatPrecision to be non-nil for value 0")
	}
	if *cfg.FloatPrecision != 0 {
		t.Errorf("expected FloatPrecision = 0, got %d", *cfg.FloatPrecision)
	}
}

func TestLoadConfig_PartialFields(t *testing.T) {
	// Only set some fields, others should be zero-valued
	yamlContent := `multipass: true
`
	path := filepath.Join(t.TempDir(), "ogvs.config.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	if !cfg.Multipass {
		t.Error("expected Multipass = true")
	}
	if cfg.FloatPrecision != nil {
		t.Error("expected FloatPrecision = nil when not set")
	}
	if cfg.Js2svg != nil {
		t.Error("expected Js2svg = nil when not set")
	}
	if len(cfg.Plugins) != 0 {
		t.Errorf("expected 0 plugins when not set, got %d", len(cfg.Plugins))
	}
}

func TestLoadConfig_PluginsWithObjects(t *testing.T) {
	// Plugins can be strings or maps — YAML unmarshals maps as map[string]any
	yamlContent := `plugins:
  - preset-default
  - name: removeComments
    params:
      removeAll: true
`
	path := filepath.Join(t.TempDir(), "ogvs.config.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	if len(cfg.Plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(cfg.Plugins))
	}

	// First should be a string
	s, ok := cfg.Plugins[0].(string)
	if !ok {
		t.Fatalf("expected Plugins[0] to be string, got %T", cfg.Plugins[0])
	}
	if s != "preset-default" {
		t.Errorf("expected Plugins[0] = %q, got %q", "preset-default", s)
	}

	// Second should be a map
	m, ok := cfg.Plugins[1].(map[string]any)
	if !ok {
		t.Fatalf("expected Plugins[1] to be map[string]any, got %T", cfg.Plugins[1])
	}
	if m["name"] != "removeComments" {
		t.Errorf("expected Plugins[1].name = %q, got %v", "removeComments", m["name"])
	}
}

func TestLoadConfig_Js2svgOnly(t *testing.T) {
	yamlContent := `js2svg:
  pretty: true
  indent: 2
`
	path := filepath.Join(t.TempDir(), "ogvs.config.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	if cfg.Js2svg == nil {
		t.Fatal("expected Js2svg to be non-nil")
	}
	if !cfg.Js2svg.Pretty {
		t.Error("expected Js2svg.Pretty = true")
	}
	if cfg.Js2svg.Indent != 2 {
		t.Errorf("expected Js2svg.Indent = 2, got %d", cfg.Js2svg.Indent)
	}
	// Unset fields should be zero-valued
	if cfg.Js2svg.EOL != "" {
		t.Errorf("expected Js2svg.EOL = %q, got %q", "", cfg.Js2svg.EOL)
	}
	if cfg.Js2svg.FinalNewline {
		t.Error("expected Js2svg.FinalNewline = false when not set")
	}
}

// ---------- resolvePluginConfigs ----------

func TestResolvePluginConfigs_String(t *testing.T) {
	raw := []any{"preset-default"}
	configs, err := resolvePluginConfigs(raw)
	if err != nil {
		t.Fatalf("resolvePluginConfigs error: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].Name != "preset-default" {
		t.Errorf("expected Name = %q, got %q", "preset-default", configs[0].Name)
	}
	if configs[0].Params != nil {
		t.Errorf("expected Params = nil for string entry, got %v", configs[0].Params)
	}
	if configs[0].Fn != nil {
		t.Error("expected Fn = nil for string entry")
	}
}

func TestResolvePluginConfigs_ObjectWithParams(t *testing.T) {
	raw := []any{
		map[string]any{
			"name": "removeComments",
			"params": map[string]any{
				"removeAll": true,
			},
		},
	}
	configs, err := resolvePluginConfigs(raw)
	if err != nil {
		t.Fatalf("resolvePluginConfigs error: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].Name != "removeComments" {
		t.Errorf("expected Name = %q, got %q", "removeComments", configs[0].Name)
	}
	if configs[0].Params == nil {
		t.Fatal("expected Params to be non-nil")
	}
	val, ok := configs[0].Params["removeAll"]
	if !ok {
		t.Fatal("expected Params to contain 'removeAll'")
	}
	if val != true {
		t.Errorf("expected Params['removeAll'] = true, got %v", val)
	}
}

func TestResolvePluginConfigs_ObjectWithoutParams(t *testing.T) {
	raw := []any{
		map[string]any{
			"name": "removeTitle",
		},
	}
	configs, err := resolvePluginConfigs(raw)
	if err != nil {
		t.Fatalf("resolvePluginConfigs error: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].Name != "removeTitle" {
		t.Errorf("expected Name = %q, got %q", "removeTitle", configs[0].Name)
	}
	if configs[0].Params != nil {
		t.Errorf("expected Params = nil when not provided, got %v", configs[0].Params)
	}
}

func TestResolvePluginConfigs_Mixed(t *testing.T) {
	raw := []any{
		"preset-default",
		map[string]any{
			"name": "removeTitle",
		},
		map[string]any{
			"name": "removeComments",
			"params": map[string]any{
				"removeAll": true,
			},
		},
	}
	configs, err := resolvePluginConfigs(raw)
	if err != nil {
		t.Fatalf("resolvePluginConfigs error: %v", err)
	}
	if len(configs) != 3 {
		t.Fatalf("expected 3 configs, got %d", len(configs))
	}

	// First: string entry
	if configs[0].Name != "preset-default" {
		t.Errorf("configs[0].Name = %q, want %q", configs[0].Name, "preset-default")
	}
	if configs[0].Params != nil {
		t.Errorf("configs[0].Params should be nil for string entry")
	}

	// Second: object without params
	if configs[1].Name != "removeTitle" {
		t.Errorf("configs[1].Name = %q, want %q", configs[1].Name, "removeTitle")
	}

	// Third: object with params
	if configs[2].Name != "removeComments" {
		t.Errorf("configs[2].Name = %q, want %q", configs[2].Name, "removeComments")
	}
	if configs[2].Params == nil || configs[2].Params["removeAll"] != true {
		t.Errorf("configs[2].Params should have removeAll=true, got %v", configs[2].Params)
	}
}

func TestResolvePluginConfigs_InvalidType(t *testing.T) {
	raw := []any{42}
	_, err := resolvePluginConfigs(raw)
	if err == nil {
		t.Fatal("expected error for integer entry, got nil")
	}
}

func TestResolvePluginConfigs_InvalidBoolType(t *testing.T) {
	raw := []any{true}
	_, err := resolvePluginConfigs(raw)
	if err == nil {
		t.Fatal("expected error for bool entry, got nil")
	}
}

func TestResolvePluginConfigs_EmptyArray(t *testing.T) {
	raw := []any{}
	configs, err := resolvePluginConfigs(raw)
	if err != nil {
		t.Fatalf("resolvePluginConfigs error: %v", err)
	}
	if configs != nil {
		t.Errorf("expected nil for empty array, got %v", configs)
	}
}

func TestResolvePluginConfigs_NilInput(t *testing.T) {
	configs, err := resolvePluginConfigs(nil)
	if err != nil {
		t.Fatalf("resolvePluginConfigs error: %v", err)
	}
	if configs != nil {
		t.Errorf("expected nil for nil input, got %v", configs)
	}
}

func TestResolvePluginConfigs_ObjectMissingName(t *testing.T) {
	// Map without "name" field should error
	raw := []any{
		map[string]any{
			"params": map[string]any{"foo": "bar"},
		},
	}
	_, err := resolvePluginConfigs(raw)
	if err == nil {
		t.Fatal("expected error for map without 'name' field, got nil")
	}
}

func TestResolvePluginConfigs_ObjectNameNotString(t *testing.T) {
	// "name" field is not a string
	raw := []any{
		map[string]any{
			"name": 123,
		},
	}
	_, err := resolvePluginConfigs(raw)
	if err == nil {
		t.Fatal("expected error when 'name' is not a string, got nil")
	}
}

func TestResolvePluginConfigs_MultipleStrings(t *testing.T) {
	raw := []any{"removeComments", "removeTitle", "removeDesc"}
	configs, err := resolvePluginConfigs(raw)
	if err != nil {
		t.Fatalf("resolvePluginConfigs error: %v", err)
	}
	if len(configs) != 3 {
		t.Fatalf("expected 3 configs, got %d", len(configs))
	}

	expected := []string{"removeComments", "removeTitle", "removeDesc"}
	for i, name := range expected {
		if configs[i].Name != name {
			t.Errorf("configs[%d].Name = %q, want %q", i, configs[i].Name, name)
		}
	}
}

func TestResolvePluginConfigs_ReturnsPluginConfig(t *testing.T) {
	// Verify the returned type is []plugin.PluginConfig
	raw := []any{"preset-default"}
	configs, err := resolvePluginConfigs(raw)
	if err != nil {
		t.Fatalf("resolvePluginConfigs error: %v", err)
	}

	// Type assertion to confirm it's plugin.PluginConfig
	var _ []plugin.PluginConfig = configs
	if configs[0].Name != "preset-default" {
		t.Errorf("unexpected name: %q", configs[0].Name)
	}
}

func TestResolvePluginConfigs_ErrorIncludesIndex(t *testing.T) {
	// Error messages should include the entry index
	raw := []any{"valid-plugin", 3.14}
	_, err := resolvePluginConfigs(raw)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	errMsg := err.Error()
	// Should mention "entry 1" (the second entry)
	if !containsSubstring(errMsg, "1") {
		t.Errorf("error message should mention index 1, got: %q", errMsg)
	}
}

func TestResolvePluginConfigs_ParamsNonMapIgnored(t *testing.T) {
	// If "params" exists but is not a map, it should be treated as nil params (not error)
	raw := []any{
		map[string]any{
			"name":   "removeComments",
			"params": "not-a-map",
		},
	}
	configs, err := resolvePluginConfigs(raw)
	if err != nil {
		t.Fatalf("resolvePluginConfigs error: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].Params != nil {
		t.Errorf("expected Params = nil when params is not a map, got %v", configs[0].Params)
	}
}

// containsSubstring checks if s contains substr.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
