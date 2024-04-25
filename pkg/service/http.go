package service

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/helpers/nerrors"
	"github.com/anoideaopen/channel-transfer/pkg/logger"
	"github.com/anoideaopen/common-component/errorshlp"
	"github.com/anoideaopen/glog"
)

type HTTPServer struct {
	log    glog.Logger
	server *http.Server
}

type HTTPHandler struct {
	Path    string
	Handler http.HandlerFunc
}

func New(ctx context.Context, address string, tlsConfig *tls.Config, handlers ...HTTPHandler) *HTTPServer {
	log := glog.FromContext(ctx).With(logger.Labels{Component: logger.ComponentService}.Fields()...)

	hs := &HTTPServer{log: log}

	router := http.NewServeMux()
	for _, handler := range handlers {
		router.Handle(handler.Path, handler.Handler)
	}

	hs.server = &http.Server{
		ReadHeaderTimeout: time.Second,
		Addr:              address,
		Handler:           router,
		TLSConfig:         tlsConfig,
	}

	return hs
}

func (hs *HTTPServer) Run(ctx context.Context) error {
	hs.log.Infof("http service listen on %s", hs.server.Addr)

	go func() {
		<-ctx.Done()
		_ = hs.server.Close()
	}()

	var err error
	if hs.server.TLSConfig != nil {
		err = hs.server.ListenAndServeTLS("", "")
	} else {
		err = hs.server.ListenAndServe()
	}

	if !errors.Is(err, http.ErrServerClosed) {
		return errorshlp.WrapWithDetails(err, nerrors.ErrTypeHTTP, nerrors.ComponenHTTP)
	}
	return nil
}
