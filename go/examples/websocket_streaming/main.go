// WebSocket Streaming Example — Real-time market data via WebSocket.
//
// This example demonstrates:
// - Connecting to Hyperliquid's WebSocket API
// - Subscribing to trades, orders, and book updates
// - Automatic reconnection handling
// - Graceful shutdown
//
// This example matches the Python websocket_streaming.py exactly.
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
		fmt.Println("Hyperliquid WebSocket Streaming Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		fmt.Println()
		fmt.Println("Or:")
		fmt.Println("  go run main.go 'https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		os.Exit(1)
	}

	fmt.Println("Hyperliquid WebSocket Streaming Example")
	fmt.Println("==================================================")
	displayEndpoint := endpoint
	if len(displayEndpoint) > 60 {
		displayEndpoint = displayEndpoint[:60] + "..."
	}
	fmt.Printf("Endpoint: %s\n", displayEndpoint)
	fmt.Println()

	// Create stream with all callbacks
	stream := hyperliquid.NewStream(endpoint, &hyperliquid.StreamConfig{
		Reconnect:    true, // Auto-reconnect on disconnect
		PingInterval: 30,   // Heartbeat every 30 seconds
		OnError: func(err error) {
			fmt.Printf("[ERROR] %v\n", err)
		},
		OnClose: func() {
			fmt.Println("[CLOSED] Stream stopped")
		},
		OnStateChange: func(state hyperliquid.ConnectionState) {
			fmt.Printf("[STATE] %s\n", state)
		},
		OnReconnect: func(attempt int) {
			fmt.Printf("[RECONNECT] Attempt %d\n", attempt)
		},
	})

	// Subscribe to BTC and ETH trades
	stream.Trades([]string{"BTC", "ETH"}, func(data map[string]any) {
		// QuickNode format: { type: 'data', stream: 'hl.trades', block: { events: [...] } }
		block, ok := data["block"].(map[string]any)
		if !ok {
			return
		}
		events, ok := block["events"].([]any)
		if !ok {
			return
		}
		for _, event := range events {
			eventArr, ok := event.([]any)
			if !ok || len(eventArr) < 2 {
				continue
			}
			trade, ok := eventArr[1].(map[string]any)
			if !ok {
				continue
			}
			coin, _ := trade["coin"].(string)
			px, _ := trade["px"].(string)
			sz, _ := trade["sz"].(string)
			side, _ := trade["side"].(string)
			pxFloat, _ := strconv.ParseFloat(px, 64)

			sideStr := "SELL"
			if side == "B" {
				sideStr = "BUY"
			}
			fmt.Printf("[TRADE] %s: %s %s @ $%.2f\n", coin, sideStr, sz, pxFloat)
		}
	})
	fmt.Println("Subscribed to: BTC, ETH trades")

	// Subscribe to BTC book updates
	stream.BookUpdates([]string{"BTC"}, func(data map[string]any) {
		// QuickNode format: { type: 'data', stream: 'hl.book_updates', block: { events: [...] } }
		block, ok := data["block"].(map[string]any)
		if !ok {
			return
		}
		events, ok := block["events"].([]any)
		if !ok {
			return
		}
		for _, event := range events {
			eventArr, ok := event.([]any)
			if !ok || len(eventArr) < 2 {
				continue
			}
			update, ok := eventArr[1].(map[string]any)
			if !ok {
				continue
			}

			coin, _ := update["coin"].(string)
			levels, ok := update["levels"].([]any)
			if !ok || len(levels) < 2 {
				continue
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

				fmt.Printf("[BOOK] %s: Bid $%.2f | Ask $%.2f | Spread $%.2f\n", coin, bidPxFloat, askPxFloat, spread)
			}
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
	fmt.Println("Streaming... Press Ctrl+C to stop")
	fmt.Println("--------------------------------------------------")

	// Run the stream (blocking)
	// Use stream.Start() for non-blocking background mode
	if err := stream.Start(); err != nil {
		fmt.Printf("Failed to start stream: %v\n", err)
		return
	}

	// Keep running until signal
	select {}
}
