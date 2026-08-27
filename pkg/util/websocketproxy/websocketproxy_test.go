package websocketproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/zapr"
	"github.com/loft-sh/vcluster/pkg/util/logtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"k8s.io/klog/v2"
)

func TestServeHTTPWritesJSONFile(t *testing.T) {
	logger, path := logtest.NewFileJSONLogger(t)
	request := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	request = request.WithContext(klog.NewContext(request.Context(), logger))

	(&WebsocketProxy{}).ServeHTTP(httptest.NewRecorder(), request)

	record := logtest.ReadJSONRecord(t, path)
	if record["logger"] != "websocketproxy" || record["msg"] != "Cannot proxy WebSocket connection" {
		t.Fatalf("unexpected file record: %#v", record)
	}
}

func TestServeHTTPLogsThroughRequestContext(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	request := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	request = request.WithContext(klog.NewContext(request.Context(), zapr.NewLogger(zap.New(core))))
	recorder := httptest.NewRecorder()

	(&WebsocketProxy{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("expected one central log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Level != zapcore.ErrorLevel {
		t.Fatalf("expected error level, got %s", entry.Level)
	}
	if entry.LoggerName != "websocketproxy" {
		t.Fatalf("expected websocketproxy logger, got %q", entry.LoggerName)
	}
	if entry.Message != "Cannot proxy WebSocket connection" {
		t.Fatalf("expected forwarded message, got %q", entry.Message)
	}
}
