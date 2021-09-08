package cache

import "sync"

type Cacher interface {
	Get(key interface{}) (value []byte, exists bool)
	Set(key interface{}, value []byte)
	Len() int
}

type FifoCache struct {
	head, tail *fifoCacheEntry
	data       map[interface{}][]byte
	capacity   int
	mutex      sync.RWMutex
}

type fifoCacheEntry struct {
	Key  interface{}
	Next *fifoCacheEntry
}

func NewFifoCache(capacity int) *FifoCache {
	return &FifoCache{data: make(map[interface{}][]byte, capacity), capacity: capacity}
}

func (c *FifoCache) Get(key interface{}) (value []byte, exists bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	value, exists = c.data[key]
	return
}

func (c *FifoCache) Set(key interface{}, value []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if len(c.data) == c.capacity {
		// Evict first entry
		evicted := c.head
		c.head = evicted.Next
		delete(c.data, evicted.Key)
	}
	entry := &fifoCacheEntry{Key: key}
	if c.head == nil {
		c.head = entry
	}
	if c.tail == nil {
		c.tail = entry
	} else {
		c.tail.Next = entry
		c.tail = entry
	}
	c.data[key] = value
}

func (c *FifoCache) Len() int {
	return len(c.data)
}
