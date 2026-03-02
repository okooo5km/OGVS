// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removexlink implements the removeXlink SVGO plugin.
// It removes xlink namespace and replaces attributes with the SVG 2
// equivalent where applicable.
package removexlink

import (
	"strings"

	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeXlink",
		Description: "remove xlink namespace and replaces attributes with the SVG 2 equivalent where applicable",
		Fn:          fn,
	})
}

// xlinkNamespace is the URI indicating the XLink namespace.
const xlinkNamespace = "http://www.w3.org/1999/xlink"

// showToTarget maps xlink:show values to the SVG 2 target attribute values.
var showToTarget = map[string]string{
	"new":     "_blank",
	"replace": "_self",
}

// legacyElements are elements that use xlink:href but were deprecated in SVG 2
// and therefore don't support the SVG 2 href attribute.
var legacyElements = map[string]bool{
	"cursor":        true,
	"filter":        true,
	"font-face-uri": true,
	"glyphRef":      true,
	"tref":          true,
}

// findPrefixedAttrs returns attribute names of the form "prefix:attr" that
// exist on the node, for each prefix in prefixes.
func findPrefixedAttrs(elem *svgast.Element, prefixes []string, attr string) []string {
	var result []string
	for _, prefix := range prefixes {
		name := prefix + ":" + attr
		if elem.Attributes.Has(name) {
			result = append(result, name)
		}
	}
	return result
}

// sliceContains checks if a string slice contains the given value.
func sliceContains(s []string, v string) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}

// sliceRemoveFirst removes the first occurrence of v from s.
func sliceRemoveFirst(s []string, v string) []string {
	for i, item := range s {
		if item == v {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	// Parse includeLegacy param (default: false)
	includeLegacy := false
	if v, ok := params["includeLegacy"]; ok {
		if b, ok := v.(bool); ok {
			includeLegacy = b
		}
	}

	// XLink namespace prefixes that are currently in the stack
	xlinkPrefixes := []string{}

	// Namespace prefixes that exist in xlinkPrefixes but were overridden
	// in a child element to point to another namespace
	overriddenPrefixes := []string{}

	// Namespace prefixes that were used in one of the legacy elements
	usedInLegacyElement := []string{}

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				// Scan xmlns: declarations to find xlink prefixes
				for _, entry := range elem.Attributes.Entries() {
					if strings.HasPrefix(entry.Name, "xmlns:") {
						prefix := entry.Name[len("xmlns:"):]

						if entry.Value == xlinkNamespace {
							xlinkPrefixes = append(xlinkPrefixes, prefix)
							continue
						}

						if sliceContains(xlinkPrefixes, prefix) {
							overriddenPrefixes = append(overriddenPrefixes, prefix)
						}
					}
				}

				// If any overridden prefix is still in xlinkPrefixes, skip processing
				for _, prefix := range overriddenPrefixes {
					if sliceContains(xlinkPrefixes, prefix) {
						return nil
					}
				}

				// Handle xlink:show attributes
				showAttrs := findPrefixedAttrs(elem, xlinkPrefixes, "show")
				showHandled := elem.Attributes.Has("target")
				for i := len(showAttrs) - 1; i >= 0; i-- {
					attr := showAttrs[i]
					value, _ := elem.Attributes.Get(attr)
					mapping, hasMappng := showToTarget[value]

					if showHandled || !hasMappng {
						elem.Attributes.Delete(attr)
						continue
					}

					// Check if the mapping differs from the element's default target
					defaultTarget := ""
					if ec, ok := collections.Elems[elem.Name]; ok && ec.Defaults != nil {
						if dt, ok := ec.Defaults["target"]; ok {
							defaultTarget = dt
						}
					}
					if mapping != defaultTarget {
						elem.Attributes.Set("target", mapping)
					}

					elem.Attributes.Delete(attr)
					showHandled = true
				}

				// Handle xlink:title attributes
				titleAttrs := findPrefixedAttrs(elem, xlinkPrefixes, "title")
				for i := len(titleAttrs) - 1; i >= 0; i-- {
					attr := titleAttrs[i]
					value, _ := elem.Attributes.Get(attr)

					// Check if there's already a <title> child element
					hasTitle := false
					for _, child := range elem.Children {
						if childElem, ok := child.(*svgast.Element); ok && childElem.Name == "title" {
							hasTitle = true
							break
						}
					}

					if hasTitle {
						elem.Attributes.Delete(attr)
						continue
					}

					// Create a <title> element and prepend it to children
					titleTag := &svgast.Element{
						Name:       "title",
						Attributes: svgast.NewOrderedAttrs(),
						Children: []svgast.Node{
							&svgast.Text{Value: value},
						},
					}

					// Prepend to children (unshift)
					newChildren := make([]svgast.Node, 0, len(elem.Children)+1)
					newChildren = append(newChildren, titleTag)
					newChildren = append(newChildren, elem.Children...)
					elem.Children = newChildren

					elem.Attributes.Delete(attr)
				}

				// Handle xlink:href attributes
				hrefAttrs := findPrefixedAttrs(elem, xlinkPrefixes, "href")

				if len(hrefAttrs) > 0 && legacyElements[elem.Name] && !includeLegacy {
					// Extract prefixes from href attrs and record as used in legacy element.
					// JS: attr.split(':', 1)[0] returns the part before the first colon.
					for _, attr := range hrefAttrs {
						colonIdx := strings.Index(attr, ":")
						if colonIdx >= 0 {
							prefix := attr[:colonIdx]
							usedInLegacyElement = append(usedInLegacyElement, prefix)
						}
					}
					return nil
				}

				for i := len(hrefAttrs) - 1; i >= 0; i-- {
					attr := hrefAttrs[i]
					value, _ := elem.Attributes.Get(attr)

					if elem.Attributes.Has("href") {
						elem.Attributes.Delete(attr)
						continue
					}

					elem.Attributes.Set("href", value)
					elem.Attributes.Delete(attr)
				}

				return nil
			},
			Exit: func(node svgast.Node, parent svgast.Parent) {
				elem := node.(*svgast.Element)

				// Snapshot entries before modification to avoid issues
				// when deleting during iteration.
				entries := elem.Attributes.Entries()

				// Collect keys to delete
				var toDelete []string

				for _, entry := range entries {
					key := entry.Name
					value := entry.Value

					parts := strings.SplitN(key, ":", 2)
					prefix := parts[0]
					attr := ""
					if len(parts) == 2 {
						attr = parts[1]
					}

					// Remove remaining xlink-prefixed attributes
					if sliceContains(xlinkPrefixes, prefix) &&
						!sliceContains(overriddenPrefixes, prefix) &&
						!sliceContains(usedInLegacyElement, prefix) &&
						!includeLegacy {
						toDelete = append(toDelete, key)
						continue
					}

					// Remove xlink namespace declarations
					if strings.HasPrefix(key, "xmlns:") && !sliceContains(usedInLegacyElement, attr) {
						if value == xlinkNamespace {
							xlinkPrefixes = sliceRemoveFirst(xlinkPrefixes, attr)
							toDelete = append(toDelete, key)
							continue
						}

						if sliceContains(overriddenPrefixes, prefix) {
							overriddenPrefixes = sliceRemoveFirst(overriddenPrefixes, attr)
						}
					}
				}

				// Perform deletions after iteration
				for _, key := range toDelete {
					elem.Attributes.Delete(key)
				}
			},
		},
	}
}
