//! WebSocket Streaming Example — Real-time market data via WebSocket.
//!
//! This example demonstrates:
//! - Connecting to Hyperliquid's WebSocket API
//! - Subscribing to trades, orders, and book updates
//! - Automatic reconnection handling
//! - Graceful shutdown
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint/TOKEN"
//! cargo run --example websocket_streaming
//! ```

use hyperliquid_sdk::HyperliquidSDK;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::Duration;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    tracing_subscriber::fmt::init();

    let endpoint = std::env::var("ENDPOINT").ok();

    println!("Hyperliquid WebSocket Streaming Example");
    println!("{}", "=".repeat(50));

    if endpoint.is_none() {
        println!();
        println!("Usage:");
        println!("  export ENDPOINT='https://YOUR-ENDPOINT/TOKEN'");
        println!("  cargo run --example websocket_streaming");
        std::process::exit(1);
    }

    if let Some(ref ep) = endpoint {
        let display_len = ep.len().min(60);
        println!("Endpoint: {}{}", &ep[..display_len], if ep.len() > 60 { "..." } else { "" });
    }
    println!();

    // Create SDK
    let sdk = HyperliquidSDK::new()
        .endpoint(endpoint.as_ref().unwrap())
        .build()
        .await?;

    // Create stream via SDK
    let trade_count = Arc::new(AtomicUsize::new(0));
    let book_count = Arc::new(AtomicUsize::new(0));
    let trade_count_cb = trade_count.clone();
    let book_count_cb = book_count.clone();

    let mut stream = sdk.stream()
        .on_open(|| {
            println!("[CONNECTED] WebSocket stream ready");
        })
        .on_error(|e| {
            eprintln!("[ERROR] {}", e);
        })
        .on_close(|| {
            println!("[CLOSED] Stream stopped");
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

        if let Some(block) = data.get("block") {
            if let Some(events) = block.get("events").and_then(|e| e.as_array()) {
                for event in events {
                    if let Some(event_arr) = event.as_array() {
                        if event_arr.len() >= 2 {
                            if let Some(trade) = event_arr.get(1) {
                                let coin = trade.get("coin").and_then(|c| c.as_str()).unwrap_or("?");
                                let px = trade.get("px").and_then(|p| p.as_str()).unwrap_or("?");
                                let sz = trade.get("sz").and_then(|s| s.as_str()).unwrap_or("?");
                                let side = trade.get("side").and_then(|s| s.as_str()).unwrap_or("?");
                                let side_name = if side == "B" { "BUY" } else { "SELL" };
                                let price: f64 = px.parse().unwrap_or(0.0);
                                println!("[TRADE] {}: {} {} @ ${:.2}", coin, side_name, sz, price);
                            }
                        }
                    }
                }
            }
        }
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Subscribe to Book Updates
    // ─────────────────────────────────────────────────────────────────────────

    println!("Subscribing to BTC book updates...");
    let _book_sub = stream.book_updates(&["BTC"], move |data| {
        book_count_cb.fetch_add(1, Ordering::SeqCst);

        if let Some(block) = data.get("block") {
            if let Some(events) = block.get("events").and_then(|e| e.as_array()) {
                for event in events {
                    if let Some(event_arr) = event.as_array() {
                        if event_arr.len() >= 2 {
                            if let Some(book_data) = event_arr.get(1) {
                                let coin = book_data.get("coin").and_then(|c| c.as_str()).unwrap_or("?");
                                if let Some(levels) = book_data.get("levels").and_then(|l| l.as_array()) {
                                    let bids = levels.get(0).and_then(|b| b.as_array());
                                    let asks = levels.get(1).and_then(|a| a.as_array());

                                    if let (Some(bids), Some(asks)) = (bids, asks) {
                                        if let (Some(best_bid), Some(best_ask)) = (bids.first(), asks.first()) {
                                            let bid_px = best_bid
                                                .get("px")
                                                .and_then(|p| p.as_str())
                                                .unwrap_or("0")
                                                .parse::<f64>()
                                                .unwrap_or(0.0);
                                            let ask_px = best_ask
                                                .get("px")
                                                .and_then(|p| p.as_str())
                                                .unwrap_or("0")
                                                .parse::<f64>()
                                                .unwrap_or(0.0);
                                            let spread = ask_px - bid_px;
                                            println!(
                                                "[BOOK] {}: Bid ${:.2} | Ask ${:.2} | Spread ${:.2}",
                                                coin, bid_px, ask_px, spread
                                            );
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Start Streaming
    // ─────────────────────────────────────────────────────────────────────────

    println!();
    println!("Streaming... (will run for 30 seconds)");
    println!("{}", "-".repeat(50));

    stream.start()?;

    // Run for 30 seconds
    tokio::time::sleep(Duration::from_secs(30)).await;

    // Stop the stream
    stream.stop();

    println!();
    println!("{}", "=".repeat(50));
    println!(
        "Received: {} trades, {} book updates",
        trade_count.load(Ordering::SeqCst),
        book_count.load(Ordering::SeqCst)
    );
    println!("Done!");

    Ok(())
}
