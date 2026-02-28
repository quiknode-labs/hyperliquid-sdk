// Trading Example - Place orders on Hyperliquid
//
// This example matches the Python trading_example.py exactly.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("QUICKNODE_ENDPOINT")
	}
	privateKey := os.Getenv("PRIVATE_KEY")

	if endpoint == "" || privateKey == "" {
		fmt.Println("Hyperliquid Trading Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Hyperliquid Trading Example")
	fmt.Println("==================================================")

	// Create SDK with private key
	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		log.Fatalf("Failed to create SDK: %v", err)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Printf("Endpoint: %s...\n", truncate(endpoint, 50))
	fmt.Println()

	// ==========================================================================
	// Example 1: Market Buy $100 of BTC
	// ==========================================================================
	fmt.Println("Example 1: Market Buy")
	fmt.Println("------------------------------")

	order, err := sdk.MarketBuy("BTC", hyperliquid.WithNotional(100))
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("Order placed: %v\n", order)
		fmt.Printf("  Order ID: %d\n", order.OID)
		fmt.Printf("  Status: %s\n", order.Status)
		fmt.Printf("  Filled: %s @ avg %s\n", order.FilledSize, order.AvgPrice)
	}
	fmt.Println()

	// ==========================================================================
	// Example 2: Limit Order
	// ==========================================================================
	fmt.Println("Example 2: Limit Order")
	fmt.Println("------------------------------")

	// Build and place limit order using Order builder (matches Python Order class)
	limitOrder := hyperliquid.Order().
		Buy("ETH").
		Size(0.1).
		Price(2000.0).
		GTC()

	result, err := sdk.PlaceOrder(limitOrder)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("Order placed: %v\n", result)
		fmt.Printf("  Order ID: %d\n", result.OID)
		fmt.Printf("  Status: %s\n", result.Status)
	}
	fmt.Println()

	// ==========================================================================
	// Example 3: Stop Loss Order
	// ==========================================================================
	fmt.Println("Example 3: Stop Loss Order")
	fmt.Println("------------------------------")

	// Build stop loss using Order builder with trigger price (matches Python Order class)
	stopOrder := hyperliquid.TriggerOrder().
		StopLoss("BTC").
		Size(0.01).
		TriggerPrice(60000.0). // Stop triggers at this price
		Limit(59900.0).        // Then executes as limit at this price
		ReduceOnly()           // Only reduce existing position

	slResult, err := sdk.PlaceTriggerOrder(stopOrder, hyperliquid.OrderGroupingNA)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("Stop loss placed: %v\n", slResult)
	}
	fmt.Println()

	// ==========================================================================
	// Example 4: Cancel Orders
	// ==========================================================================
	fmt.Println("Example 4: Cancel All Orders")
	fmt.Println("------------------------------")

	// Cancel all BTC orders
	_, err = sdk.CancelAll("BTC")
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Println("Cancelled all BTC orders")
	}
	fmt.Println()

	// ==========================================================================
	// Example 5: Close Position
	// ==========================================================================
	fmt.Println("Example 5: Close Position")
	fmt.Println("------------------------------")

	// Market close BTC position
	closeResult, err := sdk.ClosePosition("BTC")
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("Position closed: %v\n", closeResult)
	}

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
