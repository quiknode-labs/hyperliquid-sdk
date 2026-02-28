#!/usr/bin/env npx ts-node
// @ts-nocheck
/**
 * Full Demo - Comprehensive example of all SDK capabilities.
 *
 * This example demonstrates all major SDK features:
 * - Info API (market data, user info)
 * - HyperCore API (blocks, trades, orders)
 * - EVM API (chain data, balances)
 * - WebSocket streaming
 * - gRPC streaming
 * - Trading (orders, positions)
 *
 * Requirements:
 *     npm install hyperliquid-sdk ws @grpc/grpc-js @grpc/proto-loader
 *
 * Usage:
 *     export ENDPOINT="https://your-endpoint.example.com/TOKEN"
 *     export PRIVATE_KEY="0x..."  # Optional, for trading
 *     npx ts-node full_demo.ts
 */

import { HyperliquidSDK, HyperliquidError } from '@quicknode/hyperliquid-sdk';

function separator(title: string) {
  console.log();
  console.log("=".repeat(60));
  console.log(`  ${title}`);
  console.log("=".repeat(60));
}

function subsection(title: string) {
  console.log();
  console.log(`--- ${title} ---`);
}

async function demoInfoApi(sdk: HyperliquidSDK) {
  separator("INFO API");

  subsection("Market Prices");
  const mids = await sdk.info.allMids();
  console.log(`Total markets: ${Object.keys(mids).length}`);
  for (const coin of ["BTC", "ETH", "SOL", "DOGE"]) {
    if (mids[coin]) {
      console.log(`  ${coin}: $${parseFloat(mids[coin]).toLocaleString()}`);
    }
  }

  subsection("Order Book");
  const book = await sdk.info.l2Book("BTC");
  const levels = book.levels || [[], []];
  const bids = levels[0] || [];
  const asks = levels[1] || [];
  if (bids.length && asks.length) {
    console.log(`  Best Bid: ${bids[0].sz} @ $${parseFloat(bids[0].px).toLocaleString()}`);
    console.log(`  Best Ask: ${asks[0].sz} @ $${parseFloat(asks[0].px).toLocaleString()}`);
    console.log(`  Spread: $${(parseFloat(asks[0].px) - parseFloat(bids[0].px)).toLocaleString()}`);
  }

  subsection("Recent Trades");
  const trades = await sdk.info.recentTrades("ETH");
  console.log("Last 3 ETH trades:");
  for (const t of trades.slice(0, 3)) {
    const trade = t as Record<string, unknown>;
    console.log(`  ${trade.sz} @ $${parseFloat(String(trade.px || 0)).toLocaleString()} (${trade.side})`);
  }

  subsection("Exchange Metadata");
  const meta = await sdk.info.meta();
  const universe = meta.universe || [];
  console.log(`Total perp markets: ${universe.length}`);

  subsection("Predicted Funding");
  const fundings = await sdk.info.predictedFundings();
  console.log("Top 3 funding rates:");
  const sortedFundings = [...fundings].sort((a, b) => {
    const aRate = Math.abs(parseFloat(String((a as Record<string, unknown>).fundingRate || 0)));
    const bRate = Math.abs(parseFloat(String((b as Record<string, unknown>).fundingRate || 0)));
    return bRate - aRate;
  });
  for (const f of sortedFundings.slice(0, 3)) {
    const funding = f as Record<string, unknown>;
    const rate = parseFloat(String(funding.fundingRate || 0)) * 100;
    console.log(`  ${funding.coin}: ${rate >= 0 ? '+' : ''}${rate.toFixed(4)}% (8h)`);
  }
}

async function demoHypercoreApi(sdk: HyperliquidSDK) {
  separator("HYPERCORE API");

  subsection("Latest Block");
  const blockNum = await sdk.core.latestBlockNumber();
  console.log(`Latest block: ${blockNum.toLocaleString()}`);

  const block = await sdk.core.getBlock(blockNum);
  if (block) {
    const b = block as Record<string, unknown>;
    const txs = (b.transactions as unknown[]) || [];
    console.log(`Block ${blockNum}: ${txs.length} transactions`);
  }

  subsection("Recent Trades");
  const trades = await sdk.core.latestTrades({ count: 5 });
  console.log("Last 5 trades across all markets:");
  for (const t of trades) {
    const trade = t as Record<string, unknown>;
    const coin = trade.coin || "?";
    console.log(`  ${coin}: ${trade.sz || '?'} @ $${parseFloat(String(trade.px || 0)).toLocaleString()}`);
  }

  subsection("Recent Orders");
  const orders = await sdk.core.latestOrders({ count: 5 });
  console.log("Last 5 orders:");
  for (const o of orders) {
    const order = o as Record<string, unknown>;
    const coin = order.coin || "?";
    const side = order.side || "?";
    const status = order.status || "?";
    console.log(`  ${coin}: ${side} @ $${parseFloat(String(order.limitPx || 0)).toLocaleString()} - ${status}`);
  }
}

