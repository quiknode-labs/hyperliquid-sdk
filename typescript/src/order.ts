/**
 * Order Builder - Fluent, type-safe, beautiful.
 *
 * Build orders the way you think about them:
 *
 *     Order.buy("BTC").size(0.001).price(67000).gtc()
 *     Order.sell("ETH").size(1.5).market()
 *     Order.buy("xyz:SILVER").notional(100).ioc()
 *
 * Trigger Orders (Stop Loss / Take Profit):
 *
 *     TriggerOrder.stopLoss("BTC").size(0.001).triggerPrice(60000).market()
 *     TriggerOrder.takeProfit("ETH").size(1.0).triggerPrice(5000).limit(4990)
 */

import { Side, TIF, TpSl, OrderGrouping } from './types';
import { ValidationError } from './errors';

// Re-export enums for convenience
export { Side, TIF, TpSl, OrderGrouping };

// Forward declaration for HyperliquidSDK type
interface HyperliquidSDK {
  cancel(oid: number, asset?: string): Promise<Record<string, unknown>>;
  modify(
    oid: number,
    asset: string,
    side: string,
    price: string | null,
    size: string
  ): Promise<PlacedOrder>;
}

/**
 * Fluent order builder.
 *
 * Examples:
 *     // Simple limit buy
 *     Order.buy("BTC").size(0.001).price(67000)
 *
 *     // Market sell by notional
 *     Order.sell("ETH").notional(500).market()
 *
 *     // Post-only with reduceOnly
 *     Order.buy("BTC").size(0.01).price(65000).alo().reduceOnly()
 */
export class Order {
  readonly asset: string;
  readonly side: Side;
  private _size: string | null = null;
  private _price: string | null = null;
  private _tif: TIF = TIF.IOC;
  private _reduceOnly: boolean = false;
  private _notional: number | null = null;
  private _cloid: string | null = null;

  /** @internal */
  constructor(asset: string, side: Side) {
    this.asset = asset;
    this.side = side;
  }

  // ═══════════════ STATIC CONSTRUCTORS ═══════════════

  /** Create a buy order. */
  static buy(asset: string): Order {
    return new Order(asset, Side.BUY);
  }

  /** Create a sell order. */
  static sell(asset: string): Order {
    return new Order(asset, Side.SELL);
  }

  /** Alias for buy (perps terminology). */
  static long(asset: string): Order {
    return Order.buy(asset);
  }

  /** Alias for sell (perps terminology). */
  static short(asset: string): Order {
    return Order.sell(asset);
  }

  // ═══════════════ SIZE ═══════════════

  /** Set order size in asset units. */
  size(size: number | string): Order {
    this._size = String(size);
    return this;
  }

  /**
   * Set order size by USD notional value.
   * SDK will calculate size based on current price.
   */
  notional(usd: number): Order {
    this._notional = usd;
    return this;
  }

  // ═══════════════ PRICE ═══════════════

  /** Set limit price. */
  price(price: number | string): Order {
    this._price = String(price);
    return this;
  }

  /** Alias for price(). */
  limit(price: number | string): Order {
    return this.price(price);
  }

  // ═══════════════ TIME IN FORCE ═══════════════

  /** Set time in force. */
  tif(tif: TIF | string): Order {
    if (typeof tif === 'string') {
      const tifUpper = tif.toUpperCase();
      if (tifUpper === 'IOC') this._tif = TIF.IOC;
      else if (tifUpper === 'GTC') this._tif = TIF.GTC;
      else if (tifUpper === 'ALO') this._tif = TIF.ALO;
      else this._tif = tif as TIF;
    } else {
      this._tif = tif;
    }
    return this;
  }

  /** Immediate or cancel. */
  ioc(): Order {
    this._tif = TIF.IOC;
    return this;
  }

  /** Good till cancelled (resting order). */
  gtc(): Order {
    this._tif = TIF.GTC;
    return this;
  }

  /** Add liquidity only (post-only, maker only). */
  alo(): Order {
    this._tif = TIF.ALO;
    return this;
  }

