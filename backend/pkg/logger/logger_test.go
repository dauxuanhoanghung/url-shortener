package logger

import (
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestNew_ReleaseMode(t *testing.T) {
	log, err := New("release")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
	// Production config logs at InfoLevel, so a DebugLevel check should be disabled.
	if log.Core().Enabled(zapcore.DebugLevel) {
		t.Error("release logger should not have Debug enabled (production config is Info+)")
	}
	if !log.Core().Enabled(zapcore.InfoLevel) {
		t.Error("release logger should have Info enabled")
	}
}

func TestNew_DevelopmentMode(t *testing.T) {
	// Any non-"release" value should select development config.
	for _, mode := range []string{"debug", "", "test", "anything-else"} {
		t.Run(mode, func(t *testing.T) {
			log, err := New(mode)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if log == nil {
				t.Fatal("expected non-nil logger")
			}
			if !log.Core().Enabled(zapcore.DebugLevel) {
				t.Error("development logger should have Debug enabled")
			}
		})
	}
}
