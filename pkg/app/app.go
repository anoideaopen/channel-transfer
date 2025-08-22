package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/chproducer"
	"github.com/anoideaopen/channel-transfer/pkg/config"
	redis2 "github.com/anoideaopen/channel-transfer/pkg/data/redis"
	"github.com/anoideaopen/channel-transfer/pkg/demultiplexer"
	"github.com/anoideaopen/channel-transfer/pkg/hlf"
	"github.com/anoideaopen/channel-transfer/pkg/logger"
	"github.com/anoideaopen/channel-transfer/pkg/metrics"
	"github.com/anoideaopen/channel-transfer/pkg/metrics/prometheus"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/channel-transfer/pkg/service"
	"github.com/anoideaopen/channel-transfer/pkg/service/healthcheck"
	"github.com/anoideaopen/channel-transfer/pkg/telemetry"
	"github.com/anoideaopen/channel-transfer/pkg/transfer"
	"github.com/anoideaopen/common-component/basemetrics/baseprometheus"
	"github.com/anoideaopen/glog"
	"github.com/go-errors/errors"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	prometheus2 "github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	prometheus3 "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"golang.org/x/sync/errgroup"
)

const activeTransferCount = 1000

func Run(ctx context.Context, cfg *config.Config, version string) error {
	startTime := time.Now()

	ctx, log, flush, err := createLoggerWithContext(ctx, cfg, version)
	if err != nil {
		return errors.Errorf("%+v", err)
	}
	defer func() {
		if err = flush(); err != nil {
			panic(fmt.Sprintf("flush logger buffer: %+v", err))
		}
	}()

	cfgForLog, _ := json.MarshalIndent(cfg.WithoutSensitiveData(), "", "\t")
	log.Infof("version: %s", version)
	log.Infof("config: \n%s\n", cfgForLog)

	hc := healthcheck.New()

	rdb := redis.NewUniversalClient(
		&redis.UniversalOptions{
			Addrs:     cfg.RedisStorage.Addr,
			Password:  cfg.RedisStorage.Password,
			ReadOnly:  false,
			TLSConfig: cfg.RedisStorage.TLSConfig(),
		},
	)

	ctxWithLogger := glog.NewContext(ctx, log)
	if err = initTracer(ctxWithLogger, cfg, rdb); err != nil {
		log.Error(fmt.Errorf("failed to initialize tracer: %w", err))
	}
	ctx, grpcMetrics, serviceHandlers, err := initMetrics(
		ctx,
		cfg,
		[]service.HTTPHandler{
			{Path: "/", Handler: hc.LivenessProbe},
			{Path: "/ready", Handler: hc.ReadinessProbe},
		},
		log,
		version,
		rdb,
	)
	if err != nil {
		return errors.Errorf("metrics: %+v", err)
	}

	httpSrv := service.New(
		ctx,
		cfg.Service.Address,
		cfg.Service.TLSConfig(),
		serviceHandlers...,
	)

	storage, err := redis2.NewStorage(
		ctx,
		rdb,
		*cfg.Options.TTL+*cfg.RedisStorage.AfterTransferTTL,
		cfg.RedisStorage.DBPrefix,
	)
	if err != nil {
		return errors.Errorf("redis: %+v", err)
	}

	requests := make(chan model.TransferRequest, cfg.Options.NewestRequestStreamBufferSize)
	defer close(requests)

	dm, err := demultiplexer.NewDemultiplexer(ctx, requests, activeTransferCount)
	if err != nil {
		return errors.Errorf("demultiplexer: %+v", err)
	}

	connectionProfile, err := hlf.NewConnectionProfile(cfg.ProfilePath)
	if err != nil {
		return errors.Errorf("hlf config: %+v", err)
	}

	pool, err := hlf.NewPool(ctx, cfg.Channels, cfg.UserName, cfg.Options, connectionProfile, storage)
	if err != nil {
		return errors.Errorf("pool: %+v", err)
	}

	eGroup, eGroupCtx := errgroup.WithContext(ctx)

	eGroup.Go(func() error {
		return httpSrv.Run(eGroupCtx)
	})

	err = transfer.Execute(eGroupCtx, eGroup, cfg.ListenAPI, cfg.Channels, requests, storage, grpcMetrics)
	if err != nil {
		return errors.Errorf("%+v", err)
	}

	eGroup.Go(func() error {
		return dm.Run(eGroupCtx)
	})

	eGroup.Go(func() error {
		return pool.RunCollectors(eGroupCtx)
	})

	for _, channel := range pool.Channels() {
		eGroup.Go(func() error {
			producer, err := chproducer.NewHandler(
				ctx,
				channel,
				*cfg.Options.TTL,
				cfg.Options.TransfersInHandleOnChannel,
				storage,
				pool,
				dm.Stream(channel),
			)
			if err != nil {
				return err
			}
			return producer.Exec(eGroupCtx)
		})
	}

	hc.Ready()

	m := metrics.FromContext(ctx)
	dur := time.Since(startTime)
	m.AppInitDuration().Set(dur.Seconds())

	log.Infof("Channel transfer started, time - %s\n", dur.String())

	if err = eGroup.Wait(); err != nil {
		log.Error(errors.New(err))
		return err
	}

	return nil
}

