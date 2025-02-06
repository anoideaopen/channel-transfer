module github.com/anoideaopen/channel-transfer

go 1.23.5

require (
	github.com/anoideaopen/common-component v0.0.6
	github.com/anoideaopen/foundation v0.1.0
	github.com/anoideaopen/glog v0.0.3
	github.com/avast/retry-go/v4 v4.6.0
	github.com/felixge/httpsnoop v1.0.4
	github.com/flowchartsman/swaggerui v0.0.0-20221017034628-909ed4f3701b
	github.com/go-errors/errors v1.5.1
	github.com/go-playground/validator/v10 v10.23.0
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus v1.0.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.24.0
	github.com/hyperledger/fabric-protos-go-apiv2 v0.3.5
	github.com/hyperledger/fabric-sdk-go v1.0.0
	github.com/redis/go-redis/v9 v9.7.0
	github.com/slok/go-http-metrics v0.13.0
	github.com/spf13/viper v1.20.0-alpha.6
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/otel v1.32.0
	go.uber.org/zap v1.27.0
	golang.org/x/sync v0.10.0
	google.golang.org/genproto/googleapis/api v0.0.0-20241206012308-a4fef0638583
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f
	google.golang.org/grpc v1.70.0
	google.golang.org/protobuf v1.36.4
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible // indirect
	github.com/alicebob/gopher-json v0.0.0-20230218143504-906a9b012302 // indirect
	github.com/alicebob/miniredis/v2 v2.30.1
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/btcsuite/btcd/btcutil v1.1.5 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/cfssl v1.4.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/envoyproxy/protoc-gen-validate v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.5 // indirect
	github.com/go-kit/kit v0.10.0 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.0.0 // indirect
	github.com/golang/mock v1.6.0
	github.com/google/certificate-transparency-go v1.0.21 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.1.0 // indirect
	github.com/hyperledger/fabric-chaincode-go/v2 v2.1.0 // indirect
	github.com/hyperledger/fabric-config v0.3.0 // indirect
	github.com/hyperledger/fabric-lib-go v1.1.3-0.20240523144151-25edd1eaf5f5 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/miekg/pkcs11 v1.1.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/onsi/ginkgo/v2 v2.17.2 // indirect
	github.com/onsi/gomega v1.33.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.20.3
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.59.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/sagikazarmark/locafero v0.6.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/sykesm/zap-logfmt v0.0.4 // indirect
	github.com/weppos/publicsuffix-go v0.5.0 // indirect
	github.com/yuin/gopher-lua v1.1.0 // indirect
	github.com/zmap/zcrypto v0.0.0-20190729165852-9051775e6a2e // indirect
	github.com/zmap/zlint v0.0.0-20190806154020-fd021b4cfbeb // indirect
	go.opentelemetry.io/otel/trace v1.32.0 // indirect
	go.uber.org/atomic v1.11.0
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.32.0 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/tools v0.24.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/hyperledger/fabric-sdk-go => github.com/anoideaopen/fabric-sdk-go v0.1.0
