/**
 * Hyperliquid SDK Errors - Clear, actionable, helpful.
 *
 * Every error tells you:
 * 1. What went wrong
 * 2. Why it happened
 * 3. How to fix it
 */

export interface ErrorOptions {
  code?: string;
  guidance?: string;
  raw?: Record<string, unknown>;
}

/**
 * Base error for all SDK errors.
 */
export class HyperliquidError extends Error {
  readonly code: string | null;
  readonly guidance: string | null;
  readonly raw: Record<string, unknown>;

  constructor(message: string, options: ErrorOptions = {}) {
    super(HyperliquidError.formatMessage(message, options));
    this.name = 'HyperliquidError';
    this.code = options.code ?? null;
    this.guidance = options.guidance ?? null;
    this.raw = options.raw ?? {};

    // Maintains proper stack trace in V8 environments
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, this.constructor);
    }
  }

  private static formatMessage(message: string, options: ErrorOptions): string {
    const parts: string[] = [];
    if (options.code) {
      parts.push(`[${options.code}]`);
    }
    parts.push(message);
    if (options.guidance) {
      parts.push(`\n  Hint: ${options.guidance}`);
    }
    return parts.join(' ');
  }
}

/**
 * Error during order/action building phase.
 */
export class BuildError extends HyperliquidError {
  constructor(message: string, options: ErrorOptions = {}) {
    super(message, options);
    this.name = 'BuildError';
  }
}

/**
 * Error when sending signed transaction to Hyperliquid.
 */
export class SendError extends HyperliquidError {
  constructor(message: string, options: ErrorOptions = {}) {
    super(message, options);
    this.name = 'SendError';
  }
}

/**
 * Builder fee approval required or fee exceeds approved amount.
 */
export class ApprovalError extends HyperliquidError {
  readonly approvalData: Record<string, unknown>;

  constructor(
    message: string = 'Builder fee approval required',
    options: ErrorOptions & { approvalData?: Record<string, unknown> } = {}
  ) {
    super(message, options);
    this.name = 'ApprovalError';
    this.approvalData = options.approvalData ?? {};
  }
}

/**
 * Invalid order parameters (price tick, size decimals, etc.).
 */
export class ValidationError extends HyperliquidError {
  constructor(message: string, options: ErrorOptions = {}) {
    super(message, options);
    this.name = 'ValidationError';
  }
}

/**
 * Signature verification failed.
 */
export class SignatureError extends HyperliquidError {
  constructor(message: string, options: ErrorOptions = {}) {
    super(message, options);
    this.name = 'SignatureError';
  }
}

/**
 * No position to close.
 */
export class NoPositionError extends HyperliquidError {
  constructor(asset: string, _user?: string) {
    super(`No open position for ${asset}`, {
      code: 'NO_POSITION',
      guidance: `You don't have a ${asset} position to close. Check your positions via sdk.positions() or the Hyperliquid app.`,
    });
    this.name = 'NoPositionError';
  }
}

/**
 * Order not found for cancel/modify.
 */
export class OrderNotFoundError extends HyperliquidError {
  constructor(oid: number | string) {
    super(`Order ${oid} not found`, {
      code: 'ORDER_NOT_FOUND',
      guidance: 'The order may have already been filled or cancelled.',
    });
    this.name = 'OrderNotFoundError';
  }
}

/**
 * Access denied due to geographic restrictions.
 *
 * Hyperliquid blocks access from certain jurisdictions including the US.
 * The SDK detects this and raises a clear GeoBlockedError.
 */
export class GeoBlockedError extends HyperliquidError {
  readonly jurisdictions: string;
  readonly note: string;

  constructor(data: Record<string, unknown>) {
    const errorInfo = data.error as Record<string, unknown> | string | undefined;

    let message: string;
    let jurisdictions: string = '';
    let note: string = '';

    if (typeof errorInfo === 'object' && errorInfo !== null) {
      message = (errorInfo.message as string) || 'Access denied from restricted jurisdiction';
      jurisdictions = (errorInfo.restricted_jurisdictions as string) || '';
      note = (errorInfo.note as string) || '';
    } else {
      message = errorInfo ? String(errorInfo) : 'Access denied from restricted jurisdiction';
    }

    const guidanceParts: string[] = [
      "Your IP is blocked by Hyperliquid's geo-restrictions.",
    ];

    if (jurisdictions) {
      const restrictedList = jurisdictions.split(', ').slice(0, 5);
      guidanceParts.push(`Blocked regions include: ${restrictedList.join(', ')}...`);
    }

    if (note) {
      guidanceParts.push(`Note: ${note}`);
    }

    super(message, {
      code: 'GEO_BLOCKED',
      guidance: guidanceParts.join(' '),
      raw: data,
    });

    this.name = 'GeoBlockedError';
    this.jurisdictions = jurisdictions;
    this.note = note;
  }
}

