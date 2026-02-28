/**
 * HyperCore Info API Client — Market data, positions, orders, and more.
 *
 * 50+ methods for querying Hyperliquid's info endpoint.
 *
 * All requests route through your QuickNode endpoint — never directly to Hyperliquid.
 *
 * Example:
 *     import { Info } from 'hyperliquid-sdk';
 *     const info = new Info("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN");
 *     console.log(await info.meta());  // Exchange metadata
 *     console.log(await info.clearinghouseState("0x..."));  // User positions
 *     console.log(await info.allMids());  // Real-time mid prices
 */

import { HyperliquidError, GeoBlockedError } from './errors';

export interface InfoOptions {
  timeout?: number;
}

/**
 * HyperCore Info API — Market data, user accounts, positions, orders.
 *
 * 50+ methods for querying Hyperliquid data.
 *
 * Examples:
 *     const info = new Info("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN");
 *
 *     // Market data
 *     await info.allMids();
 *     await info.l2Book("BTC");
 *     await info.recentTrades("ETH");
 *     await info.candles("BTC", "1h", start, end);
 *
 *     // User data
 *     await info.clearinghouseState("0x...");
 *     await info.openOrders("0x...");
 *     await info.userFills("0x...");
 */
export class Info {
  /**
   * Methods supported by QuickNode nodes with --serve-info-endpoint
   * Other methods require fallback to worker (which proxies to public HL)
   */
  static readonly QN_SUPPORTED_METHODS = new Set([
    'meta',
    'spotMeta',
    'clearinghouseState',
    'spotClearinghouseState',
    'openOrders',
    'exchangeStatus',
    'frontendOpenOrders',
    'liquidatable',
    'activeAssetData',
    'maxMarketOrderNtls',
    'vaultSummaries',
    'userVaultEquities',
    'leadingVaults',
    'extraAgents',
    'subAccounts',
    'userFees',
    'userRateLimit',
    'spotDeployState',
    'perpDeployAuctionStatus',
    'delegations',
    'delegatorSummary',
    'maxBuilderFee',
    'userToMultiSigSigners',
    'userRole',
    'perpsAtOpenInterestCap',
    'validatorL1Votes',
    'marginTable',
    'perpDexs',
    'webData2',
  ]);

  /** Worker URL for methods not supported by QuickNode */
  static readonly DEFAULT_WORKER_INFO = 'https://send.hyperliquidapi.com/info';

  private readonly _infoUrl: string;
  private readonly _workerInfoUrl: string;
  private readonly _timeout: number;

  constructor(endpoint: string, options: InfoOptions = {}) {
    this._infoUrl = this._buildInfoUrl(endpoint);
    this._workerInfoUrl = Info.DEFAULT_WORKER_INFO;
    this._timeout = options.timeout ?? 30000;
  }

  private _buildInfoUrl(url: string): string {
    const parsed = new URL(url);
    const base = `${parsed.protocol}//${parsed.host}`;
    const pathParts = parsed.pathname
      .split('/')
      .filter((p) => p.length > 0);

    // Check if URL already ends with /info
    if (pathParts.length > 0 && pathParts[pathParts.length - 1] === 'info') {
      return url.replace(/\/$/, '');
    }

    // Find the token (not a known path like info, evm, etc.)
    const knownPaths = new Set(['info', 'hypercore', 'evm', 'nanoreth', 'ws', 'send']);
    let token: string | null = null;
    for (const part of pathParts) {
      if (!knownPaths.has(part)) {
        token = part;
        break;
      }
    }

    if (token) {
      return `${base}/${token}/info`;
    }
    return `${base}/info`;
  }

