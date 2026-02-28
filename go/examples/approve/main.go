// Approve Example — Check and manage builder fee approval status.
//
// This example matches the Python approve.py exactly.
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
		fmt.Println("Approve Example")
		fmt.Println("=" + string(make([]byte, 49)))
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Approve Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Check current approval status
	fmt.Println("Checking approval status...")
	status, err := sdk.ApprovalStatus("")
	if err != nil {
		fmt.Printf("Error checking approval: %v\n", err)
	} else {
		approved, _ := status["approved"].(bool)
		maxFeeRate, _ := status["maxFeeRate"].(string)
		fmt.Printf("Approved: %v\n", approved)
		fmt.Printf("Max Fee Rate: %s\n", maxFeeRate)
	}

	fmt.Println()
	fmt.Println("To approve builder fee (uncomment to run):")
	fmt.Println("  sdk.ApproveBuilderFee(\"1%\")")
	fmt.Println()
	fmt.Println("To revoke approval (uncomment to run):")
	fmt.Println("  sdk.RevokeBuilderFee()")

	// Uncomment to actually approve:
	// result, err := sdk.ApproveBuilderFee("1%")
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Approved: %v\n", result)
	// }

	// Uncomment to revoke:
	// result, err := sdk.RevokeBuilderFee()
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// } else {
	//     fmt.Printf("Revoked: %v\n", result)
	// }

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Done!")
}
