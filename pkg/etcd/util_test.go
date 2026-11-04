package etcd

import (
	"context"
	"flag"
	"io"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/grpclog"
	"gotest.tools/v3/assert"
	"k8s.io/klog/v2"
)

// TestGetEtcdClientLoggerVerbosity verifies that GetEtcdClient silences the
// clientv3 logger by default, so restore and startup don't print misleading
// "retrying of unary invoker failed" warnings, and only wires up the real
// logger when verbose logging (-v=1) is enabled.
//
// Regression test for ENGCP-428: before the fix every etcd client was created
// with the global logger, so the clientv3 retry interceptor logged warnings to
// the console whenever it connected before the backing store was reachable.
func TestGetEtcdClientLoggerVerbosity(t *testing.T) {
	// The lazily-dialed client logs a transport error via grpclog when it is
	// closed; discard it so it doesn't pollute the test output.
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(io.Discard, io.Discard, io.Discard))

	// Install a real global logger that is enabled at all levels but discards
	// its output, so that "the client would log" is observable without writing
	// noise to the test output.
	enabled := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(io.Discard),
		zapcore.DebugLevel,
	))
	defer zap.ReplaceGlobals(enabled)()

	t.Run("silenced by default", func(t *testing.T) {
		setKlogVerbosity(t, "0")

		cli, err := GetEtcdClient(context.Background(), nil, "127.0.0.1:2379")
		assert.NilError(t, err)
		t.Cleanup(func() { _ = cli.Close() })

		assert.Assert(t, !cli.GetLogger().Core().Enabled(zapcore.WarnLevel),
			"etcd client logger should be silenced when verbose logging is off")
	})

	t.Run("enabled with verbose logging", func(t *testing.T) {
		setKlogVerbosity(t, "1")

		cli, err := GetEtcdClient(context.Background(), nil, "127.0.0.1:2379")
		assert.NilError(t, err)
		t.Cleanup(func() { _ = cli.Close() })

		assert.Assert(t, cli.GetLogger().Core().Enabled(zapcore.WarnLevel),
			"etcd client logger should emit warnings when verbose logging is on")
	})
}

// setKlogVerbosity sets the global klog verbosity for the duration of the test
// and restores it afterwards.
func setKlogVerbosity(t *testing.T, level string) {
	t.Helper()
	var fs flag.FlagSet
	klog.InitFlags(&fs)
	assert.NilError(t, fs.Set("v", level))
	t.Cleanup(func() { _ = fs.Set("v", "0") })
}
