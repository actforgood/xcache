// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/blob/main/LICENSE.

package xcache

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is an error returned by a cache Load operation if a key does not exist.
var ErrNotFound = errors.New("key not found")

// NoExpire is the value for no expiration.
const NoExpire time.Duration = 0

// Cache provides prototype a for storing and returning a key-value into/from cache.
type Cache interface {
	// Save stores the given key-value with expiration period into cache.
	// An expiration period equal to 0 (NoExpire) means no expiration.
	// A negative expiration period triggers deletion of key.
	// It returns an error if the key could not be saved.
	Save(ctx context.Context, key string, value []byte, expire time.Duration) error

	// Load returns a key's value from cache, or an error if something bad happened.
	// If the key is not found, ErrNotFound is returned.
	Load(ctx context.Context, key string) ([]byte, error)

	// TTL returns a key's remaining time to live, or an error if something bad happened.
	// If the key is not found, a negative TTL is returned.
	// If the key has no expiration, 0 (NoExpire) is returned.
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Stats returns some statistics about cache's memory/keys.
	// It returns an error if something goes wrong.
	Stats(context.Context) (Stats, error)
}
