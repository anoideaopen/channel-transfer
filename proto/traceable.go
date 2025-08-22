package proto

import (
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
)

func (x *TransferBeginAdminRequest) GetTraceAttributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("id", x.GetIdTransfer()),
		attribute.String("request_id", x.GetGenerals().GetRequestId()),
		attribute.String("from", x.GetGenerals().GetChannel()),
		attribute.String("to", x.GetChannelTo()),
		attribute.String("token", x.GetToken()),
		attribute.String("amount", x.GetAmount()),
	}
}

func (x *TransferBeginCustomerRequest) GetTraceAttributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("id", x.GetIdTransfer()),
		attribute.String("request_id", x.GetGenerals().GetRequestId()),
		attribute.String("from", x.GetGenerals().GetChannel()),
		attribute.String("to", x.GetChannelTo()),
		attribute.String("token", x.GetToken()),
		attribute.String("amount", x.GetAmount()),
	}
}

func (x *MultiTransferBeginCustomerRequest) GetTraceAttributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("id", x.GetIdTransfer()),
		attribute.String("request_id", x.GetGenerals().GetRequestId()),
		attribute.String("from", x.GetGenerals().GetChannel()),
		attribute.String("to", x.GetChannelTo()),
		attribute.String("items", getItemsAsString(x.GetItems())),
	}
}

func (x *MultiTransferBeginAdminRequest) GetTraceAttributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("id", x.GetIdTransfer()),
		attribute.String("request_id", x.GetGenerals().GetRequestId()),
		attribute.String("from", x.GetGenerals().GetChannel()),
		attribute.String("to", x.GetChannelTo()),
		attribute.String("items", getItemsAsString(x.GetItems())),
	}
}

func getItemsAsString(itemList []*TransferItem) string {
	tokenList := make([]string, 0, len(itemList))
	for _, item := range itemList {
		tokenList = append(tokenList, fmt.Sprintf("token: %s amount: %s", item.GetToken(), item.GetAmount()))
	}
	return strings.Join(tokenList, ", ")
}
