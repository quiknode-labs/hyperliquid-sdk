//! HyperCore API Example — Low-level block and trade data via JSON-RPC.
//!
//! This example shows how to query blocks, trades, and orders using the HyperCore API.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! cargo run --example hypercore_example
//! ```

use hyperliquid_sdk::HyperliquidSDK;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").unwrap_or_else(|_| {
        eprintln!("Error: Set ENDPOINT environment variable");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        std::process::exit(1);
    });

    println!("Hyperliquid HyperCore API Example");
    println!("{}", "=".repeat(50));
    println!("Endpoint: {}...", &endpoint[..endpoint.len().min(50)]);
    println!();

    // Create SDK and get HyperCore client
    let sdk = HyperliquidSDK::new().endpoint(&endpoint).build().await?;
    let hc = sdk.core();

    // ══════════════════════════════════════════════════════════════════════════
    // Block Data
    // ══════════════════════════════════════════════════════════════════════════
    println!("Block Data");
    println!("{}", "-".repeat(30));

    // Get latest block number
    let block_num = hc.latest_block_number(None).await?;
    println!("Latest block: {}", block_num);

    // Get block by number
    let block = hc.get_block(block_num, None).await?;
    if let Some(txs) = block.get("transactions").and_then(|t| t.as_array()) {
        println!("Block {}:", block_num);
        println!("  Transactions: {}", txs.len());
        if let Some(ts) = block.get("timestamp") {
            println!("  Timestamp: {}", ts);
        }
    }
    println!();

    // ══════════════════════════════════════════════════════════════════════════
    // Recent Trades
    // ══════════════════════════════════════════════════════════════════════════
    println!("Recent Trades");
    println!("{}", "-".repeat(30));

    // Get latest trades (all coins)
    let trades = hc.latest_trades(Some(5), None).await?;
    if let Some(trades_arr) = trades.as_array() {
        println!("Last {} trades:", trades_arr.len());
        for trade in trades_arr.iter().take(5) {
            let coin = trade.get("coin").and_then(|c| c.as_str()).unwrap_or("?");
            let px = trade.get("px").and_then(|p| p.as_str()).unwrap_or("0");
            let sz = trade.get("sz").and_then(|s| s.as_str()).unwrap_or("?");
            let side = trade.get("side").and_then(|s| s.as_str()).unwrap_or("?");
            let price: f64 = px.parse().unwrap_or(0.0);
            println!("  {}: {} @ ${:.2} ({})", coin, sz, price, side);
        }
    }
    println!();

    // Get trades for specific coin
    let btc_trades = hc.latest_trades(Some(3), Some("BTC")).await?;
    if let Some(trades_arr) = btc_trades.as_array() {
        println!("Last {} BTC trades:", trades_arr.len());
        for trade in trades_arr {
            let px = trade.get("px").and_then(|p| p.as_str()).unwrap_or("0");
            let sz = trade.get("sz").and_then(|s| s.as_str()).unwrap_or("?");
            let side = trade.get("side").and_then(|s| s.as_str()).unwrap_or("?");
            let price: f64 = px.parse().unwrap_or(0.0);
            println!("  {} @ ${:.2} ({})", sz, price, side);
        }
    }
    println!();

    // ══════════════════════════════════════════════════════════════════════════
    // Recent Orders
    // ══════════════════════════════════════════════════════════════════════════
    println!("Recent Orders");
    println!("{}", "-".repeat(30));

    match hc.latest_orders(Some(5)).await {
        Ok(orders) => {
            if let Some(orders_arr) = orders.as_array() {
                println!("Last {} orders:", orders_arr.len());
                for order in orders_arr.iter().take(5) {
                    let coin = order.get("coin").and_then(|c| c.as_str()).unwrap_or("?");
                    let side = order.get("side").and_then(|s| s.as_str()).unwrap_or("?");
                    let px = order.get("limitPx").and_then(|p| p.as_str()).unwrap_or("0");
                    let sz = order.get("sz").and_then(|s| s.as_str()).unwrap_or("?");
                    let status = order.get("status").and_then(|s| s.as_str()).unwrap_or("?");
                    let price: f64 = px.parse().unwrap_or(0.0);
                    println!("  {}: {} {} @ ${:.2} - {}", coin, side, sz, price, status);
                }
            }
        }
        Err(e) => {
            println!("  Could not fetch orders: {}", e);
        }
    }
    println!();

    // ══════════════════════════════════════════════════════════════════════════
    // Block Range Query
    // ══════════════════════════════════════════════════════════════════════════
    println!("Block Range Query");
    println!("{}", "-".repeat(30));

    let start_block = block_num.saturating_sub(5);
    match hc.get_batch_blocks(start_block, block_num, None).await {
        Ok(blocks) => {
            if let Some(blocks_arr) = blocks.as_array() {
                println!("Blocks {} to {}: {} blocks", start_block, block_num, blocks_arr.len());

                // Count transactions
                let mut total_txs = 0;
                for b in blocks_arr {
                    if let Some(txs) = b.get("transactions").and_then(|t| t.as_array()) {
                        total_txs += txs.len();
                    }
                }
                println!("Total transactions: {}", total_txs);
            }
        }
        Err(e) => {
            println!("  Could not fetch blocks: {}", e);
        }
    }

    println!();
    println!("{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
