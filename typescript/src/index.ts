/**
 * Hyperliquid SDK for TypeScript
 *
 * A complete, type-safe client for the Hyperliquid exchange.
 *
 * @packageDocumentation
 */

// Main SDK client
export { HyperliquidSDK, HyperliquidSDKOptions } from './client';

// Order builders
export { Order, TriggerOrder, PlacedOrder, Side, TIF, TpSl, OrderGrouping } from './order';

// Info API client
export { Info, InfoOptions } from './info';

// HyperCore JSON-RPC client
export { HyperCore, HyperCoreOptions } from './hypercore';

// EVM JSON-RPC client
export { EVM, EVMOptions } from './evm';

// WebSocket streaming
export { Stream, StreamType, ConnectionState as StreamConnectionState, StreamOptions } from './websocket';

// gRPC streaming
export { GRPCStream, GRPCStreamType, GRPCStreamOptions, ConnectionState as GRPCConnectionState } from './grpc-stream';

// EVM WebSocket streaming
export { EVMStream, EVMSubscriptionType, EVMStreamOptions, ConnectionState as EVMConnectionState } from './evm-stream';

// Errors
export {
  HyperliquidError,
  BuildError,
  SendError,
  ApprovalError,
  ValidationError,
  SignatureError,
  NoPositionError,
  OrderNotFoundError,
  GeoBlockedError,
  InsufficientMarginError,
  LeverageError,
  RateLimitError,
  MaxOrdersError,
  ReduceOnlyError,
  DuplicateOrderError,
  UserNotFoundError,
  MustDepositError,
  InvalidNonceError,
  ErrorOptions,
  parseApiError,
} from './errors';

// Re-export commonly used types
export type { default as WebSocket } from 'ws';
