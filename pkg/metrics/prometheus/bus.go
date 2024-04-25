package prometheus

import (
	"context"
	"strings"

	"github.com/anoideaopen/channel-transfer/pkg/metrics"
	"github.com/anoideaopen/common-component/basemetrics"
	"github.com/anoideaopen/common-component/basemetrics/baseprometheus"
	"github.com/anoideaopen/glog"
)

type MetricsBus struct {
	log glog.Logger

	baseBus *baseprometheus.BaseMetricsBus[MetricsBus]

	mAppInfo                 *baseprometheus.Counter
	mTotalTransferCreated    *baseprometheus.Counter
	mTotalReconnectsToFabric *baseprometheus.Counter
	mTotalSuccessTransfer    *baseprometheus.Counter
	mTotalFailureTransfer    *baseprometheus.Counter

	mTotalInWorkTransfer      *baseprometheus.Gauge
	mAppInitDuration          *baseprometheus.Gauge
	mFabricConnectionStatus   *baseprometheus.Gauge
	mCollectorProcessBlockNum *baseprometheus.Gauge

	mTimeDurationCompleteTransferBeforeResponding *baseprometheus.Histo
	mTransferExecutionTimeDuration                *baseprometheus.Histo
	mTransferStageExecutionTimeDuration           *baseprometheus.Histo
}

//nolint:funlen
func NewMetrics(ctx context.Context, mPrefix string) (*MetricsBus, error) {
	l := glog.FromContext(ctx)
	m := &MetricsBus{
		log:     l,
		baseBus: baseprometheus.NewBus[MetricsBus](ctx, strings.TrimRight(mPrefix, "_")+"_"),
	}

	var err error

	if m.mAppInfo, err = m.baseBus.AddCounter(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mAppInfo = parent.mAppInfo.ChildWith(labels)
		},
		"app_info", "Application info",
		metrics.Labels().AppVer,
		metrics.Labels().AppSdkFabricVer); err != nil {
		return nil, err
	}

	if m.mTotalTransferCreated, err = m.baseBus.AddCounter(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTotalTransferCreated = parent.mTotalTransferCreated.ChildWith(labels)
		},
		"total_transfer_created", "Number of created transfers",
		metrics.Labels().Channel); err != nil {
		return nil, err
	}

	if m.mTotalReconnectsToFabric, err = m.baseBus.AddCounter(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTotalReconnectsToFabric = parent.mTotalReconnectsToFabric.ChildWith(labels)
		},
		"total_reconnects_to_fabric", "number of reconnect to HLF",
		metrics.Labels().Channel); err != nil {
		return nil, err
	}

	if m.mTotalSuccessTransfer, err = m.baseBus.AddCounter(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTotalSuccessTransfer = parent.mTotalSuccessTransfer.ChildWith(labels)
		},
		"total_success_transfer", "Number success of created transfers",
		metrics.Labels().Channel); err != nil {
		return nil, err
	}

	if m.mTotalFailureTransfer, err = m.baseBus.AddCounter(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTotalFailureTransfer = parent.mTotalFailureTransfer.ChildWith(labels)
		},
		"total_failure_transfer", "Number of failure ended transfers",
		metrics.Labels().Channel,
		metrics.Labels().FailTransferTag); err != nil {
		return nil, err
	}

	if m.mTotalInWorkTransfer, err = m.baseBus.AddGauge(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTotalInWorkTransfer = parent.mTotalInWorkTransfer.ChildWith(labels)
		},
		"total_in_work_transfer", "Number of transfers in work",
		metrics.Labels().Channel,
		metrics.Labels().TransferStatus); err != nil {
		return nil, err
	}

	if m.mAppInitDuration, err = m.baseBus.AddGauge(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mAppInitDuration = parent.mAppInitDuration.ChildWith(labels)
		},
		"application_init_duration", "General service status: application init duration",
	); err != nil {
		return nil, err
	}

	if m.mFabricConnectionStatus, err = m.baseBus.AddGauge(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mFabricConnectionStatus = parent.mFabricConnectionStatus.ChildWith(labels)
		},
		"fabric_connection_status", "HLF connection status",
		metrics.Labels().Channel); err != nil {
		return nil, err
	}

	if m.mCollectorProcessBlockNum, err = m.baseBus.AddGauge(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mCollectorProcessBlockNum = parent.mCollectorProcessBlockNum.ChildWith(labels)
		},
		"collector_process_block_num", "Block number processed by the collector",
		metrics.Labels().Channel); err != nil {
		return nil, err
	}

	if m.mTimeDurationCompleteTransferBeforeResponding, err = m.baseBus.AddHisto(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTimeDurationCompleteTransferBeforeResponding = parent.mTimeDurationCompleteTransferBeforeResponding.ChildWith(labels)
		},
		"time_duration_complete_transfer_before_responding",
		"Time duration to complete the transfer before responding to the user in seconds",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 20, 30, 40, 50, 60},
		metrics.Labels().Channel,
	); err != nil {
		return nil, err
	}

	if m.mTransferExecutionTimeDuration, err = m.baseBus.AddHisto(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTransferExecutionTimeDuration = parent.mTransferExecutionTimeDuration.ChildWith(labels)
		},
		"transfer_execution_time_duration",
		"Transfer execution time duration from start to finish in seconds",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 20, 30, 40, 50, 60},
		metrics.Labels().Channel,
	); err != nil {
		return nil, err
	}

	if m.mTransferStageExecutionTimeDuration, err = m.baseBus.AddHisto(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTransferStageExecutionTimeDuration = parent.mTransferStageExecutionTimeDuration.ChildWith(labels)
		},
		"time_duration_transfer_stage_execution",
		"time duration of transfer stage execution",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 20, 30, 40, 50, 60},
		metrics.Labels().Channel,
		metrics.Labels().TransferStatus,
	); err != nil {
		return nil, err
	}

	l.Info("prometheus metrics created")
	return m, nil
}

