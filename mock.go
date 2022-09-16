// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/blob/main/LICENSE.

package xcache

import (
	"context"
	"sync/atomic"
	"time"
)

// Mock is a mock to be used in UT.
type Mock struct {
	saveCallsCnt  uint32
	saveCallback  func(context.Context, string, []byte, time.Duration) error
	loadCallsCnt  uint32
	loadCallback  func(context.Context, string) ([]byte, error)
	ttlCallsCnt   uint32
	ttlCallback   func(context.Context, string) (time.Duration, error)
	statsCallsCnt uint32
	statsCallback func(context.Context) (Stats, error)
}

// Save mock logic...
func (mock *Mock) Save(
	ctx context.Context,
	key string,
	value []byte,
	expire time.Duration,
) error {
	atomic.AddUint32(&mock.saveCallsCnt, 1)
	if mock.saveCallback != nil {
		return mock.saveCallback(ctx, key, value, expire)
	}

	return nil
}

// Load mock logic...
func (mock *Mock) Load(ctx context.Context, key string) ([]byte, error) {
	atomic.AddUint32(&mock.loadCallsCnt, 1)
	if mock.loadCallback != nil {
		return mock.loadCallback(ctx, key)
	}

	return nil, ErrNotFound
}

// TTL mock logic...
func (mock *Mock) TTL(ctx context.Context, key string) (time.Duration, error) {
	atomic.AddUint32(&mock.ttlCallsCnt, 1)
	if mock.ttlCallback != nil {
		return mock.ttlCallback(ctx, key)
	}

	return -1, nil
}

// Stats mock logic...
func (mock *Mock) Stats(ctx context.Context) (Stats, error) {
	atomic.AddUint32(&mock.statsCallsCnt, 1)
	if mock.statsCallback != nil {
		return mock.statsCallback(ctx)
	}

	return Stats{}, nil
}

// SetSaveCallback sets the given callback to be executed inside Save() method.
// You can inject yourself to make assertions upon passed parameter(s) this way
// and/or control the returned value.
//
// Usage example:
//
//	mock.SetSaveCallback(func(ctx context.Context, k string, v []byte, exp time.Duration) {
//		if k != "expected-key" {
//			t.Error("expected ...")
//		}
//		if !reflect.DeepEqual(v, []byte("expected value")) {
//			t.Error("expected ...")
//		}
//		if exp != 10 * time.Minute {
//			t.Error("expected ...")
//		}
//
//		return nil
//	})
func (mock *Mock) SetSaveCallback(callback func(context.Context, string, []byte, time.Duration) error) {
	mock.saveCallback = callback
}

// SetLoadCallback sets the given callback to be executed inside Load() method.
// You can inject yourself to make assertions upon passed parameter(s) this way
// and/or control the returned value.
//
// Usage example:
//
//	mock.SetLoadCallback(func(ctx context.Context, k string) ([]byte, error) {
//		if k != "expected-key" {
//			t.Error("expected ...")
//		}
//
//		return []byte("expected value"), nil
//	})
func (mock *Mock) SetLoadCallback(callback func(context.Context, string) ([]byte, error)) {
	mock.loadCallback = callback
}

// SetTTLCallback sets the given callback to be executed inside TTL() method.
// You can inject yourself to make assertions upon passed parameter(s) this way
// and/or control the returned value.
//
// Usage example:
//
//	mock.SetTTLCallback(func(ctx context.Context, k string) (time.Duration, error) {
//		if k != "expected-key" {
//			t.Error("expected ...")
//		}
//
//		return 123 * time.Second, nil
//	})
func (mock *Mock) SetTTLCallback(callback func(context.Context, string) (time.Duration, error)) {
	mock.ttlCallback = callback
}

// SetStatsCallback sets the given callback to be executed inside Stats() method.
// You can inject yourself to make assertions upon passed parameter(s) this way
// and/or control the returned value.
//
// Usage example:
//
//	mock.SetStatsCallback(func(ctx context.Context) (xcache.Stats, error) {
//		if ctx != context.Background() {
//			t.Error("expected ...")
//		}
//
//		return xcache.Stats{Memory: 1024}, nil
//	})
func (mock *Mock) SetStatsCallback(callback func(context.Context) (Stats, error)) {
	mock.statsCallback = callback
}

// SaveCallsCount returns the no. of times Save() method was called.
func (mock *Mock) SaveCallsCount() int {
	return int(atomic.LoadUint32(&mock.saveCallsCnt))
}

// LoadCallsCount returns the no. of times Load() method was called.
func (mock *Mock) LoadCallsCount() int {
	return int(atomic.LoadUint32(&mock.loadCallsCnt))
}

// TTLCallsCount returns the no. of times TTL() method was called.
func (mock *Mock) TTLCallsCount() int {
	return int(atomic.LoadUint32(&mock.ttlCallsCnt))
}

// StatsCallsCount returns the no. of times Stats() method was called.
func (mock *Mock) StatsCallsCount() int {
	return int(atomic.LoadUint32(&mock.statsCallsCnt))
}
