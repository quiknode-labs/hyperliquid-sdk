//! Builder Fee Example — Manage builder fee approval
//!
//! Shows how to approve, check, and revoke builder fee.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example builder_fee
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
        eprintln!("  cargo run --example builder_fee");
        std::process::exit(1);
    }

    println!("Builder Fee Management Example");
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

    // Check current status
    println!("\n1. Current Approval Status:");
    match sdk.approval_status().await {
        Ok(status) => println!("   {:?}", status),
        Err(e) => println!("   Error: {}", e),
    }

    // Approve with custom max fee
    println!("\n2. Approve Builder Fee (0.5% max):");
    match sdk.approve_builder_fee(Some("0.5%")).await {
        Ok(result) => println!("   Approved: {:?}", result),
        Err(e) => println!("   Error: {}", e),
    }

    // Revoke approval (commented to avoid breaking things)
    // println!("\n3. Revoke Builder Fee:");
    // match sdk.revoke_builder_fee().await {
    //     Ok(result) => println!("   Revoked: {:?}", result),
    //     Err(e) => println!("   Error: {}", e),
    // }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
