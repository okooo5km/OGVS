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
		for _, child := range children {
			Visit(child, visitor, n)
		}

	case *Element:
		// Only visit children if element is still attached to parent
		if parent != nil && nodeInChildren(node, parent.GetChildren()) {
			children := make([]Node, len(n.Children))
			copy(children, n.Children)
			for _, child := range children {
				Visit(child, visitor, n)
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

// nodeInChildren checks if a node is still in the parent's children list.
func nodeInChildren(node Node, children []Node) bool {
	return slices.Contains(children, node)
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
