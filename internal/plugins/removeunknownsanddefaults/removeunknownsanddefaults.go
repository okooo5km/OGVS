// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removeunknownsanddefaults implements the removeUnknownsAndDefaults
// SVGO plugin. It removes unknown elements content and attributes, and removes
// attributes with default values.
package removeunknownsanddefaults

import (
	"regexp"
	"strings"

	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/css"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeUnknownsAndDefaults",
		Description: "removes unknown elements content and attributes, removes attrs with default values",
		Fn:          fn,
	})
}

// Pre-computed lookup tables, matching SVGO's module-level computation.
// These resolve all group references at init time.
var (
	allowedChildrenPerElement    map[string]map[string]bool
	allowedAttributesPerElement  map[string]map[string]bool
	attributesDefaultsPerElement map[string]map[string]string
)

func init() {
	allowedChildrenPerElement = make(map[string]map[string]bool)
	allowedAttributesPerElement = make(map[string]map[string]bool)
	attributesDefaultsPerElement = make(map[string]map[string]string)

	for name, config := range collections.Elems {
		// Build allowed children
		allowedChildren := make(map[string]bool)
		for elemName := range config.Content {
			allowedChildren[elemName] = true
		}
		for groupName := range config.ContentGroups {
			if elemsGroup, ok := collections.ElemsGroups[groupName]; ok {
				for elemName := range elemsGroup {
					allowedChildren[elemName] = true
				}
			}
		}

		// Build allowed attributes
		allowedAttributes := make(map[string]bool)
		for attrName := range config.Attrs {
			allowedAttributes[attrName] = true
		}

		// Build attribute defaults
		attributeDefaults := make(map[string]string)
		if config.Defaults != nil {
			for attrName, defaultValue := range config.Defaults {
				attributeDefaults[attrName] = defaultValue
			}
		}

		for groupName := range config.AttrsGroups {
			if attrsGroup, ok := collections.AttrsGroups[groupName]; ok {
				for attrName := range attrsGroup {
					allowedAttributes[attrName] = true
				}
			}
			if groupDefaults, ok := collections.AttrsGroupsDefaults[groupName]; ok {
				for attrName, defaultValue := range groupDefaults {
					attributeDefaults[attrName] = defaultValue
				}
			}
		}

		allowedChildrenPerElement[name] = allowedChildren
		allowedAttributesPerElement[name] = allowedAttributes
		attributesDefaultsPerElement[name] = attributeDefaults
	}
}

// regStandaloneSingle and regStandaloneDouble match standalone="no" or standalone='no'
// in XML declarations. Go RE2 doesn't support backreferences (\1), so we use two
// separate regexes to match each quote style (matching SVGO's /\s*standalone\s*=\s*(["'])no\1/).
var regStandaloneSingle = regexp.MustCompile(`\s*standalone\s*=\s*'no'`)
var regStandaloneDouble = regexp.MustCompile(`\s*standalone\s*=\s*"no"`)

// removeStandalone replaces standalone="no" or standalone='no' in the string,
// matching SVGO's regex /\s*standalone\s*=\s*(["'])no\1/.
func removeStandalone(s string) string {
	s = regStandaloneDouble.ReplaceAllString(s, "")
	s = regStandaloneSingle.ReplaceAllString(s, "")
	return s
}

