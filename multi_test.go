// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/LICENSE.

package xcache_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/actforgood/xcache"
)

func init() {
	var _ xcache.Cache = (*xcache.Multi)(nil) // ensure Multi is a Cache
}

func TestMulti_Save_Load(t *testing.T) {
	t.Parallel()

	t.Run("success - save", testMultiSaveSuccessful)
	t.Run("error all - save", testMultiSaveAllCachesReturnErr)
	t.Run("error one - save", testMultiSaveOneCacheReturnsErr)

	t.Run("success - load 1", testMultiLoadReturnsValueFoundInFirstCache)
	t.Run("success - load 2", testMultiLoadReturnsValueFoundInSecondCache)
	t.Run("success - load 2, err is ignored for cache 1", testMultiLoadReturnsValueFoundInSecondCacheEvenIfFirstCacheLoadFailed)
	t.Run("error all - load", testMultiLoadAllCachesReturnErr)
	t.Run("error not found - load", testMultiLoadReturnsNotFoundErr)
}

func testMultiSaveSuccessful(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache1       = new(xcache.Mock)
		cache2       = new(xcache.Mock)
		subject      = xcache.NewMulti(cache1, cache2)
		key          = "test-multi-save-key"
		value        = []byte("test value")
		ctx          = context.Background()
		exp          = 10 * time.Minute
		saveCallback = func(ctxx context.Context, k string, v []byte, expire time.Duration) error {
			assertEqual(t, ctx, ctxx)
			assertEqual(t, key, k)
			assertEqual(t, value, v)
			assertEqual(t, exp, expire)

			return nil
		}
	)
	cache1.SetSaveCallback(saveCallback)
	cache2.SetSaveCallback(saveCallback)

	// act
	resultErr := subject.Save(ctx, key, value, exp)

	// assert
	assertNil(t, resultErr)
	assertEqual(t, 1, cache1.SaveCallsCount())
	assertEqual(t, 1, cache2.SaveCallsCount())
}

func testMultiSaveOneCacheReturnsErr(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache1           = new(xcache.Mock)
		cache2           = new(xcache.Mock)
		subject          = xcache.NewMulti(cache1, cache2)
		key              = "test-multi-save-key-succeed-even-if-one-cache-fails"
		value            = []byte("test value")
		ctx              = context.Background()
		exp              = 10 * time.Minute
		expectedErr      = errors.New("intentionally triggered Save error 1")
		errSaveCallback1 = func(context.Context, string, []byte, time.Duration) error {
			return expectedErr
		}
	)
	cache1.SetSaveCallback(errSaveCallback1)

	// act
	resultErr := subject.Save(ctx, key, value, exp)

	// assert
	assertTrue(t, errors.Is(resultErr, expectedErr))
	assertEqual(t, 1, cache1.SaveCallsCount())
	assertEqual(t, 1, cache2.SaveCallsCount()) // cache2 is still called
}

func testMultiSaveAllCachesReturnErr(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache1          = new(xcache.Mock)
		cache2          = new(xcache.Mock)
		subject         = xcache.NewMulti(cache1, cache2)
		key             = "test-multi-save-key-fails"
		value           = []byte("test value")
		ctx             = context.Background()
		exp             = 10 * time.Minute
		expectedErr1    = errors.New("intentionally triggered Save error 1")
		expectedErr2    = errors.New("intentionally triggered Save error 2")
		errSaveCallback = func(err error) func(context.Context, string, []byte, time.Duration) error {
			return func(context.Context, string, []byte, time.Duration) error {
				return err
			}
		}
	)
	cache1.SetSaveCallback(errSaveCallback(expectedErr1))
	cache2.SetSaveCallback(errSaveCallback(expectedErr2))

	// act
	resultErr := subject.Save(ctx, key, value, exp)

	// assert
	if assertNotNil(t, resultErr) {
		assertTrue(t, errors.Is(resultErr, expectedErr1))
		assertTrue(t, errors.Is(resultErr, expectedErr2))
	}
	assertEqual(t, 1, cache1.SaveCallsCount())
	assertEqual(t, 1, cache2.SaveCallsCount())
}

