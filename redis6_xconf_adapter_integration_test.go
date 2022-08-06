//go:build integration
// +build integration

// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/LICENSE.

package xcache_test

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/actforgood/xcache"
	"github.com/actforgood/xconf"
)

func TestRedis6_withXConf_integration(t *testing.T) {
	t.Parallel()

	if redis6ConfigIntegration.IsCluster() {
		t.Skip("skip as tests rely on db, and db does not matter in cluster setup")
	}

	t.Run("expected config is changed", testRedis6WithXConfConfigIsChanged)
	t.Run("expected config is not changed", testRedis6WithXConfConfigIsNotChanged)
}

func testRedis6WithXConfConfigIsChanged(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		reloadConfig  uint32
		initialConfig = map[string]interface{}{
			xcache.RedisCfgKeyAddrs:              redis6ConfigIntegration.Addrs,
			xcache.RedisCfgKeyFailoverMasterName: redis6ConfigIntegration.MasterName,
			xcache.RedisCfgKeyDB:                 0,
			xcache.RedisCfgKeyDialTimeout:        10 * time.Second,
			xcache.RedisCfgKeyReadTimeout:        10 * time.Second,
			xcache.RedisCfgKeyWriteTimeout:       15 * time.Second,
		}
		configReloaded = map[string]interface{}{
			xcache.RedisCfgKeyAddrs:              redis6ConfigIntegration.Addrs,
			xcache.RedisCfgKeyFailoverMasterName: redis6ConfigIntegration.MasterName,
			xcache.RedisCfgKeyDB:                 1,
			xcache.RedisCfgKeyDialTimeout:        9 * time.Second,
			xcache.RedisCfgKeyReadTimeout:        9 * time.Second,
			xcache.RedisCfgKeyWriteTimeout:       14 * time.Second,
		}
		configLoader = xconf.LoaderFunc(func() (map[string]interface{}, error) {
			if atomic.LoadUint32(&reloadConfig) == 1 {
				return configReloaded, nil
			}

			return initialConfig, nil
		})
		config, _ = xconf.NewDefaultConfig(
			configLoader,
			xconf.DefaultConfigWithReloadInterval(time.Second),
		)
		subject   = xcache.NewRedis6WithConfig(config)
		keyPrefix = "test-xconf-withconfigchange-key-"
		value     = []byte("test value")
		ctx       = context.Background()
	)
	defer config.Close()
	defer subject.Close()
	// save some keys
	for i := 0; i < 10; i++ {
		key := keyPrefix + strconv.FormatInt(int64(i), 10)
		err := subject.Save(ctx, key, value, 5*time.Minute)
		requireNil(t, err)
	}

	// act
	time.Sleep(200 * time.Millisecond) // let the config reload goroutine to start
	atomic.AddUint32(&reloadConfig, 1)
	time.Sleep(1200 * time.Millisecond) // let xconf reload the configuration

	// assert
	for i := 0; i < 10; i++ { // db was switched, so keys are expected not to be found
		key := keyPrefix + strconv.FormatInt(int64(i), 10)
		_, err := subject.Load(ctx, key)
		assertTrue(t, errors.Is(err, xcache.ErrNotFound))
	}
}

func testRedis6WithXConfConfigIsNotChanged(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		reloadConfig  uint32
		initialConfig = map[string]interface{}{
			xcache.RedisCfgKeyAddrs:              redis6ConfigIntegration.Addrs,
			xcache.RedisCfgKeyFailoverMasterName: redis6ConfigIntegration.MasterName,
			xcache.RedisCfgKeyDB:                 0,
			xcache.RedisCfgKeyDialTimeout:        10 * time.Second,
			xcache.RedisCfgKeyReadTimeout:        10 * time.Second,
			xcache.RedisCfgKeyWriteTimeout:       15 * time.Second,
			"some_other_config":                  "some value",
		}
		configReloaded = map[string]interface{}{
			xcache.RedisCfgKeyAddrs:              redis6ConfigIntegration.Addrs,
			xcache.RedisCfgKeyFailoverMasterName: redis6ConfigIntegration.MasterName,
			xcache.RedisCfgKeyDB:                 0,
			xcache.RedisCfgKeyDialTimeout:        10 * time.Second,
			xcache.RedisCfgKeyReadTimeout:        10 * time.Second,
			xcache.RedisCfgKeyWriteTimeout:       15 * time.Second,
			"some_other_config":                  "some other value",
		}
		configLoader = xconf.LoaderFunc(func() (map[string]interface{}, error) {
			if atomic.LoadUint32(&reloadConfig) == 1 {
				return configReloaded, nil
			}

			return initialConfig, nil
		})
		config, _ = xconf.NewDefaultConfig(
			configLoader,
			xconf.DefaultConfigWithReloadInterval(time.Second),
		)
		subject   = xcache.NewRedis6WithConfig(config)
		keyPrefix = "test-xconf-withoutconfigchange-key-"
		value     = []byte("test value")
		ctx       = context.Background()
	)
	defer config.Close()
	defer subject.Close()
	// save some keys
	for i := 0; i < 10; i++ {
		key := keyPrefix + strconv.FormatInt(int64(i), 10)
		err := subject.Save(ctx, key, value, 5*time.Minute)
		requireNil(t, err)
	}

	// act
	time.Sleep(200 * time.Millisecond) // let the config reload goroutine to start
	atomic.AddUint32(&reloadConfig, 1)
	time.Sleep(1200 * time.Millisecond) // let xconf reload the configuration

	// assert
	for i := 0; i < 10; i++ { // check that keys are there found
		key := keyPrefix + strconv.FormatInt(int64(i), 10)
		_, err := subject.Load(ctx, key)
		assertNil(t, err)
	}
}

func TestRedis6_withXConf_concurrency(t *testing.T) {
	t.Parallel()

	var (
		readTimeout  = 3 * time.Second
		configLoader = xconf.LoaderFunc(func() (map[string]interface{}, error) {
			if time.Now().Unix()%2 == 0 {
				readTimeout += time.Second
			}

			return map[string]interface{}{
				xcache.RedisCfgKeyAddrs:              redis6ConfigIntegration.Addrs,
				xcache.RedisCfgKeyFailoverMasterName: redis6ConfigIntegration.MasterName,
				xcache.RedisCfgKeyReadTimeout:        readTimeout,
			}, nil
		})
		config, _ = xconf.NewDefaultConfig(
			configLoader,
			xconf.DefaultConfigWithReloadInterval(time.Second),
		)
		subject = xcache.NewRedis6WithConfig(config)
	)

	testCacheWithXConfConcurrency(subject)(t)

	_ = config.Close()
	err := subject.Close()
	assertNil(t, err)

	t.Logf("config changed %d times during test", (readTimeout-3*time.Second)/time.Second)
}
