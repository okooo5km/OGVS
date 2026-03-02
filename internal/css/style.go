// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package css

import (
	"sort"

	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/svgast"
)

// CollectStylesheet collects CSS rules from all <style> elements and builds
// the parent map for the entire tree.
func CollectStylesheet(root *svgast.Root) *Stylesheet {
	var rules []StylesheetRule
	parents := make(map[svgast.Node]svgast.Parent)

	// Build parent map and collect style rules
	var walk func(parent svgast.Parent, children []svgast.Node)
	walk = func(parent svgast.Parent, children []svgast.Node) {
		for _, child := range children {
			parents[child] = parent
			if elem, ok := child.(*svgast.Element); ok {
				// Collect styles from <style> elements
				if elem.Name == "style" {
					typeAttr, _ := elem.Attributes.Get("type")
					if typeAttr == "" || typeAttr == "text/css" {
						mediaAttr, _ := elem.Attributes.Get("media")
						dynamic := mediaAttr != "" && mediaAttr != "all"

						for _, c := range elem.Children {
							var cssText string
							switch n := c.(type) {
							case *svgast.Text:
								cssText = n.Value
							case *svgast.Cdata:
								cssText = n.Value
							}
							if cssText != "" {
								rules = append(rules, ParseStylesheet(cssText, dynamic)...)
							}
						}
					}
				}

				walk(elem, elem.Children)
			}
		}
	}

	walk(root, root.Children)

	// Sort rules by specificity
	sort.SliceStable(rules, func(i, j int) bool {
		return CompareSpecificity(rules[i].Specificity, rules[j].Specificity) < 0
	})

	return &Stylesheet{Rules: rules, Parents: parents}
}

// ComputeOwnStyle computes the styles for a specific element (not inherited).
func ComputeOwnStyle(stylesheet *Stylesheet, node *svgast.Element) ComputedStyles {
	computedStyle := make(ComputedStyles)
	importantStyles := make(map[string]bool)

	// Collect presentation attributes
	for _, entry := range node.Attributes.Entries() {
		if collections.PresentationAttrs[entry.Name] {
			computedStyle[entry.Name] = &ComputedStyle{
				Type: StyleStatic, Inherited: false, Value: entry.Value,
			}
			importantStyles[entry.Name] = false
		}
	}

	// Collect matching CSS rules
	for _, rule := range stylesheet.Rules {
		if Matches(node, rule.Selector, stylesheet.Parents) {
			for _, decl := range rule.Declarations {
				computed := computedStyle[decl.Name]
				if computed != nil && computed.Type == StyleDynamic {
					continue
				}
				if rule.Dynamic {
					computedStyle[decl.Name] = &ComputedStyle{
						Type: StyleDynamic, Inherited: false,
					}
					continue
				}
				if computed == nil || decl.Important || !importantStyles[decl.Name] {
					computedStyle[decl.Name] = &ComputedStyle{
						Type: StyleStatic, Inherited: false, Value: decl.Value,
					}
					importantStyles[decl.Name] = decl.Important
				}
			}
		}
	}

	// Collect inline styles
	styleAttr, hasStyle := node.Attributes.Get("style")
	if hasStyle && styleAttr != "" {
		declarations := ParseStyleDeclarations(styleAttr)
		for _, decl := range declarations {
			computed := computedStyle[decl.Name]
			if computed != nil && computed.Type == StyleDynamic {
				continue
			}
			if computed == nil || decl.Important || !importantStyles[decl.Name] {
				computedStyle[decl.Name] = &ComputedStyle{
					Type: StyleStatic, Inherited: false, Value: decl.Value,
				}
				importantStyles[decl.Name] = decl.Important
			}
		}
	}

	return computedStyle
}

// ComputeStyle computes styles for an element with inheritance from parents.
func ComputeStyle(stylesheet *Stylesheet, node *svgast.Element) ComputedStyles {
	computedStyles := ComputeOwnStyle(stylesheet, node)

	// Walk up parent chain for inherited styles
	parent := stylesheet.Parents[node]
	for parent != nil {
		if parentElem, ok := parent.(*svgast.Element); ok {
			inheritedStyles := ComputeOwnStyle(stylesheet, parentElem)
			for name, computed := range inheritedStyles {
				if computedStyles[name] == nil &&
					collections.InheritableAttrs[name] &&
					!collections.PresentationNonInheritableGroupAttrs[name] {
					computedStyles[name] = &ComputedStyle{
						Type:      computed.Type,
						Inherited: true,
						Value:     computed.Value,
					}
				}
			}
			parent = stylesheet.Parents[parentElem]
		} else {
			break
		}
	}

	return computedStyles
}