func createLoggerWithContext(ctx context.Context, cfg *config.Config, version string) (context.Context, glog.Logger, logger.SyncLoggerMethod, error) {
	lggr, method, err := logger.CreateLogger(cfg.LogType, cfg.LogLevel)
	if err != nil {
		return nil, nil, nil, err
	}

	ctx = glog.NewContext(ctx, lggr)
	log := lggr.With(logger.Labels{Version: version, Component: logger.ComponentMain}.Fields()...)

	return ctx, log, method, nil
}

func getFabricSdkVersion(log glog.Logger) string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		log.Warning("Failed to read build info")
		return ""
	}

	var m *debug.Module
	for _, dep := range bi.Deps {
		if strings.HasSuffix(dep.Path, "fabric-sdk-go") {
			if dep.Replace != nil {
				m = dep.Replace
			} else {
				m = dep
			}
		}
	}

	if m != nil {
		return fmt.Sprintf("%s %s", m.Path, m.Version)
	}

	return ""
}

// Initialize tracer
func initTracer(ctx context.Context, cfg *config.Config, rdb redis.UniversalClient) error {
	log := glog.FromContext(ctx)
	if cfg.Tracing == nil ||
		cfg.Tracing.Collector == nil ||
		cfg.Tracing.Endpoint == "" {
		log.Debug("server: tracing disabled")
		return nil
	}
	log.Debug("server: tracing enabled")
	if err := telemetry.InitTracing(
		ctx,
		cfg.Tracing.Collector,
		strings.ToLower(config.EnvPrefix),
	); err != nil {
		return fmt.Errorf("failed to initialize tracer: %w", err)
	}
	log.Debug("server: tracing enabled")
	if cfg.Tracing.EnabledTracingRedis {
		if err := redisotel.InstrumentTracing(rdb); err != nil {
			log.Warning("redisotel tracing enabling failed", errors.New(err))
		}
	} else {
		log.Debug("server: redis tracing disabled")
	}

	return nil
}

func initMetrics(
	ctx context.Context,
	cfg *config.Config,
	handlers []service.HTTPHandler,
	log glog.Logger,
	version string,
	rdb redis.UniversalClient,
) (context.Context, *grpcprom.ServerMetrics, []service.HTTPHandler, error) {
	if cfg.PromMetrics == nil {
		return ctx, nil, nil, nil
	}

	m, err := prometheus.NewMetrics(ctx, cfg.PromMetrics.PrefixForMetrics)
	if err != nil {
		return ctx, nil, nil, err
	}
	m.AppInfo().Inc(
		metrics.Labels().AppVer.Create(version),
		metrics.Labels().AppSdkFabricVer.Create(getFabricSdkVersion(log)),
	)

	ctx = metrics.NewContext(ctx, m)

	for _, channel := range cfg.Channels {
		for i := model.InProgressTransferFrom; i <= model.ExistsChannelTo; i++ {
			m.TotalInWorkTransfer().Set(
				0,
				metrics.Labels().Channel.Create(channel.Name),
				metrics.Labels().TransferStatus.Create(i.String()),
			)
		}
		m.FabricConnectionStatus().Set(0,
			metrics.Labels().Channel.Create(channel.Name),
		)
		m.CollectorProcessBlockNum().Set(0,
			metrics.Labels().Channel.Create(channel.Name),
		)
	}

	if cfg.PromMetrics.EnableMetricsRedis {
		exporter, err := prometheus3.New(prometheus3.WithRegisterer(prometheus2.DefaultRegisterer))
		if err != nil {
			log.Warning("prometheus exporter creation failed", errors.New(err))
		}
		provider := metric.NewMeterProvider(metric.WithReader(exporter))
		if err := redisotel.InstrumentMetrics(rdb, redisotel.WithMeterProvider(provider)); err != nil {
			log.Warning("redisotel metrics enabling failed", errors.New(err))
		}
		log.Debug("server: redis metrics enabled")
	}

	grpcMetrics := grpcprom.NewServerMetrics(grpcprom.WithServerHandlingTimeHistogram())
	prometheus2.MustRegister(grpcMetrics)

	handlers = append(
		handlers,
		service.HTTPHandler{
			Path: "/metrics",
			Handler: func(writer http.ResponseWriter, request *http.Request) {
				baseprometheus.MetricsHandler(ctx).ServeHTTP(writer, request)
			},
		},
	)

	return ctx, grpcMetrics, handlers, nil
}
