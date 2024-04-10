package metrics

import (
	"github.com/anoideaopen/common-component/basemetrics"
)

type LabelNames struct {
	Channel         basemetrics.LabelName
	AppVer          basemetrics.LabelName
	AppSdkFabricVer basemetrics.LabelName
	FailTransferTag basemetrics.LabelName
	TransferStatus  basemetrics.LabelName
}

func Labels() LabelNames {
	return allLabels
}

var allLabels = createLabels()

func createLabels() LabelNames {
	return LabelNames{
		Channel:         "channel",
		AppVer:          "ver",
		AppSdkFabricVer: "ver_sdk_fabric",
		FailTransferTag: "failure_tag",
		TransferStatus:  "transfer_status",
	}
}

type Metrics interface { //nolint:interfacebloat
	TotalTransferCreated() basemetrics.Counter
	TotalReconnectsToFabric() basemetrics.Counter
	TotalSuccessTransfer() basemetrics.Counter
	TotalFailureTransfer() basemetrics.Counter

	TotalInWorkTransfer() basemetrics.Gauge
	AppInitDuration() basemetrics.Gauge
	FabricConnectionStatus() basemetrics.Gauge

	TimeDurationCompleteTransferBeforeResponding() basemetrics.Histogram
	TransferExecutionTimeDuration() basemetrics.Histogram
	TransferStageExecutionTimeDuration() basemetrics.Histogram

	CollectorProcessBlockNum() basemetrics.Gauge
	AppInfo() basemetrics.Counter
	CreateChild(labels ...basemetrics.Label) Metrics
}