/**
 * Not enough margin for this order.
 */
export class InsufficientMarginError extends HyperliquidError {
  constructor(rawError: string = '') {
    super('Insufficient margin for this order', {
      code: 'INSUFFICIENT_MARGIN',
      guidance: 'Deposit more collateral or reduce order size. Check your available margin at https://app.hyperliquid.xyz',
      raw: rawError ? { rawHlError: rawError } : {},
    });
    this.name = 'InsufficientMarginError';
  }
}

/**
 * Leverage configuration conflict.
 */
export class LeverageError extends HyperliquidError {
  constructor(rawError: string = '') {
    super('Leverage configuration incompatible with this order', {
      code: 'LEVERAGE_CONFLICT',
      guidance: 'Close your existing position or update leverage first. You may need to switch between cross/isolated margin modes.',
      raw: rawError ? { rawHlError: rawError } : {},
    });
    this.name = 'LeverageError';
  }
}

/**
 * Rate limit exceeded.
 */
export class RateLimitError extends HyperliquidError {
  constructor(rawError: string = '') {
    super('Rate limit exceeded', {
      code: 'RATE_LIMITED',
      guidance: 'Wait a moment and retry. Consider batching multiple orders.',
      raw: rawError ? { rawHlError: rawError } : {},
    });
    this.name = 'RateLimitError';
  }
}

/**
 * Maximum open orders exceeded.
 */
export class MaxOrdersError extends HyperliquidError {
  constructor(rawError: string = '') {
    super('Maximum open orders exceeded', {
      code: 'MAX_ORDERS_EXCEEDED',
      guidance: 'Cancel some existing orders before placing new ones. Use sdk.openOrders() to see your orders and sdk.cancelAll() to clear them.',
      raw: rawError ? { rawHlError: rawError } : {},
    });
    this.name = 'MaxOrdersError';
  }
}

/**
 * Reduce-only order would increase position.
 */
export class ReduceOnlyError extends HyperliquidError {
  constructor(rawError: string = '') {
    super('Reduce-only order would increase position', {
      code: 'REDUCE_ONLY_VIOLATION',
      guidance: 'Remove reduceOnly flag or ensure the order direction actually reduces your current position.',
      raw: rawError ? { rawHlError: rawError } : {},
    });
    this.name = 'ReduceOnlyError';
  }
}

/**
 * Duplicate client order ID.
 */
export class DuplicateOrderError extends HyperliquidError {
  constructor(rawError: string = '') {
    super('Duplicate order (client order ID already exists)', {
      code: 'DUPLICATE_ORDER',
      guidance: 'Use a unique cloid for each order, or omit it to auto-generate.',
      raw: rawError ? { rawHlError: rawError } : {},
    });
    this.name = 'DuplicateOrderError';
  }
}

/**
 * User/wallet not recognized by Hyperliquid.
 */
export class UserNotFoundError extends HyperliquidError {
  constructor(rawError: string = '') {
    super('Wallet not recognized by Hyperliquid', {
      code: 'USER_NOT_FOUND',
      guidance: 'Deposit USDC to your Hyperliquid account first at https://app.hyperliquid.xyz - deposits go through the Arbitrum bridge.',
      raw: rawError ? { rawHlError: rawError } : {},
    });
    this.name = 'UserNotFoundError';
  }
}

/**
 * Account needs a deposit before trading.
 */
export class MustDepositError extends HyperliquidError {
  constructor(rawError: string = '') {
    super('Your account needs a deposit before trading', {
      code: 'MUST_DEPOSIT_FIRST',
      guidance: 'Go to https://app.hyperliquid.xyz and deposit USDC, then try again.',
      raw: rawError ? { rawHlError: rawError } : {},
    });
    this.name = 'MustDepositError';
  }
}

/**
 * Nonce invalid or expired.
 */
export class InvalidNonceError extends HyperliquidError {
  constructor(rawError: string = '') {
    super('Nonce invalid or expired', {
      code: 'INVALID_NONCE',
      guidance: 'The SDK handles nonces automatically. If you see this error, your request may have timed out. Retry.',
      raw: rawError ? { rawHlError: rawError } : {},
    });
    this.name = 'InvalidNonceError';
  }
}

/**
 * Parse API error response into appropriate exception.
 */
