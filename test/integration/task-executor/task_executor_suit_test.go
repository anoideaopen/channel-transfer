package task_executor

import (
	"encoding/json"
	"testing"

	"github.com/anoideaopen/foundation/test/integration/cmn/runner"
	"github.com/hyperledger/fabric/integration/nwo"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBatcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Channel transfer with batcher GRPC suite")
}

var (
	buildServer *nwo.BuildServer
	components  *nwo.Components
)

var _ = SynchronizedBeforeSuite(func() []byte {
	nwo.RequiredImages = []string{nwo.CCEnvDefaultImage, runner.RedisDefaultImage}

	buildServer = nwo.NewBuildServer()
	buildServer.Serve()

	components = buildServer.Components()
	payload, err := json.Marshal(components)
	Expect(err).NotTo(HaveOccurred())

	return payload
}, func(payload []byte) {
	err := json.Unmarshal(payload, &components)
	Expect(err).NotTo(HaveOccurred())
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	buildServer.Shutdown()
})