func testMultiLoadReturnsValueFoundInFirstCache(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache1        = new(xcache.Mock)
		cache2        = new(xcache.Mock)
		subject       = xcache.NewMulti(cache1, cache2)
		key           = "test-multi-load-key-from-first-cache"
		value         = []byte("test value")
		ctx           = context.Background()
		loadCallback1 = func(ctxx context.Context, k string) ([]byte, error) {
			assertEqual(t, ctx, ctxx)
			assertEqual(t, key, k)

			return value, nil
		}
	)
	// cache1 returns the value, cache2 is not even called.
	cache1.SetLoadCallback(loadCallback1)

	// act
	resultValue, resultErr := subject.Load(ctx, key)

	// assert
	assertNil(t, resultErr)
	assertEqual(t, value, resultValue)
	assertEqual(t, 1, cache1.LoadCallsCount())
	assertEqual(t, 0, cache2.LoadCallsCount())
}

func testMultiLoadReturnsValueFoundInSecondCache(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache1        = new(xcache.Mock)
		cache2        = new(xcache.Mock)
		subject       = xcache.NewMulti(cache1, cache2)
		key           = "test-multi-load-key-from-second-cache"
		value         = []byte("test value")
		ctx           = context.Background()
		loadCallback2 = func(ctxx context.Context, k string) ([]byte, error) {
			assertEqual(t, ctx, ctxx)
			assertEqual(t, key, k)

			return value, nil
		}
		expectedTTL  = 2 * time.Minute
		ttlCallback2 = func(ctxx context.Context, k string) (time.Duration, error) {
			assertEqual(t, ctx, ctxx)
			assertEqual(t, key, k)

			return expectedTTL, nil
		}
		saveCallback1 = func(ctxx context.Context, k string, v []byte, exp time.Duration) error {
			assertEqual(t, ctx, ctxx)
			assertEqual(t, key, k)
			assertEqual(t, value, v)
			assertEqual(t, expectedTTL, exp)

			return nil
		}
	)
	// cache1 returns ErrNotFound, cache2 returns the value.
	cache2.SetLoadCallback(loadCallback2)
	cache2.SetTTLCallback(ttlCallback2)
	cache1.SetSaveCallback(saveCallback1)

	// act
	resultValue, resultErr := subject.Load(ctx, key)

	// assert
	assertNil(t, resultErr)
	assertEqual(t, value, resultValue)
	assertEqual(t, 1, cache1.LoadCallsCount())
	assertEqual(t, 1, cache2.LoadCallsCount())
	// value is also saved into cache1:
	assertEqual(t, 1, cache2.TTLCallsCount())
	assertEqual(t, 1, cache1.SaveCallsCount())
	assertEqual(t, 0, cache2.SaveCallsCount())
}

func testMultiLoadReturnsValueFoundInSecondCacheEvenIfFirstCacheLoadFailed(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache1        = new(xcache.Mock)
		cache2        = new(xcache.Mock)
		subject       = xcache.NewMulti(cache1, cache2)
		key           = "test-multi-load-key-from-second-cache"
		value         = []byte("test value")
		ctx           = context.Background()
		loadCallback1 = func(ctxx context.Context, k string) ([]byte, error) {
			assertEqual(t, ctx, ctxx)
			assertEqual(t, key, k)

			return nil, errors.New("intentionally triggered Load error 1")
		}
		loadCallback2 = func(ctxx context.Context, k string) ([]byte, error) {
			assertEqual(t, ctx, ctxx)
			assertEqual(t, key, k)

			return value, nil
		}
	)
	// cache1 returns custom error, cache2 returns the value, in the end the value is returned.
	cache1.SetLoadCallback(loadCallback1)
	cache2.SetLoadCallback(loadCallback2)

	// act
	resultValue, resultErr := subject.Load(ctx, key)

	// assert
	assertNil(t, resultErr)
	assertEqual(t, value, resultValue)
	assertEqual(t, 1, cache1.LoadCallsCount())
	assertEqual(t, 1, cache2.LoadCallsCount())
}

func testMultiLoadAllCachesReturnErr(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache1          = new(xcache.Mock)
		cache2          = new(xcache.Mock)
		subject         = xcache.NewMulti(cache1, cache2)
		key             = "test-multi-load-key-fails"
		ctx             = context.Background()
		expectedErr1    = errors.New("intentionally triggered Load error 1")
		expectedErr2    = errors.New("intentionally triggered Load error 2")
		errLoadCallback = func(err error) func(context.Context, string) ([]byte, error) {
			return func(context.Context, string) ([]byte, error) {
				return nil, err
			}
		}
	)
	cache1.SetLoadCallback(errLoadCallback(expectedErr1))
	cache2.SetLoadCallback(errLoadCallback(expectedErr2))

	// act
	resultValue, resultErr := subject.Load(ctx, key)

	// assert
	if assertNotNil(t, resultErr) {
		assertTrue(t, errors.Is(resultErr, expectedErr1))
		assertTrue(t, errors.Is(resultErr, expectedErr2))
	}
	assertNil(t, resultValue)
	assertEqual(t, 1, cache1.LoadCallsCount())
	assertEqual(t, 1, cache2.LoadCallsCount())
}

