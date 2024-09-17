package patch

/*
  Functions that patch the channel transfer configuration, bringing it to a new format
  with external batcher information for channels. When the new configuration format is
  supported in foundation, these functions can be removed.
*/

import (
	"fmt"
	"html/template"
	"io"
	"os"

	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

// ChannelTransferConfig changes format of channel description in the configuration
// to the new form which is:
// channels:
//   - name: channel1
//   - name: channel2
func ChannelTransferConfig(networkFound *cmn.NetworkFoundation, channels []string) {
	const channelTransferConfigPatchedTemplate = `{{ with $w := . -}}
logLevel: debug
logType: console
profilePath: {{ .ConnectionPath }}
userName: backend
listenAPI:
  accessToken: {{ .AccessToken }}
  addressHTTP: {{ .HTTPAddress }}
  addressGRPC: {{ .GRPCAddress }}
service:
  address: {{ .HostAddress }}
options:
  batchTxPreimagePrefix: batchTransactions
  collectorsBufSize: 1
  executeTimeout: 0s
  retryExecuteAttempts: 3
  retryExecuteMaxDelay: 2s
  retryExecuteDelay: 500ms
  ttl: {{ .TTL }}
  transfersInHandleOnChannel: 50
  newestRequestStreamBufferSize: 50
channels:{{ range .Channels }}
  {{- if ne .Name "acl" }}
  - name: {{ .Name }}
  {{- end }}
{{- end }}
redisStorage:
  addr:{{ range .RedisAddresses }}
    - {{ . }}
  {{- end }}
  dbPrefix: transfer
  password: ""
  afterTransferTTL: 3600s	
promMetrics:
  prefix: transfer
{{ end }}
`

	type channelTransferConfigPatched struct {
		AccessToken    string
		ConnectionPath string
		HTTPAddress    string
		GRPCAddress    string
		HostAddress    string
		RedisAddresses []string
		TTL            string
		Channels       []struct {
			Name string
		}
	}

	newConfig := channelTransferConfigPatched{
		AccessToken:    networkFound.ChannelTransferAccessToken(),
		ConnectionPath: networkFound.ConnectionPath("User2"),
		HTTPAddress:    networkFound.ChannelTransferHTTPAddress(),
		GRPCAddress:    networkFound.ChannelTransferGRPCAddress(),
		HostAddress:    networkFound.ChannelTransferHostAddress(),
		RedisAddresses: networkFound.ChannelTransfer.RedisAddresses,
		TTL:            networkFound.ChannelTransferTTL(),
	}

	for _, channel := range channels {
		newConfig.Channels = append(newConfig.Channels, struct {
			Name string
		}{Name: channel})
	}

	t, err := template.New("channel_transfer").Funcs(template.FuncMap{
		"User": func() string { return "User2" },
	}).Parse(channelTransferConfigPatchedTemplate)
	Expect(err).NotTo(HaveOccurred())

	pw := gexec.NewPrefixedWriter("[channel_transfer.yaml] ", ginkgo.GinkgoWriter)
	config, err := os.Create(networkFound.ChannelTransferPath())
	Expect(err).NotTo(HaveOccurred())
	err = t.Execute(io.MultiWriter(config, pw), newConfig)
	Expect(err).NotTo(HaveOccurred())
}

// ChannelTransferConfigWithBatcher changes format of channel description in the configuration
// to the new form and adds batcher data to every channel:
// channels:
//
//   - name: channel1
//     batcher:
//       addressGRPC: "localhost:8881"
//   - name: channel2
//     batcher:
//       addressGRPC: "localhost:8881"
func ChannelTransferConfigWithBatcher(networkFound *cmn.NetworkFoundation, channels []string, batcherPort string) {
	const channelTransferConfigWithBatcherTemplate = `{{ with $w := . -}}
logLevel: debug
logType: console
profilePath: {{ .ConnectionPath }}
userName: backend
listenAPI:
  accessToken: {{ .AccessToken }}
  addressHTTP: {{ .HTTPAddress }}
  addressGRPC: {{ .GRPCAddress }}
service:
  address: {{ .HostAddress }}
options:
  batchTxPreimagePrefix: batchTransactions
  collectorsBufSize: 1
  executeTimeout: 0s
  retryExecuteAttempts: 3
  retryExecuteMaxDelay: 2s
  retryExecuteDelay: 500ms
  ttl: {{ .TTL }}
  transfersInHandleOnChannel: 50
  newestRequestStreamBufferSize: 50
channels:{{ range .Channels }}
  {{- if ne .Name "acl" }}
  - name: {{ .Name }}
    batcher:
      addressGRPC: {{ .BatcherAddress }}
  {{- end }}
{{- end }}
redisStorage:
  addr:{{ range .RedisAddresses }}
    - {{ . }}
  {{- end }}
  dbPrefix: transfer
  password: ""
  afterTransferTTL: 3600s	
promMetrics:
  prefix: transfer
{{ end }}
`

	type channelTransferConfigWithBatcher struct {
		AccessToken    string
		ConnectionPath string
		HTTPAddress    string
		GRPCAddress    string
		HostAddress    string
		RedisAddresses []string
		TTL            string
		Channels       []struct {
			Name           string
			BatcherAddress string
		}
	}

	newConfig := channelTransferConfigWithBatcher{
		AccessToken:    networkFound.ChannelTransferAccessToken(),
		ConnectionPath: networkFound.ConnectionPath("User2"),
		HTTPAddress:    networkFound.ChannelTransferHTTPAddress(),
		GRPCAddress:    networkFound.ChannelTransferGRPCAddress(),
		HostAddress:    networkFound.ChannelTransferHostAddress(),
		RedisAddresses: networkFound.ChannelTransfer.RedisAddresses,
		TTL:            networkFound.ChannelTransferTTL(),
	}

	for _, channel := range channels {
		newConfig.Channels = append(newConfig.Channels, struct {
			Name           string
			BatcherAddress string
		}{Name: channel, BatcherAddress: fmt.Sprintf("localhost:%s", batcherPort)})
	}

	t, err := template.New("channel_transfer").Funcs(template.FuncMap{
		"User": func() string { return "User2" },
	}).Parse(channelTransferConfigWithBatcherTemplate)
	Expect(err).NotTo(HaveOccurred())

	pw := gexec.NewPrefixedWriter("[channel_transfer.yaml] ", ginkgo.GinkgoWriter)
	config, err := os.Create(networkFound.ChannelTransferPath())
	Expect(err).NotTo(HaveOccurred())
	err = t.Execute(io.MultiWriter(config, pw), newConfig)
	Expect(err).NotTo(HaveOccurred())
}
