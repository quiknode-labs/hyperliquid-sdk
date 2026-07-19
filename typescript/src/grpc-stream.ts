/**
 * gRPC Stream Client — High-performance real-time data streams with automatic reconnection.
 *
 * Stream trades, orders, book updates, blocks, and more via gRPC.
 * Handles connection management, keepalive, and automatic reconnection.
 *
 * The gRPC API uses Protocol Buffers over HTTP/2 on port 10000.
 * Authentication is via x-token header with your QuickNode API token.
 *
 * Example:
 *     import { GRPCStream } from 'hyperliquid-sdk';
 *     const stream = new GRPCStream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN");
 *     stream.trades(["BTC", "ETH"], (t) => console.log(t));
 *     stream.start();
 */

// Note: This module requires @grpc/grpc-js and @grpc/proto-loader as optional dependencies

export enum GRPCStreamType {
  TRADES = 'TRADES',
  ORDERS = 'ORDERS',
  BOOK_UPDATES = 'BOOK_UPDATES',
  TWAP = 'TWAP',
  EVENTS = 'EVENTS',
  BLOCKS = 'BLOCKS',
  WRITER_ACTIONS = 'WRITER_ACTIONS',
  MEMPOOL_TXS = 'MEMPOOL_TXS',
  ORDER_PRIORITY = 'ORDER_PRIORITY',
  GOSSIP_PRIORITY = 'GOSSIP_PRIORITY',
}

export enum ConnectionState {
  DISCONNECTED = 'disconnected',
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
  RECONNECTING = 'reconnecting',
}

export interface GRPCStreamOptions {
  onError?: (error: Error) => void;
  onClose?: () => void;
  onConnect?: () => void;
  onReconnect?: (attempt: number) => void;
  onStateChange?: (state: ConnectionState) => void;
  secure?: boolean;
  reconnect?: boolean;
  maxReconnectAttempts?: number;
}

type Callback = (data: Record<string, unknown>) => void;

/** Callback for the StreamDataBytes fast path: payload bytes are NOT JSON-decoded. */
type BytesCallback = (data: { block_number: number; timestamp: number; data: Uint8Array }) => void;

interface Subscription {
  streamType: string;
  callback: Callback;
  coins?: string[];
  users?: string[];
  coin?: string;
  nSigFigs?: number;
  nLevels?: number;
  mantissa?: number;
  skipInitialSnapshot?: boolean;
  startBlock?: number;
  raw?: boolean;
  bytes?: boolean;
}

// Stream type enum values matching proto
const STREAM_TYPE_MAP: Record<string, number> = {
  TRADES: 1,
  ORDERS: 2,
  BOOK_UPDATES: 3,
  TWAP: 4,
  EVENTS: 5,
  BLOCKS: 6,
  WRITER_ACTIONS: 7,
  MEMPOOL_TXS: 8,
  ORDER_PRIORITY: 9,
  GOSSIP_PRIORITY: 10,
};

/**
 * gRPC Stream Client — High-performance real-time data streams.
 *
 * Features:
 * - Automatic reconnection with exponential backoff
 * - Keepalive pings to maintain connection
 * - Thread-safe subscription management
 * - Graceful shutdown
 * - Native Protocol Buffer support
 *
 * Streams:
 * - trades: Executed trades with price, size, direction
 * - orders: Order lifecycle events (open, filled, cancelled)
 * - book_updates: Order book changes
 * - twap: Time-weighted average price execution
 * - events: System events (funding, liquidations)
 * - blocks: Block data
 * - mempool_txs: Pre-consensus mempool transactions
 * - order_priority: Derived order/write priority actions
 * - gossip_priority: Derived gossip/read priority bid actions
 * - l2_book: Level 2 order book (aggregated price levels)
 * - l4_book: Level 4 order book (individual orders)
 * - bbo_book: Top-of-book (best bid/offer) changes
 * - l2_book_diff: Incremental L2 price-level changes
 * - l4_book_updates: Typed L4 order-level changes
 * - tpsl_updates: Trigger/TP-SL order updates
 * - l2_book_packed / bbo_book_packed: Fixed-point fast paths (px/sz scaled by 1e8)
 * - l4_book_bytes: L4 fast path with JSON-bytes diffs
 */
export class GRPCStream {
  static readonly GRPC_PORT = 10000;
  static readonly INITIAL_RECONNECT_DELAY = 1000;
  static readonly MAX_RECONNECT_DELAY = 60000;
  static readonly RECONNECT_BACKOFF_FACTOR = 2.0;
  static readonly KEEPALIVE_TIME_MS = 30000;
  static readonly KEEPALIVE_TIMEOUT_MS = 10000;
  static readonly MAX_MSG_SIZE = 100 * 1024 * 1024; // 100MB

  private readonly _host: string;
  private readonly _token: string;
  private readonly _onError?: (error: Error) => void;
  private readonly _onClose?: () => void;
  private readonly _onConnect?: () => void;
  private readonly _onReconnect?: (attempt: number) => void;
  private readonly _onStateChange?: (state: ConnectionState) => void;
  private readonly _secure: boolean;
  private readonly _reconnectEnabled: boolean;
  private readonly _maxReconnectAttempts: number | null;

