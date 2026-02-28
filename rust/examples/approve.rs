//! Approve Example — Approve builder fee for trading
//!
//! Approves the builder fee to enable trading through QuickNode endpoints.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example approve
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
        eprintln!("  cargo run --example approve");
        std::process::exit(1);
    }

    println!("Builder Fee Approval Example");
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

    // Check current approval status
    println!("\n1. Checking approval status...");
    match sdk.approval_status().await {
        Ok(status) => {
            println!("   Status: {:?}", status);
        }
        Err(e) => {
            println!("   Error: {}", e);
        }
    }

    // Approve builder fee (1% max)
    println!("\n2. Approving builder fee (1% max)...");
    match sdk.approve_builder_fee(Some("1%")).await {
        Ok(result) => {
            println!("   Result: {:?}", result);
        }
        Err(e) => {
            println!("   Error: {}", e);
        }
    }

    // Verify approval
    println!("\n3. Verifying approval...");
    match sdk.approval_status().await {
        Ok(status) => {
            println!("   Status: {:?}", status);
        }
        Err(e) => {
            println!("   Error: {}", e);
        }
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
