// Withdraw Example — Withdraw funds from Hyperliquid to external wallets.
//
// This example demonstrates:
// - Initiating withdrawals to Arbitrum
// - Checking withdrawal status
// - Understanding withdrawal fees and limits
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
		fmt.Println("Withdraw Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Withdraw Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Check available balance for withdrawal
	fmt.Println("Available Balance:")
	fmt.Println("------------------------------")
	state, err := sdk.Info().ClearinghouseState(sdk.Address())
	if err != nil {
		fmt.Printf("Error fetching state: %v\n", err)
	} else {
		if marginSummary, ok := state["marginSummary"].(map[string]any); ok {
			if available, ok := marginSummary["withdrawable"].(string); ok {
				fmt.Printf("  Withdrawable: $%s\n", available)
			}
		}
	}
	fmt.Println()

	// Withdraw to Arbitrum
	fmt.Println("1. Withdraw to Arbitrum:")
	fmt.Println("------------------------------")
	fmt.Println("  Withdraw USDC to your Arbitrum wallet")
	fmt.Println()
	fmt.Println("  result, err := sdk.Withdraw(100.0)  // Withdraw $100 USDC")
	fmt.Println()

	// Uncomment to withdraw:
	// result, err := sdk.Withdraw(100.0)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Withdrawal initiated: %v\n", result)
	// }

	// Withdraw to specific address
	fmt.Println("2. Withdraw to Specific Address:")
	fmt.Println("------------------------------")
	fmt.Println("  Send to a different Arbitrum address")
	fmt.Println()
	fmt.Println("  result, err := sdk.WithdrawTo(\"0xRecipient...\", 100.0)")
	fmt.Println()

	// Uncomment to withdraw to specific address:
	// recipient := "0x1234567890abcdef1234567890abcdef12345678"
	// result, err := sdk.WithdrawTo(recipient, 100.0)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Withdrawal initiated: %v\n", result)
	// }

	// Withdraw USDC3 (different decimals)
	fmt.Println("3. Withdraw USDC3:")
	fmt.Println("------------------------------")
	fmt.Println("  USDC3 has different decimal handling")
	fmt.Println()
	fmt.Println("  result, err := sdk.WithdrawUSDC3(50.0)")
	fmt.Println()

	// Check withdrawal history - would use ledger data
	fmt.Println("4. Withdrawal History:")
	fmt.Println("------------------------------")
	fmt.Println("  Use sdk.Info().UserNonFundingLedgerUpdates() to view withdrawal history")
	fmt.Println("  Withdrawals appear as 'withdraw' type events in the ledger")
	fmt.Println()

	fmt.Println("Notes:")
	fmt.Println("  - Withdrawals are sent to Arbitrum chain")
	fmt.Println("  - Minimum withdrawal: $1 USDC")
	fmt.Println("  - Withdrawal fee: ~$1-2 (varies with gas)")
	fmt.Println("  - Processing time: 1-10 minutes typically")
	fmt.Println("  - Cannot withdraw funds in use as margin")

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
