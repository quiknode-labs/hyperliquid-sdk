/**
 * Hyperliquid SDK Type Definitions
 */

// ═══════════════════════════════════════════════════════════════════════════
// ENUMS
// ═══════════════════════════════════════════════════════════════════════════

export enum Side {
  BUY = 'buy',
  SELL = 'sell',
  LONG = 'buy',
  SHORT = 'sell',
}

export enum TIF {
  IOC = 'ioc',
  GTC = 'gtc',
  ALO = 'alo',
  MARKET = 'market',
}

export enum TpSl {
  TP = 'tp',
  SL = 'sl',
}

export enum OrderGrouping {
  NA = 'na',
  NORMAL_TPSL = 'normalTpsl',
  POSITION_TPSL = 'positionTpsl',
}

export enum ConnectionState {
  DISCONNECTED = 'disconnected',
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
  RECONNECTING = 'reconnecting',
}

// ═══════════════════════════════════════════════════════════════════════════
// SDK OPTIONS
// ═══════════════════════════════════════════════════════════════════════════

export interface SDKOptions {
  privateKey?: string;
  testnet?: boolean;
  autoApprove?: boolean;
  maxFee?: string;
  slippage?: number;
  timeout?: number;
}

export interface InfoOptions {
  timeout?: number;
}

export interface HyperCoreOptions {
  timeout?: number;
}

export interface EVMOptions {
  debug?: boolean;
  timeout?: number;
}

export interface StreamOptions {
  reconnect?: boolean;
  maxReconnectAttempts?: number;
  reconnectInterval?: number;
  pingInterval?: number;
  onError?: (error: Error) => void;
  onClose?: () => void;
  onOpen?: () => void;
  onStateChange?: (state: ConnectionState) => void;
}

export interface GRPCStreamOptions {
  reconnect?: boolean;
  maxReconnectAttempts?: number;
  reconnectInterval?: number;
  keepaliveInterval?: number;
  onError?: (error: Error) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onStateChange?: (state: ConnectionState) => void;
}

export interface EVMStreamOptions {
  reconnect?: boolean;
  maxReconnectAttempts?: number;
  onError?: (error: Error) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onStateChange?: (state: ConnectionState) => void;
}

// ═══════════════════════════════════════════════════════════════════════════
// ORDER TYPES
// ═══════════════════════════════════════════════════════════════════════════

export interface OrderOptions {
  size?: number | string;
  notional?: number;
  price?: number | string;
  tif?: TIF | string;
  reduceOnly?: boolean;
  cloid?: string;
  grouping?: OrderGrouping;
}

export interface TriggerOrderOptions {
  size: number | string;
  triggerPrice: number | string;
  limitPrice?: number | string;
  side?: Side;
  reduceOnly?: boolean;
  cloid?: string;
  grouping?: OrderGrouping;
}

export interface OrderAction {
  type: 'order';
  orders: Array<{
    a: number; // asset index
    b: boolean; // is buy
    p: string; // price
    s: string; // size
    r: boolean; // reduce only
    t: { limit: { tif: string } } | { trigger: { isMarket: boolean; triggerPx: string; tpsl: string } };
    c?: string; // cloid
  }>;
  grouping: string;
}

export interface TriggerOrderAction {
  type: 'order';
  orders: Array<{
    a: number;
    b: boolean;
    p: string;
    s: string;
    r: boolean;
    t: { trigger: { isMarket: boolean; triggerPx: string; tpsl: string } };
    c?: string;
  }>;
  grouping: string;
}

export interface PlacedOrderResponse {
  status: string;
  response?: {
    type: string;
    data?: {
      statuses: Array<{
        resting?: { oid: number };
        filled?: { oid: number; totalSz: string; avgPx: string };
        error?: string;
      }>;
    };
  };
}

// ═══════════════════════════════════════════════════════════════════════════
// INFO API TYPES
// ═══════════════════════════════════════════════════════════════════════════

export interface AssetMeta {
  name: string;
  szDecimals: number;
  maxLeverage: number;
  onlyIsolated?: boolean;
}

export interface MetaResult {
  universe: AssetMeta[];
}

export interface SpotMeta {
  tokens: Array<{
    name: string;
    szDecimals: number;
    weiDecimals: number;
    index: number;
    isCanonical: boolean;
    evmContract: string | null;
    fullName: string | null;
  }>;
  universe: Array<{
    tokens: [number, number];
    name: string;
    index: number;
    isCanonical: boolean;
  }>;
}

