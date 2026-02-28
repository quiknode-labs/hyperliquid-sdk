package hyperliquid

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
)

// EVMClient provides access to the HyperEVM JSON-RPC API.
type EVMClient struct {
	url      string
	http     *HTTPClient
}

// NewEVMClient creates a new EVM API client.
func NewEVMClient(endpoint string, http *HTTPClient) *EVMClient {
	return &EVMClient{
		url:      buildEVMURL(endpoint),
		http:     http,
	}
}

func (c *EVMClient) call(ctx context.Context, method string, params []any) (any, error) {
	body := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}

	result, err := c.http.PostRaw(ctx, c.url, body)
	if err != nil {
		return nil, err
	}

	if m, ok := result.(map[string]any); ok {
		if errVal, ok := m["error"]; ok && errVal != nil {
			if errMap, ok := errVal.(map[string]any); ok {
				msg := ""
				if m, ok := errMap["message"].(string); ok {
					msg = m
				}
				return nil, NewError(ErrorCodeHTTPError, msg).WithRaw(errMap)
			}
		}
		if res, ok := m["result"]; ok {
			return res, nil
		}
	}

	return result, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// BASIC QUERIES
// ═══════════════════════════════════════════════════════════════════════════════

// BlockNumber returns the latest block number.
func (c *EVMClient) BlockNumber() (int64, error) {
	result, err := c.call(context.Background(), "eth_blockNumber", []any{})
	if err != nil {
		return 0, err
	}
	return parseHexInt64(result)
}

// ChainID returns the chain ID.
func (c *EVMClient) ChainID() (int64, error) {
	result, err := c.call(context.Background(), "eth_chainId", []any{})
	if err != nil {
		return 0, err
	}
	return parseHexInt64(result)
}

// GasPrice returns the current gas price.
func (c *EVMClient) GasPrice() (int64, error) {
	result, err := c.call(context.Background(), "eth_gasPrice", []any{})
	if err != nil {
		return 0, err
	}
	return parseHexInt64(result)
}

// ═══════════════════════════════════════════════════════════════════════════════
// ACCOUNT QUERIES
// ═══════════════════════════════════════════════════════════════════════════════

// GetBalance returns the balance of an address in wei.
func (c *EVMClient) GetBalance(address string, blockNumber string) (string, error) {
	if blockNumber == "" {
		blockNumber = "latest"
	}
	result, err := c.call(context.Background(), "eth_getBalance", []any{address, blockNumber})
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", nil
}

// GetNonce returns the transaction count (nonce) for an address.
func (c *EVMClient) GetNonce(address string, blockNumber string) (int64, error) {
	if blockNumber == "" {
		blockNumber = "latest"
	}
	result, err := c.call(context.Background(), "eth_getTransactionCount", []any{address, blockNumber})
	if err != nil {
		return 0, err
	}
	return parseHexInt64(result)
}

// GetCode returns the contract code at an address.
func (c *EVMClient) GetCode(address string, blockNumber string) (string, error) {
	if blockNumber == "" {
		blockNumber = "latest"
	}
	result, err := c.call(context.Background(), "eth_getCode", []any{address, blockNumber})
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", nil
}

