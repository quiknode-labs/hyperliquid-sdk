// Stream L2 Book Example — Real-time L2 orderbook updates via gRPC.
//
// This example demonstrates subscribing to L2 orderbook data with
// aggregated bid/ask levels for efficient streaming.
//
// L2 order book provides aggregated price levels (total size at each price).
// Use gRPC for L2 book streaming for lower latency.
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("ENDPOINT")

	if endpoint == "" {
		fmt.Println("============================================================")
		fmt.Println("L2 Order Book Streaming")
		fmt.Println("============================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export ENDPOINT='https://YOUR-ENDPOINT/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("============================================================")
	fmt.Println("L2 Order Book Streaming")
	fmt.Println("============================================================")
	displayEndpoint := endpoint
	if len(displayEndpoint) > 60 {
		displayEndpoint = displayEndpoint[:60] + "..."
	}
	fmt.Printf("Endpoint: %s\n", displayEndpoint)
	fmt.Println()

	// Create SDK
	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		log.Fatalf("Failed to create SDK: %v", err)
	}

	// Track update counts
	updateCount := 0

	// Create gRPC stream via SDK (L2 book is best via gRPC for low latency)
	stream := sdk.NewGRPCStream(&hyperliquid.GRPCStreamConfig{
		Secure:    true,
		Reconnect: false,
		OnConnect: func() {
			fmt.Println("[CONNECTED]")
		},
		OnError: func(err error) {
			fmt.Printf("[ERROR] %v\n", err)
		},
	})

	// Subscribe to ETH L2 book with 20 levels
	stream.L2Book("ETH", func(data map[string]any) {
		updateCount++
		bids, _ := data["bids"].([][]any)
		asks, _ := data["asks"].([][]any)

		if len(bids) > 0 && len(asks) > 0 {
			bidPx := bids[0][0]
			bidSz := bids[0][1]
			askPx := asks[0][0]
			askSz := asks[0][1]

			fmt.Printf("[%s] ETH\n", timestamp())
			fmt.Printf("  Bid: %v @ $%v\n", bidSz, bidPx)
			fmt.Printf("  Ask: %v @ $%v\n", askSz, askPx)
			fmt.Printf("  Spread: $%.2f (%.2f bps)\n", spread(bids, asks), spreadBps(bids, asks))
			fmt.Printf("  Levels: %d bids, %d asks\n", len(bids), len(asks))
		}

		if updateCount >= 5 {
			fmt.Printf("\nReceived %d L2 updates.\n", updateCount)
		}
	}, hyperliquid.L2BookNLevels(20))

	fmt.Println("Subscribing to ETH L2 order book...")
	fmt.Println("------------------------------------------------------------")

	if err := stream.Start(); err != nil {
		fmt.Printf("Start error: %v\n", err)
		return
	}

	start := time.Now()
	for updateCount < 5 && time.Since(start) < 15*time.Second {
		time.Sleep(100 * time.Millisecond)
	}
	stream.Stop()

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("Done!")
}

func timestamp() string {
	return time.Now().Format("15:04:05.000")
}

func spread(bids, asks [][]any) float64 {
	if len(bids) == 0 || len(asks) == 0 {
		return 0
	}
	bidPx, _ := bids[0][0].(float64)
	askPx, _ := asks[0][0].(float64)
	return askPx - bidPx
}

func spreadBps(bids, asks [][]any) float64 {
	if len(bids) == 0 || len(asks) == 0 {
		return 0
	}
	bidPx, _ := bids[0][0].(float64)
	askPx, _ := asks[0][0].(float64)
	mid := (bidPx + askPx) / 2
	if mid == 0 {
		return 0
	}
	return (askPx - bidPx) / mid * 10000
}
