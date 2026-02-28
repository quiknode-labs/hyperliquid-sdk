//! Market Order Example
//!
//! Place market buy and sell orders.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! export PRIVATE_KEY="0x..."
//! cargo run --example market_order
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
        eprintln!("  cargo run --example market_order");
        std::process::exit(1);
    }

    println!("Market Order Example");
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

    // Market buy by notional value ($100 worth)
    println!("\n1. Market Buy $100 of BTC:");
    match sdk.market_buy("BTC").await.notional(100.0).await {
        Ok(order) => {
            println!("   Status: {}", order.status);
            println!("   OID: {:?}", order.oid);
            println!("   Filled: {:?} @ ${:?}", order.filled_size, order.avg_price);
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Market buy by size
    println!("\n2. Market Buy 0.001 BTC:");
    match sdk.market_buy("BTC").await.size(0.001).await {
        Ok(order) => {
            println!("   Status: {}", order.status);
            println!("   OID: {:?}", order.oid);
            println!("   Filled: {:?} @ ${:?}", order.filled_size, order.avg_price);
        }
        Err(e) => println!("   Error: {}", e),
    }

    // Market sell
    println!("\n3. Market Sell 0.001 BTC:");
    match sdk.market_sell("BTC").await.size(0.001).await {
        Ok(order) => {
            println!("   Status: {}", order.status);
            println!("   OID: {:?}", order.oid);
            println!("   Filled: {:?} @ ${:?}", order.filled_size, order.avg_price);
        }
        Err(e) => println!("   Error: {}", e),
    }

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
