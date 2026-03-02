// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removeunusedns implements the removeUnusedNS SVGO plugin.
// It removes unused namespace declarations from the root svg element.
package removeunusedns

import (
	"strings"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeUnusedNS",
		Description: "removes unused namespaces declaration",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	unusedNamespaces := make(map[string]bool)

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				// Collect namespace declarations from root svg element
				if elem.Name == "svg" && isRootParent(parent) {
					for _, entry := range elem.Attributes.Entries() {
						if strings.HasPrefix(entry.Name, "xmlns:") {
							local := entry.Name[len("xmlns:"):]
							unusedNamespaces[local] = true
						}
					}
				}

				if len(unusedNamespaces) == 0 {
					return nil
				}

				// Preserve namespace used in element names
				if strings.Contains(elem.Name, ":") {
					ns := elem.Name[:strings.Index(elem.Name, ":")]
					delete(unusedNamespaces, ns)
				}

				// Preserve namespace used in attributes
				for _, entry := range elem.Attributes.Entries() {
					if strings.Contains(entry.Name, ":") {
						ns := entry.Name[:strings.Index(entry.Name, ":")]
						delete(unusedNamespaces, ns)
					}
				}

				return nil
			},
			Exit: func(node svgast.Node, parent svgast.Parent) {
				elem := node.(*svgast.Element)

				// Remove unused namespace attrs from root svg element
				if elem.Name == "svg" && isRootParent(parent) {
					for ns := range unusedNamespaces {
						elem.Attributes.Delete("xmlns:" + ns)
					}
				}
			},
		},
	}
}

func isRootParent(parent svgast.Parent) bool {
	_, ok := parent.(*svgast.Root)
	return ok
}
