//go:build integration
// +build integration

// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/LICENSE.

package xcache_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/actforgood/xcache"
	"github.com/actforgood/xlog"
)

var redis7ConfigIntegration = xcache.RedisConfig{}

func init() {
	redisAddrs := os.Getenv("XCACHE_REDIS7_ADDRS")
	redisMasterName := os.Getenv("XCACHE_REDIS7_MASTER_NAME")
	if redisAddrs != "" {
		addrs := strings.Split(redisAddrs, ",")
		redis7ConfigIntegration.Addrs = addrs
	}
	if redisMasterName != "" {
		redis7ConfigIntegration.MasterName = redisMasterName
	}

	// set the xlog.Logger Redis adapter
	loggerOpts := xlog.NewCommonOpts()
	loggerOpts.MinLevel = xlog.FixedLevelProvider(xlog.LevelInfo)
	loggerOpts.Source = xlog.SourceProvider(5, 1)
	logger := xlog.NewSyncLogger(os.Stdout, xlog.SyncLoggerWithOptions(loggerOpts))
	redisLogger := xcache.NewRedisXLogger(logger)
	xcache.SetRedis7Logger(redisLogger)
}

func TestRedis7_integration(t *testing.T) {
	t.Parallel()

	// setup
	subject := xcache.NewRedis7(redis7ConfigIntegration)

	t.Run("wait", func(t *testing.T) { // wait for parallel tests to complete
		t.Run("key that does not expire", testCacheWithNoExpireKey(subject))
		t.Run("key expires", testCacheWithExpireKey(subject))
		t.Run("key does not exist", testCacheWithNotExistKey(subject))
		t.Run("delete key", testCacheDeleteKey(subject))
		t.Run("ttl for not yet expired key", testCacheTTLWithNotYetExpiredKey(subject))
		t.Run("stats", testCacheStats(subject, 256, 1024*1024, ">=", !redis7ConfigIntegration.IsCluster()))
	})

	// tear down
	err := subject.Close()
	assertNil(t, err)
}

func BenchmarkRedis7_Save_integration(b *testing.B) {
	cache := xcache.NewRedis7(redis7ConfigIntegration)
	benchSaveSequential(cache)(b)

	b.StopTimer()
	stats, err := cache.Stats(context.Background())
	if err != nil {
		b.Error(err)
	}
	b.Log(stats)
	if err := cache.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkRedis7_Save_parallel_integration(b *testing.B) {
	cache := xcache.NewRedis7(redis7ConfigIntegration)
	benchSaveParallel(cache)(b)

	b.StopTimer()
	stats, err := cache.Stats(context.Background())
	if err != nil {
		b.Error(err)
	}
	b.Log(stats)
	if err := cache.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkRedis7_Load_integration(b *testing.B) {
	cache := xcache.NewRedis7(redis7ConfigIntegration)
	benchLoadSequential(cache)(b)

	b.StopTimer()
	stats, err := cache.Stats(context.Background())
	if err != nil {
		b.Error(err)
	}
	b.Log(stats)
	if err := cache.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkRedis7_Load_parallel_integration(b *testing.B) {
	cache := xcache.NewRedis7(redis7ConfigIntegration)
	benchLoadParallel(cache)(b)

	b.StopTimer()
	stats, err := cache.Stats(context.Background())
	if err != nil {
		b.Error(err)
	}
	b.Log(stats)
	if err := cache.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkRedis7_TTL_integration(b *testing.B) {
	cache := xcache.NewRedis7(redis7ConfigIntegration)
	benchTTLSequential(cache)(b)

	b.StopTimer()
	stats, err := cache.Stats(context.Background())
	if err != nil {
		b.Error(err)
	}
	b.Log(stats)
	if err := cache.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkRedis7_TTL_parallel_integration(b *testing.B) {
	cache := xcache.NewRedis7(redis7ConfigIntegration)
	benchTTLParallel(cache)(b)

	b.StopTimer()
	stats, err := cache.Stats(context.Background())
	if err != nil {
		b.Error(err)
	}
	b.Log(stats)
	if err := cache.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkRedis7_Stats(b *testing.B) {
	cache := xcache.NewRedis7(redis7ConfigIntegration)
	benchStatsSequential(cache)(b)

	b.StopTimer()
	if err := cache.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkRedis7_Stats_parallel(b *testing.B) {
	cache := xcache.NewRedis7(redis7ConfigIntegration)
	benchStatsParallel(cache)(b)

	b.StopTimer()
	if err := cache.Close(); err != nil {
		b.Error(err)
	}
}