func fn(root *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	unknownContent := getBoolParam(params, "unknownContent", true)
	unknownAttrs := getBoolParam(params, "unknownAttrs", true)
	defaultAttrs := getBoolParam(params, "defaultAttrs", true)
	defaultMarkupDeclarations := getBoolParam(params, "defaultMarkupDeclarations", true)
	uselessOverrides := getBoolParam(params, "uselessOverrides", true)
	keepDataAttrs := getBoolParam(params, "keepDataAttrs", true)
	keepAriaAttrs := getBoolParam(params, "keepAriaAttrs", true)
	keepRoleAttr := getBoolParam(params, "keepRoleAttr", false)

	stylesheet := css.CollectStylesheet(root)

	return &svgast.Visitor{
		Instruction: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				if defaultMarkupDeclarations {
					inst := node.(*svgast.Instruction)
					inst.Value = removeStandalone(inst.Value)
				}
				return nil
			},
		},
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parentNode svgast.Parent) error {
				elem := node.(*svgast.Element)

				// Skip namespaced elements
				if strings.Contains(elem.Name, ":") {
					return nil
				}
				// Skip visiting foreignObject subtree
				if elem.Name == "foreignObject" {
					return svgast.ErrVisitSkip
				}

				// Remove unknown element's content
				if unknownContent {
					if parentElem, ok := parentNode.(*svgast.Element); ok {
						allowedChildren := allowedChildrenPerElement[parentElem.Name]
						if allowedChildren == nil || len(allowedChildren) == 0 {
							// Parent has no spec info about children —
							// remove child if child itself is unknown
							if allowedChildrenPerElement[elem.Name] == nil {
								svgast.DetachNodeFromParent(node, parentNode)
								return nil
							}
						} else {
							// Parent has allowed children list — remove if not in it
							if !allowedChildren[elem.Name] {
								svgast.DetachNodeFromParent(node, parentNode)
								return nil
							}
						}
					}
				}

				allowedAttributes := allowedAttributesPerElement[elem.Name]
				attributeDefaults := attributesDefaultsPerElement[elem.Name]

				var computedParentStyle css.ComputedStyles
				if parentElem, ok := parentNode.(*svgast.Element); ok {
					computedParentStyle = css.ComputeStyle(stylesheet, parentElem)
				}

				// Iterate over attributes — collect names to delete, then delete
				// (matching SVGO's iteration + delete pattern; in Go we snapshot entries)
				entries := elem.Attributes.Entries()
				for _, entry := range entries {
					name := entry.Name
					value := entry.Value

					if keepDataAttrs && strings.HasPrefix(name, "data-") {
						continue
					}
					if keepAriaAttrs && strings.HasPrefix(name, "aria-") {
						continue
					}
					if keepRoleAttr && name == "role" {
						continue
					}
					// Skip xmlns attribute
					if name == "xmlns" {
						continue
					}
					// Skip namespaced attributes except xml:* and xlink:*
					if strings.Contains(name, ":") {
						prefix := name[:strings.Index(name, ":")]
						if prefix != "xml" && prefix != "xlink" {
							continue
						}
					}

					// Remove unknown attributes
					if unknownAttrs && allowedAttributes != nil && !allowedAttributes[name] {
						elem.Attributes.Delete(name)
					}

					// Remove attributes with default values (only if element has no id)
					if defaultAttrs && !elem.Attributes.Has("id") && attributeDefaults != nil {
						if defaultVal, hasDefault := attributeDefaults[name]; hasDefault && defaultVal == value {
							// Keep defaults if parent has own or inherited style for this property
							if computedParentStyle != nil && computedParentStyle[name] != nil {
								// Parent has computed style for this prop — keep the attribute
							} else if css.IncludesAttrSelector(stylesheet.Rules, name, nil) {
								// Some CSS rule references this attr — keep it
							} else {
								elem.Attributes.Delete(name)
							}
						}
					}

					// Remove useless overrides (same value as parent's computed style)
					if uselessOverrides && !elem.Attributes.Has("id") {
						if computedParentStyle != nil {
							style := computedParentStyle[name]
							if !collections.PresentationNonInheritableGroupAttrs[name] &&
								style != nil &&
								style.Type == css.StyleStatic &&
								style.Value == value {
								elem.Attributes.Delete(name)
							}
						}
					}
				}

				return nil
			},
		},
	}
}

// getBoolParam extracts a boolean parameter with a default value.
func getBoolParam(params map[string]any, key string, defaultVal bool) bool {
	if v, ok := params[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}
