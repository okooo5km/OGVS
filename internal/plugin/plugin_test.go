// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package plugin

import (
	"testing"

	"github.com/okooo5km/ogvs/internal/svgast"
)

// --- Registry Tests ---

func TestRegister_And_Get(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	p := &Plugin{
		Name:        "testPlugin",
		Description: "a test plugin",
		Fn: func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor {
			return nil
		},
	}
	Register(p)

	got := Get("testPlugin")
	if got == nil {
		t.Fatal("Get returned nil for registered plugin")
	}
	if got.Name != "testPlugin" {
		t.Errorf("Name = %q, want %q", got.Name, "testPlugin")
	}
	if got.Description != "a test plugin" {
		t.Errorf("Description = %q", got.Description)
	}
}

func TestGet_NotFound(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	if Get("nonexistent") != nil {
		t.Error("Get returned non-nil for unregistered plugin")
	}
}

func TestHas(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	Register(&Plugin{Name: "exists", Fn: func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor { return nil }})

	if !Has("exists") {
		t.Error("Has returned false for registered plugin")
	}
	if Has("doesNotExist") {
		t.Error("Has returned true for unregistered plugin")
	}
}

func TestRegister_Duplicate_Panics(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	p := &Plugin{Name: "dup", Fn: func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor { return nil }}
	Register(p)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate registration")
		}
	}()
	Register(p) // should panic
}

func TestNames(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	Register(&Plugin{Name: "alpha", Fn: func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor { return nil }})
	Register(&Plugin{Name: "beta", Fn: func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor { return nil }})

	names := Names()
	if len(names) != 2 {
		t.Errorf("Names() returned %d items, want 2", len(names))
	}

	nameSet := map[string]bool{}
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["alpha"] || !nameSet["beta"] {
		t.Errorf("Names() = %v, want [alpha beta]", names)
	}
}

// --- InvokePlugins Tests ---

