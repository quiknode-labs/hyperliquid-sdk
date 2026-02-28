//! Transfers Example
//!
//! Transfer funds between accounts.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example transfers
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
        eprintln!("  cargo run --example transfers");
        std::process::exit(1);
    }

    println!("Transfers Example");
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

    // Check balance
    println!("\n1. Current Balances:");
    let info = sdk.info();
    let address_str = sdk.address().map(|a| format!("{:?}", a)).unwrap_or_default();

    match info.clearinghouse_state(&address_str, None).await {
        Ok(state) => {
            if let Some(margin) = state.get("marginSummary") {
                let value = margin.get("accountValue").and_then(|v| v.as_str()).unwrap_or("0");
                let withdrawable = state.get("withdrawable").and_then(|v| v.as_str()).unwrap_or("0");
                println!("   Account Value: ${}", value);
                println!("   Withdrawable: ${}", withdrawable);
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Transfer USD to another address
    println!("\n2. Transfer USD:");
    println!("   sdk.transfer_usd(destination, amount)");
    // match sdk.transfer_usd("0xRecipientAddress", 100.0).await {
    //     Ok(result) => println!("   Result: {:?}", result),
    //     Err(e) => println!("   Error: {}", e),
    // }

    // Spot to Perp transfer
    println!("\n3. Spot to Perp Transfer:");
    match sdk.transfer_spot_to_perp(10.0).await {
        Ok(result) => println!("   Result: {:?}", result),
        Err(e) => println!("   Error (may need spot balance): {}", e),
    }

    // Perp to Spot transfer
    println!("\n4. Perp to Spot Transfer:");
    match sdk.transfer_perp_to_spot(10.0).await {
        Ok(result) => println!("   Result: {:?}", result),
        Err(e) => println!("   Error (may need free margin): {}", e),
    }

    // Transfer spot tokens
    println!("\n5. Transfer Spot Tokens:");
    println!("   sdk.transfer_spot(destination, token, amount, to_perp)");
    // match sdk.transfer_spot("0xRecipient", "USDC", 100.0, false).await {
    //     Ok(result) => println!("   Result: {:?}", result),
    //     Err(e) => println!("   Error: {}", e),
    // }

    println!("\n{}", "-".repeat(50));
    println!("Transfer Methods:");
    println!("  - transfer_usd: Send USDC to another address");
    println!("  - transfer_spot_to_perp: Move from spot to perp");
    println!("  - transfer_perp_to_spot: Move from perp to spot");
    println!("  - transfer_spot: Transfer spot tokens");

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
