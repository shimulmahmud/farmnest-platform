package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNew_ValidConfig(t *testing.T) {
	log, err := New("info", "json")
	assert.NoError(t, err)
	assert.NotNil(t, log)
	defer log.Sync()
}

func TestNew_ConsoleEncoding(t *testing.T) {
	log, err := New("debug", "console")
	assert.NoError(t, err)
	assert.NotNil(t, log)
	defer log.Sync()
}

func TestNew_InvalidLevel(t *testing.T) {
	_, err := New("not-a-level", "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log level")
}

func TestNew_InvalidEncoding(t *testing.T) {
	_, err := New("info", "xml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported log encoding")
}

func TestLogger_With(t *testing.T) {
	log, _ := New("info", "json")
	defer log.Sync()
	child := log.With("component", "test")
	assert.NotNil(t, child)
}

func TestLogger_Named(t *testing.T) {
	log, _ := New("info", "json")
	defer log.Sync()
	child := log.Named("subsystem")
	assert.NotNil(t, child)
}

func TestLogger_Zap(t *testing.T) {
	log, _ := New("info", "json")
	defer log.Sync()
	assert.NotNil(t, log.Zap())
	assert.IsType(t, &zap.Logger{}, log.Zap())
}

func TestLogger_LogMethods(t *testing.T) {
	log, _ := New("debug", "json")
	defer log.Sync()

	// These should not panic
	log.Debug("debug message")
	log.Info("info message")
	log.Warn("warn message")
	log.Infow("structured", "key", "value", "count", 42)
	log.Debugw("debug structured", "component", "test")
}
