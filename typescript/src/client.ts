/**
 * Hyperliquid SDK Client — The magic happens here.
 *
 * One class to rule them all:
 * - Auto build → sign → send (no ceremony)
 * - Auto approval management
 * - Smart defaults everywhere
 * - Full power when you need it
 * - Unified access: sdk.info, sdk.core, sdk.evm, sdk.stream, sdk.grpc
 */

import { Wallet } from 'ethers';

import { Order, PlacedOrder, Side, TIF, TriggerOrder, OrderGrouping } from './order';
import {
  HyperliquidError,
  BuildError,
  ValidationError,
  GeoBlockedError,
  SignerError,
  parseApiError,
} from './errors';
import { Info } from './info';
import { HyperCore } from './hypercore';
import { EVM } from './evm';
import {
  AssetInput,
  OutcomeAmount,
  PredictionMarket,
  PredictionMarketFilter,
  PredictionSide,
  assetToString,
} from './types';

// Import stream types conditionally (they depend on 'ws')
type StreamType = import('./websocket').Stream;
type GRPCStreamType = import('./grpc-stream').GRPCStream;
type EVMStreamType = import('./evm-stream').EVMStream;

function parseOutcomeDescription(description: string): Record<string, string> {
  const fields: Record<string, string> = {};
  for (const part of description.split('|')) {
    const index = part.indexOf(':');
    if (index === -1) continue;
    fields[part.slice(0, index)] = part.slice(index + 1);
  }
  return fields;
}

function formatExpiry(expiry?: string): string | undefined {
  if (!expiry || expiry.length !== 13 || expiry[8] !== '-') return expiry;
  return `${expiry.slice(0, 4)}-${expiry.slice(4, 6)}-${expiry.slice(6, 8)}T${expiry.slice(9, 11)}:${expiry.slice(11, 13)}:00Z`;
}

