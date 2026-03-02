// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package cleanupenablebackground implements the cleanupEnableBackground SVGO plugin.
// It removes or cleans up the enable-background attribute when possible.
package cleanupenablebackground

import (
	"regexp"
	"strings"

	"github.com/okooo5km/ogvs/internal/css"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "cleanupEnableBackground",
		Description: "remove or cleanup enable-background attribute when possible",
		Fn:          fn,
	})
}

var regEnableBackground = regexp.MustCompile(
	`^new\s0\s0\s([-+]?\d*\.?\d+([eE][-+]?\d+)?)\s([-+]?\d*\.?\d+([eE][-+]?\d+)?)$`,
)

// cleanupValue checks if an enable-background value matches "new 0 0 W H"
// where W and H match the element's width and height.
// Returns the cleaned value, or empty string if redundant (for svg element).
func cleanupValue(value, nodeName, width, height string) (string, bool) {
	match := regEnableBackground.FindStringSubmatch(value)
	if match != nil && width == match[1] && height == match[3] {
		if nodeName == "svg" {
			return "", true // redundant, remove entirely
		}
		return "new", true // simplify to just "new"
	}
	return value, false // keep original value
}

// removeEnableBackgroundFromStyle removes enable-background declarations from
// an inline style attribute value, returning the cleaned style string.
// If all declarations are removed, returns empty string.
func removeEnableBackgroundFromStyle(styleValue string) (cleanedStyle string, hasEB bool, ebValue string) {
	declarations := css.ParseStyleDeclarations(styleValue)

	// Find the last enable-background declaration
	lastEBIdx := -1
	for i, decl := range declarations {
		if decl.Name == "enable-background" {
			lastEBIdx = i
		}
	}

	if lastEBIdx == -1 {
		return styleValue, false, ""
	}

	ebValue = declarations[lastEBIdx].Value

	// Rebuild style without enable-background declarations
	var parts []string
	for i, decl := range declarations {
		if decl.Name == "enable-background" {
			// Keep only duplicate enable-background declarations (all but last)
			// Actually, SVGO removes all duplicates except the last, then processes the last.
			// But if we're removing, we remove all of them.
			_ = i
			continue
		}
		part := decl.Name + ":" + decl.Value
		if decl.Important {
			part += " !important"
		}
		parts = append(parts, part)
	}

	if len(parts) == 0 {
		return "", true, ebValue
	}
	return strings.Join(parts, ";"), true, ebValue
}

func fn(root *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	// First pass: check if any <filter> elements exist
	hasFilter := false
	svgast.Visit(root, &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)
				if elem.Name == "filter" {
					hasFilter = true
				}
				return nil
			},
		},
	}, nil)

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)

				// Process inline style for enable-background
				var cleanedStyle string
				var hasEB bool
				var ebStyleValue string

				if styleVal, ok := elem.Attributes.Get("style"); ok {
					cleanedStyle, hasEB, ebStyleValue = removeEnableBackgroundFromStyle(styleVal)
				}

				if !hasFilter {
					// No filter in document: remove enable-background entirely
					elem.Attributes.Delete("enable-background")

					if hasEB {
						// Remove enable-background from style
						if cleanedStyle == "" {
							elem.Attributes.Delete("style")
						} else {
							elem.Attributes.Set("style", cleanedStyle)
						}
					}

					return nil
				}

				// Has filter: only clean up matching values on svg/mask/pattern with dimensions
				hasDimensions := elem.Attributes.Has("width") && elem.Attributes.Has("height")

				if (elem.Name == "svg" || elem.Name == "mask" || elem.Name == "pattern") && hasDimensions {
					width, _ := elem.Attributes.Get("width")
					height, _ := elem.Attributes.Get("height")

					// Process attribute
					if attrValue, ok := elem.Attributes.Get("enable-background"); ok {
						cleaned, changed := cleanupValue(attrValue, elem.Name, width, height)
						if changed {
							if cleaned == "" {
								elem.Attributes.Delete("enable-background")
							} else {
								elem.Attributes.Set("enable-background", cleaned)
							}
						}
					}

					// Process style enable-background
					if hasEB {
						cleaned, changed := cleanupValue(ebStyleValue, elem.Name, width, height)
						if changed {
							if cleaned == "" {
								// Already removed from cleanedStyle
								if cleanedStyle == "" {
									elem.Attributes.Delete("style")
								} else {
									elem.Attributes.Set("style", cleanedStyle)
								}
							} else {
								// Replace enable-background value in style
								// Rebuild with the cleaned value
								if cleanedStyle == "" {
									elem.Attributes.Set("style", "enable-background:"+cleaned)
								} else {
									elem.Attributes.Set("style", cleanedStyle+";enable-background:"+cleaned)
								}
							}
						} else {
							// Value didn't match - but we still need to deduplicate
							// (remove earlier duplicates). The cleaned style already does this
							// by only keeping non-EB declarations. Re-add the last EB.
							if cleanedStyle == "" {
								elem.Attributes.Set("style", "enable-background:"+ebStyleValue)
							} else {
								elem.Attributes.Set("style", cleanedStyle+";enable-background:"+ebStyleValue)
							}
						}
					}
				} else if hasEB {
					// Not svg/mask/pattern or no dimensions - keep enable-background but deduplicate
					if cleanedStyle == "" {
						elem.Attributes.Set("style", "enable-background:"+ebStyleValue)
					} else {
						elem.Attributes.Set("style", cleanedStyle+";enable-background:"+ebStyleValue)
					}
				}

				return nil
			},
		},
	}
}
