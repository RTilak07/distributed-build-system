package main

import (
	"sync"

	"distributed-build-system/shared"
)

type BuildCache struct {
	entries map[string]shared.BuildResult
	mu      sync.RWMutex
}

func NewBuildCache() *BuildCache {
	return &BuildCache{
		entries: make(map[string]shared.BuildResult),
	}
}

func (c *BuildCache) Get(key string) (shared.BuildResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result, exists := c.entries[key]

	return result, exists
}

func (c *BuildCache) Set(key string, result shared.BuildResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = result
}

func (c *BuildCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}
