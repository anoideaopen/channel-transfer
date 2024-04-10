package nerrors

import "github.com/anoideaopen/common-component/errorshlp"

const (
	ErrTypeHlf      errorshlp.ErrType = "hlf"
	ErrTypeRedis    errorshlp.ErrType = "redis"
	ErrTypeAPI      errorshlp.ErrType = "api"
	ErrTypeParsing  errorshlp.ErrType = "parsing"
	ErrTypeInternal errorshlp.ErrType = "internal"
	ErrTypeProducer errorshlp.ErrType = "producer"
	ErrTypeHTTP     errorshlp.ErrType = "httpserver"

	ComponentStorage        errorshlp.ComponentName = "storage"
	ComponentAPI            errorshlp.ComponentName = "api"
	ComponentCollector      errorshlp.ComponentName = "collector"
	ComponentParser         errorshlp.ComponentName = "parser"
	ComponentExecutor       errorshlp.ComponentName = "executor"
	ComponentHLFStreamsPool errorshlp.ComponentName = "hlf streams pool"
	ComponentDemultiplexer  errorshlp.ComponentName = "demultiplexer"
	ComponentProducer       errorshlp.ComponentName = "producer"
	ComponenHTTP            errorshlp.ComponentName = "http"
)
