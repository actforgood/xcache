//go:build integration

// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/blob/main/LICENSE.

package xcache_test

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/actforgood/xcache"
)

var redis6ConfigIntegration = xcache.RedisConfig{}

func init() {
	redisAddrs := os.Getenv("XCACHE_REDIS6_ADDRS")
	redisMasterName := os.Getenv("XCACHE_REDIS6_MASTER_NAME")
	if redisAddrs != "" {
		addrs := strings.Split(redisAddrs, ",")
		redis6ConfigIntegration.Addrs = addrs
	}
	if redisMasterName != "" {
		redis6ConfigIntegration.MasterName = redisMasterName
	}

	// set the slog.Logger Redis adapter
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))
	xcache.SetRedis6Logger(xcache.NewRedisSLogger(logger))
}

func TestRedis6_integration(t *testing.T) {
	t.Parallel()

	// setup
	subject := xcache.NewRedis6(redis6ConfigIntegration)

	t.Run("wait", func(t *testing.T) { // wait for parallel tests to complete
		t.Run("key that does not expire", testCacheWithNoExpireKey(subject))
		t.Run("key expires", testCacheWithExpireKey(subject))
		t.Run("key does not exist", testCacheWithNotExistKey(subject))
		t.Run("delete key", testCacheDeleteKey(subject))
		t.Run("ttl for not yet expired key", testCacheTTLWithNotYetExpiredKey(subject))
		t.Run("stats", testCacheStats(subject, 256, 1024*1024, ">=", !redis6ConfigIntegration.IsCluster()))
	})

	// tear down
	err := subject.Close()
	assertNil(t, err)
}

func BenchmarkRedis6_Save_integration(b *testing.B) {
	cache := xcache.NewRedis6(redis6ConfigIntegration)
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

func BenchmarkRedis6_Save_parallel_integration(b *testing.B) {
	cache := xcache.NewRedis6(redis6ConfigIntegration)
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

func BenchmarkRedis6_Load_integration(b *testing.B) {
	cache := xcache.NewRedis6(redis6ConfigIntegration)
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

func BenchmarkRedis6_Load_parallel_integration(b *testing.B) {
	cache := xcache.NewRedis6(redis6ConfigIntegration)
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

func BenchmarkRedis6_TTL_integration(b *testing.B) {
	cache := xcache.NewRedis6(redis6ConfigIntegration)
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

func BenchmarkRedis6_TTL_parallel_integration(b *testing.B) {
	cache := xcache.NewRedis6(redis6ConfigIntegration)
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

func BenchmarkRedis6_Stats(b *testing.B) {
	cache := xcache.NewRedis6(redis6ConfigIntegration)
	benchStatsSequential(cache)(b)

	b.StopTimer()
	if err := cache.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkRedis6_Stats_parallel(b *testing.B) {
	cache := xcache.NewRedis6(redis6ConfigIntegration)
	benchStatsParallel(cache)(b)

	b.StopTimer()
	if err := cache.Close(); err != nil {
		b.Error(err)
	}
}
