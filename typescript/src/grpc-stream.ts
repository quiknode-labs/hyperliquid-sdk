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

interface Subscription {
  streamType: string;
  callback: Callback;
  coins?: string[];
  users?: string[];
  coin?: string;
  nSigFigs?: number;
  nLevels?: number;
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
 * - l2_book: Level 2 order book (aggregated price levels)
 * - l4_book: Level 4 order book (individual orders)
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
    });
  }

  /**
   * Subscribe to trade stream.
   */
  trades(coins: string[], callback: Callback): GRPCStream {
    this._addSubscription(GRPCStreamType.TRADES, callback, { coins });
    return this;
  }

  /**
   * Subscribe to order stream.
   */
  orders(coins: string[], callback: Callback, options: { users?: string[] } = {}): GRPCStream {
    this._addSubscription(GRPCStreamType.ORDERS, callback, { coins, users: options.users });
    return this;
  }

  /**
   * Subscribe to order book updates.
   */
  bookUpdates(coins: string[], callback: Callback): GRPCStream {
    this._addSubscription(GRPCStreamType.BOOK_UPDATES, callback, { coins });
    return this;
  }

  /**
   * Subscribe to TWAP execution stream.
   */
  twap(coins: string[], callback: Callback): GRPCStream {
    this._addSubscription(GRPCStreamType.TWAP, callback, { coins });
    return this;
  }

  /**
   * Subscribe to system events (funding, liquidations, governance).
   */
  events(callback: Callback): GRPCStream {
    this._addSubscription(GRPCStreamType.EVENTS, callback);
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
  writerActions(callback: Callback): GRPCStream {
    this._addSubscription(GRPCStreamType.WRITER_ACTIONS, callback);
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
   */
  l4Book(coin: string, callback: Callback): GRPCStream {
    this._addSubscription('L4_BOOK', callback, { coin });
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
   * Start streaming for a data subscription (trades, orders, etc.).
   */
  private _streamData(sub: Subscription): void {
    if (!this._streamingClient || this._stopRequested) return;

    const metadata = this._getMetadata();

    // Create bidirectional stream
    const stream = this._streamingClient.StreamData(metadata);
    this._activeStreams.push(stream);

    // Send initial subscription request
    const subscribeRequest = {
      subscribe: {
        stream_type: STREAM_TYPE_MAP[sub.streamType] || 0,
        filters: {} as Record<string, { values: string[] }>,
      },
    };

    if (sub.coins && sub.coins.length > 0) {
      subscribeRequest.subscribe.filters['coin'] = { values: sub.coins };
    }
    if (sub.users && sub.users.length > 0) {
      subscribeRequest.subscribe.filters['user'] = { values: sub.users };
    }

    stream.write(subscribeRequest);

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

          // Extract events if present
          const events = parsed.events;
          if (events && Array.isArray(events) && events.length > 0) {
            for (const event of events) {
              if (Array.isArray(event) && event.length >= 2) {
                const [user, eventData] = event;
                if (typeof eventData === 'object' && eventData !== null) {
                  eventData._block_number = blockNumber;
                  eventData._timestamp = timestamp;
                  eventData._user = user;
                  try {
                    sub.callback(eventData);
                  } catch {
                    // Ignore callback errors
                  }
                }
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