  private _running = false;
  private _state: ConnectionState = ConnectionState.DISCONNECTED;
  private _reconnectAttempt = 0;
  private _reconnectDelay = GRPCStream.INITIAL_RECONNECT_DELAY;
  private _subscriptions: Subscription[] = [];
  private _stopRequested = false;

  // gRPC objects
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private _channel: any = null;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private _streamingClient: any = null;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private _blockClient: any = null;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private _orderbookClient: any = null;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private _activeStreams: any[] = [];
  private _pingIntervals: NodeJS.Timeout[] = [];

  constructor(endpoint: string, options: GRPCStreamOptions = {}) {
    const [host, token] = this._parseEndpoint(endpoint);
    this._host = host;
    this._token = token;
    this._onError = options.onError;
    this._onClose = options.onClose;
    this._onConnect = options.onConnect;
    this._onReconnect = options.onReconnect;
    this._onStateChange = options.onStateChange;
    this._secure = options.secure ?? true;
    this._reconnectEnabled = options.reconnect ?? true;
    this._maxReconnectAttempts = options.maxReconnectAttempts ?? null;
  }

  private _setState(state: ConnectionState): void {
    if (this._state !== state) {
      this._state = state;
      if (this._onStateChange) {
        try {
          this._onStateChange(state);
        } catch {
          // Ignore callback errors
        }
      }
    }
  }

  private _parseEndpoint(url: string): [string, string] {
    const parsed = new URL(url);
    let host = parsed.host;

    // Remove port if present
    if (host.includes(':')) {
      host = host.split(':')[0];
    }

    // Extract token from path
    const pathParts = parsed.pathname.split('/').filter((p) => p.length > 0);
    const knownPaths = new Set(['info', 'hypercore', 'evm', 'nanoreth', 'ws']);
    let token = '';
    for (const part of pathParts) {
      if (!knownPaths.has(part)) {
        token = part;
        break;
      }
    }

    return [host, token];
  }

  private _getTarget(): string {
    return `${this._host}:${GRPCStream.GRPC_PORT}`;
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private _getMetadata(): any {
    // eslint-disable-next-line @typescript-eslint/no-require-imports, @typescript-eslint/no-explicit-any
    const grpc: any = require('@grpc/grpc-js');
    const metadata = new grpc.Metadata();
    metadata.set('x-token', this._token);
    return metadata;
  }

  private _addSubscription(
    streamType: string,
    callback: Callback,
    options: {
      coins?: string[];
      users?: string[];
      coin?: string;
      nSigFigs?: number;
      nLevels?: number;
      mantissa?: number;
      skipInitialSnapshot?: boolean;
      startBlock?: number;
      raw?: boolean;
      bytes?: boolean;
    } = {}
  ): void {
    this._subscriptions.push({
      streamType,
      callback,
      coins: options.coins,
      users: options.users,
      coin: options.coin,
      nSigFigs: options.nSigFigs,
      nLevels: options.nLevels ?? 20,
      mantissa: options.mantissa,
      skipInitialSnapshot: options.skipInitialSnapshot,
      startBlock: options.startBlock,
      raw: options.raw ?? false,
      bytes: options.bytes ?? false,
    });
  }

  /**
   * Subscribe to trade stream.
   */
  trades(coins: string[], callback: Callback, options: { startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.TRADES, callback, { coins, startBlock: options.startBlock });
    return this;
  }

  /**
   * Subscribe to raw trade blocks.
   */
  rawTrades(coins: string[], callback: Callback, options: { startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.TRADES, callback, { coins, startBlock: options.startBlock, raw: true });
    return this;
  }

  /**
   * Subscribe to order stream.
   */
  orders(coins: string[], callback: Callback, options: { users?: string[]; startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.ORDERS, callback, { coins, users: options.users, startBlock: options.startBlock });
    return this;
  }

  /**
   * Subscribe to raw order blocks.
   */
  rawOrders(coins: string[], callback: Callback, options: { users?: string[]; startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.ORDERS, callback, { coins, users: options.users, startBlock: options.startBlock, raw: true });
    return this;
  }

  /**
   * Subscribe to order book updates.
   */
  bookUpdates(coins: string[], callback: Callback, options: { startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.BOOK_UPDATES, callback, { coins, startBlock: options.startBlock });
    return this;
  }

  /**
   * Subscribe to raw order book update blocks.
   */
  rawBookUpdates(coins: string[], callback: Callback, options: { startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.BOOK_UPDATES, callback, { coins, startBlock: options.startBlock, raw: true });
    return this;
  }

  /**
   * Subscribe to TWAP execution stream.
   */
  twap(coins: string[], callback: Callback, options: { startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.TWAP, callback, { coins, startBlock: options.startBlock });
    return this;
  }

  /**
   * Subscribe to raw TWAP execution blocks.
   */
  rawTwap(coins: string[], callback: Callback, options: { startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.TWAP, callback, { coins, startBlock: options.startBlock, raw: true });
    return this;
  }

  /**
   * Subscribe to system events (funding, liquidations, governance).
   */
  events(callback: Callback, options: { startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.EVENTS, callback, { startBlock: options.startBlock });
    return this;
  }

  /**
   * Subscribe to raw system event blocks.
   */
  rawEvents(callback: Callback, options: { startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.EVENTS, callback, { startBlock: options.startBlock, raw: true });
    return this;
  }

  /**
   * Subscribe to block data.
   */
  blocks(callback: Callback): GRPCStream {
    this._addSubscription(GRPCStreamType.BLOCKS, callback);
    return this;
  }

  /**
   * Subscribe to writer actions (HyperCore <-> HyperEVM asset transfers).
   */
  writerActions(callback: Callback, options: { startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.WRITER_ACTIONS, callback, { startBlock: options.startBlock });
    return this;
  }

  /**
   * Subscribe to raw writer action blocks.
   */
  rawWriterActions(callback: Callback, options: { startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.WRITER_ACTIONS, callback, { startBlock: options.startBlock, raw: true });
    return this;
  }

  /**
   * Subscribe to pre-consensus mempool transactions.
   * Optional coin filter (server applies OR across values); omit coins for all.
   */
  mempoolTxs(callback: Callback): GRPCStream;
  mempoolTxs(coins: string[], callback: Callback, options?: { startBlock?: number }): GRPCStream;
  mempoolTxs(
    coinsOrCallback: string[] | Callback,
    callback?: Callback,
    options: { startBlock?: number } = {}
  ): GRPCStream {
    const coins = Array.isArray(coinsOrCallback) ? coinsOrCallback : undefined;
    const cb = Array.isArray(coinsOrCallback) ? (callback as Callback) : coinsOrCallback;
    this._addSubscription(GRPCStreamType.MEMPOOL_TXS, cb, { coins, startBlock: options.startBlock });
    return this;
  }

  /**
   * Subscribe to raw pre-consensus mempool transaction blocks.
   */
  rawMempoolTxs(callback: Callback): GRPCStream;
  rawMempoolTxs(coins: string[], callback: Callback, options?: { startBlock?: number }): GRPCStream;
  rawMempoolTxs(
    coinsOrCallback: string[] | Callback,
    callback?: Callback,
    options: { startBlock?: number } = {}
  ): GRPCStream {
    const coins = Array.isArray(coinsOrCallback) ? coinsOrCallback : undefined;
    const cb = Array.isArray(coinsOrCallback) ? (callback as Callback) : coinsOrCallback;
    this._addSubscription(GRPCStreamType.MEMPOOL_TXS, cb, { coins, startBlock: options.startBlock, raw: true });
    return this;
  }

  /**
   * Subscribe to derived order/write priority actions (from mempool and confirmed replica data).
   * Events carry server-enriched fields: coin, market_type, sz_decimals.
   */
  orderPriority(callback: Callback, options: { startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.ORDER_PRIORITY, callback, { startBlock: options.startBlock });
    return this;
  }

  /**
   * Subscribe to raw order/write priority action blocks.
   */
  rawOrderPriority(callback: Callback, options: { startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.ORDER_PRIORITY, callback, { startBlock: options.startBlock, raw: true });
    return this;
  }

  /**
   * Subscribe to derived gossip/read priority bid actions (does not measure delivery latency).
   * Events carry server-enriched fields: coin, market_type, sz_decimals.
   */
  gossipPriority(callback: Callback, options: { startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.GOSSIP_PRIORITY, callback, { startBlock: options.startBlock });
    return this;
  }

  /**
   * Subscribe to raw gossip/read priority bid action blocks.
   */
  rawGossipPriority(callback: Callback, options: { startBlock?: number } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.GOSSIP_PRIORITY, callback, { startBlock: options.startBlock, raw: true });
    return this;
  }

