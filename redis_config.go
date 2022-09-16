// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/blob/main/LICENSE.

package xcache

import (
	"bytes"
	"strconv"
	"time"
)

// Note: the difference between Redis6 and Redis7, in implementation, consists of using a different version of go-redis client.

// redisTTLNoExpire is Redis TTL command reply value for a key with no expiration.
const redisTTLNoExpire = -1

// RedisConfig contains commonly used information for Redis connection.
type RedisConfig struct {
	// Addrs contains either a single address or a seed list of host:port addresses
	// of cluster/sentinel nodes.
	// Example:
	//	Addrs: []string{"redis-single-node:6379"}
	//	Addrs: []string{"redis-sentinel-node-1:26379", "redis-sentinel-node-2:26379", "redis-sentinel-node-3:26379"}
	// 	Addrs: []string{"redis-cluster-node-1:7000", "redis-cluster-node-2:7001", "redis-cluster-node-3:7002"}
	Addrs []string

	// DB is the database to be selected after connecting to the server.
	// Only for single-node and failover clients.
	DB int

	// Common options

	// Auth represents the auth user/pwd of redis instances.
	Auth RedisAuth

	// DialTimeout is the timeout for dial op.
	DialTimeout time.Duration
	// ReadTimeout is the timeout for read ops.
	ReadTimeout time.Duration
	// WriteTimeout is the timeout for write ops.
	WriteTimeout time.Duration

	// Enables read-only commands on slave nodes. [cluster only]
	ReadOnly bool

	// MasterName represents the sentinel master name. [failover only]
	MasterName string
	// SentinelAuth represents the auth user/pwd of redis sentinel instances. [failover only]
	SentinelAuth RedisAuth
}

// RedisAuth contains user/password authentication info.
type RedisAuth struct {
	// Username to authenticate with.
	Username string
	// Password to authenticate with.
	Password string
}

// IsCluster returns true if config is for a cluster configuration.
func (rc RedisConfig) IsCluster() bool {
	return len(rc.Addrs) > 1 && rc.MasterName == ""
}

const (
	redisInfoPrefixMem            = "used_memory:"
	redisInfoPrefixMaxMem         = "maxmemory:"
	redisInfoPrefixTotalSystemMem = "total_system_memory:"
	redisInfoPrefixHits           = "keyspace_hits:"
	redisInfoPrefixMisses         = "keyspace_misses:"
	redisInfoPrefixEvictedKeys    = "evicted_keys:"
	redisInfoPrefixExpiredKeys    = "expired_keys:"
)

var clusterReplicaKeyPrefixes = []string{
	redisInfoPrefixHits,
	redisInfoPrefixMisses,
}

var clusterMasterKeyPrefixes = []string{
	redisInfoPrefixMem,
	redisInfoPrefixMaxMem,
	redisInfoPrefixTotalSystemMem,
	redisInfoPrefixEvictedKeys,
	redisInfoPrefixExpiredKeys,
	redisInfoPrefixHits,
	redisInfoPrefixMisses,
}

// parseInfoStats parses INFO command response and extracts needed information.
//
// Note: On cluster setup, no. of keys can't be retrieved directly (INFO KEYSPACE / DBSIZE don't work on Redis Cluster).
// Can be calculated with Cluster Slots and Cluster CountKeysInSlot (each slot), but there are 16384 slots,
// and due to too much overhead reasons, stats.Keys remains 0.
func parseInfoStats(info []byte, keyPrefixes []string) Stats {
	var (
		extractedDigits = make([]byte, 20)
		digitsLen       int
		stats           Stats
	)

	for _, keyPrefix := range keyPrefixes {
		if idx := bytes.Index(info, []byte(keyPrefix)); idx != -1 {
			digitsLen = 0
			for digitIdx := idx + len(keyPrefix); digitIdx < len(info) && digitsLen < cap(extractedDigits); digitIdx++ {
				if info[digitIdx] >= '0' && info[digitIdx] <= '9' {
					extractedDigits[digitsLen] = info[digitIdx]
					digitsLen++
				} else {
					break
				}
			}
			digits := extractedDigits[:digitsLen]
			infoStrValue := bytesToString(digits)
			infoIntValue, _ := strconv.ParseInt(infoStrValue, 10, 64)
			switch keyPrefix {
			case redisInfoPrefixMem:
				stats.Memory = infoIntValue
			case redisInfoPrefixMaxMem:
				if stats.MaxMemory > 0 {
					stats.MaxMemory = infoIntValue
				}
			case redisInfoPrefixTotalSystemMem:
				if stats.MaxMemory == 0 {
					stats.MaxMemory = infoIntValue
				}
			case redisInfoPrefixEvictedKeys:
				stats.Evicted = infoIntValue
			case redisInfoPrefixExpiredKeys:
				stats.Expired = infoIntValue
			case redisInfoPrefixHits:
				stats.Hits = infoIntValue
			case redisInfoPrefixMisses:
				stats.Misses = infoIntValue
			default:
				stats.Keys = infoIntValue
			}
		}
	}

	return stats
}
