package logger

import (
	"maps"
	"sort"
	"syscall"
	"time"

	"github.com/anoideaopen/glog"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	LogTypeTTY  = "console"
	LogTypeJSON = "json"
)

type sugaredLogger struct {
	sugar  *zap.SugaredLogger
	fields map[string]interface{}
	cfg    zap.Config
}

func (sl *sugaredLogger) Set(fields ...glog.Field) {
	for _, field := range fields {
		sl.fields[field.K] = field.V
	}

	args := mapToArgs(sl.fields)
	zLogger, _ := sl.cfg.Build(zap.AddCaller(), zap.AddCallerSkip(1))
	sl.sugar = zLogger.Sugar().With(args...)
}

func (sl *sugaredLogger) With(fields ...glog.Field) glog.Logger {
	fl := maps.Clone(sl.fields)
	for _, field := range fields {
		fl[field.K] = field.V
	}

	args := mapToArgs(fl)
	zLogger, _ := sl.cfg.Build(zap.AddCaller(), zap.AddCallerSkip(1))

	return &sugaredLogger{
		sugar:  zLogger.Sugar().With(args...),
		fields: fl,
		cfg:    sl.cfg,
	}
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

	return &sugaredLogger{
		sugar:  zLogger.Sugar(),
		fields: make(map[string]interface{}),
		cfg:    cfg,
	}, nil
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

func mapToArgs(fields map[string]interface{}) []interface{} {
	sortKeys := make([]string, 0, len(fields))
	for k := range fields {
		sortKeys = append(sortKeys, k)
	}
	sort.Strings(sortKeys)

	args := make([]interface{}, 0, len(sortKeys)*2) //nolint:gomnd
	for _, k := range sortKeys {
		args = append(args, k)
		args = append(args, fields[k])
	}

	return args
}
