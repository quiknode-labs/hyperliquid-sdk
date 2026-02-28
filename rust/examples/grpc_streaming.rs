//! gRPC Streaming Example — High-performance real-time market data via gRPC.
//!
//! This example demonstrates:
//! - Connecting to Hyperliquid's gRPC streaming API
//! - Subscribing to trades, orders, blocks, and L2/L4 order books
//! - Automatic reconnection handling
//! - Graceful shutdown
//!
//! gRPC offers lower latency than WebSocket for high-frequency data.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint/TOKEN"
//! cargo run --example grpc_streaming
//! ```

use hyperliquid_sdk::HyperliquidSDK;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::Duration;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    tracing_subscriber::fmt::init();

    let endpoint = std::env::var("ENDPOINT").ok();

    if endpoint.is_none() {
        println!("Hyperliquid gRPC Streaming Example");
        println!("{}", "=".repeat(50));
        println!();
        println!("Usage:");
        println!("  export ENDPOINT='https://YOUR-ENDPOINT/TOKEN'");
        println!("  cargo run --example grpc_streaming");
        std::process::exit(1);
    }

    println!("Hyperliquid gRPC Streaming Example");
    println!("{}", "=".repeat(50));
    if let Some(ref ep) = endpoint {
        let display_len = ep.len().min(60);
        println!("Endpoint: {}...", &ep[..display_len]);
    }
    println!();

    // Create SDK
    let sdk = HyperliquidSDK::new()
        .endpoint(endpoint.as_ref().unwrap())
        .build()
        .await?;

    // Create counters
    let trade_count = Arc::new(AtomicUsize::new(0));
    let book_count = Arc::new(AtomicUsize::new(0));
    let block_count = Arc::new(AtomicUsize::new(0));
    let trade_count_cb = trade_count.clone();
    let book_count_cb = book_count.clone();
    let block_count_cb = block_count.clone();

    // Create gRPC stream via SDK
    let mut stream = sdk.grpc()
        .on_connect(|| {
            println!("[CONNECTED] gRPC stream ready");
        })
        .on_error(|e| {
            eprintln!("[ERROR] {}", e);
        })
        .on_close(|| {
            println!("[CLOSED] gRPC stream stopped");
        })
        .on_state_change(|state| {
            println!("[STATE] {:?}", state);
        })
        .on_reconnect(|attempt| {
            println!("[RECONNECT] Attempt {}", attempt);
        });

    // ─────────────────────────────────────────────────────────────────────────
    // Subscribe to Trades
    // ─────────────────────────────────────────────────────────────────────────

    println!("Subscribing to BTC and ETH trades...");
    let _trade_sub = stream.trades(&["BTC", "ETH"], move |data| {
        trade_count_cb.fetch_add(1, Ordering::SeqCst);

        let coin = data.get("coin").and_then(|c| c.as_str()).unwrap_or("?");
        let px = data
            .get("px")
            .and_then(|p| p.as_str())
            .unwrap_or("0")
            .parse::<f64>()
            .unwrap_or(0.0);
        let sz = data.get("sz").and_then(|s| s.as_str()).unwrap_or("?");
        let side = data.get("side").and_then(|s| s.as_str()).unwrap_or("?");
        let side_name = if side == "B" { "BUY" } else { "SELL" };
        println!("[TRADE] {}: {} {} @ ${:.2}", coin, side_name, sz, px);
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Subscribe to Book Updates
    // ─────────────────────────────────────────────────────────────────────────

    println!("Subscribing to BTC book updates...");
    let _book_sub = stream.book_updates(&["BTC"], move |data| {
        book_count_cb.fetch_add(1, Ordering::SeqCst);

        let coin = data.get("coin").and_then(|c| c.as_str()).unwrap_or("?");
        let bids = data.get("bids").and_then(|b| b.as_array());
        let asks = data.get("asks").and_then(|a| a.as_array());

        if let (Some(bids), Some(asks)) = (bids, asks) {
            if let (Some(best_bid), Some(best_ask)) = (bids.first(), asks.first()) {
                let bid_px = best_bid
                    .get("price")
                    .and_then(|p| p.as_str())
                    .or_else(|| best_bid.get("price").and_then(|p| p.as_f64()).map(|_| "0"))
                    .unwrap_or("0")
                    .parse::<f64>()
                    .unwrap_or(0.0);
                let ask_px = best_ask
                    .get("price")
                    .and_then(|p| p.as_str())
                    .or_else(|| best_ask.get("price").and_then(|p| p.as_f64()).map(|_| "0"))
                    .unwrap_or("0")
                    .parse::<f64>()
                    .unwrap_or(0.0);
                println!("[BOOK] {}: Bid ${:.2} | Ask ${:.2}", coin, bid_px, ask_px);
            }
        }
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Subscribe to L2 Order Book
    // ─────────────────────────────────────────────────────────────────────────

    println!("Subscribing to ETH L2 order book...");
    let _l2_sub = stream.l2_book("ETH", move |data| {
        let coin = data.get("coin").and_then(|c| c.as_str()).unwrap_or("?");
        let bids = data.get("bids").and_then(|b| b.as_array());
        let asks = data.get("asks").and_then(|a| a.as_array());
        let bid_levels = bids.map(|b| b.len()).unwrap_or(0);
        let ask_levels = asks.map(|a| a.len()).unwrap_or(0);
        println!("[L2] {}: {} bid levels, {} ask levels", coin, bid_levels, ask_levels);
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Subscribe to Blocks
    // ─────────────────────────────────────────────────────────────────────────

    println!("Subscribing to blocks...");
    let _block_sub = stream.blocks(move |data| {
        block_count_cb.fetch_add(1, Ordering::SeqCst);

        let block_num = data
            .get("block_number")
            .and_then(|b| b.as_u64())
            .unwrap_or(0);
        println!("[BLOCK] #{}", block_num);
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Start Streaming
    // ─────────────────────────────────────────────────────────────────────────

    println!();
    println!("Streaming via gRPC... (will run for 30 seconds)");
    println!("{}", "-".repeat(50));

    stream.start()?;

    // Run for 30 seconds
    tokio::time::sleep(Duration::from_secs(30)).await;

    // Stop the stream
    stream.stop();

    println!();
    println!("{}", "=".repeat(50));
    println!(
        "Received: {} trades, {} book updates, {} blocks",
        trade_count.load(Ordering::SeqCst),
        book_count.load(Ordering::SeqCst),
        block_count.load(Ordering::SeqCst)
    );
    println!("Done!");

    Ok(())
}
