package logger

import (
	"maps"
	"sort"
	"syscall"
	"time"

	"github.com/anoideaopen/glog"
	"github.com/go-errors/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc/grpclog"
)

const (
	LogTypeTTY  = "console"
	LogTypeJSON = "json"
	LogTypeGCP  = "gcp"
)

type sugaredLogger struct {
	sugar  *zap.SugaredLogger
	fields map[string]any
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

func (sl *sugaredLogger) Trace(...any) {
}

func (sl *sugaredLogger) Tracef(string, ...any) {
}

func (sl *sugaredLogger) Warning(args ...any) {
	sl.sugar.Warn(args...)
}

func (sl *sugaredLogger) Warningf(format string, args ...any) {
	sl.sugar.Warnf(format, args...)
}

func (sl *sugaredLogger) Debug(args ...any) {
	sl.sugar.Debug(args...)
}

func (sl *sugaredLogger) Debugf(format string, args ...any) {
	sl.sugar.Debugf(format, args...)
}

func (sl *sugaredLogger) Error(args ...any) {
	sl.sugar.Error(args...)
}

func (sl *sugaredLogger) Errorf(format string, args ...any) {
	sl.sugar.Errorf(format, args...)
}

func (sl *sugaredLogger) Info(args ...any) {
	sl.sugar.Info(args...)
}

func (sl *sugaredLogger) Infof(format string, args ...any) {
	sl.sugar.Infof(format, args...)
}

func (sl *sugaredLogger) Panic(args ...any) {
	sl.sugar.Panic(args...)
}

func (sl *sugaredLogger) Panicf(format string, args ...any) {
	sl.sugar.Panicf(format, args...)
}

func logEncoding(loggerType string) string {
	switch loggerType {
	case LogTypeGCP:
		return LogTypeJSON
	default:
		return loggerType
	}
}

func logLevelKey(loggerType string) string {
	switch loggerType {
	case LogTypeGCP:
		return "severity"
	default:
		return "level"
	}
}

func newSugarLogger(loggerType string, level string) (*sugaredLogger, error) {
	cfg := zap.Config{
		Encoding:         logEncoding(loggerType),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "message",
			LevelKey:     logLevelKey(loggerType),
			EncodeLevel:  zapcore.CapitalLevelEncoder,
			TimeKey:      "time",
			EncodeTime:   zapcore.TimeEncoderOfLayout(time.RFC3339Nano),
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

	grpclog.SetLoggerV2(zapgrpc.NewLogger(zLogger))

	return &sugaredLogger{
		sugar:  zLogger.Sugar(),
		fields: make(map[string]any),
		cfg:    cfg,
	}, nil
}

type SyncLoggerMethod = func() error

func CreateLogger(loggerType, logLevel string) (glog.Logger, SyncLoggerMethod, error) {
	if err := verifyLoggerType(loggerType); err != nil {
		return nil, nil, errors.Errorf("failed to create logger: %w", err)
	}
	sugar, err := newSugarLogger(loggerType, logLevel)
	if err != nil {
		return nil, nil, errors.Errorf("failed to create logger: %w", err)
	}
	return sugar, sugar.Flush, nil
}

func verifyLoggerType(loggerEncoding string) error {
	switch loggerEncoding {
	case LogTypeTTY, LogTypeJSON, LogTypeGCP:
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

func mapToArgs(fields map[string]any) []any {
	sortKeys := make([]string, 0, len(fields))
	for k := range fields {
		sortKeys = append(sortKeys, k)
	}
	sort.Strings(sortKeys)

	args := make([]any, 0, len(sortKeys)*2)
	for _, k := range sortKeys {
		args = append(args, k)
		args = append(args, fields[k])
	}

	return args
}