  /**
   * Subscribe to the raw-bytes fast path of the generic data stream (StreamDataBytes).
   * The callback receives { block_number, timestamp, data } where data is the
   * undecoded payload bytes — no JSON parsing is performed.
   */
  streamDataBytes(
    streamType: GRPCStreamType,
    callback: BytesCallback,
    options: { coins?: string[]; users?: string[]; startBlock?: number } = {}
  ): GRPCStream {
    this._addSubscription(streamType, callback as unknown as Callback, {
      coins: options.coins,
      users: options.users,
      startBlock: options.startBlock,
      bytes: true,
    });
    return this;
  }

  /**
   * Subscribe to Level 2 order book updates (aggregated price levels).
   */
  l2Book(
    coin: string,
    callback: Callback,
    options: { nSigFigs?: number; nLevels?: number } = {}
  ): GRPCStream {
    this._addSubscription('L2_BOOK', callback, {
      coin,
      nSigFigs: options.nSigFigs,
      nLevels: options.nLevels ?? 20,
    });
    return this;
  }

  /**
   * Subscribe to Level 4 order book updates (individual orders).
   *
   * Note: the server may send an unsolicited full snapshot at any time after
   * subscribe; discard local book state and replace it with any snapshot
   * received mid-stream.
   */
  l4Book(coin: string, callback: Callback): GRPCStream {
    this._addSubscription('L4_BOOK', callback, { coin });
    return this;
  }

  /**
   * Subscribe to best bid/offer updates. Omitted/empty coins = all coins.
   * Emits only when the best bid or ask changes for a coin.
   */
  bboBook(callback: Callback): GRPCStream;
  bboBook(coins: string[], callback: Callback): GRPCStream;
  bboBook(coinsOrCallback: string[] | Callback, callback?: Callback): GRPCStream {
    const coins = Array.isArray(coinsOrCallback) ? coinsOrCallback : undefined;
    const cb = Array.isArray(coinsOrCallback) ? (callback as Callback) : coinsOrCallback;
    this._addSubscription('BBO_BOOK', cb, { coins });
    return this;
  }

