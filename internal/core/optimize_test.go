// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package core

import (
	"testing"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func TestOptimize_BasicPassthrough(t *testing.T) {
	// With no plugins, optimize should be a parse→stringify passthrough
	input := `<svg xmlns="http://www.w3.org/2000/svg"><g/></svg>`
	output, err := Optimize(input, &Config{
		Plugins: []plugin.PluginConfig{}, // empty plugin list = no-op
	})
	if err != nil {
		t.Fatalf("Optimize error: %v", err)
	}
	if output.Data != input {
		t.Errorf("got:\n%q\nwant:\n%q", output.Data, input)
	}
}

func TestOptimize_PrettyPrint(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg"><g/></svg>`
	output, err := Optimize(input, &Config{
		Plugins: []plugin.PluginConfig{}, // empty plugin list = no-op
		Js2svg: &svgast.StringifyOptions{
			Pretty:       true,
			Indent:       4,
			UseShortTags: true,
			EOL:          "lf",
		},
	})
	if err != nil {
		t.Fatalf("Optimize error: %v", err)
	}

	expected := "<svg xmlns=\"http://www.w3.org/2000/svg\">\n    <g/>\n</svg>\n"
	if output.Data != expected {
		t.Errorf("got:\n%q\nwant:\n%q", output.Data, expected)
	}
}

func TestOptimize_WithCustomPlugin(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg"><!--remove me--><g/></svg>`
	expected := `<svg xmlns="http://www.w3.org/2000/svg"><g/></svg>`

	output, err := Optimize(input, &Config{
		Plugins: []plugin.PluginConfig{
			{
				Name: "test-remove-comments",
				Fn: func(root *svgast.Root, params map[string]any, info *plugin.PluginInfo) *svgast.Visitor {
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
		},
	})
	if err != nil {
		t.Fatalf("Optimize error: %v", err)
	}
	if output.Data != expected {
		t.Errorf("got:\n%q\nwant:\n%q", output.Data, expected)
	}
}
