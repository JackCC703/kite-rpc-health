# CLI usage

```sh
kite-rpc-health -rpc https://rpc-testnet.gokite.ai/ -timeout 8s
```

Flags: `-rpc` selects the JSON-RPC URL and `-timeout` sets a positive HTTP timeout for each request.

Output fields: `rpc_url` (string), `chain_id` (integer), `block_number` (integer), `latency_ms` (integer), `healthy` (boolean), and optional `error` (string).

Exit codes: `0` successful KiteAI testnet RPC check, `1` RPC check failed, `2` invalid invocation/output issue.

Error examples: wrong network ID, invalid/malformed JSON-RPC result, non-200 response, timeout, or network failure. An error does not necessarily imply an on-chain fault; it can be local or network-related.