  /**
   * POST to /info endpoint.
   *
   * For QN-supported methods → routes to user's QN endpoint.
   * For unsupported methods (allMids, l2Book, etc.) → routes to worker.
   * The worker proxies these to the public HL endpoint.
   */
  private async _post<T = unknown>(body: Record<string, unknown>): Promise<T> {
    const reqType = (body.type as string) ?? '';

    // Route based on method support
    const url = Info.QN_SUPPORTED_METHODS.has(reqType)
      ? this._infoUrl
      : this._workerInfoUrl;

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

      if (!response.ok) {
        // Check for geo-blocking (403 with specific message)
        if (response.status === 403) {
          try {
            const errorData = await response.json();
            const errorStr = JSON.stringify(errorData).toLowerCase();
            if (errorStr.includes('restricted') || errorStr.includes('jurisdiction')) {
              throw new GeoBlockedError(errorData as Record<string, unknown>);
            }
          } catch (e) {
            if (e instanceof GeoBlockedError) throw e;
            // If JSON parsing fails, check raw text
            const text = await response.text();
            if (text.toLowerCase().includes('restricted') || text.toLowerCase().includes('jurisdiction')) {
              throw new GeoBlockedError({ error: text });
            }
          }
        }
        throw new HyperliquidError(`Request failed with status ${response.status}`, {
          code: 'HTTP_ERROR',
          raw: { status: response.status, body: await response.text() },
        });
      }

      return (await response.json()) as T;
    } catch (error) {
      clearTimeout(timeoutId);

      if (error instanceof HyperliquidError) throw error;

      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new HyperliquidError(`Request timed out after ${this._timeout}ms`, {
            code: 'TIMEOUT',
            raw: { type: reqType, timeout: this._timeout },
          });
        }
        throw new HyperliquidError(`Connection failed: ${error.message}`, {
          code: 'CONNECTION_ERROR',
          raw: { type: reqType, error: error.message },
        });
      }
      throw error;
    }
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // MARKET DATA
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Get all asset mid prices.
   *
   * @param options.dex - The perp dex name for HIP-3. Defaults to empty string for first perp dex.
   *                      Spot mids are only included with the first perp dex.
   */
  async allMids(options: { dex?: string } = {}): Promise<Record<string, string>> {
    const body: Record<string, unknown> = { type: 'allMids' };
    if (options.dex !== undefined) {
      body.dex = options.dex;
    }
    return this._post(body);
  }

  /**
   * Get Level 2 order book for an asset.
   *
   * @param coin - Asset name ("BTC", "ETH")
   * @param options.nSigFigs - Number of significant figures for price bucketing (2-5)
   * @param options.mantissa - Bucketing mantissa multiplier (1, 2, or 5)
   */
  async l2Book(
    coin: string,
    options: { nSigFigs?: number; mantissa?: number } = {}
  ): Promise<Record<string, unknown>> {
    const body: Record<string, unknown> = { type: 'l2Book', coin };
    if (options.nSigFigs !== undefined) {
      body.nSigFigs = options.nSigFigs;
    }
    if (options.mantissa !== undefined) {
      body.mantissa = options.mantissa;
    }
    return this._post(body);
  }

  /** Get recent trades for an asset. */
  async recentTrades(coin: string): Promise<Array<Record<string, unknown>>> {
    return this._post({ type: 'recentTrades', coin });
  }

  /**
   * Get historical OHLCV candlestick data.
   *
   * @param coin - Asset name ("BTC", "ETH")
   * @param interval - Candle interval ("1m", "5m", "15m", "1h", "4h", "1d")
   * @param startTime - Start timestamp in milliseconds
   * @param endTime - End timestamp in milliseconds
   */
  async candles(
    coin: string,
    interval: string,
    startTime: number,
    endTime: number
  ): Promise<Array<Record<string, unknown>>> {
    return this._post({
      type: 'candleSnapshot',
      req: { coin, interval, startTime, endTime },
    });
  }

  /** Get historical funding rates for an asset. */
  async fundingHistory(
    coin: string,
    startTime: number,
    endTime?: number
  ): Promise<Array<Record<string, unknown>>> {
    const body: Record<string, unknown> = { type: 'fundingHistory', coin, startTime };
    if (endTime !== undefined) {
      body.endTime = endTime;
    }
    return this._post(body);
  }

  /** Get predicted funding rates for all assets. */
  async predictedFundings(): Promise<Array<Record<string, unknown>>> {
    return this._post({ type: 'predictedFundings' });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // METADATA
  // ═══════════════════════════════════════════════════════════════════════════

  /** Get exchange metadata including assets and margin configurations. */
  async meta(): Promise<Record<string, unknown>> {
    return this._post({ type: 'meta' });
  }

  /** Get spot trading metadata. */
  async spotMeta(): Promise<Record<string, unknown>> {
    return this._post({ type: 'spotMeta' });
  }

  /** Get metadata + real-time asset context (funding rates, open interest). */
  async metaAndAssetCtxs(): Promise<Record<string, unknown>> {
    return this._post({ type: 'metaAndAssetCtxs' });
  }

  /** Get spot metadata + real-time asset context. */
  async spotMetaAndAssetCtxs(): Promise<Record<string, unknown>> {
    return this._post({ type: 'spotMetaAndAssetCtxs' });
  }

  /** Get current exchange status. */
  async exchangeStatus(): Promise<Record<string, unknown>> {
    return this._post({ type: 'exchangeStatus' });
  }

  /** Get perpetual DEX information. */
  async perpDexs(): Promise<Array<Record<string, unknown>>> {
    return this._post({ type: 'perpDexs' });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // USER ACCOUNT
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Get user's perpetual positions and margin info.
   *
   * @param user - User address
   * @param options.dex - The perp dex name. Defaults to empty string for first perp dex.
   */
  async clearinghouseState(
    user: string,
    options: { dex?: string } = {}
  ): Promise<Record<string, unknown>> {
    const body: Record<string, unknown> = { type: 'clearinghouseState', user };
    if (options.dex !== undefined) {
      body.dex = options.dex;
    }
    return this._post(body);
  }

  /** Get user's spot token balances. */
  async spotClearinghouseState(user: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'spotClearinghouseState', user });
  }

  /**
   * Get user's open orders.
   *
   * @param user - User address
   * @param options.dex - The perp dex name for HIP-3. Defaults to empty string for first perp dex.
   */
  async openOrders(
    user: string,
    options: { dex?: string } = {}
  ): Promise<Array<Record<string, unknown>>> {
    const body: Record<string, unknown> = { type: 'openOrders', user };
    if (options.dex !== undefined) {
      body.dex = options.dex;
    }
    return this._post(body);
  }

  /**
   * Get user's open orders with enhanced info.
   *
   * @param user - User address
   * @param options.dex - The perp dex name for HIP-3. Defaults to empty string for first perp dex.
   */
  async frontendOpenOrders(
    user: string,
    options: { dex?: string } = {}
  ): Promise<Array<Record<string, unknown>>> {
    const body: Record<string, unknown> = { type: 'frontendOpenOrders', user };
    if (options.dex !== undefined) {
      body.dex = options.dex;
    }
    return this._post(body);
  }

  /**
   * Get status of a specific order.
   *
   * @param user - User address
   * @param oid - Order ID
   * @param options.dex - The perp dex name for HIP-3. Defaults to empty string for first perp dex.
   */
  async orderStatus(
    user: string,
    oid: number,
    options: { dex?: string } = {}
  ): Promise<Record<string, unknown>> {
    const body: Record<string, unknown> = { type: 'orderStatus', user, oid };
    if (options.dex !== undefined) {
      body.dex = options.dex;
    }
    return this._post(body);
  }

  /** Get user's historical orders. */
  async historicalOrders(user: string): Promise<Array<Record<string, unknown>>> {
    return this._post({ type: 'historicalOrders', user });
  }

  /** Get user's trade fills. */
  async userFills(
    user: string,
    options: { aggregateByTime?: boolean } = {}
  ): Promise<Array<Record<string, unknown>>> {
    const body: Record<string, unknown> = { type: 'userFills', user };
    if (options.aggregateByTime) {
      body.aggregateByTime = true;
    }
    return this._post(body);
  }

  /** Get user's trade fills within a time range. */
  async userFillsByTime(
    user: string,
    startTime: number,
    endTime?: number
  ): Promise<Array<Record<string, unknown>>> {
    const body: Record<string, unknown> = { type: 'userFillsByTime', user, startTime };
    if (endTime !== undefined) {
      body.endTime = endTime;
    }
    return this._post(body);
  }

  /** Get user's funding payments. */
  async userFunding(
    user: string,
    options: { startTime?: number; endTime?: number } = {}
  ): Promise<Array<Record<string, unknown>>> {
    const body: Record<string, unknown> = { type: 'userFunding', user };
    if (options.startTime !== undefined) {
      body.startTime = options.startTime;
    }
    if (options.endTime !== undefined) {
      body.endTime = options.endTime;
    }
    return this._post(body);
  }

  /** Get user's fee structure (maker/taker rates). */
  async userFees(user: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'userFees', user });
  }

  /** Get user's rate limit status. */
  async userRateLimit(user: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'userRateLimit', user });
  }

  /** Get user's portfolio history. */
  async portfolio(user: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'portfolio', user });
  }

  /** Get comprehensive account snapshot. */
  async webData2(user: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'webData2', user });
  }

  /** Get user's sub-accounts. */
  async subAccounts(user: string): Promise<Array<Record<string, unknown>>> {
    return this._post({ type: 'subAccounts', user });
  }

  /** Get user's extra agents (API keys). */
  async extraAgents(user: string): Promise<Array<Record<string, unknown>>> {
    return this._post({ type: 'extraAgents', user });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // BATCH QUERIES
  // ═══════════════════════════════════════════════════════════════════════════

  /** Get clearinghouse states for multiple users in one call. */
  async batchClearinghouseStates(users: string[]): Promise<Array<Record<string, unknown>>> {
    return this._post({ type: 'batchClearinghouseStates', users });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // VAULTS
  // ═══════════════════════════════════════════════════════════════════════════

  /** Get summaries of all vaults. */
  async vaultSummaries(): Promise<Array<Record<string, unknown>>> {
    return this._post({ type: 'vaultSummaries' });
  }

  /** Get vault details. */
  async vaultDetails(
    vaultAddress: string,
    user?: string
  ): Promise<Record<string, unknown>> {
    const body: Record<string, unknown> = { type: 'vaultDetails', vaultAddress };
    if (user) {
      body.user = user;
    }
    return this._post(body);
  }

  /** Get vaults that user is leading. */
  async leadingVaults(user: string): Promise<Array<Record<string, unknown>>> {
    return this._post({ type: 'leadingVaults', user });
  }

  /** Get user's vault equities. */
  async userVaultEquities(user: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'userVaultEquities', user });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // DELEGATION / STAKING
  // ═══════════════════════════════════════════════════════════════════════════

  /** Get user's delegations. */
  async delegations(user: string): Promise<Array<Record<string, unknown>>> {
    return this._post({ type: 'delegations', user });
  }

  /** Get user's delegation history. */
  async delegatorHistory(user: string): Promise<Array<Record<string, unknown>>> {
    return this._post({ type: 'delegatorHistory', user });
  }

  /** Get user's delegation rewards. */
  async delegatorRewards(user: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'delegatorRewards', user });
  }

  /** Get user's delegation summary. */
  async delegatorSummary(user: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'delegatorSummary', user });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // TOKENS / SPOT
  // ═══════════════════════════════════════════════════════════════════════════

  /** Get token details. */
  async tokenDetails(tokenId: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'tokenDetails', tokenId });
  }

  /** Get spot deployment state for user. */
  async spotDeployState(user: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'spotDeployState', user });
  }

  /** Get list of liquidatable positions. */
  async liquidatable(): Promise<Array<Record<string, unknown>>> {
    return this._post({ type: 'liquidatable' });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // OTHER
  // ═══════════════════════════════════════════════════════════════════════════

  /** Get maximum builder fee for a user-builder pair. */
  async maxBuilderFee(user: string, builder: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'maxBuilderFee', user, builder });
  }

  /** Get user's referral information. */
  async referral(user: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'referral', user });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // ADDITIONAL METHODS
  // ═══════════════════════════════════════════════════════════════════════════

  /** Get user's active asset trading parameters. */
  async activeAssetData(user: string, coin: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'activeAssetData', user, coin });
  }

  /** Get account type (user, agent, vault, or sub-account). */
  async userRole(user: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'userRole', user });
  }

  /** Get user's non-funding ledger updates (deposits, withdrawals, transfers). */
  async userNonFundingLedgerUpdates(
    user: string,
    options: { startTime?: number; endTime?: number } = {}
  ): Promise<Array<Record<string, unknown>>> {
    const body: Record<string, unknown> = { type: 'userNonFundingLedgerUpdates', user };
    if (options.startTime !== undefined) {
      body.startTime = options.startTime;
    }
    if (options.endTime !== undefined) {
      body.endTime = options.endTime;
    }
    return this._post(body);
  }

  /** Get user's TWAP slice fills. */
  async userTwapSliceFills(
    user: string,
    options: { limit?: number } = {}
  ): Promise<Array<Record<string, unknown>>> {
    return this._post({
      type: 'userTwapSliceFills',
      user,
      limit: options.limit ?? 500,
    });
  }

  /** Get multi-sig signers for a user. */
  async userToMultiSigSigners(user: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'userToMultiSigSigners', user });
  }

  /** Get gossip root IPs for the network. */
  async gossipRootIps(): Promise<string[]> {
    return this._post({ type: 'gossipRootIps' });
  }

  /** Get maximum market order notionals per asset. */
  async maxMarketOrderNtls(): Promise<Record<string, unknown>> {
    return this._post({ type: 'maxMarketOrderNtls' });
  }

  /** Get perpetual deploy auction status. */
  async perpDeployAuctionStatus(): Promise<Record<string, unknown>> {
    return this._post({ type: 'perpDeployAuctionStatus' });
  }

  /** Get perps that are at their open interest cap. */
  async perpsAtOpenInterestCap(): Promise<string[]> {
    return this._post({ type: 'perpsAtOpenInterestCap' });
  }

  /** Get L1 validator votes. */
  async validatorL1Votes(): Promise<Record<string, unknown>> {
    return this._post({ type: 'validatorL1Votes' });
  }

  /** Get list of approved builders for a user. */
  async approvedBuilders(user: string): Promise<Array<Record<string, unknown>>> {
    return this._post({ type: 'approvedBuilders', user });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // TWAP
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Get user's TWAP order history.
   *
   * Returns up to 2000 most recent TWAP orders with status.
   *
   * @param user - User address
   */
  async userTwapHistory(user: string): Promise<Array<Record<string, unknown>>> {
    return this._post({ type: 'userTwapHistory', user });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // BORROW/LEND
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Get user's borrow/lend positions.
   *
   * Returns token-indexed borrow/supply positions, health status, health factor.
   *
   * @param user - User address
   */
  async borrowLendUserState(user: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'borrowLendUserState', user });
  }

  /**
   * Get borrow/lend reserve state for a token.
   *
   * Returns yearly rates, balance, utilization, oracle price, LTV, total supplied/borrowed.
   *
   * @param token - Token index
   */
  async borrowLendReserveState(token: number): Promise<Record<string, unknown>> {
    return this._post({ type: 'borrowLendReserveState', token });
  }

  /**
   * Get borrow/lend reserve states for all tokens.
   *
   * Returns all reserve states with rates and utilization.
   */
  async allBorrowLendReserveStates(): Promise<Array<Record<string, unknown>>> {
    return this._post({ type: 'allBorrowLendReserveStates' });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // ACCOUNT ABSTRACTION
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Get user's account abstraction mode.
   *
   * Returns abstraction mode: unifiedAccount, portfolioMargin, dexAbstraction, default, disabled.
   *
   * @param user - User address
   */
  async userAbstraction(user: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'userAbstraction', user });
  }

  /**
   * Get user's DEX abstraction eligibility.
   *
   * @param user - User address
   */
  async userDexAbstraction(user: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'userDexAbstraction', user });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // EXTENDED PERP DEX INFO
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Get consolidated universe, margin tables, asset contexts across all DEXs.
   *
   * More comprehensive than meta() - includes all perp DEXs.
   */
  async allPerpMetas(): Promise<Record<string, unknown>> {
    return this._post({ type: 'allPerpMetas' });
  }

  /** Get asset classifications for perps. */
  async perpCategories(): Promise<Record<string, unknown>> {
    return this._post({ type: 'perpCategories' });
  }

  /**
   * Get metadata descriptions for a perp.
   *
   * @param asset - Asset index
   */
  async perpAnnotation(asset: number): Promise<Record<string, unknown>> {
    return this._post({ type: 'perpAnnotation', asset });
  }

  /**
   * Get OI caps and transfer limits for builder-deployed markets.
   *
   * @param dex - DEX name
   */
  async perpDexLimits(dex: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'perpDexLimits', dex });
  }

  /**
   * Get total net deposits for builder-deployed markets.
   *
   * @param dex - DEX name
   */
  async perpDexStatus(dex: string): Promise<Record<string, unknown>> {
    return this._post({ type: 'perpDexStatus', dex });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // SPOT DEPLOYMENT
  // ═══════════════════════════════════════════════════════════════════════════

  /** Get Dutch auction status for spot pair deployments. */
  async spotPairDeployAuctionStatus(): Promise<Record<string, unknown>> {
    return this._post({ type: 'spotPairDeployAuctionStatus' });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // ALIGNED QUOTE TOKEN
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Get aligned quote token information.
   *
   * Returns alignment status, first aligned timestamp, EVM minted supply,
   * daily amounts, predicted rate.
   *
   * @param token - Token index
   */
  async alignedQuoteTokenInfo(token: number): Promise<Record<string, unknown>> {
    return this._post({ type: 'alignedQuoteTokenInfo', token });
  }
}
