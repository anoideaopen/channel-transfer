# Channel-transfer — agent guide

## What it is

Go service that transfers tokens across Hyperledger Fabric channels. Exposes
gRPC + HTTP (grpc-gateway) APIs, backed by Redis, with optional
OpenTelemetry tracing and Prometheus metrics.

- Entrypoint: `main.go` → `pkg/app.Run()`
- Config: spf13/viper, env prefix `CHANNEL_TRANSFER_` (e.g.
  `CHANNEL_TRANSFER_REDISSTORAGE_PASSWORD`)
- Config file search order: `-c` flag → env var `CHANNEL_TRANSFER_CONFIG` → `./config.yaml` → `/etc/config.yaml`

## Commands

```shell
# build (version is injected)
go build -ldflags="-X 'main.AppInfoVer=$VERSION'"

# regenerate protos + swagger client
go generate ./...

# fetch proto dependencies
protodep up

# apply Go fixes (modernize old API usage)
go fix ./...

# unit tests (no race by default; CI adds -race)
go test -count 1 ./...

# single package
go test -count 1 ./pkg/config/

# integration test (run from test/integration/, needs Docker)
go run github.com/onsi/ginkgo/v2/ginkgo \
  --keep-going --poll-progress-after 60s --timeout 24h <suite>
# suites: grpc, http, chaos, task-executor
```

## CI pipeline order

1. check no Cyrillic in source
2. `go mod tidy` + `go fmt` (must be idempotent)
3. `golangci-lint` (config: `.golangci.yml`)
4. unit test + coverage + integration (parallel after lint)

## Key directories

| Path | Purpose |
|------|---------|
| `pkg/app/` | Bootstrap wiring |
| `pkg/transfer/` | Core logic, gRPC/HTTP servers, controllers |
| `pkg/config/` | Config struct + viper loading |
| `pkg/hlf/` | Fabric SDK wrappers |
| `pkg/chproducer/` | Per-channel transfer handler |
| `pkg/data/redis/` | Redis storage |
| `proto/` | Protobuf definitions, generated code, embedded swagger |
| `test/integration/` | Ginkgo integration tests (separate `go.mod`) |

## Quirks & gotchas

- `fabric-sdk-go` is replaced with `anoideaopen/fabric-sdk-go` in both
  `go.mod` and `test/integration/go.mod`
- Integration tests use Ginkgo v2, spin up real Fabric network via Docker,
  require `redis:7.2.4`, `hyperledger/fabric-ccenv`, `fabric-baseos` images
- `go.mod` toolchain pulls in Fabric binaries (`configtxgen`, `cryptogen`,
  `peer`, `orderer`, etc.)
- `generate.go`: runs `buf generate` then `swagger generate client`. The swagger
  client is regenerated into `test/integration/clihttp/`
- Swagger JSON is embedded at compile time via `//go:embed` in `proto/embed.go`
- Coverage config (`.testcoverage.yml`) excludes `*.pb.go` from stats
- `golangci-lint` v2 config disables test file linting (`run.tests: false`)

## Coding Principles (Karpathy Skills)

Behavioral guidelines to reduce common LLM coding mistakes. These bias toward caution over speed; for trivial tasks, use judgment.

1. **Think Before Coding** — Don't assume. Don't hide confusion. Surface tradeoffs.
- State assumptions explicitly; ask if uncertain.
- Present multiple interpretations rather than picking silently.
- If a simpler approach exists or something is unclear, say so and ask.

2. **Simplicity First** — Minimum code that solves the problem. Nothing speculative.
- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If 200 lines could be 50, rewrite it.

3. **Surgical Changes** — Touch only what you must. Clean up only your own mess.
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken. Match existing style.
- If you notice unrelated dead code, mention it — don't delete it.
- Remove only what YOUR changes made unused.
- Every changed line should trace directly to the request.

4. **Goal-Driven Execution** — Define success criteria. Loop until verified.
- "Add validation" → "Write tests for invalid inputs, then make them pass".
- "Fix the bug" → "Write a test that reproduces it, then make it pass".
- "Refactor X" → "Ensure tests pass before and after".
- For multi-step tasks, state a brief plan with `verify:` checks per step.
