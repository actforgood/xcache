// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/LICENSE.

package xcache_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/actforgood/xcache"
)

func testCacheWithNoExpireKey(subject xcache.Cache) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		// arrange
		var (
			key   = "test-no-expire-key"
			value = []byte("test value")
			ctx   = context.Background()
			exp   = xcache.NoExpire
		)

		// act & assert save
		resultErr := subject.Save(ctx, key, value, exp)
		requireNil(t, resultErr)

		for i := 0; i < 50; i++ {
			// act & assert load
			resultValue, resultErr := subject.Load(ctx, key)
			assertNil(t, resultErr)
			assertEqual(t, value, resultValue)
		}

		// act & assert ttl
		resultTTL, resultErr := subject.TTL(ctx, key)
		assertNil(t, resultErr)
		assertEqual(t, xcache.NoExpire, resultTTL)
	}
}

func testCacheWithExpireKey(subject xcache.Cache) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		// arrange
		var (
			key   = "test-expire-key"
			value = []byte("test value")
			ctx   = context.Background()
			exp   = 500 * time.Millisecond
		)

		// act & assert save
		resultErr := subject.Save(ctx, key, value, exp)
		requireNil(t, resultErr)

		// act & assert load
		resultValue, resultErr := subject.Load(ctx, key)
		assertNil(t, resultErr)
		assertEqual(t, value, resultValue)

		// act & assert load after expire time passed
		time.Sleep(1100 * time.Millisecond)
		resultValue, resultErr = subject.Load(ctx, key)
		assertTrue(t, errors.Is(resultErr, xcache.ErrNotFound))
		assertNil(t, resultValue)

		// act & assert ttl
		resultTTL, resultErr := subject.TTL(ctx, key)
		assertNil(t, resultErr)
		assertTrue(t, resultTTL < 0)
	}
}

func testCacheWithNotExistKey(subject xcache.Cache) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		// arrange
		var (
			key = "test-this-key-does-not-exist"
			ctx = context.Background()
		)

		// act & assert load
		resultValue, resultErr := subject.Load(ctx, key)
		assertTrue(t, errors.Is(resultErr, xcache.ErrNotFound))
		assertNil(t, resultValue)

		// act & assert ttl
		resultTTL, resultErr := subject.TTL(ctx, key)
		assertNil(t, resultErr)
		assertTrue(t, resultTTL < 0)
	}
}

func testCacheDeleteKey(subject xcache.Cache) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		// arrange
		var (
			key   = "test-delete-key"
			value = []byte("test value")
			ctx   = context.Background()
			exp   = xcache.NoExpire
		)
		// act & assert save
		resultErr := subject.Save(ctx, key, value, exp)
		requireNil(t, resultErr)

		// act & assert load
		resultValue, resultErr := subject.Load(ctx, key)
		assertNil(t, resultErr)
		assertEqual(t, value, resultValue)

		// act & assert delete
		resultErr = subject.Save(ctx, key, nil, -1)
		requireNil(t, resultErr)

		// act & assert load
		resultValue, resultErr = subject.Load(ctx, key)
		assertTrue(t, errors.Is(resultErr, xcache.ErrNotFound))
		assertNil(t, resultValue)

		// act & assert ttl
		resultTTL, resultErr := subject.TTL(ctx, key)
		assertNil(t, resultErr)
		assertTrue(t, resultTTL < 0)
	}
}

func testCacheTTLWithNotYetExpiredKey(subject xcache.Cache) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		// arrange
		var (
			key   = "test-expire-ttl-key"
			value = []byte("test value")
			ctx   = context.Background()
			exp   = time.Duration(time.Minute)
		)

		// act & assert save
		resultErr := subject.Save(ctx, key, value, exp)
		requireNil(t, resultErr)
		time.Sleep(time.Second)

		// act & assert ttl
		resultTTL, resultErr := subject.TTL(ctx, key)
		assertNil(t, resultErr)
		assertTrue(t, resultTTL < exp)
		assertTrue(t, resultTTL > 0)
	}
}

func testCacheStats(
	subject xcache.Cache,
	expectedMem, expectedMaxMem int64, memCheckOp string,
	checkKeys bool,
) func(t *testing.T) {
	return func(t *testing.T) {
		// arrange
		var (
			hitKey     = "test-stats-hit-key-"
			missKey    = "test-stats-miss-key"
			expKey     = "test-stats-exp-key-"
			value      = []byte("test value")
			ctx        = context.Background()
			smallerExp = 2 * time.Second
			biggerExp  = 10 * time.Minute
		)
		for i := 0; i < 20; i++ { // delete keys needed for Keys reporting
			key := hitKey + strconv.FormatInt(int64(i), 10)
			_ = subject.Save(ctx, key, nil, -1)
		}
		prevStats, resultErr := subject.Stats(ctx)
		requireNil(t, resultErr)

		for i := 0; i < 20; i++ { // 20 x hit
			key := hitKey + strconv.FormatInt(int64(i), 10)
			resultErr = subject.Save(ctx, key, value, biggerExp)
			requireNil(t, resultErr)
			_, resultErr = subject.Load(ctx, key)
			requireNil(t, resultErr)
		}
		for i := 0; i < 10; i++ { // 10 x hit, 10 x expired
			key := expKey + strconv.FormatInt(int64(i), 10)
			resultErr = subject.Save(ctx, key, value, smallerExp)
			requireNil(t, resultErr)
			_, resultErr = subject.Load(ctx, key)
			requireNil(t, resultErr)
		}
		for i := 0; i < 25; i++ { // 25 x miss
			_, resultErr = subject.Load(ctx, missKey)
			assertTrue(t, errors.Is(resultErr, xcache.ErrNotFound))
		}
		time.Sleep(2500 * time.Millisecond) // let keys with smallerExp expire
		for i := 0; i < 10; i++ {           // 10 x miss
			key := expKey + strconv.FormatInt(int64(i), 10) // load expired keys to let Freecache count the expiration
			_, resultErr = subject.Load(ctx, key)
			assertTrue(t, errors.Is(resultErr, xcache.ErrNotFound))
		}

		// act
		resultStats, resultErr := subject.Stats(ctx)

		// assert
		assertNil(t, resultErr)
		if memCheckOp == "==" {
			assertEqual(t, expectedMem, resultStats.Memory)
			assertEqual(t, expectedMaxMem, resultStats.MaxMemory)
		} else {
			assertTrue(t, resultStats.Memory >= expectedMem)
			assertTrue(t, resultStats.MaxMemory >= expectedMaxMem)
		}
		assertTrue(t, resultStats.Hits >= prevStats.Hits+30)
		assertTrue(t, resultStats.Misses >= prevStats.Misses+35)
		assertTrue(t, resultStats.Expired >= prevStats.Expired+10)
		assertTrue(t, resultStats.Evicted >= prevStats.Evicted)
		if checkKeys {
			assertTrue(t, resultStats.Keys >= prevStats.Keys+20)
		}
	}
}
