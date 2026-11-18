# Architecture

`cmd/kite-rpc-health/main.go` parses flags, runs one check and prints JSON.
`internal/checker/checker.go` sends sequential JSON-RPC requests over HTTP.

Flow: `CLI -> eth_chainId -> validate 2368 -> eth_blockNumber -> JSON output + exit code`.

Network calls use Go's standard `http.Client` timeout. The checker validates HTTP status, JSON-RPC version and request ID, JSON result shape, and hexadecimal conversion. It never signs transactions or manages keys.

## Development

Run `go test ./...` and `go vet ./...`. Extend behavior with test-server fixtures so tests remain deterministic without a public RPC dependency.