  /**
   * Subscribe to incremental L2 price-level changes. Omitted/empty coins = all coins.
   * Changed levels with sz=0 mean the level was removed.
   */
  l2BookDiff(
    callback: Callback,
    options: {
      coins?: string[];
      nSigFigs?: number;
      nLevels?: number;
      mantissa?: number;
      skipInitialSnapshot?: boolean;
    } = {}
  ): GRPCStream {
    this._addSubscription('L2_BOOK_DIFF', callback, {
      coins: options.coins,
      nSigFigs: options.nSigFigs,
      nLevels: options.nLevels ?? 20,
      mantissa: options.mantissa,
      skipInitialSnapshot: options.skipInitialSnapshot,
    });
    return this;
  }

  /**
   * Subscribe to typed L4 order book updates. Omitted/empty coins = all coins.
   *
   * Note: updates with snapshot=true carry a full reset snapshot; discard
   * local book state and replace it whenever one arrives mid-stream.
   */
  l4BookUpdates(callback: Callback): GRPCStream;
  l4BookUpdates(coins: string[], callback: Callback): GRPCStream;
  l4BookUpdates(coinsOrCallback: string[] | Callback, callback?: Callback): GRPCStream {
    const coins = Array.isArray(coinsOrCallback) ? coinsOrCallback : undefined;
    const cb = Array.isArray(coinsOrCallback) ? (callback as Callback) : coinsOrCallback;
    this._addSubscription('L4_BOOK_UPDATES', cb, { coins });
    return this;
  }

  /**
   * Subscribe to trigger/TP-SL order updates. Omitted/empty coins = all perp coins.
   */
  tpslUpdates(callback: Callback): GRPCStream;
  tpslUpdates(coins: string[], callback: Callback): GRPCStream;
  tpslUpdates(coinsOrCallback: string[] | Callback, callback?: Callback): GRPCStream {
    const coins = Array.isArray(coinsOrCallback) ? coinsOrCallback : undefined;
    const cb = Array.isArray(coinsOrCallback) ? (callback as Callback) : coinsOrCallback;
    this._addSubscription('TPSL_UPDATES', cb, { coins });
    return this;
  }

  /**
   * Subscribe to the fixed-point L2 book fast path.
   * Prices/sizes are u64 fixed-point integers scaled by 1e8.
   */
  l2BookPacked(
    coin: string,
    callback: Callback,
    options: { nSigFigs?: number; nLevels?: number; mantissa?: number } = {}
  ): GRPCStream {
    this._addSubscription('L2_BOOK_PACKED', callback, {
      coin,
      nSigFigs: options.nSigFigs,
      nLevels: options.nLevels ?? 20,
      mantissa: options.mantissa,
    });
    return this;
  }

  /**
   * Subscribe to the fixed-point BBO fast path. Omitted/empty coins = all coins.
   * Prices/sizes are u64 fixed-point integers scaled by 1e8.
   */
  bboBookPacked(callback: Callback): GRPCStream;
  bboBookPacked(coins: string[], callback: Callback): GRPCStream;
  bboBookPacked(coinsOrCallback: string[] | Callback, callback?: Callback): GRPCStream {
    const coins = Array.isArray(coinsOrCallback) ? coinsOrCallback : undefined;
    const cb = Array.isArray(coinsOrCallback) ? (callback as Callback) : coinsOrCallback;
    this._addSubscription('BBO_BOOK_PACKED', cb, { coins });
    return this;
  }

  /**
   * Subscribe to the L4 book fast path: diffs are delivered as undecoded JSON
   * bytes ({order_statuses, book_diffs}) instead of a protobuf string.
   *
   * Note: the server may send an unsolicited full snapshot at any time after
   * subscribe; discard local book state and replace it with any snapshot
   * received mid-stream.
   */
  l4BookBytes(coin: string, callback: Callback): GRPCStream {
    this._addSubscription('L4_BOOK_BYTES', callback, { coin });
    return this;
  }

