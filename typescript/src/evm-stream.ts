/**
 * EVM WebSocket Streaming — eth_subscribe/eth_unsubscribe for HyperEVM.
 *
 * Stream EVM events via WebSocket on the /nanoreth namespace:
 * - newHeads: New block headers
 * - logs: Contract event logs
 * - newPendingTransactions: Pending transaction hashes
 *
 * Example:
 *     import { EVMStream } from 'hyperliquid-sdk';
 *     const stream = new EVMStream("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN");
 *     stream.newHeads((h) => console.log(`New block: ${h.number}`));
 *     stream.logs({ address: "0x..." }, (log) => console.log(log));
 *     stream.start();
 */

import WebSocket from 'ws';

/** EVM WebSocket subscription types. */
export enum EVMSubscriptionType {
  NEW_HEADS = 'newHeads',
  LOGS = 'logs',
  NEW_PENDING_TRANSACTIONS = 'newPendingTransactions',
}

/** Connection state enum. */
export enum ConnectionState {
  DISCONNECTED = 'disconnected',
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
  RECONNECTING = 'reconnecting',
}

export interface EVMStreamOptions {
  onError?: (error: Error) => void;
  onOpen?: () => void;
  onClose?: () => void;
  onStateChange?: (state: ConnectionState) => void;
  reconnect?: boolean;
  maxReconnectAttempts?: number;
  reconnectDelay?: number;
  pingInterval?: number;
  pingTimeout?: number;
}

interface PendingSubscription {
  type: EVMSubscriptionType;
  params: Record<string, unknown> | null;
  callback: (data: unknown) => void;
}

/**
 * EVM WebSocket Streaming — eth_subscribe/eth_unsubscribe.
 *
 * Stream EVM events via WebSocket on the /nanoreth namespace.
 *
 * Subscription types:
 * - newHeads: Fires when a new block header is appended
 * - logs: Logs matching filter criteria (address, topics)
 * - newPendingTransactions: Pending transaction hashes
 */
export class EVMStream {
  private readonly _wsUrl: string;
  private readonly _onError?: (error: Error) => void;
  private readonly _onOpen?: () => void;
  private readonly _onClose?: () => void;
  private readonly _onStateChange?: (state: ConnectionState) => void;
  private readonly _reconnect: boolean;
  private readonly _maxReconnectAttempts: number;
  private readonly _reconnectDelay: number;
  private readonly _pingInterval: number;

  private _ws: WebSocket | null = null;
  private _running = false;
  private _state: ConnectionState = ConnectionState.DISCONNECTED;
  private _reconnectCount = 0;
  private _requestId = 0;
  private _pingTimer: NodeJS.Timeout | null = null;
  private _reconnectTimer: NodeJS.Timeout | null = null;

  private _pendingSubscriptions: PendingSubscription[] = [];
  private _activeSubscriptions: Map<string, PendingSubscription> = new Map();
  private _callbacks: Map<string, (data: unknown) => void> = new Map();

  constructor(endpoint: string, options: EVMStreamOptions = {}) {
    this._wsUrl = this._buildWsUrl(endpoint);
    this._onError = options.onError;
    this._onOpen = options.onOpen;
    this._onClose = options.onClose;
    this._onStateChange = options.onStateChange;
    this._reconnect = options.reconnect ?? true;
    this._maxReconnectAttempts = options.maxReconnectAttempts ?? 10;
    this._reconnectDelay = options.reconnectDelay ?? 1000;
    this._pingInterval = options.pingInterval ?? 30000;
  }

