package logger

import (
	"context"
	"os"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	sugarLog *zap.SugaredLogger
)

// Init initializes Zap logger with colored console output
func Init() error {
	var level zapcore.Level
	switch config.LogLevel() {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// Configure encoder for colored console output
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	var encoder zapcore.Encoder
	if config.LogEncoding() == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)

	sugarLog = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)).Sugar()
	return nil
}

// Sync flushes any buffered log entries
func Sync() error {
	if sugarLog != nil {
		return sugarLog.Sync()
	}
	return nil
}

func Infof(ctx context.Context, format string, args ...interface{}) {
	sugarLog.Infof(format, args...)
}
func Errorf(ctx context.Context, format string, args ...interface{}) {
	sugarLog.Errorf(format, args...)
}
func Fatalf(ctx context.Context, format string, args ...interface{}) {
	sugarLog.Fatalf(format, args...)
}
func Debugf(ctx context.Context, format string, args ...interface{}) {
	sugarLog.Debugf(format, args...)
}
func Warnf(ctx context.Context, format string, args ...interface{}) {
	sugarLog.Warnf(format, args...)
}
func Panicf(ctx context.Context, format string, args ...interface{}) {
	sugarLog.Panicf(format, args...)
}
func Infoln(ctx context.Context, args ...interface{}) {
	sugarLog.Infoln(args...)
}
func Errorln(ctx context.Context, args ...interface{}) {
	sugarLog.Errorln(args...)
}
func Fatalln(ctx context.Context, args ...interface{}) {
	sugarLog.Fatalln(args...)
}
func Debugln(ctx context.Context, args ...interface{}) {
	sugarLog.Debugln(args...)
}
func Warnln(ctx context.Context, args ...interface{}) {
	sugarLog.Warnln(args...)
}
func Panicln(ctx context.Context, args ...interface{}) {
	sugarLog.Panicln(args...)
}
func withFields(rest ...interface{}) *zap.SugaredLogger {
	if len(rest) == 0 {
		return sugarLog
	}
	if len(rest) == 1 {
		if fields, ok := rest[0].([]zap.Field); ok {
			return sugarLog.Desugar().With(fields...).Sugar()
		}
		if _, ok := rest[0].(zap.Field); ok {
			return sugarLog.Desugar().With(rest[0].(zap.Field)).Sugar()
		}
	}
	// 检查是否全是 zap.Field（如 logger.Error(ctx, "msg", zap.Error(err))）
	var zapFields []zap.Field
	for _, r := range rest {
		f, ok := r.(zap.Field)
		if !ok {
			return sugarLog.With(rest...)
		}
		zapFields = append(zapFields, f)
	}
	return sugarLog.Desugar().With(zapFields...).Sugar()
}

func Info(ctx context.Context, args ...interface{}) {
	if len(args) == 0 {
		sugarLog.Info()
		return
	}
	msg, ok := args[0].(string)
	if !ok || len(args) == 1 {
		sugarLog.Info(args...)
		return
	}
	withFields(args[1:]...).Info(msg)
}
func Error(ctx context.Context, args ...interface{}) {
	if len(args) == 0 {
		sugarLog.Error()
		return
	}
	msg, ok := args[0].(string)
	if !ok || len(args) == 1 {
		sugarLog.Error(args...)
		return
	}
	withFields(args[1:]...).Error(msg)
}
func Fatal(ctx context.Context, args ...interface{}) {
	if len(args) == 0 {
		sugarLog.Fatal()
		return
	}
	msg, ok := args[0].(string)
	if !ok || len(args) == 1 {
		sugarLog.Fatal(args...)
		return
	}
	withFields(args[1:]...).Fatal(msg)
}
func Debug(ctx context.Context, args ...interface{}) {
	if len(args) == 0 {
		sugarLog.Debug()
		return
	}
	msg, ok := args[0].(string)
	if !ok || len(args) == 1 {
		sugarLog.Debug(args...)
		return
	}
	withFields(args[1:]...).Debug(msg)
}
func Warn(ctx context.Context, args ...interface{}) {
	if len(args) == 0 {
		sugarLog.Warn()
		return
	}
	msg, ok := args[0].(string)
	if !ok || len(args) == 1 {
		sugarLog.Warn(args...)
		return
	}
	withFields(args[1:]...).Warn(msg)
}
func Panic(ctx context.Context, args ...interface{}) {
	if len(args) == 0 {
		sugarLog.Panic()
		return
	}
	msg, ok := args[0].(string)
	if !ok || len(args) == 1 {
		sugarLog.Panic(args...)
		return
	}
	withFields(args[1:]...).Panic(msg)
}
