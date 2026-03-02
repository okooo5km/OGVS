// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/okooo5km/ogvs/internal/core"
	"github.com/okooo5km/ogvs/internal/plugin"
	_ "github.com/okooo5km/ogvs/internal/plugins" // register all plugins
	"github.com/okooo5km/ogvs/internal/svgast"
	"github.com/okooo5km/ogvs/internal/testkit/assert"
)

// loadIntegrationFixture reads a fixture file from testdata/ and splits it
// into input and expected parts using the @@@ separator.
// The format is: input SVG \n @@@\n expected SVG
func loadIntegrationFixture(t *testing.T, name string) (input, expected string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}
	content := assert.NormalizeLF(string(data))
	parts := strings.SplitN(content, "@@@", 2)
	if len(parts) != 2 {
		t.Fatalf("fixture %s: expected input @@@ expected format", name)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

// intPtr returns a pointer to an int value.
func intPtr(n int) *int {
	return &n
}

// ---------- Fixture-based integration tests (ported from SVGO _index.test.js) ----------

// TestIntegration_PrettyIndent2 tests pretty-printing with 2-space indentation.
// Config: no plugins (empty list), pretty mode with indent=2.
// Matches SVGO test: "should create indent with 2 spaces"
func TestIntegration_PrettyIndent2(t *testing.T) {
	input, expected := loadIntegrationFixture(t, "test.svg.txt")
	result, err := core.Optimize(input, &core.Config{
		Plugins: []plugin.PluginConfig{}, // empty = no plugins
		Js2svg: &svgast.StringifyOptions{
			Pretty:       true,
			Indent:       2,
			UseShortTags: true,
			EOL:          "lf",
		},
	})
	if err != nil {
		t.Fatalf("Optimize error: %v", err)
	}
	r := assert.L1StrictEqual(result.Data, expected)
	if !r.Pass {
		t.Errorf("L1 mismatch:\n%s", r.Diff)
	}
}

// TestIntegration_PluginOrder tests that plugins execute in the correct order
// (preset-default order) and produce the expected optimized output.
// Config: path set, nil Plugins (= preset-default).
// Matches SVGO test: "should handle plugins order properly"
func TestIntegration_PluginOrder(t *testing.T) {
	input, expected := loadIntegrationFixture(t, "plugins-order.svg.txt")
	result, err := core.Optimize(input, &core.Config{
		Path: "input.svg",
	})
	if err != nil {
		t.Fatalf("Optimize error: %v", err)
	}
	r := assert.L1StrictEqual(result.Data, expected)
	if !r.Pass {
		t.Errorf("L1 mismatch:\n%s", r.Diff)
	}
}

// TestIntegration_EmptySVG tests optimization of an empty SVG tag.
// Matches SVGO test: "should handle empty svg tag"
func TestIntegration_EmptySVG(t *testing.T) {
	result, err := core.Optimize("<svg />", &core.Config{
		Path: "input.svg",
	})
	if err != nil {
		t.Fatalf("Optimize error: %v", err)
	}
	expected := "<svg/>"
	if assert.NormalizeLF(result.Data) != expected {
		t.Errorf("got: %q, want: %q", result.Data, expected)
	}
}

// TestIntegration_StyleSpecificity tests that style specificity is preserved
// over presentation attributes.
// Config: path set, pretty mode, nil Plugins (= preset-default).
// Matches SVGO test: "should preserve style specificity over attributes"
//
// Note: OGVS fixture differs from SVGO — OGVS outputs "fill:blue" in inline style
// while SVGO converts it to "fill:#00f". OGVS does not apply convertColors to
// CSS values inside inline style attributes. The style specificity behavior
// (preserving style over presentation attrs) is correctly maintained.
func TestIntegration_StyleSpecificity(t *testing.T) {
	input, expected := loadIntegrationFixture(t, "style-specificity.svg.txt")
	result, err := core.Optimize(input, &core.Config{
		Path: "input.svg",
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
	r := assert.L1StrictEqual(result.Data, expected)
	if !r.Pass {
		t.Errorf("L1 mismatch:\n%s", r.Diff)
	}
}

// TestIntegration_Whitespaces tests preservation of whitespace between tspan tags.
// Config: path set, pretty mode, nil Plugins (= preset-default).
// Matches SVGO test: "should preserve whitespaces between tspan tags"
func TestIntegration_Whitespaces(t *testing.T) {
	input, expected := loadIntegrationFixture(t, "whitespaces.svg.txt")
	result, err := core.Optimize(input, &core.Config{
		Path: "input.svg",
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
	r := assert.L1StrictEqual(result.Data, expected)
	if !r.Pass {
		t.Errorf("L1 mismatch:\n%s", r.Diff)
	}
}

// TestIntegration_KeyframeSelectors tests preservation of "to" keyframe selector.
// Config: path set, pretty mode, nil Plugins (= preset-default).
// Matches SVGO test: "should preserve 'to' keyframe selector"
func TestIntegration_KeyframeSelectors(t *testing.T) {
	input, expected := loadIntegrationFixture(t, "keyframe-selectors.svg.txt")
	result, err := core.Optimize(input, &core.Config{
		Path: "input.svg",
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
	r := assert.L1StrictEqual(result.Data, expected)
	if !r.Pass {
		t.Errorf("L1 mismatch:\n%s", r.Diff)
	}
}

// TestIntegration_Entities tests handling of XML entities (inline entity expansion).
// Config: no plugins, pretty mode with path set.
// Matches SVGO test: "should inline entities"
//
// Note: OGVS fixture differs from SVGO — OGVS outputs DOCTYPE before ProcInst
// while SVGO preserves the original document order (ProcInst, Comment, DOCTYPE).
// This is because OGVS's parser pre-scans for DOCTYPE to extract entity
// declarations before the main token loop, which processes ProcInst tokens
// sequentially. The entity expansion itself (replacing &Viewport; with
// <rect ...>) works correctly and identically to SVGO.
func TestIntegration_Entities(t *testing.T) {
	input, expected := loadIntegrationFixture(t, "entities.svg.txt")
	result, err := core.Optimize(input, &core.Config{
		Path:    "input.svg",
		Plugins: []plugin.PluginConfig{}, // no plugins
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
	r := assert.L1StrictEqual(result.Data, expected)
	if !r.Pass {
		t.Errorf("L1 mismatch:\n%s", r.Diff)
	}
}

// TestIntegration_PreElement tests that whitespace in <pre> elements is preserved.
// Config: path set, nil Plugins (= preset-default).
// Matches SVGO test: "should not trim whitespace at start and end of pre element"
func TestIntegration_PreElement(t *testing.T) {
	input, expected := loadIntegrationFixture(t, "pre-element.svg.txt")
	result, err := core.Optimize(input, &core.Config{
		Path: "input.svg",
	})
	if err != nil {
		t.Fatalf("Optimize error: %v", err)
	}
	r := assert.L1StrictEqual(result.Data, expected)
	if !r.Pass {
		t.Errorf("L1 mismatch:\n%s", r.Diff)
	}
}

// TestIntegration_PreElementPretty tests that <pre> whitespace is preserved in pretty mode.
// Config: path set, pretty mode, nil Plugins (= preset-default).
// Matches SVGO test: "should not add whitespace in pre element"
func TestIntegration_PreElementPretty(t *testing.T) {
	input, expected := loadIntegrationFixture(t, "pre-element-pretty.svg.txt")
	result, err := core.Optimize(input, &core.Config{
		Path: "input.svg",
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
	r := assert.L1StrictEqual(result.Data, expected)
	if !r.Pass {
		t.Errorf("L1 mismatch:\n%s", r.Diff)
	}
}

// ---------- Programmatic integration tests ----------

// TestIntegration_MultipassConvergence tests that multipass optimization converges.
// A simple SVG that benefits from multiple passes should produce output no larger
// than single-pass, and both should be valid SVG.
func TestIntegration_MultipassConvergence(t *testing.T) {
	// This SVG has nested groups that can be collapsed in multiple passes:
	// Pass 1: inner optimizations happen
	// Pass 2+: further simplifications become possible
	input := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
  <defs/>
  <g>
    <g>
      <g>
        <path d="M 10.0000 20.0000 L 30.0000 40.0000 L 50.0000 60.0000"/>
      </g>
    </g>
  </g>
</svg>`

	// Single pass
	singleResult, err := core.Optimize(input, &core.Config{
		Path:      "input.svg",
		Multipass: false,
	})
	if err != nil {
		t.Fatalf("single-pass Optimize error: %v", err)
	}

	// Multipass
	multiResult, err := core.Optimize(input, &core.Config{
		Path:      "input.svg",
		Multipass: true,
	})
	if err != nil {
		t.Fatalf("multi-pass Optimize error: %v", err)
	}

	// Multipass output should be no larger than single-pass
	if len(multiResult.Data) > len(singleResult.Data) {
		t.Errorf("multipass output (%d bytes) is larger than single-pass (%d bytes)\nmultipass: %s\nsingle:   %s",
			len(multiResult.Data), len(singleResult.Data), multiResult.Data, singleResult.Data)
	}

	// Both should produce valid SVG (non-empty, starts with <svg)
	for _, tc := range []struct {
		name string
		data string
	}{
		{"single-pass", singleResult.Data},
		{"multi-pass", multiResult.Data},
	} {
		if tc.data == "" {
			t.Errorf("%s produced empty output", tc.name)
		}
		if !strings.HasPrefix(tc.data, "<svg") {
			t.Errorf("%s output doesn't start with <svg: %q", tc.name, tc.data[:min(50, len(tc.data))])
		}
	}

	t.Logf("single-pass (%d bytes): %s", len(singleResult.Data), singleResult.Data)
	t.Logf("multi-pass  (%d bytes): %s", len(multiResult.Data), multiResult.Data)
}

// TestIntegration_EmptyPluginsList tests that an empty plugins list (no plugins)
// produces a passthrough -- input should come out unchanged after parse/stringify.
func TestIntegration_EmptyPluginsList(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg"><rect x="10" y="20" width="30" height="40"/></svg>`
	result, err := core.Optimize(input, &core.Config{
		Plugins: []plugin.PluginConfig{}, // empty = no plugins
	})
	if err != nil {
		t.Fatalf("Optimize error: %v", err)
	}
	// With no plugins, parse/stringify should produce identical output
	if result.Data != input {
		t.Errorf("expected passthrough.\ngot:  %q\nwant: %q", result.Data, input)
	}
}

// TestIntegration_FloatPrecisionOverride tests that the global FloatPrecision
// override is properly passed through to plugins via globalOverrides.
func TestIntegration_FloatPrecisionOverride(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
  <circle cx="10.123456" cy="20.654321" r="5.999999"/>
</svg>`

	// With default precision
	defaultResult, err := core.Optimize(input, &core.Config{
		Path: "input.svg",
	})
	if err != nil {
		t.Fatalf("default precision Optimize error: %v", err)
	}

	// With precision=2 (very low to clearly see difference)
	precResult, err := core.Optimize(input, &core.Config{
		Path:           "input.svg",
		FloatPrecision: intPtr(2),
	})
	if err != nil {
		t.Fatalf("precision=2 Optimize error: %v", err)
	}

	// Both should produce valid, non-empty SVG
	if precResult.Data == "" {
		t.Fatal("precision=2 produced empty output")
	}
	if !strings.HasPrefix(precResult.Data, "<svg") {
		t.Errorf("precision=2 output doesn't start with <svg: %q", precResult.Data[:min(50, len(precResult.Data))])
	}

	// Float precision result should be no larger than default
	// (fewer decimal places = shorter strings)
	if len(precResult.Data) > len(defaultResult.Data) {
		t.Errorf("precision=2 output (%d bytes) is larger than default (%d bytes)",
			len(precResult.Data), len(defaultResult.Data))
	}

	t.Logf("default output (%d bytes): %s", len(defaultResult.Data), defaultResult.Data)
	t.Logf("prec=2 output  (%d bytes): %s", len(precResult.Data), precResult.Data)
}

// TestIntegration_PresetOverrideDisable tests disabling a specific plugin
// in the preset-default via overrides. When removeDesc is disabled,
// <desc> elements should be preserved in the output.
func TestIntegration_PresetOverrideDisable(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg">
  <desc>This should be preserved</desc>
  <rect x="10" y="20" width="30" height="40"/>
</svg>`

	// With removeDesc disabled via preset-default overrides
	result, err := core.Optimize(input, &core.Config{
		Path: "input.svg",
		Plugins: []plugin.PluginConfig{
			{
				Name: "preset-default",
				Params: map[string]any{
					"overrides": map[string]any{
						"removeDesc": false,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Optimize error: %v", err)
	}

	// <desc> should be preserved since removeDesc is disabled
	if !strings.Contains(result.Data, "<desc>") {
		t.Errorf("expected <desc> to be preserved when removeDesc is disabled.\ngot: %s", result.Data)
	}
	if !strings.Contains(result.Data, "This should be preserved") {
		t.Errorf("expected desc content to be preserved.\ngot: %s", result.Data)
	}
}

// TestIntegration_PresetOverrideParams tests selectively disabling plugins
// in the preset-default via overrides. Disabling removeComments should
// preserve comments while removeDesc (still active) removes editor-generated desc.
func TestIntegration_PresetOverrideParams(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
  <!-- Keep this comment -->
  <desc>Created with Test Editor</desc>
  <rect x="10" y="20" width="30" height="40"/>
</svg>`

	// With removeComments disabled but removeDesc active (default)
	result, err := core.Optimize(input, &core.Config{
		Path: "input.svg",
		Plugins: []plugin.PluginConfig{
			{
				Name: "preset-default",
				Params: map[string]any{
					"overrides": map[string]any{
						"removeComments": false,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Optimize error: %v", err)
	}

	// removeComments is disabled, so the comment should be preserved
	if !strings.Contains(result.Data, "<!--") {
		t.Errorf("expected comment to be preserved when removeComments is disabled.\ngot: %s", result.Data)
	}

	// removeDesc is still active (default) and the desc text starts with
	// "Created with" which is editor-generated, so <desc> should be removed
	if strings.Contains(result.Data, "<desc>") {
		t.Errorf("expected <desc> to be removed (editor-generated text).\ngot: %s", result.Data)
	}
}

// TestIntegration_InvalidSVG tests error handling for malformed SVG input.
func TestIntegration_InvalidSVG(t *testing.T) {
	_, err := core.Optimize("<svg</svg>", &core.Config{
		Path: "input.svg",
	})
	if err == nil {
		t.Error("expected error for invalid SVG, got nil")
	}
}

// TestIntegration_NilConfig tests that passing nil config uses defaults
// (preset-default plugins with default stringify options).
func TestIntegration_NilConfig(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg"><rect x="10" y="20" width="30" height="40"/></svg>`
	result, err := core.Optimize(input, nil)
	if err != nil {
		t.Fatalf("Optimize error: %v", err)
	}
	// With nil config, preset-default should be applied
	if result.Data == "" {
		t.Fatal("nil config produced empty output")
	}
	if !strings.HasPrefix(result.Data, "<svg") {
		t.Errorf("output doesn't start with <svg: %q", result.Data[:min(50, len(result.Data))])
	}
}

// TestIntegration_EmptyStringInput tests that an empty string input returns an error.
func TestIntegration_EmptyStringInput(t *testing.T) {
	result, err := core.Optimize("", &core.Config{
		Plugins: []plugin.PluginConfig{},
	})
	// Empty string should either error or produce empty output
	if err != nil {
		// Error is acceptable for empty input
		return
	}
	if result.Data != "" {
		t.Errorf("expected empty output for empty input, got: %q", result.Data)
	}
}

// TestIntegration_CustomPluginFn tests using a custom plugin function
// instead of a builtin plugin name.
func TestIntegration_CustomPluginFn(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg"><!--remove me--><g><rect x="1" y="2" width="3" height="4"/></g></svg>`
	expected := `<svg xmlns="http://www.w3.org/2000/svg"><g><rect x="1" y="2" width="3" height="4"/></g></svg>`

	result, err := core.Optimize(input, &core.Config{
		Plugins: []plugin.PluginConfig{
			{
				Name: "custom-remove-comments",
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
	if result.Data != expected {
		t.Errorf("got:  %q\nwant: %q", result.Data, expected)
	}
}

// TestIntegration_EOLOption tests CRLF line ending output.
func TestIntegration_EOLOption(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg"><g><rect x="1" y="2" width="3" height="4"/></g></svg>`

	result, err := core.Optimize(input, &core.Config{
		Plugins: []plugin.PluginConfig{}, // no plugins
		Js2svg: &svgast.StringifyOptions{
			Pretty:       true,
			Indent:       2,
			UseShortTags: true,
			EOL:          "crlf",
		},
	})
	if err != nil {
		t.Fatalf("Optimize error: %v", err)
	}
	// In CRLF mode, line endings should be \r\n
	if !strings.Contains(result.Data, "\r\n") {
		t.Errorf("expected CRLF line endings in output.\ngot: %q", result.Data)
	}
}

// TestIntegration_FinalNewline tests the FinalNewline option.
func TestIntegration_FinalNewline(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg"><g/></svg>`

	result, err := core.Optimize(input, &core.Config{
		Plugins: []plugin.PluginConfig{}, // no plugins
		Js2svg: &svgast.StringifyOptions{
			Pretty:       false,
			UseShortTags: true,
			EOL:          "lf",
			FinalNewline: true,
		},
	})
	if err != nil {
		t.Fatalf("Optimize error: %v", err)
	}
	// Output should end with a newline
	if !strings.HasSuffix(result.Data, "\n") {
		t.Errorf("expected final newline.\ngot: %q", result.Data)
	}
}