  /**
   * Connect and create gRPC clients.
   */
  private async _connect(): Promise<void> {
    // eslint-disable-next-line @typescript-eslint/no-require-imports, @typescript-eslint/no-explicit-any
    const grpc: any = require('@grpc/grpc-js');
    // eslint-disable-next-line @typescript-eslint/no-require-imports, @typescript-eslint/no-explicit-any
    const protoLoader: any = require('@grpc/proto-loader');
    const path = await import('path');
    const fs = await import('fs');

    this._setState(ConnectionState.CONNECTING);

    const target = this._getTarget();

    // Load proto files
    const streamingProtoPath = path.resolve(__dirname, 'proto', 'streaming.proto');
    const orderbookProtoPath = path.resolve(__dirname, 'proto', 'orderbook.proto');

    // Check if proto files exist
    if (!fs.existsSync(streamingProtoPath)) {
      throw new Error(`Proto file not found: ${streamingProtoPath}`);
    }
    if (!fs.existsSync(orderbookProtoPath)) {
      throw new Error(`Proto file not found: ${orderbookProtoPath}`);
    }

    const packageDefinition = protoLoader.loadSync([streamingProtoPath, orderbookProtoPath], {
      keepCase: true,
      longs: String,
      enums: Number,
      defaults: true,
      oneofs: true,
    });

    const protoDescriptor = grpc.loadPackageDefinition(packageDefinition);
    const hyperliquid = protoDescriptor.hyperliquid;

    // Create credentials
    const channelOptions = {
      'grpc.keepalive_time_ms': GRPCStream.KEEPALIVE_TIME_MS,
      'grpc.keepalive_timeout_ms': GRPCStream.KEEPALIVE_TIMEOUT_MS,
      'grpc.keepalive_permit_without_calls': 1,
      'grpc.max_receive_message_length': GRPCStream.MAX_MSG_SIZE,
      'grpc.max_send_message_length': GRPCStream.MAX_MSG_SIZE,
    };

    let credentials;
    if (this._secure) {
      credentials = grpc.credentials.createSsl();
    } else {
      credentials = grpc.credentials.createInsecure();
    }

    // Create clients
    this._streamingClient = new hyperliquid.Streaming(target, credentials, channelOptions);
    this._blockClient = new hyperliquid.BlockStreaming(target, credentials, channelOptions);
    this._orderbookClient = new hyperliquid.OrderBookStreaming(target, credentials, channelOptions);

    this._setState(ConnectionState.CONNECTED);
    this._reconnectAttempt = 0;
    this._reconnectDelay = GRPCStream.INITIAL_RECONNECT_DELAY;

    if (this._onConnect) {
      try {
        this._onConnect();
      } catch {
        // Ignore callback errors
      }
    }
  }

  /**
   * Build the initial StreamSubscribe request for a data subscription.
   */
  private _buildSubscribeRequest(sub: Subscription): {
    subscribe: { stream_type: number; filters: Record<string, { values: string[] }>; start_block?: number };
  } {
    const subscribe: { stream_type: number; filters: Record<string, { values: string[] }>; start_block?: number } = {
      stream_type: STREAM_TYPE_MAP[sub.streamType] || 0,
      filters: {},
    };

    if (sub.startBlock !== undefined) {
      subscribe.start_block = sub.startBlock;
    }
    if (sub.coins && sub.coins.length > 0) {
      subscribe.filters['coin'] = { values: sub.coins };
    }
    if (sub.users && sub.users.length > 0) {
      subscribe.filters['user'] = { values: sub.users };
    }

    return { subscribe };
  }

  /**
   * Start streaming for a data subscription (trades, orders, etc.).
   */
  private _streamData(sub: Subscription): void {
    if (!this._streamingClient || this._stopRequested) return;

    const metadata = this._getMetadata();

    // Create bidirectional stream
    const stream = this._streamingClient.StreamData(metadata);
    this._activeStreams.push(stream);

    // Send initial subscription request
    stream.write(this._buildSubscribeRequest(sub));

    // Set up ping interval
    const pingInterval = setInterval(() => {
      if (this._running && !this._stopRequested) {
        try {
          stream.write({ ping: { timestamp: Date.now() } });
        } catch {
          // Stream might be closed
        }
      }
    }, 30000);
    this._pingIntervals.push(pingInterval);

    // Handle incoming data
    stream.on('data', (response: { data?: { block_number: number; timestamp: number; data: string }; pong?: { timestamp: number } }) => {
      if (response.data) {
        try {
          const parsed = JSON.parse(response.data.data);
          const blockNumber = response.data.block_number;
          const timestamp = response.data.timestamp;

          if (sub.raw) {
            parsed._block_number = blockNumber;
            parsed._timestamp = timestamp;
            try {
              sub.callback(parsed);
            } catch {
              // Ignore callback errors
            }
            return;
          }

          // Extract events if present
          const events = parsed.events;
          if (events && Array.isArray(events) && events.length > 0) {
            let emittedEvents = false;
            for (let index = 0; index < events.length; index += 1) {
              const event = events[index];
              let user: unknown;
              let eventData: Record<string, unknown> | null = null;

              if (Array.isArray(event) && event.length >= 2) {
                user = event[0];
                if (typeof event[1] === 'object' && event[1] !== null) {
                  eventData = event[1] as Record<string, unknown>;
                }
              } else if (typeof event === 'object' && event !== null) {
                eventData = event as Record<string, unknown>;
              }

              if (eventData) {
                eventData._block_number = blockNumber;
                eventData._timestamp = timestamp;
                eventData._event_index = index;
                if (user !== undefined) {
                  eventData._user = user;
                }
                try {
                  sub.callback(eventData);
                } catch {
                  // Ignore callback errors
                }
                emittedEvents = true;
              }
            }
            if (!emittedEvents) {
              parsed._block_number = blockNumber;
              parsed._timestamp = timestamp;
              try {
                sub.callback(parsed);
              } catch {
                // Ignore callback errors
              }
            }
          } else {
            // No events structure, return raw data
            parsed._block_number = blockNumber;
            parsed._timestamp = timestamp;
            try {
              sub.callback(parsed);
            } catch {
              // Ignore callback errors
            }
          }
        } catch {
          // JSON parse error
        }
      }
    });

    stream.on('error', (err: Error) => {
      clearInterval(pingInterval);
      if (this._running && !this._stopRequested) {
        if (this._onError) {
          try {
            this._onError(err);
          } catch {
            // Ignore
          }
        }
        if (this._reconnectEnabled) {
          this._scheduleReconnect();
        }
      }
    });

    stream.on('end', () => {
      clearInterval(pingInterval);
      if (this._running && !this._stopRequested && this._reconnectEnabled) {
        this._scheduleReconnect();
      }
    });
  }

