//go:build integration
// +build integration

// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/blob/main/LICENSE.

package xcache_test

import (
	"context"
	"testing"

	"github.com/actforgood/xcache"
)

func BenchmarkMulti_Save_integration(b *testing.B) {
	cache1 := xcache.NewMemory(memoryBenchSize)
	cache2 := xcache.NewRedis6(redis6ConfigIntegration)
	cache := xcache.NewMulti(cache1, cache2)
	benchSaveSequential(cache)(b)

	b.StopTimer()
	stats, err := cache.Stats(context.Background())
	if err != nil {
		b.Error(err)
	}
	b.Log(stats)
	if err := cache2.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkMulti_Save_parallel_integration(b *testing.B) {
	cache1 := xcache.NewMemory(memoryBenchSize)
	cache2 := xcache.NewRedis6(redis6ConfigIntegration)
	cache := xcache.NewMulti(cache1, cache2)
	benchSaveParallel(cache)(b)

	b.StopTimer()
	stats, err := cache.Stats(context.Background())
	if err != nil {
		b.Error(err)
	}
	b.Log(stats)
	if err := cache2.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkMulti_Load_integration(b *testing.B) {
	cache1 := xcache.NewMemory(memoryBenchSize)
	cache2 := xcache.NewRedis6(redis6ConfigIntegration)
	cache := xcache.NewMulti(cache1, cache2)
	benchLoadSequential(cache)(b)

	b.StopTimer()
	stats, err := cache.Stats(context.Background())
	if err != nil {
		b.Error(err)
	}
	b.Log(stats)
	if err := cache2.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkMulti_Load_parallel_integration(b *testing.B) {
	cache1 := xcache.NewMemory(memoryBenchSize)
	cache2 := xcache.NewRedis6(redis6ConfigIntegration)
	cache := xcache.NewMulti(cache1, cache2)
	benchLoadParallel(cache)(b)

	b.StopTimer()
	stats, err := cache.Stats(context.Background())
	if err != nil {
		b.Error(err)
	}
	b.Log(stats)
	if err := cache2.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkMulti_TTL_integration(b *testing.B) {
	cache1 := xcache.NewMemory(memoryBenchSize)
	cache2 := xcache.NewRedis6(redis6ConfigIntegration)
	cache := xcache.NewMulti(cache1, cache2)
	benchTTLSequential(cache)(b)

	b.StopTimer()
	stats, err := cache.Stats(context.Background())
	if err != nil {
		b.Error(err)
	}
	b.Log(stats)
	if err := cache2.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkMulti_TTL_parallel_integration(b *testing.B) {
	cache1 := xcache.NewMemory(memoryBenchSize)
	cache2 := xcache.NewRedis6(redis6ConfigIntegration)
	cache := xcache.NewMulti(cache1, cache2)
	benchTTLParallel(cache)(b)

	b.StopTimer()
	stats, err := cache.Stats(context.Background())
	if err != nil {
		b.Error(err)
	}
	b.Log(stats)
	if err := cache2.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkMulti_Stats(b *testing.B) {
	cache1 := xcache.NewMemory(memoryBenchSize)
	cache2 := xcache.NewRedis6(redis6ConfigIntegration)
	cache := xcache.NewMulti(cache1, cache2)
	benchStatsSequential(cache)(b)

	b.StopTimer()
	if err := cache2.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkMulti_Stats_parallel(b *testing.B) {
	cache1 := xcache.NewMemory(memoryBenchSize)
	cache2 := xcache.NewRedis6(redis6ConfigIntegration)
	cache := xcache.NewMulti(cache1, cache2)
	benchStatsParallel(cache)(b)

	b.StopTimer()
	if err := cache2.Close(); err != nil {
		b.Error(err)
	}
}
