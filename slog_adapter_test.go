// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/blob/main/LICENSE.

package xcache_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"

	"github.com/actforgood/xcache"
)

// MockKeyNotFound is the type of value returned by [LogHandlerMock.ValueAt]
// in case the searched key is not found.
type MockKeyNotFound struct{}

// MockAny can be passed to [logHandlerMock.ValueAt] as call index,
// in case the order is not known/important/multiple logs happen concurrently.
const MockAny uint = 0

// LogHandlerMock is a mock for [slog.Handler],
// which allows us to intercept the log calls and assert on the log content.
type LogHandlerMock struct {
	loggedKeyValues [][]slog.Attr
	logCallsCnt     map[slog.Level]uint32
	mu              sync.RWMutex
}

// NewLogHandlerMock instantiates a new LogHandlerMock object.
func NewLogHandlerMock() *LogHandlerMock {
	return &LogHandlerMock{
		logCallsCnt: make(map[slog.Level]uint32, 4),
	}
}

func (mock *LogHandlerMock) Handle(_ context.Context, record slog.Record) error {
	mock.mu.Lock()
	defer mock.mu.Unlock()

	lvl := record.Level
	mock.logCallsCnt[lvl]++

	// default behaviour is to store values in a slice.
	// values can be retrieved later with ValueAt.
	attrs := make([]slog.Attr, 0, record.NumAttrs()+1)
	attrs = append(attrs, slog.Attr{Key: "msg", Value: slog.StringValue(record.Message)})
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)

		return true
	})
	mock.loggedKeyValues = append(mock.loggedKeyValues, attrs)

	return nil
}

func (mock *LogHandlerMock) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (mock *LogHandlerMock) WithAttrs(_ []slog.Attr) slog.Handler {
	return mock
}

func (mock *LogHandlerMock) WithGroup(_ string) slog.Handler {
	return mock
}

// ValueAt returns the value for a key at given call, in case no callback was set.
// Calls are positive numbers (starting with 1).
// If the order of the calls is not known/important/multiple logs happen concurrently,
// you can use [MockAtAnyCall].
// If the key is not found,  [MockKeyNotFound] is returned.
func (mock *LogHandlerMock) ValueAt(callNo uint, forKey string) any {
	mock.mu.RLock()
	defer mock.mu.RUnlock()

	if callNo == MockAny {
		for call := range len(mock.loggedKeyValues) {
			value := mock.valueAt(call+1, forKey)
			if _, isNotFound := value.(MockKeyNotFound); !isNotFound {
				return value
			}
		}

		return MockKeyNotFound{}
	}

	return mock.valueAt(int(callNo), forKey)
}

func (mock *LogHandlerMock) valueAt(callNo int, forKey any) any {
	if len(mock.loggedKeyValues) >= callNo {
		for _, attr := range mock.loggedKeyValues[callNo-1] {
			if attr.Key == forKey {
				return attr.Value.Any()
			}
		}
	}

	return MockKeyNotFound{}
}

// LogCallsCount returns the no. of times Critical/Error/Warn/Info/Debug/Log was called.
// Differentiate methods calls count by passing appropriate level.
func (mock *LogHandlerMock) LogCallsCount(lvl slog.Level) int {
	mock.mu.RLock()
	defer mock.mu.RUnlock()

	return int(mock.logCallsCnt[lvl])
}

// LogHandlerNop is a no-op implementation of [slog.Handler], which does nothing.
type LogHandlerNop struct{}

// NewLogHandlerNop instantiates a new LogHandlerNop object.
func NewLogHandlerNop() LogHandlerNop {
	return LogHandlerNop{}
}

func (mock LogHandlerNop) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

func (mock LogHandlerNop) Enabled(_ context.Context, _ slog.Level) bool {
	return false // disable loggin
}

func (mock LogHandlerNop) WithAttrs(_ []slog.Attr) slog.Handler {
	return mock
}

func (mock LogHandlerNop) WithGroup(_ string) slog.Handler {
	return mock
}

func TestRedisSLogger(t *testing.T) {
	t.Parallel()

	t.Run("error message", testRedisSLoggerByLevel(slog.LevelError))
	t.Run("info message", testRedisSLoggerByLevel(slog.LevelInfo))
}

func testRedisSLoggerByLevel(lvl slog.Level) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		// arrange
		var (
			logHandlerMock = NewLogHandlerMock()
			logger         = slog.New(logHandlerMock)
			subject        = xcache.NewRedisSLogger(logger)
			ctx            = context.Background()
			expectedFormat = map[slog.Level]string{
				slog.LevelInfo:  "some redis message about master=%q",
				slog.LevelError: "some redis message about master=%q failed due some err",
			}
			masterName  = "testMaster"
			expectedMsg = fmt.Sprintf(expectedFormat[lvl], masterName)
		)

		// act
		subject.Printf(ctx, expectedFormat[lvl], masterName)

		// assert
		if assertEqual(t, 1, logHandlerMock.LogCallsCount(lvl)) {
			assertEqual(t, expectedMsg, logHandlerMock.ValueAt(1, "msg"))
			assertEqual(t, "redis", logHandlerMock.ValueAt(1, "pkg"))
		}
	}
}

func ExampleRedisSLogger() {
	// somewhere in your bootstrap process...

	// initialize an slog.Logger, here we use the default one...
	logger := slog.Default()
	// set the slog.Logger Redis adapter
	xcache.SetRedis7Logger(xcache.NewRedisSLogger(logger))
}

func BenchmarkRedisSLogger(b *testing.B) {
	logger := slog.New(LogHandlerNop{})
	redisLogger := xcache.NewRedisSLogger(logger)
	message := "some redis message about master=%q failed due some err"
	masterName := "benchLoggerMaster"
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		redisLogger.Printf(ctx, message, masterName)
	}
}
