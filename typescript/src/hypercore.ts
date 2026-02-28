/**
 * HyperCore JSON-RPC API Client — Blocks, trading, and real-time data.
 *
 * Access HyperCore-specific methods via QuickNode's /hypercore endpoint.
 *
 * Example:
 *     import { HyperCore } from 'hyperliquid-sdk';
 *     const hc = new HyperCore("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN");
 *     console.log(await hc.latestBlockNumber());  // Get latest block
 *     console.log(await hc.latestTrades({ count: 10 }));  // Get recent trades
 */

import { HyperliquidError } from './errors';

export interface HyperCoreOptions {
  timeout?: number;
}

/**
 * HyperCore JSON-RPC API — Blocks, trading, and real-time data.
 *
 * Access block data, trading operations, and real-time streams via JSON-RPC.
 *
 * Examples:
 *     const hc = new HyperCore("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN");
 *
 *     // Block data
 *     await hc.latestBlockNumber();
 *     await hc.getBlock(12345);
 *     await hc.getBatchBlocks(100, 102);
 *     await hc.latestBlocks({ count: 10 });
 *
 *     // Recent data (alternative to Info methods not on QuickNode)
 *     await hc.latestTrades({ count: 10 });
 *     await hc.latestOrders({ count: 10 });
 *     await hc.latestBookUpdates({ count: 10 });
 *
 *     // Discovery
 *     await hc.listDexes();
 *     await hc.listMarkets();
 *     await hc.listMarkets({ dex: "hyperliquidity" });
 */
export class HyperCore {
  private readonly _hypercoreUrl: string;
  private readonly _timeout: number;
  private _id: number = 0;

  constructor(endpoint: string, options: HyperCoreOptions = {}) {
    this._hypercoreUrl = this._buildUrl(endpoint);
    this._timeout = options.timeout ?? 30000;
  }

  private _buildUrl(url: string): string {
    const parsed = new URL(url);
    const base = `${parsed.protocol}//${parsed.host}`;
    const pathParts = parsed.pathname.split('/').filter((p) => p.length > 0);

    // Find the token (not a known path)
    const knownPaths = new Set(['info', 'hypercore', 'evm', 'nanoreth', 'ws']);
    let token: string | null = null;
    for (const part of pathParts) {
      if (!knownPaths.has(part)) {
        token = part;
        break;
      }
    }

    if (token) {
      return `${base}/${token}/hypercore`;
    }
    return `${base}/hypercore`;
  }

  /** Make a JSON-RPC 2.0 request. */
  private async _rpc<T = unknown>(
    method: string,
    params: unknown = {}
  ): Promise<T> {
    this._id += 1;
    const body = {
      jsonrpc: '2.0',
      method,
      id: this._id,
      params,
    };

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this._timeout);

