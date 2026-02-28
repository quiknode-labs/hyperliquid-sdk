//! Isolated Margin Example
//!
//! Manage isolated margin for positions.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example isolated_margin
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
        eprintln!("  cargo run --example isolated_margin");
        std::process::exit(1);
    }

    println!("Isolated Margin Example");
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

    // Check current positions
    println!("\n1. Current Positions:");
    let info = sdk.info();
    let address_str = sdk.address().map(|a| format!("{:?}", a)).unwrap_or_default();

    match info.clearinghouse_state(&address_str, None).await {
        Ok(state) => {
            if let Some(positions) = state.get("assetPositions").and_then(|p| p.as_array()) {
                if positions.is_empty() {
                    println!("   No open positions");
                } else {
                    for pos in positions {
                        if let Some(position) = pos.get("position") {
                            let coin = position.get("coin").and_then(|v| v.as_str()).unwrap_or("?");
                            let szi = position.get("szi").and_then(|v| v.as_str()).unwrap_or("0");
                            let margin = position.get("marginUsed").and_then(|v| v.as_str()).unwrap_or("?");
                            println!("   {} size={} margin={}", coin, szi, margin);
                        }
                    }
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Update isolated margin (add $100 to BTC long position)
    println!("\n2. Add $100 Isolated Margin to BTC:");
    match sdk.update_isolated_margin("BTC", true, 100.0).await {
        Ok(result) => println!("   Result: {:?}", result),
        Err(e) => println!("   Error (may need open position): {}", e),
    }

    // Remove isolated margin
    println!("\n3. Remove $50 Isolated Margin from BTC:");
    match sdk.update_isolated_margin("BTC", true, -50.0).await {
        Ok(result) => println!("   Result: {:?}", result),
        Err(e) => println!("   Error: {}", e),
    }

    // Top up isolated-only margin
    println!("\n4. Top Up Isolated-Only Margin:");
    match sdk.top_up_isolated_only_margin("BTC", 10.0).await {
        Ok(result) => println!("   Result: {:?}", result),
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "-".repeat(50));
    println!("Isolated Margin Notes:");
    println!("  - Isolated margin limits losses to allocated margin");
    println!("  - Add margin to reduce liquidation risk");
    println!("  - Remove margin to free up capital");

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
