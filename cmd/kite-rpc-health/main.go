package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/JackCC703/kite-rpc-health/internal/checker"
)

func main() {
	rpcURL := flag.String("rpc", "https://rpc-testnet.gokite.ai/", "KiteAI testnet RPC endpoint")
	timeout := flag.Duration("timeout", 8*time.Second, "per-request timeout")
	flag.Parse()
	if *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "timeout must be positive")
		os.Exit(2)
	}
	result := checker.NewClient(*timeout).Check(context.Background(), *rpcURL)
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if !result.Healthy {
		os.Exit(1)
	}
}
