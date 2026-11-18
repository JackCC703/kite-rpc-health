# Testing and deployment

```sh
go test -race ./...
go vet ./...
go build ./cmd/kite-rpc-health
```

Test cases include successful KiteAI chain ID and block height, a wrong chain, JSON-RPC errors, malformed JSON, and HTTP failure. All network tests use `httptest` and are deterministic.

Deployment: build and copy the binary for the target OS/architecture using `GOOS`/`GOARCH`; invoke it from a cron job or CI monitor. Do not treat a one-shot CLI success as production-grade availability monitoring.

Troubleshooting: if a public RPC is unreachable, verify internet/DNS connectivity and try an operator-approved endpoint. If the result is a wrong chain ID, check the URL rather than bypassing validation.
