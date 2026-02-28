//! Close Position Example
//!
//! Close an open position with a market order.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example close_position
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
        eprintln!("  cargo run --example close_position");
        std::process::exit(1);
    }

    println!("Close Position Example");
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
                            let entry = position.get("entryPx").and_then(|v| v.as_str()).unwrap_or("?");
                            let pnl = position.get("unrealizedPnl").and_then(|v| v.as_str()).unwrap_or("?");
                            println!("   {} size={} entry={} pnl={}", coin, szi, entry, pnl);
                        }
                    }
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Close BTC position if any
    println!("\n2. Close BTC Position:");
    match sdk.close_position("BTC").await {
        Ok(order) => {
            println!("   Status: {}", order.status);
            if let Some(oid) = order.oid {
                println!("   Order ID: {}", oid);
            }
        }
        Err(e) => println!("   No position or error: {}", e),
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
