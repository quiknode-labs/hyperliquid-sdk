//! Orderbook Stream Example
//!
//! Stream real-time orderbook updates.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint/TOKEN"
//! cargo run --example stream_orderbook
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
        eprintln!("  cargo run --example stream_orderbook");
        std::process::exit(1);
    }

    println!("Orderbook Stream Example");
    println!("{}", "=".repeat(50));

    // Create SDK
    let sdk = HyperliquidSDK::new()
        .endpoint(endpoint.as_ref().unwrap())
        .build()
        .await?;

    // Create stream via SDK
    println!("\n1. Creating WebSocket stream...");

    let update_count = Arc::new(AtomicUsize::new(0));
    let update_count_cb = update_count.clone();

    let mut stream = sdk.stream()
        .on_open(|| {
            println!("   [Connected]");
        })
        .on_error(|e| {
            eprintln!("   [Error] {}", e);
        });

    // Subscribe to BTC L2 book
    println!("2. Subscribing to BTC orderbook...");
    let _sub = stream.l2_book("BTC", move |data| {
        let count = update_count_cb.fetch_add(1, Ordering::SeqCst) + 1;

        if let Some(levels) = data.get("levels").and_then(|v| v.as_array()) {
            if levels.len() >= 2 {
                let bids = levels[0].as_array();
                let asks = levels[1].as_array();
                if let (Some(bids), Some(asks)) = (bids, asks) {
                    if let (Some(bid), Some(ask)) = (bids.first(), asks.first()) {
                        let bid_px: f64 = bid.get("px")
                            .and_then(|v| v.as_str())
                            .and_then(|s| s.parse().ok())
                            .unwrap_or(0.0);
                        let ask_px: f64 = ask.get("px")
                            .and_then(|v| v.as_str())
                            .and_then(|s| s.parse().ok())
                            .unwrap_or(0.0);
                        let spread = ask_px - bid_px;
                        let spread_bps = if bid_px > 0.0 { (spread / bid_px) * 10000.0 } else { 0.0 };

                        // Only print first 20 updates
                        if count <= 20 {
                            println!("   [{}] BTC: {:.2} / {:.2} ({:.2} bps)",
                                count, bid_px, ask_px, spread_bps);
                        }
                    }
                }
            }
        }
    });

    // Start streaming
    println!("\n3. Orderbook Updates (10 seconds):");
    println!("   Format: best_bid / best_ask (spread)");

    stream.start()?;

    // Run for 10 seconds
    tokio::time::sleep(Duration::from_secs(10)).await;

    stream.stop();

    let total = update_count.load(Ordering::SeqCst);
    println!("\n   Total updates: {}", total);

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
