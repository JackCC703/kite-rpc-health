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
	BlockTimestamp  *uint64 `json:"block_timestamp,omitempty"`
	BlockAgeSeconds *uint64 `json:"block_age_seconds,omitempty"`
	RPCURL          string  `json:"rpc_url"`
	ChainID         uint64  `json:"chain_id"`
	BlockNumber     uint64  `json:"block_number"`
	LatencyMS       int64   `json:"latency_ms"`
	Healthy         bool    `json:"healthy"`
	Error           string  `json:"error,omitempty"`
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

func (c *Client) request(ctx context.Context, endpoint, method string, params []any) (json.RawMessage, error) {
	reqBody, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s request: %w", method, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s HTTP status %d", method, resp.StatusCode)
	}
	// Limit untrusted RPC responses to 1 MiB.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
	if err != nil {
		return nil, fmt.Errorf("%s response read: %w", method, err)
	}
	if len(body) > 1<<20 {
		return nil, fmt.Errorf("%s response too large", method)
	}
	var parsed rpcResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("%s invalid JSON: %w", method, err)
	}
	if parsed.JSONRPC != "2.0" {
		return nil, fmt.Errorf("%s invalid JSON-RPC version", method)
	}
	if string(parsed.ID) != "1" {
		return nil, fmt.Errorf("%s mismatched response ID", method)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("%s RPC error %d: %s", method, parsed.Error.Code, parsed.Error.Message)
	}
	if len(parsed.Result) == 0 || string(parsed.Result) == "null" {
		return nil, fmt.Errorf("%s missing result", method)
	}
	return parsed.Result, nil
}

func (c *Client) call(ctx context.Context, endpoint, method string) (uint64, error) {
	result, err := c.request(ctx, endpoint, method, []any{})
	if err != nil {
		return 0, err
	}
	return parseQuantity(result, method)
}

func parseQuantity(result json.RawMessage, method string) (uint64, error) {
	var hexValue string
	if err := json.Unmarshal(result, &hexValue); err != nil {
		return 0, fmt.Errorf("%s result not a hex string", method)
	}
	if !strings.HasPrefix(hexValue, "0x") || len(hexValue) <= 2 {
		return 0, fmt.Errorf("%s result lacks 0x hex value", method)
	}
	// Ethereum JSON-RPC quantities use minimal hexadecimal encoding.
	digits := hexValue[2:]
	if len(digits) > 1 && digits[0] == '0' {
		return 0, fmt.Errorf("%s invalid hex result: leading zero", method)
	}
	for _, digit := range digits {
		if !((digit >= '0' && digit <= '9') || (digit >= 'a' && digit <= 'f') || (digit >= 'A' && digit <= 'F')) {
			return 0, fmt.Errorf("%s invalid hex result: non-hexadecimal digit", method)
		}
	}
	n, err := strconv.ParseUint(digits, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("%s invalid hex result: %w", method, err)
	}
	return n, nil
}

// Check validates KiteAI chain identity and RPC responsiveness.
func (c *Client) Check(ctx context.Context, endpoint string) Result {
	return c.CheckWithMinBlock(ctx, endpoint, 0)
}

// CheckWithMinBlock can additionally reject endpoints that lag behind a caller-
// supplied minimum block height. Zero disables the threshold.
func (c *Client) CheckWithMinBlock(ctx context.Context, endpoint string, minBlock uint64) Result {
	return c.CheckWithFreshness(ctx, endpoint, minBlock, 0)
}

// CheckWithFreshness optionally validates the latest block timestamp.
func (c *Client) CheckWithFreshness(ctx context.Context, endpoint string, minBlock uint64, maxAge time.Duration) Result {
	r := Result{RPCURL: endpoint}
	if endpoint == "" {
		r.Error = "RPC URL is empty"
		return r
	}
	start := time.Now()
	chain, err := c.call(ctx, endpoint, "eth_chainId")
	if err == nil {
		// Preserve the observed chain on errors so operators can diagnose
		// misconfigured endpoints rather than seeing a misleading zero value.
		r.ChainID = chain
		if chain != KiteTestnetChainID {
			err = fmt.Errorf("unexpected chain ID %d (expected %d)", chain, KiteTestnetChainID)
		} else {
			if maxAge > 0 {
				err = c.checkLatestBlock(ctx, endpoint, &r, maxAge)
			} else {
				r.BlockNumber, err = c.call(ctx, endpoint, "eth_blockNumber")
			}
		}
	}
	if err == nil && minBlock > 0 && r.BlockNumber < minBlock {
		err = fmt.Errorf("block number %d below minimum %d", r.BlockNumber, minBlock)
	}
	r.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		r.Error = err.Error()
		return r
	}
	r.Healthy = true
	return r
}
