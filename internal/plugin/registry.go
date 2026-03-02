// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package plugin

import (
	"fmt"
	"sync"
)

// registry is the global plugin registry mapping names to Plugin definitions.
var (
	registryMu sync.RWMutex
	registry   = make(map[string]*Plugin)
)

// Register adds a plugin to the global registry.
// Panics if a plugin with the same name is already registered.
func Register(p *Plugin) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[p.Name]; exists {
		panic(fmt.Sprintf("plugin %q already registered", p.Name))
	}
	registry[p.Name] = p
}

// Get returns a registered plugin by name, or nil if not found.
func Get(name string) *Plugin {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[name]
}

// Has checks if a plugin is registered.
func Has(name string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, exists := registry[name]
	return exists
}

// Names returns all registered plugin names in no particular order.
func Names() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// resetRegistry clears the registry (for testing only).
func resetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = make(map[string]*Plugin)
}