  /** Market order (price computed automatically with slippage). */
  market(): Order {
    this._tif = TIF.MARKET;
    this._price = null; // Price will be computed by API
    return this;
  }

  // ═══════════════ OPTIONS ═══════════════

  /** Mark as reduce-only (close position only, no new exposure). */
  reduceOnly(value: boolean = true): Order {
    this._reduceOnly = value;
    return this;
  }

  /** Set client order ID for tracking. */
  cloid(clientOrderId: string): Order {
    this._cloid = clientOrderId;
    return this;
  }

  // ═══════════════ GETTERS ═══════════════

  getSize(): string | null {
    return this._size;
  }

  getPrice(): string | null {
    return this._price;
  }

  getTif(): TIF {
    return this._tif;
  }

  getNotional(): number | null {
    return this._notional;
  }

  isReduceOnly(): boolean {
    return this._reduceOnly;
  }

  getCloid(): string | null {
    return this._cloid;
  }

  isMarket(): boolean {
    return this._tif === TIF.MARKET;
  }

  // ═══════════════ BUILD ACTION ═══════════════

  /** Convert to API action format. */
  toAction(): Record<string, unknown> {
    const orderSpec: Record<string, unknown> = {
      asset: this.asset,
      side: this.side,
      size: this._size,
    };

    // Price (omit for market orders)
    if (this._tif === TIF.MARKET) {
      orderSpec.tif = 'market';
    } else {
      if (this._price) {
        orderSpec.price = this._price;
      }
      orderSpec.tif = this._tif; // enum value is already lowercase string
    }

    // Optional fields
    if (this._reduceOnly) {
      orderSpec.reduceOnly = true;
    }

    if (this._cloid) {
      orderSpec.cloid = this._cloid;
    }

    return {
      type: 'order',
      orders: [orderSpec],
    };
  }

  // ═══════════════ VALIDATION ═══════════════

  /** Validate order before sending. */
  validate(): void {
    if (!this.asset) {
      throw new ValidationError('Asset is required');
    }

    if (this._size === null && this._notional === null) {
      throw new ValidationError('Either size or notional is required', {
        guidance: 'Use .size(0.001) or .notional(100)',
      });
    }

    // Validate size is positive
    if (this._size !== null) {
      const sizeVal = parseFloat(this._size);
      if (isNaN(sizeVal)) {
        throw new ValidationError(`Invalid size value: ${this._size}`, {
          guidance: 'Size must be a valid number',
        });
      }
      if (sizeVal <= 0) {
        throw new ValidationError('Size must be positive', {
          guidance: `Got size=${this._size}, use a positive value like 0.001`,
        });
      }
    }

    // Validate notional is positive
    if (this._notional !== null && this._notional <= 0) {
      throw new ValidationError('Notional must be positive', {
        guidance: `Got notional=${this._notional}, use a positive value like 100`,
      });
    }

    // Validate price for limit orders
    if (this._tif !== TIF.MARKET && this._price === null && this._notional === null) {
      throw new ValidationError('Price is required for limit orders', {
        guidance: 'Use .price(67000) or .market() for market orders',
      });
    }

    // Validate price is positive for limit orders
    if (this._price !== null) {
      const priceVal = parseFloat(this._price);
      if (isNaN(priceVal)) {
        throw new ValidationError(`Invalid price value: ${this._price}`, {
          guidance: 'Price must be a valid number',
        });
      }
      if (priceVal <= 0) {
        throw new ValidationError('Price must be positive', {
          guidance: `Got price=${this._price}, use a positive value like 67000`,
        });
      }
    }
  }

  // ═══════════════ REPR ═══════════════

  toString(): string {
    const parts: string[] = [`Order.${this.side}('${this.asset}')`];
    if (this._size) parts.push(`.size(${this._size})`);
    if (this._notional) parts.push(`.notional(${this._notional})`);
    if (this._price) parts.push(`.price(${this._price})`);
    if (this._tif !== TIF.IOC) parts.push(`.${this._tif.toLowerCase()}()`);
    if (this._reduceOnly) parts.push('.reduceOnly()');
    return parts.join('');
  }
}

