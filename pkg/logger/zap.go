package logger

import (
	"syscall"
	"time"

	"github.com/newity/glog"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	LogTypeTTY  = "console"
	LogTypeJSON = "json"
)

type sugaredLogger struct {
	sugar *zap.SugaredLogger
}

func (sl *sugaredLogger) Set(...glog.Field) {
}

func (sl *sugaredLogger) With(fields ...glog.Field) glog.Logger {
	args := make([]interface{}, 0, len(fields)*2) //nolint:gomnd
	for _, field := range fields {
		args = append(args, field.K)
		args = append(args, field.V)
	}
	return &sugaredLogger{sugar: sl.sugar.With(args...)}
}

func (sl *sugaredLogger) Trace(...interface{}) {
}

func (sl *sugaredLogger) Tracef(string, ...interface{}) {
}

func (sl *sugaredLogger) Warning(args ...interface{}) {
	sl.sugar.Warn(args...)
}

func (sl *sugaredLogger) Warningf(format string, args ...interface{}) {
	sl.sugar.Warnf(format, args...)
}

func (sl *sugaredLogger) Debug(args ...interface{}) {
	sl.sugar.Debug(args...)
}

func (sl *sugaredLogger) Debugf(format string, args ...interface{}) {
	sl.sugar.Debugf(format, args...)
}

func (sl *sugaredLogger) Error(args ...interface{}) {
	sl.sugar.Error(args...)
}

func (sl *sugaredLogger) Errorf(format string, args ...interface{}) {
	sl.sugar.Errorf(format, args...)
}

func (sl *sugaredLogger) Info(args ...interface{}) {
	sl.sugar.Info(args...)
}

func (sl *sugaredLogger) Infof(format string, args ...interface{}) {
	sl.sugar.Infof(format, args...)
}

func (sl *sugaredLogger) Panic(args ...interface{}) {
	sl.sugar.Panic(args...)
}

func (sl *sugaredLogger) Panicf(format string, args ...interface{}) {
	sl.sugar.Panicf(format, args...)
}

func newSugarLogger(outputEncoding string, level string) (*sugaredLogger, error) {
	cfg := zap.Config{
		Encoding:         outputEncoding,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "message",
			LevelKey:     "level",
			EncodeLevel:  zapcore.CapitalLevelEncoder,
			TimeKey:      "time",
			EncodeTime:   zapcore.TimeEncoderOfLayout(time.RFC3339),
			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}

	err := cfg.Level.UnmarshalText([]byte(level))
	if err != nil {
		return nil, err
	}

	zLogger, err := cfg.Build(zap.AddCaller(), zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}

	return &sugaredLogger{sugar: zLogger.Sugar()}, nil
}

type SyncLoggerMethod = func() error

func CreateLogger(loggerType, logLevel string) (glog.Logger, SyncLoggerMethod, error) {
	if err := verifyLoggerType(loggerType); err != nil {
		return nil, nil, errors.WithStack(errors.Wrap(err, "failed to create logger"))
	}
	sugar, err := newSugarLogger(loggerType, logLevel)
	if err != nil {
		return nil, nil, errors.WithStack(errors.Wrap(err, "failed to create logger"))
	}
	return sugar, sugar.Flush, nil
}

func verifyLoggerType(loggerEncoding string) error {
	switch loggerEncoding {
	case LogTypeTTY, LogTypeJSON:
		return nil
	}
	return errors.Errorf("unknown logger's encoding %s", loggerEncoding)
}

func (sl *sugaredLogger) Flush() error {
	if err := sl.sugar.Sync(); err != nil && (!errors.Is(err, syscall.EBADF) && !errors.Is(err, syscall.ENOTTY) && !errors.Is(err, syscall.EINVAL)) {
		return err
	}
	return nil
}