export interface ExchangeStatus {
  time?: number;
  specialStatuses?: Record<string, string> | null;
}

export interface L2BookLevel {
  px: string;
  sz: string;
  n: number;
}

export interface L2BookResult {
  coin: string;
  time: number;
  levels: [L2BookLevel[], L2BookLevel[]];
}

export interface Trade {
  coin: string;
  side: string;
  px: string;
  sz: string;
  hash: string;
  time: number;
  tid: number;
}

export interface Candle {
  t: number; // timestamp
  T: number; // close time
  s: string; // symbol
  i: string; // interval
  o: string; // open
  c: string; // close
  h: string; // high
  l: string; // low
  v: string; // volume
  n: number; // number of trades
}

export interface FundingRate {
  coin: string;
  fundingRate: string;
  premium: string;
  time: number;
}

export interface PredictedFunding {
  coin: string;
  fundingRate: string;
  premium: string;
}

export interface Position {
  coin: string;
  entryPx: string | null;
  leverage: { type: string; value: number; rawUsd?: string };
  liquidationPx: string | null;
  marginUsed: string;
  maxTradeSzs: [string, string];
  positionValue: string;
  returnOnEquity: string;
  szi: string;
  unrealizedPnl: string;
}

export interface ClearinghouseState {
  assetPositions: Array<{ position: Position; type: string }>;
  crossMaintenanceMarginUsed: string;
  crossMarginSummary: {
    accountValue: string;
    totalMarginUsed: string;
    totalNtlPos: string;
    totalRawUsd: string;
  };
  marginSummary: {
    accountValue: string;
    totalMarginUsed: string;
    totalNtlPos: string;
    totalRawUsd: string;
  };
  time: number;
  withdrawable: string;
}

export interface SpotBalance {
  coin: string;
  hold: string;
  total: string;
  entryNtl: string;
}

export interface SpotClearinghouseState {
  balances: SpotBalance[];
}

export interface OpenOrder {
  coin: string;
  limitPx: string;
  oid: number;
  side: string;
  sz: string;
  timestamp: number;
  cloid?: string;
  reduceOnly?: boolean;
  orderType?: string;
  origSz?: string;
  tif?: string;
}

export interface FrontendOpenOrder extends OpenOrder {
  children?: OpenOrder[];
  isPositionTpsl?: boolean;
  isTrigger?: boolean;
  triggerCondition?: string;
  triggerPx?: string;
}

export interface OrderStatus {
  order?: OpenOrder;
  status: string;
  statusTimestamp?: number;
}

export interface Fill {
  coin: string;
  px: string;
  sz: string;
  side: string;
  time: number;
  startPosition: string;
  dir: string;
  closedPnl: string;
  hash: string;
  oid: number;
  crossed: boolean;
  fee: string;
  tid: number;
  feeToken?: string;
  builderFee?: string;
}

export interface FundingPayment {
  coin: string;
  fundingRate: string;
  szi: string;
  time: number;
  usdc: string;
  hash: string;
  nSamples?: number;
}

export interface UserFees {
  activeReferralDiscount: string;
  dailyUserVlm: Array<{ date: string; exchange: string; ntl: string }>;
  feeSchedule: {
    add: string;
    remove: string;
    tiers: Record<string, unknown>;
  };
  userAddRate: string;
  userCrossRate: string;
}

export interface RateLimitInfo {
  cumVlm: string;
  nRequestsUsed: number;
  nRequestsCap: number;
}

export interface SubAccount {
  clearinghouseState: ClearinghouseState;
  master: string;
  name: string;
  subAccountUser: string;
}

export interface ExtraAgent {
  address: string;
  name: string;
  validUntil: number;
}

export interface VaultSummary {
  vaultAddress: string;
  name: string;
  leader: string;
  tvl: string;
  apr: string;
  followerState?: {
    vaultEquity: string;
    pnl: string;
  };
}

export interface VaultDetails {
  name: string;
  vaultAddress: string;
  leader: string;
  description: string;
  portfolio: Position[];
  apr: string;
  followerState?: {
    vaultEquity: string;
    pnl: string;
    lockupUntil: number;
    allTimePnl: string;
  };
}