/**
 * Trigger order builder for stop-loss and take-profit orders.
 *
 * A trigger order activates when the market price reaches the trigger price:
 * - Stop Loss (SL): Triggers when price moves AGAINST your position
 * - Take Profit (TP): Triggers when price moves IN FAVOR of your position
 *
 * Once triggered, the order executes as either:
 * - Market order: Executes immediately at best available price
 * - Limit order: Rests at the specified limit price
 *
 * Examples:
 *     // Stop loss: sell when price drops to 60000 (market)
 *     TriggerOrder.stopLoss("BTC").size(0.001).triggerPrice(60000).market()
 *
 *     // Stop loss: sell at limit 59900 when price drops to 60000
 *     TriggerOrder.stopLoss("BTC").size(0.001).triggerPrice(60000).limit(59900)
 *
 *     // Take profit: sell when price rises to 80000 (market)
 *     TriggerOrder.takeProfit("BTC").size(0.001).triggerPrice(80000).market()
 */
export class TriggerOrder {
  readonly asset: string;
  readonly tpsl: TpSl;
  readonly side: Side;
  private _size: string | null = null;
  private _triggerPx: string | null = null;
  private _limitPx: string | null = null;
  private _isMarket: boolean = true;
  private _reduceOnly: boolean = false;
  private _cloid: string | null = null;

  private constructor(asset: string, tpsl: TpSl, side: Side) {
    this.asset = asset;
    this.tpsl = tpsl;
    this.side = side;
  }

  // ═══════════════ STATIC CONSTRUCTORS ═══════════════

  /**
   * Create a stop-loss trigger order.
   *
   * Stop loss triggers when price moves against your position:
   * - For longs: triggers when price FALLS to triggerPrice (sell to exit)
   * - For shorts: triggers when price RISES to triggerPrice (buy to exit)
   */
  static stopLoss(asset: string, options?: { side?: Side }): TriggerOrder {
    return new TriggerOrder(asset, TpSl.SL, options?.side ?? Side.SELL);
  }

  /**
   * Create a take-profit trigger order.
   *
   * Take profit triggers when price moves in favor of your position:
   * - For longs: triggers when price RISES to triggerPrice (sell to take profits)
   * - For shorts: triggers when price FALLS to triggerPrice (buy to take profits)
   */
  static takeProfit(asset: string, options?: { side?: Side }): TriggerOrder {
    return new TriggerOrder(asset, TpSl.TP, options?.side ?? Side.SELL);
  }

  /** Alias for stopLoss(). */
  static sl(asset: string, options?: { side?: Side }): TriggerOrder {
    return TriggerOrder.stopLoss(asset, options);
  }

  /** Alias for takeProfit(). */
  static tp(asset: string, options?: { side?: Side }): TriggerOrder {
    return TriggerOrder.takeProfit(asset, options);
  }

  // ═══════════════ SIZE ═══════════════

  /** Set order size in asset units. */
  size(size: number | string): TriggerOrder {
    this._size = String(size);
    return this;
  }

  // ═══════════════ TRIGGER PRICE ═══════════════

  /**
   * Set the trigger price.
   * The order activates when the market price reaches this level.
   */
  triggerPrice(price: number | string): TriggerOrder {
    this._triggerPx = String(price);
    return this;
  }

  /** Alias for triggerPrice(). */
  trigger(price: number | string): TriggerOrder {
    return this.triggerPrice(price);
  }

  // ═══════════════ ORDER TYPE ═══════════════

  /**
   * Execute as market order when triggered.
   * The order will fill immediately at the best available price.
   */
  market(): TriggerOrder {
    this._isMarket = true;
    this._limitPx = null;
    return this;
  }

  /**
   * Execute as limit order when triggered.
   * The order will rest at the specified limit price.
   */
  limit(price: number | string): TriggerOrder {
    this._isMarket = false;
    this._limitPx = String(price);
    return this;
  }

  // ═══════════════ OPTIONS ═══════════════

  /** Mark as reduce-only (close position only). */
  reduceOnly(value: boolean = true): TriggerOrder {
    this._reduceOnly = value;
    return this;
  }

  /** Set client order ID for tracking. */
  cloid(clientOrderId: string): TriggerOrder {
    this._cloid = clientOrderId;
    return this;
  }

