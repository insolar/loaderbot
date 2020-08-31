/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.SugaredLogger
}

func (m *Logger) With(args ...interface{}) *Logger {
	m.SugaredLogger = m.SugaredLogger.With(args...)
	return m
}

func (m *Logger) Clone() Logger {
	return *m
}

func setupLogger(encoding string, level string) *Logger {
	rawJSON := []byte(fmt.Sprintf(`{
	  "level": "%s",
	  "encoding": "%s",
	  "outputPaths": ["stdout"],
	  "errorOutputPaths": ["stderr"],
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
		"levelEncoder": "uppercase",
        "timeKey": "time",
		"timeEncoder": "ISO8601",
		"callerKey": "caller",
		"callerEncoder": "short"
	  }
	}`, level, encoding))

	var cfg zap.Config
	if err := jsoniter.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	_ = logger.Sync()
	return &Logger{logger.Sugar()}
}

func NewLogger(cfg *RunnerConfig) *Logger {
	return setupLogger(cfg.LogEncoding, cfg.LogLevel)
}
