// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/LICENSE.

package xcache

import (
	"context"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// Stats holds memory and keys statistics.
//
// They can be useful to be reported to metrics systems like Prometheus / DataDog, or they
// can just be used for debug purposes.
//
// Note: As Redis is distributed, stats information is shared (and affected) between/by all services that use it.
type Stats struct {
	// Memory represents the in use memory.
	// Notes:
	// - for Memory Cache it's equal to the memory size used to initialize the cache,
	// as Freecache allocates that amount of memory from the start. Thus, Memory is always equal to MaxMemory.
	// To figure out that the memory is effectively full, a raise in Evicted number of keys should be considered.
	// - for Redis Cache it's the used memory.
	Memory int64
	// MaxMemory represents the maximum memory.
	// Notes:
	// - for Memory Cache it's equal to the memory size used to initialize the cache.
	// - for Redis Cache it's the max memory Redis was configured with, or system total memory, if max memory is 0.
	// On a Redis Cluster configuration, it's calculated as the sum of max memory or system total memory of all masters.
	MaxMemory int64
	// Hits represents the number of successful accesses of keys.
	// Notes:
	// - for Redis Cache, also TTL calls to a key are reported, for Memory Cache this does not happen
	// (if you need this consistency, you can make your own Memory Cache and use Freecache's GetWithExpiration api
	// for TTL implementation - not used here as it's more costly than TTL api).
	Hits int64
	// Misses represents the number of times keys were not found.
	// Notes:
	// - for Redis Cache, also TTL calls to a not found key are reported, for Memory Cache this does not happen
	// (if you need this consistency, you can make your own Memory Cache and use Freecache's GetWithExpiration api
	// for TTL implementation - not used here as it's more costly than TTL api).
	Misses int64
	// Keys represents the current number of keys in cache.
	// Notes:
	// - for Redis Cache, if you have a Redis Cluster, this will be 0.
	Keys int64
	// Expired represents the number of expired keys reported by cache.
	Expired int64
	// Evicted represents the number of evicted keys reported by cache.
	Evicted int64
}

// String implements fmt.Stringer.
// Returns a human friendly stats representation.
//
// Example:
//
//	mem=1.25M maxMem=7.77G memPerc=0.02% hits=101701 misses=0 hitRate=100.00% keys=1 expired=14473 evicted=0
func (s Stats) String() string {
	buf := make([]byte, 0, 128)
	buf = append(buf, "mem="...)
	buf = append(buf, bytesHumanFriendly(s.Memory)...)
	buf = append(buf, " maxMem="...)
	buf = append(buf, bytesHumanFriendly(s.MaxMemory)...)

	memPerc := 100.0
	if s.MaxMemory > 0 {
		memPerc = float64(s.Memory) / float64(s.MaxMemory) * 100
	}
	buf = append(buf, " memUsage="...)
	buf = append(buf, strconv.FormatFloat(memPerc, 'f', 2, 32)...)
	buf = append(buf, '%')
	buf = append(buf, " hits="...)
	buf = append(buf, strconv.FormatInt(s.Hits, 10)...)
	buf = append(buf, " misses="...)
	buf = append(buf, strconv.FormatInt(s.Misses, 10)...)

	lookups := s.Hits + s.Misses
	hitRatePerc := 100.0
	if lookups > 0 {
		hitRatePerc = float64(s.Hits) / float64(lookups) * 100
	}
	buf = append(buf, " hitRate="...)
	buf = append(buf, strconv.FormatFloat(hitRatePerc, 'f', 2, 32)...)
	buf = append(buf, '%')
	buf = append(buf, " keys="...)
	buf = append(buf, strconv.FormatInt(s.Keys, 10)...)
	buf = append(buf, " expired="...)
	buf = append(buf, strconv.FormatInt(s.Expired, 10)...)
	buf = append(buf, " evicted="...)
	buf = append(buf, strconv.FormatInt(s.Evicted, 10)...)

	return bytesToString(buf)
}

// bytesHumanFriendly returns bytes converted to easier to read value.
// Example: bytesHumanFriendly(2 * 1024 * 1024) => "2M" .
func bytesHumanFriendly(bytes int64) string {
	const (
		unit    = 1024
		measure = "BKMGTPE"
	)
	var (
		div     = 1
		exp     = 0
		buf     []byte
		bufSize = 8 // max 4 numbers + dot + 2 for precision + final char, example 1023.90M
		prec    = 2
	)

	for n := bytes; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	result := float64(bytes) / float64(div)
	if result-float64(int(result)) < 0.001 { // don't show ".00" precision if it's integer.
		prec = 0
		bufSize = 5 // max 4 numbers + final char, example 1023B
	}

	buf = make([]byte, 0, bufSize)
	buf = append(buf, strconv.FormatFloat(result, 'f', prec, 32)...)
	buf = append(buf, measure[exp])

	return bytesToString(buf)
}

// StatsWatcher can be used to execute a given callback
// upon stats, interval based.
// It implements io.Closer and should be closed at your application shutdown.
type StatsWatcher struct {
	*watcher  // so we can use finalizer
	watchOnce sync.Once
	closeOnce sync.Once
}

type watcher struct {
	interval time.Duration
	ticker   *time.Ticker
	wg       sync.WaitGroup // used to notify that goroutine has finished
	closed   chan struct{}  // used to notify the goroutine to finish
	cache    Cache          // watched cache stats
}

// NewStatsWatcher instantiates a new StatsWatcher object.
func NewStatsWatcher(cache Cache, interval time.Duration) *StatsWatcher {
	return &StatsWatcher{
		watcher: &watcher{
			interval: interval,
			cache:    cache,
		},
	}
}

// Watch executes fn asynchronously, interval based.
// Calling Watch multiple times has no effect.
func (sw *StatsWatcher) Watch(ctx context.Context, fn func(context.Context, Stats, error)) {
	sw.watchOnce.Do(func() {
		sw.watcher.watch(ctx, fn)
		// register also a finalizer, just in case, user forgets to call Close().
		// Note: user should do not rely on this, it's recommended to explicitly call Close().
		runtime.SetFinalizer(sw, (*StatsWatcher).Close)
	})
}

// Close stops the underlying ticker used to execute the callback, interval based, avoiding memory leaks.
// It should be called at your application shutdown.
// It implements io.Closer interface, and the returned error can be disregarded (is nil all the time).
func (sw *StatsWatcher) Close() error {
	if sw != nil && sw.ticker != nil {
		sw.closeOnce.Do(func() {
			sw.watcher.close()
			runtime.SetFinalizer(sw, nil)
		})
	}

	return nil
}

// watch executes fn, interval based.
func (w *watcher) watch(ctx context.Context, fn func(context.Context, Stats, error)) {
	w.ticker = time.NewTicker(w.interval)
	w.closed = make(chan struct{}, 1)
	w.wg.Add(1)
	go w.watchAsync(ctx, fn)
}

// watchAsync executes fn asynchronous, interval based.
// Calling Close() will stop this goroutine, or using a cancel context for example.
func (w *watcher) watchAsync(ctx context.Context, fn func(context.Context, Stats, error)) {
	defer w.ticker.Stop()
	defer w.wg.Done()

	for {
		select {
		case <-w.closed:
			return
		case <-ctx.Done():
			return
		case <-w.ticker.C:
			stats, err := w.cache.Stats(ctx)
			fn(ctx, stats, err)
		}
	}
}

// close stops the underlying ticker used to execute the callback, interval based, avoiding memory leaks.
func (w *watcher) close() {
	if w != nil {
		close(w.closed)
		w.wg.Wait()
	}
}
