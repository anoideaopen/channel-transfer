package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/newity/glog"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/status"
)

// NewRESTErrorHandler replaces the default error handler of the package
// [github.com/grpc-ecosystem/grpc-gateway/v2/runtime] with a custom one that
// logs errors and modifies the header behavior.
func NewRESTErrorHandler(logger glog.Logger) runtime.ServeMuxOption {
	return runtime.WithErrorHandler(
		func(
			ctx context.Context,
			mux *runtime.ServeMux,
			marshaler runtime.Marshaler,
			writer http.ResponseWriter,
			request *http.Request,
			err error,
		) {
			if logger == nil {
				logger = empty
			}

			writer.Header().Del("Trailer")
			writer.Header().Del("Transfer-Encoding")
			writer.Header().Set("Content-Type", "application/json")

			reqErr := &spb.Status{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			}

			if s, ok := status.FromError(err); ok {
				reqErr.Code = int32(runtime.HTTPStatusFromCode(s.Code()))
				reqErr.Message = s.String()
			}

			writer.WriteHeader(int(reqErr.Code))

			buf, err := json.Marshal(&reqErr)
			if err != nil {
				logger.Errorf("response of api error marshal : %+v", err)
				// form the json object spb.Status manually
				buf = []byte(fmt.Sprintf(
					`{"code":%d,"message":"%s"}`,
					http.StatusInternalServerError,
					err.Error(),
				))
			}

			if _, err = writer.Write(buf); err != nil {
				logger.Errorf("response of api request writing : %+v", err)
			}

			logger.Errorf("response of api request : %s", string(buf))
		},
	)
}

var empty glog.Logger = &emptyLogger{}

type emptyLogger struct{}

func (el *emptyLogger) Set(...glog.Field)               {}
func (el *emptyLogger) With(...glog.Field) glog.Logger  { return el }
func (el *emptyLogger) Trace(...interface{})            {}
func (el *emptyLogger) Tracef(string, ...interface{})   {}
func (el *emptyLogger) Debug(...interface{})            {}
func (el *emptyLogger) Debugf(string, ...interface{})   {}
func (el *emptyLogger) Info(...interface{})             {}
func (el *emptyLogger) Infof(string, ...interface{})    {}
func (el *emptyLogger) Warning(...interface{})          {}
func (el *emptyLogger) Warningf(string, ...interface{}) {}
func (el *emptyLogger) Error(...interface{})            {}
func (el *emptyLogger) Errorf(string, ...interface{})   {}
func (el *emptyLogger) Panic(...interface{})            {}
func (el *emptyLogger) Panicf(string, ...interface{})   {}
