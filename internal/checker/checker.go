package checker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const KiteTestnetChainID uint64 = 2368

type Result struct {
	RPCURL      string `json:"rpc_url"`
	ChainID     uint64 `json:"chain_id"`
	BlockNumber uint64 `json:"block_number"`
	LatencyMS   int64  `json:"latency_ms"`
	Healthy     bool   `json:"healthy"`
	Error       string `json:"error,omitempty"`
}

type Client struct{ http *http.Client }

func NewClient(timeout time.Duration) *Client { return &Client{http: &http.Client{Timeout: timeout}} }

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) call(ctx context.Context, endpoint, method string) (uint64, error) {
	reqBody, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": []any{}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%s request: %w", method, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%s HTTP status %d", method, resp.StatusCode)
	}
	// Limit untrusted RPC responses to 1 MiB.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
	if err != nil {
		return 0, fmt.Errorf("%s response read: %w", method, err)
	}
	if len(body) > 1<<20 {
		return 0, fmt.Errorf("%s response too large", method)
	}
	var parsed rpcResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, fmt.Errorf("%s invalid JSON: %w", method, err)
	}
	if parsed.JSONRPC != "2.0" {
		return 0, fmt.Errorf("%s invalid JSON-RPC version", method)
	}
	if string(parsed.ID) != "1" {
		return 0, fmt.Errorf("%s mismatched response ID", method)
	}
	if parsed.Error != nil {
		return 0, fmt.Errorf("%s RPC error %d: %s", method, parsed.Error.Code, parsed.Error.Message)
	}
	if len(parsed.Result) == 0 || string(parsed.Result) == "null" {
		return 0, fmt.Errorf("%s missing result", method)
	}
	var hexValue string
	if err := json.Unmarshal(parsed.Result, &hexValue); err != nil {
		return 0, fmt.Errorf("%s result not a hex string", method)
	}
	if !strings.HasPrefix(hexValue, "0x") || len(hexValue) <= 2 {
		return 0, fmt.Errorf("%s result lacks 0x hex value", method)
	}
	n, err := strconv.ParseUint(hexValue[2:], 16, 64)
	if err != nil {
		return 0, fmt.Errorf("%s invalid hex result: %w", method, err)
	}
	return n, nil
}

func (c *Client) Check(ctx context.Context, endpoint string) Result {
	r := Result{RPCURL: endpoint}
	if endpoint == "" {
		r.Error = "RPC URL is empty"
		return r
	}
	start := time.Now()
	chain, err := c.call(ctx, endpoint, "eth_chainId")
	if err == nil && chain != KiteTestnetChainID {
		err = fmt.Errorf("unexpected chain ID %d (expected %d)", chain, KiteTestnetChainID)
	}
	if err == nil {
		r.ChainID = chain
		r.BlockNumber, err = c.call(ctx, endpoint, "eth_blockNumber")
	}
	r.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		r.Error = err.Error()
		return r
	}
	r.Healthy = true
	return r
}
