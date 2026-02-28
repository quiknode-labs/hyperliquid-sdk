// Close Position Example — Close an open position completely.
//
// This example matches the Python close_position.py exactly.
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
		fmt.Println("Close Position Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Close Position Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Close BTC position
	// The SDK automatically:
	// 1. Queries your current BTC position
	// 2. Determines size and direction
	// 3. Places a counter-order to close
	fmt.Println("Closing BTC position...")
	result, err := sdk.ClosePosition("BTC")
	if err != nil {
		// Common error: no position exists
		fmt.Printf("Error: %v\n", err)
		fmt.Println()
		fmt.Println("Note: This error is expected if you have no BTC position.")
	} else {
		fmt.Printf("Position closed: %v\n", result)
		fmt.Printf("  OID: %d\n", result.OID)
		fmt.Printf("  Status: %s\n", result.Status)
		fmt.Printf("  Filled Size: %s\n", result.FilledSize)
	}

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
