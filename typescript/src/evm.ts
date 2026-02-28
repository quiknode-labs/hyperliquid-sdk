/**
 * HyperEVM Client — Ethereum JSON-RPC for Hyperliquid's EVM.
 *
 * Standard Ethereum JSON-RPC methods plus debug/trace capabilities.
 *
 * Example:
 *     import { EVM } from 'hyperliquid-sdk';
 *     const evm = new EVM("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN");
 *     console.log(await evm.blockNumber());
 *     console.log(await evm.getBalance("0x..."));
 */

import { HyperliquidError } from './errors';

export interface EVMOptions {
  /** Use nanoreth path for debug/trace APIs (mainnet only) */
  debug?: boolean;
  /** Request timeout in milliseconds (default: 30000) */
  timeout?: number;
}

/**
 * HyperEVM Client — Ethereum JSON-RPC for Hyperliquid's EVM.
 *
 * Standard Ethereum methods plus debug/trace APIs (nanoreth path).
 *
 * Examples:
 *     const evm = new EVM("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN");
 *
 *     // Standard methods
 *     await evm.blockNumber();
 *     await evm.chainId();
 *     await evm.getBalance("0x...");
 *     await evm.getBlockByNumber(12345);
 *
 *     // Debug/Trace (mainnet only)
 *     const evmDebug = new EVM(endpoint, { debug: true });
 *     await evmDebug.traceTransaction("0x...");
 */
export class EVM {
  private readonly _baseUrl: string;
  private readonly _timeout: number;
  private _requestId: number = 0;

  constructor(endpoint: string, options: EVMOptions = {}) {
    this._baseUrl = this._buildUrl(endpoint, options.debug ?? false);
    this._timeout = options.timeout ?? 30000;
  }

  private _buildUrl(url: string, useNanoreth: boolean): string {
    const parsed = new URL(url);
    const base = `${parsed.protocol}//${parsed.host}`;
    const pathParts = parsed.pathname.split('/').filter((p) => p.length > 0);

    const knownPaths = new Set(['info', 'hypercore', 'evm', 'nanoreth']);
    let token: string | null = null;
    for (const part of pathParts) {
      if (!knownPaths.has(part)) {
        token = part;
        break;
      }
    }

    const path = useNanoreth ? 'nanoreth' : 'evm';
    if (token) {
      return `${base}/${token}/${path}`;
    }
    return `${base}/${path}`;
  }

  /** Make a JSON-RPC call. */
  private async _rpc<T = unknown>(
    method: string,
    params: unknown[] = []
  ): Promise<T> {
    this._requestId += 1;
    const payload = {
      jsonrpc: '2.0',
      method,
      params,
      id: this._requestId,
    };

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this._timeout);