  // ═══════════════ GETTERS ═══════════════

  getSize(): string | null {
    return this._size;
  }

  getTriggerPrice(): string | null {
    return this._triggerPx;
  }

  getLimitPrice(): string | null {
    return this._limitPx;
  }

  getIsMarket(): boolean {
    return this._isMarket;
  }

  isReduceOnly(): boolean {
    return this._reduceOnly;
  }

  getCloid(): string | null {
    return this._cloid;
  }

  // ═══════════════ BUILD ACTION ═══════════════

  /**
   * Convert to API action format.
   */
  toAction(grouping: OrderGrouping = OrderGrouping.NA): Record<string, unknown> {
    // For trigger orders, limit_px is always required by the API
    // For market orders, we use trigger_px as a placeholder
    const limitPx = this._isMarket ? this._triggerPx : this._limitPx;

    const orderSpec: Record<string, unknown> = {
      a: this.asset,
      b: this.side === Side.BUY,
      p: limitPx,
      s: this._size,
      r: this._reduceOnly,
      t: {
        trigger: {
          isMarket: this._isMarket,
          triggerPx: this._triggerPx,
          tpsl: this.tpsl, // enum value is already the string (e.g., 'tp' or 'sl')
        },
      },
    };

    if (this._cloid) {
      orderSpec.c = this._cloid;
    }

    return {
      type: 'order',
      orders: [orderSpec],
      grouping: grouping, // enum value is already the string (e.g., 'na', 'normalTpsl')
    };
  }

  // ═══════════════ VALIDATION ═══════════════

  /** Validate trigger order before sending. */
  validate(): void {
    if (!this.asset) {
      throw new ValidationError('Asset is required');
    }

    if (this._size === null) {
      throw new ValidationError('Size is required for trigger orders', {
        guidance: 'Use .size(0.001) to set the order size',
      });
    }

    // Validate size is positive
    const sizeVal = parseFloat(this._size);
    if (isNaN(sizeVal)) {
      throw new ValidationError(`Invalid size value: ${this._size}`);
    }
    if (sizeVal <= 0) {
      throw new ValidationError('Size must be positive', {
        guidance: `Got size=${this._size}, use a positive value`,
      });
    }

    if (this._triggerPx === null) {
      throw new ValidationError('Trigger price is required', {
        guidance: 'Use .triggerPrice(60000) to set when the order activates',
      });
    }

    // Validate trigger price is positive
    const triggerVal = parseFloat(this._triggerPx);
    if (isNaN(triggerVal)) {
      throw new ValidationError(`Invalid trigger price: ${this._triggerPx}`);
    }
    if (triggerVal <= 0) {
      throw new ValidationError('Trigger price must be positive', {
        guidance: `Got triggerPrice=${this._triggerPx}`,
      });
    }

    // Validate limit price for limit orders
    if (!this._isMarket) {
      if (this._limitPx === null) {
        throw new ValidationError('Limit price is required for limit trigger orders', {
          guidance: 'Use .limit(59900) or .market() for market execution',
        });
      }
      const limitVal = parseFloat(this._limitPx);
      if (isNaN(limitVal)) {
        throw new ValidationError(`Invalid limit price: ${this._limitPx}`);
      }
      if (limitVal <= 0) {
        throw new ValidationError('Limit price must be positive', {
          guidance: `Got limit=${this._limitPx}`,
        });
      }
    }
  }

  // ═══════════════ REPR ═══════════════

  toString(): string {
    const name = this.tpsl === TpSl.SL ? 'stopLoss' : 'takeProfit';
    const parts: string[] = [`TriggerOrder.${name}('${this.asset}')`];
    if (this._size) parts.push(`.size(${this._size})`);
    if (this._triggerPx) parts.push(`.triggerPrice(${this._triggerPx})`);
    if (this._isMarket) {
      parts.push('.market()');
    } else if (this._limitPx) {
      parts.push(`.limit(${this._limitPx})`);
    }
    if (this._reduceOnly) parts.push('.reduceOnly()');
    return parts.join('');
  }
}

