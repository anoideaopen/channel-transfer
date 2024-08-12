package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/chproducer"
	"github.com/anoideaopen/channel-transfer/pkg/config"
	redis2 "github.com/anoideaopen/channel-transfer/pkg/data/redis"
	"github.com/anoideaopen/channel-transfer/pkg/demultiplexer"
	"github.com/anoideaopen/channel-transfer/pkg/hlf"
	"github.com/anoideaopen/channel-transfer/pkg/hlf/hlfprofile"
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

var AppInfoVer = "undefined-ver"

//nolint:funlen
func main() {
	startTime := time.Now()

	cfg, err := config.GetConfig()
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	ctx, cancel, log, flush, err := createLoggerWithContext(cfg)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	defer func() {
		if err := flush(); err != nil {
			panic(fmt.Sprintf("flush logger buffer: %+v", err))
		}
	}()

	cfgForLog, _ := json.MarshalIndent(cfg.WithoutSensitiveData(), "", "\t")
	log.Infof("version: %s", AppInfoVer)
	log.Infof("config: \n%s\n", cfgForLog)

	go func() {
		interruptCh := make(chan os.Signal, 1)
		defer close(interruptCh)

		signal.Notify(interruptCh, os.Interrupt, syscall.SIGTERM)
		signal.Stop(interruptCh)

		s := <-interruptCh

		cancel()
		log.Infof("received signal %s", s.String())
	}()

	hlfProfile, err := hlfprofile.ParseProfile(cfg.ProfilePath)
	if err != nil {
		panic(fmt.Sprintf("hlf config: %+v", err))
	}

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
	)
	if err != nil {
		panic(fmt.Sprintf("metrics: %+v", err))
	}

	httpSrv := service.New(
		ctx,
		cfg.Service.Address,
		cfg.Service.TLSConfig(),
		serviceHandlers...,
	)

	storage, err := redis2.NewStorage(
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
		panic(fmt.Sprintf("redis: %+v", err))
	}

	requests := make(chan model.TransferRequest, cfg.Options.NewestRequestStreamBufferSize)
	defer close(requests)

	dm, err := demultiplexer.NewDemultiplexer(ctx, requests, activeTransferCount)
	if err != nil {
		panic(fmt.Sprintf("demultiplexer: %+v", err))
	}

	pool, err := hlf.NewPool(ctx, cfg.Channels, cfg.UserName, cfg.Options, cfg.ProfilePath, *hlfProfile, storage)
	if err != nil {
		panic(fmt.Sprintf("pool: %+v", err))
	}

	eGroup, eGroupCtx := errgroup.WithContext(ctx)

	eGroup.Go(func() error {
		return httpSrv.Run(eGroupCtx)
	})

	err = transfer.Execute(eGroupCtx, eGroup, cfg.ListenAPI, cfg.Channels, requests, storage, grpcMetrics)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	eGroup.Go(func() error {
		return dm.Run(eGroupCtx)
	})

	eGroup.Go(func() error {
		return pool.RunCollectors(eGroupCtx)
	})

	for _, channel := range cfg.Channels {
		chName := channel
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
	}
}

func createLoggerWithContext(cfg *config.Config) (context.Context, context.CancelFunc, glog.Logger, logger.SyncLoggerMethod, error) {
	ctx, cancel := context.WithCancel(context.Background())

	lggr, method, err := logger.CreateLogger(cfg.LogType, cfg.LogLevel)
	if err != nil {
		return ctx, cancel, nil, nil, err
	}

	ctx = glog.NewContext(ctx, lggr)
	log := lggr.With(logger.Labels{Version: AppInfoVer, Component: logger.ComponentMain}.Fields()...)

	return ctx, cancel, log, method, nil
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
) (context.Context, *grpcprom.ServerMetrics, []service.HTTPHandler, error) {
	if cfg.PromMetrics == nil {
		return ctx, nil, nil, nil
	}

	m, err := prometheus.NewMetrics(ctx, cfg.PromMetrics.PrefixForMetrics)
	if err != nil {
		return ctx, nil, nil, err
	}
	m.AppInfo().Inc(
		metrics.Labels().AppVer.Create(AppInfoVer),
		metrics.Labels().AppSdkFabricVer.Create(getFabricSdkVersion(log)),
	)

	ctx = metrics.NewContext(ctx, m)

	for _, channel := range cfg.Channels {
		for i := model.InProgressTransferFrom; i <= model.ExistsChannelTo; i++ {
			m.TotalInWorkTransfer().Set(
				0,
				metrics.Labels().Channel.Create(channel),
				metrics.Labels().TransferStatus.Create(i.String()),
			)
		}
		m.FabricConnectionStatus().Set(0,
			metrics.Labels().Channel.Create(channel),
		)
		m.CollectorProcessBlockNum().Set(0,
			metrics.Labels().Channel.Create(channel),
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
