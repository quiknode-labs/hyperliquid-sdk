//! L2 Orderbook Stream Example
//!
//! Stream L2 orderbook updates (top-of-book).
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint/TOKEN"
//! cargo run --example stream_l2_book
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
        eprintln!("  cargo run --example stream_l2_book");
        std::process::exit(1);
    }

    println!("L2 Orderbook Stream Example");
    println!("{}", "=".repeat(50));

    // Create SDK
    let sdk = HyperliquidSDK::new()
        .endpoint(endpoint.as_ref().unwrap())
        .build()
        .await?;

    // Create stream via SDK
    println!("\n1. Creating WebSocket stream...");

    let update_count = Arc::new(AtomicUsize::new(0));
    let btc_count = update_count.clone();
    let eth_count = update_count.clone();
    let sol_count = update_count.clone();

    let mut stream = sdk.stream()
        .on_open(|| {
            println!("   [Connected]");
        })
        .on_error(|e| {
            eprintln!("   [Error] {}", e);
        });

    // Subscribe to L2 book for multiple assets
    println!("2. Subscribing to L2 books: BTC, ETH, SOL");

    let _btc_sub = stream.l2_book("BTC", move |data| {
        let count = btc_count.fetch_add(1, Ordering::SeqCst) + 1;
        if count <= 30 {
            if let Some(levels) = data.get("levels").and_then(|v| v.as_array()) {
                if levels.len() >= 2 {
                    let bids = levels[0].as_array();
                    let asks = levels[1].as_array();
                    if let (Some(bids), Some(asks)) = (bids, asks) {
                        if let (Some(bid), Some(ask)) = (bids.first(), asks.first()) {
                            let bid_px = bid.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                            let bid_sz = bid.get("sz").and_then(|v| v.as_str()).unwrap_or("?");
                            let ask_px = ask.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                            let ask_sz = ask.get("sz").and_then(|v| v.as_str()).unwrap_or("?");
                            println!("   [{}] BTC: bid={} ({}) / ask={} ({})",
                                count, bid_px, bid_sz, ask_px, ask_sz);
                        }
                    }
                }
            }
        }
    });

    let _eth_sub = stream.l2_book("ETH", move |data| {
        let count = eth_count.fetch_add(1, Ordering::SeqCst) + 1;
        if count <= 30 {
            if let Some(levels) = data.get("levels").and_then(|v| v.as_array()) {
                if levels.len() >= 2 {
                    let bids = levels[0].as_array();
                    let asks = levels[1].as_array();
                    if let (Some(bids), Some(asks)) = (bids, asks) {
                        if let (Some(bid), Some(ask)) = (bids.first(), asks.first()) {
                            let bid_px = bid.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                            let ask_px = ask.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                            println!("   [{}] ETH: bid={} / ask={}", count, bid_px, ask_px);
                        }
                    }
                }
            }
        }
    });

    let _sol_sub = stream.l2_book("SOL", move |data| {
        let count = sol_count.fetch_add(1, Ordering::SeqCst) + 1;
        if count <= 30 {
            if let Some(levels) = data.get("levels").and_then(|v| v.as_array()) {
                if levels.len() >= 2 {
                    let bids = levels[0].as_array();
                    let asks = levels[1].as_array();
                    if let (Some(bids), Some(asks)) = (bids, asks) {
                        if let (Some(bid), Some(ask)) = (bids.first(), asks.first()) {
                            let bid_px = bid.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                            let ask_px = ask.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                            println!("   [{}] SOL: bid={} / ask={}", count, bid_px, ask_px);
                        }
                    }
                }
            }
        }
    });

    // Start streaming
    println!("\n3. Receiving L2 updates (10 seconds):");

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
