// HyperCore Block Data Example
//
// Shows how to get real-time trades, orders, and block data via the HyperCore API.
//
// This is the alternative to Info methods (allMids, l2Book, recentTrades) that
// are not available on QuickNode endpoints.
//
// Usage:
//
//	export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/YOUR_TOKEN"
//	go run main.go
package main

import (
	"fmt"
	"os"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("ENDPOINT")
	if endpoint == "" {
		fmt.Println("Set ENDPOINT environment variable")
		os.Exit(1)
	}

	// Single SDK instance — access everything through sdk.Info(), sdk.Core(), sdk.EVM(), etc.
	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		fmt.Printf("Error creating SDK: %v\n", err)
		os.Exit(1)
	}
	hc := sdk.Core()

	fmt.Println(string(repeat('=', 50)))
	fmt.Println("HyperCore Block Data")
	fmt.Println(string(repeat('=', 50)))

	// Latest block number
	fmt.Println("\n1. Latest Block:")
	blockNum, err := hc.LatestBlockNumber()
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		return
	}
	fmt.Printf("   Block #%d\n", blockNum)

	// Recent trades
	fmt.Println("\n2. Recent Trades (all coins):")
	trades, err := hc.LatestTrades(5, "")
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		for _, t := range trades {
			side := "BUY"
			if t["side"] == "A" {
				side = "SELL"
			}
			fmt.Printf("   %s %v %v @ $%v\n", side, t["sz"], t["coin"], t["px"])
		}
	}

	// Recent BTC trades only
	fmt.Println("\n3. BTC Trades:")
	btcTrades, err := hc.LatestTrades(10, "BTC")
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else if len(btcTrades) == 0 {
		fmt.Println("   No BTC trades in recent blocks")
	} else {
		for _, t := range btcTrades[:min(3, len(btcTrades))] {
			side := "BUY"
			if t["side"] == "A" {
				side = "SELL"
			}
			fmt.Printf("   %s %v @ $%v\n", side, t["sz"], t["px"])
		}
	}

	// Get a specific block
	fmt.Println("\n4. Get Block Data:")
	block, err := hc.GetBlock(blockNum - 1)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Block #%d\n", blockNum-1)
		fmt.Printf("   Time: %v\n", block["block_time"])
		events, _ := block["events"].([]any)
		fmt.Printf("   Events: %d\n", len(events))
	}

	// Get batch of blocks
	fmt.Println("\n5. Batch Blocks:")
	blocks, err := hc.GetBatchBlocks(blockNum-5, blockNum-1)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Retrieved %d blocks\n", len(blocks))
	}

	fmt.Println("\n" + string(repeat('=', 50)))
	fmt.Println("Done!")
}

func repeat(b byte, n int) []byte {
	result := make([]byte, n)
	for i := range result {
		result[i] = b
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
