package transfer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/logger"
	"github.com/anoideaopen/channel-transfer/pkg/middleware"
	"github.com/anoideaopen/channel-transfer/proto"
	"github.com/felixge/httpsnoop"
	"github.com/flowchartsman/swaggerui"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/anoideaopen/glog"
	metrics "github.com/slok/go-http-metrics/metrics/prometheus"
	middleware2 "github.com/slok/go-http-metrics/middleware"
	mstd "github.com/slok/go-http-metrics/middleware/std"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RunHTTP serve http with optional TLS by hardcoded url path /v1
func runHTTP(ctx context.Context, tlsConfig *tls.Config, addressHTTP string, addressGRPC string, useMetrics bool) error {
	log := glog.FromContext(ctx).With(logger.Labels{Component: logger.ComponentAPI}.Fields()...)

	conn, err := grpc.DialContext(ctx, addressGRPC, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("grpc dial: %w", err)
	}

	mux := runtime.NewServeMux(middleware.NewRESTErrorHandler(log))
	if err = proto.RegisterAPIHandler(ctx, mux, conn); err != nil {
		return fmt.Errorf("register gateway: %w", err)
	}

	var handler http.Handler

	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1") {
			mux.ServeHTTP(w, r)
			return
		}
		swaggerui.Handler(proto.SwaggerJSON).ServeHTTP(w, r)
	})

	if useMetrics {
		mdlw := middleware2.New(
			middleware2.Config{
				Recorder: metrics.NewRecorder(metrics.Config{}),
			},
		)
		handler = mstd.Handler("api", mdlw, handler)
	}

	s := &http.Server{
		ReadHeaderTimeout: time.Second,
		Addr:              addressHTTP,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m := httpsnoop.CaptureMetrics(handler, w, r)
			log.Infof("request: ip %s, path %s, duration %v, code %d", r.RemoteAddr, r.RequestURI, m.Duration, m.Code)
		}),
		TLSConfig: tlsConfig,
	}

	log.Infof("http listen on %s", addressHTTP)

	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()

	if tlsConfig != nil {
		err = s.ListenAndServeTLS("", "")
	} else {
		err = s.ListenAndServe()
	}

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