func TestInvokePlugins_EmptyList(t *testing.T) {
	root := &svgast.Root{
		Children: []svgast.Node{
			&svgast.Element{Name: "svg", Attributes: svgast.NewOrderedAttrs()},
		},
	}
	// Empty plugin list should not error
	err := InvokePlugins(root, &PluginInfo{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInvokePlugins_CustomPlugin(t *testing.T) {
	root := &svgast.Root{
		Children: []svgast.Node{
			&svgast.Element{
				Name:       "svg",
				Attributes: svgast.NewOrderedAttrs(),
				Children: []svgast.Node{
					&svgast.Comment{Value: "remove me"},
					&svgast.Element{Name: "g", Attributes: svgast.NewOrderedAttrs()},
				},
			},
		},
	}

	configs := []PluginConfig{
		{
			Name: "remove-comments",
			Fn: func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor {
				return &svgast.Visitor{
					Comment: &svgast.VisitorCallbacks{
						Enter: func(node svgast.Node, parent svgast.Parent) error {
							svgast.DetachNodeFromParent(node, parent)
							return nil
						},
					},
				}
			},
		},
	}

	err := InvokePlugins(root, &PluginInfo{}, configs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svg := root.Children[0].(*svgast.Element)
	if len(svg.Children) != 1 {
		t.Errorf("svg children = %d, want 1 (comment should be removed)", len(svg.Children))
	}
	if svg.Children[0].(*svgast.Element).Name != "g" {
		t.Error("remaining child should be <g>")
	}
}

func TestInvokePlugins_PluginReturnsNil(t *testing.T) {
	root := &svgast.Root{
		Children: []svgast.Node{
			&svgast.Element{Name: "svg", Attributes: svgast.NewOrderedAttrs()},
		},
	}

	configs := []PluginConfig{
		{
			Name: "no-op",
			Fn: func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor {
				return nil // skip
			},
		},
	}

	err := InvokePlugins(root, &PluginInfo{}, configs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInvokePlugins_UnknownBuiltin(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	configs := []PluginConfig{
		{Name: "nonexistent-plugin"},
	}

	root := &svgast.Root{}
	err := InvokePlugins(root, &PluginInfo{}, configs, nil)
	if err == nil {
		t.Error("expected error for unknown plugin")
	}
}

func TestInvokePlugins_ParamsMerge(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	var receivedParams map[string]any

	Register(&Plugin{
		Name: "param-test",
		Fn: func(_ *svgast.Root, params map[string]any, _ *PluginInfo) *svgast.Visitor {
			receivedParams = params
			return nil
		},
	})

	configs := []PluginConfig{
		{
			Name:   "param-test",
			Params: map[string]any{"a": 1, "b": 2},
		},
	}

	globalOverrides := map[string]any{"b": 99, "c": 3}

	root := &svgast.Root{}
	err := InvokePlugins(root, &PluginInfo{}, configs, globalOverrides)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Global overrides should override plugin params
	if receivedParams["a"] != 1 {
		t.Errorf("param 'a' = %v, want 1", receivedParams["a"])
	}
	if receivedParams["b"] != 99 {
		t.Errorf("param 'b' = %v, want 99 (global override)", receivedParams["b"])
	}
	if receivedParams["c"] != 3 {
		t.Errorf("param 'c' = %v, want 3", receivedParams["c"])
	}
}

func TestInvokePlugins_PluginInfo(t *testing.T) {
	var receivedInfo *PluginInfo

	configs := []PluginConfig{
		{
			Name: "info-test",
			Fn: func(_ *svgast.Root, _ map[string]any, info *PluginInfo) *svgast.Visitor {
				receivedInfo = info
				return nil
			},
		},
	}

	info := &PluginInfo{Path: "/test.svg", MultipassCount: 3}
	root := &svgast.Root{}
	_ = InvokePlugins(root, info, configs, nil)

	if receivedInfo.Path != "/test.svg" {
		t.Errorf("Path = %q, want %q", receivedInfo.Path, "/test.svg")
	}
	if receivedInfo.MultipassCount != 3 {
		t.Errorf("MultipassCount = %d, want 3", receivedInfo.MultipassCount)
	}
}

func TestInvokePlugins_SequentialExecution(t *testing.T) {
	var order []string

	configs := []PluginConfig{
		{
			Name: "first",
			Fn: func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor {
				order = append(order, "first")
				return nil
			},
		},
		{
			Name: "second",
			Fn: func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor {
				order = append(order, "second")
				return nil
			},
		},
		{
			Name: "third",
			Fn: func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor {
				order = append(order, "third")
				return nil
			},
		},
	}

	root := &svgast.Root{}
	_ = InvokePlugins(root, &PluginInfo{}, configs, nil)

	if len(order) != 3 || order[0] != "first" || order[1] != "second" || order[2] != "third" {
		t.Errorf("execution order = %v, want [first second third]", order)
	}
}

// --- Preset Tests ---

func TestCreatePreset_Basic(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	var executed []string

	p1 := &Plugin{
		Name: "plugin-a",
		Fn: func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor {
			executed = append(executed, "a")
			return nil
		},
	}
	p2 := &Plugin{
		Name: "plugin-b",
		Fn: func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor {
			executed = append(executed, "b")
			return nil
		},
	}

	preset := CreatePreset("test-preset", []*Plugin{p1, p2})

	root := &svgast.Root{}
	preset.Fn(root, map[string]any{}, &PluginInfo{})

	if len(executed) != 2 || executed[0] != "a" || executed[1] != "b" {
		t.Errorf("preset executed = %v, want [a b]", executed)
	}
}

func TestCreatePreset_WithOverrides(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	var executed []string
	var paramsB map[string]any

	p1 := &Plugin{
		Name: "plugin-a",
		Fn: func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor {
			executed = append(executed, "a")
			return nil
		},
	}
	p2 := &Plugin{
		Name: "plugin-b",
		Fn: func(_ *svgast.Root, params map[string]any, _ *PluginInfo) *svgast.Visitor {
			executed = append(executed, "b")
			paramsB = params
			return nil
		},
	}
	p3 := &Plugin{
		Name: "plugin-c",
		Fn: func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor {
			executed = append(executed, "c")
			return nil
		},
	}

	preset := CreatePreset("test-preset", []*Plugin{p1, p2, p3})

	root := &svgast.Root{}
	preset.Fn(root, map[string]any{
		"overrides": map[string]any{
			"plugin-a": false,                        // disable plugin-a
			"plugin-b": map[string]any{"key": "val"}, // override params for plugin-b
		},
	}, &PluginInfo{})

	// plugin-a should be skipped
	if len(executed) != 2 || executed[0] != "b" || executed[1] != "c" {
		t.Errorf("preset executed = %v, want [b c]", executed)
	}

	// plugin-b should receive override params
	if paramsB["key"] != "val" {
		t.Errorf("plugin-b params = %v, want key=val", paramsB)
	}
}

func TestCreatePreset_FloatPrecision(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	var receivedParams map[string]any

	p := &Plugin{
		Name: "plugin-x",
		Fn: func(_ *svgast.Root, params map[string]any, _ *PluginInfo) *svgast.Visitor {
			receivedParams = params
			return nil
		},
	}

	preset := CreatePreset("test-preset", []*Plugin{p})

	root := &svgast.Root{}
	preset.Fn(root, map[string]any{
		"floatPrecision": 3,
	}, &PluginInfo{})

	if receivedParams["floatPrecision"] != 3 {
		t.Errorf("floatPrecision = %v, want 3", receivedParams["floatPrecision"])
	}
}

func TestRegisterPresetDefault_NoPluginsRegistered(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	// When no individual plugins are registered, preset-default should still work
	// (just with an empty plugin list)
	preset := RegisterPresetDefault()
	if preset.Name != "preset-default" {
		t.Errorf("Name = %q, want %q", preset.Name, "preset-default")
	}
	if !preset.IsPreset {
		t.Error("IsPreset should be true")
	}

	// Should be registered
	if !Has("preset-default") {
		t.Error("preset-default should be registered")
	}
}

func TestResolvePluginConfig_Builtin(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	Register(&Plugin{
		Name: "test-plugin",
		Fn: func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor {
			return nil
		},
	})

	resolved, err := resolvePluginConfig(PluginConfig{Name: "test-plugin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Name != "test-plugin" {
		t.Errorf("Name = %q, want %q", resolved.Name, "test-plugin")
	}
	if resolved.Fn == nil {
		t.Error("Fn should not be nil")
	}
}

func TestResolvePluginConfig_CustomFn(t *testing.T) {
	customFn := func(_ *svgast.Root, _ map[string]any, _ *PluginInfo) *svgast.Visitor {
		return nil
	}

	resolved, err := resolvePluginConfig(PluginConfig{
		Name:   "custom",
		Fn:     customFn,
		Params: map[string]any{"x": 1},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Name != "custom" {
		t.Errorf("Name = %q", resolved.Name)
	}
	if resolved.Params["x"] != 1 {
		t.Errorf("Params = %v", resolved.Params)
	}
}

func TestResolvePluginConfig_Unknown(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	_, err := resolvePluginConfig(PluginConfig{Name: "no-such-plugin"})
	if err == nil {
		t.Error("expected error for unknown plugin")
	}
}

// --- PresetDefaultPluginNames Test ---

func TestPresetDefaultPluginNames_Count(t *testing.T) {
	if len(PresetDefaultPluginNames) != 34 {
		t.Errorf("PresetDefaultPluginNames has %d entries, want 34", len(PresetDefaultPluginNames))
	}
}

func TestPresetDefaultPluginNames_Order(t *testing.T) {
	// Verify first and last entries match SVGO's preset-default
	if PresetDefaultPluginNames[0] != "removeDoctype" {
		t.Errorf("first plugin = %q, want %q", PresetDefaultPluginNames[0], "removeDoctype")
	}
	if PresetDefaultPluginNames[33] != "removeDesc" {
		t.Errorf("last plugin = %q, want %q", PresetDefaultPluginNames[33], "removeDesc")
	}
}
