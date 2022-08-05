// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/LICENSE.

package xcache

import (
	"sync"

	"github.com/actforgood/xconf"
	"github.com/coocood/freecache"
)

const (
	// MemoryCfgKeyMemorySize is the key under which xconf.Config expects memory size in bytes.
	MemoryCfgKeyMemorySize      = "xcache.memory.memsizebytes"
	memoryCfgDefValueMemorySize = 10 * 1024 * 1024 // 10 Mb
)

// NewMemoryWithConfig initializes a Memory Cache with memory size taken from a xconf.Config.
//
// The key under which memory size is expected to be found is "xcache.memory.memsizebytes"
// (note, you can have a different config key defined in your project, you'll have to create an alias
// for it to expected "xcache.memory.memsizebytes").
// If "xcache.memory.memsizebytes" config key is not found, a default value of 10M is used.
//
// An observer is registered to xconf.DefaultConfig (which knows to reload configuration).
// In case "xcache.memory.memsizebytes" config is changed, the Memory is reinitialized with the new memory size,
// and all items from old freecache instance are copied to the new one. Note: host machine/container needs to have
// additional to current occupied memory, the new memory size available (until old memory is garbage collected, old memory size is still occupied).
func NewMemoryWithConfig(config xconf.Config) *Memory {
	mem := config.Get(MemoryCfgKeyMemorySize, memoryCfgDefValueMemorySize).(int)

	cache := NewMemory(mem)
	cache.mu = new(sync.RWMutex)

	if defConfig, ok := config.(*xconf.DefaultConfig); ok {
		defConfig.RegisterObserver(cache.onConfigChange)
	}

	return cache
}

// onConfigChange is a callback to be registered to xconf.DefaultConfig that knows to reload configuration.
// In case "xcache.memory.memsizebytes" config is changed, the Memory is reinitialized with the new memory size,
// and all items from old freecache instance are copied to the new one.
// This callback is automatically registered on instantiation of a Memory object with NewMemoryWithConfig.
func (cache *Memory) onConfigChange(config xconf.Config, changedKeys ...string) {
	memSize := 0
	for _, changedKey := range changedKeys {
		if changedKey == MemoryCfgKeyMemorySize {
			memSize = config.Get(MemoryCfgKeyMemorySize, memoryCfgDefValueMemorySize).(int)
			memSize = getRealMemorySize(memSize)

			break
		}
	}
	if memSize == 0 {
		return
	}

	cache.mu.Lock()
	if memSize != int(cache.memSize) {
		// note 1: stats will be reset on the new client.
		// note 2: during this code execution memory occupied will be oldMemorySize + newMemorySize so machine needs to have to this memory available.
		// note 3: not tested performance if a large number of keys needs to be copied.

		newClient := freecache.NewCache(memSize)
		oldClient := cache.client

		// copy old cache items in new cache
		iter := oldClient.NewIterator()
		for {
			entry := iter.Next()
			if entry == nil {
				break
			}
			if ttl, err := oldClient.TTL(entry.Key); err == nil {
				_ = newClient.Set(entry.Key, entry.Value, int(ttl))
			}
		}
		cache.client = newClient
		cache.memSize = int64(memSize)
	}
	cache.mu.Unlock()
}
