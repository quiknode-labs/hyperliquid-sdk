//! Fluent Builder Example
//!
//! Demonstrates the fluent API for building orders.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example fluent_builder
//! ```

use hyperliquid_sdk::{HyperliquidSDK, Order, TriggerOrder};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();
    let private_key = std::env::var("PRIVATE_KEY").ok();

    if endpoint.is_none() || private_key.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        eprintln!("  export PRIVATE_KEY='0x...'");
        eprintln!("  cargo run --example fluent_builder");
        std::process::exit(1);
    }

    println!("Fluent Order Builder Example");
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

    // ══════════════════════════════════════════════════════════════════════════
    // Order Builder
    // ══════════════════════════════════════════════════════════════════════════

    println!("\n{}", "─".repeat(50));
    println!("Order Builder Examples");
    println!("{}", "─".repeat(50));

    // GTC Limit Buy
    println!("\n1. GTC Limit Buy:");
    let order = Order::buy("BTC")
        .size(0.001)
        .price(mid * 0.97)
        .gtc();
    println!("   {:?}", order);

    // IOC Limit Sell
    println!("\n2. IOC Limit Sell:");
    let order = Order::sell("BTC")
        .size(0.001)
        .price(mid * 1.03)
        .ioc();
    println!("   {:?}", order);

    // ALO (Post-Only) Buy
    println!("\n3. ALO (Post-Only) Buy:");
    let order = Order::buy("BTC")
        .size(0.001)
        .price(mid * 0.95)
        .alo();
    println!("   {:?}", order);

    // Market Order
    println!("\n4. Market Buy:");
    let order = Order::buy("BTC")
        .notional(100.0)
        .market();
    println!("   {:?}", order);

    // Reduce-Only Order
    println!("\n5. Reduce-Only Sell:");
    let order = Order::sell("BTC")
        .size(0.001)
        .price(mid * 1.05)
        .gtc()
        .reduce_only();
    println!("   {:?}", order);

    // ══════════════════════════════════════════════════════════════════════════
    // Trigger Order Builder
    // ══════════════════════════════════════════════════════════════════════════

    println!("\n{}", "─".repeat(50));
    println!("Trigger Order Builder Examples");
    println!("{}", "─".repeat(50));

    // Stop Loss
    println!("\n6. Stop Loss:");
    let trigger = TriggerOrder::stop_loss("BTC")
        .size(0.001)
        .trigger_price(mid * 0.95);
    println!("   {:?}", trigger);

    // Take Profit
    println!("\n7. Take Profit:");
    let trigger = TriggerOrder::take_profit("BTC")
        .size(0.001)
        .trigger_price(mid * 1.10);
    println!("   {:?}", trigger);

    // Stop Loss with Limit
    println!("\n8. Stop Loss with Limit:");
    let trigger = TriggerOrder::stop_loss("BTC")
        .size(0.001)
        .trigger_price(mid * 0.95)
        .limit(mid * 0.94);
    println!("   {:?}", trigger);

    // ══════════════════════════════════════════════════════════════════════════
    // Execute an order
    // ══════════════════════════════════════════════════════════════════════════

    println!("\n{}", "─".repeat(50));
    println!("Execute Order");
    println!("{}", "─".repeat(50));

    let order = Order::buy("BTC")
        .size(0.001)
        .price(mid * 0.97)
        .gtc();

    println!("\n9. Placing order...");
    match sdk.order(order).await {
        Ok(result) => {
            println!("   Status: {}", result.status);
            println!("   OID: {:?}", result.oid);

            // Cancel if resting
            if result.is_resting() {
                if let Some(oid) = result.oid {
                    let _ = sdk.cancel(oid, "BTC").await;
                    println!("   (Cancelled)");
                }
            }
        }
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