// GetStorageAt returns the value from a storage position at an address.
func (c *EVMClient) GetStorageAt(address, position, blockNumber string) (string, error) {
	if blockNumber == "" {
		blockNumber = "latest"
	}
	result, err := c.call(context.Background(), "eth_getStorageAt", []any{address, position, blockNumber})
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// CONTRACT CALLS
// ═══════════════════════════════════════════════════════════════════════════════

// Call executes a new message call without creating a transaction.
func (c *EVMClient) Call(to, data string, blockNumber string) (string, error) {
	if blockNumber == "" {
		blockNumber = "latest"
	}
	callObj := map[string]any{
		"to":   to,
		"data": data,
	}
	result, err := c.call(context.Background(), "eth_call", []any{callObj, blockNumber})
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", nil
}

// EstimateGas estimates the gas needed for a transaction.
func (c *EVMClient) EstimateGas(to, data string) (int64, error) {
	callObj := map[string]any{
		"to":   to,
		"data": data,
	}
	result, err := c.call(context.Background(), "eth_estimateGas", []any{callObj})
	if err != nil {
		return 0, err
	}
	return parseHexInt64(result)
}

// ═══════════════════════════════════════════════════════════════════════════════
// TRANSACTION QUERIES
// ═══════════════════════════════════════════════════════════════════════════════

// SendRawTransaction sends a signed transaction.
func (c *EVMClient) SendRawTransaction(signedTx string) (string, error) {
	result, err := c.call(context.Background(), "eth_sendRawTransaction", []any{signedTx})
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", nil
}

// GetTransactionByHash returns transaction info by hash.
func (c *EVMClient) GetTransactionByHash(hash string) (map[string]any, error) {
	result, err := c.call(context.Background(), "eth_getTransactionByHash", []any{hash})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// GetTransactionReceipt returns transaction receipt by hash.
func (c *EVMClient) GetTransactionReceipt(hash string) (map[string]any, error) {
	result, err := c.call(context.Background(), "eth_getTransactionReceipt", []any{hash})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// BLOCK QUERIES
// ═══════════════════════════════════════════════════════════════════════════════

// GetBlockByNumber returns block info by number.
func (c *EVMClient) GetBlockByNumber(blockNumber string, fullTx bool) (map[string]any, error) {
	result, err := c.call(context.Background(), "eth_getBlockByNumber", []any{blockNumber, fullTx})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// GetBlockByHash returns block info by hash.
func (c *EVMClient) GetBlockByHash(hash string, fullTx bool) (map[string]any, error) {
	result, err := c.call(context.Background(), "eth_getBlockByHash", []any{hash, fullTx})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// LOGS
// ═══════════════════════════════════════════════════════════════════════════════

// GetLogs returns logs matching a filter.
func (c *EVMClient) GetLogs(filter map[string]any) ([]any, error) {
	result, err := c.call(context.Background(), "eth_getLogs", []any{filter})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// ADDITIONAL STANDARD METHODS
// ═══════════════════════════════════════════════════════════════════════════════

// NetVersion returns the network version.
func (c *EVMClient) NetVersion() (string, error) {
	result, err := c.call(context.Background(), "net_version", []any{})
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", nil
}

// Web3ClientVersion returns the client version.
func (c *EVMClient) Web3ClientVersion() (string, error) {
	result, err := c.call(context.Background(), "web3_clientVersion", []any{})
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", nil
}

// Syncing returns sync status. Returns false if not syncing, or sync info if syncing.
func (c *EVMClient) Syncing() (any, error) {
	return c.call(context.Background(), "eth_syncing", []any{})
}

// Accounts returns list of accounts (usually empty for remote nodes).
func (c *EVMClient) Accounts() ([]string, error) {
	result, err := c.call(context.Background(), "eth_accounts", []any{})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		accounts := make([]string, len(arr))
		for i, v := range arr {
			if s, ok := v.(string); ok {
				accounts[i] = s
			}
		}
		return accounts, nil
	}
	return nil, nil
}

// FeeHistory returns fee history for a range of blocks.
func (c *EVMClient) FeeHistory(blockCount int, newestBlock string, rewardPercentiles []float64) (map[string]any, error) {
	params := []any{fmt.Sprintf("0x%x", blockCount), newestBlock}
	if len(rewardPercentiles) > 0 {
		params = append(params, rewardPercentiles)
	}
	result, err := c.call(context.Background(), "eth_feeHistory", params)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// MaxPriorityFeePerGas returns max priority fee per gas.
func (c *EVMClient) MaxPriorityFeePerGas() (int64, error) {
	result, err := c.call(context.Background(), "eth_maxPriorityFeePerGas", []any{})
	if err != nil {
		return 0, err
	}
	return parseHexInt64(result)
}

// GetBlockReceipts returns all receipts for a block.
func (c *EVMClient) GetBlockReceipts(blockNumber string) ([]any, error) {
	result, err := c.call(context.Background(), "eth_getBlockReceipts", []any{blockNumber})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// GetBlockTransactionCountByHash returns transaction count in a block by hash.
func (c *EVMClient) GetBlockTransactionCountByHash(blockHash string) (int64, error) {
	result, err := c.call(context.Background(), "eth_getBlockTransactionCountByHash", []any{blockHash})
	if err != nil {
		return 0, err
	}
	return parseHexInt64(result)
}

// GetBlockTransactionCountByNumber returns transaction count in a block by number.
func (c *EVMClient) GetBlockTransactionCountByNumber(blockNumber string) (int64, error) {
	result, err := c.call(context.Background(), "eth_getBlockTransactionCountByNumber", []any{blockNumber})
	if err != nil {
		return 0, err
	}
	return parseHexInt64(result)
}

// GetTransactionByBlockHashAndIndex returns transaction by block hash and index.
func (c *EVMClient) GetTransactionByBlockHashAndIndex(blockHash string, index int) (map[string]any, error) {
	result, err := c.call(context.Background(), "eth_getTransactionByBlockHashAndIndex", []any{blockHash, fmt.Sprintf("0x%x", index)})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// GetTransactionByBlockNumberAndIndex returns transaction by block number and index.
func (c *EVMClient) GetTransactionByBlockNumberAndIndex(blockNumber string, index int) (map[string]any, error) {
	result, err := c.call(context.Background(), "eth_getTransactionByBlockNumberAndIndex", []any{blockNumber, fmt.Sprintf("0x%x", index)})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// HYPERLIQUID-SPECIFIC EVM METHODS
// ═══════════════════════════════════════════════════════════════════════════════

// BigBlockGasPrice returns gas price for big blocks.
func (c *EVMClient) BigBlockGasPrice() (int64, error) {
	result, err := c.call(context.Background(), "eth_bigBlockGasPrice", []any{})
	if err != nil {
		return 0, err
	}
	return parseHexInt64(result)
}

// UsingBigBlocks returns whether the node is using big blocks.
func (c *EVMClient) UsingBigBlocks() (bool, error) {
	result, err := c.call(context.Background(), "eth_usingBigBlocks", []any{})
	if err != nil {
		return false, err
	}
	if b, ok := result.(bool); ok {
		return b, nil
	}
	return false, nil
}

// GetSystemTxsByBlockHash returns system transactions by block hash.
func (c *EVMClient) GetSystemTxsByBlockHash(blockHash string) ([]any, error) {
	result, err := c.call(context.Background(), "eth_getSystemTxsByBlockHash", []any{blockHash})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// GetSystemTxsByBlockNumber returns system transactions by block number.
func (c *EVMClient) GetSystemTxsByBlockNumber(blockNumber string) ([]any, error) {
	result, err := c.call(context.Background(), "eth_getSystemTxsByBlockNumber", []any{blockNumber})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// DEBUG/TRACE METHODS (requires nanoreth endpoint)
// ═══════════════════════════════════════════════════════════════════════════════

// DebugTraceTransaction traces a transaction's execution.
func (c *EVMClient) DebugTraceTransaction(txHash string, tracerConfig map[string]any) (map[string]any, error) {
	params := []any{txHash}
	if tracerConfig != nil {
		params = append(params, tracerConfig)
	}
	result, err := c.call(context.Background(), "debug_traceTransaction", params)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// TraceTransaction returns trace of a transaction.
func (c *EVMClient) TraceTransaction(txHash string) ([]any, error) {
	result, err := c.call(context.Background(), "trace_transaction", []any{txHash})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// TraceBlock returns traces of all transactions in a block.
func (c *EVMClient) TraceBlock(blockNumber string) ([]any, error) {
	result, err := c.call(context.Background(), "trace_block", []any{blockNumber})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// DebugGetBadBlocks returns bad blocks.
func (c *EVMClient) DebugGetBadBlocks() ([]any, error) {
	result, err := c.call(context.Background(), "debug_getBadBlocks", []any{})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// DebugGetRawBlock returns raw block data.
func (c *EVMClient) DebugGetRawBlock(blockNumber string) (string, error) {
	result, err := c.call(context.Background(), "debug_getRawBlock", []any{blockNumber})
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", nil
}

// DebugGetRawHeader returns raw block header.
func (c *EVMClient) DebugGetRawHeader(blockNumber string) (string, error) {
	result, err := c.call(context.Background(), "debug_getRawHeader", []any{blockNumber})
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", nil
}

// DebugGetRawReceipts returns raw receipts for a block.
func (c *EVMClient) DebugGetRawReceipts(blockNumber string) ([]string, error) {
	result, err := c.call(context.Background(), "debug_getRawReceipts", []any{blockNumber})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		receipts := make([]string, len(arr))
		for i, v := range arr {
			if s, ok := v.(string); ok {
				receipts[i] = s
			}
		}
		return receipts, nil
	}
	return nil, nil
}

// DebugGetRawTransaction returns raw transaction data.
func (c *EVMClient) DebugGetRawTransaction(txHash string) (string, error) {
	result, err := c.call(context.Background(), "debug_getRawTransaction", []any{txHash})
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", nil
}

// DebugStorageRangeAt returns storage range at a specific point.
func (c *EVMClient) DebugStorageRangeAt(blockHash string, txIndex int, contractAddress, keyStart string, maxResult int) (map[string]any, error) {
	result, err := c.call(context.Background(), "debug_storageRangeAt", []any{blockHash, txIndex, contractAddress, keyStart, maxResult})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// DebugTraceBlock traces a block by RLP.
func (c *EVMClient) DebugTraceBlock(blockRLP string, tracerConfig map[string]any) ([]any, error) {
	params := []any{blockRLP}
	if tracerConfig != nil {
		params = append(params, tracerConfig)
	}
	result, err := c.call(context.Background(), "debug_traceBlock", params)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// DebugTraceBlockByHash traces a block by hash.
func (c *EVMClient) DebugTraceBlockByHash(blockHash string, tracerConfig map[string]any) ([]any, error) {
	params := []any{blockHash}
	if tracerConfig != nil {
		params = append(params, tracerConfig)
	}
	result, err := c.call(context.Background(), "debug_traceBlockByHash", params)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// DebugTraceBlockByNumber traces a block by number.
func (c *EVMClient) DebugTraceBlockByNumber(blockNumber string, tracerConfig map[string]any) ([]any, error) {
	params := []any{blockNumber}
	if tracerConfig != nil {
		params = append(params, tracerConfig)
	}
	result, err := c.call(context.Background(), "debug_traceBlockByNumber", params)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// DebugTraceCall traces a call.
func (c *EVMClient) DebugTraceCall(tx map[string]any, blockNumber string, tracerConfig map[string]any) (map[string]any, error) {
	if blockNumber == "" {
		blockNumber = "latest"
	}
	params := []any{tx, blockNumber}
	if tracerConfig != nil {
		params = append(params, tracerConfig)
	}
	result, err := c.call(context.Background(), "debug_traceCall", params)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// TraceCall traces a call with specified trace types.
func (c *EVMClient) TraceCall(tx map[string]any, traceTypes []string, blockNumber string) (map[string]any, error) {
	if blockNumber == "" {
		blockNumber = "latest"
	}
	result, err := c.call(context.Background(), "trace_call", []any{tx, traceTypes, blockNumber})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// TraceCallMany traces multiple calls.
func (c *EVMClient) TraceCallMany(calls []any, blockNumber string) ([]any, error) {
	if blockNumber == "" {
		blockNumber = "latest"
	}
	result, err := c.call(context.Background(), "trace_callMany", []any{calls, blockNumber})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// TraceFilter filters traces.
func (c *EVMClient) TraceFilter(filterParams map[string]any) ([]any, error) {
	result, err := c.call(context.Background(), "trace_filter", []any{filterParams})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// TraceRawTransaction traces a raw transaction.
func (c *EVMClient) TraceRawTransaction(rawTx string, traceTypes []string) (map[string]any, error) {
	result, err := c.call(context.Background(), "trace_rawTransaction", []any{rawTx, traceTypes})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// TraceReplayBlockTransactions replays and traces all transactions in a block.
func (c *EVMClient) TraceReplayBlockTransactions(blockNumber string, traceTypes []string) ([]any, error) {
	result, err := c.call(context.Background(), "trace_replayBlockTransactions", []any{blockNumber, traceTypes})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// TraceReplayTransaction replays and traces a transaction.
func (c *EVMClient) TraceReplayTransaction(txHash string, traceTypes []string) (map[string]any, error) {
	result, err := c.call(context.Background(), "trace_replayTransaction", []any{txHash, traceTypes})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════════════════════════════════════════

func parseHexInt64(v any) (int64, error) {
	s, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("expected hex string, got %T", v)
	}
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return 0, nil
	}

	b, err := hex.DecodeString(padHex(s))
	if err != nil {
		return 0, err
	}

	var result int64
	for _, by := range b {
		result = result<<8 | int64(by)
	}
	return result, nil
}

func padHex(s string) string {
	if len(s)%2 != 0 {
		s = "0" + s
	}
	return s
}
