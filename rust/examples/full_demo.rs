//! Full Demo — Comprehensive example of all SDK capabilities.
//!
//! This example demonstrates all major SDK features:
//! - Info API (market data, user info)
//! - HyperCore API (blocks, trades, orders)
//! - EVM API (chain data, balances)
//! - WebSocket streaming
//! - gRPC streaming (optional)
//! - Trading (orders, positions)
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint/TOKEN"
//! export PRIVATE_KEY="0x..."  # Optional, for trading
//! cargo run --example full_demo
//! ```

use hyperliquid_sdk::{HyperliquidSDK, Order};
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::Duration;

fn separator(title: &str) {
    println!();
    println!("{}", "=".repeat(60));
    println!("  {}", title);
    println!("{}", "=".repeat(60));
}

fn subsection(title: &str) {
    println!();
    println!("--- {} ---", title);
}

async fn demo_info_api(sdk: &HyperliquidSDK) -> Result<(), Box<dyn std::error::Error>> {
    separator("INFO API");

    let info = sdk.info();

    subsection("Market Prices");
    let mids = info.all_mids(None).await?;
    if let Some(obj) = mids.as_object() {
        println!("Total markets: {}", obj.len());
        for coin in &["BTC", "ETH", "SOL", "DOGE"] {
            if let Some(price) = mids.get(*coin).and_then(|v| v.as_str()) {
                let p: f64 = price.parse().unwrap_or(0.0);
                println!("  {}: ${:.2}", coin, p);
            }
        }
    }

    subsection("Order Book");
    let book = info.l2_book("BTC", None, None).await?;
    if let Some(levels) = book.get("levels").and_then(|l| l.as_array()) {
        let bids = levels.get(0).and_then(|b| b.as_array());
        let asks = levels.get(1).and_then(|a| a.as_array());

        if let (Some(bids), Some(asks)) = (bids, asks) {
            if let (Some(best_bid), Some(best_ask)) = (bids.first(), asks.first()) {
                let bid_px: f64 = best_bid
                    .get("px")
                    .and_then(|p| p.as_str())
                    .unwrap_or("0")
                    .parse()
                    .unwrap_or(0.0);
                let bid_sz = best_bid.get("sz").and_then(|s| s.as_str()).unwrap_or("?");
                let ask_px: f64 = best_ask
                    .get("px")
                    .and_then(|p| p.as_str())
                    .unwrap_or("0")
                    .parse()
                    .unwrap_or(0.0);
                let ask_sz = best_ask.get("sz").and_then(|s| s.as_str()).unwrap_or("?");
                let spread = ask_px - bid_px;

                println!("  Best Bid: {} @ ${:.2}", bid_sz, bid_px);
                println!("  Best Ask: {} @ ${:.2}", ask_sz, ask_px);
                println!("  Spread: ${:.2}", spread);
            }
        }
    }

    subsection("Recent Trades");
    let trades = info.recent_trades("ETH").await?;
    if let Some(trades_arr) = trades.as_array() {
        println!("Last 3 ETH trades:");
        for t in trades_arr.iter().take(3) {
            let sz = t.get("sz").and_then(|s| s.as_str()).unwrap_or("?");
            let px: f64 = t
                .get("px")
                .and_then(|p| p.as_str())
                .unwrap_or("0")
                .parse()
                .unwrap_or(0.0);
            let side = t.get("side").and_then(|s| s.as_str()).unwrap_or("?");
            println!("  {} @ ${:.2} ({})", sz, px, side);
        }
    }

    subsection("Exchange Metadata");
    let meta = info.meta().await?;
    if let Some(universe) = meta.get("universe").and_then(|u| u.as_array()) {
        println!("Total perp markets: {}", universe.len());
    }

    subsection("Predicted Funding");
    let fundings = info.predicted_fundings().await?;
    if let Some(fundings_arr) = fundings.as_array() {
        // Sort by absolute funding rate
        let mut sorted: Vec<_> = fundings_arr
            .iter()
            .filter_map(|f| {
                let coin = f.get("coin").and_then(|c| c.as_str())?;
                let rate: f64 = f
                    .get("fundingRate")
                    .and_then(|r| r.as_str())
                    .and_then(|s| s.parse().ok())?;
                Some((coin, rate))
            })
            .collect();
        sorted.sort_by(|a, b| b.1.abs().partial_cmp(&a.1.abs()).unwrap());

        println!("Top 3 funding rates:");
        for (coin, rate) in sorted.iter().take(3) {
            println!("  {}: {:+.4}% (8h)", coin, rate * 100.0);
        }
    }

    Ok(())
}

