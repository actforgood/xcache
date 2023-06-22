// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/blob/main/LICENSE.

package xcache_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/actforgood/xcache"
	"github.com/actforgood/xlog"
)

func TestRedisXLogger(t *testing.T) {
	t.Parallel()

	t.Run("error message", testRedisXLoggerByLevel(xlog.LevelError))
	t.Run("info message", testRedisXLoggerByLevel(xlog.LevelInfo))
}

func testRedisXLoggerByLevel(lvl xlog.Level) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		// arrange
		var (
			logger          = xlog.NewMockLogger()
			subject         = xcache.NewRedisXLogger(logger)
			ctx             = context.Background()
			foundNeededInfo = 0
			expectedFormat  = map[xlog.Level]string{
				xlog.LevelInfo:  "some redis message about master=%q",
				xlog.LevelError: "some redis message about master=%q failed due some err",
			}
			masterName  = "testMaster"
			logCallback = func(expectedMsg string) func(keyValues ...any) {
				return func(keyValues ...any) {
					for i := 0; i < len(keyValues); i += 2 {
						if keyValues[i] == xlog.MessageKey {
							assertEqual(t, expectedMsg, keyValues[i+1])
							foundNeededInfo++
						} else if keyValues[i] == "pkg" {
							assertEqual(t, "redis", keyValues[i+1])
							foundNeededInfo++
						}
					}
				}
			}
		)
		defer logger.Close()
		logger.SetLogCallback(lvl, logCallback(fmt.Sprintf(expectedFormat[lvl], masterName)))

		// act
		subject.Printf(ctx, expectedFormat[lvl], masterName)

		// assert
		assertEqual(t, 1, logger.LogCallsCount(lvl))
		assertEqual(t, 2, foundNeededInfo)
	}
}

func ExampleRedisXLogger() {
	// somewhere in your bootstrap process...

	// initialize an xlog.Logger
	loggerOpts := xlog.NewCommonOpts()
	loggerOpts.MinLevel = xlog.FixedLevelProvider(xlog.LevelInfo)
	loggerOpts.Source = xlog.SourceProvider(5, 1)
	logger := xlog.NewSyncLogger(os.Stdout, xlog.SyncLoggerWithOptions(loggerOpts))
	// set the xlog.Logger Redis adapter
	redisLogger := xcache.NewRedisXLogger(logger)
	xcache.SetRedis6Logger(redisLogger) // or xcache.SetRedis7Logger(redisLogger),
	// depending which ver. of Redis you're using.

	// somewhere in your shutdown process ...
	_ = logger.Close()
}

func BenchmarkRedisXLogger(b *testing.B) {
	logger := xlog.NopLogger{}
	redisLogger := xcache.NewRedisXLogger(logger)
	message := "some redis message about master=%q failed due some err"
	masterName := "benchLoggerMaster"
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		redisLogger.Printf(ctx, message, masterName)
	}
}