function slugify(value: string): string {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

// Only used to generate app-style slug aliases such as
// btc-above-78213-yes-may-04-0600 from outcomeMeta expiry 20260504-0600.
const APP_STYLE_PREDICTION_SLUG_MONTHS: Record<string, string> = {
  '01': 'jan',
  '02': 'feb',
  '03': 'mar',
  '04': 'apr',
  '05': 'may',
  '06': 'jun',
  '07': 'jul',
  '08': 'aug',
  '09': 'sep',
  '10': 'oct',
  '11': 'nov',
  '12': 'dec',
};

function appStylePredictionSlug(fields: Record<string, string>, side?: string): string | undefined {
  const underlying = fields.underlying;
  const targetPrice = fields.targetPrice;
  const expiry = fields.expiry;
  if (!underlying || !targetPrice || !expiry || expiry.length !== 13) return undefined;
  const month = APP_STYLE_PREDICTION_SLUG_MONTHS[expiry.slice(4, 6)];
  if (!month) return undefined;
  const parts = [underlying, 'above', targetPrice];
  if (side) parts.push(side);
  parts.push(month, expiry.slice(6, 8), expiry.slice(9, 13));
  return slugify(parts.join('-'));
}

function titleFromFields(fields: Record<string, string>): string {
  const underlying = fields.underlying ?? 'Outcome';
  const targetPrice = fields.targetPrice;
  const expiry = formatExpiry(fields.expiry);
  if (targetPrice && expiry) return `${underlying} above ${targetPrice} on ${expiry}`;
  if (targetPrice) return `${underlying} above ${targetPrice}`;
  return underlying;
}

function predictionMarketMatches(market: PredictionMarket, query: string): boolean {
  const normalized = query.toLowerCase();
  const values = [
    market.slug,
    market.title.toLowerCase(),
    market.name.toLowerCase(),
    market.underlying?.toLowerCase() ?? '',
    market.yes.symbol.toLowerCase(),
    market.no.symbol.toLowerCase(),
    market.yes.token.toLowerCase(),
    market.no.token.toLowerCase(),
    ...market.aliases,
  ];
  return values.some((value) => value === normalized || value.includes(normalized));
}

export class PredictionMarkets extends Array<PredictionMarket> {
  findMarket(queryOrFilter?: string | PredictionMarketFilter): PredictionMarket | undefined {
    const filter: PredictionMarketFilter =
      queryOrFilter !== undefined && typeof queryOrFilter === 'object' ? queryOrFilter : {};
    const query = typeof queryOrFilter === 'string' ? queryOrFilter : filter.query;

    return this.find((market) => {
      if (query !== undefined && !predictionMarketMatches(market, query)) return false;
      if (filter.underlying !== undefined && (market.underlying ?? '').toLowerCase() !== filter.underlying.toLowerCase()) return false;
      if (filter.targetPrice !== undefined && market.targetPrice !== String(filter.targetPrice)) return false;
      if (filter.expiry !== undefined && market.expiry !== filter.expiry && market.expiry !== formatExpiry(filter.expiry)) return false;
      return true;
    });
  }
}

function buildPredictionMarkets(outcomes: Array<Record<string, unknown>>, mids: Record<string, string>): PredictionMarkets {
  const markets = new PredictionMarkets();
  for (const outcome of outcomes) {
    const outcomeId = Number(outcome.outcome);
    const description = String(outcome.description ?? '');
    const fields = parseOutcomeDescription(description);
    const title = titleFromFields(fields);
    const sideSpecs = (outcome.sideSpecs as Array<Record<string, unknown>> | undefined) ?? [];

    const sides = sideSpecs.map((sideSpec, sideIndex): PredictionSide => {
      const encoding = outcomeId * 10 + sideIndex;
      const symbol = `#${encoding}`;
      return {
        outcome: outcomeId,
        side: sideIndex,
        name: String(sideSpec.name ?? sideIndex),
        symbol,
        token: `+${encoding}`,
        assetId: 100000000 + encoding,
        mid: mids[symbol],
        szDecimals: 0,
        supportsPriorityFee: false,
      };
    });

    if (sides.length < 2) continue;

    const slug = appStylePredictionSlug(fields) ?? slugify(title);
    const aliases = [
      slugify(title),
      appStylePredictionSlug(fields, sides[0].name),
      appStylePredictionSlug(fields, sides[1].name),
    ].filter((alias): alias is string => alias !== undefined);

    markets.push({
      outcome: outcomeId,
      name: String(outcome.name ?? ''),
      description,
      title,
      slug,
      underlying: fields.underlying,
      targetPrice: fields.targetPrice,
      expiry: formatExpiry(fields.expiry),
      period: fields.period,
      collateral: 'USDH',
      minOrderValue: '10',
      aliases,
      yes: sides[0],
      no: sides[1],
      sides,
    });
  }
  return markets;
}

/** ECDSA signature components returned by a signer. `v` is 27 or 28. */
export interface Signature {
  r: string;
  s: string;
  v: number;
}

/**
 * Signs the 32-byte order hash produced by Hyperliquid's build step.
 *
 * `hashHex` may be 0x-prefixed. May be sync or async. The external-signing
 * analog of an in-process wallet, letting callers sign via a remote
 * KMS/HSM/signing service without exposing a private key to the SDK. The
 * provided `signal` is an AbortSignal bounded by the SDK timeout, so a slow
 * remote signer is cancelled along with the rest of the request — honour it.
 */
export type Signer = (
  hashHex: string,
  options?: { signal?: AbortSignal }
) => Signature | Promise<Signature>;

/**
 * Runtime guard for a signature returned by a user-supplied {@link Signer}.
 * The callback has no compile-time guarantee at runtime (callers may be plain
 * JS), so a nil/malformed result is rejected before it reaches the venue.
 */
function isValidSignature(sig: unknown): sig is Signature {
  if (typeof sig !== 'object' || sig === null) return false;
  const s = sig as Signature;
  return (
    typeof s.r === 'string' && s.r !== '' &&
    typeof s.s === 'string' && s.s !== '' &&
    (s.v === 27 || s.v === 28)
  );
}

export interface HyperliquidSDKOptions {
  /** Hex private key (with or without 0x). Falls back to PRIVATE_KEY env var. */
  privateKey?: string;
  /**
   * External signing callback (KMS/HSM/remote service). When set, the SDK is
   * sign-capable without an in-process wallet, no private key is read (including
   * the PRIVATE_KEY env var), and builder-fee auto-approval is skipped. Mutually
   * exclusive with privateKey.
   */
  signer?: Signer;
  /** Acting agent address used when signing via an external signer. Exposed as sdk.address. */
  signerAddress?: string;
  /** Use testnet (default: false). */
  testnet?: boolean;
  /** Automatically approve builder fee for trading (default: true). Skipped under an external signer. */
  autoApprove?: boolean;
  /** Max builder fee to approve (default: "1%"). */
  maxFee?: string;
  /** Default slippage for market orders (default: 0.03 = 3%). */
  slippage?: number;
  /** Request timeout in milliseconds (default: 30000). */
  timeout?: number;
}

/**
 * Hyperliquid SDK — Stupidly simple, insanely powerful.
 *
 * Unified access to ALL Hyperliquid APIs through a single SDK instance.
 * Requests route through your QuickNode endpoint when available. Worker-only
 * operations (/approval, /markets, /preflight) and unsupported Info methods
 * route through the public worker (send.hyperliquidapi.com).
 *
 * Examples:
 *     // Initialize with QuickNode endpoint (required):
 *     const sdk = new HyperliquidSDK("https://your-endpoint.quiknode.pro/TOKEN");
 *
 *     // Access everything through the SDK:
 *     await sdk.info.meta();                    // Info API
 *     await sdk.info.clearinghouseState("0x...");
 *     await sdk.core.latestBlockNumber();       // HyperCore
 *     await sdk.evm.blockNumber();              // HyperEVM
 *     sdk.stream.trades(["BTC"], cb);           // WebSocket
 *     sdk.grpc.trades(["BTC"], cb);             // gRPC
 *
 *     // TRADING — Add private key:
 *     const sdk = new HyperliquidSDK("https://...", { privateKey: "0x..." });
 *
 *     // Or use environment variable:
 *     // export PRIVATE_KEY=0x...
 *     const sdk = new HyperliquidSDK("https://...");
 *
 *     // Then trade:
 *     await sdk.marketBuy("BTC", { size: 0.001 });
 *     await sdk.buy("BTC", { size: 0.001, price: 67000 });
 *     await sdk.closePosition("BTC");
 *     await sdk.cancelAll();
 *
 *     // Read-only operations (no private key):
 *     const sdk = new HyperliquidSDK("https://...");
 *     await sdk.markets();     // Get all markets
 *     await sdk.getMid("BTC"); // Get mid price
 */
export class HyperliquidSDK {
  static readonly DEFAULT_SLIPPAGE = 0.03; // 3% for market orders
  static readonly DEFAULT_TIMEOUT = 30000; // milliseconds
  static readonly CACHE_TTL = 300000; // 5 minutes in ms

  /** Default URL - Worker handles requests and routes to Hyperliquid */
  static readonly DEFAULT_WORKER_URL = 'https://send.hyperliquidapi.com';

  /** Info API methods supported by QuickNode nodes with --serve-info-endpoint */
  static readonly QN_SUPPORTED_INFO_METHODS = new Set([
    'meta', 'spotMeta', 'clearinghouseState', 'spotClearinghouseState',
    'openOrders', 'exchangeStatus', 'frontendOpenOrders', 'liquidatable',
    'activeAssetData', 'maxMarketOrderNtls', 'vaultSummaries', 'userVaultEquities',
    'leadingVaults', 'extraAgents', 'subAccounts', 'userFees', 'userRateLimit',
    'spotDeployState', 'perpDeployAuctionStatus', 'delegations', 'delegatorSummary',
    'maxBuilderFee', 'userToMultiSigSigners', 'userRole', 'perpsAtOpenInterestCap',
    'validatorL1Votes', 'marginTable', 'perpDexs', 'webData2', 'outcomeMeta',
  ]);

  private readonly _endpoint?: string;
  private readonly _timeout: number;
  private readonly _slippage: number;
  private readonly _maxFee: string;
  private readonly _autoApprove: boolean;
  private readonly _testnet: boolean;
  private readonly _chain: string;
  private readonly _chainId: string;

  private readonly _wallet: Wallet | null;
  private readonly _signer: Signer | null;
  readonly address: string | null;

  private readonly _publicWorkerUrl: string;
  private readonly _exchangeUrl: string;
  private readonly _infoUrl: string;

  // Cache
  private _marketsCache: Record<string, unknown> | null = null;
  private _marketsCacheTime: number = 0;
  private _szDecimalsCache: Map<string, number> = new Map();

  // Lazy-initialized sub-clients
  private _info: Info | null = null;
  private _core: HyperCore | null = null;
  private _evm: EVM | null = null;
  private _stream: StreamType | null = null;
  private _grpc: GRPCStreamType | null = null;
  private _evmStream: EVMStreamType | null = null;

  constructor(endpoint?: string, options: HyperliquidSDKOptions = {}) {
    this._endpoint = endpoint;
    this._timeout = options.timeout ?? HyperliquidSDK.DEFAULT_TIMEOUT;
    this._slippage = options.slippage ?? HyperliquidSDK.DEFAULT_SLIPPAGE;
    this._maxFee = options.maxFee ?? '1%';
    this._autoApprove = options.autoApprove ?? true;
    this._testnet = options.testnet ?? false;

    // Chain configuration
    this._chain = this._testnet ? 'Testnet' : 'Mainnet';
    this._chainId = this._testnet ? '0x66eee' : '0xa4b1';

    // An external signer takes precedence: the SDK is sign-capable without an
    // in-process wallet and never reads a private key (the key must never enter
    // the SDK process).
    this._signer = options.signer ?? null;

    if (this._signer) {
      this._wallet = null;
      this.address = options.signerAddress ?? null;
    } else {
      const pk = options.privateKey ?? (typeof process !== 'undefined' ? process.env?.PRIVATE_KEY : undefined);
      if (pk) {
        this._wallet = new Wallet(pk);
        this.address = this._wallet.address;
      } else {
        this._wallet = null;
        this.address = null;
      }
    }

    // Build URLs
    this._publicWorkerUrl = HyperliquidSDK.DEFAULT_WORKER_URL;

    if (endpoint) {
      const baseUrl = this._buildBaseUrl(endpoint);
      this._infoUrl = `${baseUrl}/info`;
    } else {
      this._infoUrl = `${HyperliquidSDK.DEFAULT_WORKER_URL}/info`;
    }

    // Trading/exchange ALWAYS goes through the public worker
    // QuickNode /send endpoint is not used for trading
    this._exchangeUrl = `${HyperliquidSDK.DEFAULT_WORKER_URL}/exchange`;

    // Auto-approve will be called on first trade if needed
  }

  private _buildBaseUrl(url: string): string {
    const parsed = new URL(url);
    const base = `${parsed.protocol}//${parsed.host}`;
    const pathParts = parsed.pathname.split('/').filter((p) => p.length > 0);

    const knownPaths = new Set(['info', 'hypercore', 'evm', 'nanoreth', 'ws', 'send']);

    let token: string | null = null;
    for (const part of pathParts) {
      if (!knownPaths.has(part)) {
        token = part;
        break;
      }
    }

    if (token) {
      return `${base}/${token}`;
    }
    return base;
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // CONFIGURATION PROPERTIES
  // ═══════════════════════════════════════════════════════════════════════════

  get testnet(): boolean {
    return this._testnet;
  }

  get chain(): string {
    return this._chain;
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // UNIFIED SUB-CLIENTS — Lazy initialization
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Info API — Market data, positions, orders, vaults.
   *
   * Examples:
   *     await sdk.info.meta();
   *     await sdk.info.allMids();
   *     await sdk.info.clearinghouseState("0x...");
   *     await sdk.info.openOrders("0x...");
   */
  get info(): Info {
    if (this._info === null) {
      if (!this._endpoint) {
        throw new Error(
          'Endpoint required for Info API. Pass endpoint to HyperliquidSDK() constructor.'
        );
      }
      this._info = new Info(this._endpoint, { timeout: this._timeout });
    }
    return this._info;
  }

  /**
   * HyperCore API — Blocks, trades, orders from the L1.
   *
   * Examples:
   *     await sdk.core.latestBlockNumber();
   *     await sdk.core.latestTrades({ count: 10 });
   *     await sdk.core.getBlock(12345);
   */
  get core(): HyperCore {
    if (this._core === null) {
      if (!this._endpoint) {
        throw new Error(
          'Endpoint required for HyperCore API. Pass endpoint to HyperliquidSDK() constructor.'
        );
      }
      this._core = new HyperCore(this._endpoint, { timeout: this._timeout });
    }
    return this._core;
  }

  /**
   * HyperEVM API — Ethereum JSON-RPC for HyperEVM.
   *
   * Examples:
   *     await sdk.evm.blockNumber();
   *     await sdk.evm.getBalance("0x...");
   *     await sdk.evm.call(tx);
   */
  get evm(): EVM {
    if (this._evm === null) {
      if (!this._endpoint) {
        throw new Error(
          'Endpoint required for EVM API. Pass endpoint to HyperliquidSDK() constructor.'
        );
      }
      this._evm = new EVM(this._endpoint, { timeout: this._timeout });
    }
    return this._evm;
  }

  /**
   * WebSocket Streaming — Real-time trades, orderbook, orders.
   *
   * Examples:
   *     sdk.stream.trades(["BTC"], (t) => console.log(t));
   *     sdk.stream.l2Book("ETH", callback);
   *     await sdk.stream.start();
   */
  get stream(): StreamType {
    if (this._stream === null) {
      if (!this._endpoint) {
        throw new Error(
          'Endpoint required for WebSocket streaming. Pass endpoint to HyperliquidSDK() constructor.'
        );
      }
      // Dynamic import to avoid requiring 'ws' if not used
      // eslint-disable-next-line @typescript-eslint/no-require-imports
      const { Stream } = require('./websocket');
      this._stream = new Stream(this._endpoint);
    }
    return this._stream!;
  }

  /**
   * gRPC Streaming — High-performance real-time data.
   *
   * Examples:
   *     sdk.grpc.trades(["BTC"], (t) => console.log(t));
   *     sdk.grpc.l2Book("ETH", callback);
   *     await sdk.grpc.start();
   */
  get grpc(): GRPCStreamType {
    if (this._grpc === null) {
      if (!this._endpoint) {
        throw new Error(
          'Endpoint required for gRPC streaming. Pass endpoint to HyperliquidSDK() constructor.'
        );
      }
      // Dynamic import to avoid requiring '@grpc/grpc-js' if not used
      // eslint-disable-next-line @typescript-eslint/no-require-imports
      const { GRPCStream } = require('./grpc-stream');
      this._grpc = new GRPCStream(this._endpoint) as GRPCStreamType;
    }
    return this._grpc!;
  }

  /**
   * EVM WebSocket Streaming — eth_subscribe/eth_unsubscribe.
   *
   * Stream EVM events via WebSocket on the /nanoreth namespace.
   *
   * Examples:
   *     sdk.evmStream.newHeads((h) => console.log(h));
   *     sdk.evmStream.logs({ address: "0x..." }, callback);
   *     await sdk.evmStream.start();
   */
  get evmStream(): EVMStreamType {
    if (this._evmStream === null) {
      if (!this._endpoint) {
        throw new Error(
          'Endpoint required for EVM WebSocket streaming. Pass endpoint to HyperliquidSDK() constructor.'
        );
      }
      // Dynamic import
      // eslint-disable-next-line @typescript-eslint/no-require-imports
      const { EVMStream } = require('./evm-stream');
      this._evmStream = new EVMStream(this._endpoint) as EVMStreamType;
    }
    return this._evmStream!;
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // ORDER PLACEMENT — The Simple Way
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Place a buy order.
   *
   * @param asset - Asset to buy ("BTC", "ETH", "xyz:SILVER", or market.yes/no)
   * @param options.size - Size in asset units
   * @param options.notional - Size in USD (alternative to size)
   * @param options.price - Limit price (omit for market order)
   * @param options.tif - Time in force ("ioc", "gtc", "alo", "market")
   * @param options.reduceOnly - Close position only, no new exposure
   * @param options.grouping - Order grouping for TP/SL attachment
   * @param options.priorityFee - Optional Hyperliquid order priority p. p=10000 is 1 bp.
   *
   * @returns PlacedOrder with oid, status, and cancel/modify methods
   */
  async buy(
    asset: AssetInput,
    options: {
      size?: number | string;
      notional?: number;
      price?: number | string;
      tif?: TIF | string;
      reduceOnly?: boolean;
      grouping?: OrderGrouping;
      priorityFee?: number | string;
      /** Slippage tolerance for market orders (e.g. 0.05 = 5%). Overrides SDK default. */
      slippage?: number;
    } = {}
  ): Promise<PlacedOrder> {
    return this._placeOrder({
      asset,
      side: Side.BUY,
      size: options.size,
      notional: options.notional,
      price: options.price,
      tif: options.tif ?? TIF.IOC,
      reduceOnly: options.reduceOnly ?? false,
      grouping: options.grouping ?? OrderGrouping.NA,
      slippage: options.slippage,
      priorityFee: options.priorityFee,
    });
  }

  /**
   * Place a sell order.
   */
  async sell(
    asset: AssetInput,
    options: {
      size?: number | string;
      notional?: number;
      price?: number | string;
      tif?: TIF | string;
      reduceOnly?: boolean;
      grouping?: OrderGrouping;
      priorityFee?: number | string;
      /** Slippage tolerance for market orders (e.g. 0.05 = 5%). Overrides SDK default. */
      slippage?: number;
    } = {}
  ): Promise<PlacedOrder> {
    return this._placeOrder({
      asset,
      side: Side.SELL,
      size: options.size,
      notional: options.notional,
      price: options.price,
      tif: options.tif ?? TIF.IOC,
      reduceOnly: options.reduceOnly ?? false,
      grouping: options.grouping ?? OrderGrouping.NA,
      slippage: options.slippage,
      priorityFee: options.priorityFee,
    });
  }

  // Aliases for perp traders
  long = this.buy;
  short = this.sell;

  /**
   * Market buy — executes immediately at best available price.
   * @param options.slippage - Slippage tolerance (e.g. 0.05 = 5%). Default: SDK setting (3%).
   */
  async marketBuy(
    asset: AssetInput,
    options: { size?: number | string; notional?: number; slippage?: number; priorityFee?: number | string } = {}
  ): Promise<PlacedOrder> {
    return this.buy(asset, { ...options, tif: 'market' });
  }

  /**
   * Market sell — executes immediately at best available price.
   * @param options.slippage - Slippage tolerance (e.g. 0.05 = 5%). Default: SDK setting (3%).
   */
  async marketSell(
    asset: AssetInput,
    options: { size?: number | string; notional?: number; slippage?: number; priorityFee?: number | string } = {}
  ): Promise<PlacedOrder> {
    return this.sell(asset, { ...options, tif: 'market' });
  }

  /**
   * Place an order using the fluent Order builder.
   *
   * @param order - Order built with Order.buy() or Order.sell()
   */
  async order(order: Order): Promise<PlacedOrder> {
    order.validate();

    // Handle notional orders
    if (order['_notional'] && !order['_size']) {
      const mid = await this.getMid(order.asset);
      if (mid === 0) {
        throw new ValidationError(`Could not fetch price for ${order.asset}`, {
          guidance: 'Check the asset name or try again.',
        });
      }
      const szDecimals = await this._getSizeDecimals(order.asset);
      const factor = Math.pow(10, szDecimals);
      const rawSize = order['_notional'] / mid;
      const size = this._isPredictionAsset(order.asset)
        ? Math.ceil(rawSize)
        : Math.round(rawSize * factor) / factor;
      order['_size'] = String(size);
    }

    return this._executeOrder(order);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // TRIGGER ORDERS (Stop Loss / Take Profit)
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Place a stop-loss trigger order.
   */
  async stopLoss(
    asset: string,
    options: {
      size: number | string;
      triggerPrice: number | string;
      limitPrice?: number | string;
      side?: Side;
      reduceOnly?: boolean;
      grouping?: OrderGrouping;
    }
  ): Promise<PlacedOrder> {
    const trigger = TriggerOrder.stopLoss(asset, { side: options.side ?? Side.SELL });
    trigger.size(options.size);
    trigger.triggerPrice(options.triggerPrice);
    if (options.limitPrice !== undefined) {
      trigger.limit(options.limitPrice);
    } else {
      trigger.market();
    }
    trigger.reduceOnly(options.reduceOnly ?? true);

    return this._executeTriggerOrder(trigger, options.grouping ?? OrderGrouping.NA);
  }

  /**
   * Place a take-profit trigger order.
   */
  async takeProfit(
    asset: string,
    options: {
      size: number | string;
      triggerPrice: number | string;
      limitPrice?: number | string;
      side?: Side;
      reduceOnly?: boolean;
      grouping?: OrderGrouping;
    }
  ): Promise<PlacedOrder> {
    const trigger = TriggerOrder.takeProfit(asset, { side: options.side ?? Side.SELL });
    trigger.size(options.size);
    trigger.triggerPrice(options.triggerPrice);
    if (options.limitPrice !== undefined) {
      trigger.limit(options.limitPrice);
    } else {
      trigger.market();
    }
    trigger.reduceOnly(options.reduceOnly ?? true);

    return this._executeTriggerOrder(trigger, options.grouping ?? OrderGrouping.NA);
  }

  // Aliases
  sl = this.stopLoss;
  tp = this.takeProfit;

  /**
   * Place a trigger order using the fluent TriggerOrder builder.
   */
  async triggerOrder(
    trigger: TriggerOrder,
    grouping: OrderGrouping = OrderGrouping.NA
  ): Promise<PlacedOrder> {
    return this._executeTriggerOrder(trigger, grouping);
  }

  private async _executeTriggerOrder(
    trigger: TriggerOrder,
    grouping: OrderGrouping = OrderGrouping.NA
  ): Promise<PlacedOrder> {
    trigger.validate();
    const action = trigger.toAction(grouping);
    const result = await this._buildSignSend(action);

    const order = new Order(trigger.asset, trigger.side);
    order['_size'] = trigger['_size'];
    order['_price'] = trigger['_limitPx'] ?? trigger['_triggerPx'];

    return PlacedOrder.fromResponse(
      (result as Record<string, unknown>).exchangeResponse as Record<string, unknown> ?? {},
      order,
      this
    );
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // TWAP ORDERS (Time-Weighted Average Price)
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Place a TWAP (Time-Weighted Average Price) order.
   */
  async twapOrder(
    asset: string | number,
    options: {
      size: number | string;
      isBuy: boolean;
      durationMinutes: number;
      reduceOnly?: boolean;
      randomize?: boolean;
    }
  ): Promise<Record<string, unknown>> {
    const assetIdx = typeof asset === 'string' ? await this._resolveAssetIndex(asset) : asset;

    const action = {
      type: 'twapOrder',
      twap: {
        a: assetIdx,
        b: options.isBuy,
        s: String(options.size),
        r: options.reduceOnly ?? false,
        m: options.durationMinutes,
        t: options.randomize ?? true,
      },
    };
    return this._buildSignSend(action);
  }

  /**
   * Cancel an active TWAP order.
   */
  async twapCancel(asset: string | number, twapId: number): Promise<Record<string, unknown>> {
    const assetIdx = typeof asset === 'string' ? await this._resolveAssetIndex(asset) : asset;

    const action = {
      type: 'twapCancel',
      a: assetIdx,
      t: twapId,
    };
    return this._buildSignSend(action);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // LEVERAGE MANAGEMENT
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Update leverage for an asset.
   */
  async updateLeverage(
    asset: string | number,
    leverage: number,
    options: { isCross?: boolean } = {}
  ): Promise<Record<string, unknown>> {
    const assetIdx = typeof asset === 'string' ? await this._resolveAssetIndex(asset) : asset;

    const action = {
      type: 'updateLeverage',
      asset: assetIdx,
      isCross: options.isCross ?? true,
      leverage,
    };
    return this._buildSignSend(action);
  }

  /**
   * Add or remove margin from an isolated position.
   */
  async updateIsolatedMargin(
    asset: string | number,
    options: { isBuy: boolean; amount: number }
  ): Promise<Record<string, unknown>> {
    const assetIdx = typeof asset === 'string' ? await this._resolveAssetIndex(asset) : asset;
    const ntli = Math.floor(options.amount * 1_000_000);

    const action = {
      type: 'updateIsolatedMargin',
      asset: assetIdx,
      isBuy: options.isBuy,
      ntli,
    };
    return this._buildSignSend(action);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // TRANSFER OPERATIONS
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Transfer USDC to another Hyperliquid address.
   */
  async transferUsd(
    destination: string,
    amount: number | string
  ): Promise<Record<string, unknown>> {
    const action = {
      type: 'usdSend',
      hyperliquidChain: this._chain,
      signatureChainId: this._chainId,
      destination,
      amount: String(amount),
      time: Date.now(),
    };
    return this._buildSignSend(action);
  }

  /**
   * Transfer spot tokens to another Hyperliquid address.
   */
  async transferSpot(
    token: string,
    destination: string,
    amount: number | string
  ): Promise<Record<string, unknown>> {
    const action = {
      type: 'spotSend',
      hyperliquidChain: this._chain,
      signatureChainId: this._chainId,
      token,
      destination,
      amount: String(amount),
      time: Date.now(),
    };
    return this._buildSignSend(action);
  }

  /**
   * Initiate a withdrawal to Arbitrum.
   */
  async withdraw(
    amount: number | string,
    destination?: string
  ): Promise<Record<string, unknown>> {
    if (destination === undefined) {
      this._requireWallet();
      destination = this.address!;
    }

    const action = {
      type: 'withdraw3',
      hyperliquidChain: this._chain,
      signatureChainId: this._chainId,
      destination,
      amount: String(amount),
      time: Date.now(),
    };
    return this._buildSignSend(action);
  }

  /**
   * Transfer USDC from spot balance to perp balance.
   */
  async transferSpotToPerp(amount: number | string): Promise<Record<string, unknown>> {
    const nonce = Date.now();
    const action = {
      type: 'usdClassTransfer',
      hyperliquidChain: this._chain,
      signatureChainId: this._chainId,
      amount: String(amount),
      toPerp: true,
      nonce,
    };
    return this._buildSignSend(action);
  }

  /**
   * Transfer USDC from perp balance to spot balance.
   */
  async transferPerpToSpot(amount: number | string): Promise<Record<string, unknown>> {
    const nonce = Date.now();
    const action = {
      type: 'usdClassTransfer',
      hyperliquidChain: this._chain,
      signatureChainId: this._chainId,
      amount: String(amount),
      toPerp: false,
      nonce,
    };
    return this._buildSignSend(action);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // VAULT OPERATIONS
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Deposit USDC into a vault.
   */
  async vaultDeposit(
    vaultAddress: string,
    amount: number
  ): Promise<Record<string, unknown>> {
    const action = {
      type: 'vaultTransfer',
      vaultAddress,
      isDeposit: true,
      usd: amount,
    };
    return this._buildSignSend(action);
  }

  /**
   * Withdraw USDC from a vault.
   */
  async vaultWithdraw(
    vaultAddress: string,
    amount: number
  ): Promise<Record<string, unknown>> {
    const action = {
      type: 'vaultTransfer',
      vaultAddress,
      isDeposit: false,
      usd: amount,
    };
    return this._buildSignSend(action);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // AGENT/API KEY MANAGEMENT
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Approve an agent (API wallet) to trade on your behalf.
   */
  async approveAgent(
    agentAddress: string,
    name?: string
  ): Promise<Record<string, unknown>> {
    const nonce = Date.now();
    const action = {
      type: 'approveAgent',
      hyperliquidChain: this._chain,
      signatureChainId: this._chainId,
      agentAddress,
      agentName: name ?? null,
      nonce,
    };
    return this._buildSignSend(action);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // STAKING OPERATIONS
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Move spot HYPE into undelegated staking HYPE for order priority fees.
   */
  async fundPriorityFees(amountHype: number | string): Promise<Record<string, unknown>> {
    return this.stake(amountHype);
  }

  /**
   * Move spot HYPE into undelegated staking HYPE.
   */
  async stake(amount: number | string): Promise<Record<string, unknown>> {
    const wei = this._hypeToWei(amount);
    const nonce = Date.now();

    const action = {
      type: 'cDeposit',
      wei,
      nonce,
    };
    return this._buildSignSend(action);
  }

  /**
   * Withdraw undelegated staking HYPE back to spot HYPE.
   */
  async unstake(amount: number | string): Promise<Record<string, unknown>> {
    const wei = this._hypeToWei(amount);
    const nonce = Date.now();

    const action = {
      type: 'cWithdraw',
      wei,
      nonce,
    };
    return this._buildSignSend(action);
  }

  /**
   * Delegate staked tokens to a validator.
   */
  async delegate(
    validator: string,
    amount: number | string
  ): Promise<Record<string, unknown>> {
    const wei = this._hypeToWei(amount);
    const nonce = Date.now();

    const action = {
      type: 'tokenDelegate',
      validator,
      isUndelegate: false,
      wei,
      nonce,
    };
    return this._buildSignSend(action);
  }

  /**
   * Undelegate staked tokens from a validator.
   */
  async undelegate(
    validator: string,
    amount: number | string
  ): Promise<Record<string, unknown>> {
    const wei = this._hypeToWei(amount);
    const nonce = Date.now();

    const action = {
      type: 'tokenDelegate',
      validator,
      isUndelegate: true,
      wei,
      nonce,
    };
    return this._buildSignSend(action);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // ACCOUNT ABSTRACTION
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Set account abstraction mode.
   */
  async setAbstraction(
    mode: string,
    options: { user?: string } = {}
  ): Promise<Record<string, unknown>> {
    let user = options.user;
    if (user === undefined) {
      this._requireWallet();
      user = this.address!;
    }

    const action = {
      type: 'userSetAbstraction',
      hyperliquidChain: this._chain,
      signatureChainId: this._chainId,
      user,
      abstraction: mode,
      nonce: Date.now(),
    };
    return this._buildSignSend(action);
  }

  /**
   * Set account abstraction mode as an agent.
   *
   * This is the agent-level version of setAbstraction, used when
   * operating as an approved agent on behalf of a user.
   *
   * @param mode - Abstraction mode: "disabled" (or "i"), "unifiedAccount" (or "u"),
   *               or "portfolioMargin" (or "p")
   *
   * @example
   * // As an agent, enable unified account
   * await sdk.agentSetAbstraction("unifiedAccount");
   *
   * // Disable abstraction
   * await sdk.agentSetAbstraction("disabled");
   */
  async agentSetAbstraction(mode: string): Promise<Record<string, unknown>> {
    // Map full mode names to short codes per API spec
    const modeMap: Record<string, string> = {
      disabled: 'i',
      unifiedAccount: 'u',
      portfolioMargin: 'p',
      i: 'i',
      u: 'u',
      p: 'p',
    };

    const shortMode = modeMap[mode];
    if (!shortMode) {
      throw new ValidationError(
        `Invalid mode: ${mode}`,
        { guidance: 'Use "disabled", "unifiedAccount", or "portfolioMargin"' }
      );
    }

    const action = {
      type: 'agentSetAbstraction',
      abstraction: shortMode,
    };
    return this._buildSignSend(action);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // ADVANCED TRANSFERS
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Generalized asset transfer between DEXs and accounts.
   *
   * @param token - Token in "TokenName:tokenId" format (e.g., "USDC:0x...")
   * @param amount - Amount to transfer
   * @param destination - Destination address
   * @param options - Transfer options
   * @param options.sourceDex - Source DEX ("" for USDC, "spot" for spot)
   * @param options.destinationDex - Destination DEX
   * @param options.fromSubAccount - Optional sub-account address to send from
   *
   * @example
   * await sdk.sendAsset("USDC:0x...", 100, "0xDest...", {
   *   sourceDex: "spot",
   *   destinationDex: ""
   * });
   */
  async sendAsset(
    token: string,
    amount: number | string,
    destination: string,
    options: {
      sourceDex?: string;
      destinationDex?: string;
      fromSubAccount?: string;
    } = {}
  ): Promise<Record<string, unknown>> {
    const nonce = Date.now();
    const action = {
      type: 'sendAsset',
      hyperliquidChain: this._chain,
      signatureChainId: this._chainId,
      destination,
      sourceDex: options.sourceDex ?? '',
      destinationDex: options.destinationDex ?? '',
      token,
      amount: String(amount),
      fromSubAccount: options.fromSubAccount ?? '',
      nonce,
    };
    return this._buildSignSend(action);
  }

  /**
   * Transfer tokens to HyperEVM with custom data payload.
   *
   * Send tokens from HyperCore to a HyperEVM address with arbitrary data attached,
   * enabling interactions with smart contracts.
   *
   * @param token - Token in "tokenName:tokenId" format (e.g., "PURR:0xc4bf...")
   * @param amount - Amount to transfer
   * @param destination - Destination address on HyperEVM (42-char hex)
   * @param data - Hex-encoded data payload (e.g., "0x...")
   * @param options - Required transfer options
   * @param options.sourceDex - Source DEX name (perp DEX name to transfer from)
   * @param options.destinationChainId - Target chain ID (e.g., 999 for HyperEVM mainnet)
   * @param options.gasLimit - Gas limit for the EVM transaction
   * @param options.addressEncoding - Address encoding format ("hex" or "base58", default: "hex")
   *
   * @example
   * // Send 100 PURR to a contract with custom calldata
   * await sdk.sendToEvmWithData(
   *   "PURR:0xc4bf...",
   *   100,
   *   "0xContract...",
   *   "0x1234abcd...",
   *   {
   *     sourceDex: "",
   *     destinationChainId: 999,
   *     gasLimit: 100000,
   *   }
   * );
   */
  async sendToEvmWithData(
    token: string,
    amount: number | string,
    destination: string,
    data: string,
    options: {
      sourceDex: string;
      destinationChainId: number;
      gasLimit: number;
      addressEncoding?: string;
    }
  ): Promise<Record<string, unknown>> {
    const nonce = Date.now();
    const action = {
      type: 'sendToEvmWithData',
      hyperliquidChain: this._chain,
      signatureChainId: this._chainId,
      token,
      amount: String(amount),
      sourceDex: options.sourceDex,
      destinationRecipient: destination,
      addressEncoding: options.addressEncoding ?? 'hex',
      destinationChainId: options.destinationChainId,
      gasLimit: options.gasLimit,
      data,
      nonce,
    };
    return this._buildSignSend(action);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // RATE LIMITING
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Purchase additional rate limit capacity.
   *
   * Cost: 0.0005 USDC per request from Perps balance.
   *
   * @param weight - Number of requests to reserve
   *
   * @example
   * await sdk.reserveRequestWeight(1000);  // Reserve 1000 additional requests
   */
  async reserveRequestWeight(weight: number): Promise<Record<string, unknown>> {
    const action = {
      type: 'reserveRequestWeight',
      weight,
    };
    return this._buildSignSend(action);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // ADDITIONAL MARGIN OPERATIONS
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Top up isolated-only margin to target a specific leverage.
   *
   * This is an alternative to updateIsolatedMargin that targets a specific
   * leverage level instead of specifying a USDC margin amount. The system
   * will add the required margin to achieve the target leverage.
   *
   * @param asset - Asset name (str) or index (int)
   * @param leverage - Target leverage as a float (e.g., 5.0 for 5x leverage)
   *
   * @example
   * // Adjust margin to achieve 5x leverage on BTC position
   * await sdk.topUpIsolatedOnlyMargin("BTC", 5.0);
   */
  async topUpIsolatedOnlyMargin(
    asset: string | number,
    leverage: number | string
  ): Promise<Record<string, unknown>> {
    const assetIdx = typeof asset === 'string' ? await this._resolveAssetIndex(asset) : asset;

    const action = {
      type: 'topUpIsolatedOnlyMargin',
      asset: assetIdx,
      leverage: String(leverage),  // API expects leverage as string
    };
    return this._buildSignSend(action);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // NOOP (NONCE CONSUMPTION)
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * No-operation action to consume a nonce.
   *
   * This action does nothing except increment the nonce. Useful for:
   * - Invalidating previously signed but unsent transactions
   * - Syncing nonce state
   * - Testing signing infrastructure
   *
   * @example
   * await sdk.noop();  // Consume nonce without any action
   */
  async noop(): Promise<Record<string, unknown>> {
    const action = {
      type: 'noop',
    };
    return this._buildSignSend(action);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // VALIDATOR OPERATIONS
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Submit a validator vote for the risk-free rate.
   *
   * This is a validator-only action for participating in the L1 consensus
   * on the risk-free rate used for funding calculations.
   *
   * @param riskFreeRate - Proposed rate as a decimal string (e.g., "0.04" for 4%)
   *
   * @example
   * // Validator submits their rate vote (4%)
   * await sdk.validatorL1Stream("0.04");
   */
  async validatorL1Stream(riskFreeRate: string): Promise<Record<string, unknown>> {
    const action = {
      type: 'validatorL1Stream',
      riskFreeRate,
    };
    return this._buildSignSend(action);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // POSITION MANAGEMENT
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Close an open position completely.
   */
  async closePosition(asset: string, options: { slippage?: number } = {}): Promise<PlacedOrder> {
    const action = {
      type: 'closePosition',
      asset,
      user: this.address,
    };

    const result = await this._buildSignSend(action, options.slippage);

    const order = Order.sell(asset);
    order['_size'] = '0';
    return PlacedOrder.fromResponse(
      (result as Record<string, unknown>).exchangeResponse as Record<string, unknown> ?? {},
      order,
      this
    );
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // ORDER MANAGEMENT
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Cancel an order by OID.
   */
  async cancel(
    oid: number,
    asset?: string | number
  ): Promise<Record<string, unknown>> {
    if (!Number.isInteger(oid) || oid <= 0) {
      throw new ValidationError(`oid must be a positive integer, got: ${oid}`);
    }

    let assetIdx = 0;
    if (asset !== undefined) {
      assetIdx = typeof asset === 'string' ? await this._resolveAssetIndex(asset) : asset;
    }

    const cancelAction = {
      type: 'cancel',
      cancels: [{ a: assetIdx, o: oid }],
    };
    return this._buildSignSend(cancelAction);
  }

  /**
   * Cancel all open orders.
   */
  async cancelAll(asset?: string): Promise<Record<string, unknown>> {
    const orders = await this.openOrders();

    if (!orders.orders || (orders.orders as unknown[]).length === 0) {
      return { message: 'No orders to cancel' };
    }

    if (asset) {
      const cancelActions = orders.cancelActions as Record<string, unknown>;
      const byAsset = cancelActions?.byAsset as Record<string, unknown>;
      if (!byAsset?.[asset]) {
        return { message: `No ${asset} orders to cancel` };
      }
      return this._buildSignSend(byAsset[asset] as Record<string, unknown>);
    } else {
      const cancelActions = orders.cancelActions as Record<string, unknown>;
      if (!cancelActions?.all) {
        return { message: 'No orders to cancel' };
      }
      return this._buildSignSend(cancelActions.all as Record<string, unknown>);
    }
  }

  /**
   * Cancel an order by client order ID (cloid).
   */
  async cancelByCloid(
    cloid: string,
    asset: string | number
  ): Promise<Record<string, unknown>> {
    const assetIdx = typeof asset === 'string' ? await this._resolveAssetIndex(asset) : asset;

    const cancelAction = {
      type: 'cancelByCloid',
      cancels: [{ asset: assetIdx, cloid }],
    };
    return this._buildSignSend(cancelAction);
  }

  /**
   * Schedule cancellation of all orders after a delay.
   */
  async scheduleCancel(time?: number): Promise<Record<string, unknown>> {
    const cancelAction: Record<string, unknown> = { type: 'scheduleCancel' };
    if (time !== undefined) {
      cancelAction.time = time;
    }
    return this._buildSignSend(cancelAction);
  }

  /**
   * Modify an existing order.
   */
  async modify(
    oid: number,
    asset: string,
    side: Side | string,
    price: string,
    size: string,
    options: { tif?: TIF | string; reduceOnly?: boolean } = {}
  ): Promise<PlacedOrder> {
    if (!Number.isInteger(oid) || oid <= 0) {
      throw new ValidationError(`oid must be a positive integer, got: ${oid}`);
    }
    if (!asset || typeof asset !== 'string') {
      throw new ValidationError(`asset must be a non-empty string, got: ${asset}`);
    }

    // In TypeScript string enums, the enum value IS the string
    const sideStr = side as string;
    if (sideStr !== 'buy' && sideStr !== 'sell') {
      throw new ValidationError(`side must be 'buy' or 'sell', got: ${sideStr}`);
    }

    const tif = options.tif ?? TIF.GTC;
    // In TypeScript string enums, the enum value IS the string
    const tifStr = tif as string;
    if (!['gtc', 'ioc', 'alo'].includes(tifStr)) {
      throw new ValidationError(`tif must be 'gtc', 'ioc', or 'alo', got: ${tifStr}`);
    }

    const modifyAction = {
      type: 'batchModify',
      modifies: [{
        oid,
        order: {
          a: asset,
          b: sideStr === 'buy',
          p: String(price),
          s: String(size),
          r: options.reduceOnly ?? false,
          t: { limit: { tif: tifStr.charAt(0).toUpperCase() + tifStr.slice(1) } },
        },
      }],
    };

    const result = await this._buildSignSend(modifyAction);

    const order = new Order(asset, sideStr === 'buy' ? Side.BUY : Side.SELL);
    order['_price'] = price;
    order['_size'] = size;

    return PlacedOrder.fromResponse(
      (result as Record<string, unknown>).exchangeResponse as Record<string, unknown> ?? {},
      order,
      this
    );
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // QUERIES
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Get open orders with enriched info and pre-built cancel actions.
   */
  async openOrders(
    user?: string,
    options: { dex?: string } = {}
  ): Promise<Record<string, unknown>> {
    if (user === undefined) {
      this._requireWallet();
      user = this.address!;
    }
    const body: Record<string, unknown> = { user };
    if (options.dex !== undefined) {
      body.dex = options.dex;
    }
    return this._post('/openOrders', body);
  }

  /**
   * Get detailed status for an order.
   */
  async orderStatus(
    oid: number,
    user?: string,
    options: { dex?: string } = {}
  ): Promise<Record<string, unknown>> {
    if (user === undefined) {
      this._requireWallet();
      user = this.address!;
    }
    const body: Record<string, unknown> = { user, oid };
    if (options.dex !== undefined) {
      body.dex = options.dex;
    }
    return this._post('/orderStatus', body);
  }

  /**
   * Get all available markets.
   */
  async markets(): Promise<Record<string, unknown>> {
    return this._get('/markets');
  }

  /**
   * List active HIP-4 prediction markets with tradeable yes/no sides.
   */
  async predictionMarkets(): Promise<PredictionMarkets> {
    const outcomeMeta = await this._postInfo({ type: 'outcomeMeta' }) as Record<string, unknown>;
    const mids = await this._postInfo({ type: 'allMids' }) as Record<string, string>;
    const outcomes = (outcomeMeta.outcomes as Array<Record<string, unknown>> | undefined) ?? [];
    return buildPredictionMarkets(outcomes, mids);
  }

  /**
   * Alias for predictionMarkets().
   */
  async predictions(): Promise<PredictionMarkets> {
    return this.predictionMarkets();
  }

  /**
   * Find one HIP-4 prediction market by text or structured fields.
   */
  async predictionMarket(queryOrFilter?: string | PredictionMarketFilter): Promise<PredictionMarket> {
    const market = (await this.predictionMarkets()).findMarket(queryOrFilter);
    if (market === undefined) {
      throw new ValidationError('No matching prediction market found', {
        guidance: 'Call sdk.predictionMarkets() to list active HIP-4 markets.',
      });
    }
    return market;
  }

  /**
   * Return a flat list of tradeable HIP-4 sides.
   */
  async predictionSides(): Promise<PredictionSide[]> {
    return (await this.predictionMarkets()).flatMap((market) => market.sides);
  }

  /**
   * Buy USDH with USDC for HIP-4 prediction markets.
   */
  async buyUsdh(amountUsdc: number | string, options: { slippage?: number } = {}): Promise<PlacedOrder> {
    return this.marketBuy('@230', { notional: Number(amountUsdc), slippage: options.slippage });
  }

  /**
   * Sell USDH back to USDC.
   */
  async sellUsdh(amountUsdh: number | string, options: { slippage?: number } = {}): Promise<PlacedOrder> {
    return this.marketSell('@230', { size: String(amountUsdh), slippage: options.slippage });
  }

  /**
   * List enriched HIP-4 outcome metadata and helper action shapes.
   */
  async outcomes(): Promise<Record<string, unknown>> {
    return this._get('/outcomes');
  }

  /**
   * Get USDH and Yes/No balances for one HIP-4 outcome.
   */
  async outcomeBalances(outcome: number | string, user?: string): Promise<Record<string, unknown>> {
    if (user === undefined) {
      this._requireWallet();
      user = this.address!;
    }
    return this._get('/outcomes/balances', { user, outcome: String(outcome) });
  }

  /**
   * Spend USDH and mint equal Yes/No shares for an outcome.
   */
  async outcomeSplit(outcome: number | string, amount: OutcomeAmount): Promise<Record<string, unknown>> {
    return this._buildSignSend({
      type: 'outcomeSplit',
      outcome: Number(outcome),
      amount: String(amount),
    });
  }

  /**
   * Burn matching Yes/No shares and return USDH. amount null/undefined means max.
   */
  async outcomeMerge(
    outcome: number | string,
    amount?: OutcomeAmount | null
  ): Promise<Record<string, unknown>> {
    return this._buildSignSend({
      type: 'outcomeMerge',
      outcome: Number(outcome),
      amount: amount == null ? null : String(amount),
    });
  }

  /**
   * Merge every outcome in a question. amount null/undefined means max.
   */
  async outcomeMergeQuestion(
    question: number | string,
    amount?: OutcomeAmount | null
  ): Promise<Record<string, unknown>> {
    return this._buildSignSend({
      type: 'outcomeMergeQuestion',
      question: Number(question),
      amount: amount == null ? null : String(amount),
    });
  }

  /**
   * Negate one outcome share into the complementary outcome set.
   */
  async outcomeNegate(
    question: number | string,
    outcome: number | string,
    amount: OutcomeAmount
  ): Promise<Record<string, unknown>> {
    return this._buildSignSend({
      type: 'outcomeNegate',
      question: Number(question),
      outcome: Number(outcome),
      amount: String(amount),
    });
  }

  /**
   * Get all HIP-3 DEXes.
   */
  async dexes(): Promise<Record<string, unknown>> {
    return this._get('/dexes');
  }

  /**
   * Validate an order before signing (preflight check).
   */
  async preflight(
    asset: string,
    side: Side | string,
    price: number | string,
    size: number | string,
    options: { tif?: TIF | string; reduceOnly?: boolean } = {}
  ): Promise<Record<string, unknown>> {
    // In TypeScript string enums, the enum value IS the string
    const sideStr = side as string;
    const tif = options.tif ?? TIF.GTC;
    const tifStr = tif as string;

    const order = {
      a: asset,
      b: sideStr === 'buy',
      p: String(price),
      s: String(size),
      r: options.reduceOnly ?? false,
      t: { limit: { tif: tifStr.charAt(0).toUpperCase() + tifStr.slice(1) } },
    };

    return this._post('/preflight', { action: { type: 'order', orders: [order] } });
  }

  /**
   * Check builder fee approval status.
   */
  async approvalStatus(user?: string): Promise<Record<string, unknown>> {
    if (user === undefined) {
      this._requireWallet();
      user = this.address!;
    }
    return this._get('/approval', { user });
  }

  /**
   * Get current mid price for an asset.
   */
  async getMid(asset: AssetInput): Promise<number> {
    let assetName = assetToString(asset);
    if (this._isPredictionAsset(assetName)) {
      assetName = this._predictionSymbol(assetName);
    }

    let data: Record<string, string>;
    if (assetName.includes(':')) {
      const dex = assetName.split(':')[0];
      data = await this._postInfo({ type: 'allMids', dex });
    } else {
      data = await this._postInfo({ type: 'allMids' });
    }

    return parseFloat(data[assetName] ?? '0');
  }

  /**
   * Force refresh of market metadata cache.
   */
  async refreshMarkets(): Promise<Record<string, unknown>> {
    this._marketsCache = await this.markets();
    this._marketsCacheTime = Date.now();
    return this._marketsCache;
  }

  private async _getSizeDecimals(asset: AssetInput): Promise<number> {
    const assetName = assetToString(asset);
    if (this._isPredictionAsset(assetName)) {
      this._szDecimalsCache.set(assetName, 0);
      return 0;
    }

    if (this._szDecimalsCache.has(assetName)) {
      return this._szDecimalsCache.get(assetName)!;
    }

    try {
      const now = Date.now();
      if (this._marketsCache === null || (now - this._marketsCacheTime) > HyperliquidSDK.CACHE_TTL) {
        this._marketsCache = await this.markets();
        this._marketsCacheTime = now;
      }

      const markets = this._marketsCache;

      // Check perps
      for (const m of (markets.perps as Array<Record<string, unknown>>) ?? []) {
        if (m.name === assetName) {
          const decimals = (m.szDecimals as number) ?? 5;
          this._szDecimalsCache.set(assetName, decimals);
          return decimals;
        }
      }
      // Check spot
      for (const m of (markets.spot as Array<Record<string, unknown>>) ?? []) {
        if (m.name === assetName) {
          const decimals = (m.szDecimals as number) ?? 5;
          this._szDecimalsCache.set(assetName, decimals);
          return decimals;
        }
      }
      // Check HIP-3 markets
      for (const dexMarkets of Object.values((markets.hip3 as Record<string, Array<Record<string, unknown>>>) ?? {})) {
        for (const m of dexMarkets) {
          if (m.name === assetName) {
            const decimals = (m.szDecimals as number) ?? 5;
            this._szDecimalsCache.set(assetName, decimals);
            return decimals;
          }
        }
      }
      for (const m of (markets.hip4 as Array<Record<string, unknown>>) ?? []) {
        if (m.name === assetName) {
          const decimals = (m.szDecimals as number) ?? 0;
          this._szDecimalsCache.set(assetName, decimals);
          return decimals;
        }
      }
    } catch {
      // Fall through to default
    }

    return 5;
  }

  private async _resolveAssetIndex(asset: AssetInput): Promise<number> {
    const assetName = assetToString(asset);
    if (/^#\d+$/.test(assetName) || /^\+\d+$/.test(assetName)) {
      return 100000000 + Number(assetName.slice(1));
    }
    if (/^\d+$/.test(assetName)) {
      return Number(assetName);
    }

    try {
      const now = Date.now();
      if (this._marketsCache === null || (now - this._marketsCacheTime) > HyperliquidSDK.CACHE_TTL) {
        this._marketsCache = await this.markets();
        this._marketsCacheTime = now;
      }

      const markets = this._marketsCache;

      // Check perps
      const perps = (markets.perps as Array<Record<string, unknown>>) ?? [];
      for (let i = 0; i < perps.length; i++) {
        if (perps[i].name === assetName) {
          return i;
        }
      }

      // Check spot
      const spot = (markets.spot as Array<Record<string, unknown>>) ?? [];
      for (let i = 0; i < spot.length; i++) {
        if (spot[i].name === assetName) {
          return 10000 + i;
        }
      }

      // Check HIP-3 markets
      for (const dexMarkets of Object.values((markets.hip3 as Record<string, Array<Record<string, unknown>>>) ?? {})) {
        for (const m of dexMarkets) {
          if (m.name === assetName) {
            return (m.assetId as number) ?? 0;
          }
        }
      }
      for (const m of (markets.hip4 as Array<Record<string, unknown>>) ?? []) {
        if (m.name === assetName) {
          return Number(m.index ?? 0);
        }
      }
    } catch {
      // Fall through to error
    }

    throw new ValidationError(`Could not resolve asset '${assetName}' to index`, {
      guidance: 'Check the asset name or use numeric index directly.',
    });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // APPROVAL MANAGEMENT
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Approve builder fee for trading.
   */
  async approveBuilderFee(
    maxFee: string = '1%',
    builder?: string
  ): Promise<Record<string, unknown>> {
    if (builder === undefined) {
      builder = '0x8D62d3000eF0639d1fc9667D06BE7BB98d9993F5';
    }

    const action = {
      type: 'approveBuilderFee',
      hyperliquidChain: this._chain,
      signatureChainId: this._chainId,
      maxFeeRate: maxFee,
      builder,
      nonce: Date.now(),
    };
    return this._buildSignSend(action);
  }

  /**
   * Revoke builder fee approval.
   */
  async revokeBuilderFee(builder?: string): Promise<Record<string, unknown>> {
    return this.approveBuilderFee('0%', builder);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // INTERNAL METHODS
  // ═══════════════════════════════════════════════════════════════════════════

  private async _placeOrder(params: {
    asset: AssetInput;
    side: Side;
    size?: number | string;
    notional?: number;
    price?: number | string;
    tif: TIF | string;
    reduceOnly: boolean;
    grouping: OrderGrouping;
    slippage?: number;
    priorityFee?: number | string;
  }): Promise<PlacedOrder> {
    const asset = assetToString(params.asset);
    const order = new Order(asset, params.side);

    // Handle size
    let size = params.size;
    if (params.notional) {
      const mid = await this.getMid(asset);
      if (mid === 0) {
        throw new ValidationError(`Could not fetch price for ${asset}`, {
          guidance: 'Check the asset name or try again.',
        });
      }
      const szDecimals = await this._getSizeDecimals(asset);
      const factor = Math.pow(10, szDecimals);
      const rawSize = params.notional / mid;
      size = this._isPredictionAsset(asset)
        ? Math.ceil(rawSize)
        : Math.round(rawSize * factor) / factor;
    }

    if (size === undefined) {
      throw new ValidationError('Either size or notional is required', {
        guidance: 'Use size: 0.001 or notional: 100',
      });
    }

    order['_size'] = String(size);

    // Handle TIF
    const tif = typeof params.tif === 'string' ? TIF[params.tif.toUpperCase() as keyof typeof TIF] ?? TIF.IOC : params.tif;
    order['_tif'] = tif;

    // Handle price
    if (tif !== TIF.MARKET && params.price !== undefined) {
      order['_price'] = String(params.price);
    }

    if (params.reduceOnly) {
      order['_reduceOnly'] = true;
    }

    return this._executeOrder(order, params.grouping, params.slippage, params.priorityFee);
  }

  private async _executeOrder(
    order: Order,
    grouping: OrderGrouping = OrderGrouping.NA,
    slippage?: number,
    priorityFee?: number | string
  ): Promise<PlacedOrder> {
    const action = order.toAction();
    const effectivePriorityFee = priorityFee ?? order.getPriorityFee();
    if (grouping !== OrderGrouping.NA && effectivePriorityFee !== null) {
      throw new ValidationError('priorityFee cannot be combined with TP/SL grouping', {
        guidance: 'Use priorityFee on standalone IOC/market orders, or omit grouping.',
      });
    }
    await this._validatePredictionOrder(
      order.asset,
      order.getSize(),
      order.getPrice(),
      order.isMarket(),
      order.side === Side.BUY,
      effectivePriorityFee ?? undefined
    );
    if (grouping !== OrderGrouping.NA) {
      action.grouping = grouping; // enum value is already the string (e.g., 'na', 'normalTpsl')
    }
    const result = await this._buildSignSend(action, slippage, effectivePriorityFee ?? undefined);
    return PlacedOrder.fromResponse(
      (result as Record<string, unknown>).exchangeResponse as Record<string, unknown> ?? {},
      order,
      this
    );
  }

  private _requireWallet(): void {
    if (this._wallet === null && this._signer === null) {
      throw new Error(
        'Private key or signer required for this operation. ' +
        'Pass privateKey/signer to HyperliquidSDK() or set PRIVATE_KEY env var.'
      );
    }
  }

  private async _buildSignSend(
    action: Record<string, unknown>,
    slippage?: number,
    priorityFee?: number | string
  ): Promise<Record<string, unknown>> {
    this._requireWallet();

    // Ensure approval on first trading action. Skipped under an external signer
    // (no in-process wallet to sign the approval) — call approveBuilderFee()
    // yourself if you need builder fees.
    if (this._autoApprove && this._wallet !== null && this._signer === null) {
      try {
        const status = await this.approvalStatus();
        if (!status.approved) {
          await this.approveBuilderFee(this._maxFee);
        }
      } catch {
        // Ignore approval check errors
      }
    }

    // Step 1: Build
    const effectiveSlippage = slippage ?? this._slippage;
    const buildPayload: Record<string, unknown> = { action, slippage: effectiveSlippage };
    const normalizedPriorityFee = this._normalizePriorityFee(priorityFee);
    if (normalizedPriorityFee !== undefined) {
      buildPayload.priorityFee = normalizedPriorityFee;
    }
    const buildResult = await this._exchange(buildPayload);

    if (!buildResult.hash) {
      throw new BuildError('Build response missing hash', { raw: buildResult });
    }

    // Step 2: Sign. Prefer the external signer, bounding it with an AbortSignal
    // derived from the SDK timeout, and surface any failure as SignerError (the
    // caller's KMS/HSM/service erred) — distinct from a venue rejection.
    let sig: Signature;
    if (this._signer) {
      const signController = new AbortController();
      const signTimeout = setTimeout(() => signController.abort(), this._timeout);
      try {
        sig = await this._signer(buildResult.hash as string, { signal: signController.signal });
      } catch (err) {
        throw new SignerError(
          `failed to sign: ${err instanceof Error ? err.message : String(err)}`,
          {
            guidance: 'The signer callback threw or rejected (KMS/HSM/remote signing service). This is not a venue rejection; check your signer.',
            cause: err,
          }
        );
      } finally {
        clearTimeout(signTimeout);
      }
      // The signer is user-supplied and may resolve to a malformed value
      // without throwing. Validate it so a null/incomplete result surfaces as
      // a clear SignerError instead of a confusing venue rejection.
      if (!isValidSignature(sig)) {
        throw new SignerError(
          `signer returned an invalid signature (expected { r, s, v in {27,28} }): ${JSON.stringify(sig)}`,
          { guidance: 'The signer callback returned a malformed signature (KMS/HSM/remote signing service). This is not a venue rejection; check your signer.' }
        );
      }
    } else {
      sig = this._signHash(buildResult.hash as string);
    }

    // Step 3: Send
    const sendPayload = {
      action: buildResult.action ?? action,
      nonce: buildResult.nonce,
      signature: sig,
    };

    return this._exchange(sendPayload);
  }

  private _isPredictionAsset(asset: AssetInput): boolean {
    const name = assetToString(asset);
    if (/^[#+]\d+$/.test(name)) return true;
    return /^\d+$/.test(name) && Number(name) >= 100000000;
  }

  private _predictionSymbol(asset: AssetInput): string {
    const name = assetToString(asset);
    if (/^\+\d+$/.test(name)) return `#${name.slice(1)}`;
    if (/^\d+$/.test(name) && Number(name) >= 100000000) return `#${Number(name) - 100000000}`;
    return name;
  }

  private async _validatePredictionOrder(
    asset: AssetInput,
    size: string | null,
    price: string | null,
    isMarket: boolean,
    isBuy: boolean,
    priorityFee?: number | string | null
  ): Promise<void> {
    if (!this._isPredictionAsset(asset)) return;

    if (priorityFee !== undefined && priorityFee !== null) {
      throw new ValidationError('priorityFee is not supported for HIP-4 prediction markets', {
        guidance: 'Omit priorityFee when trading market.yes, market.no, or # markets.',
      });
    }

    if (size === null) return;

    const sizeValue = Number(size);
    if (!Number.isFinite(sizeValue) || !Number.isInteger(sizeValue)) {
      throw new ValidationError('HIP-4 prediction market size must be a whole number of contracts', {
        guidance: 'Use size: 20, not size: 20.5.',
      });
    }

    let px = price !== null ? Number(price) : undefined;
    if (px === undefined && isMarket) {
      const mid = await this.getMid(this._predictionSymbol(asset));
      if (mid > 0) px = mid;
    }

    if (isBuy && px !== undefined && sizeValue * px < 10) {
      throw new ValidationError('HIP-4 prediction market orders must have minimum value of 10 USDH', {
        guidance: 'Increase size or price, or call sdk.buyUsdh(...) before trading.',
      });
    }
  }

  private _hypeToWei(amount: number | string): string {
    const raw = String(amount).trim();
    const normalized = this._normalizeDecimalAmount(raw);
    if (!normalized) {
      throw new ValidationError('HYPE amount must be a positive number', {
        guidance: 'Use a positive amount like 0.001',
      });
    }
    const [whole, frac = ''] = normalized.split('.');
    const fracPadded = frac.slice(0, 8).padEnd(8, '0');
    const wei = BigInt(whole) * 100000000n + BigInt(fracPadded || '0');
    if (wei <= 0n) {
      throw new ValidationError('HYPE amount is too small', {
        guidance: 'Minimum unit is 0.00000001 HYPE',
      });
    }
    return wei.toString();
  }

  private _normalizeDecimalAmount(raw: string): string | null {
    if (!/^\d+(\.\d+)?([eE][+-]?\d+)?$/.test(raw)) {
      return null;
    }

    const [mantissa, exponentPart] = raw.toLowerCase().split('e');
    if (exponentPart === undefined) {
      return mantissa;
    }

    const exponent = Number(exponentPart);
    if (!Number.isInteger(exponent)) {
      return null;
    }

    const [whole, frac = ''] = mantissa.split('.');
    const digits = `${whole}${frac}`;
    const decimalPlaces = frac.length - exponent;

    if (decimalPlaces <= 0) {
      return `${digits}${'0'.repeat(-decimalPlaces)}`;
    }

    if (decimalPlaces >= digits.length) {
      return `0.${'0'.repeat(decimalPlaces - digits.length)}${digits}`;
    }

    const pointIndex = digits.length - decimalPlaces;
    return `${digits.slice(0, pointIndex)}.${digits.slice(pointIndex)}`;
  }

  private _normalizePriorityFee(priorityFee?: number | string): number | undefined {
    if (priorityFee === undefined || priorityFee === null) {
      return undefined;
    }
    const value = Number(priorityFee);
    if (!Number.isInteger(value) || value < 0) {
      throw new ValidationError('priorityFee must be an unsigned integer', {
        guidance: 'Use priorityFee: 10000 for 1 bp.',
      });
    }
    return value;
  }

  private _signHash(hashHex: string): Signature {
    const hashBytes = Buffer.from(hashHex.replace(/^0x/, ''), 'hex');
    const sig = this._wallet!.signingKey.sign(hashBytes);
    return {
      r: sig.r,
      s: sig.s,
      v: sig.v,
    };
  }

  private async _exchange(body: Record<string, unknown>): Promise<Record<string, unknown>> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this._timeout);

    try {
      const response = await fetch(this._exchangeUrl, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      const data = await response.json() as Record<string, unknown>;

      if (data.error) {
        throw parseApiError(data, response.status);
      }

      return data;
    } catch (error) {
      clearTimeout(timeoutId);

      if (error instanceof HyperliquidError) throw error;

      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new HyperliquidError(`Exchange request timed out after ${this._timeout}ms`, {
            code: 'TIMEOUT',
          });
        }
        throw new HyperliquidError(`Connection failed: ${error.message}`, {
          code: 'CONNECTION_ERROR',
        });
      }
      throw error;
    }
  }

  private async _get(
    path: string,
    params?: Record<string, string>
  ): Promise<Record<string, unknown>> {
    const url = new URL(`${this._publicWorkerUrl}${path}`);
    if (params) {
      for (const [key, value] of Object.entries(params)) {
        url.searchParams.set(key, value);
      }
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this._timeout);

    try {
      const response = await fetch(url.toString(), {
        method: 'GET',
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      const data = await response.json() as Record<string, unknown>;

      if (data.error) {
        throw parseApiError(data, response.status);
      }

      return data;
    } catch (error) {
      clearTimeout(timeoutId);

      if (error instanceof HyperliquidError) throw error;

      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new HyperliquidError(`Request timed out after ${this._timeout}ms`, {
            code: 'TIMEOUT',
          });
        }
        throw new HyperliquidError(`Connection failed: ${error.message}`, {
          code: 'CONNECTION_ERROR',
        });
      }
      throw error;
    }
  }

  private async _post(
    path: string,
    body: Record<string, unknown>
  ): Promise<Record<string, unknown>> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this._timeout);

    try {
      const response = await fetch(`${this._publicWorkerUrl}${path}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      const data = await response.json() as Record<string, unknown>;

      if (data.error) {
        throw parseApiError(data, response.status);
      }

      return data;
    } catch (error) {
      clearTimeout(timeoutId);

      if (error instanceof HyperliquidError) throw error;

      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new HyperliquidError(`Request timed out after ${this._timeout}ms`, {
            code: 'TIMEOUT',
          });
        }
        throw new HyperliquidError(`Connection failed: ${error.message}`, {
          code: 'CONNECTION_ERROR',
        });
      }
      throw error;
    }
  }

  private async _postInfo(body: Record<string, unknown>): Promise<Record<string, string>> {
    const reqType = (body.type as string) ?? '';

    // Route based on method support
    const url = HyperliquidSDK.QN_SUPPORTED_INFO_METHODS.has(reqType)
      ? this._infoUrl
      : `${this._publicWorkerUrl}/info`;

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this._timeout);

    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      // Check for geo-blocking
      if (response.status === 403) {
        try {
          const errorData = await response.json() as Record<string, unknown>;
          const errorStr = JSON.stringify(errorData).toLowerCase();
          if (errorStr.includes('restricted') || errorStr.includes('jurisdiction')) {
            throw new GeoBlockedError(errorData);
          }
        } catch (e) {
          if (e instanceof GeoBlockedError) throw e;
        }
      }

      const data = await response.json() as Record<string, unknown>;

      if (data.error) {
        throw parseApiError(data, response.status);
      }

      return data as Record<string, string>;
    } catch (error) {
      clearTimeout(timeoutId);

      if (error instanceof HyperliquidError) throw error;

      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new HyperliquidError(`Info request timed out after ${this._timeout}ms`, {
            code: 'TIMEOUT',
          });
        }
        throw new HyperliquidError(`Connection failed: ${error.message}`, {
          code: 'CONNECTION_ERROR',
        });
      }
      throw error;
    }
  }
}
