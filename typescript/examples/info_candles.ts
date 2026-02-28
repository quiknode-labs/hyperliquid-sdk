#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Historical Candles Example
 *
 * Shows how to fetch historical candlestick (OHLCV) data.
 *
 * Note: candleSnapshot may not be available on all QuickNode endpoints.
 * Check the QuickNode docs for method availability.
 *
 * Usage:
 *     export QUICKNODE_ENDPOINT="https://your-endpoint.hype-mainnet.quiknode.pro/YOUR_TOKEN"
 *     npx ts-node info_candles.ts
 */

import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const ENDPOINT = process.env.QUICKNODE_ENDPOINT;

if (!ENDPOINT) {
  console.error("Set QUICKNODE_ENDPOINT environment variable");
  process.exit(1);
}

async function main() {
  // Single SDK instance — access everything through sdk.info, sdk.core, sdk.evm, etc.
  const sdk = new HyperliquidSDK(ENDPOINT);
  const info = sdk.info;

  console.log("=".repeat(50));
  console.log("Historical Candles");
  console.log("=".repeat(50));

  // Last 24 hours
  const now = Date.now();
  const dayAgo = now - (24 * 60 * 60 * 1000);

  // Fetch BTC 1-hour candles
  console.log("\n1. BTC 1-Hour Candles (last 24h):");
  try {
    const candles = await info.candles("BTC", "1h", dayAgo, now);
    console.log(`   Retrieved ${candles.length} candles`);
    if (candles.length > 0) {
      for (const c of candles.slice(-3)) {
        console.log(`   O:${c.o} H:${c.h} L:${c.l} C:${c.c}`);
      }
    }
  } catch (e) {
    console.log(`   Error: ${e}`);
    console.log("   Note: candleSnapshot may not be available on this endpoint");
  }

  // Predicted funding rates (supported on QuickNode)
  console.log("\n2. Predicted Funding Rates:");
  try {
    const fundings = await info.predictedFundings();
    console.log(`   ${fundings.length} assets with funding rates:`);
    // Structure: [[coin, [[source, {fundingRate, ...}], ...]], ...]
    for (const item of fundings.slice(0, 5)) {
      if (Array.isArray(item) && item.length >= 2) {
        const coin = item[0];
        const sources = item[1];
        if (sources && Array.isArray(sources) && sources.length > 0) {
          // Get HlPerp funding rate if available
          for (const src of sources) {
            if (Array.isArray(src) && src.length >= 2 && src[0] === "HlPerp") {
              const rate = parseFloat(src[1].fundingRate || "0") * 100;
              console.log(`   ${coin}: ${rate.toFixed(4)}%`);
              break;
            }
          }
        }
      }
    }
  } catch (e) {
    console.log(`   Error: ${e}`);
  }

  console.log("\n" + "=".repeat(50));
  console.log("Done!");
}

main().catch(console.error);
