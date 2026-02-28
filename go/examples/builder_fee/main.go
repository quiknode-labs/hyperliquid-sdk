// Builder Fee Example — Builder fee authorization workflow.
//
// This example matches the Python builder_fee.py exactly.
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
		fmt.Println("Builder Fee Example")
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  export PRIVATE_KEY='0xYourPrivateKey'")
		fmt.Println("  export QUICKNODE_ENDPOINT='https://YOUR-ENDPOINT.quiknode.pro/TOKEN'")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Println("Builder Fee Example")
	fmt.Println("==================================================")

	sdk, err := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
	if err != nil {
		fmt.Printf("Failed to create SDK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address: %s\n", sdk.Address())
	fmt.Println()

	// Step 1: Check current approval status
	fmt.Println("Step 1: Check Approval Status")
	fmt.Println("------------------------------")
	status, err := sdk.ApprovalStatus("")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		approved, _ := status["approved"].(bool)
		maxFeeRate, _ := status["maxFeeRate"].(string)
		fmt.Printf("Approved: %v\n", approved)
		fmt.Printf("Max Fee Rate: %s\n", maxFeeRate)
	}
	fmt.Println()

	// Step 2: Approve builder fee (if not already approved)
	fmt.Println("Step 2: Approve Builder Fee")
	fmt.Println("------------------------------")
	fmt.Println("To approve with 1% max fee:")
	fmt.Println("  result, err := sdk.ApproveBuilderFee(\"1%\")")
	fmt.Println()

	// Uncomment to actually approve:
	// if !status.Approved {
	//     result, err := sdk.ApproveBuilderFee("1%")
	//     if err != nil {
	//         fmt.Printf("Error approving: %v\n", err)
	//     } else {
	//         fmt.Printf("Approval result: %v\n", result)
	//     }
	// }

	// Step 3: Revoke builder fee
	fmt.Println("Step 3: Revoke Builder Fee")
	fmt.Println("------------------------------")
	fmt.Println("To revoke approval:")
	fmt.Println("  result, err := sdk.RevokeBuilderFee()")
	fmt.Println()

	// Uncomment to actually revoke:
	// result, err := sdk.RevokeBuilderFee()
	// if err != nil {
	//     fmt.Printf("Error revoking: %v\n", err)
	// } else {
	//     fmt.Printf("Revoke result: %v\n", result)
	// }

	fmt.Println("==================================================")
	fmt.Println("Done!")
}
