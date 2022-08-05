// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/LICENSE.

package xcache

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	redis7 "github.com/go-redis/redis/v9"
)

// Redis7 is Redis (distributed, ver.7) based implementation for Cache.
// It implements io.Closer, and thus it should be closed at your
// application shutdown.
type Redis7 struct {
	client               redis7.UniversalClient
	isCluster            bool          // flag indicating if cache is on a Cluster setup.
	statsInfoKeyPrefixes []string      // stats INFO command keys.
	mu                   *sync.RWMutex // concurrency semaphore used for xconf adapter.
}

// NewRedis7 instantiates a new Redis7 Cache instance (compatible with Redis ver.7).
//
// 1. If the MasterName option is specified, a sentinel-backed FailoverClient is used behind.
// 2. If the number of Addrs is two or more, a ClusterClient is used behind.
// 3. Otherwise, a single-node Client is used.
func NewRedis7(config RedisConfig) *Redis7 {
	cache := &Redis7{
		client:    redis7.NewUniversalClient(getRedis7UniversalOptions(config)),
		isCluster: config.IsCluster(),
	}
	cache.setStatsKeyPrefixes(config.DB)

	return cache
}

// setStatsKeyPrefixes sets key prefixes used to find Stats.
// If it's not a cluster configuration, adds the keys count prefix,
// otherwise, this information is not retrieved.
func (cache *Redis7) setStatsKeyPrefixes(db int) {
	if cache.isCluster {
		cache.statsInfoKeyPrefixes = make([]string, len(clusterMasterKeyPrefixes))
		copy(cache.statsInfoKeyPrefixes, clusterMasterKeyPrefixes)
	} else {
		cache.statsInfoKeyPrefixes = make([]string, 0, len(clusterMasterKeyPrefixes)+1)
		cache.statsInfoKeyPrefixes = append(cache.statsInfoKeyPrefixes, clusterMasterKeyPrefixes...)
		// example: db0:keys=59,expires=1,avg_ttl=98929
		keysCountPrefix := "db" + strconv.FormatInt(int64(db), 10) + ":keys="
		cache.statsInfoKeyPrefixes = append(cache.statsInfoKeyPrefixes, keysCountPrefix)
	}
}

// Save stores the given key-value with expiration period into cache.
// An expiration period equal to 0 (NoExpire) means no expiration.
// A negative expiration period triggers deletion of key.
// It returns an error if the key could not be saved.
func (cache *Redis7) Save(
	ctx context.Context,
	key string,
	value []byte,
	expire time.Duration,
) error {
	cache.rLock()
	defer cache.rUnlock()

	if expire < 0 {
		return cache.client.Del(ctx, key).Err()
	}

	return cache.client.Set(ctx, key, value, expire).Err()
}

// Load returns a key's value from cache, or an error if something bad happened.
// If the key is not found, ErrNotFound is returned.
func (cache *Redis7) Load(ctx context.Context, key string) ([]byte, error) {
	cache.rLock()
	value, err := cache.client.Get(ctx, key).Bytes()
	cache.rUnlock()

	if errors.Is(err, redis7.Nil) {
		return nil, ErrNotFound
	}

	return value, err
}

// TTL returns a key's expiration from cache, or an error if something bad happened.
// If the key is not found, a negative TTL is returned.
// If the key has no expiration, 0 (NoExpire) is returned.
func (cache *Redis7) TTL(ctx context.Context, key string) (time.Duration, error) {
	cache.rLock()
	ttl, err := cache.client.TTL(ctx, key).Result()
	cache.rUnlock()

	if err != nil || ttl == 0 {
		return -1, err
	}
	if ttl == redisTTLNoExpire {
		return NoExpire, nil
	}

	return ttl, nil
}

// Stats returns some statistics about cache memory/keys.
// It returns an error if something goes wrong (for example,
// client might not be able to connect to Redis server).
func (cache *Redis7) Stats(ctx context.Context) (Stats, error) {
	cache.rLock()
	defer cache.rUnlock()

	if cache.isCluster {
		if clusterClient, ok := cache.client.(*redis7.ClusterClient); ok {
			return cache.getClusterStats(ctx, clusterClient)
		}
	}

	info, err := cache.client.Info(ctx).Bytes()
	if err != nil {
		return Stats{}, err
	}

	return parseInfoStats(info, cache.statsInfoKeyPrefixes), nil
}

func (cache *Redis7) getClusterStats(ctx context.Context, cc *redis7.ClusterClient) (Stats, error) {
	var stats Stats
	err := cc.ForEachMaster(ctx, func(ctxx context.Context, client *redis7.Client) error {
		info, errInfo := client.Info(ctxx).Bytes()
		if errInfo != nil {
			return errInfo
		}

		masterStats := parseInfoStats(info, cache.statsInfoKeyPrefixes)
		atomic.AddInt64(&stats.Memory, masterStats.Memory)
		atomic.AddInt64(&stats.MaxMemory, masterStats.MaxMemory)
		atomic.AddInt64(&stats.Hits, masterStats.Hits)
		atomic.AddInt64(&stats.Misses, masterStats.Misses)
		atomic.AddInt64(&stats.Expired, masterStats.Expired)
		atomic.AddInt64(&stats.Evicted, masterStats.Evicted)

		return nil
	})
	if err != nil {
		return Stats{}, err
	}
	// If ReadOnly option is enabled, requests will end up on replicas,
	// we must take into account the hits and misses from there.
	err = cc.ForEachSlave(ctx, func(ctxx context.Context, client *redis7.Client) error {
		info, errInfo := client.Info(ctxx, "stats").Bytes()
		if errInfo != nil {
			return errInfo
		}

		replicaStats := parseInfoStats(info, clusterReplicaKeyPrefixes)
		atomic.AddInt64(&stats.Hits, replicaStats.Hits)
		atomic.AddInt64(&stats.Misses, replicaStats.Misses)

		return nil
	})
	if err != nil {
		return Stats{}, err
	}

	return stats, nil
}

// Close closes the underlying Redis client.
func (cache *Redis7) Close() (err error) {
	cache.rLock()
	err = cache.client.Close()
	cache.rUnlock()

	return
}

func (cache *Redis7) rLock() {
	if cache.mu != nil {
		cache.mu.RLock()
	}
}

func (cache *Redis7) rUnlock() {
	if cache.mu != nil {
		cache.mu.RUnlock()
	}
}

// getRedis7UniversalOptions converts a RedisConfig object to a redis7.UniversalOptions object.
func getRedis7UniversalOptions(cfg RedisConfig) *redis7.UniversalOptions {
	return &redis7.UniversalOptions{
		Addrs:        cfg.Addrs,
		DB:           cfg.DB,
		Username:     cfg.Auth.Username,
		Password:     cfg.Auth.Password,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,

		ReadOnly: cfg.ReadOnly,

		MasterName:       cfg.MasterName,
		SentinelUsername: cfg.SentinelAuth.Username,
		SentinelPassword: cfg.SentinelAuth.Password,
	}
}