export function parseApiError(data: Record<string, unknown>, statusCode: number = 0): HyperliquidError {
  const error = data.error ?? 'UNKNOWN_ERROR';
  let message = (data.message as string) ?? (error ? String(error) : 'Unknown error');
  const guidance = data.guidance as string | undefined;
  const raw = data;
  const rawHlError = (data.rawHlError as string) ?? '';

  // Handle nested error object (geo-blocking returns {"error": {...}})
  let errorCode: string;
  if (typeof error === 'object' && error !== null) {
    const errorObj = error as Record<string, unknown>;
    if (errorObj.code === 403 || String(errorObj.message ?? '').toLowerCase().includes('restricted jurisdiction')) {
      return new GeoBlockedError(data);
    }
    message = (errorObj.message as string) ?? message;
    errorCode = String(errorObj.code ?? 'UNKNOWN_ERROR');
  } else {
    errorCode = String(error);
  }

  // Normalize error code
  errorCode = errorCode.toUpperCase().replace(/-/g, '_').replace(/ /g, '_');

  // Check for translated HL errors (these have errorCode field)
  const hlErrorCode = data.errorCode as string | undefined;
  if (hlErrorCode) {
    errorCode = hlErrorCode;
  }

  // Map error codes to specific exceptions

  // Approval errors
  if (errorCode === 'NOT_APPROVED' || errorCode === 'BUILDER_APPROVAL_REQUIRED') {
    return new ApprovalError(message, {
      guidance: guidance ?? "Builder fee not approved. Run sdk.approveBuilderFee('1%') or use HyperliquidSDK({ autoApprove: true }).",
      approvalData: data.approvalRequired as Record<string, unknown>,
      code: errorCode,
      raw,
    });
  }

  if (errorCode === 'FEE_EXCEEDS_APPROVED') {
    return new ApprovalError(message, {
      guidance: guidance ?? "Your approved max fee is too low. Re-approve with a higher rate: sdk.approveBuilderFee('1%')",
      code: errorCode,
      raw,
    });
  }

  // Validation errors
  if (['INVALID_JSON', 'MISSING_FIELD', 'INVALID_PARAMS', 'INVALID_ORDER_PARAMS'].includes(errorCode)) {
    return new ValidationError(message, { code: errorCode, guidance, raw });
  }

  if (errorCode === 'INVALID_PRICE_TICK' || errorCode === 'INVALID_SIZE') {
    return new ValidationError(message, {
      code: errorCode,
      guidance: guidance ?? 'Use sdk.preflight() to validate orders before placing them.',
      raw,
    });
  }

  // Signature errors
  if (errorCode === 'SIGNATURE_INVALID') {
    return new SignatureError(message, {
      code: errorCode,
      guidance: guidance ?? 'Signature verification failed. This is usually an SDK bug - please report it.',
      raw,
    });
  }

  // Position errors
  if (errorCode === 'NO_POSITION') {
    const asset = (data.asset as string) ?? 'unknown';
    return new NoPositionError(asset);
  }

  // Geo-blocking
  if (statusCode === 403 || message.toLowerCase().includes('restricted') || message.toLowerCase().includes('jurisdiction')) {
    return new GeoBlockedError(data);
  }

  // Translated HL errors
  if (errorCode === 'INSUFFICIENT_MARGIN') {
    return new InsufficientMarginError(rawHlError);
  }

  if (errorCode === 'LEVERAGE_CONFLICT') {
    return new LeverageError(rawHlError);
  }

  if (errorCode === 'RATE_LIMITED') {
    return new RateLimitError(rawHlError);
  }

  if (errorCode === 'MAX_ORDERS_EXCEEDED') {
    return new MaxOrdersError(rawHlError);
  }

  if (errorCode === 'REDUCE_ONLY_VIOLATION') {
    return new ReduceOnlyError(rawHlError);
  }

  if (errorCode === 'DUPLICATE_ORDER') {
    return new DuplicateOrderError(rawHlError);
  }

  if (errorCode === 'USER_NOT_FOUND') {
    return new UserNotFoundError(rawHlError);
  }

  if (errorCode === 'MUST_DEPOSIT_FIRST') {
    return new MustDepositError(rawHlError);
  }

  if (errorCode === 'INVALID_NONCE') {
    return new InvalidNonceError(rawHlError);
  }

  // Generic categorization based on phase
  const phase = String(data.phase ?? '').toLowerCase();
  if (phase.includes('build')) {
    return new BuildError(message, { code: errorCode, guidance, raw });
  }

  // Default to SendError for exchange-related errors
  return new SendError(message, { code: errorCode, guidance, raw });
}