  private _buildWsUrl(url: string): string {
    const parsed = new URL(url);
    const wsScheme = parsed.protocol === 'https:' ? 'wss:' : 'ws:';
    const base = `${wsScheme}//${parsed.host}`;

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

    // WebSocket for EVM is on /nanoreth namespace
    if (token) {
      return `${base}/${token}/nanoreth`;
    }
    return `${base}/nanoreth`;
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

  private _nextId(): number {
    this._requestId += 1;
    return this._requestId;
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // SUBSCRIPTION METHODS
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Subscribe to new block headers.
   *
   * Fires a notification each time a new header is appended to the chain.
   */
  newHeads(callback: (header: Record<string, unknown>) => void): EVMStream {
    this._pendingSubscriptions.push({
      type: EVMSubscriptionType.NEW_HEADS,
      params: null,
      callback: callback as (data: unknown) => void,
    });
    return this;
  }

  /**
   * Subscribe to contract event logs.
   *
   * @param filterParams - Filter parameters (address, topics)
   * @param callback - Function called with each matching log
   */
  logs(
    filterParams: Record<string, unknown> | null,
    callback: (log: Record<string, unknown>) => void
  ): EVMStream {
    this._pendingSubscriptions.push({
      type: EVMSubscriptionType.LOGS,
      params: filterParams,
      callback: callback as (data: unknown) => void,
    });
    return this;
  }

  /**
   * Subscribe to pending transaction hashes.
   */
  newPendingTransactions(callback: (txHash: string) => void): EVMStream {
    this._pendingSubscriptions.push({
      type: EVMSubscriptionType.NEW_PENDING_TRANSACTIONS,
      params: null,
      callback: callback as (data: unknown) => void,
    });
    return this;
  }

  private _sendSubscriptions(): void {
    if (!this._ws || this._ws.readyState !== WebSocket.OPEN) return;

    for (const sub of this._pendingSubscriptions) {
      const reqId = this._nextId();
      const params: unknown[] = [sub.type];
      if (sub.params) {
        params.push(sub.params);
      }

      const msg = {
        jsonrpc: '2.0',
        method: 'eth_subscribe',
        params,
        id: reqId,
      };

      // Store callback temporarily by request ID
      this._callbacks.set(`req_${reqId}`, sub.callback);

      try {
        this._ws.send(JSON.stringify(msg));
      } catch (error) {
        if (this._onError && error instanceof Error) {
          this._onError(error);
        }
      }
    }
  }

  /**
   * Unsubscribe from a subscription.
   */
  unsubscribe(subscriptionId: string): boolean {
    if (!this._ws || !this._activeSubscriptions.has(subscriptionId)) {
      return false;
    }

    const reqId = this._nextId();
    const msg = {
      jsonrpc: '2.0',
      method: 'eth_unsubscribe',
      params: [subscriptionId],
      id: reqId,
    };

    try {
      this._ws.send(JSON.stringify(msg));
      this._activeSubscriptions.delete(subscriptionId);
      this._callbacks.delete(subscriptionId);
      return true;
    } catch (error) {
      if (this._onError && error instanceof Error) {
        this._onError(error);
      }
      return false;
    }
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // WEBSOCKET HANDLERS
  // ═══════════════════════════════════════════════════════════════════════════

  private _onWsOpen(): void {
    this._setState(ConnectionState.CONNECTED);
    this._reconnectCount = 0;
    this._sendSubscriptions();
    if (this._onOpen) {
      try {
        this._onOpen();
      } catch {
        // Ignore callback errors
      }
    }
  }

  private _onWsClose(): void {
    this._setState(ConnectionState.DISCONNECTED);
    if (this._onClose) {
      try {
        this._onClose();
      } catch {
        // Ignore callback errors
      }
    }

    if (this._running && this._reconnect) {
      this._tryReconnect();
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

  private _onWsMessage(data: WebSocket.Data): void {
    try {
      const message = JSON.parse(data.toString());

      // Check for subscription confirmation
      if (message.id !== undefined && message.result !== undefined) {
        const reqKey = `req_${message.id}`;
        const callback = this._callbacks.get(reqKey);
        if (callback) {
          const subId = message.result;
          this._callbacks.delete(reqKey);
          this._callbacks.set(subId, callback);
          this._activeSubscriptions.set(subId, { id: subId } as unknown as PendingSubscription);
        }
        return;
      }

      // Check for subscription data
      if (message.method === 'eth_subscription') {
        const params = message.params ?? {};
        const subId = params.subscription;
        const result = params.result;

        const callback = this._callbacks.get(subId);
        if (callback) {
          try {
            callback(result);
          } catch (error) {
            if (this._onError && error instanceof Error) {
              this._onError(error);
            }
          }
        }
      }
    } catch {
      // Ignore parse errors
    }
  }

  private _tryReconnect(): void {
    if (this._reconnectCount >= this._maxReconnectAttempts) {
      this._running = false;
      return;
    }

    this._setState(ConnectionState.RECONNECTING);
    const delay = this._reconnectDelay * Math.pow(2, this._reconnectCount);
    this._reconnectCount += 1;

    this._reconnectTimer = setTimeout(() => {
      if (this._running) {
        this._connect();
      }
    }, Math.min(delay, 30000));
  }

  private _connect(): void {
    this._setState(ConnectionState.CONNECTING);
    this._ws = new WebSocket(this._wsUrl);

    this._ws.on('open', () => this._onWsOpen());
    this._ws.on('message', (data) => this._onWsMessage(data));
    this._ws.on('error', (error) => this._onWsError(error));
    this._ws.on('close', () => this._onWsClose());
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // PUBLIC CONTROL METHODS
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Start the WebSocket connection.
   */
  start(): void {
    if (this._running) return;

    this._running = true;
    this._connect();
  }

  /**
   * Stop the WebSocket connection.
   */
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
  }

  /** Get current connection state. */
  get state(): ConnectionState {
    return this._state;
  }

  /** Check if connected. */
  get connected(): boolean {
    return this._state === ConnectionState.CONNECTED;
  }

  /** Get list of active subscription IDs. */
  get subscriptions(): string[] {
    return Array.from(this._activeSubscriptions.keys());
  }
}
