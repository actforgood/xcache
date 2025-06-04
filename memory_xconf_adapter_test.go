// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/blob/main/LICENSE.

package xcache_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/actforgood/xcache"
	"github.com/actforgood/xconf"
)

func TestMemory_withXConf(t *testing.T) {
	t.Parallel()

	t.Run("expected config is changed", testMemoryWithXConfConfigIsChanged)
	t.Run("expected config is not changed", testMemoryWithXConfConfigIsNotChanged)
}

func testMemoryWithXConfConfigIsChanged(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		reloadConfig  uint32
		memSize1      int64 = freecacheMinMem // 512 Kb
		initialConfig       = map[string]any{
			xcache.MemoryCfgKeyMemorySize: memSize1,
		}
		memSize2       int64 = 1024 * 1024 // 1 Mb
		configReloaded       = map[string]any{
			xcache.MemoryCfgKeyMemorySize: memSize2,
		}
		configLoader = xconf.LoaderFunc(func() (map[string]any, error) {
			if atomic.LoadUint32(&reloadConfig) == 1 {
				return configReloaded, nil
			}

			return initialConfig, nil
		})
		config, _ = xconf.NewDefaultConfig(
			configLoader,
			xconf.DefaultConfigWithReloadInterval(time.Second),
		)
		subject   = xcache.NewMemoryWithConfig(config)
		keyPrefix = "test-xconf-key-"
		value     = []byte("test value")
		ctx       = context.Background()
	)
	defer config.Close()
	// save some keys
	for i := range 10 {
		key := keyPrefix + strconv.FormatInt(int64(i), 10)
		err := subject.Save(ctx, key, value, xcache.NoExpire)
		requireNil(t, err)
	}
	for i := 10; i < 20; i++ {
		key := keyPrefix + strconv.FormatInt(int64(i), 10)
		err := subject.Save(ctx, key, value, 5*time.Second)
		requireNil(t, err)
	}
	for i := 20; i < 30; i++ { // keys that will expire
		key := keyPrefix + strconv.FormatInt(int64(i), 10)
		err := subject.Save(ctx, key, value, time.Second)
		requireNil(t, err)
	}

	// act
	stats1, _ := subject.Stats(ctx)
	time.Sleep(1300 * time.Millisecond) // let the keys with 1s expiration to expire
	atomic.AddUint32(&reloadConfig, 1)
	time.Sleep(1300 * time.Millisecond) // let xconf reload the configuration
	stats2, _ := subject.Stats(ctx)

	// assert
	assertEqual(t, memSize1, stats1.MaxMemory)
	assertEqual(t, int64(30), stats1.Keys)
	assertEqual(t, memSize2, stats2.MaxMemory)
	assertEqual(t, int64(20), stats2.Keys) // 10 expired
	for i := range 20 {
		key := keyPrefix + strconv.FormatInt(int64(i), 10)
		_, err := subject.Load(ctx, key)
		assertNil(t, err)
	}
	for i := 20; i < 30; i++ {
		key := keyPrefix + strconv.FormatInt(int64(i), 10)
		_, err := subject.Load(ctx, key)
		assertTrue(t, errors.Is(err, xcache.ErrNotFound))
	}
}

