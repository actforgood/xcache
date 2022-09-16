// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/blob/main/LICENSE.

package xcache

import (
	"time"

	"github.com/actforgood/xconf"
)

const (
	// RedisCfgKeyAddrs is the key under which xconf.Config expects Redis server(s).
	// Value should be a slice of string(s).
	RedisCfgKeyAddrs = "xcache.redis.addrs"
	// RedisCfgKeyDB is the key under which xconf.Config expects Redis DB.
	RedisCfgKeyDB = "xcache.redis.db"
	// RedisCfgKeyAuthUsername is the key under which xconf.Config expects auth username.
	RedisCfgKeyAuthUsername = "xcache.redis.auth.username"
	// RedisCfgKeyAuthPassword is the key under which xconf.Config expects auth password.
	RedisCfgKeyAuthPassword = "xcache.redis.auth.password"
	// RedisCfgKeyDialTimeout is the key under which xconf.Config expects dial timeout.
	RedisCfgKeyDialTimeout = "xcache.redis.timeout.dial"
	// RedisCfgKeyReadTimeout is the key under which xconf.Config expects read timeout.
	RedisCfgKeyReadTimeout = "xcache.redis.timeout.read"
	// RedisCfgKeyWriteTimeout is the key under which xconf.Config expects write timeout.
	RedisCfgKeyWriteTimeout = "xcache.redis.timeout.write"
	// RedisCfgKeyClusterReadonly is the key under which xconf.Config expects readonly flag.
	RedisCfgKeyClusterReadonly = "xcache.redis.cluster.readonly"
	// RedisCfgKeyFailoverMasterName is the key under which xconf.Config expects master name.
	RedisCfgKeyFailoverMasterName = "xcache.redis.failover.mastername"
	// RedisCfgKeyFailoverAuthUsername is the key under which xconf.Config expects sentinel auth username.
	RedisCfgKeyFailoverAuthUsername = "xcache.redis.failover.auth.usernmae"
	// RedisCfgKeyFailoverAuthPassword is the key under which xconf.Config expects sentinel auth password.
	RedisCfgKeyFailoverAuthPassword = "xcache.redis.failover.auth.password"
)

// getRedisConfig returns a RedisConfig object populated with values taken from a xconf.Config.
func getRedisConfig(config xconf.Config) RedisConfig {
	return RedisConfig{
		Addrs: config.Get(RedisCfgKeyAddrs, []string{"127.0.0.1:6379"}).([]string),
		DB:    config.Get(RedisCfgKeyDB, 0).(int),
		Auth: RedisAuth{
			Username: config.Get(RedisCfgKeyAuthUsername, "").(string),
			Password: config.Get(RedisCfgKeyAuthPassword, "").(string),
		},
		DialTimeout:  config.Get(RedisCfgKeyDialTimeout, 5*time.Second).(time.Duration),
		ReadTimeout:  config.Get(RedisCfgKeyReadTimeout, 3*time.Second).(time.Duration),
		WriteTimeout: config.Get(RedisCfgKeyWriteTimeout, 5*time.Second).(time.Duration),
		ReadOnly:     config.Get(RedisCfgKeyClusterReadonly, false).(bool),
		MasterName:   config.Get(RedisCfgKeyFailoverMasterName, "").(string),
		SentinelAuth: RedisAuth{
			Username: config.Get(RedisCfgKeyFailoverAuthUsername, "").(string),
			Password: config.Get(RedisCfgKeyFailoverAuthPassword, "").(string),
		},
	}
}

// isRedisConfigKey checks of give key is one of RedisCfgKey*. config keys.
func isRedisConfigKey(key string) bool {
	return key == RedisCfgKeyAddrs ||
		key == RedisCfgKeyDB ||
		key == RedisCfgKeyAuthUsername ||
		key == RedisCfgKeyAuthPassword ||
		key == RedisCfgKeyDialTimeout ||
		key == RedisCfgKeyReadTimeout ||
		key == RedisCfgKeyWriteTimeout ||
		key == RedisCfgKeyClusterReadonly ||
		key == RedisCfgKeyFailoverMasterName ||
		key == RedisCfgKeyFailoverAuthUsername ||
		key == RedisCfgKeyFailoverAuthPassword
}
