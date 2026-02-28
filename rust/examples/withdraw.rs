//! Withdraw Example
//!
//! Withdraw funds from Hyperliquid to Arbitrum.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example withdraw
//! ```

use hyperliquid_sdk::HyperliquidSDK;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();
    let private_key = std::env::var("PRIVATE_KEY").ok();

    if endpoint.is_none() || private_key.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        eprintln!("  export PRIVATE_KEY='0x...'");
        eprintln!("  cargo run --example withdraw");
        std::process::exit(1);
    }

    println!("Withdraw Example");
    println!("{}", "=".repeat(50));

    let mut builder = HyperliquidSDK::new();
    if let Some(ep) = &endpoint {
        builder = builder.endpoint(ep);
    }
    if let Some(pk) = &private_key {
        builder = builder.private_key(pk);
    }
    let sdk = builder.build().await?;

    if let Some(addr) = sdk.address() {
        println!("Address: {}", addr);
    }

    // Check withdrawable balance
    println!("\n1. Withdrawable Balance:");
    let info = sdk.info();
    let address_str = sdk.address().map(|a| format!("{:?}", a)).unwrap_or_default();

    match info.clearinghouse_state(&address_str, None).await {
        Ok(state) => {
            let withdrawable = state.get("withdrawable").and_then(|v| v.as_str()).unwrap_or("0");
            println!("   Withdrawable: ${}", withdrawable);
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Withdraw to wallet address on Arbitrum
    println!("\n2. Withdraw to Arbitrum:");
    println!("   Amount: $100");
    println!("   Destination: Your wallet address");

    // Uncomment to execute withdrawal
    // match sdk.withdraw(100.0, None).await {
    //     Ok(result) => println!("   Result: {:?}", result),
    //     Err(e) => println!("   Error: {}", e),
    // }

    // Withdraw to specific address
    println!("\n3. Withdraw to Specific Address:");
    println!("   sdk.withdraw(amount, Some(\"0xRecipientAddress\"))");
    // match sdk.withdraw(100.0, Some("0xRecipientAddress")).await {
    //     Ok(result) => println!("   Result: {:?}", result),
    //     Err(e) => println!("   Error: {}", e),
    // }

    println!("\n{}", "-".repeat(50));
    println!("Withdrawal Notes:");
    println!("  - Minimum withdrawal: $1 USDC");
    println!("  - Bridge fee applies (~$0.5-2)");
    println!("  - Cannot withdraw margin in use");
    println!("  - Takes 5-20 minutes to arrive on Arbitrum");

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
