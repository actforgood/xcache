// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/blob/main/LICENSE.

package xcache_test

import (
	"context"
	"encoding/binary"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/actforgood/xcache"
)

func benchLoadSequential(cache xcache.Cache) func(b *testing.B) {
	return func(b *testing.B) {
		ctx, expire, key, value := getBenchInput()
		if err := cache.Save(ctx, key, value, expire); err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			if _, err := cache.Load(ctx, key); err != nil {
				b.Error(err)
			}
		}
	}
}

func benchLoadParallel(cache xcache.Cache) func(b *testing.B) {
	return func(b *testing.B) {
		ctx, expire, key, value := getBenchInput()
		if err := cache.Save(ctx, key, value, expire); err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if _, err := cache.Load(ctx, key); err != nil {
					b.Error(err)
				}
			}
		})
	}
}

func benchSaveSequential(cache xcache.Cache) func(b *testing.B) {
	return func(b *testing.B) {
		ctx, expire, keyPrefix, value := getBenchInput()
		// Used byte strategy to generate distinct key
		// because in this way, no extra allocation is reported.
		// Something more simple like key := keyPrefix + strconv.FormatInt(int64(n), 10) would end up
		// reporting 2 extra allocations which have nothing to do with the tested cache.
		keyPrefixLen := len(keyPrefix)
		keyBytes := make([]byte, len(keyPrefix)+8)
		for i := range keyPrefixLen {
			keyBytes[i] = keyPrefix[i]
		}

		b.ReportAllocs()
		b.ResetTimer()

		for n := range b.N {
			binary.LittleEndian.PutUint64(keyBytes[keyPrefixLen:], uint64(n))
			key := *(*string)(unsafe.Pointer(&keyBytes))
			if err := cache.Save(ctx, key, value, expire); err != nil {
				b.Error(err)
			}
		}
	}
}

func benchSaveParallel(cache xcache.Cache) func(b *testing.B) {
	return func(b *testing.B) {
		ctx, expire, keyPrefix, value := getBenchInput()
		var counter uint64

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// 1 extra allocation will be reported from keyBytes,
				// so in _Save_parallel benchmarks real value should be considered the reported one - 1.
				keyPrefixLen := len(keyPrefix)
				keyBytes := make([]byte, len(keyPrefix)+8)
				for i := range keyPrefixLen {
					keyBytes[i] = keyPrefix[i]
				}
				binary.LittleEndian.PutUint64(keyBytes[keyPrefixLen:], atomic.LoadUint64(&counter))
				key := *(*string)(unsafe.Pointer(&keyBytes))
				if err := cache.Save(ctx, key, value, expire); err != nil {
					b.Error(err)
				}
				atomic.AddUint64(&counter, 1)
			}
		})
	}
}

func benchTTLSequential(cache xcache.Cache) func(b *testing.B) {
	return func(b *testing.B) {
		ctx, expire, key, value := getBenchInput()
		if err := cache.Save(ctx, key, value, expire); err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			if _, err := cache.TTL(ctx, key); err != nil {
				b.Error(err)
			}
		}
	}
}

func benchTTLParallel(cache xcache.Cache) func(b *testing.B) {
	return func(b *testing.B) {
		ctx, expire, key, value := getBenchInput()
		if err := cache.Save(ctx, key, value, expire); err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if _, err := cache.TTL(ctx, key); err != nil {
					b.Error(err)
				}
			}
		})
	}
}

func benchStatsSequential(cache xcache.Cache) func(b *testing.B) {
	return func(b *testing.B) {
		ctx := context.Background()

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			if _, err := cache.Stats(ctx); err != nil {
				b.Error(err)
			}
		}
	}
}

func benchStatsParallel(cache xcache.Cache) func(b *testing.B) {
	return func(b *testing.B) {
		ctx := context.Background()

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if _, err := cache.Stats(ctx); err != nil {
					b.Error(err)
				}
			}
		})
	}
}

func getBenchInput() (context.Context, time.Duration, string, []byte) {
	return context.Background(), 3 * time.Minute, "xcache_bench_key", []byte("benchmark")
}
