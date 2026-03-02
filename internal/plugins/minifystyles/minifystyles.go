// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package minifystyles minifies <style> elements and style attributes using CSS minification.
package minifystyles

import (
	"regexp"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
	"github.com/okooo5km/ogvs/internal/tools"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "minifyStyles",
		Description: "minifies styles and removes unused styles",
		Fn:          fn,
	})
}

// usageConfig controls which types of unused selectors to remove.
type usageConfig struct {
	force   bool // override deoptimization (remove unused even with scripts)
	ids     bool // remove unused ID selectors
	classes bool // remove unused class selectors
	tags    bool // remove unused tag selectors
}

func fn(root *svgast.Root, params map[string]any, info *plugin.PluginInfo) *svgast.Visitor {
	type styleEntry struct {
		node   *svgast.Element
		parent svgast.Parent
	}

	var styleElements []styleEntry
	var elementsWithStyleAttr []*svgast.Element
	deoptimized := false

	// Parse usage config
	usageEnabled := true // whether to remove unused selectors at all
	usage := usageConfig{
		force:   false,
		ids:     true,
		classes: true,
		tags:    true,
	}
	if v, ok := params["usage"].(bool); ok {
		usageEnabled = v
	} else if v, ok := params["usage"].(map[string]any); ok {
		if f, ok := v["force"].(bool); ok {
			usage.force = f
		}
		if f, ok := v["ids"].(bool); ok {
			usage.ids = f
		}
		if f, ok := v["classes"].(bool); ok {
			usage.classes = f
		}
		if f, ok := v["tags"].(bool); ok {
			usage.tags = f
		}
	}

	// Collect used tags, classes, IDs
	usedTags := make(map[string]bool)
	usedClasses := make(map[string]bool)
	usedIDs := make(map[string]bool)

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem, ok := node.(*svgast.Element)
				if !ok {
					return nil
				}

				// detect scripts that deoptimize processing
				if tools.HasScripts(elem) {
					deoptimized = true
				}

				// collect style elements
				if elem.Name == "style" && len(elem.Children) > 0 {
					styleElements = append(styleElements, styleEntry{node: elem, parent: parent})
				} else {
					if elem.Attributes.Has("style") {
						elementsWithStyleAttr = append(elementsWithStyleAttr, elem)
					}

					// collect usage data
					usedTags[elem.Name] = true
					if classAttr, has := elem.Attributes.Get("class"); has {
						for _, cls := range strings.Fields(classAttr) {
							usedClasses[cls] = true
						}
					}
					if idAttr, has := elem.Attributes.Get("id"); has && idAttr != "" {
						usedIDs[idAttr] = true
					}
				}

				return nil
			},
		},
		Root: &svgast.VisitorCallbacks{
			Exit: func(node svgast.Node, parent svgast.Parent) {
				m := minify.New()
				m.AddFunc("text/css", css.Minify)

				// Determine if we should remove unused selectors
				removeUnused := usageEnabled && (!deoptimized || usage.force)

				// minify style elements
				for _, entry := range styleElements {
					child := entry.node.Children[0]
					var cssText string
					var isCdata bool

					switch c := child.(type) {
					case *svgast.Text:
						cssText = c.Value
					case *svgast.Cdata:
						cssText = c.Value
						isCdata = true
					default:
						continue
					}

					// First minify to normalize
					minified, err := m.String("text/css", cssText)
					if err != nil {
						continue
					}

					// Collapse longhand properties into shorthand
					minified = collapseLonghandsInCSS(minified)

					// Then remove unused selectors
					if removeUnused {
						minified = removeUnusedRules(minified, usedTags, usedClasses, usedIDs, usage)
					}

					if minified == "" {
						svgast.DetachNodeFromParent(entry.node, entry.parent)
						continue
					}

					// preserve cdata if CSS contains < or >
					if strings.ContainsAny(minified, "<>") {
						entry.node.Children[0] = &svgast.Cdata{Value: minified}
					} else if isCdata {
						entry.node.Children[0] = &svgast.Text{Value: minified}
					} else {
						entry.node.Children[0] = &svgast.Text{Value: minified}
					}
				}

				// minify style attributes
				for _, elem := range elementsWithStyleAttr {
					styleVal, _ := elem.Attributes.Get("style")
					minified, err := m.String("text/css", "{"+styleVal+"}")
					if err != nil {
						continue
					}
					// Remove the surrounding braces we added
					minified = strings.TrimPrefix(minified, "{")
					minified = strings.TrimSuffix(minified, "}")
					// Collapse longhand properties into shorthand
					minified = collapseDeclarations(minified)
					elem.Attributes.Set("style", minified)
				}
			},
		},
	}
}

