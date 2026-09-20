package api

import "sync"

const defaultCacheCapacity = 256

type answerCache struct {
	mu       sync.Mutex
	capacity int
	values   map[string]string
	order    []string
}

func newAnswerCache(capacity int) *answerCache {
	return &answerCache{
		capacity: capacity,
		values:   make(map[string]string),
	}
}

func (c *answerCache) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, ok := c.values[key]
	if !ok {
		return "", false
	}
	c.touch(key)
	return value, true
}

func (c *answerCache) set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.values[key]; exists {
		c.values[key] = value
		c.touch(key)
		return
	}
	if len(c.values) >= c.capacity {
		evicted := c.order[0]
		c.order = c.order[1:]
		delete(c.values, evicted)
	}
	c.values[key] = value
	c.order = append(c.order, key)
}

func (c *answerCache) touch(key string) {
	for index, current := range c.order {
		if current == key {
			c.order = append(c.order[:index], c.order[index+1:]...)
			break
		}
	}
	c.order = append(c.order, key)
}
