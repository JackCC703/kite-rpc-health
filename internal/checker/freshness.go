package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Read height and timestamp from the same block, avoiding inconsistent
// observations if the chain advances between two separate RPC requests.
func (c *Client) checkLatestBlock(ctx context.Context, endpoint string, r *Result, maxAge time.Duration) error {
	const method = "eth_getBlockByNumber"
	raw, err := c.request(ctx, endpoint, method, []any{"latest", false})
	if err != nil {
		return err
	}
	var block struct {
		Number    json.RawMessage `json:"number"`
		Timestamp json.RawMessage `json:"timestamp"`
	}
	if err := json.Unmarshal(raw, &block); err != nil {
		return fmt.Errorf("%s invalid block: %w", method, err)
	}
	r.BlockNumber, err = parseQuantity(block.Number, method+" number")
	if err != nil {
		return err
	}
	timestamp, err := parseQuantity(block.Timestamp, method+" timestamp")
	if err != nil {
		return err
	}
	r.BlockTimestamp = &timestamp
	now := time.Now()
	if timestamp > uint64(now.Unix()) {
		return fmt.Errorf("block timestamp %d is in the future", timestamp)
	}
	age := uint64(now.Unix()) - timestamp
	r.BlockAgeSeconds = &age
	// Compare seconds without converting an untrusted timestamp to a duration.
	if age > uint64(maxAge/time.Second) {
		return fmt.Errorf("block age %ds exceeds maximum %s", age, maxAge)
	}
	return nil
}
