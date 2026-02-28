// HyperCore API Example — Low-level block and trade data via JSON-RPC.
//
// This example shows how to query blocks, trades, and orders using the HyperCore API.
// This example matches the Python hypercore_example.py exactly.
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("QUICKNODE_ENDPOINT")
	}

	if endpoint == "" {
		fmt.Println("Error: Set QUICKNODE_ENDPOINT environment variable")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'")
		os.Exit(1)
	}

	fmt.Println("Hyperliquid HyperCore API Example")
	fmt.Println("==================================================")
	fmt.Printf("Endpoint: %s...\n", truncate(endpoint, 50))
	fmt.Println()

	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		log.Fatalf("Failed to create SDK: %v", err)
	}

	// Create HyperCore client
	hc := sdk.Core()

	// ==========================================================================
	// Block Data
	// ==========================================================================
	fmt.Println("Block Data")
	fmt.Println("------------------------------")

	// Get latest block number (default stream is "trades")
	blockNum, err := hc.LatestBlockNumber()
	if err != nil {
		log.Printf("Failed to get latest block: %v", err)
	} else {
		fmt.Printf("Latest block: %d\n", blockNum)
	}

	// Get block by number (default stream is "trades")
	block, err := hc.GetBlock(blockNum)
	if err != nil {
		log.Printf("Failed to get block: %v", err)
	} else if block != nil {
		events, _ := block["events"].([]any)
		fmt.Printf("Block %d:\n", blockNum)
		fmt.Printf("  Events: %d\n", len(events))
		fmt.Printf("  Timestamp: %v\n", block["timestamp"])
	}
	fmt.Println()

	// ==========================================================================
	// Recent Trades
	// ==========================================================================
	fmt.Println("Recent Trades")
	fmt.Println("------------------------------")

	// Get latest trades (all coins)
	trades, err := hc.LatestTrades(5, "")
	if err != nil {
		log.Printf("Failed to get trades: %v", err)
	} else {
		fmt.Printf("Last %d trades:\n", len(trades))
		for i, trade := range trades {
			if i >= 5 {
				break
			}
			coin, _ := trade["coin"].(string)
			px, _ := trade["px"].(string)
			sz, _ := trade["sz"].(string)
			side, _ := trade["side"].(string)
			pxFloat, _ := strconv.ParseFloat(px, 64)
			fmt.Printf("  %s: %s @ $%.2f (%s)\n", coin, sz, pxFloat, side)
		}
	}
	fmt.Println()

	// Get trades for specific coin
	btcTrades, err := hc.LatestTrades(3, "BTC")
	if err != nil {
		log.Printf("Failed to get BTC trades: %v", err)
	} else {
		fmt.Printf("Last %d BTC trades:\n", len(btcTrades))
		for _, trade := range btcTrades {
			px, _ := trade["px"].(string)
			sz, _ := trade["sz"].(string)
			side, _ := trade["side"].(string)
			pxFloat, _ := strconv.ParseFloat(px, 64)
			fmt.Printf("  %s @ $%.2f (%s)\n", sz, pxFloat, side)
		}
	}
	fmt.Println()

	// ==========================================================================
	// Recent Orders
	// ==========================================================================
	fmt.Println("Recent Orders")
	fmt.Println("------------------------------")

	// Get latest orders (all coins)
	orders, err := hc.LatestOrders(5, "")
	if err != nil {
		fmt.Printf("  Could not fetch orders: %v\n", err)
	} else {
		fmt.Printf("Last %d orders:\n", len(orders))
		for i, order := range orders {
			if i >= 5 {
				break
			}
			coin, _ := order["coin"].(string)
			side, _ := order["side"].(string)
			px, _ := order["limitPx"].(string)
			sz, _ := order["sz"].(string)
			status, _ := order["status"].(string)
			pxFloat, _ := strconv.ParseFloat(px, 64)
			fmt.Printf("  %s: %s %s @ $%.2f - %s\n", coin, side, sz, pxFloat, status)
		}
	}
	fmt.Println()

	// ==========================================================================
	// Block Range Query
	// ==========================================================================
	fmt.Println("Block Range Query")
	fmt.Println("------------------------------")

	// Get batch blocks (default stream is "trades")
	startBlock := blockNum - 5
	if startBlock < 0 {
		startBlock = 0
	}
	blocks, err := hc.GetBatchBlocks(startBlock, blockNum)
	if err != nil {
		fmt.Printf("  Could not fetch blocks: %v\n", err)
	} else {
		fmt.Printf("Blocks %d to %d: %d blocks\n", startBlock, blockNum, len(blocks))

		// Safely count events
		totalEvents := 0
		for _, b := range blocks {
			if bMap, ok := b.(map[string]any); ok {
				if events, ok := bMap["events"].([]any); ok {
					totalEvents += len(events)
				}
			}
		}
		fmt.Printf("Total events: %d\n", totalEvents)
	}

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
