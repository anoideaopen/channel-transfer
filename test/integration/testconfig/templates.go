package testconfig

import "fmt"

func ChannelTransferConfigWithBatcherTemplate(batcherPort string) string {
	return fmt.Sprintf(`{{ with $w := . -}}
logLevel: debug
logType: console
profilePath: {{ .ConnectionPath User }}
userName: backend
listenAPI:
  accessToken: {{ .ChannelTransferAccessToken }}
  addressHTTP: {{ .ChannelTransferHTTPAddress }}
  addressGRPC: {{ .ChannelTransferGRPCAddress }}
service:
  address: {{ .ChannelTransferHostAddress }}
options:
  batchTxPreimagePrefix: batchTransactions
  collectorsBufSize: 1
  executeTimeout: 0s
  retryExecuteAttempts: 3
  retryExecuteMaxDelay: 2s
  retryExecuteDelay: 500ms
  ttl: {{ .ChannelTransferTTL }}
  transfersInHandleOnChannel: 50
  newestRequestStreamBufferSize: 50
channels:{{ range .Channels }}
  {{- if ne . "acl" }}
  - name: {{ . }}
    batcher:
      addressGRPC: "localhost:%s"
  {{- end }}
{{- end }}
redisStorage:
  addr:{{ range .ChannelTransfer.RedisAddresses }}
    - {{ . }}
  {{- end }}
  dbPrefix: transfer
  password: ""
  afterTransferTTL: 3600s	
promMetrics:
  prefix: transfer
{{ end }}
`, batcherPort)
}

func ChannelTransferConfigTemplate() string {
	return `{{ with $w := . -}}
logLevel: debug
logType: console
profilePath: {{ .ConnectionPath User }}
userName: backend
listenAPI:
  accessToken: {{ .ChannelTransferAccessToken }}
  addressHTTP: {{ .ChannelTransferHTTPAddress }}
  addressGRPC: {{ .ChannelTransferGRPCAddress }}
service:
  address: {{ .ChannelTransferHostAddress }}
options:
  batchTxPreimagePrefix: batchTransactions
  collectorsBufSize: 1
  executeTimeout: 0s
  retryExecuteAttempts: 3
  retryExecuteMaxDelay: 2s
  retryExecuteDelay: 500ms
  ttl: {{ .ChannelTransferTTL }}
  transfersInHandleOnChannel: 50
  newestRequestStreamBufferSize: 50
channels:{{ range .Channels }}
  {{- if ne . "acl" }}
  - name: {{ . }}
  {{- end }}
{{- end }}
redisStorage:
  addr:{{ range .ChannelTransfer.RedisAddresses }}
    - {{ . }}
  {{- end }}
  dbPrefix: transfer
  password: ""
  afterTransferTTL: 3600s	
promMetrics:
  prefix: transfer
{{ end }}
`
}
