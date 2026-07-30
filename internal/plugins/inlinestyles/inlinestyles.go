// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package inlinestyles inlines CSS rules from <style> elements into
// inline style attributes, ported from SVGO's inlineStyles.js.
package inlinestyles

import (
	"sort"
	"strings"

	"github.com/okooo5km/ogvs/internal/collections"
	ogcss "github.com/okooo5km/ogvs/internal/css"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "inlineStyles",
		Description: "inline styles (additional options)",
		Fn:          fn,
	})
}

type styleInfo struct {
	node    *svgast.Element
	parent  svgast.Parent
	cssText string // raw CSS text from this style element
	// mediaQuery is the media query the element's own media attribute imposes
	// on its top-level rules; empty for an unconditional style element.
	mediaQuery string
}

type selectorInfo struct {
	selector         string // cleaned selector (no pseudo-classes), for matching
	originalSelector string // original selector (with pseudo-classes), for CSS output
	mediaQuery       string // media query context the rule was collected under
	specificity      ogcss.Specificity
	rule             ogcss.StylesheetRule
}

// appliedKey identifies a rule by both its selector text and its media query
// context. Selector text alone is ambiguous: the same selector may appear both
// at the top level and inside an at-rule, and only the top-level one is ever
// inlined.
func appliedKey(mediaQuery, selector string) string {
	return mediaQuery + "|" + selector
}

// styledDecl represents a single CSS declaration with order tracking.
type styledDecl struct {
	name      string
	value     string
	important bool
}

