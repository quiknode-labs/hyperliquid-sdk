// Preflight Example — Validate order parameters before submitting.
//
// This example matches the Python preflight.py exactly.
package main

import (
	"fmt"
	"os"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	endpoint := os.Getenv("QUICKNODE_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("ENDPOINT")
	}

	fmt.Println("Preflight Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint)
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	// Get current mid price
	fmt.Println()
	fmt.Println("Getting BTC mid price...")
	mid, err := sdk.GetMid("BTC")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("BTC mid: $%.2f\n", mid)
	fmt.Println()

	// Example 1: Valid order parameters
	fmt.Println("Example 1: Valid Order")
	fmt.Println("------------------------------")
	result, err := sdk.Preflight("BTC", "B", int(mid*0.95), 0.001)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		valid, _ := result["valid"].(bool)
		fmt.Printf("Valid: %v\n", valid)
		if !valid {
			if msg, ok := result["message"].(string); ok {
				fmt.Printf("Message: %s\n", msg)
			}
		}
	}
	fmt.Println()

	// Example 2: Invalid price (too many decimals)
	fmt.Println("Example 2: Invalid Price (too many decimals)")
	fmt.Println("------------------------------")
	result, err = sdk.Preflight("BTC", "B", 65432.123456789, 0.001)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		valid, _ := result["valid"].(bool)
		fmt.Printf("Valid: %v\n", valid)
		if !valid {
			if msg, ok := result["message"].(string); ok {
				fmt.Printf("Message: %s\n", msg)
			}
			if suggestion, ok := result["suggestion"].(string); ok {
				fmt.Printf("Suggestion: %s\n", suggestion)
			}
		}
	}
	fmt.Println()

	// Example 3: Invalid size (below minimum)
	fmt.Println("Example 3: Invalid Size")
	fmt.Println("------------------------------")
	result, err = sdk.Preflight("BTC", "B", int(mid*0.95), 0.0000001)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		valid, _ := result["valid"].(bool)
		fmt.Printf("Valid: %v\n", valid)
		if !valid {
			if msg, ok := result["message"].(string); ok {
				fmt.Printf("Message: %s\n", msg)
			}
		}
	}

	fmt.Println()
	fmt.Println("Preflight validation catches errors before signing:")
	fmt.Println("  - Tick size issues")
	fmt.Println("  - Lot size issues")
	fmt.Println("  - Minimum order value")
	fmt.Println("  - Price precision")
	fmt.Println("  - Size precision")

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