    try {
      const response = await fetch(this._hypercoreUrl, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        throw new HyperliquidError(
          `Request failed with status ${response.status}`,
          {
            code: 'HTTP_ERROR',
            raw: { status: response.status, body: await response.text() },
          }
        );
      }

      const data = await response.json() as Record<string, unknown>;

      if (data.error) {
        const errorObj = data.error as Record<string, unknown>;
        const message =
          typeof errorObj === 'object' ? (errorObj.message as string) ?? String(errorObj) : String(errorObj);
        const code =
          typeof errorObj === 'object' ? String(errorObj.code ?? 'RPC_ERROR') : 'RPC_ERROR';
        throw new HyperliquidError(message, { code, raw: errorObj });
      }

      return data.result as T;
    } catch (error) {
      clearTimeout(timeoutId);

      if (error instanceof HyperliquidError) throw error;

      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new HyperliquidError(`Request timed out after ${this._timeout}ms`, {
            code: 'TIMEOUT',
            raw: { method, timeout: this._timeout },
          });
        }
        throw new HyperliquidError(`Connection failed: ${error.message}`, {
          code: 'CONNECTION_ERROR',
          raw: { method, error: error.message },
        });
      }
      throw error;
    }
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // BLOCK DATA
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Get the latest block number for a stream.
   *
   * @param stream - Stream type ("trades", "orders", "events", "book", "twap", "writer_actions")
   */
  async latestBlockNumber(stream: string = 'trades'): Promise<number> {
    return this._rpc('hl_getLatestBlockNumber', { stream });
  }

  /**
   * Get a specific block by number.
   *
   * @param blockNumber - Block number to fetch
   * @param stream - Stream type ("trades", "orders", "events", "book", "twap", "writer_actions")
   */
  async getBlock(
    blockNumber: number,
    stream: string = 'trades'
  ): Promise<Record<string, unknown>> {
    // Uses array format: [stream, block_number]
    return this._rpc('hl_getBlock', [stream, blockNumber]);
  }

  /**
   * Get a range of blocks.
   *
   * @param fromBlock - Starting block number
   * @param toBlock - Ending block number (inclusive)
   * @param stream - Stream type ("trades", "orders", "events", "book", "twap", "writer_actions")
   */
  async getBatchBlocks(
    fromBlock: number,
    toBlock: number,
    stream: string = 'trades'
  ): Promise<Array<Record<string, unknown>>> {
    return this._rpc('hl_getBatchBlocks', { stream, from: fromBlock, to: toBlock });
  }

  /**
   * Get the latest blocks for a stream.
   *
   * @param stream - Stream type ("trades", "orders", "book_updates", "twap", "events", "writer_actions")
   * @param options.count - Number of blocks to return (default: 10)
   */
  async latestBlocks(
    stream: string = 'trades',
    options: { count?: number } = {}
  ): Promise<Record<string, unknown>> {
    return this._rpc('hl_getLatestBlocks', { stream, count: options.count ?? 10 });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // RECENT DATA (Alternative to unsupported Info methods)
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Get recent trades from latest blocks.
   *
   * This is an alternative to Info.recentTrades() which is not available on QuickNode.
   *
   * @param options.count - Number of blocks to fetch (default: 10)
   * @param options.coin - Optional coin filter (e.g., "BTC", "ETH")
   */
  async latestTrades(
    options: { count?: number; coin?: string } = {}
  ): Promise<Array<Record<string, unknown>>> {
    const result = (await this.latestBlocks('trades', {
      count: options.count ?? 10,
    })) as { blocks?: Array<{ events?: Array<unknown[]> }> };
    const trades: Array<Record<string, unknown>> = [];

    for (const block of result.blocks ?? []) {
      for (const event of block.events ?? []) {
        if (Array.isArray(event) && event.length >= 2) {
          const user = event[0] as string;
          const trade = event[1] as Record<string, unknown>;
          if (options.coin === undefined || trade.coin === options.coin) {
            trades.push({ user, ...trade });
          }
        }
      }
    }
    return trades;
  }

  /**
   * Get recent order events from latest blocks.
   *
   * @param options.count - Number of blocks to fetch (default: 10)
   * @param options.coin - Optional coin filter
   */
  async latestOrders(
    options: { count?: number; coin?: string } = {}
  ): Promise<Array<Record<string, unknown>>> {
    const result = (await this.latestBlocks('orders', {
      count: options.count ?? 10,
    })) as { blocks?: Array<{ events?: Array<unknown[]> }> };
    const orders: Array<Record<string, unknown>> = [];

    for (const block of result.blocks ?? []) {
      for (const event of block.events ?? []) {
        if (Array.isArray(event) && event.length >= 2) {
          const user = event[0] as string;
          const order = event[1] as Record<string, unknown>;
          if (options.coin === undefined || order.coin === options.coin) {
            orders.push({ user, ...order });
          }
        }
      }
    }
    return orders;
  }

  /**
   * Get recent book updates from latest blocks.
   *
   * This is an alternative to Info.l2Book() for real-time updates.
   *
   * @param options.count - Number of blocks to fetch (default: 10)
   * @param options.coin - Optional coin filter
   */
  async latestBookUpdates(
    options: { count?: number; coin?: string } = {}
  ): Promise<Array<Record<string, unknown>>> {
    const result = (await this.latestBlocks('book_updates', {
      count: options.count ?? 10,
    })) as { blocks?: Array<{ events?: Array<Record<string, unknown>> }> };
    const updates: Array<Record<string, unknown>> = [];

    for (const block of result.blocks ?? []) {
      for (const event of block.events ?? []) {
        if (typeof event === 'object' && event !== null) {
          if (options.coin === undefined || event.coin === options.coin) {
            updates.push(event);
          }
        }
      }
    }
    return updates;
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // DISCOVERY
  // ═══════════════════════════════════════════════════════════════════════════

  /** List all available DEXes. */
  async listDexes(): Promise<Array<Record<string, unknown>>> {
    return this._rpc('hl_listDexes');
  }

  /**
   * List available markets.
   *
   * @param options.dex - Optional DEX filter (e.g., "hyperliquidity")
   */
  async listMarkets(
    options: { dex?: string } = {}
  ): Promise<Array<Record<string, unknown>>> {
    const params = options.dex ? { dex: options.dex } : undefined;
    return this._rpc('hl_listMarkets', params);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // ORDER QUERIES
  // ═══════════════════════════════════════════════════════════════════════════

  /** Get open orders for a user. */
  async openOrders(user: string): Promise<Array<Record<string, unknown>>> {
    return this._rpc('hl_openOrders', { user });
  }

  /** Get status of a specific order. */
  async orderStatus(user: string, oid: number): Promise<Record<string, unknown>> {
    return this._rpc('hl_orderStatus', { user, oid });
  }

  /**
   * Validate an order before signing (preflight check).
   *
   * @param coin - Trading pair (e.g., "BTC")
   * @param isBuy - True for buy, False for sell
   * @param limitPx - Limit price as string
   * @param sz - Size as string
   * @param user - User address
   * @param options.reduceOnly - Whether order is reduce-only
   * @param options.orderType - Optional order type configuration
   */
  async preflight(
    coin: string,
    isBuy: boolean,
    limitPx: string,
    sz: string,
    user: string,
    options: { reduceOnly?: boolean; orderType?: Record<string, unknown> } = {}
  ): Promise<Record<string, unknown>> {
    const params: Record<string, unknown> = {
      coin,
      isBuy,
      limitPx,
      sz,
      user,
      reduceOnly: options.reduceOnly ?? false,
    };
    if (options.orderType) {
      params.orderType = options.orderType;
    }
    return this._rpc('hl_preflight', params);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // BUILDER FEE
  // ═══════════════════════════════════════════════════════════════════════════

  /** Get maximum builder fee for a user-builder pair. */
  async getMaxBuilderFee(
    user: string,
    builder: string
  ): Promise<Record<string, unknown>> {
    return this._rpc('hl_getMaxBuilderFee', { user, builder });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // ORDER BUILDING (Returns unsigned actions for signing)
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Build an order for signing.
   *
   * @param coin - Trading pair (e.g., "BTC")
   * @param isBuy - True for buy, False for sell
   * @param limitPx - Limit price as string
   * @param sz - Size as string
   * @param user - User address
   * @param options.reduceOnly - Whether order is reduce-only
   * @param options.orderType - Optional order type configuration
   * @param options.cloid - Optional client order ID
   */
  async buildOrder(
    coin: string,
    isBuy: boolean,
    limitPx: string,
    sz: string,
    user: string,
    options: {
      reduceOnly?: boolean;
      orderType?: Record<string, unknown>;
      cloid?: string;
    } = {}
  ): Promise<Record<string, unknown>> {
    const params: Record<string, unknown> = {
      coin,
      isBuy,
      limitPx,
      sz,
      user,
      reduceOnly: options.reduceOnly ?? false,
    };
    if (options.orderType) {
      params.orderType = options.orderType;
    }
    if (options.cloid) {
      params.cloid = options.cloid;
    }
    return this._rpc('hl_buildOrder', params);
  }

  /**
   * Build a cancel action for signing.
   *
   * @param coin - Trading pair
   * @param oid - Order ID to cancel
   * @param user - User address
   */
  async buildCancel(
    coin: string,
    oid: number,
    user: string
  ): Promise<Record<string, unknown>> {
    return this._rpc('hl_buildCancel', { coin, oid, user });
  }

  /**
   * Build a modify action for signing.
   *
   * @param coin - Trading pair
   * @param oid - Order ID to modify
   * @param user - User address
   * @param options.limitPx - New limit price
   * @param options.sz - New size
   * @param options.isBuy - New side
   */
  async buildModify(
    coin: string,
    oid: number,
    user: string,
    options: { limitPx?: string; sz?: string; isBuy?: boolean } = {}
  ): Promise<Record<string, unknown>> {
    const params: Record<string, unknown> = { coin, oid, user };
    if (options.limitPx !== undefined) {
      params.limitPx = options.limitPx;
    }
    if (options.sz !== undefined) {
      params.sz = options.sz;
    }
    if (options.isBuy !== undefined) {
      params.isBuy = options.isBuy;
    }
    return this._rpc('hl_buildModify', params);
  }

  /**
   * Build a builder fee approval for signing.
   *
   * @param user - User address
   * @param builder - Builder address
   * @param maxFeeRate - Maximum fee rate (e.g., "0.001" for 0.1%)
   * @param nonce - Nonce for the action
   */
  async buildApproveBuilderFee(
    user: string,
    builder: string,
    maxFeeRate: string,
    nonce: number
  ): Promise<Record<string, unknown>> {
    return this._rpc('hl_buildApproveBuilderFee', {
      user,
      builder,
      maxFeeRate,
      nonce,
    });
  }

  /**
   * Build a builder fee revocation for signing.
   *
   * @param user - User address
   * @param builder - Builder address to revoke
   * @param nonce - Nonce for the action
   */
  async buildRevokeBuilderFee(
    user: string,
    builder: string,
    nonce: number
  ): Promise<Record<string, unknown>> {
    return this._rpc('hl_buildRevokeBuilderFee', { user, builder, nonce });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // SENDING SIGNED ACTIONS
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Send a signed order.
   *
   * @param action - The order action from buildOrder
   * @param signature - EIP-712 signature
   * @param nonce - Nonce used in signing
   */
  async sendOrder(
    action: Record<string, unknown>,
    signature: string,
    nonce: number
  ): Promise<Record<string, unknown>> {
    return this._rpc('hl_sendOrder', { action, signature, nonce });
  }

  /**
   * Send a signed cancel.
   *
   * @param action - The cancel action from buildCancel
   * @param signature - EIP-712 signature
   * @param nonce - Nonce used in signing
   */
  async sendCancel(
    action: Record<string, unknown>,
    signature: string,
    nonce: number
  ): Promise<Record<string, unknown>> {
    return this._rpc('hl_sendCancel', { action, signature, nonce });
  }

  /**
   * Send a signed modify.
   *
   * @param action - The modify action from buildModify
   * @param signature - EIP-712 signature
   * @param nonce - Nonce used in signing
   */
  async sendModify(
    action: Record<string, unknown>,
    signature: string,
    nonce: number
  ): Promise<Record<string, unknown>> {
    return this._rpc('hl_sendModify', { action, signature, nonce });
  }

  /**
   * Send a signed builder fee approval.
   *
   * @param action - The approval action from buildApproveBuilderFee
   * @param signature - EIP-712 signature
   */
  async sendApproval(
    action: Record<string, unknown>,
    signature: string
  ): Promise<Record<string, unknown>> {
    return this._rpc('hl_sendApproval', { action, signature });
  }

  /**
   * Send a signed builder fee revocation.
   *
   * @param action - The revocation action from buildRevokeBuilderFee
   * @param signature - EIP-712 signature
   */
  async sendRevocation(
    action: Record<string, unknown>,
    signature: string
  ): Promise<Record<string, unknown>> {
    return this._rpc('hl_sendRevocation', { action, signature });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // WEBSOCKET SUBSCRIPTIONS (via JSON-RPC)
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Subscribe to a WebSocket stream via JSON-RPC.
   *
   * @param subscription - Subscription parameters (type, coin, user, etc.)
   *
   * Example:
   *     await hc.subscribe({ type: "trades", coin: "BTC" });
   */
  async subscribe(
    subscription: Record<string, unknown>
  ): Promise<Record<string, unknown>> {
    return this._rpc('hl_subscribe', { subscription });
  }

  /**
   * Unsubscribe from a WebSocket stream.
   *
   * @param subscription - Subscription parameters to unsubscribe from
   */
  async unsubscribe(
    subscription: Record<string, unknown>
  ): Promise<Record<string, unknown>> {
    return this._rpc('hl_unsubscribe', { subscription });
  }
}
