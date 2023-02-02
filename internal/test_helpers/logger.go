package test_helpers

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"testing"
)

// testWriterSyncer is used by tests
// as an output for zap logger
type testWriterSyncer struct {
	t *testing.T
}

func (l *testWriterSyncer) Write(p []byte) (n int, err error) {
	msg := string(p)
	l.t.Logf("%s", msg)
	return len(msg), nil
}

func (l *testWriterSyncer) Sync() error {
	return nil
}

func NewTestZapLogger(t *testing.T) *zap.Logger {
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "logger",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		&testWriterSyncer{t: t},
		zap.DebugLevel,
	)
	return zap.New(core)
}
