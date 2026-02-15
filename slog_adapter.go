// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/blob/main/LICENSE.

package xcache

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	redis6 "github.com/go-redis/redis/v8"
	redis7 "github.com/redis/go-redis/v9"
)

// RedisSLogger is a log/slog adapter for Redis internal logging contract.
// Redis default logger has an unstructured format (and relies upon standard Go Logger).
// Through this adapter, you can achieve a structured output of the log as a whole,
// but the message inside will still be unstructured. See also Printf method doc.
type RedisSLogger struct {
	logger *slog.Logger
}

// NewRedisSLogger instantiates a new RedisSLogger object.
func NewRedisSLogger(logger *slog.Logger) RedisSLogger {
	return RedisSLogger{
		logger: logger,
	}
}

// Printf implements redis pkg internal.Logging contract,
// see also https://github.com/redis/go-redis/blob/v8.11.5/internal/log.go .
//
// Example of default redis logger output (which goes to StdErr):
//
//	redis: 2022/07/29 07:16:34 sentinel.go:661: sentinel: new master="xcacheMaster" addr="some-redis-master:6380"
//
// Example of RedisSLogger output:
//
//	{"date":"2022-07-29T09:07:54.915902723Z","level":"INFO","msg":"sentinel: new master=\"xcacheMaster\" addr=\"some-redis-master:6380\"","pkg":"redis","src":"/sentinel.go:661"}
//
// Method categorizes the message as error/info based on presence of some words
// like "failed"/"error".
// nolint:lll
func (l RedisSLogger) Printf(_ context.Context, format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	if strings.Contains(msg, "failed") || strings.Contains(msg, "error") {
		l.logger.Error(msg, "pkg", "redis")
	} else {
		l.logger.Info(msg, "pkg", "redis")
	}
}

// SetRedis6Logger sets given slog logger for a Redis6 client.
func SetRedis6Logger(redisSLogger RedisSLogger) {
	redis6.SetLogger(redisSLogger)
}

// SetRedis7Logger sets given slog logger for a Redis7 client.
func SetRedis7Logger(redisSLogger RedisSLogger) {
	redis7.SetLogger(redisSLogger)
}
