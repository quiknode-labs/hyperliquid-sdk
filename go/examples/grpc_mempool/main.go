// Example: stream pre-consensus mempool transactions over gRPC with a coin filter.
//
// Usage:
//
//	ENDPOINT="https://your-endpoint.hyperliquid-mainnet.quiknode.pro/TOKEN" go run ./examples/grpc_mempool
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("ENDPOINT")
	if endpoint == "" {
		log.Fatal("set ENDPOINT to your Quicknode Hyperliquid endpoint URL")
	}

	stream := hyperliquid.NewGRPCStream(endpoint, &hyperliquid.GRPCStreamConfig{
		Reconnect: true,
		OnError:   func(err error) { log.Printf("stream error: %v", err) },
		OnConnect: func() { log.Println("connected") },
	})

	// Mempool transactions filtered to BTC and ETH.
	// Pass nil coins for all transactions (unfiltered).
	stream.MempoolTxs([]string{"BTC", "ETH"}, func(tx map[string]any) {
		fmt.Printf("mempool tx: %v\n", tx)
	})

	// Derived order/write priority actions. Events carry server-enriched
	// fields: coin, market_type, sz_decimals.
	stream.OrderPriority(func(action map[string]any) {
		fmt.Printf("order priority: %v\n", action)
	})

	if err := stream.Start(); err != nil {
		log.Fatal(err)
	}
	defer stream.Stop()

	// Wait for Ctrl-C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