func testMultiLoadReturnsNotFoundErr(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache1  = new(xcache.Mock)
		cache2  = new(xcache.Mock)
		subject = xcache.NewMulti(cache1, cache2)
		key     = "this-key-does-not-exist-in-any-cache"
		ctx     = context.Background()
	)

	// act
	resultValue, resultErr := subject.Load(ctx, key)

	// assert
	assertTrue(t, errors.Is(resultErr, xcache.ErrNotFound))
	assertNil(t, resultValue)
	assertEqual(t, 1, cache1.LoadCallsCount())
	assertEqual(t, 1, cache2.LoadCallsCount())
}

func TestMulti_TTL(t *testing.T) {
	t.Parallel()

	t.Run("ttl found in first cache", testMultiTTLFoundInFirstCache)
	t.Run("ttl found in second cache", testMultiTTLFoundInSecondCache)
	t.Run("ttl found in second cache, first cache returns err", testMultiTTLFoundInSecondCacheEvenIfFirstCacheTTLFailed)
	t.Run("not found key", testMultiTTLWithNotFoundKey)
	t.Run("error is returned key is not found in any cache and at least one cache returns err", testMultiTTLReturnsErr)
}

func testMultiTTLFoundInFirstCache(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache1       = new(xcache.Mock)
		cache2       = new(xcache.Mock)
		subject      = xcache.NewMulti(cache1, cache2)
		key          = "test-this-ttl-is-found-in-first-cache"
		ctx          = context.Background()
		expectedTTL  = 30 * time.Second
		ttlCallback1 = func(ctxx context.Context, k string) (time.Duration, error) {
			assertEqual(t, ctx, ctxx)
			assertEqual(t, key, k)

			return expectedTTL, nil
		}
	)
	cache1.SetTTLCallback(ttlCallback1)

	// act
	resultTTL, resultErr := subject.TTL(ctx, key)

	// assert
	assertNil(t, resultErr)
	assertEqual(t, expectedTTL, resultTTL)
	assertEqual(t, 1, cache1.TTLCallsCount())
	assertEqual(t, 0, cache2.TTLCallsCount())
}

func testMultiTTLFoundInSecondCache(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache1       = new(xcache.Mock)
		cache2       = new(xcache.Mock)
		subject      = xcache.NewMulti(cache1, cache2)
		key          = "test-this-ttl-is-found-in-second-cache"
		ctx          = context.Background()
		expectedTTL  = 30 * time.Second
		ttlCallback1 = func(ctxx context.Context, k string) (time.Duration, error) {
			assertEqual(t, ctx, ctxx)
			assertEqual(t, key, k)

			return -1, nil
		}
		ttlCallback2 = func(ctxx context.Context, k string) (time.Duration, error) {
			assertEqual(t, ctx, ctxx)
			assertEqual(t, key, k)

			return expectedTTL, nil
		}
	)
	cache1.SetTTLCallback(ttlCallback1)
	cache2.SetTTLCallback(ttlCallback2)

	// act
	resultTTL, resultErr := subject.TTL(ctx, key)

	// assert
	assertNil(t, resultErr)
	assertEqual(t, expectedTTL, resultTTL)
	assertEqual(t, 1, cache1.TTLCallsCount())
	assertEqual(t, 1, cache2.TTLCallsCount())
}

func testMultiTTLFoundInSecondCacheEvenIfFirstCacheTTLFailed(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache1       = new(xcache.Mock)
		cache2       = new(xcache.Mock)
		subject      = xcache.NewMulti(cache1, cache2)
		key          = "test-this-ttl-is-found-in-second-cache"
		ctx          = context.Background()
		expectedTTL  = 30 * time.Second
		ttlCallback1 = func(ctxx context.Context, k string) (time.Duration, error) {
			assertEqual(t, ctx, ctxx)
			assertEqual(t, key, k)

			return -1, errors.New("intentionally triggered cache 1 TTL error")
		}
		ttlCallback2 = func(ctxx context.Context, k string) (time.Duration, error) {
			assertEqual(t, ctx, ctxx)
			assertEqual(t, key, k)

			return expectedTTL, nil
		}
	)
	cache1.SetTTLCallback(ttlCallback1)
	cache2.SetTTLCallback(ttlCallback2)

	// act
	resultTTL, resultErr := subject.TTL(ctx, key)

	// assert
	assertNil(t, resultErr)
	assertEqual(t, expectedTTL, resultTTL)
	assertEqual(t, 1, cache1.TTLCallsCount())
	assertEqual(t, 1, cache2.TTLCallsCount())
}

