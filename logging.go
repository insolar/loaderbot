package loaderbot

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
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
	defer logger.Sync()
	return &Logger{logger.Sugar()}
}

func NewLogger() *Logger {
	lvl := viper.GetString("logging.level")
	if lvl == "" {
		lvl = "info"
	}
	encoding := viper.GetString("logging.encoding")
	if encoding == "" {
		encoding = "console"
	}
	return setupLogger(encoding, lvl)
}
