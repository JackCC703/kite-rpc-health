# kite-rpc-health

A small, original Go CLI that checks a KiteAI Testnet JSON-RPC endpoint.

## What it checks

- `eth_chainId` must return **2368** (`0x940`), the KiteAI Testnet chain ID.
- `eth_blockNumber` must return a valid hexadecimal block height.
- Measures the time spent fetching both results, including network latency.
- Emits one JSON result and exits with status 0 on success or 1 on RPC failure.

This tool makes actual calls to KiteAI RPC. It does **not** perform payments, verify x402 settlement, check mainnet, or prove chain finality.

## Requirements

Go 1.23 or later. No third-party Go dependencies.

## Run

```sh
go run ./cmd/kite-rpc-health
# Or run with a custom endpoint and timeout:
go run ./cmd/kite-rpc-health -rpc http://127.0.0.1:8545 -timeout 3s
```

Example response format (illustrative, not a live result):

```json
{"rpc_url":"https://rpc-testnet.gokite.ai/","chain_id":2368,"block_number":123,"latency_ms":120,"healthy":true}
```

If the endpoint is unavailable, the output includes `healthy:false` and an `error` string. Never pass credentials in public command transcripts.

## Build and test

```sh
go fmt ./...
go vet ./...
go test ./...
go build -o kite-rpc-health ./cmd/kite-rpc-health
```

## KiteAI relationship

The application performs actual Ethereum-compatible JSON-RPC requests to the [KiteAI testnet endpoint](https://rpc-testnet.gokite.ai/) and verifies KiteAI's chain ID (2368). See `internal/checker/checker.go` for the implementation. Local tests use mock HTTP servers and do not assert live network availability.

## Docs

- [Architecture](docs/ARCHITECTURE.md)
- [Usage and error behavior](docs/USAGE.md)
- [Testing and deployment](docs/TESTING.md)
- [Changelog](CHANGELOG.md)

## Limitations

- Testnet only; the chain ID check intentionally rejects other EVM networks.
- Checks endpoint responsiveness, not consensus, finality, or chain data accuracy.
- Each run performs two sequential JSON-RPC requests.
- No API authentication options; intended for public RPC or local testing.

Licensed under MIT; see `LICENSE`.
