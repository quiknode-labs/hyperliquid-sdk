// Cancel All Example — Cancel all open orders globally or for a specific asset.
//
// This example matches the Python cancel_all.py exactly.
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
		fmt.Println("Cancel All Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Cancel All Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Check open orders first
	fmt.Println("Checking open orders...")
	ordersResp, err := sdk.OpenOrders("")
	if err != nil {
		fmt.Printf("Error fetching orders: %v\n", err)
	} else {
		orders, _ := ordersResp["orders"].([]any)
		fmt.Printf("Open orders: %d\n", len(orders))
	}
	fmt.Println()

	// Cancel all orders for a specific asset (BTC)
	fmt.Println("Canceling all BTC orders...")
	result, err := sdk.CancelAll("BTC")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Cancel result: %v\n", result)
	}
	fmt.Println()

	// Cancel all orders globally (uncomment to run)
	// fmt.Println("Canceling ALL orders...")
	// result, err = sdk.CancelAll("")
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Cancel result: %v\n", result)
	// }

	fmt.Println("==================================================")
	fmt.Println("Done!")
}