func (m *MetricsBus) CreateChild(labels ...basemetrics.Label) metrics.Metrics {
	if len(labels) == 0 {
		return m
	}

	return m.baseBus.CreateChild(func(baseChildBus *baseprometheus.BaseMetricsBus[MetricsBus]) *MetricsBus {
		return &MetricsBus{
			log:     m.log,
			baseBus: baseChildBus,
		}
	}, m, labels...)
}

func (m *MetricsBus) AppInfo() basemetrics.Counter {
	return m.mAppInfo
}

func (m *MetricsBus) CollectorProcessBlockNum() basemetrics.Gauge {
	return m.mCollectorProcessBlockNum
}

func (m *MetricsBus) TotalTransferCreated() basemetrics.Counter {
	return m.mTotalTransferCreated
}

func (m *MetricsBus) TotalReconnectsToFabric() basemetrics.Counter {
	return m.mTotalReconnectsToFabric
}

func (m *MetricsBus) TotalSuccessTransfer() basemetrics.Counter {
	return m.mTotalSuccessTransfer
}

func (m *MetricsBus) TotalFailureTransfer() basemetrics.Counter {
	return m.mTotalFailureTransfer
}

func (m *MetricsBus) TotalInWorkTransfer() basemetrics.Gauge {
	return m.mTotalInWorkTransfer
}

func (m *MetricsBus) AppInitDuration() basemetrics.Gauge {
	return m.mAppInitDuration
}

func (m *MetricsBus) FabricConnectionStatus() basemetrics.Gauge {
	return m.mFabricConnectionStatus
}

func (m *MetricsBus) TimeDurationCompleteTransferBeforeResponding() basemetrics.Histogram {
	return m.mTimeDurationCompleteTransferBeforeResponding
}

func (m *MetricsBus) TransferExecutionTimeDuration() basemetrics.Histogram {
	return m.mTransferExecutionTimeDuration
}

func (m *MetricsBus) TransferStageExecutionTimeDuration() basemetrics.Histogram {
	return m.mTransferStageExecutionTimeDuration
}
