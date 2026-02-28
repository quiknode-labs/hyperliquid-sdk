// Staking Example — HYPE token staking, unstaking, and delegation.
//
// This example matches the Python staking.py exactly.
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
		fmt.Println("Staking Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Staking Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Staking HYPE tokens
	fmt.Println("Stake HYPE Tokens:")
	fmt.Println("------------------------------")
	fmt.Println("  sdk.Stake(1000)  // Stake 1000 HYPE")
	fmt.Println()

	// Uncomment to stake:
	// result, err := sdk.Stake(1000)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Staked: %v\n", result)
	// }

	// Unstaking HYPE tokens
	fmt.Println("Unstake HYPE Tokens:")
	fmt.Println("------------------------------")
	fmt.Println("  sdk.Unstake(500)  // Unstake 500 HYPE (7-day queue)")
	fmt.Println()

	// Uncomment to unstake:
	// result, err := sdk.Unstake(500)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Unstaked: %v\n", result)
	// }

	// Delegating to a validator
	fmt.Println("Delegate to Validator:")
	fmt.Println("------------------------------")
	fmt.Println("  sdk.Delegate(\"0xValidatorAddress...\", 500)")
	fmt.Println()

	// Uncomment to delegate:
	// validator := "0x1234..."
	// result, err := sdk.Delegate(validator, 500)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Delegated: %v\n", result)
	// }

	// Undelegating from a validator
	fmt.Println("Undelegate from Validator:")
	fmt.Println("------------------------------")
	fmt.Println("  sdk.Undelegate(\"0xValidatorAddress...\", 250)")
	fmt.Println()

	// Uncomment to undelegate:
	// result, err := sdk.Undelegate(validator, 250)
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Undelegated: %v\n", result)
	// }

	fmt.Println("Notes:")
	fmt.Println("  - Unstaking has a 7-day waiting period")
	fmt.Println("  - Delegation earns staking rewards")
	fmt.Println("  - Check validator addresses before delegating")

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