func testMultiTTLWithNotFoundKey(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache1      = new(xcache.Mock)
		cache2      = new(xcache.Mock)
		subject     = xcache.NewMulti(cache1, cache2)
		key         = "test-ttl-does-not-exist-in-any-cache"
		ctx         = context.Background()
		ttlCallback = func(ctxx context.Context, k string) (time.Duration, error) {
			assertEqual(t, ctx, ctxx)
			assertEqual(t, key, k)

			return -1, nil
		}
	)
	cache1.SetTTLCallback(ttlCallback)
	cache2.SetTTLCallback(ttlCallback)

	// act
	resultTTL, resultErr := subject.TTL(ctx, key)

	// assert
	assertNil(t, resultErr)
	assertTrue(t, resultTTL < 0)
	assertEqual(t, 1, cache1.TTLCallsCount())
	assertEqual(t, 1, cache2.TTLCallsCount())
}

func testMultiTTLReturnsErr(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache1       = new(xcache.Mock)
		cache2       = new(xcache.Mock)
		subject      = xcache.NewMulti(cache1, cache2)
		key          = "test-ttl-fails"
		ctx          = context.Background()
		expectedErr  = errors.New("intentionally triggered cache 2 TTL error")
		ttlCallback1 = func(ctxx context.Context, k string) (time.Duration, error) {
			assertEqual(t, ctx, ctxx)
			assertEqual(t, key, k)

			return -1, nil
		}
		ttlCallback2 = func(ctxx context.Context, k string) (time.Duration, error) {
			assertEqual(t, ctx, ctxx)
			assertEqual(t, key, k)

			return -1, expectedErr
		}
	)
	cache1.SetTTLCallback(ttlCallback1)
	cache2.SetTTLCallback(ttlCallback2)

	// act
	resultTTL, resultErr := subject.TTL(ctx, key)

	// assert
	assertTrue(t, errors.Is(resultErr, expectedErr))
	assertTrue(t, resultTTL < 0)
	assertEqual(t, 1, cache1.TTLCallsCount())
	assertEqual(t, 1, cache2.TTLCallsCount())
}

func TestMulti_Stats(t *testing.T) {
	t.Parallel()

	t.Run("stats success", testMultiStatsSuccess)
	t.Run("stats error", testMultiStatsReturnsErr)
}

func testMultiStatsSuccess(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache1         = new(xcache.Mock)
		cache2         = new(xcache.Mock)
		subject        = xcache.NewMulti(cache1, cache2)
		ctx            = context.Background()
		statsCallback1 = func(ctxx context.Context) (xcache.Stats, error) {
			assertEqual(t, ctx, ctxx)

			return xcache.Stats{
				Memory:    1024,
				MaxMemory: 10 * 1024,
				Hits:      1,
				Misses:    2,
				Keys:      3,
				Expired:   4,
				Evicted:   5,
			}, nil
		}
		statsCallback2 = func(ctxx context.Context) (xcache.Stats, error) {
			assertEqual(t, ctx, ctxx)

			return xcache.Stats{
				Memory:    2 * 1024,
				MaxMemory: 20 * 1024,
				Hits:      10,
				Misses:    11,
				Keys:      12,
				Expired:   13,
				Evicted:   14,
			}, nil
		}
		expectedStats = xcache.Stats{
			Memory:    3 * 1024,
			MaxMemory: 30 * 1024,
			Hits:      11,
			Misses:    13,
			Keys:      15,
			Expired:   17,
			Evicted:   19,
		}
	)
	cache1.SetStatsCallback(statsCallback1)
	cache2.SetStatsCallback(statsCallback2)

	// act
	resultStats, resultErr := subject.Stats(ctx)

	// assert
	assertNil(t, resultErr)
	assertEqual(t, expectedStats, resultStats)
	assertEqual(t, 1, cache1.StatsCallsCount())
	assertEqual(t, 1, cache2.StatsCallsCount())
}

