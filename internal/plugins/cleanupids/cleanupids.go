// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package cleanupids implements the cleanupIds SVGO plugin.
// It removes unused IDs and minifies used IDs (only if there are no
// <style> or <script> nodes, unless force is true).
package cleanupids

import (
	"net/url"
	"strings"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
	"github.com/okooo5km/ogvs/internal/tools"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "cleanupIds",
		Description: "removes unused IDs and minifies used",
		Fn:          fn,
	})
}

// generateIdChars is the character set for minified IDs: a-z, A-Z
var generateIdChars = []byte{
	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm',
	'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
	'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M',
	'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
}

var maxIdIndex = len(generateIdChars) - 1

// hasStringPrefix checks if a string starts with any of the given prefixes.
func hasStringPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// generateId increments the ID counter array (base-52 counting).
// nil input starts at [0]. Each element is an index into generateIdChars.
func generateId(currentId []int) []int {
	if currentId == nil {
		return []int{0}
	}
	currentId[len(currentId)-1]++
	for i := len(currentId) - 1; i > 0; i-- {
		if currentId[i] > maxIdIndex {
			currentId[i] = 0
			currentId[i-1]++
		}
	}
	if currentId[0] > maxIdIndex {
		currentId[0] = 0
		// Prepend a new digit
		currentId = append([]int{0}, currentId...)
	}
	return currentId
}

// getIdString converts an ID counter array to a string.
func getIdString(arr []int) string {
	var sb strings.Builder
	for _, i := range arr {
		sb.WriteByte(generateIdChars[i])
	}
	return sb.String()
}

// refEntry represents a reference to an ID from another element's attribute.
type refEntry struct {
	element *svgast.Element
	name    string
}

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	// Parse parameters
	remove := true
	minify := true
	force := false
	var preserveIds map[string]bool
	var preserveIdPrefixes []string

	if v, ok := params["remove"].(bool); ok {
		remove = v
	}
	if v, ok := params["minify"].(bool); ok {
		minify = v
	}
	if v, ok := params["force"].(bool); ok {
		force = v
	}

	// Handle preserve: can be string or []any
	preserveIds = make(map[string]bool)
	switch v := params["preserve"].(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				preserveIds[s] = true
			}
		}
	case string:
		if v != "" {
			preserveIds[v] = true
		}
	}

	// Handle preservePrefixes: can be string or []any
	switch v := params["preservePrefixes"].(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				preserveIdPrefixes = append(preserveIdPrefixes, s)
			}
		}
	case string:
		if v != "" {
			preserveIdPrefixes = append(preserveIdPrefixes, v)
		}
	}

	// nodeById: maps ID to the element node that has that ID
	nodeById := make(map[string]*svgast.Element)
	// nodeByIdOrder: preserves insertion order of IDs in nodeById
	var nodeByIdOrder []string

	// referencesById: maps ID to all elements/attributes that reference it
	referencesById := make(map[string][]refEntry)
	// referencesByIdOrder: preserves insertion order of IDs in referencesById
	var referencesByIdOrder []string

	deoptimized := false

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				if !force {
					// deoptimize if style or scripts are present
					if (elem.Name == "style" && len(elem.Children) != 0) ||
						tools.HasScripts(elem) {
						deoptimized = true
						return nil
					}

					// avoid removing IDs if the whole SVG consists only of defs
					if elem.Name == "svg" {
						hasDefsOnly := true
						for _, child := range elem.Children {
							childElem, ok := child.(*svgast.Element)
							if !ok || childElem.Name != "defs" {
								hasDefsOnly = false
								break
							}
						}
						if hasDefsOnly {
							return svgast.ErrVisitSkip
						}
					}
				}

				for _, entry := range elem.Attributes.Entries() {
					if entry.Name == "id" {
						// collect all ids
						id := entry.Value
						if _, exists := nodeById[id]; exists {
							// remove repeated id
							elem.Attributes.Delete("id")
						} else {
							nodeById[id] = elem
							nodeByIdOrder = append(nodeByIdOrder, id)
						}
					} else {
						ids := tools.FindReferences(entry.Name, entry.Value)
						for _, id := range ids {
							if _, exists := referencesById[id]; !exists {
								referencesByIdOrder = append(referencesByIdOrder, id)
							}
							referencesById[id] = append(referencesById[id], refEntry{
								element: elem,
								name:    entry.Name,
							})
						}
					}
				}

				return nil
			},
		},
		Root: &svgast.VisitorCallbacks{
			Exit: func(_ svgast.Node, _ svgast.Parent) {
				if deoptimized {
					return
				}

				isIdPreserved := func(id string) bool {
					return preserveIds[id] || hasStringPrefix(id, preserveIdPrefixes)
				}

				var currentId []int

				// Iterate referencesById in insertion order
				for _, id := range referencesByIdOrder {
					refs := referencesById[id]
					nodeForId, nodeExists := nodeById[id]
					if !nodeExists {
						continue
					}

					// replace referenced IDs with the minified ones
					if minify && !isIdPreserved(id) {
						var currentIdString string
						for {
							currentId = generateId(currentId)
							currentIdString = getIdString(currentId)
							// Skip preserved IDs and IDs that are referenced but have no node
							if !isIdPreserved(currentIdString) {
								_, refExists := referencesById[currentIdString]
								_, nodeForGenerated := nodeById[currentIdString]
								if !(refExists && !nodeForGenerated) {
									break
								}
							}
						}

						nodeForId.Attributes.Set("id", currentIdString)
						for _, ref := range refs {
							value, _ := ref.element.Attributes.Get(ref.name)
							if strings.Contains(value, "#") {
								// replace id in href and url()
								encodedId := url.PathEscape(id)
								value = strings.Replace(value, "#"+encodedId, "#"+currentIdString, 1)
								value = strings.Replace(value, "#"+id, "#"+currentIdString, 1)
								ref.element.Attributes.Set(ref.name, value)
							} else {
								// replace id in begin attribute
								value = strings.Replace(value, id+".", currentIdString+".", 1)
								ref.element.Attributes.Set(ref.name, value)
							}
						}
					}

					// keep referenced node
					delete(nodeById, id)
					// Also remove from nodeByIdOrder tracking (not strictly needed
					// since we check nodeById below)
				}

				// remove non-referenced IDs attributes from elements
				if remove {
					for _, id := range nodeByIdOrder {
						nodeForId, stillExists := nodeById[id]
						if !stillExists {
							continue
						}
						if !isIdPreserved(id) {
							nodeForId.Attributes.Delete("id")
						}
					}
				}
			},
		},
	}
}
