// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package svgast

// AttrEntry represents a single attribute key-value pair.
type AttrEntry struct {
	Name  string
	Value string
}

// OrderedAttrs maintains attributes in insertion order,
// matching JavaScript object property iteration semantics.
//
// Go maps do not preserve insertion order, so this custom type
// uses a slice for ordering and a map for O(1) lookups.
type OrderedAttrs struct {
	entries []AttrEntry
	index   map[string]int // name → position in entries
}

// NewOrderedAttrs creates an empty OrderedAttrs.
func NewOrderedAttrs() *OrderedAttrs {
	return &OrderedAttrs{
		index: make(map[string]int),
	}
}

// NewOrderedAttrsFromEntries creates an OrderedAttrs from a slice of entries.
func NewOrderedAttrsFromEntries(entries []AttrEntry) *OrderedAttrs {
	oa := &OrderedAttrs{
		entries: make([]AttrEntry, len(entries)),
		index:   make(map[string]int, len(entries)),
	}
	copy(oa.entries, entries)
	for i, e := range oa.entries {
		oa.index[e.Name] = i
	}
	return oa
}

// Get returns the value for a given attribute name and whether it exists.
func (oa *OrderedAttrs) Get(name string) (string, bool) {
	if idx, ok := oa.index[name]; ok {
		return oa.entries[idx].Value, true
	}
	return "", false
}

// Set adds or updates an attribute. New attributes are appended at the end.
func (oa *OrderedAttrs) Set(name, value string) {
	if idx, ok := oa.index[name]; ok {
		oa.entries[idx].Value = value
		return
	}
	oa.index[name] = len(oa.entries)
	oa.entries = append(oa.entries, AttrEntry{Name: name, Value: value})
}

// Delete removes an attribute by name.
func (oa *OrderedAttrs) Delete(name string) {
	idx, ok := oa.index[name]
	if !ok {
		return
	}
	// Remove from slice
	oa.entries = append(oa.entries[:idx], oa.entries[idx+1:]...)
	// Rebuild index
	delete(oa.index, name)
	for i := idx; i < len(oa.entries); i++ {
		oa.index[oa.entries[i].Name] = i
	}
}

// Has returns true if the attribute exists.
func (oa *OrderedAttrs) Has(name string) bool {
	_, ok := oa.index[name]
	return ok
}

// Len returns the number of attributes.
func (oa *OrderedAttrs) Len() int {
	return len(oa.entries)
}

// Entries returns all attributes in insertion order.
func (oa *OrderedAttrs) Entries() []AttrEntry {
	result := make([]AttrEntry, len(oa.entries))
	copy(result, oa.entries)
	return result
}

// SetEntries replaces all attributes with the given entries.
// Used by plugins like sortAttrs that need to reorder attributes.
func (oa *OrderedAttrs) SetEntries(entries []AttrEntry) {
	oa.entries = make([]AttrEntry, len(entries))
	copy(oa.entries, entries)
	oa.index = make(map[string]int, len(entries))
	for i, e := range oa.entries {
		oa.index[e.Name] = i
	}
}

// Clone returns a deep copy of the OrderedAttrs.
func (oa *OrderedAttrs) Clone() *OrderedAttrs {
	return NewOrderedAttrsFromEntries(oa.entries)
}