  /**
   * Start streaming raw bytes for a data subscription (StreamDataBytes fast path).
   * Payload bytes are passed through to the callback without JSON decoding.
   */
  private _streamDataBytes(sub: Subscription): void {
    if (!this._streamingClient || this._stopRequested) return;

    const metadata = this._getMetadata();

    // Create bidirectional stream
    const stream = this._streamingClient.StreamDataBytes(metadata);
    this._activeStreams.push(stream);

    // Send initial subscription request
    stream.write(this._buildSubscribeRequest(sub));

    // Set up ping interval
    const pingInterval = setInterval(() => {
      if (this._running && !this._stopRequested) {
        try {
          stream.write({ ping: { timestamp: Date.now() } });
        } catch {
          // Stream might be closed
        }
      }
    }, 30000);
    this._pingIntervals.push(pingInterval);

    // Handle incoming data
    stream.on('data', (response: { data?: { block_number: number; timestamp: number; data: Uint8Array }; pong?: { timestamp: number } }) => {
      if (response.data) {
        try {
          sub.callback(response.data as unknown as Record<string, unknown>);
        } catch {
          // Ignore callback errors
        }
      }
    });

    stream.on('error', (err: Error) => {
      clearInterval(pingInterval);
      if (this._running && !this._stopRequested) {
        if (this._onError) {
          try {
            this._onError(err);
          } catch {
            // Ignore
          }
        }
        if (this._reconnectEnabled) {
          this._scheduleReconnect();
        }
      }
    });

    stream.on('end', () => {
      clearInterval(pingInterval);
      if (this._running && !this._stopRequested && this._reconnectEnabled) {
        this._scheduleReconnect();
      }
    });
  }

  /**
   * Start streaming blocks.
   */
  private _streamBlocks(sub: Subscription): void {
    if (!this._blockClient || this._stopRequested) return;

    const metadata = this._getMetadata();
    const request = { timestamp: Date.now() };

    const stream = this._blockClient.StreamBlocks(request, metadata);
    this._activeStreams.push(stream);

    stream.on('data', (block: { data_json: string }) => {
      try {
        const data = JSON.parse(block.data_json);
        sub.callback(data);
      } catch {
        // JSON parse error
      }
    });

    stream.on('error', (err: Error) => {
      if (this._running && !this._stopRequested) {
        if (this._onError) {
          try {
            this._onError(err);
          } catch {
            // Ignore
          }
        }
        if (this._reconnectEnabled) {
          this._scheduleReconnect();
        }
      }
    });

    stream.on('end', () => {
      if (this._running && !this._stopRequested && this._reconnectEnabled) {
        this._scheduleReconnect();
      }
    });
  }

  /**
   * Start streaming L2 order book.
   */
  private _streamL2Book(sub: Subscription): void {
    if (!this._orderbookClient || this._stopRequested) return;

    const metadata = this._getMetadata();
    const request: { coin: string; n_levels: number; n_sig_figs?: number } = {
      coin: sub.coin || '',
      n_levels: sub.nLevels || 20,
    };
    if (sub.nSigFigs !== undefined) {
      request.n_sig_figs = sub.nSigFigs;
    }

    const stream = this._orderbookClient.StreamL2Book(request, metadata);
    this._activeStreams.push(stream);

    stream.on('data', (update: { coin: string; time: number; block_number: number; bids: Array<{ px: string; sz: string; n: number }>; asks: Array<{ px: string; sz: string; n: number }> }) => {
      const data = {
        coin: update.coin,
        time: update.time,
        block_number: update.block_number,
        bids: update.bids.map((l) => [l.px, l.sz, l.n]),
        asks: update.asks.map((l) => [l.px, l.sz, l.n]),
      };
      try {
        sub.callback(data);
      } catch {
        // Ignore callback errors
      }
    });

    stream.on('error', (err: Error) => {
      if (this._running && !this._stopRequested) {
        if (this._onError) {
          try {
            this._onError(err);
          } catch {
            // Ignore
          }
        }
        if (this._reconnectEnabled) {
          this._scheduleReconnect();
        }
      }
    });

    stream.on('end', () => {
      if (this._running && !this._stopRequested && this._reconnectEnabled) {
        this._scheduleReconnect();
      }
    });
  }

