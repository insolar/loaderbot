package loaderbot

import (
	"encoding/json"
	"fmt"

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
	  "outputPaths": ["stdout", "/tmp/logs"],
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
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
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
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.LogEncoding == "" {
		cfg.LogEncoding = "console"
	}
	return setupLogger(cfg.LogEncoding, cfg.LogLevel)
}
