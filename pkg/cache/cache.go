package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"digital.vasic.formatters/pkg/formatter"
)

// FormatCache is the interface for format result caching.
type FormatCache interface {
	Get(req *formatter.FormatRequest) (*formatter.FormatResult, bool)
	Set(req *formatter.FormatRequest, result *formatter.FormatResult)
	Invalidate(req *formatter.FormatRequest)
}

// Config configures the cache.
type Config struct {
	MaxEntries  int           // Maximum number of cache entries
	TTL         time.Duration // Time to live for cache entries
	CleanupFreq time.Duration // Cleanup frequency
}

// DefaultCacheConfig returns a default cache configuration.
func DefaultCacheConfig() Config {
	return Config{
		MaxEntries:  10000,
		TTL:         1 * time.Hour,
		CleanupFreq: 5 * time.Minute,
	}
}

// cacheEntry represents a cached result.
type cacheEntry struct {
	result    *formatter.FormatResult
	timestamp time.Time
}

// InMemoryCache is an in-memory implementation of FormatCache.
type InMemoryCache struct {
	mu          sync.RWMutex
	cache       map[string]*cacheEntry
	config      Config
	stopCleanup chan struct{}
}

// NewInMemoryCache creates a new in-memory cache.
func NewInMemoryCache(config Config) *InMemoryCache {
	c := &InMemoryCache{
		cache:       make(map[string]*cacheEntry),
		config:      config,
		stopCleanup: make(chan struct{}),
	}

	go c.cleanupLoop()

	return c
}

// Get retrieves a cached result.
func (c *InMemoryCache) Get(
	req *formatter.FormatRequest,
) (*formatter.FormatResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := cacheKey(req)
	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	if time.Since(entry.timestamp) > c.config.TTL {
		return nil, false
	}

	return entry.result, true
}

// Set stores a result in the cache.
func (c *InMemoryCache) Set(
	req *formatter.FormatRequest,
	result *formatter.FormatResult,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cache) >= c.config.MaxEntries {
		c.evictOldest()
	}

	key := cacheKey(req)
	c.cache[key] = &cacheEntry{
		result:    result,
		timestamp: time.Now(),
	}
}

// Invalidate removes a specific entry from the cache.
func (c *InMemoryCache) Invalidate(req *formatter.FormatRequest) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := cacheKey(req)
	delete(c.cache, key)
}

// Clear clears the entire cache.
func (c *InMemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*cacheEntry)
}

// Size returns the number of cached entries.
func (c *InMemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}

// Stop stops the cleanup goroutine.
func (c *InMemoryCache) Stop() {
	close(c.stopCleanup)
}

// Stats returns cache statistics.
func (c *InMemoryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Size:       len(c.cache),
		MaxEntries: c.config.MaxEntries,
		TTL:        c.config.TTL,
	}
}

// CacheStats provides cache statistics.
type CacheStats struct {
	Size       int
	MaxEntries int
	TTL        time.Duration
}

// cacheKey generates a cache key for a request.
func cacheKey(req *formatter.FormatRequest) string {
	h := sha256.New()
	h.Write([]byte(req.Content))
	h.Write([]byte(req.Language))
	h.Write([]byte(req.FilePath))
	return hex.EncodeToString(h.Sum(nil))
}

// evictOldest removes the oldest cache entry.
func (c *InMemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.cache {
		if oldestKey == "" || entry.timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.timestamp
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
	}
}

// cleanupLoop periodically removes expired entries.
func (c *InMemoryCache) cleanupLoop() {
	ticker := time.NewTicker(c.config.CleanupFreq)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// cleanup removes expired entries.
func (c *InMemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expired := make([]string, 0)

	for key, entry := range c.cache {
		if now.Sub(entry.timestamp) > c.config.TTL {
			expired = append(expired, key)
		}
	}

	for _, key := range expired {
		delete(c.cache, key)
	}
}
