package metrics

import "github.com/anoideaopen/common-component/basemetrics"

type devNullMetrics struct{}

func (m *devNullMetrics) TotalTransferCreated() basemetrics.Counter { return m }

func (m *devNullMetrics) TotalReconnectsToFabric() basemetrics.Counter { return m }

func (m *devNullMetrics) TotalReconnectsToRedis() basemetrics.Counter { return m }

func (m *devNullMetrics) TotalSuccessTransfer() basemetrics.Counter { return m }

func (m *devNullMetrics) TotalFailureTransfer() basemetrics.Counter { return m }

func (m *devNullMetrics) TotalInWorkTransfer() basemetrics.Gauge { return m }

func (m *devNullMetrics) AppInitDuration() basemetrics.Gauge { return m }

func (m *devNullMetrics) FabricConnectionStatus() basemetrics.Gauge { return m }

func (m *devNullMetrics) TimeDurationCompleteTransferBeforeResponding() basemetrics.Histogram {
	return m
}

func (m *devNullMetrics) TransferExecutionTimeDuration() basemetrics.Histogram { return m }

func (m *devNullMetrics) TransferStageExecutionTimeDuration() basemetrics.Histogram { return m }

func (m *devNullMetrics) CollectorProcessBlockNum() basemetrics.Gauge { return m }

func (m *devNullMetrics) AppInfo() basemetrics.Counter { return m }

func (m *devNullMetrics) Inc(_ ...basemetrics.Label) { /* nothing just stub */ }

func (m *devNullMetrics) Add(_ float64, _ ...basemetrics.Label) { /* nothing just stub */ }

func (m *devNullMetrics) Set(_ float64, _ ...basemetrics.Label) { /* nothing just stub */ }

func (m *devNullMetrics) Observe(_ float64, _ ...basemetrics.Label) { /* nothing just stub */ }

func (m *devNullMetrics) CreateChild(_ ...basemetrics.Label) Metrics { return m }
