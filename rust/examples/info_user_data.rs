//! Info API User Data Example
//!
//! Fetch user-specific data: balances, positions, orders, fills.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example info_user_data
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
        eprintln!("  cargo run --example info_user_data");
        std::process::exit(1);
    }

    println!("Info API User Data Example");
    println!("{}", "=".repeat(50));

    let mut builder = HyperliquidSDK::new();
    if let Some(ep) = &endpoint {
        builder = builder.endpoint(ep);
    }
    if let Some(pk) = &private_key {
        builder = builder.private_key(pk);
    }
    let sdk = builder.build().await?;

    let address_str = sdk.address().map(|a| format!("{:?}", a)).unwrap_or_default();
    println!("Address: {}", address_str);

    let info = sdk.info();

    // Account state (balances)
    println!("\n1. Account State:");
    match info.clearinghouse_state(&address_str, None).await {
        Ok(state) => {
            if let Some(margin) = state.get("marginSummary") {
                let value = margin.get("accountValue").and_then(|v| v.as_str()).unwrap_or("0");
                let margin_used = margin.get("totalMarginUsed").and_then(|v| v.as_str()).unwrap_or("0");
                println!("   Account Value: ${}", value);
                println!("   Margin Used: ${}", margin_used);
            }
            if let Some(withdrawable) = state.get("withdrawable").and_then(|v| v.as_str()) {
                println!("   Withdrawable: ${}", withdrawable);
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Positions
    println!("\n2. Open Positions:");
    match info.clearinghouse_state(&address_str, None).await {
        Ok(state) => {
            if let Some(positions) = state.get("assetPositions").and_then(|v| v.as_array()) {
                if positions.is_empty() {
                    println!("   No open positions");
                } else {
                    for pos in positions.iter().take(5) {
                        if let Some(position) = pos.get("position") {
                            let coin = position.get("coin").and_then(|v| v.as_str()).unwrap_or("?");
                            let szi = position.get("szi").and_then(|v| v.as_str()).unwrap_or("0");
                            let entry = position.get("entryPx").and_then(|v| v.as_str()).unwrap_or("?");
                            let pnl = position.get("unrealizedPnl").and_then(|v| v.as_str()).unwrap_or("?");
                            println!("   {}: size={} entry={} pnl={}", coin, szi, entry, pnl);
                        }
                    }
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Open orders
    println!("\n3. Open Orders:");
    match info.open_orders(&address_str, None).await {
        Ok(orders) => {
            if let Some(arr) = orders.as_array() {
                if arr.is_empty() {
                    println!("   No open orders");
                } else {
                    for order in arr.iter().take(5) {
                        let coin = order.get("coin").and_then(|v| v.as_str()).unwrap_or("?");
                        let side = order.get("side").and_then(|v| v.as_str()).unwrap_or("?");
                        let side_str = if side == "B" { "BUY" } else { "SELL" };
                        let sz = order.get("sz").and_then(|v| v.as_str()).unwrap_or("?");
                        let px = order.get("limitPx").and_then(|v| v.as_str()).unwrap_or("?");
                        println!("   {} {} {} @ {}", coin, side_str, sz, px);
                    }
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // User fills
    println!("\n4. Recent Fills:");
    match info.user_fills(&address_str, false).await {
        Ok(fills) => {
            if let Some(arr) = fills.as_array() {
                if arr.is_empty() {
                    println!("   No recent fills");
                } else {
                    for fill in arr.iter().take(5) {
                        let coin = fill.get("coin").and_then(|v| v.as_str()).unwrap_or("?");
                        let side = fill.get("side").and_then(|v| v.as_str()).unwrap_or("?");
                        let side_str = if side == "B" { "BUY" } else { "SELL" };
                        let sz = fill.get("sz").and_then(|v| v.as_str()).unwrap_or("?");
                        let px = fill.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                        println!("   {} {} {} @ {}", coin, side_str, sz, px);
                    }
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    // User fees
    println!("\n5. Fee Structure:");
    match info.user_fees(&address_str).await {
        Ok(fees) => {
            let maker = fees.get("makerRate").and_then(|v| v.as_str()).unwrap_or("?");
            let taker = fees.get("takerRate").and_then(|v| v.as_str()).unwrap_or("?");
            println!("   Maker: {}", maker);
            println!("   Taker: {}", taker);
        }
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
