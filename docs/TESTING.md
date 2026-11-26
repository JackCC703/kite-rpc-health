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

## Error-handling regression coverage (2026-11-22)

Local HTTP test servers verify that a slow RPC request fails with a deadline error and that mismatched JSON-RPC response IDs, unsupported protocol versions, missing results, and invalid hex quantities are rejected. No public endpoint or credentials are needed for these deterministic tests.

## Diagnostic output and response limits (2026-11-23)

When a reachable RPC reports a different chain ID, or the block-number RPC fails after chain identification, the health-check JSON includes the observed `chain_id` alongside `healthy: false` and the error. This helps diagnose wrong-network URLs and partial endpoint failures. The checker also rejects JSON-RPC bodies larger than 1 MiB; deterministic local-server tests cover both partial failure and oversize-response handling.

## Minimum-block freshness threshold (2026-11-25)

The `-min-block` CLI flag passes a nonnegative minimum acceptable block number to `CheckWithMinBlock`. The default `0` leaves the prior behavior unchanged. If the reported chain is Kite Testnet but its block height is lower than the threshold, the result includes the actual `chain_id`, `block_number`, `healthy: false`, and a specific "below minimum" error; the process exits 1. A failed JSON-RPC read remains the primary error instead of being confused with a stale-block error. Tests cover disabled, lower, equal, and higher thresholds as well as RPC failure.

Example: `go run ./cmd/kite-rpc-health -min-block 22000000`. Use a recent trusted reference height when setting the threshold; this option alone cannot establish finality or detect chain reorganization.

## Multi-endpoint comparison (2026-11-26)

Local HTTP fixtures cover lagging nodes, the exact lag threshold, wrong-chain exclusion, HTTP failures, the absence of a usable reference, valid zero block heights, and interaction with the absolute minimum block threshold. The comparison preserves input ordering and shares an HTTP client across bounded concurrent workers. Run `go test -race ./...` to check concurrent access.