// rulePattern matches CSS rules: selector { declarations }
// This handles nested @-rules by matching balanced braces.
var rulePattern = regexp.MustCompile(`([^{}]+)\{([^{}]*)\}`)

// removeUnusedRules removes CSS rules whose selectors don't match any used elements.
func removeUnusedRules(cssText string, usedTags, usedClasses, usedIDs map[string]bool, usage usageConfig) string {
	var result strings.Builder
	remaining := cssText

	for len(remaining) > 0 {
		// Find next rule or @-rule
		if strings.HasPrefix(remaining, "@") {
			// @-rule: find matching braces
			atRule, rest := extractAtRule(remaining)
			result.WriteString(atRule)
			remaining = rest
			continue
		}

		// Find next rule: selector { declarations }
		loc := rulePattern.FindStringIndex(remaining)
		if loc == nil {
			result.WriteString(remaining)
			break
		}

		// Include any text before the match
		if loc[0] > 0 {
			result.WriteString(remaining[:loc[0]])
		}

		match := remaining[loc[0]:loc[1]]
		braceIdx := strings.Index(match, "{")
		selector := strings.TrimSpace(match[:braceIdx])
		declarations := match[braceIdx:]

		if isSelectorUsed(selector, usedTags, usedClasses, usedIDs, usage) {
			result.WriteString(selector)
			result.WriteString(declarations)
		}

		remaining = remaining[loc[1]:]
	}

	return result.String()
}

// extractAtRule extracts an @-rule with its balanced braces.
func extractAtRule(s string) (string, string) {
	depth := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '{' {
			depth++
		} else if s[i] == '}' {
			depth--
			if depth == 0 {
				return s[:i+1], s[i+1:]
			}
		}
	}
	return s, ""
}

// isSelectorUsed checks if a CSS selector matches any used element.
func isSelectorUsed(selector string, usedTags, usedClasses, usedIDs map[string]bool, usage usageConfig) bool {
	// Handle comma-separated selectors: each must be checked
	parts := strings.Split(selector, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if isSingleSelectorUsed(part, usedTags, usedClasses, usedIDs, usage) {
			return true
		}
	}
	return false
}

// isSingleSelectorUsed checks if a single (non-comma) selector is used.
func isSingleSelectorUsed(selector string, usedTags, usedClasses, usedIDs map[string]bool, usage usageConfig) bool {
	// Check for class selectors: .className
	if usage.classes {
		classRe := regexp.MustCompile(`\.([a-zA-Z_-][\w-]*)`)
		matches := classRe.FindAllStringSubmatch(selector, -1)
		for _, m := range matches {
			if !usedClasses[m[1]] {
				return false
			}
		}
	}

	// Check for ID selectors: #idName
	if usage.ids {
		idRe := regexp.MustCompile(`#([a-zA-Z_-][\w-]*)`)
		matches := idRe.FindAllStringSubmatch(selector, -1)
		for _, m := range matches {
			if !usedIDs[m[1]] {
				return false
			}
		}
	}

	// Check for tag selectors (simple element names)
	if usage.tags {
		// Extract tag names: tokens that aren't preceded by . or # and aren't pseudo-classes
		tagRe := regexp.MustCompile(`(?:^|[\s>+~])([a-zA-Z][\w-]*)`)
		matches := tagRe.FindAllStringSubmatch(selector, -1)
		for _, m := range matches {
			tagName := m[1]
			if !usedTags[tagName] {
				return false
			}
		}
	}

	return true
}

// shorthandGroup defines a CSS shorthand property and its longhands in top/right/bottom/left order.
type shorthandGroup struct {
	shorthand string
	longhands [4]string
}

