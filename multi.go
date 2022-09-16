// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/blob/main/LICENSE.

package xcache

import (
	"context"
	"errors"
	"time"

	"github.com/actforgood/xerr"
)

// Multi is a composite Cache.
// Saving a key triggers saving in all contained caches.
// A key is loaded from the first cache it is found in
// (in the order caches were provided in the constructor).
type Multi struct {
	caches []Cache
}

// NewMulti initializes a new Multi instance.
func NewMulti(caches ...Cache) Multi {
	return Multi{
		caches: caches,
	}
}

// Save stores the given key-value with expiration period into all caches.
// An expiration period equal to 0 (NoExpire) means no expiration.
// A negative expiration period triggers deletion of key.
// It returns an error if the key could not be saved (in any of the
// caches - note, that the key can end up being saved in other cache(s)).
func (cache Multi) Save(
	ctx context.Context,
	key string,
	value []byte,
	expire time.Duration,
) error {
	var mErr *xerr.MultiError
	for _, c := range cache.caches {
		if err := c.Save(ctx, key, value, expire); err != nil {
			mErr = mErr.Add(err)
		}
	}

	return mErr.ErrOrNil()
}

// Load returns a key's value from the first cache it finds it.
// If the key is found in a deeper cache, key is tried to be saved also in upfront cache(s).
// Note: if a cache returns an error, but the next cache returns the value,
// the value and nil error will be returned (method aims to be successful).
// If the key is not found in any of the caches, ErrNotFound is returned.
// If the key is not found in any of the caches, and any cache gave an error,
// that error will be returned.
func (cache Multi) Load(ctx context.Context, key string) ([]byte, error) {
	var mErr *xerr.MultiError
	for idx, c := range cache.caches {
		val, err := c.Load(ctx, key)
		if err == nil {
			if idx > 0 { // save upfront the key
				if ttl, errTTL := c.TTL(ctx, key); errTTL == nil {
					for i := idx - 1; i >= 0; i-- {
						_ = cache.caches[i].Save(ctx, key, val, ttl)
					}
				}
			}

			return val, nil
		}
		if errors.Is(err, ErrNotFound) {
			continue
		}
		mErr = mErr.Add(err)
	}

	err := mErr.ErrOrNil()
	if err == nil {
		return nil, ErrNotFound
	}

	return nil, err
}

// TTL returns a key's remaining time to live from the first cache it finds it.
// If the key is not found (in any of the caches), a negative TTL is returned.
// If the key has no expiration, 0 (NoExpire) is returned.
// Note: if a cache returns an error, but the next cache returns the ttl,
// the ttl and nil error will be returned (method aims to be successful).
// If the key is not found in any of the caches, and any cache gave an error,
// that error will be returned.
func (cache Multi) TTL(ctx context.Context, key string) (time.Duration, error) {
	var mErr *xerr.MultiError
	for _, c := range cache.caches {
		if ttl, err := c.TTL(ctx, key); err != nil {
			mErr = mErr.Add(err)
		} else if ttl >= 0 {
			return ttl, nil
		}
	}

	return -1, mErr.ErrOrNil()
}

// Stats returns statistics about memory cache, or an error if something bad happens within any of the caches.
// Returned statistics are just summed up for all contained caches.
func (cache Multi) Stats(ctx context.Context) (Stats, error) {
	var mErr *xerr.MultiError
	var mStats Stats
	for _, c := range cache.caches {
		if stats, err := c.Stats(ctx); err != nil {
			mErr = mErr.Add(err)
		} else {
			mStats.Memory += stats.Memory
			mStats.MaxMemory += stats.MaxMemory
			mStats.Hits += stats.Hits
			mStats.Misses += stats.Misses
			mStats.Keys += stats.Keys
			mStats.Expired += stats.Expired
			mStats.Evicted += stats.Evicted
		}
	}

	err := mErr.ErrOrNil()
	if err != nil {
		return Stats{}, err
	}

	return mStats, nil
}