func testMemoryWithXConfConfigIsNotChanged(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		reloadConfig  uint32
		memSize       int64 = freecacheMinMem // 512 Kb
		initialConfig       = map[string]any{
			xcache.MemoryCfgKeyMemorySize: memSize,
			"some_other_config":           "some value",
		}
		configReloaded = map[string]any{
			xcache.MemoryCfgKeyMemorySize: memSize,
			"some_other_config":           "some other value",
		}
		configLoader = xconf.LoaderFunc(func() (map[string]any, error) {
			if atomic.LoadUint32(&reloadConfig) == 1 {
				return configReloaded, nil
			}

			return initialConfig, nil
		})
		config, _ = xconf.NewDefaultConfig(
			configLoader,
			xconf.DefaultConfigWithReloadInterval(time.Second),
		)
		subject   = xcache.NewMemoryWithConfig(config)
		keyPrefix = "test-xconf-key-"
		value     = []byte("test value")
		ctx       = context.Background()
	)
	defer config.Close()
	// save some keys
	for i := range 10 {
		key := keyPrefix + strconv.FormatInt(int64(i), 10)
		err := subject.Save(ctx, key, value, xcache.NoExpire)
		requireNil(t, err)
	}
	for i := 10; i < 20; i++ {
		key := keyPrefix + strconv.FormatInt(int64(i), 10)
		err := subject.Save(ctx, key, value, 15*time.Second)
		requireNil(t, err)
	}
	for i := 20; i < 30; i++ { // keys that will expire
		key := keyPrefix + strconv.FormatInt(int64(i), 10)
		err := subject.Save(ctx, key, value, time.Second)
		requireNil(t, err)
	}

	// act
	stats1, _ := subject.Stats(ctx)
	time.Sleep(1300 * time.Millisecond) // let the keys with 1s expiration to expire
	atomic.AddUint32(&reloadConfig, 1)
	time.Sleep(1300 * time.Millisecond) // let xconf reload the configuration
	stats2, _ := subject.Stats(ctx)

	// assert
	assertEqual(t, memSize, stats1.MaxMemory)
	assertEqual(t, int64(30), stats1.Keys)
	assertEqual(t, memSize, stats2.MaxMemory)
	assertEqual(t, int64(30), stats2.Keys) // 10 expired, but freecache does not deletes them until load
	for i := range 20 {
		key := keyPrefix + strconv.FormatInt(int64(i), 10)
		_, err := subject.Load(ctx, key)
		assertNil(t, err)
	}
	for i := 20; i < 30; i++ {
		key := keyPrefix + strconv.FormatInt(int64(i), 10)
		_, err := subject.Load(ctx, key)
		assertTrue(t, errors.Is(err, xcache.ErrNotFound))
	}
}

func TestMemory_withXConf_concurrency(t *testing.T) {
	t.Parallel()

	var (
		memSize      = freecacheMinMem
		configLoader = xconf.LoaderFunc(func() (map[string]any, error) {
			if time.Now().Unix()%2 == 0 {
				memSize++
			}

			return map[string]any{
				xcache.MemoryCfgKeyMemorySize: memSize,
			}, nil
		})
		config, _ = xconf.NewDefaultConfig(
			configLoader,
			xconf.DefaultConfigWithReloadInterval(time.Second),
		)
		subject = xcache.NewMemoryWithConfig(config)
	)

	testCacheWithXConfConcurrency(subject)(t)

	_ = config.Close()

	t.Logf("config changed %d times during test", memSize-freecacheMinMem)
}

func ExampleMemory_withXConf() {
	// Setup an env (assuming your application configuration comes from env,
	// it's not mandatory to be env, you can use any source loader you want)
	_ = os.Setenv("MY_APP_CACHE_MEM_SIZE", "1048576")
	defer os.Unsetenv("MY_APP_CACHE_MEM_SIZE")

	// Initialize config, we set an alias, as example, as our config key is custom ("MY_APP_CACHE_MEM_SIZE").
	config, err := xconf.NewDefaultConfig(
		xconf.AliasLoader(
			xconf.EnvLoader(),
			xcache.MemoryCfgKeyMemorySize, "MY_APP_CACHE_MEM_SIZE",
		),
		xconf.DefaultConfigWithReloadInterval(2*time.Second),
	)
	if err != nil {
		panic(err)
	}
	defer config.Close()

	// Initialize the cache our application will use.
	cache := xcache.NewMemoryWithConfig(config)

	// From this point forward you can use the cache object however you want.

	// Let's print some stats to see the memory size.
	stats, err := cache.Stats(context.Background())
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(stats)
	}

	// Let's assume we monitor our cache - see StatsWatcher for that,
	// and we notice existing keys count is kind of constant and an increase in evictions,
	// meaning our memory cache is probably full.

	// We decide to increase the memory size.
	_ = os.Setenv("MY_APP_CACHE_MEM_SIZE", "5242880")
	time.Sleep(2500 * time.Millisecond) // wait for config to reload

	// Print again the stats, we can see that memory was changed,
	// without the need of restarting/redeploying our application.
	stats, err = cache.Stats(context.Background())
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(stats)
	}

	// Output:
	// mem=1M maxMem=1M memUsage=100.00% hits=0 misses=0 hitRate=100.00% keys=0 expired=0 evicted=0
	// mem=5M maxMem=5M memUsage=100.00% hits=0 misses=0 hitRate=100.00% keys=0 expired=0 evicted=0
}
