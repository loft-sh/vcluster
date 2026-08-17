package v2

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestPluginLoggerForwardsToZap(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	logger := newPluginLogger(zap.New(core))

	logger.With("pluginPath", "/plugins/example/plugin").Warn("plugin connection failed", "attempt", 3)

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("expected one central log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Level != zapcore.WarnLevel {
		t.Fatalf("expected warning level, got %s", entry.Level)
	}
	if entry.LoggerName != "plugin" {
		t.Fatalf("expected plugin logger, got %q", entry.LoggerName)
	}
	if entry.Message != "plugin connection failed" {
		t.Fatalf("expected forwarded message, got %q", entry.Message)
	}

	fields := entry.ContextMap()
	if fields["pluginPath"] != "/plugins/example/plugin" {
		t.Fatalf("expected pluginPath field, got %#v", fields["pluginPath"])
	}
	if fields["attempt"] != int64(3) {
		t.Fatalf("expected attempt field, got %#v", fields["attempt"])
	}
}

func TestPluginLoggerFiltersVerboseLogs(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	logger := newPluginLogger(zap.New(core))

	logger.Debug("plugin debug output")
	logger.Trace("plugin trace output")

	if entries := observed.All(); len(entries) != 0 {
		t.Fatalf("expected verbose plugin logs to be filtered, got %#v", entries)
	}
}
