//! Trades Stream Example
//!
//! Stream real-time trade executions.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint/TOKEN"
//! cargo run --example stream_trades
//! ```

use hyperliquid_sdk::HyperliquidSDK;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::Duration;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").ok();

    if endpoint.is_none() {
        eprintln!("Usage:");
        eprintln!("  export ENDPOINT='https://your-endpoint/TOKEN'");
        eprintln!("  cargo run --example stream_trades");
        std::process::exit(1);
    }

    println!("Trades Stream Example");
    println!("{}", "=".repeat(50));

    // Create SDK
    let sdk = HyperliquidSDK::new()
        .endpoint(endpoint.as_ref().unwrap())
        .build()
        .await?;

    // Create stream via SDK
    println!("\n1. Creating WebSocket stream...");

    let trade_count = Arc::new(AtomicUsize::new(0));
    let trade_count_cb = trade_count.clone();

    let mut stream = sdk.stream()
        .on_open(|| {
            println!("   [Connected]");
        })
        .on_error(|e| {
            eprintln!("   [Error] {}", e);
        });

    // Subscribe to trades
    let assets = ["BTC", "ETH"];
    println!("2. Subscribing to trades: {:?}", assets);

    let _sub = stream.trades(&assets, move |data| {
        if let Some(trades) = data.get("data").and_then(|v| v.as_array()) {
            for trade in trades {
                let count = trade_count_cb.fetch_add(1, Ordering::SeqCst) + 1;
                let coin = trade.get("coin").and_then(|v| v.as_str()).unwrap_or("?");
                let side = trade.get("side").and_then(|v| v.as_str()).unwrap_or("?");
                let side_str = if side == "B" { "BUY " } else { "SELL" };
                let sz = trade.get("sz").and_then(|v| v.as_str()).unwrap_or("?");
                let px = trade.get("px").and_then(|v| v.as_str()).unwrap_or("?");

                // Only print first 50 trades
                if count <= 50 {
                    println!("   [{}] {} {} {} @ ${}", count, coin, side_str, sz, px);
                }
            }
        }
    });

    // Start streaming
    println!("\n3. Trade Executions (60 seconds):");
    println!("   Format: ASSET SIDE SIZE @ PRICE");

    stream.start()?;

    // Run for 60 seconds
    tokio::time::sleep(Duration::from_secs(60)).await;

    stream.stop();

    let total = trade_count.load(Ordering::SeqCst);
    println!("\n   Total trades: {}", total);

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