  /**
   * Start streaming L4 order book.
   */
  private _streamL4Book(sub: Subscription): void {
    if (!this._orderbookClient || this._stopRequested) return;

    const metadata = this._getMetadata();
    const request = { coin: sub.coin || '' };

    const stream = this._orderbookClient.StreamL4Book(request, metadata);
    this._activeStreams.push(stream);

    stream.on('data', (update: { snapshot?: { coin: string; time: number; height: number; bids: L4Order[]; asks: L4Order[] }; diff?: { time: number; height: number; data: string } }) => {
      let data: Record<string, unknown>;

      if (update.snapshot) {
        data = {
          type: 'snapshot',
          coin: update.snapshot.coin,
          time: update.snapshot.time,
          height: update.snapshot.height,
          bids: update.snapshot.bids.map(this._l4OrderToObject),
          asks: update.snapshot.asks.map(this._l4OrderToObject),
        };
      } else if (update.diff) {
        let diffData = {};
        try {
          diffData = JSON.parse(update.diff.data);
        } catch {
          // Ignore parse error
        }
        data = {
          type: 'diff',
          time: update.diff.time,
          height: update.diff.height,
          data: diffData,
        };
      } else {
        return;
      }

      try {
        sub.callback(data);
      } catch {
        // Ignore callback errors
      }
    });

    stream.on('error', (err: Error) => {
      if (this._running && !this._stopRequested) {
        if (this._onError) {
          try {
            this._onError(err);
          } catch {
            // Ignore
          }
        }
        if (this._reconnectEnabled) {
          this._scheduleReconnect();
        }
      }
    });

    stream.on('end', () => {
      if (this._running && !this._stopRequested && this._reconnectEnabled) {
        this._scheduleReconnect();
      }
    });
  }

  /**
   * Start a server-streaming order book RPC (BBO, diffs, packed, bytes, TP/SL).
   * Updates are forwarded to the callback as decoded, unless a transform is given.
   */
  private _streamOrderbookRpc(
    sub: Subscription,
    rpcName: string,
    request: Record<string, unknown>,
    transform?: (update: Record<string, unknown>) => Record<string, unknown> | null
  ): void {
    if (!this._orderbookClient || this._stopRequested) return;

    const metadata = this._getMetadata();
    const stream = this._orderbookClient[rpcName](request, metadata);
    this._activeStreams.push(stream);

    stream.on('data', (update: Record<string, unknown>) => {
      const data = transform ? transform(update) : update;
      if (data === null) return;
      try {
        sub.callback(data);
      } catch {
        // Ignore callback errors
      }
    });

    stream.on('error', (err: Error) => {
      if (this._running && !this._stopRequested) {
        if (this._onError) {
          try {
            this._onError(err);
          } catch {
            // Ignore
          }
        }
        if (this._reconnectEnabled) {
          this._scheduleReconnect();
        }
      }
    });

    stream.on('end', () => {
      if (this._running && !this._stopRequested && this._reconnectEnabled) {
        this._scheduleReconnect();
      }
    });
  }

  /**
   * Map an L4BookBytesUpdate oneof to the snapshot/diff shape used by l4Book,
   * keeping the diff payload as undecoded JSON bytes.
   */
  private _l4BytesToObject(update: Record<string, unknown>): Record<string, unknown> | null {
    const u = update as {
      snapshot?: { coin: string; time: number; height: number; bids: L4Order[]; asks: L4Order[] };
      diff?: { time: number; height: number; data: Uint8Array };
    };

    if (u.snapshot) {
      return {
        type: 'snapshot',
        coin: u.snapshot.coin,
        time: u.snapshot.time,
        height: u.snapshot.height,
        bids: u.snapshot.bids.map(this._l4OrderToObject),
        asks: u.snapshot.asks.map(this._l4OrderToObject),
      };
    }
    if (u.diff) {
      return {
        type: 'diff',
        time: u.diff.time,
        height: u.diff.height,
        data: u.diff.data, // JSON bytes, not decoded
      };
    }
    return null;
  }

  private _l4OrderToObject(order: L4Order): Record<string, unknown> {
    return {
      user: order.user,
      coin: order.coin,
      side: order.side,
      limit_px: order.limit_px,
      sz: order.sz,
      oid: order.oid,
      timestamp: order.timestamp,
      trigger_condition: order.trigger_condition,
      is_trigger: order.is_trigger,
      trigger_px: order.trigger_px,
      is_position_tpsl: order.is_position_tpsl,
      reduce_only: order.reduce_only,
      order_type: order.order_type,
      tif: order.tif,
      cloid: order.cloid,
    };
  }

  private _scheduleReconnect(): void {
    if (!this._running || this._stopRequested) return;

    if (this._maxReconnectAttempts !== null && this._reconnectAttempt >= this._maxReconnectAttempts) {
      this._running = false;
      this._setState(ConnectionState.DISCONNECTED);
      if (this._onClose) {
        try {
          this._onClose();
        } catch {
          // Ignore callback errors
        }
      }
      return;
    }

    this._reconnectAttempt += 1;
    this._setState(ConnectionState.RECONNECTING);

    if (this._onReconnect) {
      try {
        this._onReconnect(this._reconnectAttempt);
      } catch {
        // Ignore callback errors
      }
    }

    setTimeout(async () => {
      this._reconnectDelay = Math.min(
        this._reconnectDelay * GRPCStream.RECONNECT_BACKOFF_FACTOR,
        GRPCStream.MAX_RECONNECT_DELAY
      );

      if (this._running && !this._stopRequested) {
        this._cleanup();
        try {
          await this._connect();
          this._startStreams();
        } catch (err) {
          if (this._onError && err instanceof Error) {
            this._onError(err);
          }
          if (this._reconnectEnabled && this._running) {
            this._scheduleReconnect();
          }
        }
      }
    }, this._reconnectDelay);
  }