    try {
      const response = await fetch(this._baseUrl, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        throw new HyperliquidError(`Request failed: ${response.status}`, {
          code: 'HTTP_ERROR',
          raw: { body: await response.text() },
        });
      }

      const data = await response.json() as Record<string, unknown>;

      if (data.error) {
        const errorObj = data.error as Record<string, unknown>;
        throw new HyperliquidError(
          (errorObj.message as string) ?? 'RPC error',
          {
            code: String(errorObj.code ?? 'RPC_ERROR'),
            raw: data,
          }
        );
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
  // STANDARD ETHEREUM METHODS
  // ═══════════════════════════════════════════════════════════════════════════

  /** Get the current block number. */
  async blockNumber(): Promise<number> {
    const result = await this._rpc<string>('eth_blockNumber');
    return parseInt(result, 16);
  }

  /** Get the chain ID (999 mainnet, 998 testnet). */
  async chainId(): Promise<number> {
    const result = await this._rpc<string>('eth_chainId');
    return parseInt(result, 16);
  }

  /** Get the current gas price in wei. */
  async gasPrice(): Promise<bigint> {
    const result = await this._rpc<string>('eth_gasPrice');
    return BigInt(result);
  }

  /** Get account balance in wei. */
  async getBalance(address: string, block: number | string = 'latest'): Promise<bigint> {
    const blockParam = typeof block === 'number' ? `0x${block.toString(16)}` : block;
    const result = await this._rpc<string>('eth_getBalance', [address, blockParam]);
    return BigInt(result);
  }

  /** Get the nonce (transaction count) for an address. */
  async getTransactionCount(
    address: string,
    block: number | string = 'latest'
  ): Promise<number> {
    const blockParam = typeof block === 'number' ? `0x${block.toString(16)}` : block;
    const result = await this._rpc<string>('eth_getTransactionCount', [address, blockParam]);
    return parseInt(result, 16);
  }

  /** Get the contract bytecode at an address. */
  async getCode(address: string, block: number | string = 'latest'): Promise<string> {
    const blockParam = typeof block === 'number' ? `0x${block.toString(16)}` : block;
    return this._rpc('eth_getCode', [address, blockParam]);
  }

  /** Get storage at a specific position. */
  async getStorageAt(
    address: string,
    position: string,
    block: number | string = 'latest'
  ): Promise<string> {
    const blockParam = typeof block === 'number' ? `0x${block.toString(16)}` : block;
    return this._rpc('eth_getStorageAt', [address, position, blockParam]);
  }

  /** Execute a read-only call. */
  async call(
    tx: Record<string, unknown>,
    block: number | string = 'latest'
  ): Promise<string> {
    const blockParam = typeof block === 'number' ? `0x${block.toString(16)}` : block;
    return this._rpc('eth_call', [tx, blockParam]);
  }

  /** Estimate gas for a transaction. */
  async estimateGas(tx: Record<string, unknown>): Promise<bigint> {
    const result = await this._rpc<string>('eth_estimateGas', [tx]);
    return BigInt(result);
  }

  /** Submit a signed transaction. */
  async sendRawTransaction(signedTx: string): Promise<string> {
    return this._rpc('eth_sendRawTransaction', [signedTx]);
  }

  /** Get transaction by hash. */
  async getTransactionByHash(
    txHash: string
  ): Promise<Record<string, unknown> | null> {
    return this._rpc('eth_getTransactionByHash', [txHash]);
  }

  /** Get transaction receipt. */
  async getTransactionReceipt(
    txHash: string
  ): Promise<Record<string, unknown> | null> {
    return this._rpc('eth_getTransactionReceipt', [txHash]);
  }

  /** Get block by number. */
  async getBlockByNumber(
    blockNumber: number | string,
    fullTransactions: boolean = false
  ): Promise<Record<string, unknown> | null> {
    const blockParam =
      typeof blockNumber === 'number' ? `0x${blockNumber.toString(16)}` : blockNumber;
    return this._rpc('eth_getBlockByNumber', [blockParam, fullTransactions]);
  }

  /** Get block by hash. */
  async getBlockByHash(
    blockHash: string,
    fullTransactions: boolean = false
  ): Promise<Record<string, unknown> | null> {
    return this._rpc('eth_getBlockByHash', [blockHash, fullTransactions]);
  }

  /** Get logs matching filter (max 4 topics, 50 block range). */
  async getLogs(
    filterParams: Record<string, unknown>
  ): Promise<Array<Record<string, unknown>>> {
    return this._rpc('eth_getLogs', [filterParams]);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // DEBUG/TRACE METHODS (requires debug=true, mainnet only)
  // ═══════════════════════════════════════════════════════════════════════════

  /** Trace a transaction's execution. */
  async debugTraceTransaction(
    txHash: string,
    tracerConfig?: Record<string, unknown>
  ): Promise<Record<string, unknown>> {
    const params: unknown[] = [txHash];
    if (tracerConfig) {
      params.push(tracerConfig);
    }
    return this._rpc('debug_traceTransaction', params);
  }

  /** Get trace of a transaction. */
  async traceTransaction(
    txHash: string
  ): Promise<Array<Record<string, unknown>>> {
    return this._rpc('trace_transaction', [txHash]);
  }

  /** Get traces of all transactions in a block. */
  async traceBlock(
    blockNumber: number | string
  ): Promise<Array<Record<string, unknown>>> {
    const blockParam =
      typeof blockNumber === 'number' ? `0x${blockNumber.toString(16)}` : blockNumber;
    return this._rpc('trace_block', [blockParam]);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // ADDITIONAL STANDARD METHODS
  // ═══════════════════════════════════════════════════════════════════════════

  /** Get the network version. */
  async netVersion(): Promise<string> {
    return this._rpc('net_version');
  }

  /** Get the client version. */
  async web3ClientVersion(): Promise<string> {
    return this._rpc('web3_clientVersion');
  }

  /** Check if the node is syncing. */
  async syncing(): Promise<boolean | Record<string, unknown>> {
    return this._rpc('eth_syncing');
  }

  /** Get list of accounts (usually empty for remote nodes). */
  async accounts(): Promise<string[]> {
    return this._rpc('eth_accounts');
  }

  /** Get fee history for a range of blocks. */
  async feeHistory(
    blockCount: number,
    newestBlock: number | string,
    rewardPercentiles?: number[]
  ): Promise<Record<string, unknown>> {
    const blockParam =
      typeof newestBlock === 'number' ? `0x${newestBlock.toString(16)}` : newestBlock;
    const params: unknown[] = [`0x${blockCount.toString(16)}`, blockParam];
    if (rewardPercentiles) {
      params.push(rewardPercentiles);
    }
    return this._rpc('eth_feeHistory', params);
  }

  /** Get max priority fee per gas. */
  async maxPriorityFeePerGas(): Promise<bigint> {
    const result = await this._rpc<string>('eth_maxPriorityFeePerGas');
    return BigInt(result);
  }

  /** Get all receipts for a block. */
  async getBlockReceipts(
    blockNumber: number | string
  ): Promise<Array<Record<string, unknown>>> {
    const blockParam =
      typeof blockNumber === 'number' ? `0x${blockNumber.toString(16)}` : blockNumber;
    return this._rpc('eth_getBlockReceipts', [blockParam]);
  }

  /** Get transaction count in a block by hash. */
  async getBlockTransactionCountByHash(blockHash: string): Promise<number> {
    const result = await this._rpc<string>('eth_getBlockTransactionCountByHash', [
      blockHash,
    ]);
    return parseInt(result, 16);
  }

  /** Get transaction count in a block by number. */
  async getBlockTransactionCountByNumber(
    blockNumber: number | string
  ): Promise<number> {
    const blockParam =
      typeof blockNumber === 'number' ? `0x${blockNumber.toString(16)}` : blockNumber;
    const result = await this._rpc<string>('eth_getBlockTransactionCountByNumber', [
      blockParam,
    ]);
    return parseInt(result, 16);
  }

  /** Get transaction by block hash and index. */
  async getTransactionByBlockHashAndIndex(
    blockHash: string,
    index: number
  ): Promise<Record<string, unknown> | null> {
    return this._rpc('eth_getTransactionByBlockHashAndIndex', [
      blockHash,
      `0x${index.toString(16)}`,
    ]);
  }

  /** Get transaction by block number and index. */
  async getTransactionByBlockNumberAndIndex(
    blockNumber: number | string,
    index: number
  ): Promise<Record<string, unknown> | null> {
    const blockParam =
      typeof blockNumber === 'number' ? `0x${blockNumber.toString(16)}` : blockNumber;
    return this._rpc('eth_getTransactionByBlockNumberAndIndex', [
      blockParam,
      `0x${index.toString(16)}`,
    ]);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // HYPERLIQUID-SPECIFIC EVM METHODS
  // ═══════════════════════════════════════════════════════════════════════════

  /** Get gas price for big blocks. */
  async bigBlockGasPrice(): Promise<bigint> {
    const result = await this._rpc<string>('eth_bigBlockGasPrice');
    return BigInt(result);
  }

  /** Check if using big blocks. */
  async usingBigBlocks(): Promise<boolean> {
    return this._rpc('eth_usingBigBlocks');
  }

  /** Get system transactions by block hash. */
  async getSystemTxsByBlockHash(
    blockHash: string
  ): Promise<Array<Record<string, unknown>>> {
    return this._rpc('eth_getSystemTxsByBlockHash', [blockHash]);
  }

  /** Get system transactions by block number. */
  async getSystemTxsByBlockNumber(
    blockNumber: number | string
  ): Promise<Array<Record<string, unknown>>> {
    const blockParam =
      typeof blockNumber === 'number' ? `0x${blockNumber.toString(16)}` : blockNumber;
    return this._rpc('eth_getSystemTxsByBlockNumber', [blockParam]);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // ADDITIONAL DEBUG METHODS
  // ═══════════════════════════════════════════════════════════════════════════

  /** Get bad blocks. */
  async debugGetBadBlocks(): Promise<Array<Record<string, unknown>>> {
    return this._rpc('debug_getBadBlocks');
  }

  /** Get raw block data. */
  async debugGetRawBlock(blockNumber: number | string): Promise<string> {
    const blockParam =
      typeof blockNumber === 'number' ? `0x${blockNumber.toString(16)}` : blockNumber;
    return this._rpc('debug_getRawBlock', [blockParam]);
  }

  /** Get raw block header. */
  async debugGetRawHeader(blockNumber: number | string): Promise<string> {
    const blockParam =
      typeof blockNumber === 'number' ? `0x${blockNumber.toString(16)}` : blockNumber;
    return this._rpc('debug_getRawHeader', [blockParam]);
  }

  /** Get raw receipts for a block. */
  async debugGetRawReceipts(blockNumber: number | string): Promise<string[]> {
    const blockParam =
      typeof blockNumber === 'number' ? `0x${blockNumber.toString(16)}` : blockNumber;
    return this._rpc('debug_getRawReceipts', [blockParam]);
  }

  /** Get raw transaction data. */
  async debugGetRawTransaction(txHash: string): Promise<string> {
    return this._rpc('debug_getRawTransaction', [txHash]);
  }

  /** Get storage range at a specific point. */
  async debugStorageRangeAt(
    blockHash: string,
    txIndex: number,
    contractAddress: string,
    keyStart: string,
    maxResult: number
  ): Promise<Record<string, unknown>> {
    return this._rpc('debug_storageRangeAt', [
      blockHash,
      txIndex,
      contractAddress,
      keyStart,
      maxResult,
    ]);
  }

  /** Trace a block by RLP. */
  async debugTraceBlock(
    blockRlp: string,
    tracerConfig?: Record<string, unknown>
  ): Promise<Array<Record<string, unknown>>> {
    const params: unknown[] = [blockRlp];
    if (tracerConfig) {
      params.push(tracerConfig);
    }
    return this._rpc('debug_traceBlock', params);
  }

  /** Trace a block by hash. */
  async debugTraceBlockByHash(
    blockHash: string,
    tracerConfig?: Record<string, unknown>
  ): Promise<Array<Record<string, unknown>>> {
    const params: unknown[] = [blockHash];
    if (tracerConfig) {
      params.push(tracerConfig);
    }
    return this._rpc('debug_traceBlockByHash', params);
  }

  /** Trace a block by number. */
  async debugTraceBlockByNumber(
    blockNumber: number | string,
    tracerConfig?: Record<string, unknown>
  ): Promise<Array<Record<string, unknown>>> {
    const blockParam =
      typeof blockNumber === 'number' ? `0x${blockNumber.toString(16)}` : blockNumber;
    const params: unknown[] = [blockParam];
    if (tracerConfig) {
      params.push(tracerConfig);
    }
    return this._rpc('debug_traceBlockByNumber', params);
  }

  /** Trace a call. */
  async debugTraceCall(
    tx: Record<string, unknown>,
    block: number | string = 'latest',
    tracerConfig?: Record<string, unknown>
  ): Promise<Record<string, unknown>> {
    const blockParam = typeof block === 'number' ? `0x${block.toString(16)}` : block;
    const params: unknown[] = [tx, blockParam];
    if (tracerConfig) {
      params.push(tracerConfig);
    }
    return this._rpc('debug_traceCall', params);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // ADDITIONAL TRACE METHODS
  // ═══════════════════════════════════════════════════════════════════════════

  /** Trace a call with specified trace types. */
  async traceCall(
    tx: Record<string, unknown>,
    traceTypes: string[],
    block: number | string = 'latest'
  ): Promise<Record<string, unknown>> {
    const blockParam = typeof block === 'number' ? `0x${block.toString(16)}` : block;
    return this._rpc('trace_call', [tx, traceTypes, blockParam]);
  }

  /** Trace multiple calls. */
  async traceCallMany(
    calls: Array<[Record<string, unknown>, string[]]>,
    block: number | string = 'latest'
  ): Promise<Array<Record<string, unknown>>> {
    const blockParam = typeof block === 'number' ? `0x${block.toString(16)}` : block;
    return this._rpc('trace_callMany', [calls, blockParam]);
  }

  /** Filter traces. */
  async traceFilter(
    filterParams: Record<string, unknown>
  ): Promise<Array<Record<string, unknown>>> {
    return this._rpc('trace_filter', [filterParams]);
  }

  /** Trace a raw transaction. */
  async traceRawTransaction(
    rawTx: string,
    traceTypes: string[]
  ): Promise<Record<string, unknown>> {
    return this._rpc('trace_rawTransaction', [rawTx, traceTypes]);
  }

  /** Replay and trace all transactions in a block. */
  async traceReplayBlockTransactions(
    blockNumber: number | string,
    traceTypes: string[]
  ): Promise<Array<Record<string, unknown>>> {
    const blockParam =
      typeof blockNumber === 'number' ? `0x${blockNumber.toString(16)}` : blockNumber;
    return this._rpc('trace_replayBlockTransactions', [blockParam, traceTypes]);
  }

  /** Replay and trace a transaction. */
  async traceReplayTransaction(
    txHash: string,
    traceTypes: string[]
  ): Promise<Record<string, unknown>> {
    return this._rpc('trace_replayTransaction', [txHash, traceTypes]);
  }
}
