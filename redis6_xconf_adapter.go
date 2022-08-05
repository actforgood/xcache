// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/LICENSE.

package xcache

import (
	"sync"

	"github.com/actforgood/xconf"
	redis6 "github.com/go-redis/redis/v8"
)

// NewRedis6WithConfig initializes a Redis6 Cache with configuration taken from a xconf.Config.
//
// Keys under which configuration is expected are defined in RedisCfgKey* constants
// (note, you can have different config keys defined in your project, you'll have to create an alias
// for them to expected values by this package).
//
// An observer is registered to xconf.DefaultConfig (which knows to reload configuration).
// In case any config value requested by Redis6 is changed, the Redis6 is reinitialized with the new config.
func NewRedis6WithConfig(config xconf.Config) *Redis6 {
	cache := NewRedis6(getRedisConfig(config))
	cache.mu = new(sync.RWMutex)

	if defConfig, ok := config.(*xconf.DefaultConfig); ok {
		defConfig.RegisterObserver(cache.onConfigChange)
	}

	return cache
}

// onConfigChange is a callback to be registered to xconf.DefaultConfig knows knows to reload configuration.
// In case one of RedisCfgKey* configs is changed, the Redis6 is reinitialized with the new config.
// This callback is automatically registered on instantiation of a Redis6 object with NewRedis6WithConfig.
func (cache *Redis6) onConfigChange(config xconf.Config, changedKeys ...string) {
	configHasChanged := false
	for _, changedKey := range changedKeys {
		if isRedisConfigKey(changedKey) {
			configHasChanged = true

			break
		}
	}

	if !configHasChanged {
		return
	}

	redisConfig := getRedisConfig(config)
	newClient := redis6.NewUniversalClient(getRedis6UniversalOptions(redisConfig))

	cache.mu.Lock()
	oldClient := cache.client
	cache.client = newClient
	cache.isCluster = redisConfig.IsCluster()
	cache.setStatsKeyPrefixes(redisConfig.DB)
	cache.mu.Unlock()

	_ = oldClient.Close()
}
