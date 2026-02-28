// Stream Orderbook Example — Real-time orderbook snapshots via WebSocket.
//
// This example demonstrates receiving full orderbook snapshots
// which can be used to maintain a local orderbook state.
//
// This example matches the Python streaming patterns exactly.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/quiknode-labs/raptor/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("QUICKNODE_ENDPOINT")
	}

	if endpoint == "" {
		fmt.Println("Stream Orderbook Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Stream Orderbook Example")
	fmt.Println("==================================================")
	displayEndpoint := endpoint
	if len(displayEndpoint) > 60 {
		displayEndpoint = displayEndpoint[:60] + "..."
	}
	fmt.Printf("Endpoint: %s\n", displayEndpoint)
	fmt.Println()

	// Track update counts
	updateCount := 0

	// Create stream
	stream := hyperliquid.NewStream(endpoint, &hyperliquid.StreamConfig{
		Reconnect:    true,
		PingInterval: 30,
		OnError: func(err error) {
			fmt.Printf("[ERROR] %v\n", err)
		},
		OnClose: func() {
			fmt.Println("[CLOSED] Stream stopped")
		},
		OnStateChange: func(state hyperliquid.ConnectionState) {
			fmt.Printf("[STATE] %s\n", state)
		},
	})

	// Subscribe to book updates (snapshots)
	stream.BookUpdates([]string{"BTC", "ETH", "SOL"}, func(data map[string]any) {
		bookData, ok := data["data"].(map[string]any)
		if !ok {
			return
		}

		updateCount++
		coin, _ := bookData["coin"].(string)
		levels, ok := bookData["levels"].([]any)
		if !ok || len(levels) < 2 {
			return
		}

		bids, _ := levels[0].([]any)
		asks, _ := levels[1].([]any)

		if len(bids) > 0 && len(asks) > 0 {
			bestBid := bids[0].(map[string]any)
			bestAsk := asks[0].(map[string]any)

			bidPx, _ := bestBid["px"].(string)
			askPx, _ := bestAsk["px"].(string)
			bidPxFloat, _ := strconv.ParseFloat(bidPx, 64)
			askPxFloat, _ := strconv.ParseFloat(askPx, 64)
			midPrice := (bidPxFloat + askPxFloat) / 2

			fmt.Printf("[BOOK #%d] %s: Mid $%.2f | Bid $%.2f | Ask $%.2f | Levels: %d/%d\n",
				updateCount, coin, midPrice, bidPxFloat, askPxFloat, len(bids), len(asks))
		}
	})
	fmt.Println("Subscribed to: BTC, ETH, SOL orderbook")

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down gracefully...")
		fmt.Printf("Total updates received: %d\n", updateCount)
		stream.Stop()
		os.Exit(0)
	}()

	fmt.Println()
	fmt.Println("Streaming orderbook snapshots... Press Ctrl+C to stop")
	fmt.Println("--------------------------------------------------")

	// Start the stream
	if err := stream.Start(); err != nil {
		fmt.Printf("Failed to start stream: %v\n", err)
		return
	}

	// Keep running until signal
	select {}
}
