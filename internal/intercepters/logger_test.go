package intercepters_test

import (
	"context"
	"testing"

	"github.com/atinyakov/go-url-shortener/internal/intercepters"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestInterceptorLogger(t *testing.T) {
	// Create a zap observer to capture logs
	core, observedLogs := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	il := intercepters.InterceptorLogger(logger)

	ctx := context.Background()

	tests := []struct {
		name     string
		level    logging.Level
		msg      string
		fields   []any
		wantLvl  zapcore.Level
		wantMsg  string
		wantKeys []string
	}{
		{
			name:    "Info level with string and int fields",
			level:   logging.LevelInfo,
			msg:     "test info",
			fields:  []any{"key1", "value1", "key2", 42},
			wantLvl: zap.InfoLevel,
			wantMsg: "test info",
			wantKeys: []string{
				"key1",
				"key2",
			},
		},
		{
			name:    "Debug level with bool field",
			level:   logging.LevelDebug,
			msg:     "debug message",
			fields:  []any{"enabled", true},
			wantLvl: zap.DebugLevel,
			wantMsg: "debug message",
			wantKeys: []string{
				"enabled",
			},
		},
		{
			name:    "Warn level with unknown field type",
			level:   logging.LevelWarn,
			msg:     "warn message",
			fields:  []any{"data", struct{ A int }{A: 1}},
			wantLvl: zap.WarnLevel,
			wantMsg: "warn message",
			wantKeys: []string{
				"data",
			},
		},
		{
			name:    "Error level no fields",
			level:   logging.LevelError,
			msg:     "error occurred",
			fields:  nil,
			wantLvl: zap.ErrorLevel,
			wantMsg: "error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear previous logs before call
			observedLogs.TakeAll()

			il.Log(ctx, tt.level, tt.msg, tt.fields...)

			logs := observedLogs.TakeAll()
			if len(logs) != 1 {
				t.Fatalf("expected 1 log entry, got %d", len(logs))
			}
			logEntry := logs[0]

			if logEntry.Level != tt.wantLvl {
				t.Errorf("got level %v, want %v", logEntry.Level, tt.wantLvl)
			}
			if logEntry.Message != tt.wantMsg {
				t.Errorf("got message %q, want %q", logEntry.Message, tt.wantMsg)
			}

			// Check keys present in fields
			for _, key := range tt.wantKeys {
				found := false
				for _, f := range logEntry.Context {
					if f.Key == key {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected field key %q not found in log context", key)
				}
			}
		})
	}
}

func TestInterceptorLogger_UnknownLevelPanics(t *testing.T) {
	core, _ := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	il := intercepters.InterceptorLogger(logger)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for unknown logging level, but did not panic")
		}
	}()

	il.Log(context.Background(), logging.Level(999), "panic test")
}