  private _startStreams(): void {
    for (const sub of this._subscriptions) {
      if (sub.bytes) {
        this._streamDataBytes(sub);
        continue;
      }
      switch (sub.streamType) {
        case 'L2_BOOK':
          this._streamL2Book(sub);
          break;
        case 'L4_BOOK':
          this._streamL4Book(sub);
          break;
        case 'BLOCKS':
          this._streamBlocks(sub);
          break;
        case 'BBO_BOOK':
          this._streamOrderbookRpc(sub, 'StreamBboBook', { coins: sub.coins ?? [] });
          break;
        case 'L2_BOOK_DIFF': {
          const request: Record<string, unknown> = {
            coins: sub.coins ?? [],
            n_levels: sub.nLevels || 20,
            skip_initial_snapshot: sub.skipInitialSnapshot ?? false,
          };
          if (sub.nSigFigs !== undefined) {
            request.n_sig_figs = sub.nSigFigs;
          }
          if (sub.mantissa !== undefined) {
            request.mantissa = sub.mantissa;
          }
          this._streamOrderbookRpc(sub, 'StreamL2BookDiff', request);
          break;
        }
        case 'L4_BOOK_UPDATES':
          this._streamOrderbookRpc(sub, 'StreamL4BookUpdates', { coins: sub.coins ?? [] });
          break;
        case 'TPSL_UPDATES':
          this._streamOrderbookRpc(sub, 'StreamTpslUpdates', { coins: sub.coins ?? [] });
          break;
        case 'L2_BOOK_PACKED': {
          const request: Record<string, unknown> = {
            coin: sub.coin || '',
            n_levels: sub.nLevels || 20,
          };
          if (sub.nSigFigs !== undefined) {
            request.n_sig_figs = sub.nSigFigs;
          }
          if (sub.mantissa !== undefined) {
            request.mantissa = sub.mantissa;
          }
          this._streamOrderbookRpc(sub, 'StreamL2BookPacked', request);
          break;
        }
        case 'BBO_BOOK_PACKED':
          this._streamOrderbookRpc(sub, 'StreamBboBookPacked', { coins: sub.coins ?? [] });
          break;
        case 'L4_BOOK_BYTES':
          this._streamOrderbookRpc(sub, 'StreamL4BookBytes', { coin: sub.coin || '' }, (u) => this._l4BytesToObject(u));
          break;
        default:
          this._streamData(sub);
          break;
      }
    }
  }

  private _cleanup(): void {
    // Clear ping intervals
    for (const interval of this._pingIntervals) {
      clearInterval(interval);
    }
    this._pingIntervals = [];

    // Cancel active streams
    for (const stream of this._activeStreams) {
      try {
        if (stream.cancel) {
          stream.cancel();
        }
      } catch {
        // Ignore
      }
    }
    this._activeStreams = [];

    // Close clients
    this._streamingClient = null;
    this._blockClient = null;
    this._orderbookClient = null;
    this._channel = null;
  }

  /**
   * Start the gRPC stream.
   */
  async start(): Promise<void> {
    this._running = true;
    this._stopRequested = false;

    try {
      await this._connect();
      this._startStreams();
    } catch (error) {
      this._setState(ConnectionState.DISCONNECTED);
      if (this._onError && error instanceof Error) {
        this._onError(error);
      }

      if (this._reconnectEnabled && this._running) {
        this._scheduleReconnect();
      } else {
        throw error;
      }
    }
  }

  /**
   * Run the gRPC stream (blocking).
   */
  async run(): Promise<void> {
    await this.start();

    // Keep running until stopped
    return new Promise<void>((resolve) => {
      const checkStop = setInterval(() => {
        if (!this._running) {
          clearInterval(checkStop);
          resolve();
        }
      }, 500);
    });
  }

  /**
   * Stop the gRPC stream.
   */
  stop(): void {
    this._running = false;
    this._stopRequested = true;
    this._cleanup();
    this._setState(ConnectionState.DISCONNECTED);

    if (this._onClose) {
      try {
        this._onClose();
      } catch {
        // Ignore callback errors
      }
    }
  }

  /**
   * Test connectivity with a ping request.
   *
   * @returns True if ping successful, false otherwise
   */
  ping(): Promise<boolean> {
    return new Promise((resolve) => {
      if (!this._streamingClient) {
        resolve(false);
        return;
      }

      const metadata = this._getMetadata();
      const request = { count: 1 };

      this._streamingClient.Ping(request, metadata, (err: Error | null, response: { count: number }) => {
        if (err) {
          resolve(false);
        } else {
          resolve(response.count === 1);
        }
      });
    });
  }

  /** Check if stream is connected. */
  get connected(): boolean {
    return this._state === ConnectionState.CONNECTED;
  }

  /** Get current connection state. */
  get state(): ConnectionState {
    return this._state;
  }

  /** Get number of reconnection attempts. */
  get reconnectAttempts(): number {
    return this._reconnectAttempt;
  }
}

// Type for L4 order
interface L4Order {
  user: string;
  coin: string;
  side: string;
  limit_px: string;
  sz: string;
  oid: number;
  timestamp: number;
  trigger_condition: string;
  is_trigger: boolean;
  trigger_px: string;
  is_position_tpsl: boolean;
  reduce_only: boolean;
  order_type: string;
  tif?: string;
  cloid?: string;
}