var boxShorthands = []shorthandGroup{
	{"padding", [4]string{"padding-top", "padding-right", "padding-bottom", "padding-left"}},
	{"margin", [4]string{"margin-top", "margin-right", "margin-bottom", "margin-left"}},
	{"border-width", [4]string{"border-top-width", "border-right-width", "border-bottom-width", "border-left-width"}},
	{"border-style", [4]string{"border-top-style", "border-right-style", "border-bottom-style", "border-left-style"}},
	{"border-color", [4]string{"border-top-color", "border-right-color", "border-bottom-color", "border-left-color"}},
}

// collapseLonghandsInCSS collapses longhand properties into shorthand
// within declaration blocks in minified CSS text.
func collapseLonghandsInCSS(css string) string {
	var result strings.Builder
	i := 0
	for i < len(css) {
		braceIdx := strings.IndexByte(css[i:], '{')
		if braceIdx < 0 {
			result.WriteString(css[i:])
			break
		}
		// Write everything up to and including '{'
		result.WriteString(css[i : i+braceIdx+1])
		i = i + braceIdx + 1

		// Find matching '}' tracking depth
		start := i
		depth := 1
		for i < len(css) && depth > 0 {
			if css[i] == '{' {
				depth++
			} else if css[i] == '}' {
				depth--
			}
			if depth > 0 {
				i++
			}
		}

		content := css[start:i]

		// Check if content has nested braces (it's a container like @media, not declarations)
		if strings.ContainsAny(content, "{}") {
			// Recursively process nested content
			result.WriteString(collapseLonghandsInCSS(content))
		} else {
			// It's a declaration block, collapse longhands
			result.WriteString(collapseDeclarations(content))
		}

		// Write the closing '}'
		if i < len(css) {
			result.WriteByte('}')
			i++
		}
	}
	return result.String()
}

// collapseDeclarations collapses longhand properties into shorthand
// within a semicolon-separated declaration list.
func collapseDeclarations(decls string) string {
	type decl struct {
		prop string
		val  string
	}

	// Parse declarations preserving order
	parts := strings.Split(decls, ";")
	var declList []decl
	declIdx := make(map[string]int) // prop → index in declList

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		colonIdx := strings.IndexByte(part, ':')
		if colonIdx < 0 {
			declList = append(declList, decl{part, ""})
			continue
		}
		prop := part[:colonIdx]
		val := part[colonIdx+1:]
		declIdx[prop] = len(declList)
		declList = append(declList, decl{prop, val})
	}

	// Try collapsing each shorthand group
	for _, group := range boxShorthands {
		values := [4]string{}
		allPresent := true
		for i, longhand := range group.longhands {
			if idx, ok := declIdx[longhand]; ok {
				values[i] = declList[idx].val
			} else {
				allPresent = false
				break
			}
		}
		if !allPresent {
			continue
		}

		// Don't collapse if any value contains !important (complex case)
		hasImportant := false
		for _, v := range values {
			if strings.Contains(v, "!important") {
				hasImportant = true
				break
			}
		}
		if hasImportant {
			continue
		}

		// Determine shorthand value
		var shortVal string
		if values[0] == values[1] && values[1] == values[2] && values[2] == values[3] {
			shortVal = values[0]
		} else if values[0] == values[2] && values[1] == values[3] {
			shortVal = values[0] + " " + values[1]
		} else if values[1] == values[3] {
			shortVal = values[0] + " " + values[1] + " " + values[2]
		} else {
			shortVal = values[0] + " " + values[1] + " " + values[2] + " " + values[3]
		}

		// Replace first longhand with shorthand, mark rest for removal
		firstIdx := -1
		removeSet := make(map[int]bool)
		for _, longhand := range group.longhands {
			idx := declIdx[longhand]
			if firstIdx < 0 {
				firstIdx = idx
			} else {
				removeSet[idx] = true
			}
			delete(declIdx, longhand)
		}
		declList[firstIdx] = decl{group.shorthand, shortVal}
		declIdx[group.shorthand] = firstIdx

		// Rebuild list without removed entries
		var newList []decl
		newIdx := make(map[string]int)
		for i, d := range declList {
			if removeSet[i] {
				continue
			}
			newIdx[d.prop] = len(newList)
			newList = append(newList, d)
		}
		declList = newList
		declIdx = newIdx
	}

	// Rebuild declaration string
	var result []string
	for _, d := range declList {
		if d.val == "" {
			result = append(result, d.prop)
		} else {
			result = append(result, d.prop+":"+d.val)
		}
	}
	return strings.Join(result, ";")
}
