#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Info API Example - Query market data and user info.
 *
 * This example shows how to query exchange metadata, prices, user positions, and more.
 *
 * Requirements:
 *     npm install hyperliquid-sdk
 *
 * Usage:
 *     export ENDPOINT="https://your-endpoint.example.com/TOKEN"
 *     npx ts-node info_example.ts
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

// Get endpoint from environment
const ENDPOINT = process.env.ENDPOINT;

if (!ENDPOINT) {
  console.error("Error: Set ENDPOINT environment variable");
  console.error("  export ENDPOINT='https://your-endpoint.example.com/TOKEN'");
  process.exit(1);
}

async function main() {
  console.log("Hyperliquid Info API Example");
  console.log("=".repeat(50));
  console.log(`Endpoint: ${ENDPOINT.slice(0, 50)}...`);
  console.log();

  // Create SDK client
  const sdk = new HyperliquidSDK(ENDPOINT);

  // ==========================================================================
  // Market Data
  // ==========================================================================
  console.log("Market Data");
  console.log("-".repeat(30));

  // Get all mid prices
  const mids = await sdk.info.allMids();
  const btcMid = parseFloat(mids['BTC'] || '0');
  const ethMid = parseFloat(mids['ETH'] || '0');
  console.log(`BTC mid: $${btcMid.toLocaleString(undefined, { minimumFractionDigits: 2 })}`);
  console.log(`ETH mid: $${ethMid.toLocaleString(undefined, { minimumFractionDigits: 2 })}`);
  console.log(`Total assets: ${Object.keys(mids).length}`);
  console.log();

  // Get L2 order book
  const book = await sdk.info.l2Book("BTC");
  const levels = book.levels || [[], []];
  const bids = levels[0] || [];
  const asks = levels[1] || [];

  if (bids.length && asks.length) {
    const bestBid = bids[0];
    const bestAsk = asks[0];
    const spread = parseFloat(bestAsk.px) - parseFloat(bestBid.px);
    console.log("BTC Book:");
    console.log(`  Best Bid: ${bestBid.sz} @ $${parseFloat(bestBid.px).toLocaleString()}`);
    console.log(`  Best Ask: ${bestAsk.sz} @ $${parseFloat(bestAsk.px).toLocaleString()}`);
    console.log(`  Spread: $${spread.toLocaleString()}`);
  }
  console.log();

  // Get recent trades
  const trades = await sdk.info.recentTrades("ETH");
  console.log(`Recent ETH trades: ${trades.length}`);
  if (trades.length) {
    const lastTrade = trades[0] as Record<string, unknown>;
    console.log(`  Last: ${lastTrade.sz} @ $${parseFloat(String(lastTrade.px || 0)).toLocaleString()}`);
  }
  console.log();

  // ==========================================================================
  // Exchange Metadata
  // ==========================================================================
  console.log("Exchange Metadata");
  console.log("-".repeat(30));

  const meta = await sdk.info.meta();
  const universe = meta.universe || [];
  console.log(`Total perp markets: ${universe.length}`);

  // Show a few markets
  for (const asset of universe.slice(0, 5)) {
    const name = asset.name || "?";
    const szDecimals = asset.szDecimals ?? "?";
    console.log(`  ${name}: ${szDecimals} size decimals`);
  }
  console.log();

  // ==========================================================================
  // User Account (requires a valid address)
  // ==========================================================================
  console.log("User Account");
  console.log("-".repeat(30));

  // Example address - replace with your address
  const userAddress = "0x0000000000000000000000000000000000000000";

  try {
    // Get clearinghouse state (positions, margin)
    const state = await sdk.info.clearinghouseState(userAddress);
    const equity = parseFloat(state.marginSummary?.accountValue || "0");
    console.log(`Account equity: $${equity.toLocaleString()}`);

    const positions = state.assetPositions || [];
    if (positions.length) {
      console.log(`Open positions: ${positions.length}`);
      for (const pos of positions.slice(0, 3)) {
        const coin = pos.position?.coin || "?";
        const size = pos.position?.szi || "0";
        const entry = pos.position?.entryPx || "?";
        const pnl = parseFloat(pos.position?.unrealizedPnl || "0");
        console.log(`  ${coin}: ${size} @ ${entry} (PnL: $${pnl.toLocaleString()})`);
      }
    } else {
      console.log("  No open positions");
    }
  } catch (e) {
    console.log(`  Could not fetch user data: ${e}`);
  }

  console.log();

  // ==========================================================================
  // Funding Rates
  // ==========================================================================
  console.log("Funding Rates");
  console.log("-".repeat(30));

  try {
    const fundings = await sdk.info.predictedFundings();
    console.log(`Predicted funding rates for ${fundings.length} assets:`);
    for (const f of fundings.slice(0, 5)) {
      const coin = (f as Record<string, unknown>).coin || "?";
      const rate = parseFloat(String((f as Record<string, unknown>).fundingRate || 0)) * 100;
      console.log(`  ${coin}: ${rate.toFixed(4)}%`);
    }
  } catch (e) {
    console.log(`  Could not fetch funding: ${e}`);
  }

  console.log();
  console.log("=".repeat(50));
  console.log("Done!");
}

main().catch(console.error);
