/**
 * WebSocket Client — Real-time data streams with automatic reconnection.
 *
 * Subscribe to trades, orders, book updates, TWAP, events, and more.
 * Handles connection management, ping/pong, and automatic reconnection.
 *
 * Example:
 *     import { Stream } from 'hyperliquid-sdk';
 *     const stream = new Stream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN");
 *     stream.trades(["BTC", "ETH"], (t) => console.log(t));
 *     stream.start();
 */

import WebSocket from 'ws';

/** Available stream types. */
export enum StreamType {
  // QuickNode-supported streams (snake_case)
  TRADES = 'trades',
  ORDERS = 'orders',
  BOOK_UPDATES = 'book_updates',
  TWAP = 'twap',
  EVENTS = 'events',
  WRITER_ACTIONS = 'writer_actions',

  // Public Hyperliquid API streams only
  L2_BOOK = 'l2Book',
  ALL_MIDS = 'allMids',
  CANDLE = 'candle',
  BBO = 'bbo',
  OPEN_ORDERS = 'openOrders',
  ORDER_UPDATES = 'orderUpdates',
  USER_EVENTS = 'userEvents',
  USER_FILLS = 'userFills',
  USER_FUNDINGS = 'userFundings',
  USER_NON_FUNDING_LEDGER = 'userNonFundingLedgerUpdates',
  CLEARINGHOUSE_STATE = 'clearinghouseState',
  ACTIVE_ASSET_CTX = 'activeAssetCtx',
  ACTIVE_ASSET_DATA = 'activeAssetData',
  TWAP_STATES = 'twapStates',
  USER_TWAP_SLICE_FILLS = 'userTwapSliceFills',
  USER_TWAP_HISTORY = 'userTwapHistory',
  NOTIFICATION = 'notification',
  WEB_DATA_3 = 'webData3',
}

/** WebSocket connection states. */
export enum ConnectionState {
  DISCONNECTED = 'disconnected',
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
  RECONNECTING = 'reconnecting',
}

export interface StreamOptions {
  onError?: (error: Error) => void;
  onClose?: () => void;
  onOpen?: () => void;
  onReconnect?: (attempt: number) => void;
  onStateChange?: (state: ConnectionState) => void;
  reconnect?: boolean;
  maxReconnectAttempts?: number;
  pingInterval?: number;
}

type Callback = (data: Record<string, unknown>) => void;

interface Subscription {
  streamType: string;
  coin?: string[];
  user?: string[];
  interval?: string;
}

/**
 * WebSocket Client — Real-time data streams with automatic reconnection.
 *
 * Features:
 * - Automatic reconnection with exponential backoff
 * - Ping/pong heartbeat to detect stale connections
 * - Thread-safe subscription management
 * - Graceful shutdown
 *
 * Streams:
 * - trades: Executed trades with price, size, direction
 * - orders: Order lifecycle events (open, filled, cancelled)
 * - book_updates: Order book changes
 * - twap: Time-weighted average price execution
 * - events: System events (funding, liquidations)
 * - writer_actions: Spot token transfers
 */
export class Stream {
  static readonly INITIAL_RECONNECT_DELAY = 1000; // ms
  static readonly MAX_RECONNECT_DELAY = 60000; // ms
  static readonly RECONNECT_BACKOFF_FACTOR = 2.0;
  static readonly PING_INTERVAL = 30000; // ms
  static readonly PING_TIMEOUT = 10000; // ms

  private readonly _wsUrl: string;
  private readonly _isQuickNode: boolean;
  private readonly _onError?: (error: Error) => void;
  private readonly _onClose?: () => void;
  private readonly _onOpen?: () => void;
  private readonly _onReconnect?: (attempt: number) => void;
  private readonly _onStateChange?: (state: ConnectionState) => void;
  private readonly _reconnectEnabled: boolean;
  private readonly _maxReconnectAttempts: number | null;
  private readonly _pingInterval: number;

  private _ws: WebSocket | null = null;
  private _running = false;
  private _state: ConnectionState = ConnectionState.DISCONNECTED;
  private _reconnectAttempt = 0;
  private _reconnectDelay = Stream.INITIAL_RECONNECT_DELAY;
  private _lastPong = 0;
  private _jsonrpcId = 0;
  private _subId = 0;
  private _pingTimer: NodeJS.Timeout | null = null;
  private _reconnectTimer: NodeJS.Timeout | null = null;

