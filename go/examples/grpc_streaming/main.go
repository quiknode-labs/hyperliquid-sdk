// gRPC Streaming Example — High-performance real-time market data via gRPC.
//
// This example demonstrates:
// - Connecting to Hyperliquid's gRPC streaming API
// - Subscribing to trades, orders, blocks, and L2/L4 order books
// - Automatic reconnection handling
// - Graceful shutdown
//
// gRPC offers lower latency than WebSocket for high-frequency data.
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
		fmt.Println("Hyperliquid gRPC Streaming Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export ENDPOINT='https://YOUR-ENDPOINT/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Hyperliquid gRPC Streaming Example")
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

	// Create gRPC stream with all callbacks via SDK
	stream := sdk.NewGRPCStream(&hyperliquid.GRPCStreamConfig{
		Reconnect: true, // Auto-reconnect on disconnect
		OnError: func(err error) {
			fmt.Printf("[ERROR] %v\n", err)
		},
		OnClose: func() {
			fmt.Println("[CLOSED] gRPC stream stopped")
		},
		OnConnect: func() {
			fmt.Println("[CONNECTED] gRPC stream ready")
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
		coin, _ := data["coin"].(string)
		px, _ := data["px"].(string)
		sz, _ := data["sz"].(string)
		side, _ := data["side"].(string)

		pxFloat, _ := strconv.ParseFloat(px, 64)
		sideStr := "SELL"
		if side == "B" {
			sideStr = "BUY"
		}
		fmt.Printf("[TRADE] %s: %s %s @ $%.2f\n", coin, sideStr, sz, pxFloat)
	})
	fmt.Println("Subscribed to: BTC, ETH trades")

	// Subscribe to book updates
	stream.BookUpdates([]string{"BTC"}, func(data map[string]any) {
		coin, _ := data["coin"].(string)
		bids, _ := data["bids"].([]any)
		asks, _ := data["asks"].([]any)

		if len(bids) > 0 && len(asks) > 0 {
			bestBid := bids[0].(map[string]any)
			bestAsk := asks[0].(map[string]any)
			bidPrice, _ := bestBid["price"].(string)
			askPrice, _ := bestAsk["price"].(string)
			bidPxFloat, _ := strconv.ParseFloat(bidPrice, 64)
			askPxFloat, _ := strconv.ParseFloat(askPrice, 64)
			fmt.Printf("[BOOK] %s: Bid $%.2f | Ask $%.2f\n", coin, bidPxFloat, askPxFloat)
		}
	})
	fmt.Println("Subscribed to: BTC book updates")

	// Subscribe to L2 order book
	stream.L2Book("ETH", func(data map[string]any) {
		coin, _ := data["coin"].(string)
		bids, _ := data["bids"].([][]any)
		asks, _ := data["asks"].([][]any)
		fmt.Printf("[L2] %s: %d bid levels, %d ask levels\n", coin, len(bids), len(asks))
	})
	fmt.Println("Subscribed to: ETH L2 order book")

	// Subscribe to blocks
	stream.Blocks(func(data map[string]any) {
		blockNum, _ := data["block_number"].(string)
		fmt.Printf("[BLOCK] #%s\n", blockNum)
	})
	fmt.Println("Subscribed to: blocks")

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
	fmt.Println("Streaming via gRPC... Press Ctrl+C to stop")
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
