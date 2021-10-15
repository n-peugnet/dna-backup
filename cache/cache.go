/* Copyright (C) 2021 Nicolas Peugnet <n.peugnet@free.fr>

   This file is part of dna-backup.

   dna-backup is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   dna-backup is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with dna-backup.  If not, see <https://www.gnu.org/licenses/>. */

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
