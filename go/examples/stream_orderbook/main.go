// Stream Orderbook Example — L2 and L4 orderbook streaming.
//
// Demonstrates both L2 (aggregated price levels) and L4 (individual orders)
// orderbook streaming via gRPC.
//
// L2: Aggregated - shows total size at each price level
// L4: Individual - shows each order with its own ID and size
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("ENDPOINT")

	if endpoint == "" {
		fmt.Println("Order Book Streaming Examples")
		fmt.Println("============================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export ENDPOINT='https://YOUR-ENDPOINT/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Order Book Streaming Examples")
	fmt.Println("============================================================")

	// Create SDK
	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		log.Fatalf("Failed to create SDK: %v", err)
	}

	// ========================================================================
	// L2 ORDER BOOK (Aggregated Price Levels)
	// ========================================================================
	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("L2 ORDER BOOK (Aggregated Price Levels)")
	fmt.Println("============================================================")

	l2Count := 0

	l2Stream := sdk.NewGRPCStream(&hyperliquid.GRPCStreamConfig{
		Secure:    true,
		Reconnect: false,
	})

	l2Stream.L2Book("BTC", func(data map[string]any) {
		l2Count++
		bids, _ := data["bids"].([][]any)
		asks, _ := data["asks"].([][]any)

		if len(bids) > 0 && len(asks) > 0 {
			bidPx := parseFloat(bids[0][0])
			askPx := parseFloat(asks[0][0])
			spread := askPx - bidPx
			mid := (bidPx + askPx) / 2
			spreadBps := spread / mid * 10000

			fmt.Printf("[%s] BTC L2 Update #%d\n", timestamp(), l2Count)
			fmt.Printf("  Best Bid: $%.0f\n", bidPx)
			fmt.Printf("  Best Ask: $%.0f\n", askPx)
			fmt.Printf("  Spread: $%.2f (%.2f bps)\n", spread, spreadBps)
			fmt.Printf("  Depth: %d bid levels, %d ask levels\n", len(bids), len(asks))
		}

		if l2Count >= 3 {
			fmt.Printf("\nReceived %d L2 updates.\n", l2Count)
		}
	}, hyperliquid.L2BookNLevels(10))

	if err := l2Stream.Start(); err != nil {
		fmt.Printf("Start error: %v\n", err)
		return
	}

	start := time.Now()
	for l2Count < 3 && time.Since(start) < 15*time.Second {
		time.Sleep(100 * time.Millisecond)
	}
	l2Stream.Stop()

	// ========================================================================
	// L4 ORDER BOOK (Individual Orders)
	// ========================================================================
	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("L4 ORDER BOOK (Individual Orders)")
	fmt.Println("============================================================")

	l4Count := 0

	l4Stream := sdk.NewGRPCStream(&hyperliquid.GRPCStreamConfig{
		Secure:    true,
		Reconnect: false,
	})

	l4Stream.L4Book("ETH", func(data map[string]any) {
		l4Count++
		isSnapshot, _ := data["snapshot"].(bool)

		if isSnapshot {
			bids, _ := data["bids"].([]any)
			asks, _ := data["asks"].([]any)
			fmt.Printf("[%s] ETH L4 Snapshot\n", timestamp())
			fmt.Printf("  %d individual bid orders\n", len(bids))
			fmt.Printf("  %d individual ask orders\n", len(asks))
		} else {
			height, _ := data["height"].(int64)
			fmt.Printf("[%s] ETH L4 Diff (height: %d)\n", timestamp(), height)
		}

		if l4Count >= 3 {
			fmt.Printf("\nReceived %d L4 updates.\n", l4Count)
		}
	})

	if err := l4Stream.Start(); err != nil {
		fmt.Printf("Start error: %v\n", err)
		return
	}

	start = time.Now()
	for l4Count < 3 && time.Since(start) < 15*time.Second {
		time.Sleep(100 * time.Millisecond)
	}
	l4Stream.Stop()

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("Done!")
}

func timestamp() string {
	return time.Now().Format("15:04:05.000")
}

func parseFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}