/**
 * A successfully placed order with full context.
 *
 * Returned from SDK order methods, provides:
 * - Order ID and status
 * - Methods to modify/cancel
 * - Original order details
 */
export class PlacedOrder {
  readonly oid: number | null;
  readonly status: string;
  readonly asset: string;
  readonly side: string;
  readonly size: string;
  readonly price: string | null;
  readonly filledSize: string | null;
  readonly avgPrice: string | null;
  readonly rawResponse: Record<string, unknown>;
  private _sdk: HyperliquidSDK | null;

  constructor(params: {
    oid: number | null;
    status: string;
    asset: string;
    side: string;
    size: string;
    price: string | null;
    filledSize?: string | null;
    avgPrice?: string | null;
    rawResponse?: Record<string, unknown>;
    sdk?: HyperliquidSDK | null;
  }) {
    this.oid = params.oid;
    this.status = params.status;
    this.asset = params.asset;
    this.side = params.side;
    this.size = params.size;
    this.price = params.price;
    this.filledSize = params.filledSize ?? null;
    this.avgPrice = params.avgPrice ?? null;
    this.rawResponse = params.rawResponse ?? {};
    this._sdk = params.sdk ?? null;
  }

  /** Parse exchange response into PlacedOrder. */
  static fromResponse(
    response: Record<string, unknown>,
    order: Order,
    sdk?: HyperliquidSDK | null
  ): PlacedOrder {
    const responseData = response.response as Record<string, unknown> | undefined;
    const data = responseData?.data as Record<string, unknown> | undefined;
    const statuses = (data?.statuses as Array<Record<string, unknown>>) ?? [];

    let oid: number | null = null;
    let status = 'unknown';
    let filledSize: string | null = null;
    let avgPrice: string | null = null;

    if (statuses.length > 0) {
      const s = statuses[0];
      if (typeof s === 'object' && s !== null) {
        if ('resting' in s) {
          const resting = s.resting as Record<string, unknown>;
          oid = resting.oid as number;
          status = 'resting';
        } else if ('filled' in s) {
          const filled = s.filled as Record<string, unknown>;
          oid = filled.oid as number;
          status = 'filled';
          filledSize = filled.totalSz as string;
          avgPrice = filled.avgPx as string;
        } else if ('error' in s) {
          status = `error: ${s.error}`;
        }
      } else if (s === 'success') {
        status = 'success';
      }
    }

    return new PlacedOrder({
      oid,
      status,
      asset: order.asset,
      side: order.side,
      size: order.getSize() ?? '',
      price: order.getPrice(),
      filledSize,
      avgPrice,
      rawResponse: response,
      sdk,
    });
  }

  // ═══════════════ ORDER ACTIONS ═══════════════

  /** Cancel this order. */
  async cancel(): Promise<Record<string, unknown>> {
    if (!this._sdk) {
      throw new Error('Order not linked to SDK');
    }
    if (!this.oid) {
      throw new Error('No OID to cancel');
    }
    return this._sdk.cancel(this.oid, this.asset);
  }

  /** Modify this order's price and/or size. */
  async modify(options?: {
    price?: number | string;
    size?: number | string;
  }): Promise<PlacedOrder> {
    if (!this._sdk) {
      throw new Error('Order not linked to SDK');
    }
    if (!this.oid) {
      throw new Error('No OID to modify');
    }

    return this._sdk.modify(
      this.oid,
      this.asset,
      this.side,
      options?.price ? String(options.price) : this.price,
      options?.size ? String(options.size) : this.size
    );
  }

  // ═══════════════ STATUS ═══════════════

  get isResting(): boolean {
    return this.status === 'resting';
  }

  get isFilled(): boolean {
    return this.status === 'filled';
  }

  get isError(): boolean {
    return this.status.startsWith('error');
  }

  toString(): string {
    if (this.oid) {
      return `<PlacedOrder ${this.side.toUpperCase()} ${this.size} ${this.asset} @ ${this.price} | ${this.status} (oid=${this.oid})>`;
    }
    return `<PlacedOrder ${this.side.toUpperCase()} ${this.size} ${this.asset} | ${this.status}>`;
  }
}
