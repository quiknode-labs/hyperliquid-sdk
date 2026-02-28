// Market Order Example — Place market orders that execute immediately.
//
// This example matches the Python market_order.py exactly.
package main

import (
	"fmt"
	"os"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	privateKey := os.Getenv("PRIVATE_KEY")
	endpoint := os.Getenv("QUICKNODE_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("ENDPOINT")
	}

	if privateKey == "" {
		fmt.Println("Market Order Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Market Order Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Market buy with notional (dollar amount)
	fmt.Println("Market Buy (by notional):")
	fmt.Println("------------------------------")
	order, err := sdk.MarketBuy("BTC", hyperliquid.WithNotional(11)) // $11 minimum
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Order placed: OID=%d\n", order.OID)
		fmt.Printf("  Status: %s\n", order.Status)
		fmt.Printf("  Filled Size: %s\n", order.FilledSize)
	}
	fmt.Println()

	// Market sell with size
	fmt.Println("Market Sell (by size):")
	fmt.Println("------------------------------")
	order, err = sdk.MarketSell("BTC", hyperliquid.WithSize(0.0001))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Order placed: OID=%d\n", order.OID)
		fmt.Printf("  Status: %s\n", order.Status)
		fmt.Printf("  Filled Size: %s\n", order.FilledSize)
	}
	fmt.Println()

	fmt.Println("Note: Minimum order value is $10")
	fmt.Println("      Market orders execute at best available price")

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
