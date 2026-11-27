package checker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Comparison reports relative lag against the highest successful observation.
// It does not establish which endpoint reports the canonical chain.
type Comparison struct {
	ReferenceBlock *uint64          `json:"reference_block"`
	MaxLag         uint64           `json:"max_lag"`
	Healthy        bool             `json:"healthy"`
	Results        []ComparedResult `json:"results"`
}

type ComparedResult struct {
	Result
	Lag *uint64 `json:"lag,omitempty"`
}

// Compare checks endpoints concurrently with bounded concurrency and preserves
// input order. Failed or wrong-chain endpoints cannot establish the reference.
func (c *Client) Compare(ctx context.Context, endpoints []string, minBlock, maxLag uint64) Comparison {
	return c.CompareWithFreshness(ctx, endpoints, minBlock, maxLag, 0)
}

func (c *Client) CompareWithFreshness(ctx context.Context, endpoints []string, minBlock, maxLag uint64, maxAge time.Duration) Comparison {
	report := Comparison{MaxLag: maxLag, Results: make([]ComparedResult, len(endpoints))}
	var wg sync.WaitGroup
	jobs := make(chan int)
	workers := min(8, len(endpoints))
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				report.Results[i].Result = c.CheckWithFreshness(ctx, endpoints[i], 0, maxAge)
			}
		}()
	}
	for i := range endpoints {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	var highest uint64
	for _, r := range report.Results {
		if r.Healthy && (report.ReferenceBlock == nil || r.BlockNumber > highest) {
			highest = r.BlockNumber
			report.ReferenceBlock = &highest
		}
	}
	report.Healthy = len(endpoints) >= 2 && report.ReferenceBlock != nil
	for i := range report.Results {
		r := &report.Results[i]
		if r.Healthy {
			lag := highest - r.BlockNumber
			r.Lag = &lag
			switch {
			case r.BlockNumber < minBlock:
				r.Error = fmt.Sprintf("block number %d below minimum %d", r.BlockNumber, minBlock)
			case lag > maxLag:
				r.Error = fmt.Sprintf("block lag %d exceeds maximum %d", lag, maxLag)
			}
			r.Healthy = r.Error == ""
		}
		report.Healthy = report.Healthy && r.Healthy
	}
	return report
}
