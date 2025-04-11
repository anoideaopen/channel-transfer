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
	"github.com/anoideaopen/channel-transfer/pkg/transfer"
	"github.com/anoideaopen/common-component/basemetrics/baseprometheus"
	"github.com/anoideaopen/glog"
	"github.com/go-errors/errors"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	prometheus2 "github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
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

	var (
		grpcMetrics     *grpcprom.ServerMetrics
		serviceHandlers []service.HTTPHandler
	)

	ctx, grpcMetrics, serviceHandlers, err = initMetrics(
		ctx,
		cfg,
		[]service.HTTPHandler{
			{Path: "/", Handler: hc.LivenessProbe},
			{Path: "/ready", Handler: hc.ReadinessProbe},
		},
		log,
		version,
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
		redis.NewUniversalClient(
			&redis.UniversalOptions{
				Addrs:     cfg.RedisStorage.Addr,
				Password:  cfg.RedisStorage.Password,
				ReadOnly:  false,
				TLSConfig: cfg.RedisStorage.TLSConfig(),
			},
		),
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

	connectionProfile, err := hlf.NewConnectionProfile(cfg.ProfilePath, cfg.ProfileRaw)
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

	for _, channel := range cfg.Channels {
		chName := channel.Name
		eGroup.Go(func() error {
			producer, err := chproducer.NewHandler(
				ctx,
				chName,
				*cfg.Options.TTL,
				cfg.Options.TransfersInHandleOnChannel,
				storage,
				pool,
				dm.Stream(chName),
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

func initMetrics(
	ctx context.Context,
	cfg *config.Config,
	handlers []service.HTTPHandler,
	log glog.Logger,
	version string,
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
