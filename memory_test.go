// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/LICENSE.

package xcache_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/actforgood/xcache"
)

const (
	freecacheMinMem = 512 * 1024       // 512 Kb
	memoryBenchSize = 10 * 1024 * 1024 // 10 Mb
)

func init() {
	var _ xcache.Cache = (*xcache.Memory)(nil) // test Memory is a Cache
}

func TestMemory(t *testing.T) {
	t.Parallel()

	subject := xcache.NewMemory(1)

	t.Run("key that does not expire", testCacheWithNoExpireKey(subject))
	t.Run("key expires", testCacheWithExpireKey(subject))
	t.Run("key does not exist", testCacheWithNotExistKey(subject))
	t.Run("delete key", testCacheDeleteKey(subject))
	t.Run("ttl for not yet expired key", testCacheTTLWithNotYetExpiredKey(subject))
	t.Run("stats", testCacheStats(subject, freecacheMinMem, freecacheMinMem, "==", true))
}

func BenchmarkMemory_Save(b *testing.B) {
	cache := xcache.NewMemory(memoryBenchSize)
	benchSaveSequential(cache)(b)

	b.StopTimer()
	stats, _ := cache.Stats(context.Background())
	b.Log(stats)
}

func BenchmarkMemory_Save_parallel(b *testing.B) {
	cache := xcache.NewMemory(memoryBenchSize)
	benchSaveParallel(cache)(b)

	b.StopTimer()
	stats, _ := cache.Stats(context.Background())
	b.Log(stats)
}

func BenchmarkMemory_Load(b *testing.B) {
	cache := xcache.NewMemory(memoryBenchSize)
	benchLoadSequential(cache)(b)

	b.StopTimer()
	stats, _ := cache.Stats(context.Background())
	b.Log(stats)
}

func BenchmarkMemory_Load_parallel(b *testing.B) {
	cache := xcache.NewMemory(memoryBenchSize)
	benchLoadParallel(cache)(b)

	b.StopTimer()
	stats, _ := cache.Stats(context.Background())
	b.Log(stats)
}

func BenchmarkMemory_TTL(b *testing.B) {
	cache := xcache.NewMemory(memoryBenchSize)
	benchTTLSequential(cache)(b)

	b.StopTimer()
	stats, _ := cache.Stats(context.Background())
	b.Log(stats)
}

func BenchmarkMemory_TTL_parallel(b *testing.B) {
	cache := xcache.NewMemory(memoryBenchSize)
	benchTTLParallel(cache)(b)

	b.StopTimer()
	stats, _ := cache.Stats(context.Background())
	b.Log(stats)
}

func BenchmarkMemory_Stats(b *testing.B) {
	cache := xcache.NewMemory(memoryBenchSize)
	benchStatsSequential(cache)(b)
}

func BenchmarkMemory_Stats_parallel(b *testing.B) {
	cache := xcache.NewMemory(memoryBenchSize)
	benchStatsParallel(cache)(b)
}

func ExampleMemory() {
	cache := xcache.NewMemory(10 * 1024 * 1024) // 10 Mb

	ctx := context.Background()
	key := "example-memory"
	value := []byte("Hello Memory Cache")
	ttl := 10 * time.Minute

	// save a key for 10 minutes
	if err := cache.Save(ctx, key, value, ttl); err != nil {
		fmt.Println(err)
	}

	// load the key's value
	if value, err := cache.Load(ctx, key); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(string(value))
	}

	// Output:
	// Hello Memory Cache
}
