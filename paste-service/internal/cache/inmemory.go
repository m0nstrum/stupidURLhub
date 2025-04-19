// лучше использовать redis но пока что так

package cache

import (
	"encoding/json"
	"reflect"
	"sync"
	"time"
)

type entry struct {
	value      interface{}
	expiration int64
}

type InMemoryCache struct {
	mu         sync.RWMutex
	items      map[string]entry
	stopGC     chan struct{}
	refreshTTL bool
}

func NewInMemoryCache(refreshTTL bool) *InMemoryCache {
	cache := &InMemoryCache{
		items:      make(map[string]entry),
		stopGC:     make(chan struct{}),
		refreshTTL: refreshTTL,
	}
	go cache.startGC(1 * time.Minute) // покрутить время позже
	return cache
}

func (c *InMemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	ent, exists := c.items[key]

	if !exists || (ent.expiration > 0 && time.Now().UnixNano() > ent.expiration) {
		c.mu.RUnlock()
		return nil, false
	}

	if c.refreshTTL && ent.expiration > 0 {
		c.mu.RUnlock()
		c.mu.Lock()

		if newEnt, stillExists := c.items[key]; stillExists {
			ttl := time.Duration(newEnt.expiration - time.Now().UnixNano())
			if ttl > 0 {
				c.items[key] = entry{
					value:      newEnt.value,
					expiration: time.Now().Add(ttl).UnixNano(),
				}
			}
			c.mu.Unlock()
			return newEnt.value, true
		}
		c.mu.Unlock()
		return nil, false
	}

	c.mu.RUnlock()
	return ent.value, true
}

func (c *InMemoryCache) GetTyped(key string, result interface{}) bool {
	value, ok := c.Get(key)
	if !ok {
		return false
	}

	resultVal := reflect.ValueOf(result)
	if resultVal.Kind() != reflect.Ptr || resultVal.IsNil() {
		return false
	}

	resultElem := resultVal.Elem()
	sourceVal := reflect.ValueOf(value)

	if resultElem.Type().AssignableTo(sourceVal.Type()) {
		resultElem.Set(sourceVal)
		return true
	}

	data, err := json.Marshal(value)
	if err != nil {
		return false
	}

	if err := json.Unmarshal(data, result); err != nil {
		return false
	}

	return true
}

func (c *InMemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	var expire int64
	if ttl > 0 {
		expire = time.Now().Add(ttl).UnixNano()
	}
	c.mu.Lock()
	c.items[key] = entry{value: value, expiration: expire}
	c.mu.Unlock()
}

func (c *InMemoryCache) Invalidate(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

func (c *InMemoryCache) Clear() {
	c.mu.Lock()
	c.items = make(map[string]entry)
	c.mu.Unlock()
}

func (c *InMemoryCache) Stop() {
	close(c.stopGC)
}

func (c *InMemoryCache) startGC(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now().UnixNano()
			c.mu.Lock()
			for k, v := range c.items {
				if v.expiration > 0 && now > v.expiration {
					delete(c.items, k)
				}
			}
			c.mu.Unlock()
		case <-c.stopGC:
			return
		}
	}
}
