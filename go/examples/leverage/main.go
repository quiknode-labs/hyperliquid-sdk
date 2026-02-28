// Leverage Example — Update leverage for positions.
//
// This example matches the Python leverage.py exactly.
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
		fmt.Println("Leverage Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Leverage Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Set cross margin leverage (default)
	fmt.Println("Set Cross Margin (10x):")
	fmt.Println("------------------------------")
	result, err := sdk.UpdateLeverage("BTC", 10)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}
	fmt.Println()

	// Set isolated margin leverage
	fmt.Println("Set Isolated Margin (5x):")
	fmt.Println("------------------------------")
	result, err = sdk.UpdateLeverage("ETH", 5, hyperliquid.LeverageWithIsolated())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}
	fmt.Println()

	fmt.Println("Parameters:")
	fmt.Println("  asset: The trading pair (e.g., \"BTC\", \"ETH\")")
	fmt.Println("  leverage: Numeric value (e.g., 10 for 10x)")
	fmt.Println("  LeverageWithIsolated(): Option to use isolated margin")

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
