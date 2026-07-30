// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package svgast

import (
	"errors"
	"slices"
)

// ErrVisitSkip is returned from enter callbacks to skip visiting children.
// The exit callback is still called.
var ErrVisitSkip = errors.New("visitSkip")

// Visitor defines callbacks for each node type during AST traversal.
// All fields are optional — nil callbacks are skipped.
type Visitor struct {
	Root        *VisitorCallbacks
	Element     *VisitorCallbacks
	Text        *VisitorCallbacks
	Comment     *VisitorCallbacks
	Cdata       *VisitorCallbacks
	Instruction *VisitorCallbacks
	Doctype     *VisitorCallbacks
}

// VisitorCallbacks holds enter/exit callbacks for a node type.
type VisitorCallbacks struct {
	// Enter is called before visiting children.
	// Return ErrVisitSkip to skip children (exit is still called).
	// The node parameter is the concrete node type (*Element, *Text, etc.)
	Enter func(node Node, parent Parent) error

	// Exit is called after visiting children (or after skip).
	Exit func(node Node, parent Parent)
}

// Visit performs a depth-first traversal of the AST, calling visitor callbacks.
//
// This matches SVGO's visit() behavior:
//   - enter callbacks are called before children
//   - returning ErrVisitSkip skips children but still calls exit
//   - for root nodes, all children are visited
//   - for element nodes, children are only visited if the element is still
//     attached to its parent (enables safe removal during traversal)
//   - exit callbacks are always called
func Visit(node Node, visitor *Visitor, parent Parent) {
	var live *liveChildren
	if parent != nil {
		live = &liveChildren{parent: parent}
	}
	visitNode(node, visitor, parent, live)
}

// liveChildren answers "is this node still one of the parent's children" in
// O(1) instead of scanning the sibling list on every child. The scan made a
// flat document with k children cost O(k^2) per plugin.
//
// The traversal walks a snapshot of the children in order, and every mutation
// plugins perform through DetachNodeFromParent (or an equivalent filter)
// preserves the relative order of the survivors. So the node being asked about
// is normally sitting at cursor: a survivor advances the cursor, a detached
// node leaves it pointing at whatever shifted into its place. Anything the
// cursor cannot model — an insertion, a reorder — flips the tracker into
// degraded mode, where it falls back to a linear scan.
type liveChildren struct {
	parent   Parent
	cursor   int // index in the live slice where the next surviving child must sit
	expected int // len(parent.GetChildren()) at the previous query
	primed   bool
	degraded bool
}

// contains reports whether node is currently among the parent's children.
func (l *liveChildren) contains(node Node) bool {
	if l == nil || l.parent == nil {
		return false
	}
	children := l.parent.GetChildren()
	if !l.primed {
		l.primed = true
		l.expected = len(children)
	}
	if !l.degraded && len(children) > l.expected {
		// Children were inserted; the cursor no longer tracks the snapshot.
		l.degraded = true
	}
	if l.degraded {
		l.expected = len(children)
		return slices.Contains(children, node)
	}
	if l.cursor < len(children) && children[l.cursor] == node {
		l.cursor++
		l.expected = len(children)
		return true
	}
	l.expected = len(children)
	// Not where it was expected. Normally that means the enter callback
	// detached it; confirm with one scan and drop to the scanning path if the
	// children turn out to have been rearranged instead.
	if slices.Contains(children, node) {
		l.degraded = true
		return true
	}
	return false
}

func visitNode(node Node, visitor *Visitor, parent Parent, attached *liveChildren) {
	callbacks := getCallbacks(node, visitor)

	// Enter phase
	if callbacks != nil && callbacks.Enter != nil {
		err := callbacks.Enter(node, parent)
		if errors.Is(err, ErrVisitSkip) {
			// Skip children but still call exit
			if callbacks.Exit != nil {
				callbacks.Exit(node, parent)
			}
			return
		}
	}

	// Visit children
	switch n := node.(type) {
	case *Root:
		// Copy children slice to handle modifications during iteration
		children := make([]Node, len(n.Children))
		copy(children, n.Children)
		live := &liveChildren{parent: n}
		for _, child := range children {
			visitNode(child, visitor, n, live)
		}

	case *Element:
		// Only visit children if element is still attached to parent
		if parent != nil && attached.contains(node) {
			children := make([]Node, len(n.Children))
			copy(children, n.Children)
			live := &liveChildren{parent: n}
			for _, child := range children {
				visitNode(child, visitor, n, live)
			}
		}
	}

	// Exit phase
	if callbacks != nil && callbacks.Exit != nil {
		callbacks.Exit(node, parent)
	}
}

// getCallbacks returns the visitor callbacks for a given node type.
func getCallbacks(node Node, visitor *Visitor) *VisitorCallbacks {
	switch node.Type() {
	case NodeRoot:
		return visitor.Root
	case NodeElement:
		return visitor.Element
	case NodeText:
		return visitor.Text
	case NodeComment:
		return visitor.Comment
	case NodeCdata:
		return visitor.Cdata
	case NodeInstruction:
		return visitor.Instruction
	case NodeDoctype:
		return visitor.Doctype
	default:
		return nil
	}
}

// DetachNodeFromParent removes a node from its parent's children list.
// Uses filter (not splice) to avoid breaking for-loops, matching SVGO behavior.
func DetachNodeFromParent(node Node, parent Parent) {
	children := parent.GetChildren()
	filtered := make([]Node, 0, len(children))
	for _, child := range children {
		if child != node {
			filtered = append(filtered, child)
		}
	}
	parent.SetChildren(filtered)
}