export interface Delegation {
  validator: string;
  amount: string;
  lockedUntil: number;
}

export interface DelegatorSummary {
  delegated: string;
  undelegating: string;
  totalPendingRewards: string;
  nDelegations: number;
}

export interface DelegatorHistory {
  time: number;
  delta: string;
  type: string;
  validator: string;
}

export interface DelegatorRewards {
  totalPendingRewards: string;
  rewardsByValidator: Array<{
    validator: string;
    pendingRewards: string;
  }>;
}

export interface TokenDetails {
  name: string;
  szDecimals: number;
  weiDecimals: number;
  index: number;
  tokenId: string;
  isCanonical: boolean;
  evmContract: string | null;
  fullName: string | null;
  maxMintable?: string;
  totalSupply?: string;
  deployer?: string;
  deployTime?: number;
}

export interface LiquidatablePosition {
  user: string;
  coin: string;
  leverage: { type: string; value: number };
  marginUsed: string;
  ntlValue: string;
  rawUsd: string;
}

// ═══════════════════════════════════════════════════════════════════════════
// HYPERCORE TYPES
// ═══════════════════════════════════════════════════════════════════════════

export interface Block {
  blockNumber: number;
  time: string;
  signedActionBundles: Array<{
    actions: unknown[];
    signature: string;
  }>;
  resps: unknown[];
}

export interface LatestBlocksResult {
  blockNumber: number;
  blocks: Block[];
}

export interface HyperCoreTrade {
  coin: string;
  side: string;
  px: string;
  sz: string;
  time: number;
  hash: string;
  users: [string, string];
}

export interface HyperCoreOrder {
  coin: string;
  side: string;
  limitPx: string;
  sz: string;
  oid: number;
  timestamp: number;
  user: string;
  orderType: string;
}

export interface BookUpdate {
  coin: string;
  side: string;
  px: string;
  sz: string;
  time: number;
}

export interface Dex {
  name: string;
  index: number;
}

export interface Market {
  name: string;
  index: number;
  szDecimals: number;
}

export interface BuildResult {
  action: Record<string, unknown>;
  nonce: number;
  vaultAddress?: string;
}

export interface SendResult {
  status: string;
  response?: {
    type: string;
    data?: unknown;
  };
}

export interface PreflightResult {
  isValid: boolean;
  errors?: string[];
  warnings?: string[];
  marginRequired?: string;
  estimatedFee?: string;
}

// ═══════════════════════════════════════════════════════════════════════════
// EVM TYPES
// ═══════════════════════════════════════════════════════════════════════════

export interface EVMBlock {
  number: string;
  hash: string;
  parentHash: string;
  nonce: string;
  sha3Uncles: string;
  logsBloom: string;
  transactionsRoot: string;
  stateRoot: string;
  receiptsRoot: string;
  miner: string;
  difficulty: string;
  totalDifficulty: string;
  extraData: string;
  size: string;
  gasLimit: string;
  gasUsed: string;
  timestamp: string;
  transactions: string[] | EVMTransaction[];
  uncles: string[];
  baseFeePerGas?: string;
}

export interface EVMTransaction {
  hash: string;
  nonce: string;
  blockHash: string | null;
  blockNumber: string | null;
  transactionIndex: string | null;
  from: string;
  to: string | null;
  value: string;
  gasPrice: string;
  gas: string;
  input: string;
  v: string;
  r: string;
  s: string;
  type?: string;
  maxFeePerGas?: string;
  maxPriorityFeePerGas?: string;
  accessList?: Array<{ address: string; storageKeys: string[] }>;
}

export interface EVMTransactionReceipt {
  transactionHash: string;
  transactionIndex: string;
  blockHash: string;
  blockNumber: string;
  from: string;
  to: string | null;
  cumulativeGasUsed: string;
  gasUsed: string;
  contractAddress: string | null;
  logs: EVMLog[];
  logsBloom: string;
  status: string;
  effectiveGasPrice: string;
  type?: string;
}

export interface EVMLog {
  address: string;
  topics: string[];
  data: string;
  blockNumber: string;
  transactionHash: string;
  transactionIndex: string;
  blockHash: string;
  logIndex: string;
  removed: boolean;
}

export interface EVMCallParams {
  to: string;
  from?: string;
  gas?: string;
  gasPrice?: string;
  value?: string;
  data?: string;
}

