package pricing

import "sync"

// CachedProvider wraps a PricingProvider with a read-through cache for Find().
type CachedProvider struct {
	inner PricingProvider
	mu    sync.RWMutex
	cache map[string]*Pricing
}

// NewCachedProvider creates a caching wrapper around a PricingProvider.
func NewCachedProvider(inner PricingProvider) *CachedProvider {
	return &CachedProvider{
		inner: inner,
		cache: make(map[string]*Pricing),
	}
}

// Find looks up a model, caching the result.
func (c *CachedProvider) Find(model string) *Pricing {
	c.mu.RLock()
	if p, ok := c.cache[model]; ok {
		c.mu.RUnlock()
		return p
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	// Double-check after acquiring write lock.
	if p, ok := c.cache[model]; ok {
		return p
	}
	p := c.inner.Find(model)
	c.cache[model] = p
	return p
}
