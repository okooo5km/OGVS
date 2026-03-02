// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package svgast provides the XAST (XML Abstract Syntax Tree) data model,
// parser, visitor, and stringifier for SVG documents.
//
// The data model mirrors SVGO's xast types with 7 node types:
// Root, Element, Text, Comment, Cdata, Instruction, Doctype.
package svgast

// NodeType identifies the kind of AST node.
type NodeType int

const (
	NodeRoot        NodeType = iota // document root
	NodeElement                     // SVG/XML element
	NodeText                        // text content
	NodeComment                     // XML comment
	NodeCdata                       // CDATA section
	NodeInstruction                 // processing instruction (<?...?>)
	NodeDoctype                     // DOCTYPE declaration
)

// String returns the SVGO-compatible type name.
func (t NodeType) String() string {
	switch t {
	case NodeRoot:
		return "root"
	case NodeElement:
		return "element"
	case NodeText:
		return "text"
	case NodeComment:
		return "comment"
	case NodeCdata:
		return "cdata"
	case NodeInstruction:
		return "instruction"
	case NodeDoctype:
		return "doctype"
	default:
		return "unknown"
	}
}

// Node is the common interface for all AST nodes.
type Node interface {
	Type() NodeType
}

// Parent is the interface for nodes that can have children (Root, Element).
type Parent interface {
	Node
	GetChildren() []Node
	SetChildren([]Node)
}

// UndefinedAttrValue is a sentinel indicating an attribute has no value.
// Attributes with this value are stringified as just the name (e.g., "data-icon")
// rather than with an empty value (e.g., data-icon=""). This matches SVGO's
// behavior where attr.value === undefined.
const UndefinedAttrValue = "\x00"

// --- Concrete node types ---

// Root is the top-level document node.
type Root struct {
	Children []Node
}

func (n *Root) Type() NodeType       { return NodeRoot }
func (n *Root) GetChildren() []Node  { return n.Children }
func (n *Root) SetChildren(c []Node) { n.Children = c }

// Element represents an SVG/XML element with a name, ordered attributes, and children.
type Element struct {
	Name       string
	Attributes *OrderedAttrs
	Children   []Node
}

func (n *Element) Type() NodeType       { return NodeElement }
func (n *Element) GetChildren() []Node  { return n.Children }
func (n *Element) SetChildren(c []Node) { n.Children = c }

// Text represents text content inside an element.
type Text struct {
	Value string
}

func (n *Text) Type() NodeType { return NodeText }

// Comment represents an XML comment (<!-- ... -->).
type Comment struct {
	Value string
}

func (n *Comment) Type() NodeType { return NodeComment }

// Cdata represents a CDATA section (<![CDATA[ ... ]]>).
type Cdata struct {
	Value string
}

func (n *Cdata) Type() NodeType { return NodeCdata }

// Instruction represents a processing instruction (<?name value?>).
type Instruction struct {
	Name  string
	Value string
}

func (n *Instruction) Type() NodeType { return NodeInstruction }

// Doctype represents a DOCTYPE declaration.
type Doctype struct {
	Name string
	Data DoctypeData
}

// DoctypeData holds the raw doctype string.
type DoctypeData struct {
	Doctype string
}

func (n *Doctype) Type() NodeType { return NodeDoctype }