export interface EVMFilterParams {
  fromBlock?: string | number;
  toBlock?: string | number;
  address?: string | string[];
  topics?: (string | string[] | null)[];
  blockHash?: string;
}

export interface TraceResult {
  gas: number;
  failed: boolean;
  returnValue: string;
  structLogs: Array<{
    pc: number;
    op: string;
    gas: number;
    gasCost: number;
    depth: number;
    stack: string[];
    memory?: string[];
    storage?: Record<string, string>;
  }>;
}

export interface Trace {
  type: string;
  action: {
    from: string;
    to: string;
    value: string;
    gas: string;
    input: string;
    callType?: string;
  };
  result?: {
    gasUsed: string;
    output: string;
  };
  subtraces: number;
  traceAddress: number[];
  blockNumber: number;
  blockHash: string;
  transactionHash: string;
  transactionPosition: number;
}

export interface FeeHistory {
  oldestBlock: string;
  baseFeePerGas: string[];
  gasUsedRatio: number[];
  reward?: string[][];
}

export interface SyncingStatus {
  startingBlock: string;
  currentBlock: string;
  highestBlock: string;
}

// ═══════════════════════════════════════════════════════════════════════════
// STREAM TYPES
// ═══════════════════════════════════════════════════════════════════════════

export enum StreamType {
  TRADES = 'trades',
  ORDERS = 'orders',
  BOOK_UPDATES = 'bookUpdates',
  TWAP = 'twap',
  EVENTS = 'events',
  WRITER_ACTIONS = 'writerActions',
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

export enum GRPCStreamType {
  TRADES = 1,
  ORDERS = 2,
  BOOK_UPDATES = 3,
  TWAP = 4,
  EVENTS = 5,
  BLOCKS = 6,
  WRITER_ACTIONS = 7,
}

export enum EVMSubscriptionType {
  NEW_HEADS = 'newHeads',
  LOGS = 'logs',
  NEW_PENDING_TRANSACTIONS = 'newPendingTransactions',
}

export interface StreamMessage {
  channel: string;
  data: unknown;
}

export interface TradeMessage {
  coin: string;
  side: string;
  px: string;
  sz: string;
  hash: string;
  time: number;
  tid: number;
}

export interface OrderMessage {
  coin: string;
  side: string;
  limitPx: string;
  sz: string;
  oid: number;
  timestamp: number;
  user: string;
  status: string;
}

export interface BookUpdateMessage {
  coin: string;
  side: string;
  px: string;
  sz: string;
  time: number;
}

export interface L2BookMessage {
  coin: string;
  time: number;
  levels: [Array<{ px: string; sz: string; n: number }>, Array<{ px: string; sz: string; n: number }>];
}

export interface L4BookSnapshot {
  coin: string;
  time: number;
  bids: L4Order[];
  asks: L4Order[];
}

export interface L4BookDiff {
  coin: string;
  time: number;
  changes: L4Change[];
}

export interface L4Order {
  user: string;
  coin: string;
  side: string;
  limitPx: string;
  sz: string;
  oid: number;
  timestamp: number;
  reduceOnly: boolean;
  orderType: string;
  tif?: string;
  cloid?: string;
}

export interface L4Change {
  type: 'add' | 'remove' | 'modify';
  order: L4Order;
}

export interface BlockMessage {
  abciBlock: {
    time: string;
    signedActionBundles: unknown[];
  };
  resps: unknown[];
}

// ═══════════════════════════════════════════════════════════════════════════
// TRADING RESULT TYPES
// ═══════════════════════════════════════════════════════════════════════════

export interface ApprovalStatus {
  approved: boolean;
  maxFeeRate?: string;
  builder?: string;
}

export interface MarketsResult {
  universe: AssetMeta[];
  spotUniverse?: SpotMeta['universe'];
}

export interface DexesResult {
  dexes: Dex[];
}

export interface TwapResult {
  twapId: number;
  status: string;
}

// ═══════════════════════════════════════════════════════════════════════════
// SIGNATURE TYPES
// ═══════════════════════════════════════════════════════════════════════════

export interface Signature {
  r: string;
  s: string;
  v: number;
}

export interface SignedAction {
  action: Record<string, unknown>;
  signature: Signature;
  nonce: number;
  vaultAddress?: string;
}
