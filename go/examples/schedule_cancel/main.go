// Schedule Cancel Example — Dead-man's switch for automatic order cancellation.
//
// This example matches the Python schedule_cancel.py exactly.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"
)

func main() {
	privateKey := os.Getenv("PRIVATE_KEY")
	endpoint := os.Getenv("QUICKNODE_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("ENDPOINT")
	}

	if privateKey == "" {
		fmt.Println("Schedule Cancel Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Schedule Cancel Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	fmt.Println("IMPORTANT: Schedule cancel requires $1M+ trading volume!")
	fmt.Println()

	// Calculate cancel time (1 minute from now)
	cancelTime := time.Now().Add(60 * time.Second).UnixMilli()

	fmt.Println("Schedule Cancel (Dead-Man's Switch):")
	fmt.Println("------------------------------")
	fmt.Printf("Would schedule cancellation at: %d (60 seconds from now)\n", cancelTime)
	fmt.Println()

	// Uncomment to actually schedule (requires $1M volume):
	// result, err := sdk.ScheduleCancel(cancelTime)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Scheduled: %v\n", result)
	// }

	fmt.Println("To schedule cancellation:")
	fmt.Println("  sdk.ScheduleCancel(cancelTimeMs)")
	fmt.Println()

	fmt.Println("To cancel the scheduled cancellation:")
	fmt.Println("  sdk.ScheduleCancel(0)  // or pass nil")
	fmt.Println()

	// Uncomment to cancel scheduled cancellation:
	// result, err := sdk.ScheduleCancel(0)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Cancelled schedule: %v\n", result)
	// }

	fmt.Println("How it works:")
	fmt.Println("  1. Schedule a cancellation time")
	fmt.Println("  2. All orders auto-cancel at that time")
	fmt.Println("  3. Keep renewing before expiry to stay alive")
	fmt.Println("  4. If you go offline, orders are safely cancelled")

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