func fn(root *svgast.Root, params map[string]any, info *plugin.PluginInfo) *svgast.Visitor {
	onlyMatchedOnce := true
	removeMatchedSelectors := true
	useMqs := map[string]bool{"": true, "screen": true}
	usePseudos := map[string]bool{"": true}

	if v, ok := params["onlyMatchedOnce"].(bool); ok {
		onlyMatchedOnce = v
	}
	if v, ok := params["removeMatchedSelectors"].(bool); ok {
		removeMatchedSelectors = v
	}
	if v, ok := params["useMqs"].([]any); ok {
		useMqs = make(map[string]bool)
		for _, item := range v {
			if s, ok := item.(string); ok {
				useMqs[s] = true
			}
		}
	}
	if v, ok := params["usePseudos"].([]any); ok {
		usePseudos = make(map[string]bool)
		for _, item := range v {
			if s, ok := item.(string); ok {
				usePseudos[s] = true
			}
		}
	}

	var styles []styleInfo
	// inlineable selectors (passed media query + pseudo-class filter)
	var selectors []selectorInfo

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

				if elem.Name != "style" || len(elem.Children) == 0 {
					return nil
				}

				// skip non-CSS style elements
				if typeAttr, has := elem.Attributes.Get("type"); has &&
					typeAttr != "" && typeAttr != "text/css" {
					return nil
				}

				// Extract CSS text
				var cssText strings.Builder
				for _, child := range elem.Children {
					switch c := child.(type) {
					case *svgast.Text:
						cssText.WriteString(c.Value)
					case *svgast.Cdata:
						cssText.WriteString(c.Value)
					}
				}

				// Determine default media query from <style media="..."> attribute
				mediaAttr, _ := elem.Attributes.Get("media")
				defaultMediaQuery := ""
				if mediaAttr != "" && mediaAttr != "all" {
					defaultMediaQuery = mediaAttr
				}

				// Parse rules from the CSS
				dynamic := defaultMediaQuery != ""
				rules := ogcss.ParseStylesheet(cssText.String(), dynamic)

				// Check if this style element has any processable content
				hasProcessable := false
				for _, rule := range rules {
					// Determine effective media query for this rule
					mq := rule.MediaQuery
					if mq == "" {
						mq = defaultMediaQuery
					}

					if useMqs[mq] {
						hasProcessable = true
						break
					}
				}

				// Also check: even if no rules are processable, the style element
				// might contain at-rules that need to be compacted
				styles = append(styles, styleInfo{
					node:       elem,
					parent:     parent,
					cssText:    cssText.String(),
					mediaQuery: defaultMediaQuery,
				})

				if !hasProcessable {
					return nil
				}

				for _, rule := range rules {
					// Determine effective media query for this rule
					mq := rule.MediaQuery
					if mq == "" {
						mq = defaultMediaQuery
					}

					// Skip rules whose media query is not in useMqs
					if !useMqs[mq] {
						continue
					}

					si := selectorInfo{
						selector:         rule.Selector,
						originalSelector: rule.OriginalSelector,
						mediaQuery:       mq,
						specificity:      rule.Specificity,
						rule:             rule,
					}

					if rule.Dynamic {
						// Check if the pseudo-class is allowed
						pseudo := extractPseudo(rule.OriginalSelector)
						if !usePseudos[pseudo] {
							// Don't inline, will be preserved by CompactStylesheet
							continue
						}
					}
					selectors = append(selectors, si)
				}

				return nil
			},
		},
		Root: &svgast.VisitorCallbacks{
			Exit: func(node svgast.Node, parent svgast.Parent) {
				if len(styles) == 0 {
					return
				}

				// Build parent map
				parents := buildParentMap(root)

				// Sort selectors: ascending by specificity, then reverse.
				// This gives us highest-specificity first, and for equal specificity,
				// later CSS rules first (which is correct per CSS cascade).
				sorted := make([]selectorInfo, len(selectors))
				copy(sorted, selectors)
				sort.SliceStable(sorted, func(i, j int) bool {
					return ogcss.CompareSpecificity(sorted[i].specificity, sorted[j].specificity) < 0
				})
				// Reverse to get highest specificity first
				for i, j := 0, len(sorted)-1; i < j; i, j = i+1, j-1 {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}

				// Track which selectors were applied (by original selector string)
				appliedSelectors := make(map[int]bool)
				appliedOrigSelectors := make(map[string]bool) // for CompactStylesheet
				// Elements each applied selector matched, captured before any
				// class or id attribute is cleaned up.
				matchedElements := make(map[int][]*svgast.Element)

				for idx, sel := range sorted {
					// SVGO wraps querySelectorAll in try/catch and skips the
					// selector when css-select throws.
					matched, ok := ogcss.QuerySelectorAllErr(root, sel.selector, parents)
					if !ok || len(matched) == 0 {
						continue
					}

					if onlyMatchedOnce && len(matched) > 1 {
						continue
					}

					// Apply declarations to matched elements
					for _, elem := range matched {
						mergeDeclarationsIntoElement(elem, sel.rule.Declarations)
					}

					appliedSelectors[idx] = true
					appliedOrigSelectors[appliedKey(sel.mediaQuery, sel.originalSelector)] = true
					matchedElements[idx] = matched
				}

				if !removeMatchedSelectors {
					return
				}

				// Rewrite each style element from its own CSS so element
				// boundaries, media attributes and CDATA framing survive.
				// CompactStylesheet preserves at-rules and handles
				// comma-separated selector groups.
				for _, style := range styles {
					shouldSkip := func(selector, mediaQuery string) bool {
						if mediaQuery == "" {
							mediaQuery = style.mediaQuery
						}
						return appliedOrigSelectors[appliedKey(mediaQuery, selector)]
					}

					css := ogcss.CompactStylesheet(style.cssText, shouldSkip)
					if css == "" {
						svgast.DetachNodeFromParent(style.node, style.parent)
						continue
					}
					setStyleText(style.node, css)
				}

				// Collect all remaining selectors for attribute reference checks
				var remainingSelectors []selectorInfo
				for idx, sel := range sorted {
					if !appliedSelectors[idx] {
						remainingSelectors = append(remainingSelectors, sel)
					}
				}

				// Clean up class and ID attributes for applied selectors
				for idx, sel := range sorted {
					if !appliedSelectors[idx] {
						continue
					}
					for _, elem := range matchedElements[idx] {
						cleanupClassAndID(elem, sel, sorted)
						cleanupPresentationAttrs(elem, remainingSelectors)
					}
				}
			},
		},
	}
}

