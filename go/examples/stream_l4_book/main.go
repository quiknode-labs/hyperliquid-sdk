// Stream L4 Book Example — Real-time L4 orderbook updates via WebSocket.
//
// L4 book provides more granular orderbook data with additional depth
// compared to L2, useful for market making and algorithmic trading.
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("ENDPOINT")

	if endpoint == "" {
		fmt.Println("Stream L4 Book Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export ENDPOINT='https://YOUR-ENDPOINT/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Stream L4 Book Example")
	fmt.Println("==================================================")
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

	// Create stream via SDK
	stream := sdk.NewStream(&hyperliquid.StreamConfig{
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

	// Subscribe to book updates for BTC (provides detailed book data)
	stream.BookUpdates([]string{"BTC"}, func(data map[string]any) {
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

			// Calculate total volume at top 5 levels
			var bidVol, askVol float64
			for i := 0; i < 5 && i < len(bids); i++ {
				b := bids[i].(map[string]any)
				sz, _ := b["sz"].(string)
				v, _ := strconv.ParseFloat(sz, 64)
				bidVol += v
			}
			for i := 0; i < 5 && i < len(asks); i++ {
				a := asks[i].(map[string]any)
				sz, _ := a["sz"].(string)
				v, _ := strconv.ParseFloat(sz, 64)
				askVol += v
			}

			fmt.Printf("[BOOK] %s: Bid $%.2f | Ask $%.2f | Spread $%.2f | Top5 Vol: %.4f/%.4f\n",
				coin, bidPxFloat, askPxFloat, spread, bidVol, askVol)
		}
	})
	fmt.Println("Subscribed to: BTC book updates")

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
	fmt.Println("Streaming book updates... Press Ctrl+C to stop")
	fmt.Println("--------------------------------------------------")

	// Start the stream
	if err := stream.Start(); err != nil {
		fmt.Printf("Failed to start stream: %v\n", err)
		return
	}

	// Keep running until signal
	select {}
}
