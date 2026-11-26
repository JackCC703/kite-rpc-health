# Changelog

## 0.1.0 - 2026-11-18

- Add KiteAI testnet chain ID and block-height health check.
- Add JSON output, exit status, error validation, timeout configuration.
- Add deterministic HTTP test fixtures and project documentation.

## 2026-11-25
- Add an optional minimum block-height health threshold (`-min-block`) for detecting lagging KiteAI testnet RPC endpoints, with regression tests and documentation.

## 2026-11-26
- Add bounded concurrent multi-RPC comparison with configurable maximum relative block lag, per-endpoint diagnostics, and backward-compatible single-node output.