  private _subscriptions: Map<string, Subscription> = new Map();
  private _callbacks: Map<string, Callback> = new Map();
  private _channelCallbacks: Map<string, Callback[]> = new Map();

  constructor(endpoint: string, options: StreamOptions = {}) {
    const result = this._buildWsUrl(endpoint);
    this._wsUrl = result.url;
    this._isQuickNode = result.isQuickNode;
    this._onError = options.onError;
    this._onClose = options.onClose;
    this._onOpen = options.onOpen;
    this._onReconnect = options.onReconnect;
    this._onStateChange = options.onStateChange;
    this._reconnectEnabled = options.reconnect ?? true;
    this._maxReconnectAttempts = options.maxReconnectAttempts ?? null;
    this._pingInterval = options.pingInterval ?? 30000;
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

  private _buildWsUrl(url: string): { url: string; isQuickNode: boolean } {
    const parsed = new URL(url);

    // If already a ws/wss URL, use it directly
    if (parsed.protocol === 'ws:' || parsed.protocol === 'wss:') {
      const path = parsed.pathname.replace(/\/$/, '');
      if (path.endsWith('/ws')) {
        return { url, isQuickNode: !url.includes('hyperliquid.xyz') };
      }
      return { url: `${url.replace(/\/$/, '')}/ws`, isQuickNode: !url.includes('hyperliquid.xyz') };
    }

    // Convert https to wss
    const scheme = parsed.protocol === 'https:' ? 'wss:' : 'ws:';
    const base = `${scheme}//${parsed.host}`;

    // Check if this is the public Hyperliquid API
    if (parsed.host.includes('hyperliquid.xyz') || parsed.host.includes('api.hyperliquid')) {
      return { url: `${base}/ws`, isQuickNode: false };
    }

    // QuickNode endpoint - extract token and build /hypercore/ws path
    const pathParts = parsed.pathname.split('/').filter((p) => p.length > 0);
    const knownPaths = new Set(['info', 'hypercore', 'evm', 'nanoreth', 'ws']);
    let token = '';
    for (const part of pathParts) {
      if (!knownPaths.has(part)) {
        token = part;
        break;
      }
    }

    if (token) {
      return { url: `${base}/${token}/hypercore/ws`, isQuickNode: true };
    }
    return { url: `${base}/hypercore/ws`, isQuickNode: true };
  }

  private _getSubId(): string {
    this._subId += 1;
    return `sub_${this._subId}`;
  }

  private _getJsonRpcId(): number {
    this._jsonrpcId += 1;
    return this._jsonrpcId;
  }

  /**
   * Subscribe to a stream.
   */
  subscribe(
    streamType: string,
    callback: Callback,
    options: { coins?: string[]; users?: string[] } = {}
  ): string {
    const subId = this._getSubId();
    const params: Subscription = { streamType };
    if (options.coins) params.coin = options.coins;
    if (options.users) params.user = options.users;

    this._subscriptions.set(subId, params);
    this._callbacks.set(subId, callback);

    // Update channel callbacks
    if (!this._channelCallbacks.has(streamType)) {
      this._channelCallbacks.set(streamType, []);
    }
    this._channelCallbacks.get(streamType)!.push(callback);

    if (this._ws && this._state === ConnectionState.CONNECTED) {
      this._sendSubscribe(params);
    }

    return subId;
  }

  /** Subscribe to trade stream. */
  trades(coins: string[], callback: Callback): string {
    return this.subscribe(StreamType.TRADES, callback, { coins });
  }

  /** Subscribe to order stream. */
  orders(coins: string[], callback: Callback, options: { users?: string[] } = {}): string {
    return this.subscribe(StreamType.ORDERS, callback, { coins, users: options.users });
  }

  /** Subscribe to order book updates. */
  bookUpdates(coins: string[], callback: Callback): string {
    return this.subscribe(StreamType.BOOK_UPDATES, callback, { coins });
  }

  /** Subscribe to TWAP execution stream. */
  twap(coins: string[], callback: Callback): string {
    return this.subscribe(StreamType.TWAP, callback, { coins });
  }

  /** Subscribe to system events (funding, liquidations, governance). */
  events(callback: Callback): string {
    return this.subscribe(StreamType.EVENTS, callback);
  }

  /** Subscribe to spot token transfers. */
  writerActions(callback: Callback): string {
    return this.subscribe(StreamType.WRITER_ACTIONS, callback);
  }

  /** Subscribe to L2 order book snapshots. */
  l2Book(coin: string, callback: Callback): string {
    return this.subscribe(StreamType.L2_BOOK, callback, { coins: [coin] });
  }

  /** Subscribe to all mid price updates. */
  allMids(callback: Callback): string {
    return this.subscribe(StreamType.ALL_MIDS, callback);
  }

  /** Subscribe to candlestick data. */
  candle(coin: string, interval: string, callback: Callback): string {
    const subId = this._getSubId();
    const params: Subscription = { streamType: StreamType.CANDLE, coin: [coin], interval };

    this._subscriptions.set(subId, params);
    this._callbacks.set(subId, callback);

    const streamType = StreamType.CANDLE;
    if (!this._channelCallbacks.has(streamType)) {
      this._channelCallbacks.set(streamType, []);
    }
    this._channelCallbacks.get(streamType)!.push(callback);

    if (this._ws && this._state === ConnectionState.CONNECTED) {
      this._sendSubscribe(params);
    }

    return subId;
  }

  /** Subscribe to best bid/offer updates. */
  bbo(coin: string, callback: Callback): string {
    return this.subscribe(StreamType.BBO, callback, { coins: [coin] });
  }

  /** Subscribe to user's open orders. */
  openOrders(user: string, callback: Callback): string {
    return this.subscribe(StreamType.OPEN_ORDERS, callback, { users: [user] });
  }

  /** Subscribe to user's order status changes. */
  orderUpdates(user: string, callback: Callback): string {
    return this.subscribe(StreamType.ORDER_UPDATES, callback, { users: [user] });
  }

  /** Subscribe to comprehensive user events. */
  userEvents(user: string, callback: Callback): string {
    return this.subscribe(StreamType.USER_EVENTS, callback, { users: [user] });
  }

  /** Subscribe to user's trade fills. */
  userFills(user: string, callback: Callback): string {
    return this.subscribe(StreamType.USER_FILLS, callback, { users: [user] });
  }

  /** Subscribe to user's funding payment updates. */
  userFundings(user: string, callback: Callback): string {
    return this.subscribe(StreamType.USER_FUNDINGS, callback, { users: [user] });
  }

  /** Subscribe to user's ledger changes. */
  userNonFundingLedger(user: string, callback: Callback): string {
    return this.subscribe(StreamType.USER_NON_FUNDING_LEDGER, callback, { users: [user] });
  }

  /** Subscribe to user's clearinghouse state updates. */
  clearinghouseState(user: string, callback: Callback): string {
    return this.subscribe(StreamType.CLEARINGHOUSE_STATE, callback, { users: [user] });
  }

  /** Subscribe to asset context data. */
  activeAssetCtx(coin: string, callback: Callback): string {
    return this.subscribe(StreamType.ACTIVE_ASSET_CTX, callback, { coins: [coin] });
  }

  /** Subscribe to user's active asset trading parameters. */
  activeAssetData(user: string, coin: string, callback: Callback): string {
    const subId = this._getSubId();
    const params: Subscription = { streamType: StreamType.ACTIVE_ASSET_DATA, user: [user], coin: [coin] };

    this._subscriptions.set(subId, params);
    this._callbacks.set(subId, callback);

    const streamType = StreamType.ACTIVE_ASSET_DATA;
    if (!this._channelCallbacks.has(streamType)) {
      this._channelCallbacks.set(streamType, []);
    }
    this._channelCallbacks.get(streamType)!.push(callback);

    if (this._ws && this._state === ConnectionState.CONNECTED) {
      this._sendSubscribe(params);
    }

    return subId;
  }

  /** Subscribe to TWAP algorithm states. */
  twapStates(user: string, callback: Callback): string {
    return this.subscribe(StreamType.TWAP_STATES, callback, { users: [user] });
  }

  /** Subscribe to individual TWAP order slice fills. */
  userTwapSliceFills(user: string, callback: Callback): string {
    return this.subscribe(StreamType.USER_TWAP_SLICE_FILLS, callback, { users: [user] });
  }

  /** Subscribe to TWAP execution history and status. */
  userTwapHistory(user: string, callback: Callback): string {
    return this.subscribe(StreamType.USER_TWAP_HISTORY, callback, { users: [user] });
  }

  /** Subscribe to user notifications. */
  notification(user: string, callback: Callback): string {
    return this.subscribe(StreamType.NOTIFICATION, callback, { users: [user] });
  }

  /** Subscribe to aggregate user information. */
  webData3(user: string, callback: Callback): string {
    return this.subscribe(StreamType.WEB_DATA_3, callback, { users: [user] });
  }

  /** Unsubscribe from a stream. */
  unsubscribe(subId: string): void {
    const params = this._subscriptions.get(subId);
    const callback = this._callbacks.get(subId);

    if (params) {
      this._subscriptions.delete(subId);
      this._callbacks.delete(subId);

      // Remove from channel callbacks
      const streamType = params.streamType;
      const callbacks = this._channelCallbacks.get(streamType);
      if (callbacks && callback) {
        const index = callbacks.indexOf(callback);
        if (index !== -1) {
          callbacks.splice(index, 1);
        }
        if (callbacks.length === 0) {
          this._channelCallbacks.delete(streamType);
        }
      }

      if (this._ws && this._state === ConnectionState.CONNECTED) {
        this._sendUnsubscribe(params);
      }
    }
  }

  private _sendSubscribe(params: Subscription): void {
    if (!this._ws || this._ws.readyState !== WebSocket.OPEN) return;

    try {
      const streamType = params.streamType;

      if (this._isQuickNode) {
        // QuickNode JSON-RPC format
        const qnParams: Record<string, unknown> = { streamType };
        const filters: Record<string, string[]> = {};
        if (params.coin) filters.coin = params.coin;
        if (params.user) filters.user = params.user;
        if (Object.keys(filters).length > 0) {
          qnParams.filters = filters;
        }

        const msg = {
          jsonrpc: '2.0',
          method: 'hl_subscribe',
          params: qnParams,
          id: this._getJsonRpcId(),
        };
        this._ws.send(JSON.stringify(msg));
      } else {
        // Standard Hyperliquid API format
        const subscription: Record<string, unknown> = { type: streamType };

        if (params.coin) {
          for (const coin of params.coin) {
            const sub = { ...subscription, coin };
            this._ws.send(JSON.stringify({ method: 'subscribe', subscription: sub }));
          }
          return;
        }

        if (params.user) {
          for (const user of params.user) {
            const sub = { ...subscription, user };
            this._ws.send(JSON.stringify({ method: 'subscribe', subscription: sub }));
          }
          return;
        }

        this._ws.send(JSON.stringify({ method: 'subscribe', subscription }));
      }
    } catch {
      // Ignore send errors
    }
  }

  private _sendUnsubscribe(params: Subscription): void {
    if (!this._ws || this._ws.readyState !== WebSocket.OPEN) return;

    try {
      const streamType = params.streamType;

      if (this._isQuickNode) {
        const msg = {
          jsonrpc: '2.0',
          method: 'hl_unsubscribe',
          params: { streamType },
          id: this._getJsonRpcId(),
        };
        this._ws.send(JSON.stringify(msg));
      } else {
        const subscription: Record<string, unknown> = { type: streamType };
        if (params.coin) {
          for (const coin of params.coin) {
            const sub = { ...subscription, coin };
            this._ws.send(JSON.stringify({ method: 'unsubscribe', subscription: sub }));
          }
          return;
        }
        this._ws.send(JSON.stringify({ method: 'unsubscribe', subscription }));
      }
    } catch {
      // Ignore send errors
    }
  }

  private _onMessage(data: WebSocket.Data): void {
    try {
      const message = JSON.parse(data.toString());

      // Handle pong response
      const channel = message.channel ?? '';
      if (channel === 'pong' || message.type === 'pong') {
        this._lastPong = Date.now();
        return;
      }

      // Skip subscription confirmations
      if (channel === 'subscriptionResponse') return;
      if (message.jsonrpc && message.result !== undefined) return;

      // Determine stream type
      let streamType = '';
      if (this._isQuickNode) {
        const stream = message.stream ?? '';
        if (stream.startsWith('hl.')) {
          streamType = stream.slice(3);
        }
      } else {
        streamType = channel;
      }

      // Invoke callbacks
      const callbacks = this._channelCallbacks.get(streamType) ?? this._channelCallbacks.get(channel);
      if (callbacks) {
        for (const callback of callbacks) {
          try {
            callback(message);
          } catch {
            // Ignore callback errors
          }
        }
      }
    } catch {
      // Ignore parse errors
    }
  }

  private _onWsError(error: Error): void {
    if (this._onError) {
      try {
        this._onError(error);
      } catch {
        // Ignore callback errors
      }
    }
  }

  private _onWsClose(): void {
    this._setState(ConnectionState.DISCONNECTED);

    if (this._reconnectEnabled && this._running) {
      this._scheduleReconnect();
    } else if (this._onClose) {
      try {
        this._onClose();
      } catch {
        // Ignore callback errors
      }
    }
  }

  private _onWsOpen(): void {
    this._setState(ConnectionState.CONNECTED);
    this._reconnectAttempt = 0;
    this._reconnectDelay = Stream.INITIAL_RECONNECT_DELAY;
    this._lastPong = Date.now();

    // Resubscribe to all streams
    for (const params of this._subscriptions.values()) {
      this._sendSubscribe(params);
    }

    // Start ping timer
    this._startPingTimer();

    if (this._onOpen) {
      try {
        this._onOpen();
      } catch {
        // Ignore callback errors
      }
    }
  }

  private _scheduleReconnect(): void {
    if (!this._running) return;

    if (this._maxReconnectAttempts !== null && this._reconnectAttempt >= this._maxReconnectAttempts) {
      this._running = false;
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

    this._reconnectTimer = setTimeout(() => {
      this._reconnectDelay = Math.min(
        this._reconnectDelay * Stream.RECONNECT_BACKOFF_FACTOR,
        Stream.MAX_RECONNECT_DELAY
      );

      if (this._running) {
        this._connect();
      }
    }, this._reconnectDelay);
  }

  private _startPingTimer(): void {
    if (this._pingTimer) {
      clearInterval(this._pingTimer);
    }

    this._pingTimer = setInterval(() => {
      if (!this._running || this._state !== ConnectionState.CONNECTED) {
        return;
      }

      // Check for stale connection
      if (this._lastPong > 0 && Date.now() - this._lastPong > this._pingInterval + Stream.PING_TIMEOUT) {
        if (this._ws) {
          this._ws.close();
        }
        return;
      }

      // Send ping
      if (this._ws && this._ws.readyState === WebSocket.OPEN) {
        try {
          this._ws.send(JSON.stringify({ method: 'ping' }));
        } catch {
          // Ignore send errors
        }
      }
    }, this._pingInterval);
  }

  private _connect(): void {
    this._setState(ConnectionState.CONNECTING);
    this._ws = new WebSocket(this._wsUrl);

    this._ws.on('message', (data) => this._onMessage(data));
    this._ws.on('error', (error) => this._onWsError(error));
    this._ws.on('close', () => this._onWsClose());
    this._ws.on('open', () => this._onWsOpen());
  }

  /** Start the stream. */
  start(): void {
    this._running = true;
    this._connect();
  }

  /** Run the stream in background (alias for start). */
  runInBackground(): void {
    this.start();
  }

  /** Stop the stream gracefully. */
  stop(): void {
    this._running = false;

    if (this._pingTimer) {
      clearInterval(this._pingTimer);
      this._pingTimer = null;
    }

    if (this._reconnectTimer) {
      clearTimeout(this._reconnectTimer);
      this._reconnectTimer = null;
    }

    if (this._ws) {
      try {
        this._ws.close();
      } catch {
        // Ignore close errors
      }
      this._ws = null;
    }

    this._setState(ConnectionState.DISCONNECTED);

    if (this._onClose) {
      try {
        this._onClose();
      } catch {
        // Ignore callback errors
      }
    }
  }

  /** Check if stream is connected. */
  get connected(): boolean {
    return this._state === ConnectionState.CONNECTED;
  }

  /** Get current connection state. */
  get state(): ConnectionState {
    return this._state;
  }

  /** Get number of reconnection attempts since last successful connection. */
  get reconnectAttempts(): number {
    return this._reconnectAttempt;
  }
}
