package grpclog

import (
	"context"
	"path"
	"time"

	"github.com/anoideaopen/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a new unary server interceptors
// that adds Logger to the context.
func UnaryServerInterceptor(l glog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		var (
			logger = l.With()
			logCtx = glog.NewContext(ctx, logger)
			start  = time.Now()
		)

		addStdFields(ctx, logger, info.FullMethod, start)

		resp, err = handler(
			logCtx,
			req,
		)

		logger.Set(
			glog.Field{K: "grpc.code", V: status.Code(err).String()},
			glog.Field{
				K: "grpc.time_ms",
				V: float32(time.Since(start).Nanoseconds()/1e3) / 1e3,
			},
		)

		if err != nil {
			logger.Set(
				glog.Field{K: "error", V: err},
			)
		}

		levelLogf(
			logger,
			status.Code(err),
			"finished unary call with code "+status.Code(err).String())

		return resp, err
	}
}

// StreamServerInterceptor returns a new streaming server interceptor
// that adds Logger to the context.
func StreamServerInterceptor(l glog.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		var (
			logger = l.With()
			logCtx = glog.NewContext(stream.Context(), logger)
			start  = time.Now()
		)

		addStdFields(stream.Context(), logger, info.FullMethod, start)

		err = handler(
			srv,
			&streamContextWrapper{stream, logCtx},
		)

		logger.Set(
			glog.Field{K: "grpc.code", V: status.Code(err).String()},
			glog.Field{
				K: "grpc.time_ms",
				V: float32(time.Since(start).Nanoseconds()/1e3) / 1e3,
			},
		)

		if err != nil {
			logger.Set(
				glog.Field{K: "error", V: err},
			)
		}

		levelLogf(
			logger,
			status.Code(err),
			"finished streaming call with code "+status.Code(err).String())

		return err
	}
}

type streamContextWrapper struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *streamContextWrapper) Context() context.Context {
	return s.ctx
}

func addStdFields(
	ctx context.Context,
	logger glog.Logger,
	fullMethodString string,
	start time.Time,
) {
	service := path.Dir(fullMethodString)[1:]
	method := path.Base(fullMethodString)

	logger.Set(
		glog.Field{K: "system", V: "grpc"},
		glog.Field{K: "span.kind", V: "server"},
		glog.Field{K: "grpc.service", V: service},
		glog.Field{K: "grpc.method", V: method},
		glog.Field{K: "grpc.start_time", V: start.Format(time.RFC3339Nano)},
	)

	if d, ok := ctx.Deadline(); ok {
		logger.Set(
			glog.Field{K: "grpc.request.deadline", V: d.Format(time.RFC3339)},
		)
	}
}

func levelLogf(
	logger glog.Logger,
	code codes.Code,
	format string,
	args ...interface{},
) {
	switch code {
	case codes.OK, codes.Canceled, codes.InvalidArgument, codes.NotFound,
		codes.AlreadyExists, codes.Unauthenticated:
		logger.Infof(format, args...)
	case codes.DeadlineExceeded, codes.PermissionDenied,
		codes.ResourceExhausted, codes.FailedPrecondition,
		codes.Aborted, codes.OutOfRange, codes.Unavailable:
		logger.Warningf(format, args...)
	case codes.Unknown, codes.Unimplemented, codes.Internal, codes.DataLoss:
		logger.Errorf(format, args...)
	default:
		logger.Errorf(format, args...)
	}
}