// setStyleText writes css back into a <style> element, keeping the first child's
// existing node kind so a CDATA section stays a CDATA section.
func setStyleText(elem *svgast.Element, css string) {
	if len(elem.Children) > 0 {
		switch c := elem.Children[0].(type) {
		case *svgast.Cdata:
			c.Value = css
			elem.Children = elem.Children[:1]
			return
		case *svgast.Text:
			c.Value = css
			elem.Children = elem.Children[:1]
			return
		}
	}
	elem.Children = []svgast.Node{&svgast.Text{Value: css}}
}

// mergeDeclarationsIntoElement merges CSS rule declarations into an element's
// inline style. New declarations are prepended before existing inline styles.
// Existing inline styles take priority unless the CSS rule is !important.
func mergeDeclarationsIntoElement(elem *svgast.Element, ruleDecls []ogcss.StylesheetDeclaration) {
	// Parse existing inline style into ordered list
	existingStyle, _ := elem.Attributes.Get("style")
	existingDecls := ogcss.ParseStyleDeclarations(existingStyle)

	var declList []styledDecl
	declIndex := make(map[string]int) // name → index in declList
	for _, d := range existingDecls {
		declIndex[d.Name] = len(declList)
		declList = append(declList, styledDecl{d.Name, d.Value, d.Important})
	}

	// Merge: prepend new declarations before existing ones
	var prepend []styledDecl
	for _, decl := range ruleDecls {
		if idx, has := declIndex[decl.Name]; has {
			// Property exists in current inline style
			if !declList[idx].important && decl.Important {
				// CSS !important overrides non-!important inline
				declList[idx] = styledDecl{decl.Name, decl.Value, true}
			}
			// Otherwise: existing inline wins (skip)
		} else {
			// New property: prepend before existing inline styles
			prepend = append(prepend, styledDecl{decl.Name, decl.Value, decl.Important})
			declIndex[decl.Name] = -1 // mark as existing
		}
	}

	// Always rebuild the style to normalize format
	result := make([]styledDecl, 0, len(prepend)+len(declList))
	result = append(result, prepend...)
	result = append(result, declList...)

	// Build style string
	var parts []string
	for _, d := range result {
		part := d.name + ":" + d.value
		if d.important {
			part += "!important"
		}
		parts = append(parts, part)
	}
	if len(parts) > 0 {
		elem.Attributes.Set("style", strings.Join(parts, ";"))
	}
}

// cleanupPresentationAttrs removes presentation attributes that are now
// overridden by the inline style, unless they're referenced by remaining CSS selectors.
func cleanupPresentationAttrs(elem *svgast.Element, remainingSelectors []selectorInfo) {
	styleVal, has := elem.Attributes.Get("style")
	if !has {
		return
	}
	decls := ogcss.ParseStyleDeclarations(styleVal)
	for _, decl := range decls {
		propLower := strings.ToLower(decl.Name)
		if collections.PresentationAttrs[propLower] {
			if _, attrHas := elem.Attributes.Get(propLower); attrHas {
				// Don't remove if any remaining selector uses attribute selector for this
				referenced := false
				for _, sel := range remainingSelectors {
					s := sel.selector
					if sel.originalSelector != "" {
						s = sel.originalSelector
					}
					if strings.Contains(s, "["+propLower) {
						referenced = true
						break
					}
				}
				if !referenced {
					elem.Attributes.Delete(propLower)
				}
			}
		}
	}
}

