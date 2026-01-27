package cache

import (
	"sync"
	"time"
)

// Entry represents a cached value with expiration
type Entry struct {
	Value     interface{}
	ExpiresAt time.Time
}

// Cache is a simple in-memory cache with TTL
type Cache struct {
	mu    sync.RWMutex
	items map[string]*Entry
}

// New creates a new cache
func New() *Cache {
	return &Cache{items: map[string]*Entry{}}
}

// Set stores a value in the cache with a given TTL
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = &Entry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Get retrieves a value from the cache if it hasn't expired
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, exists := c.items[key]
	if !exists {
		return nil, false
	}
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	return entry.Value, true
}

// Delete removes a key from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = map[string]*Entry{}
}

// Invalidate removes all items matching a prefix
func (c *Cache) Invalidate(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.items, key)
		}
	}
}
