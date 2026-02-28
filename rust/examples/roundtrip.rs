//! Roundtrip Example
//!
//! Complete trading flow: buy, monitor, sell.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example roundtrip
//! ```

use hyperliquid_sdk::{HyperliquidSDK, TriggerOrder};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();
    let private_key = std::env::var("PRIVATE_KEY").ok();

    if endpoint.is_none() || private_key.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        eprintln!("  export PRIVATE_KEY='0x...'");
        eprintln!("  cargo run --example roundtrip");
        std::process::exit(1);
    }

    println!("Trading Roundtrip Example");
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

    // Step 1: Get market data
    println!("\n{}", "─".repeat(50));
    println!("STEP 1: Market Data");
    println!("{}", "─".repeat(50));

    let mid = sdk.get_mid("BTC").await?;
    println!("BTC mid price: ${:.2}", mid);

    // Step 2: Check account state
    println!("\n{}", "─".repeat(50));
    println!("STEP 2: Account State");
    println!("{}", "─".repeat(50));

    let info = sdk.info();
    let address_str = sdk.address().map(|a| format!("{:?}", a)).unwrap_or_default();

    match info.clearinghouse_state(&address_str, None).await {
        Ok(state) => {
            if let Some(margin) = state.get("marginSummary") {
                let value = margin.get("accountValue").and_then(|v| v.as_str()).unwrap_or("0");
                println!("Account Value: ${}", value);
            }
        }
        Err(e) => println!("Error: {}", e),
    }

    // Step 3: Place entry order
    println!("\n{}", "─".repeat(50));
    println!("STEP 3: Entry Order");
    println!("{}", "─".repeat(50));

    println!("Market buy $100 of BTC:");
    match sdk.market_buy("BTC").await.notional(100.0).await {
        Ok(order) => {
            println!("   Status: {}", order.status);
            println!("   OID: {:?}", order.oid);
            println!("   Filled: {:?} @ ${:?}", order.filled_size, order.avg_price);
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Step 4: Set stop loss
    println!("\n{}", "─".repeat(50));
    println!("STEP 4: Stop Loss");
    println!("{}", "─".repeat(50));

    let stop_loss = TriggerOrder::stop_loss("BTC")
        .size(0.001)
        .trigger_price(mid * 0.95);

    match sdk.trigger_order(stop_loss).await {
        Ok(order) => {
            println!("Stop loss set at ${:.2}", mid * 0.95);
            println!("   OID: {:?}", order.oid);
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Step 5: Set take profit
    println!("\n{}", "─".repeat(50));
    println!("STEP 5: Take Profit");
    println!("{}", "─".repeat(50));

    let take_profit = TriggerOrder::take_profit("BTC")
        .size(0.001)
        .trigger_price(mid * 1.05);

    match sdk.trigger_order(take_profit).await {
        Ok(order) => {
            println!("Take profit set at ${:.2}", mid * 1.05);
            println!("   OID: {:?}", order.oid);
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Step 6: Close position
    println!("\n{}", "─".repeat(50));
    println!("STEP 6: Close Position");
    println!("{}", "─".repeat(50));

    match sdk.close_position("BTC").await {
        Ok(order) => {
            println!("Position closed");
            println!("   Status: {}", order.status);
        }
        Err(e) => println!("   Error (may have no position): {}", e),
    }

    // Clean up
    println!("\n{}", "─".repeat(50));
    println!("CLEANUP");
    println!("{}", "─".repeat(50));

    match sdk.cancel_all(Some("BTC")).await {
        Ok(_) => println!("Cancelled all BTC orders"),
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