// extractPseudo extracts the pseudo-class from a CSS selector.
// Returns the pseudo-class (e.g., ":hover") or "" if none found.
func extractPseudo(selector string) string {
	for i := 0; i < len(selector); i++ {
		if selector[i] == ':' {
			// Skip :: pseudo-elements
			if i+1 < len(selector) && selector[i+1] == ':' {
				i++ // skip the second colon
				// Skip the pseudo-element name
				for i+1 < len(selector) && selector[i+1] != ' ' && selector[i+1] != ':' &&
					selector[i+1] != '.' && selector[i+1] != '#' && selector[i+1] != '[' &&
					selector[i+1] != '>' && selector[i+1] != '+' && selector[i+1] != '~' {
					i++
				}
				continue
			}
			// Found pseudo-class, extract it
			end := i + 1
			for end < len(selector) && selector[end] != ' ' && selector[end] != ':' &&
				selector[end] != '.' && selector[end] != '#' && selector[end] != '[' &&
				selector[end] != ')' && selector[end] != '>' && selector[end] != '+' &&
				selector[end] != '~' {
				end++
			}
			// Include closing paren if present (e.g., :nth-child(2n))
			if end < len(selector) && selector[end] == ')' {
				end++
			}
			return selector[i:end]
		}
	}
	return ""
}

// cleanupClassAndID drops the class and id attribute values that sel consumed
// while matching elem. Only the sub-selectors of sel itself are considered:
// class names the selector never mentioned survive, and so does anything some
// selector in the sheet still reaches through an attribute selector.
//
// sel.selector is used rather than sel.originalSelector because SVGO walks the
// selector AST after the non-evaluatable pseudo-classes have been spliced out
// of it, which is exactly what selector holds.
func cleanupClassAndID(elem *svgast.Element, sel selectorInfo, allSelectors []selectorInfo) {
	// class
	classAttr, hasClass := elem.Attributes.Get("class")
	classList := newClassList(classAttr, hasClass)

	for _, name := range ogcss.SelectorClassNames(sel.selector) {
		if referencedByAttrSelector(allSelectors, "class", name) {
			continue
		}
		classList.delete(name)
	}

	if classList.len() == 0 {
		elem.Attributes.Delete("class")
	} else {
		elem.Attributes.Set("class", classList.join())
	}

	// id
	if name, ok := ogcss.SelectorLeadingID(sel.selector); ok {
		if id, hasID := elem.Attributes.Get("id"); hasID && id == name &&
			!referencedByAttrSelector(allSelectors, "id", name) {
			elem.Attributes.Delete("id")
		}
	}
}

// referencedByAttrSelector reports whether any collected selector reaches the
// given attribute value through a segment that is followed by a combinator,
// which means removing the value would break that selector.
func referencedByAttrSelector(selectors []selectorInfo, name, value string) bool {
	for _, sel := range selectors {
		if ogcss.SelectorIncludesAttr(sel.selector, name, &value, true) {
			return true
		}
	}
	return false
}

// classList is the ordered set SVGO builds from a class attribute by splitting
// it on single spaces.
type classList struct {
	order []string
	seen  map[string]bool
}

func newClassList(value string, present bool) *classList {
	cl := &classList{seen: make(map[string]bool)}
	if !present {
		return cl
	}
	for _, name := range strings.Split(value, " ") {
		if cl.seen[name] {
			continue
		}
		cl.seen[name] = true
		cl.order = append(cl.order, name)
	}
	return cl
}

func (cl *classList) delete(name string) {
	if !cl.seen[name] {
		return
	}
	delete(cl.seen, name)
	for i, n := range cl.order {
		if n == name {
			cl.order = append(cl.order[:i], cl.order[i+1:]...)
			break
		}
	}
}

func (cl *classList) len() int { return len(cl.order) }

func (cl *classList) join() string { return strings.Join(cl.order, " ") }

// buildParentMap builds a parent map for the entire tree.
func buildParentMap(root *svgast.Root) map[svgast.Node]svgast.Parent {
	parents := make(map[svgast.Node]svgast.Parent)
	var walk func(parent svgast.Parent, children []svgast.Node)
	walk = func(parent svgast.Parent, children []svgast.Node) {
		for _, child := range children {
			parents[child] = parent
			if elem, ok := child.(*svgast.Element); ok {
				walk(elem, elem.Children)
			}
		}
	}
	walk(root, root.Children)
	return parents
}