async fn demo_hypercore_api(sdk: &HyperliquidSDK) -> Result<(), Box<dyn std::error::Error>> {
    separator("HYPERCORE API");

    let hc = sdk.core();

    subsection("Latest Block");
    let block_num = hc.latest_block_number(None).await?;
    println!("Latest block: {}", block_num);

    let block = hc.get_block(block_num, None).await?;
    if let Some(txs) = block.get("transactions").and_then(|t| t.as_array()) {
        println!("Block {}: {} transactions", block_num, txs.len());
    }

    subsection("Recent Trades");
    let trades = hc.latest_trades(Some(5), None).await?;
    if let Some(trades_arr) = trades.as_array() {
        println!("Last 5 trades across all markets:");
        for t in trades_arr {
            let coin = t.get("coin").and_then(|c| c.as_str()).unwrap_or("?");
            let sz = t.get("sz").and_then(|s| s.as_str()).unwrap_or("?");
            let px: f64 = t
                .get("px")
                .and_then(|p| p.as_str())
                .unwrap_or("0")
                .parse()
                .unwrap_or(0.0);
            println!("  {}: {} @ ${:.2}", coin, sz, px);
        }
    }

    subsection("Recent Orders");
    match hc.latest_orders(Some(5)).await {
        Ok(orders) => {
            if let Some(orders_arr) = orders.as_array() {
                println!("Last 5 orders:");
                for o in orders_arr {
                    let coin = o.get("coin").and_then(|c| c.as_str()).unwrap_or("?");
                    let side = o.get("side").and_then(|s| s.as_str()).unwrap_or("?");
                    let status = o.get("status").and_then(|s| s.as_str()).unwrap_or("?");
                    let px: f64 = o
                        .get("limitPx")
                        .and_then(|p| p.as_str())
                        .unwrap_or("0")
                        .parse()
                        .unwrap_or(0.0);
                    println!("  {}: {} @ ${:.2} - {}", coin, side, px, status);
                }
            }
        }
        Err(e) => {
            println!("  Could not fetch orders: {}", e);
        }
    }

    Ok(())
}

async fn demo_evm_api(sdk: &HyperliquidSDK) -> Result<(), Box<dyn std::error::Error>> {
    separator("EVM API");

    let evm = sdk.evm();

    subsection("Chain Info");
    let chain_id = evm.chain_id().await?;
    let block_num = evm.block_number().await?;
    let gas_price = evm.gas_price().await?;

    let network = match chain_id {
        999 => "Mainnet",
        998 => "Testnet",
        _ => "Unknown",
    };

    println!("Chain ID: {} ({})", chain_id, network);
    println!("Block: {}", block_num);
    println!("Gas: {:.2} Gwei", gas_price as f64 / 1e9);

    subsection("Latest Block");
    let block_hex = format!("0x{:x}", block_num);
    let block = evm.get_block_by_number(&block_hex, false).await?;
    if !block.is_null() {
        println!("Block {}:", block_num);
        if let Some(hash) = block.get("hash").and_then(|h| h.as_str()) {
            println!("  Hash: {}...", &hash[..hash.len().min(30)]);
        }
        if let Some(gas_used) = block.get("gasUsed").and_then(|g| g.as_str()) {
            let gas = u64::from_str_radix(gas_used.trim_start_matches("0x"), 16).unwrap_or(0);
            println!("  Gas Used: {}", gas);
        }
    }

    Ok(())
}

