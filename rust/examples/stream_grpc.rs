//! gRPC Streaming Example
//!
//! High-performance streaming via gRPC (lower latency than WebSocket).
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint/TOKEN"
//! cargo run --example stream_grpc
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
        eprintln!("  cargo run --example stream_grpc");
        std::process::exit(1);
    }

    println!("gRPC Streaming Example");
    println!("{}", "=".repeat(50));

    // Create SDK
    let sdk = HyperliquidSDK::new()
        .endpoint(endpoint.as_ref().unwrap())
        .build()
        .await?;

    // Create gRPC stream via SDK
    println!("\n1. Creating gRPC stream...");

    let trade_count = Arc::new(AtomicUsize::new(0));
    let book_count = Arc::new(AtomicUsize::new(0));
    let trade_count_cb = trade_count.clone();
    let book_count_cb = book_count.clone();

    let mut grpc = sdk.grpc()
        .on_connect(|| {
            println!("   [Connected]");
        })
        .on_error(|e| {
            eprintln!("   [Error] {}", e);
        });

    // Subscribe to trades
    println!("2. Subscribing to BTC, ETH trades...");
    let _trade_sub = grpc.trades(&["BTC", "ETH"], move |data| {
        let count = trade_count_cb.fetch_add(1, Ordering::SeqCst) + 1;
        if count <= 10 {
            if let Some(trades) = data.get("data").and_then(|v| v.as_array()) {
                for trade in trades {
                    let coin = trade.get("coin").and_then(|v| v.as_str()).unwrap_or("?");
                    let px = trade.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                    let sz = trade.get("sz").and_then(|v| v.as_str()).unwrap_or("?");
                    println!("   [trades {}] {} {} @ {}", count, coin, sz, px);
                }
            }
        }
    });

    // Subscribe to L2 book
    println!("3. Subscribing to BTC L2 book...");
    let _book_sub = grpc.l2_book("BTC", move |data| {
        let count = book_count_cb.fetch_add(1, Ordering::SeqCst) + 1;
        if count <= 10 {
            if let Some(levels) = data.get("levels").and_then(|v| v.as_array()) {
                if levels.len() >= 2 {
                    let bids = levels[0].as_array();
                    let asks = levels[1].as_array();
                    if let (Some(bids), Some(asks)) = (bids, asks) {
                        if let (Some(bid), Some(ask)) = (bids.first(), asks.first()) {
                            let bid_px = bid.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                            let ask_px = ask.get("px").and_then(|v| v.as_str()).unwrap_or("?");
                            println!("   [l2Book {}] BTC: {} / {}", count, bid_px, ask_px);
                        }
                    }
                }
            }
        }
    });

    // Start streaming
    println!("\n4. Receiving messages (30 seconds):");

    grpc.start()?;

    // Run for 30 seconds
    tokio::time::sleep(Duration::from_secs(30)).await;

    grpc.stop();

    let total_trades = trade_count.load(Ordering::SeqCst);
    let total_books = book_count.load(Ordering::SeqCst);
    println!("\n   Trades: {}, L2 Book: {}", total_trades, total_books);

    println!("\n{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
