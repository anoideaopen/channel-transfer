package logger

import (
	"github.com/newity/glog"
)

type Labels struct {
	Version   string
	Component ComponentName
	UserName  string
	OrgName   string
	Channel   string
}

type ComponentName string

var (
	ComponentMain           ComponentName = "main"
	ComponentAPI            ComponentName = "api"
	ComponentStorage        ComponentName = "redis"
	ComponentParser         ComponentName = "parser"
	ComponentCollector      ComponentName = "collector"
	ComponentSubscribeEvent ComponentName = "SubscribeEvent"
	ComponentExecutor       ComponentName = "executor"
	ComponentDemultiplexer  ComponentName = "demultiplexer"
	ComponentHLFStreamsPool ComponentName = "hlf steams pool"
	ComponentHealth         ComponentName = "healthcheck"
	ComponentService        ComponentName = "service"
	ComponentProducer       ComponentName = "producer"
)

func (l Labels) Fields() (fields []glog.Field) {
	if l.Version != "" {
		fields = append(fields, glog.Field{K: "version", V: l.Version})
	}
	if l.UserName != "" {
		fields = append(fields, glog.Field{K: "user", V: l.UserName})
	}
	if l.OrgName != "" {
		fields = append(fields, glog.Field{K: "org", V: l.OrgName})
	}
	if l.Component != "" {
		fields = append(fields, glog.Field{K: "component", V: l.Component})
	}
	if l.Channel != "" {
		fields = append(fields, glog.Field{K: "channel", V: l.Channel})
	}
	return
}
