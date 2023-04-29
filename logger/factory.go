package logger

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

//
// Builder factory.
type Builder interface {
	New() logr.LogSink
	V(int, logr.LogSink) logr.LogSink
}

//
// ZapBuilder factory.
type ZapBuilder struct {
}

//
// New returns a new logger.
func (b *ZapBuilder) New() (sink logr.LogSink) {
	var encoder zapcore.Encoder
	sinker := zapcore.AddSync(os.Stderr)
	level := zap.NewAtomicLevelAt(zap.DebugLevel)
	options := []zap.Option{
		zap.AddStacktrace(zap.ErrorLevel),
		zap.ErrorOutput(sinker),
		zap.AddCallerSkip(1),
	}
	if Settings.Development {
		cfg := zap.NewDevelopmentEncoderConfig()
		encoder = zapcore.NewConsoleEncoder(cfg)
		options = append(options, zap.Development())
	} else {
		cfg := zap.NewProductionEncoderConfig()
		encoder = zapcore.NewJSONEncoder(cfg)
	}
	logger := zapr.NewLogger(
		zap.New(
			zapcore.NewCore(
				encoder,
				sinker,
				level)).WithOptions(options...))
	sink = logger.GetSink()
	return
}

//
// V returns a logger with level.
func (b *ZapBuilder) V(level int, in logr.Logger) (out logr.LogSink) {
	if Settings.atDebug(level) {
		out = in.V(1).GetSink()
	} else {
		out = in.V(0).GetSink()
	}
	return
}
