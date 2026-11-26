package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/JackCC703/kite-rpc-health/internal/checker"
)

func main() {
	rpcURL := flag.String("rpc", "https://rpc-testnet.gokite.ai/", "KiteAI testnet RPC endpoint")
	timeout := flag.Duration("timeout", 8*time.Second, "per-request timeout")
	minBlock := flag.Uint64("min-block", 0, "minimum acceptable block number (0 disables threshold)")
	peers := flag.String("compare", "", "additional comma-separated RPC endpoints")
	maxLag := flag.Uint64("max-lag", 5, "maximum block lag in comparison mode")
	flag.Parse()
	if *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "timeout must be positive")
		os.Exit(2)
	}
	client := checker.NewClient(*timeout)
	var result any
	var healthy bool
	if *peers == "" {
		single := client.CheckWithMinBlock(context.Background(), *rpcURL, *minBlock)
		result, healthy = single, single.Healthy
	} else {
		endpoints := []string{strings.TrimSpace(*rpcURL)}
		seen := map[string]bool{endpoints[0]: true}
		for _, raw := range strings.Split(*peers, ",") {
			endpoint := strings.TrimSpace(raw)
			if endpoint == "" || seen[endpoint] {
				fmt.Fprintln(os.Stderr, "comparison endpoints must be nonempty and distinct")
				os.Exit(2)
			}
			seen[endpoint] = true
			endpoints = append(endpoints, endpoint)
		}
		if endpoints[0] == "" {
			fmt.Fprintln(os.Stderr, "primary RPC endpoint must not be empty")
			os.Exit(2)
		}
		comparison := client.Compare(context.Background(), endpoints, *minBlock, *maxLag)
		result, healthy = comparison, comparison.Healthy
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if !healthy {
		os.Exit(1)
	}
}
