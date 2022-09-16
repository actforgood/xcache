// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/blob/main/LICENSE.

package xcache_test

import (
	"context"
	"errors"
	"testing"

	"github.com/actforgood/xcache"
)

func init() {
	var _ xcache.Cache = (*xcache.Nop)(nil) // test Nop is a Cache
}

func TestNop(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		subject = xcache.Nop{}
		key     = "test-nop-key"
		value   = []byte("test ignored value")
		ctx     = context.Background()
		exp     = xcache.NoExpire
	)

	// act & assert save
	resultErr := subject.Save(ctx, key, value, exp)
	requireNil(t, resultErr)

	// act & assert load
	resultValue, resultErr := subject.Load(ctx, key)
	assertTrue(t, errors.Is(resultErr, xcache.ErrNotFound))
	assertNil(t, resultValue)

	// act & assert ttl
	resultExp, resultErr := subject.TTL(ctx, key)
	assertNil(t, resultErr)
	assertTrue(t, resultExp < 0)

	// act & assert stats
	resultStats, resultErr := subject.Stats(ctx)
	assertEqual(t, xcache.Stats{}, resultStats)
	assertNil(t, resultErr)
}
