// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/LICENSE.

package xcache_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/actforgood/xcache"
)

func TestStats_String(t *testing.T) {
	t.Parallel()

	// arrange
	tests := [...]struct {
		name           string
		subject        xcache.Stats
		expectedResult interface{}
	}{
		{
			name: "mb memory/gb memory",
			subject: xcache.Stats{
				Memory:    10.5 * 1024 * 1024,
				MaxMemory: 4 * 1024 * 1024 * 1024,
				Hits:      50,
				Misses:    100,
				Keys:      355,
				Expired:   129,
				Evicted:   3,
			},
			expectedResult: "mem=10.50M maxMem=4G memUsage=0.26% hits=50 misses=100 hitRate=33.33% keys=355 expired=129 evicted=3",
		},
		{
			name: "b memory/kb memory",
			subject: xcache.Stats{
				Memory:    999,
				MaxMemory: 1998,
				Hits:      30,
				Misses:    70,
				Keys:      1,
				Expired:   0,
				Evicted:   0,
			},
			expectedResult: "mem=999B maxMem=1.95K memUsage=50.00% hits=30 misses=70 hitRate=30.00% keys=1 expired=0 evicted=0",
		},
		{
			name: "tb memory, no max mem, no hits, no misses",
			subject: xcache.Stats{
				Memory:    1024 * 1024 * 1024 * 1024,
				MaxMemory: 0,
				Hits:      0,
				Misses:    0,
				Keys:      1001,
				Expired:   1000002,
				Evicted:   50000,
			},
			expectedResult: "mem=1T maxMem=0B memUsage=100.00% hits=0 misses=0 hitRate=100.00% keys=1001 expired=1000002 evicted=50000",
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			// act
			result1 := test.subject.String()
			result2 := fmt.Sprint(test.subject)

			// assert
			assertEqual(t, result2, result1)
			assertEqual(t, test.expectedResult, result1)
		})
	}
}

func TestStatsWatcher(t *testing.T) {
	t.Parallel()

	t.Run("callback is executed periodically", testStatsWatcherCallbackIsExecutedPeriodically)
	t.Run("Close stops watching", testStatsWatcherCloseStopsWatching)
	t.Run("cancel context stops watching", testStatsWatcherCancelContextStopsWatching)
	t.Run("finalizer is called", testStatsWatcherFinalizerIsCalled)
}

func testStatsWatcherCallbackIsExecutedPeriodically(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache          = new(xcache.Mock)
		subject        = xcache.NewStatsWatcher(cache, 200*time.Millisecond)
		ctx            = context.Background()
		expectedStats1 = xcache.Stats{
			Memory:    1024,
			MaxMemory: 2048,
			Hits:      100,
			Misses:    1001,
			Keys:      300,
			Expired:   200,
			Evicted:   900,
		}
		expectedStats2 = xcache.Stats{
			Memory:    4096,
			MaxMemory: 4096,
			Hits:      1,
			Misses:    2,
			Keys:      3,
			Expired:   4,
			Evicted:   5,
		}
		expectedErr = errors.New("intentionally triggered Stats error")
		callsCnt    uint32
		fn          = func(ctxx context.Context, s xcache.Stats, err error) {
			atomic.AddUint32(&callsCnt, 1)
			assertEqual(t, ctx, ctxx)
			switch atomic.LoadUint32(&callsCnt) {
			case 1:
				assertEqual(t, expectedStats1, s)
				assertNil(t, err)
			case 2:
				assertEqual(t, xcache.Stats{}, s)
				assertTrue(t, errors.Is(err, expectedErr))
			case 3:
				assertEqual(t, expectedStats2, s)
				assertNil(t, err)
			}
		}
	)
	defer subject.Close()
	cache.SetStatsCallback(func(ctxx context.Context) (xcache.Stats, error) {
		assertEqual(t, ctx, ctxx)
		switch cache.StatsCallsCount() {
		case 1:
			return expectedStats1, nil
		case 2:
			return xcache.Stats{}, expectedErr
		case 3:
			return expectedStats2, nil
		}

		return xcache.Stats{}, nil
	})

	// act
	subject.Watch(ctx, fn)

	// assert
	time.Sleep(700 * time.Millisecond)
	assertEqual(t, 3, cache.StatsCallsCount())
	assertEqual(t, uint32(3), atomic.LoadUint32(&callsCnt))
}

