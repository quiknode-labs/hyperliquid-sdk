//! Trigger Orders Example
//!
//! Place stop-loss and take-profit orders.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example trigger_orders
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
        eprintln!("  cargo run --example trigger_orders");
        std::process::exit(1);
    }

    println!("Trigger Orders Example");
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

    // Get current price
    let mid = sdk.get_mid("BTC").await?;
    println!("\nBTC mid price: ${:.2}", mid);

    // Stop Loss - triggers sell when price drops below trigger
    println!("\n1. Stop Loss Order:");
    let stop_loss = TriggerOrder::stop_loss("BTC")
        .size(0.001)
        .trigger_price(mid * 0.95);  // 5% below current price

    match sdk.trigger_order(stop_loss).await {
        Ok(order) => {
            println!("   Status: {}", order.status);
            println!("   OID: {:?}", order.oid);
            println!("   Trigger: ${:.2}", mid * 0.95);
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Take Profit - triggers sell when price rises above trigger
    println!("\n2. Take Profit Order:");
    let take_profit = TriggerOrder::take_profit("BTC")
        .size(0.001)
        .trigger_price(mid * 1.10);  // 10% above current price

    match sdk.trigger_order(take_profit).await {
        Ok(order) => {
            println!("   Status: {}", order.status);
            println!("   OID: {:?}", order.oid);
            println!("   Trigger: ${:.2}", mid * 1.10);
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Stop Loss with limit price
    println!("\n3. Stop Loss with Limit:");
    let stop_loss_limit = TriggerOrder::stop_loss("BTC")
        .size(0.001)
        .trigger_price(mid * 0.95)
        .limit(mid * 0.94);  // Limit price below trigger

    match sdk.trigger_order(stop_loss_limit).await {
        Ok(order) => {
            println!("   Status: {}", order.status);
            println!("   OID: {:?}", order.oid);
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Using sdk convenience methods
    println!("\n4. Stop Loss (convenience method):");
    match sdk.stop_loss("BTC", 0.001, mid * 0.93).await {
        Ok(order) => {
            println!("   Status: {}", order.status);
            println!("   OID: {:?}", order.oid);
        }
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n5. Take Profit (convenience method):");
    match sdk.take_profit("BTC", 0.001, mid * 1.15).await {
        Ok(order) => {
            println!("   Status: {}", order.status);
            println!("   OID: {:?}", order.oid);
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Clean up - cancel all trigger orders
    println!("\n6. Cleaning up orders...");
    match sdk.cancel_all(Some("BTC")).await {
        Ok(_) => println!("   Cancelled all BTC orders"),
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
