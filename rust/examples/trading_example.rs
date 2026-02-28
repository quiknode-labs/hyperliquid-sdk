//! Trading Example — Place orders on Hyperliquid.
//!
//! This example shows how to place market and limit orders using the SDK.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example trading_example
//! ```

use hyperliquid_sdk::{HyperliquidSDK, Order};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Get endpoint and private key from environment
    let endpoint = std::env::var("ENDPOINT").ok();
    let private_key = std::env::var("PRIVATE_KEY").ok();

    if endpoint.is_none() {
        eprintln!("Error: Set ENDPOINT environment variable");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        std::process::exit(1);
    }

    if private_key.is_none() {
        eprintln!("Error: Set PRIVATE_KEY environment variable");
        eprintln!("  export PRIVATE_KEY='0xYourPrivateKey'");
        std::process::exit(1);
    }

    println!("Hyperliquid Trading Example");
    println!("{}", "=".repeat(50));

    // Initialize SDK with QuickNode endpoint and private key
    // All requests route through QuickNode - never directly to Hyperliquid
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
    if let Some(ep) = &endpoint {
        let display_len = ep.len().min(50);
        println!("Endpoint: {}...", &ep[..display_len]);
    }
    println!();

    // ══════════════════════════════════════════════════════════════════════════
    // Example 1: Market Buy $100 of BTC
    // ══════════════════════════════════════════════════════════════════════════
    println!("Example 1: Market Buy");
    println!("{}", "-".repeat(30));

    match sdk.market_buy("BTC").await.notional(100.0).await {
        Ok(order) => {
            println!("Order placed: {:?}", order);
            println!("  Order ID: {:?}", order.oid);
            println!("  Status: {}", order.status);
            println!(
                "  Filled: {:?} @ avg {:?}",
                order.filled_size, order.avg_price
            );
        }
        Err(e) => {
            println!("  Error: {}", e);
        }
    }

    println!();

    // ══════════════════════════════════════════════════════════════════════════
    // Example 2: Limit Order
    // ══════════════════════════════════════════════════════════════════════════
    println!("Example 2: Limit Order");
    println!("{}", "-".repeat(30));

    // Build limit order
    let order = Order::buy("ETH")
        .size(0.1)
        .price(2000.0)
        .gtc();

    // Place order
    match sdk.order(order).await {
        Ok(result) => {
            println!("Order placed: {:?}", result);
            println!("  Order ID: {:?}", result.oid);
            println!("  Status: {}", result.status);

            // Cancel if resting
            if result.is_resting() {
                let _ = result.cancel().await;
            }
        }
        Err(e) => {
            println!("  Error: {}", e);
        }
    }

    println!();

    // ══════════════════════════════════════════════════════════════════════════
    // Example 3: Stop Loss Order
    // ══════════════════════════════════════════════════════════════════════════
    println!("Example 3: Stop Loss Order");
    println!("{}", "-".repeat(30));

    use hyperliquid_sdk::TriggerOrder;

    // Build stop loss order
    let trigger = TriggerOrder::stop_loss("BTC")
        .size(0.01)
        .trigger_price(60000.0)
        .limit(59900.0);

    match sdk.trigger_order(trigger).await {
        Ok(result) => {
            println!("Stop loss placed: {:?}", result);
        }
        Err(e) => {
            println!("  Error: {}", e);
        }
    }

    println!();

    // ══════════════════════════════════════════════════════════════════════════
    // Example 4: Cancel Orders
    // ══════════════════════════════════════════════════════════════════════════
    println!("Example 4: Cancel All Orders");
    println!("{}", "-".repeat(30));

    match sdk.cancel_all(Some("BTC")).await {
        Ok(_) => {
            println!("Cancelled all BTC orders");
        }
        Err(e) => {
            println!("  Error: {}", e);
        }
    }

    println!();

    // ══════════════════════════════════════════════════════════════════════════
    // Example 5: Close Position
    // ══════════════════════════════════════════════════════════════════════════
    println!("Example 5: Close Position");
    println!("{}", "-".repeat(30));

    match sdk.close_position("BTC").await {
        Ok(result) => {
            println!("Position closed: {:?}", result);
        }
        Err(e) => {
            println!("  Error: {}", e);
        }
    }

    println!();
    println!("{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
