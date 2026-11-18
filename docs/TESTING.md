# Testing and deployment

```sh
go test -race ./...
go vet ./...
go build ./cmd/kite-rpc-health
```

Test cases include successful KiteAI chain ID and block height, a wrong chain, JSON-RPC errors, malformed JSON, and HTTP failure. All network tests use `httptest` and are deterministic.

Deployment: build and copy the binary for the target OS/architecture using `GOOS`/`GOARCH`; invoke it from a cron job or CI monitor. Do not treat a one-shot CLI success as production-grade availability monitoring.

Troubleshooting: if a public RPC is unreachable, verify internet/DNS connectivity and try an operator-approved endpoint. If the result is a wrong chain ID, check the URL rather than bypassing validation.

## Live KiteAI testnet verification (2026-11-18)

On the contributor's macOS machine, the following commands succeeded:

```sh
go test ./...
go vet ./...
go run ./cmd/kite-rpc-health -timeout 8s
```

Observed JSON-RPC response (snapshot, **not** a constant or mock value):

```json
{"rpc_url":"https://rpc-testnet.gokite.ai/","chain_id":2368,"block_number":22045141,"latency_ms":6597,"healthy":true}
```

The check completed with exit code 0. Latency and block height are point-in-time observations; availability can change. This demonstrates basic RPC connectivity and chain-ID validation, not transaction settlement or consensus correctness.
