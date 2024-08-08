module github.com/anoideaopen/channel-transfer

go 1.22.5

require (
	github.com/anoideaopen/common-component v0.0.6
	github.com/anoideaopen/foundation v0.0.5
	github.com/anoideaopen/glog v0.0.3
	github.com/avast/retry-go/v4 v4.6.0
	github.com/felixge/httpsnoop v1.0.4
	github.com/flowchartsman/swaggerui v0.0.0-20221017034628-909ed4f3701b
	github.com/go-errors/errors v1.5.1
	github.com/go-playground/validator/v10 v10.10.1
	github.com/golang/protobuf v1.5.4
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus v1.0.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0
	github.com/hyperledger/fabric-protos-go v0.3.2
	github.com/hyperledger/fabric-sdk-go v1.0.1-0.20210927191040-3e3a3c6aeec9
	github.com/redis/go-redis/v9 v9.0.4
	github.com/slok/go-http-metrics v0.10.0
	github.com/spf13/viper v1.19.0
	github.com/stretchr/testify v1.9.0
	go.uber.org/zap v1.27.0
	golang.org/x/sync v0.8.0
	google.golang.org/genproto v0.0.0-20240213162025-012b6fc9bca9
	google.golang.org/genproto/googleapis/api v0.0.0-20240528184218-531527333157
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240604185151-ef581f913117
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/sagikazarmark/locafero v0.6.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56 // indirect
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.5.1 // indirect
)

require (
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible // indirect
	github.com/alicebob/gopher-json v0.0.0-20230218143504-906a9b012302 // indirect
	github.com/alicebob/miniredis/v2 v2.30.1
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/cfssl v1.4.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/envoyproxy/protoc-gen-validate v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-kit/kit v0.10.0 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/golang/mock v1.6.0
	github.com/google/certificate-transparency-go v1.0.21 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.1.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20230228194215-b84622ba6a7a // indirect
	github.com/hyperledger/fabric-config v0.2.1 // indirect
	github.com/hyperledger/fabric-lib-go v1.1.1 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/onsi/gomega v1.33.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.14.0
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.39.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.7.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/weppos/publicsuffix-go v0.5.0 // indirect
	github.com/yuin/gopher-lua v1.1.0 // indirect
	github.com/zmap/zcrypto v0.0.0-20190729165852-9051775e6a2e // indirect
	github.com/zmap/zlint v0.0.0-20190806154020-fd021b4cfbeb // indirect
	go.uber.org/atomic v1.9.0
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.23.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/anoideaopen/foundation => ../foundation
	github.com/hyperledger/fabric-sdk-go => github.com/anoideaopen/fabric-sdk-go v0.0.2
)
