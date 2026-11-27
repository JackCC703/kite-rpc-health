# CLI usage

```sh
kite-rpc-health -rpc https://rpc-testnet.gokite.ai/ -timeout 8s
```

Flags: `-rpc` selects the JSON-RPC URL and `-timeout` sets a positive HTTP timeout for each request.

Output fields: `rpc_url` (string), `chain_id` (integer), `block_number` (integer), `latency_ms` (integer), `healthy` (boolean), and optional `error` (string).

Exit codes: `0` successful KiteAI testnet RPC check, `1` RPC check failed, `2` invalid invocation/output issue.

Error examples: wrong network ID, invalid/malformed JSON-RPC result, non-200 response, timeout, or network failure. An error does not necessarily imply an on-chain fault; it can be local or network-related.

## Compare multiple endpoints

Use `-rpc` for the primary endpoint and `-compare` for additional comma-separated, distinct endpoints. For example, replace the local URLs with your own KiteAI testnet nodes:

```sh
kite-rpc-health -rpc http://127.0.0.1:8545 -compare http://127.0.0.1:8546,http://127.0.0.1:8547 -max-lag 5
```

Comparison mode returns a JSON object containing `reference_block` (highest successfully observed KiteAI block, or null if none), `max_lag`, overall `healthy`, and ordered `results`. Each successful RPC observation includes its relative `lag`; failed and wrong-chain endpoints retain their error and have no lag. The overall exit status is 1 if any endpoint fails or exceeds the lag limit. The default maximum is 5 blocks; zero requires equal observed heights. The optional `-min-block` threshold still applies. Single-endpoint output remains unchanged.

Checks run concurrently with at most eight workers. The highest observed block is a comparison reference, not proof of canonical state or finality. Normal block production and response timing can cause temporary differences; use trusted independent endpoints and a suitable tolerance.

## Latest block freshness

Use `-max-block-age 2m` to reject endpoints whose latest block timestamp is older than two minutes. The default (0) disables this check and retains the existing output format. When enabled, the checker uses `eth_getBlockByNumber("latest", false)` instead of `eth_blockNumber` so height and timestamp refer to the same observed block.

The JSON output adds `block_timestamp` (Unix seconds) and `block_age_seconds` when available. Stale blocks, future timestamps, missing block data and malformed quantities make the endpoint unhealthy. In comparison mode, unhealthy observations cannot establish the relative height reference. Age is evaluated at response time with one-second resolution, against the local clock. Set a tolerance appropriate for the network and keep the machine clock synchronized. This is not a proof of finality or canonical chain state.