async fn demo_websocket(sdk: &HyperliquidSDK, duration: u64) {
    separator("WEBSOCKET STREAMING");

    let trade_count = Arc::new(AtomicUsize::new(0));
    let book_count = Arc::new(AtomicUsize::new(0));
    let trade_count_cb = trade_count.clone();
    let book_count_cb = book_count.clone();

    let mut stream = sdk.stream()
        .on_open(|| {
            println!("  [CONNECTED] WebSocket stream ready");
        })
        .on_error(|e| {
            println!("  [ERROR] {}", e);
        });

    stream.trades(&["BTC", "ETH"], move |data| {
        let count = trade_count_cb.fetch_add(1, Ordering::SeqCst);
        if count < 3 {
            if let Some(trades) = data.get("data").and_then(|d| d.as_array()) {
                for trade in trades.iter().take(1) {
                    let coin = trade.get("coin").and_then(|c| c.as_str()).unwrap_or("?");
                    let sz = trade.get("sz").and_then(|s| s.as_str()).unwrap_or("?");
                    let px = trade.get("px").and_then(|p| p.as_str()).unwrap_or("?");
                    println!("  [TRADE] {}: {} @ {}", coin, sz, px);
                }
            }
        }
    });

    stream.book_updates(&["BTC"], move |_data| {
        let count = book_count_cb.fetch_add(1, Ordering::SeqCst);
        if count < 2 {
            println!("  [BOOK] BTC update");
        }
    });

    println!("Streaming for {} seconds...", duration);

    if stream.start().is_ok() {
        tokio::time::sleep(Duration::from_secs(duration)).await;
        stream.stop();
    }

    println!();
    println!(
        "Received: {} trades, {} book updates",
        trade_count.load(Ordering::SeqCst),
        book_count.load(Ordering::SeqCst)
    );
}

async fn demo_grpc(sdk: &HyperliquidSDK, duration: u64) {
    separator("GRPC STREAMING");

    let trade_count = Arc::new(AtomicUsize::new(0));
    let trade_count_cb = trade_count.clone();

    let mut stream = sdk.grpc()
        .on_connect(|| {
            println!("  [CONNECTED] gRPC stream ready");
        })
        .on_error(|e| {
            println!("  [ERROR] {}", e);
        });

    stream.trades(&["BTC", "ETH"], move |data| {
        let count = trade_count_cb.fetch_add(1, Ordering::SeqCst);
        if count < 3 {
            println!("  [TRADE] {:?}", data);
        }
    });

    println!("Streaming for {} seconds...", duration);

    if stream.start().is_ok() {
        tokio::time::sleep(Duration::from_secs(duration)).await;
        stream.stop();
    }

    println!();
    println!("Received: {} trades", trade_count.load(Ordering::SeqCst));
}

async fn demo_trading(sdk: &HyperliquidSDK) {
    separator("TRADING");

    if let Some(addr) = sdk.address() {
        println!("Address: {}", addr);
    }

    subsection("Account Check");
    println!("  Trading SDK initialized successfully");
    println!("  Ready to place orders (not executing in demo)");

    subsection("Order Building (Example)");
    println!("  Market buy: sdk.market_buy(\"BTC\").notional(100.0).await");

    let order = Order::buy("ETH").size(1.0).price(4000.0).gtc();
    println!("  Limit sell: Order::sell(\"ETH\").size(1.0).price(4000.0).gtc()");
    println!("  Order built: {:?} {:?} @ {:?}", order.get_asset(), order.get_side(), order.get_price());

    println!("  Close pos:  sdk.close_position(\"BTC\").await");
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    tracing_subscriber::fmt::init();

    println!();
    println!("{}", "*".repeat(60));
    println!("  HYPERLIQUID SDK - FULL DEMO");
    println!("{}", "*".repeat(60));

    let endpoint = std::env::var("ENDPOINT").ok();
    let private_key = std::env::var("PRIVATE_KEY").ok();

    if endpoint.is_none() {
        println!();
        println!("Error: ENDPOINT not set");
        println!();
        println!("Usage:");
        println!("  export ENDPOINT='https://your-endpoint/TOKEN'");
        println!("  cargo run --example full_demo");
        std::process::exit(1);
    }

    println!();
    if let Some(ref ep) = endpoint {
        let display_len = ep.len().min(50);
        println!("Endpoint: {}...", &ep[..display_len]);
    }

    // Build SDK
    let mut builder = HyperliquidSDK::new();
    if let Some(ref ep) = endpoint {
        builder = builder.endpoint(ep);
    }
    if let Some(ref pk) = private_key {
        builder = builder.private_key(pk);
    }

    let sdk = builder.build().await?;

    // Run all demos using the same SDK instance
    demo_info_api(&sdk).await?;
    demo_hypercore_api(&sdk).await?;
    demo_evm_api(&sdk).await?;
    demo_websocket(&sdk, 5).await;
    demo_grpc(&sdk, 5).await;

    if private_key.is_some() {
        demo_trading(&sdk).await;
    } else {
        println!();
        println!("--- TRADING (skipped - no PRIVATE_KEY) ---");
    }

    separator("DONE");
    println!("All demos completed successfully!");
    println!();

    Ok(())
}
