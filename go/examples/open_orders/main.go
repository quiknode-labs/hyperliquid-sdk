// Open Orders Example — Retrieve all open orders for an authenticated user.
//
// This example matches the Python open_orders.py exactly.
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
		fmt.Println("Open Orders Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Open Orders Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Get all open orders (pass empty string to use SDK's address)
	fmt.Println("Fetching open orders...")
	ordersResp, err := sdk.OpenOrders("")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	orders, _ := ordersResp["orders"].([]any)
	fmt.Printf("Open orders: %d\n", len(orders))
	fmt.Println("------------------------------")

	for i, order := range orders {
		if i >= 10 {
			fmt.Printf("  ... and %d more\n", len(orders)-10)
			break
		}
		o := order.(map[string]any)
		oid, _ := o["oid"].(float64)
		coin, _ := o["coin"].(string)
		side, _ := o["side"].(string)
		sz, _ := o["sz"].(string)
		limitPx, _ := o["limitPx"].(string)

		sideStr := "SELL"
		if side == "B" {
			sideStr = "BUY"
		}

		pxFloat, _ := strconv.ParseFloat(limitPx, 64)
		fmt.Printf("  OID: %d | %s %s %s @ $%.2f\n", int(oid), coin, sideStr, sz, pxFloat)
	}

	// Get specific order status (example)
	// if len(orders) > 0 {
	//     oid := int64(orders[0].(map[string]any)["oid"].(float64))
	//     status, err := sdk.OrderStatus(oid, "")
	//     if err != nil {
	//         fmt.Printf("Error: %v\n", err)
	//     } else {
	//         fmt.Printf("Order %d status: %v\n", oid, status)
	//     }
	// }

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
