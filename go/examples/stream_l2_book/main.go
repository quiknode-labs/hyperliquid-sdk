// Stream L2 Book Example — Real-time L2 orderbook updates via WebSocket.
//
// This example demonstrates subscribing to L2 orderbook data with
// aggregated bid/ask levels for efficient streaming.
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
		fmt.Println("Stream L2 Book Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Stream L2 Book Example")
	fmt.Println("==================================================")
	displayEndpoint := endpoint
	if len(displayEndpoint) > 60 {
		displayEndpoint = displayEndpoint[:60] + "..."
	}
	fmt.Printf("Endpoint: %s\n", displayEndpoint)
	fmt.Println()

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

	// Subscribe to L2 book updates for BTC (single coin per subscription)
	stream.L2Book("BTC", func(data map[string]any) {
		bookData, ok := data["data"].(map[string]any)
		if !ok {
			return
		}

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
			bidSz, _ := bestBid["sz"].(string)
			askSz, _ := bestAsk["sz"].(string)
			bidPxFloat, _ := strconv.ParseFloat(bidPx, 64)
			askPxFloat, _ := strconv.ParseFloat(askPx, 64)
			spread := askPxFloat - bidPxFloat

			fmt.Printf("[L2] %s: Bid %s @ $%.2f | Ask %s @ $%.2f | Spread $%.2f | Depth: %d/%d\n",
				coin, bidSz, bidPxFloat, askSz, askPxFloat, spread, len(bids), len(asks))
		}
	})

	// Also subscribe to ETH
	stream.L2Book("ETH", func(data map[string]any) {
		bookData, ok := data["data"].(map[string]any)
		if !ok {
			return
		}

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
			spread := askPxFloat - bidPxFloat

			fmt.Printf("[L2] %s: Bid $%.2f | Ask $%.2f | Spread $%.2f | Depth: %d/%d\n",
				coin, bidPxFloat, askPxFloat, spread, len(bids), len(asks))
		}
	})
	fmt.Println("Subscribed to: BTC, ETH L2 book")

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down gracefully...")
		stream.Stop()
		os.Exit(0)
	}()

	fmt.Println()
	fmt.Println("Streaming L2 book updates... Press Ctrl+C to stop")
	fmt.Println("--------------------------------------------------")

	// Start the stream
	if err := stream.Start(); err != nil {
		fmt.Printf("Failed to start stream: %v\n", err)
		return
	}

	// Keep running until signal
	select {}
}
