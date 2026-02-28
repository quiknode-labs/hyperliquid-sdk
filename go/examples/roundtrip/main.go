// Roundtrip Example — Complete buy-and-sell trading cycle.
//
// This example matches the Python roundtrip.py exactly.
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	privateKey := os.Getenv("PRIVATE_KEY")
	endpoint := os.Getenv("QUICKNODE_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("ENDPOINT")
	}

	if privateKey == "" {
		fmt.Println("Roundtrip Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Roundtrip Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Step 1: Market buy
	fmt.Println("Step 1: Market Buy")
	fmt.Println("------------------------------")
	buyOrder, err := sdk.MarketBuy("BTC", hyperliquid.WithNotional(11))
	if err != nil {
		fmt.Printf("Error buying: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Buy order: OID=%d, Status=%s\n", buyOrder.OID, buyOrder.Status)
	fmt.Printf("  Filled Size: %s\n", buyOrder.FilledSize)
	fmt.Println()

	// Get the size to sell (use filled size or original size)
	sizeToSell := buyOrder.FilledSize
	if sizeToSell == "" || sizeToSell == "0" {
		sizeToSell = buyOrder.Size
	}
	size, _ := strconv.ParseFloat(sizeToSell, 64)

	// Step 2: Market sell to close
	fmt.Println("Step 2: Market Sell")
	fmt.Println("------------------------------")
	sellOrder, err := sdk.MarketSell("BTC", hyperliquid.WithSize(size))
	if err != nil {
		fmt.Printf("Error selling: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Sell order: OID=%d, Status=%s\n", sellOrder.OID, sellOrder.Status)
	fmt.Printf("  Filled Size: %s\n", sellOrder.FilledSize)
	fmt.Println()

	fmt.Println("Roundtrip complete!")
	fmt.Println("  - Bought BTC at market")
	fmt.Println("  - Sold BTC at market")

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
