// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/blob/main/LICENSE.

package xcache

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/coocood/freecache"
)

const freecacheMinBufSize = 512 * 1024

// Memory is an in memory implementation for Cache.
// It is not distributed, keys are stored in memory,
// only for current instance.
// It relies upon Freecache package.
type Memory struct {
	client  *freecache.Cache
	memSize int64         // memory size in bytes
	mu      *sync.RWMutex // concurrency semaphore used for xconf adapter.
}

// NewMemory initializes a new Memory instance.
//
// Relaying package additional notes:
// The cache size will be set to 512KB at minimum.
// If the size is set relatively large, you should call
// [runtime/debug.SetGCPercent], set it to a much smaller value
// to limit the memory consumption and GC pause time.
func NewMemory(memSize int) *Memory {
	mem := getRealMemorySize(memSize)
	client := freecache.NewCache(mem)

	return &Memory{
		client:  client,
		memSize: int64(mem),
	}
}

// Save stores the given key-value with expiration period into cache.
// An expiration period equal to 0 (NoExpire) means no expiration.
// A negative expiration period triggers deletion of key.
// It returns an error if the key could not be saved.
//
// Additional relaying package notes:
// If the key is larger than 65535 or value is larger than 1/1024 of the cache size,
// the entry will not be written to the cache.
// Items can be evicted when cache is full.
func (cache *Memory) Save(
	_ context.Context,
	key string,
	value []byte,
	expire time.Duration,
) error {
	if expire < 0 { // delete the key
		cache.rLock()
		_ = cache.client.Del([]byte(key))
		cache.rUnlock()

		return nil
	}
	expireSeconds := int(expire.Seconds())
	if expire > 0 && expireSeconds == 0 {
		// convert expire < 1s to 1s as Freecache expects seconds, and 0 means no expiration.
		// highly improbable to enter here, as items are usually cached for longer periods.
		expireSeconds = 1
	}

	cache.rLock()
	err := cache.client.Set([]byte(key), value, expireSeconds)
	cache.rUnlock()

	return err
}

// Load returns a key's value from cache, or an error if something bad happened.
// If the key is not found, ErrNotFound is returned.
func (cache *Memory) Load(_ context.Context, key string) ([]byte, error) {
	cache.rLock()
	value, err := cache.client.Get([]byte(key))
	cache.rUnlock()

	if errors.Is(err, freecache.ErrNotFound) {
		return nil, ErrNotFound
	}

	return value, err
}

// TTL returns a key's remaining time to live. Error is always nil.
// If the key is not found, a negative TTL is returned.
// If the key has no expiration, 0 (NoExpire) is returned.
func (cache *Memory) TTL(_ context.Context, key string) (time.Duration, error) {
	cache.rLock()
	ttl, err := cache.client.TTL([]byte(key))
	cache.rUnlock()

	if errors.Is(err, freecache.ErrNotFound) {
		return -1, nil
	}

	return time.Duration(ttl), err
}

// Stats returns statistics about memory cache.
// Returned error is always nil and can be safely disregarded.
func (cache *Memory) Stats(_ context.Context) (Stats, error) {
	cache.rLock()
	stats := Stats{
		Memory:    cache.memSize,
		MaxMemory: cache.memSize,
		Hits:      cache.client.HitCount(),
		Misses:    cache.client.MissCount(),
		Keys:      cache.client.EntryCount(),
		Expired:   cache.client.ExpiredCount(),
		Evicted:   cache.client.EvacuateCount(),
	}
	cache.rUnlock()

	return stats, nil
}

func (cache *Memory) rLock() {
	if cache.mu != nil {
		cache.mu.RLock()
	}
}

func (cache *Memory) rUnlock() {
	if cache.mu != nil {
		cache.mu.RUnlock()
	}
}

// getRealMemorySize returns memory according to Freecache min limit (512 Kb).
func getRealMemorySize(memSize int) int {
	mem := memSize
	if mem < freecacheMinBufSize {
		mem = freecacheMinBufSize
	}

	return mem
}