func testStatsWatcherCloseStopsWatching(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache    = new(xcache.Mock)
		subject  = xcache.NewStatsWatcher(cache, 500*time.Millisecond)
		callsCnt uint32
		fn       = func(context.Context, xcache.Stats, error) {
			atomic.AddUint32(&callsCnt, 1)
		}
	)
	subject.Watch(context.Background(), fn)

	// act
	time.Sleep(50 * time.Millisecond)
	err := subject.Close()
	time.Sleep(700 * time.Millisecond)

	// assert
	assertNil(t, err)
	assertEqual(t, 0, cache.StatsCallsCount())
	assertEqual(t, uint32(0), atomic.LoadUint32(&callsCnt))
}

func testStatsWatcherCancelContextStopsWatching(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		cache          = new(xcache.Mock)
		ctx, cancelCtx = context.WithCancel(context.Background())
		subject        = xcache.NewStatsWatcher(cache, 500*time.Millisecond)
		callsCnt       uint32
		fn             = func(context.Context, xcache.Stats, error) {
			atomic.AddUint32(&callsCnt, 1)
		}
	)
	subject.Watch(ctx, fn)

	// act
	time.Sleep(50 * time.Millisecond)
	cancelCtx()
	time.Sleep(700 * time.Millisecond)

	// assert
	assertEqual(t, 0, cache.StatsCallsCount())
	assertEqual(t, uint32(0), atomic.LoadUint32(&callsCnt))
}

func testStatsWatcherFinalizerIsCalled(t *testing.T) {
	// test finalizer is called if we "forget" to call Close.
	// arrange
	var (
		cache    = new(xcache.Mock)
		subject  = xcache.NewStatsWatcher(cache, 500*time.Millisecond)
		callsCnt uint32
		fn       = func(context.Context, xcache.Stats, error) {
			atomic.AddUint32(&callsCnt, 1)
		}
	)
	subject.Watch(context.Background(), fn)

	// act
	time.Sleep(50 * time.Millisecond)
	runtime.GC()
	time.Sleep(700 * time.Millisecond)

	// assert
	assertEqual(t, 0, cache.StatsCallsCount())
	assertEqual(t, uint32(0), atomic.LoadUint32(&callsCnt))
}

func BenchmarkStats_String(b *testing.B) {
	stats := xcache.Stats{
		Memory:    512 * 1024,
		MaxMemory: 1.5 * 1024 * 1024,
		Hits:      9000000,
		Misses:    1000000,
		Keys:      9000000,
		Expired:   14000000,
		Evicted:   10000000,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_ = stats.String()
	}
}

func ExampleStatsWatcher() {
	// initialize our application cache...
	cache := xcache.NewMemory(10 * 1024 * 1024) // 10 Mb
	ctx, cancelCtx := context.WithCancel(context.Background())

	// perform some operations upon cache to have some data...
	var wg sync.WaitGroup
	wg.Add(1)
	go generateRandomStats(ctx, cache, &wg)

	// initialize our stats watcher, which will exceute a logging callback every second.
	subject := xcache.NewStatsWatcher(cache, time.Second)
	defer subject.Close() // close your watcher! (at your app shutdown eventually)

	// start watching
	subject.Watch(
		context.Background(),
		func(_ context.Context, stats xcache.Stats, err error) {
			if err != nil {
				fmt.Println("could not get cache stats:" + err.Error())
			} else { // do something useful with stats, log / sent it to a metrics system...
				fmt.Println(stats)
			}
		},
	)

	time.Sleep(3 * time.Second)
	cancelCtx() // cancel data generator goroutine
	wg.Wait()   // wait for data generator goroutine to finish

	// should output periodically something like:
	// mem=10M maxMem=10M memUsage=100.00% hits=10 misses=1 hitRate=90.91% keys=10 expired=0 evicted=0
}

func generateRandomStats(ctx context.Context, cache xcache.Cache, wg *sync.WaitGroup) {
	defer wg.Done()

	keyPrefix := "example-stats-watcher-"
	value := []byte("Hello Memory Cache")
	ttl := 5 * time.Minute

	for {
		select {
		case <-ctx.Done():
			return
		default:
			randLoop := rand.Intn(10)
			for i := 0; i <= randLoop; i++ {
				key := keyPrefix + strconv.FormatInt(time.Now().UnixNano(), 10)
				_ = cache.Save(ctx, key, value, ttl)
				_, _ = cache.Load(ctx, key)
			}
			_, _ = cache.Load(ctx, keyPrefix+"miss")
			time.Sleep(100 * time.Millisecond)
		}
	}
}
