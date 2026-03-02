// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package convertstylettoattrs converts style attribute declarations
// to SVG presentation attributes.
package convertstylettoattrs

import (
	"strings"

	ogcss "github.com/okooo5km/ogvs/internal/css"

	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "convertStyleToAttrs",
		Description: "converts style to attributes",
		Fn:          fn,
	})
}

func fn(root *svgast.Root, params map[string]any, info *plugin.PluginInfo) *svgast.Visitor {
	keepImportant := false
	if v, ok := params["keepImportant"].(bool); ok {
		keepImportant = v
	}

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem, ok := node.(*svgast.Element)
				if !ok {
					return nil
				}

				styleVal, has := elem.Attributes.Get("style")
				if !has || styleVal == "" {
					return nil
				}

				// Parse declarations using CSS parser
				decls := ogcss.ParseStyleDeclarations(styleVal)

				type styleDecl struct {
					prop      string
					val       string
					important bool
				}
				var remaining []styleDecl
				var newAttrs []styleDecl

				for _, decl := range decls {
					propLower := strings.ToLower(strings.TrimSpace(decl.Name))
					val := strings.TrimSpace(decl.Value)

					// Keep !important declarations in style if keepImportant
					if keepImportant && decl.Important {
						remaining = append(remaining, styleDecl{decl.Name, val, true})
						continue
					}

					if collections.PresentationAttrs[propLower] {
						// Unquote for attribute values only
						attrVal := val
						if len(attrVal) >= 2 &&
							((attrVal[0] == '\'' && attrVal[len(attrVal)-1] == '\'') ||
								(attrVal[0] == '"' && attrVal[len(attrVal)-1] == '"')) {
							attrVal = attrVal[1 : len(attrVal)-1]
						}
						newAttrs = append(newAttrs, styleDecl{propLower, attrVal, decl.Important})
					} else {
						remaining = append(remaining, styleDecl{decl.Name, val, decl.Important})
					}
				}

				// Apply new attributes (only if not already set)
				for _, attr := range newAttrs {
					if _, exists := elem.Attributes.Get(attr.prop); !exists {
						elem.Attributes.Set(attr.prop, attr.val)
					}
				}

				// Update or remove style attribute
				if len(remaining) > 0 {
					var parts []string
					for _, s := range remaining {
						part := s.prop + ":" + s.val
						if s.important {
							part += "!important"
						}
						parts = append(parts, part)
					}
					elem.Attributes.Set("style", strings.Join(parts, ";"))
				} else if len(newAttrs) > 0 {
					elem.Attributes.Delete("style")
				}

				return nil
			},
		},
	}
}