func testMultiStatsReturnsErr(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache1         = new(xcache.Mock)
		cache2         = new(xcache.Mock)
		cache3         = new(xcache.Mock)
		cache4         = new(xcache.Mock)
		subject        = xcache.NewMulti(cache1, cache2, cache3, cache4)
		ctx            = context.Background()
		expectedErr1   = errors.New("intentionally triggered error 1")
		expectedErr3   = errors.New("intentionally triggered error 3")
		statsCallback1 = func(ctxx context.Context) (xcache.Stats, error) {
			assertEqual(t, ctx, ctxx)

			return xcache.Stats{}, expectedErr1
		}
		statsCallback2 = func(ctxx context.Context) (xcache.Stats, error) {
			assertEqual(t, ctx, ctxx)

			return xcache.Stats{
				Memory:    1024,
				MaxMemory: 2 * 1024,
				Hits:      10,
				Misses:    11,
				Keys:      12,
				Expired:   13,
				Evicted:   14,
			}, nil
		}
		statsCallback3 = func(ctxx context.Context) (xcache.Stats, error) {
			assertEqual(t, ctx, ctxx)

			return xcache.Stats{}, expectedErr3
		}
	)
	cache1.SetStatsCallback(statsCallback1)
	cache2.SetStatsCallback(statsCallback2)
	cache3.SetStatsCallback(statsCallback3)

	// act
	resultStats, resultErr := subject.Stats(ctx)

	// assert
	if assertNotNil(t, resultErr) {
		assertTrue(t, errors.Is(resultErr, expectedErr1))
		assertTrue(t, errors.Is(resultErr, expectedErr3))
	}
	assertEqual(t, xcache.Stats{}, resultStats)
	assertEqual(t, 1, cache1.StatsCallsCount())
	assertEqual(t, 1, cache2.StatsCallsCount())
	assertEqual(t, 1, cache3.StatsCallsCount())
	assertEqual(t, 1, cache4.StatsCallsCount())
}

func BenchmarkMulti_Save(b *testing.B) {
	cache := xcache.NewMulti(xcache.Nop{}, xcache.Nop{})
	benchSaveSequential(cache)(b)
}

func BenchmarkMulti_Save_parallel(b *testing.B) {
	cache := xcache.NewMulti(xcache.Nop{}, xcache.Nop{})
	benchSaveParallel(cache)(b)
}

func BenchmarkMulti_Load(b *testing.B) {
	cache1 := new(xcache.Mock)
	cache2 := new(xcache.Mock)
	result := []byte("bench")
	cache2.SetLoadCallback(func(_ context.Context, _ string) ([]byte, error) {
		return result, nil
	})
	cache := xcache.NewMulti(cache1, cache2)
	benchLoadSequential(cache)(b)
}

func BenchmarkMulti_Load_parallel(b *testing.B) {
	cache1 := new(xcache.Mock)
	cache2 := new(xcache.Mock)
	result := []byte("bench")
	cache2.SetLoadCallback(func(_ context.Context, _ string) ([]byte, error) {
		return result, nil
	})
	cache := xcache.NewMulti(cache1, cache2)
	benchLoadParallel(cache)(b)
}

func BenchmarkMulti_TTL(b *testing.B) {
	cache := xcache.NewMulti(xcache.Nop{}, xcache.Nop{})
	benchTTLSequential(cache)(b)
}

func BenchmarkMulti_TTL_parallel(b *testing.B) {
	cache := xcache.NewMulti(xcache.Nop{}, xcache.Nop{})
	benchTTLParallel(cache)(b)
}

func ExampleMulti() {
	// create a frontend - backend multi cache.
	frontCache := xcache.NewMemory(10 * 1024 * 1024) // 10 Mb
	backCache := xcache.NewRedis6(xcache.RedisConfig{
		Addrs: []string{"127.0.0.1:6379"},
	})
	defer backCache.Close()
	cache := xcache.NewMulti(frontCache, backCache)

	ctx := context.Background()
	key := "example-multi"
	value := []byte("Hello Multi Cache")
	ttl := 10 * time.Minute

	// save a key for 10 minutes
	if err := cache.Save(ctx, key, value, ttl); err != nil {
		fmt.Println("could not save Multi cache key: " + err.Error())
	}

	// load the key's value
	if value, err := cache.Load(ctx, key); err != nil {
		fmt.Println("could not get Multi cache key: " + err.Error())
	} else {
		fmt.Println(string(value))
	}

	// should output:
	// Hello Multi Cache
}
