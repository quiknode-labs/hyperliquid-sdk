// Trigger Orders Example — Stop loss, take profit, and other trigger orders.
//
// This example demonstrates:
// - Stop loss orders (sell when price drops below trigger)
// - Take profit orders (sell when price rises above trigger)
// - Trailing stop orders
// - OCO (One-Cancels-Other) order combinations
//
// This example matches the Python SDK patterns exactly.
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
		fmt.Println("Trigger Orders Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Trigger Orders Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Stop Loss Order
	fmt.Println("1. Stop Loss Order:")
	fmt.Println("------------------------------")
	fmt.Println("  Sells when price drops below trigger price")
	fmt.Println()
	fmt.Println("  // Using TriggerOrder builder:")
	fmt.Println("  order := hyperliquid.TriggerOrder().")
	fmt.Println("      StopLoss(\"BTC\").")
	fmt.Println("      Size(0.01).")
	fmt.Println("      TriggerPrice(58000)")
	fmt.Println()
	fmt.Println("  result, err := sdk.PlaceTriggerOrder(order)")
	fmt.Println()

	// Uncomment to place stop loss:
	// order := hyperliquid.TriggerOrder().StopLoss("BTC").Size(0.01).TriggerPrice(58000)
	// result, err := sdk.PlaceTriggerOrder(order)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Stop loss placed: %v\n", result)
	// }

	// Take Profit Order
	fmt.Println("2. Take Profit Order:")
	fmt.Println("------------------------------")
	fmt.Println("  Sells when price rises above trigger price")
	fmt.Println()
	fmt.Println("  order := hyperliquid.TriggerOrder().")
	fmt.Println("      TakeProfit(\"BTC\").")
	fmt.Println("      Size(0.01).")
	fmt.Println("      TriggerPrice(72000)")
	fmt.Println()
	fmt.Println("  result, err := sdk.PlaceTriggerOrder(order)")
	fmt.Println()

	// Uncomment to place take profit:
	// order := hyperliquid.TriggerOrder().TakeProfit("BTC").Size(0.01).TriggerPrice(72000)
	// result, err := sdk.PlaceTriggerOrder(order)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Take profit placed: %v\n", result)
	// }

	// Stop Limit Order
	fmt.Println("3. Stop Limit Order:")
	fmt.Println("------------------------------")
	fmt.Println("  Creates limit order when trigger price is hit")
	fmt.Println()
	fmt.Println("  order := hyperliquid.TriggerOrder().")
	fmt.Println("      StopLimit(\"BTC\").")
	fmt.Println("      Size(0.01).")
	fmt.Println("      TriggerPrice(58000).")
	fmt.Println("      LimitPrice(57900)")
	fmt.Println()

	// Trailing Stop Order
	fmt.Println("4. Trailing Stop Order:")
	fmt.Println("------------------------------")
	fmt.Println("  Stop that trails the price by a percentage")
	fmt.Println()
	fmt.Println("  order := hyperliquid.TriggerOrder().")
	fmt.Println("      TrailingStop(\"BTC\").")
	fmt.Println("      Size(0.01).")
	fmt.Println("      TrailPercent(5.0)  // 5% trail")
	fmt.Println()

	// Reduce Only Trigger
	fmt.Println("5. Reduce Only Trigger:")
	fmt.Println("------------------------------")
	fmt.Println("  Trigger order that only reduces position")
	fmt.Println()
	fmt.Println("  order := hyperliquid.TriggerOrder().")
	fmt.Println("      StopLoss(\"BTC\").")
	fmt.Println("      Size(0.01).")
	fmt.Println("      TriggerPrice(58000).")
	fmt.Println("      ReduceOnly()")
	fmt.Println()

	// Check open trigger orders
	fmt.Println("Checking Open Trigger Orders:")
	fmt.Println("------------------------------")
	ordersResp, err := sdk.OpenOrders("")
	if err != nil {
		fmt.Printf("Error fetching orders: %v\n", err)
	} else {
		orders, _ := ordersResp["orders"].([]any)
		triggerCount := 0
		for _, order := range orders {
			if o, ok := order.(map[string]any); ok {
				if orderType, _ := o["orderType"].(string); orderType == "trigger" {
					triggerCount++
					coin, _ := o["coin"].(string)
					side, _ := o["side"].(string)
					triggerPx, _ := o["triggerPx"].(string)
					fmt.Printf("  %s %s trigger @ $%s\n", coin, side, triggerPx)
				}
			}
		}
		if triggerCount == 0 {
			fmt.Println("  No open trigger orders")
		}
	}
	fmt.Println()

	fmt.Println("Notes:")
	fmt.Println("  - Stop loss triggers when price <= trigger price")
	fmt.Println("  - Take profit triggers when price >= trigger price")
	fmt.Println("  - Use reduce_only to prevent opening new positions")
	fmt.Println("  - Trigger orders remain active until filled or cancelled")

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
