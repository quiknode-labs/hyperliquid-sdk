// gRPC Streaming Example — High-Performance Real-Time Data
//
// Stream trades, orders, L2 book, L4 book, and blocks via gRPC.
// gRPC provides lower latency than WebSocket for high-frequency trading.
//
// Usage:
//
//	export ENDPOINT="https://your-endpoint/YOUR_TOKEN"
//	go run main.go
//
// The SDK:
// - Connects to port 10000 automatically
// - Passes token via x-token header
// - Handles reconnection with exponential backoff
// - Manages keepalive pings
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
		fmt.Println("gRPC Streaming Example")
		fmt.Println(string(repeat('=', 60)))
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export ENDPOINT='https://YOUR-ENDPOINT/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println(string(repeat('=', 60)))
	fmt.Println("gRPC Streaming Examples")
	fmt.Println(string(repeat('=', 60)))

	// Create SDK
	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		log.Fatalf("Failed to create SDK: %v", err)
	}

	// Example 1: Stream Trades
	fmt.Println("\nExample 1: Streaming Trades")
	fmt.Println(string(repeat('-', 60)))

	tradeCount := 0

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

	stream.Trades([]string{"BTC", "ETH"}, func(data map[string]any) {
		tradeCount++
		coin := data["coin"]
		px := data["px"]
		sz := data["sz"]
		side := "BUY "
		if data["side"] == "A" {
			side = "SELL"
		}
		fmt.Printf("[%s] %s %v %v @ $%v\n", timestamp(), side, sz, coin, px)

		if tradeCount >= 5 {
			fmt.Printf("\nReceived %d trades.\n", tradeCount)
		}
	})

	fmt.Println("Subscribing to BTC and ETH trades...")

	if err := stream.Start(); err != nil {
		fmt.Printf("Start error: %v\n", err)
		return
	}

	start := time.Now()
	for tradeCount < 5 && time.Since(start) < 15*time.Second {
		time.Sleep(100 * time.Millisecond)
	}
	stream.Stop()

	// Example 2: Stream L2 Book
	fmt.Println("\nExample 2: Streaming L2 Order Book")
	fmt.Println(string(repeat('-', 60)))

	l2Count := 0

	l2Stream := sdk.NewGRPCStream(&hyperliquid.GRPCStreamConfig{
		Secure:    true,
		Reconnect: false,
	})

	l2Stream.L2Book("ETH", func(data map[string]any) {
		l2Count++
		bids, _ := data["bids"].([][]any)
		asks, _ := data["asks"].([][]any)
		if len(bids) > 0 && len(asks) > 0 {
			fmt.Printf("[%s] ETH: %d bids, %d asks\n", timestamp(), len(bids), len(asks))
		}

		if l2Count >= 3 {
			fmt.Printf("\nReceived %d L2 updates.\n", l2Count)
		}
	}, hyperliquid.L2BookNLevels(10))

	if err := l2Stream.Start(); err != nil {
		fmt.Printf("Start error: %v\n", err)
		return
	}

	start = time.Now()
	for l2Count < 3 && time.Since(start) < 10*time.Second {
		time.Sleep(100 * time.Millisecond)
	}
	l2Stream.Stop()

	// Example 3: Stream Blocks
	fmt.Println("\nExample 3: Streaming Blocks")
	fmt.Println(string(repeat('-', 60)))

	blockCount := 0

	blockStream := sdk.NewGRPCStream(&hyperliquid.GRPCStreamConfig{
		Secure:    true,
		Reconnect: false,
	})

	blockStream.Blocks(func(data map[string]any) {
		blockCount++
		// Block data contains the full block info from DataJson
		fmt.Printf("[%s] Block received\n", timestamp())

		if blockCount >= 3 {
			fmt.Printf("\nReceived %d blocks.\n", blockCount)
		}
	})

	if err := blockStream.Start(); err != nil {
		fmt.Printf("Start error: %v\n", err)
		return
	}

	start = time.Now()
	for blockCount < 3 && time.Since(start) < 15*time.Second {
		time.Sleep(100 * time.Millisecond)
	}
	blockStream.Stop()

	fmt.Println("\n" + string(repeat('=', 60)))
	fmt.Println("Done!")
}

func timestamp() string {
	return time.Now().Format("15:04:05.000")
}

func repeat(b byte, n int) []byte {
	result := make([]byte, n)
	for i := range result {
		result[i] = b
	}
	return result
}
