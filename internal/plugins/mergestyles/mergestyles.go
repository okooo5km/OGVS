// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package mergestyles merges multiple <style> elements into one.
package mergestyles

import (
	"strings"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "mergeStyles",
		Description: "merge multiple style elements into one",
		Fn:          fn,
	})
}

func fn(root *svgast.Root, params map[string]any, info *plugin.PluginInfo) *svgast.Visitor {
	var firstStyleElement *svgast.Element
	var collectedStyles strings.Builder
	styleContentType := "text" // "text" or "cdata"

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem, ok := node.(*svgast.Element)
				if !ok {
					return nil
				}

				// skip <foreignObject> content
				if elem.Name == "foreignObject" {
					return svgast.ErrVisitSkip
				}

				// collect style elements
				if elem.Name != "style" {
					return nil
				}

				// skip <style> with invalid type attribute
				if typeAttr, has := elem.Attributes.Get("type"); has &&
					typeAttr != "" && typeAttr != "text/css" {
					return nil
				}

				// extract style element content
				var css strings.Builder
				for _, child := range elem.Children {
					switch c := child.(type) {
					case *svgast.Text:
						css.WriteString(c.Value)
					case *svgast.Cdata:
						styleContentType = "cdata"
						css.WriteString(c.Value)
					}
				}

				// remove empty style elements
				if strings.TrimSpace(css.String()) == "" {
					svgast.DetachNodeFromParent(node, parent)
					return nil
				}

				// collect css and wrap with media query if present in attribute
				if mediaAttr, has := elem.Attributes.Get("media"); !has || mediaAttr == "" {
					collectedStyles.WriteString(css.String())
				} else {
					collectedStyles.WriteString("@media ")
					collectedStyles.WriteString(mediaAttr)
					collectedStyles.WriteString("{")
					collectedStyles.WriteString(css.String())
					collectedStyles.WriteString("}")
					elem.Attributes.Delete("media")
				}

				// combine collected styles in the first style element
				if firstStyleElement == nil {
					firstStyleElement = elem
				} else {
					svgast.DetachNodeFromParent(node, parent)
					// update first style element with all collected styles
					var child svgast.Node
					if styleContentType == "cdata" {
						child = &svgast.Cdata{Value: collectedStyles.String()}
					} else {
						child = &svgast.Text{Value: collectedStyles.String()}
					}
					firstStyleElement.Children = []svgast.Node{child}
				}

				return nil
			},
		},
	}
}