async function demoEvmApi(sdk: HyperliquidSDK) {
  separator("EVM API");

  subsection("Chain Info");
  const chainId = await sdk.evm.chainId();
  const blockNum = await sdk.evm.blockNumber();
  const gasPrice = await sdk.evm.gasPrice();

  console.log(`Chain ID: ${chainId} (${chainId === 999 ? 'Mainnet' : 'Testnet'})`);
  console.log(`Block: ${blockNum.toLocaleString()}`);
  console.log(`Gas: ${(Number(gasPrice) / 1e9).toFixed(2)} Gwei`);

  subsection("Latest Block");
  const block = await sdk.evm.getBlockByNumber(blockNum);
  if (block) {
    const b = block as Record<string, unknown>;
    console.log(`Block ${blockNum}:`);
    console.log(`  Hash: ${String(b.hash || '?').slice(0, 30)}...`);
    console.log(`  Gas Used: ${parseInt(String(b.gasUsed || '0x0'), 16).toLocaleString()}`);
  }
}

async function demoWebsocket(sdk: HyperliquidSDK, duration: number = 5) {
  separator("WEBSOCKET STREAMING");

  let tradeCount = 0;
  let bookCount = 0;

  const onTrade = (data: Record<string, unknown>) => {
    tradeCount++;
    if (tradeCount <= 3) {
      const d = data.data as Record<string, unknown> || {};
      console.log(`  [TRADE] ${d.coin || '?'}: ${d.sz || '?'} @ ${d.px || '?'}`);
    }
  };

  const onBook = (data: Record<string, unknown>) => {
    bookCount++;
    if (bookCount <= 2) {
      const d = data.data as Record<string, unknown> || {};
      console.log(`  [BOOK] ${d.coin || '?'} update`);
    }
  };

  console.log(`Streaming for ${duration} seconds...`);

  sdk.stream.trades(["BTC", "ETH"], onTrade);
  sdk.stream.bookUpdates(["BTC"], onBook);

  // Run in background
  await sdk.stream.start();
  await new Promise(resolve => setTimeout(resolve, duration * 1000));
  sdk.stream.stop();

  console.log();
  console.log(`Received: ${tradeCount} trades, ${bookCount} book updates`);
}

async function demoGrpc(endpoint: string, duration: number = 5) {
  separator("GRPC STREAMING");

  let tradeCount = 0;

  // Create new SDK for gRPC demo
  const sdk = new HyperliquidSDK(endpoint);

  const onTrade = (data: Record<string, unknown>) => {
    tradeCount++;
    if (tradeCount <= 3) {
      console.log(`  [TRADE] ${JSON.stringify(data)}`);
    }
  };

  console.log(`Streaming for ${duration} seconds...`);

  try {
    sdk.grpc.trades(["BTC", "ETH"], onTrade);

    // Run in background
    await sdk.grpc.start();
    await new Promise(resolve => setTimeout(resolve, duration * 1000));
    sdk.grpc.stop();

    console.log();
    console.log(`Received: ${tradeCount} trades`);
  } catch (e) {
    console.log("  gRPC not available. Install: npm install @grpc/grpc-js @grpc/proto-loader");
  }
}

async function demoTrading(endpoint: string, privateKey: string) {
  separator("TRADING");

  const sdk = new HyperliquidSDK(endpoint, { privateKey });

  console.log(`Address: ${sdk.address}`);
  console.log(`Endpoint: ${endpoint.slice(0, 50)}...`);

  subsection("Account Check");
  try {
    console.log("  Trading SDK initialized successfully");
    console.log("  Ready to place orders (not executing in demo)");
  } catch (e) {
    console.log(`  Note: ${e}`);
  }

  subsection("Order Building (Example)");
  console.log("  Market buy: await sdk.marketBuy('BTC', { notional: 100 })");
  console.log("  Limit sell: await sdk.sell('ETH', { size: 1.0, price: 4000 })");
  console.log("  Close pos:  await sdk.closePosition('BTC')");
}

async function main() {
  console.log();
  console.log("*".repeat(60));
  console.log("  HYPERLIQUID SDK - FULL DEMO");
  console.log("*".repeat(60));

  const endpoint = process.argv[2] || process.env.ENDPOINT;
  const privateKey = process.env.PRIVATE_KEY;

  if (!endpoint) {
    console.log();
    console.log("Error: ENDPOINT not set");
    console.log();
    console.log("Usage:");
    console.log("  export ENDPOINT='https://your-endpoint.example.com/TOKEN'");
    console.log("  npx ts-node full_demo.ts");
    console.log();
    console.log("Or:");
    console.log("  npx ts-node full_demo.ts 'https://your-endpoint.example.com/TOKEN'");
    process.exit(1);
  }

  console.log();
  console.log(`Endpoint: ${endpoint.slice(0, 50)}...`);

  // Create single SDK instance for most demos
  const sdk = new HyperliquidSDK(endpoint);

  // Run all demos
  try {
    await demoInfoApi(sdk);
    await demoHypercoreApi(sdk);
    await demoEvmApi(sdk);
    await demoWebsocket(sdk, 5);
    await demoGrpc(endpoint, 5);

    if (privateKey) {
      await demoTrading(endpoint, privateKey);
    } else {
      console.log();
      console.log("--- TRADING (skipped - no PRIVATE_KEY) ---");
    }
  } catch (e) {
    if (e instanceof HyperliquidError) {
      console.log(`\nError: ${e.message}`);
      console.log(`Code: ${e.code}`);
      process.exit(1);
    }
    throw e;
  }

  separator("DONE");
  console.log("All demos completed successfully!");
  console.log();
}

main().catch(console.error);
