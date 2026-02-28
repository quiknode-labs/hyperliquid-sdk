//! Info API Example — Query market data and user info.
//!
//! This example shows how to query exchange metadata, prices, user positions, and more.
//!
//! # Usage
//! ```bash
//! export ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN"
//! cargo run --example info_example
//! ```

use hyperliquid_sdk::HyperliquidSDK;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("ENDPOINT").unwrap_or_else(|_| {
        eprintln!("Error: Set ENDPOINT environment variable");
        eprintln!("  export ENDPOINT='https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN'");
        std::process::exit(1);
    });

    println!("Hyperliquid Info API Example");
    println!("{}", "=".repeat(50));
    println!("Endpoint: {}...", &endpoint[..endpoint.len().min(50)]);
    println!();

    // Create SDK and get Info client
    let sdk = HyperliquidSDK::new().endpoint(&endpoint).build().await?;
    let info = sdk.info();

    // ══════════════════════════════════════════════════════════════════════════
    // Market Data
    // ══════════════════════════════════════════════════════════════════════════
    println!("Market Data");
    println!("{}", "-".repeat(30));

    // Get all mid prices
    let mids = info.all_mids(None).await?;
    if let Some(btc) = mids.get("BTC").and_then(|v| v.as_str()) {
        println!("BTC mid: ${}", btc);
    }
    if let Some(eth) = mids.get("ETH").and_then(|v| v.as_str()) {
        println!("ETH mid: ${}", eth);
    }
    if let Some(obj) = mids.as_object() {
        println!("Total assets: {}", obj.len());
    }
    println!();

    // Get L2 order book
    let book = info.l2_book("BTC", None, None).await?;
    if let Some(levels) = book.get("levels").and_then(|l| l.as_array()) {
        let bids = levels.get(0).and_then(|b| b.as_array());
        let asks = levels.get(1).and_then(|a| a.as_array());

        if let (Some(bids), Some(asks)) = (bids, asks) {
            if let (Some(best_bid), Some(best_ask)) = (bids.first(), asks.first()) {
                let bid_px = best_bid.get("px").and_then(|p| p.as_str()).unwrap_or("?");
                let bid_sz = best_bid.get("sz").and_then(|s| s.as_str()).unwrap_or("?");
                let ask_px = best_ask.get("px").and_then(|p| p.as_str()).unwrap_or("?");
                let ask_sz = best_ask.get("sz").and_then(|s| s.as_str()).unwrap_or("?");

                let bid_price: f64 = bid_px.parse().unwrap_or(0.0);
                let ask_price: f64 = ask_px.parse().unwrap_or(0.0);
                let spread = ask_price - bid_price;

                println!("BTC Book:");
                println!("  Best Bid: {} @ ${:.2}", bid_sz, bid_price);
                println!("  Best Ask: {} @ ${:.2}", ask_sz, ask_price);
                println!("  Spread: ${:.2}", spread);
            }
        }
    }
    println!();

    // Get recent trades
    let trades = info.recent_trades("ETH").await?;
    if let Some(trades_arr) = trades.as_array() {
        println!("Recent ETH trades: {}", trades_arr.len());
        if let Some(last_trade) = trades_arr.first() {
            let sz = last_trade.get("sz").and_then(|s| s.as_str()).unwrap_or("?");
            let px = last_trade.get("px").and_then(|p| p.as_str()).unwrap_or("0");
            let price: f64 = px.parse().unwrap_or(0.0);
            println!("  Last: {} @ ${:.2}", sz, price);
        }
    }
    println!();

    // ══════════════════════════════════════════════════════════════════════════
    // Exchange Metadata
    // ══════════════════════════════════════════════════════════════════════════
    println!("Exchange Metadata");
    println!("{}", "-".repeat(30));

    let meta = info.meta().await?;
    if let Some(universe) = meta.get("universe").and_then(|u| u.as_array()) {
        println!("Total perp markets: {}", universe.len());

        // Show a few markets
        for asset in universe.iter().take(5) {
            let name = asset.get("name").and_then(|n| n.as_str()).unwrap_or("?");
            let sz_decimals = asset.get("szDecimals").and_then(|s| s.as_u64()).unwrap_or(0);
            println!("  {}: {} size decimals", name, sz_decimals);
        }
    }
    println!();

    // ══════════════════════════════════════════════════════════════════════════
    // User Account (requires a valid address)
    // ══════════════════════════════════════════════════════════════════════════
    println!("User Account");
    println!("{}", "-".repeat(30));

    // Example address - replace with your address
    let user_address = "0x0000000000000000000000000000000000000000";

    match info.clearinghouse_state(user_address, None).await {
        Ok(state) => {
            let equity = state
                .get("marginSummary")
                .and_then(|m| m.get("accountValue"))
                .and_then(|v| v.as_str())
                .unwrap_or("0")
                .parse::<f64>()
                .unwrap_or(0.0);
            println!("Account equity: ${:.2}", equity);

            if let Some(positions) = state.get("assetPositions").and_then(|p| p.as_array()) {
                if !positions.is_empty() {
                    println!("Open positions: {}", positions.len());
                    for pos in positions.iter().take(3) {
                        if let Some(position) = pos.get("position") {
                            let coin = position.get("coin").and_then(|c| c.as_str()).unwrap_or("?");
                            let size = position.get("szi").and_then(|s| s.as_str()).unwrap_or("0");
                            let entry = position.get("entryPx").and_then(|e| e.as_str()).unwrap_or("?");
                            let pnl = position
                                .get("unrealizedPnl")
                                .and_then(|p| p.as_str())
                                .unwrap_or("0")
                                .parse::<f64>()
                                .unwrap_or(0.0);
                            println!("  {}: {} @ {} (PnL: ${:.2})", coin, size, entry, pnl);
                        }
                    }
                } else {
                    println!("  No open positions");
                }
            }
        }
        Err(e) => {
            println!("  Could not fetch user data: {}", e);
        }
    }
    println!();

    // ══════════════════════════════════════════════════════════════════════════
    // Funding Rates
    // ══════════════════════════════════════════════════════════════════════════
    println!("Funding Rates");
    println!("{}", "-".repeat(30));

    match info.predicted_fundings().await {
        Ok(fundings) => {
            if let Some(fundings_arr) = fundings.as_array() {
                println!("Predicted funding rates for {} assets:", fundings_arr.len());
                for f in fundings_arr.iter().take(5) {
                    let coin = f.get("coin").and_then(|c| c.as_str()).unwrap_or("?");
                    let rate = f
                        .get("fundingRate")
                        .and_then(|r| r.as_str())
                        .unwrap_or("0")
                        .parse::<f64>()
                        .unwrap_or(0.0)
                        * 100.0;
                    println!("  {}: {:.4}%", coin, rate);
                }
            }
        }
        Err(e) => {
            println!("  Could not fetch funding: {}", e);
        }
    }

    println!();
    println!("{}", "=".repeat(50));
    println!("Done!");

    Ok(())
}
