package hyperliquid

import (
	"context"
)

// HyperCoreClient provides access to the HyperCore JSON-RPC API.
type HyperCoreClient struct {
	url  string
	http *HTTPClient
}

// NewHyperCoreClient creates a new HyperCore API client.
func NewHyperCoreClient(endpoint string, http *HTTPClient) *HyperCoreClient {
	return &HyperCoreClient{
		url:  buildHyperCoreURL(endpoint),
		http: http,
	}
}

func (c *HyperCoreClient) call(ctx context.Context, method string, params any) (any, error) {
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
// BLOCK QUERIES
// ═══════════════════════════════════════════════════════════════════════════════

// LatestBlockNumber returns the latest block number for a stream.
// Stream options: "trades", "orders", "events", "book", "twap", "writer_actions"
func (c *HyperCoreClient) LatestBlockNumber(stream ...string) (int64, error) {
	s := "trades"
	if len(stream) > 0 && stream[0] != "" {
		s = stream[0]
	}
	result, err := c.call(context.Background(), "hl_getLatestBlockNumber", map[string]any{"stream": s})
	if err != nil {
		return 0, err
	}
	if n, ok := result.(float64); ok {
		return int64(n), nil
	}
	return 0, nil
}

// GetBlock returns block data by number.
// Stream options: "trades", "orders", "events", "book", "twap", "writer_actions"
func (c *HyperCoreClient) GetBlock(blockNumber int64, stream ...string) (map[string]any, error) {
	s := "trades"
	if len(stream) > 0 && stream[0] != "" {
		s = stream[0]
	}
	// Uses array format: [stream, block_number]
	result, err := c.call(context.Background(), "hl_getBlock", []any{s, blockNumber})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// GetBatchBlocks returns a range of blocks.
// Stream options: "trades", "orders", "events", "book", "twap", "writer_actions"
func (c *HyperCoreClient) GetBatchBlocks(fromBlock, toBlock int64, stream ...string) ([]any, error) {
	s := "trades"
	if len(stream) > 0 && stream[0] != "" {
		s = stream[0]
	}
	result, err := c.call(context.Background(), "hl_getBatchBlocks", map[string]any{
		"stream": s,
		"from":   fromBlock,
		"to":     toBlock,
	})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// LatestBlocks returns the latest N blocks for a stream.
// Stream options: "trades", "orders", "book_updates", "twap", "events", "writer_actions"
func (c *HyperCoreClient) LatestBlocks(stream string, count int) (map[string]any, error) {
	result, err := c.call(context.Background(), "hl_getLatestBlocks", map[string]any{
		"stream": stream,
		"count":  count,
	})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// RECENT DATA (Alternative to unsupported Info methods)
// ═══════════════════════════════════════════════════════════════════════════════

// LatestTrades returns recent trades from latest blocks.
// This is an alternative to Info.RecentTrades() which may not be available on QuickNode.
func (c *HyperCoreClient) LatestTrades(count int, coin string) ([]map[string]any, error) {
	result, err := c.LatestBlocks("trades", count)
	if err != nil {
		return nil, err
	}

	var trades []map[string]any
	blocks, _ := result["blocks"].([]any)
	for _, block := range blocks {
		blockMap, ok := block.(map[string]any)
		if !ok {
			continue
		}
		events, _ := blockMap["events"].([]any)
		for _, event := range events {
			eventArr, ok := event.([]any)
			if !ok || len(eventArr) < 2 {
				continue
			}
			user, _ := eventArr[0].(string)
			trade, ok := eventArr[1].(map[string]any)
			if !ok {
				continue
			}
			// Filter by coin if specified
			if coin != "" {
				tradeCoin, _ := trade["coin"].(string)
				if tradeCoin != coin {
					continue
				}
			}
			// Add user to trade data
			tradeCopy := make(map[string]any)
			tradeCopy["user"] = user
			for k, v := range trade {
				tradeCopy[k] = v
			}
			trades = append(trades, tradeCopy)
		}
	}
	return trades, nil
}

// LatestOrders returns recent order events from latest blocks.
func (c *HyperCoreClient) LatestOrders(count int, coin string) ([]map[string]any, error) {
	result, err := c.LatestBlocks("orders", count)
	if err != nil {
		return nil, err
	}

	var orders []map[string]any
	blocks, _ := result["blocks"].([]any)
	for _, block := range blocks {
		blockMap, ok := block.(map[string]any)
		if !ok {
			continue
		}
		events, _ := blockMap["events"].([]any)
		for _, event := range events {
			// Orders stream returns dict events with 'user', 'order', 'status' fields
			eventMap, ok := event.(map[string]any)
			if !ok {
				continue
			}
			orderData, ok := eventMap["order"].(map[string]any)
			if !ok {
				continue
			}
			// Filter by coin if specified
			if coin != "" {
				orderCoin, _ := orderData["coin"].(string)
				if orderCoin != coin {
					continue
				}
			}
			// Build order entry with user and status
			orderCopy := make(map[string]any)
			orderCopy["user"] = eventMap["user"]
			orderCopy["status"] = eventMap["status"]
			for k, v := range orderData {
				orderCopy[k] = v
			}
			orders = append(orders, orderCopy)
		}
	}
	return orders, nil
}

// LatestBookUpdates returns recent book updates from latest blocks.
func (c *HyperCoreClient) LatestBookUpdates(count int, coin string) ([]map[string]any, error) {
	result, err := c.LatestBlocks("book_updates", count)
	if err != nil {
		return nil, err
	}

	var updates []map[string]any
	blocks, _ := result["blocks"].([]any)
	for _, block := range blocks {
		blockMap, ok := block.(map[string]any)
		if !ok {
			continue
		}
		events, _ := blockMap["events"].([]any)
		for _, event := range events {
			eventMap, ok := event.(map[string]any)
			if !ok {
				continue
			}
			// Filter by coin if specified
			if coin != "" {
				eventCoin, _ := eventMap["coin"].(string)
				if eventCoin != coin {
					continue
				}
			}
			updates = append(updates, eventMap)
		}
	}
	return updates, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// MARKET DISCOVERY
// ═══════════════════════════════════════════════════════════════════════════════

// ListDexes returns all available DEX names.
func (c *HyperCoreClient) ListDexes() ([]string, error) {
	result, err := c.call(context.Background(), "hl_listDexes", []any{})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		var dexes []string
		for _, d := range arr {
			if s, ok := d.(string); ok {
				dexes = append(dexes, s)
			}
		}
		return dexes, nil
	}
	return nil, nil
}

// ListMarkets returns all available markets, optionally filtered by DEX.
func (c *HyperCoreClient) ListMarkets(dex string) ([]any, error) {
	params := map[string]any{}
	if dex != "" {
		params["dex"] = dex
	}
	result, err := c.call(context.Background(), "hl_listMarkets", params)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// ORDER QUERIES (User-specific)
// ═══════════════════════════════════════════════════════════════════════════════

// OpenOrders returns open orders for a user.
func (c *HyperCoreClient) OpenOrders(user string) ([]any, error) {
	result, err := c.call(context.Background(), "hl_openOrders", map[string]any{"user": user})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		return arr, nil
	}
	return nil, nil
}

// OrderStatus returns status of a specific order.
func (c *HyperCoreClient) OrderStatus(user string, oid int64) (map[string]any, error) {
	result, err := c.call(context.Background(), "hl_orderStatus", map[string]any{"user": user, "oid": oid})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// Preflight validates an order before signing.
func (c *HyperCoreClient) Preflight(coin string, isBuy bool, limitPx, sz, user string, reduceOnly bool, orderType map[string]any) (map[string]any, error) {
	params := map[string]any{
		"coin":       coin,
		"isBuy":      isBuy,
		"limitPx":    limitPx,
		"sz":         sz,
		"user":       user,
		"reduceOnly": reduceOnly,
	}
	if orderType != nil {
		params["orderType"] = orderType
	}
	result, err := c.call(context.Background(), "hl_preflight", params)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// BUILDER FEE
// ═══════════════════════════════════════════════════════════════════════════════

// GetMaxBuilderFee returns maximum builder fee for a user-builder pair.
func (c *HyperCoreClient) GetMaxBuilderFee(user, builder string) (map[string]any, error) {
	result, err := c.call(context.Background(), "hl_getMaxBuilderFee", map[string]any{"user": user, "builder": builder})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// ORDER BUILDING (Returns unsigned actions for signing)
// ═══════════════════════════════════════════════════════════════════════════════

// BuildOrder builds an order action for signing.
func (c *HyperCoreClient) BuildOrder(coin string, isBuy bool, limitPx, sz, user string, reduceOnly bool, orderType map[string]any, cloid string) (map[string]any, error) {
	params := map[string]any{
		"coin":       coin,
		"isBuy":      isBuy,
		"limitPx":    limitPx,
		"sz":         sz,
		"user":       user,
		"reduceOnly": reduceOnly,
	}
	if orderType != nil {
		params["orderType"] = orderType
	}
	if cloid != "" {
		params["cloid"] = cloid
	}
	result, err := c.call(context.Background(), "hl_buildOrder", params)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// BuildCancel builds a cancel action for signing.
func (c *HyperCoreClient) BuildCancel(coin string, oid int64, user string) (map[string]any, error) {
	result, err := c.call(context.Background(), "hl_buildCancel", map[string]any{"coin": coin, "oid": oid, "user": user})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// BuildModify builds a modify action for signing.
func (c *HyperCoreClient) BuildModify(coin string, oid int64, user string, limitPx, sz *string, isBuy *bool) (map[string]any, error) {
	params := map[string]any{"coin": coin, "oid": oid, "user": user}
	if limitPx != nil {
		params["limitPx"] = *limitPx
	}
	if sz != nil {
		params["sz"] = *sz
	}
	if isBuy != nil {
		params["isBuy"] = *isBuy
	}
	result, err := c.call(context.Background(), "hl_buildModify", params)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// BuildApproveBuilderFee builds a builder fee approval for signing.
func (c *HyperCoreClient) BuildApproveBuilderFee(user, builder, maxFeeRate string, nonce int64) (map[string]any, error) {
	result, err := c.call(context.Background(), "hl_buildApproveBuilderFee", map[string]any{
		"user":       user,
		"builder":    builder,
		"maxFeeRate": maxFeeRate,
		"nonce":      nonce,
	})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// BuildRevokeBuilderFee builds a builder fee revocation for signing.
func (c *HyperCoreClient) BuildRevokeBuilderFee(user, builder string, nonce int64) (map[string]any, error) {
	result, err := c.call(context.Background(), "hl_buildRevokeBuilderFee", map[string]any{
		"user":    user,
		"builder": builder,
		"nonce":   nonce,
	})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// SENDING SIGNED ACTIONS
// ═══════════════════════════════════════════════════════════════════════════════

// SendOrder sends a signed order.
func (c *HyperCoreClient) SendOrder(action map[string]any, signature string, nonce int64) (map[string]any, error) {
	result, err := c.call(context.Background(), "hl_sendOrder", map[string]any{
		"action":    action,
		"signature": signature,
		"nonce":     nonce,
	})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// SendCancel sends a signed cancel.
func (c *HyperCoreClient) SendCancel(action map[string]any, signature string, nonce int64) (map[string]any, error) {
	result, err := c.call(context.Background(), "hl_sendCancel", map[string]any{
		"action":    action,
		"signature": signature,
		"nonce":     nonce,
	})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// SendModify sends a signed modify.
func (c *HyperCoreClient) SendModify(action map[string]any, signature string, nonce int64) (map[string]any, error) {
	result, err := c.call(context.Background(), "hl_sendModify", map[string]any{
		"action":    action,
		"signature": signature,
		"nonce":     nonce,
	})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// SendApproval sends a signed builder fee approval.
func (c *HyperCoreClient) SendApproval(action map[string]any, signature string) (map[string]any, error) {
	result, err := c.call(context.Background(), "hl_sendApproval", map[string]any{
		"action":    action,
		"signature": signature,
	})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// SendRevocation sends a signed builder fee revocation.
func (c *HyperCoreClient) SendRevocation(action map[string]any, signature string) (map[string]any, error) {
	result, err := c.call(context.Background(), "hl_sendRevocation", map[string]any{
		"action":    action,
		"signature": signature,
	})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// WEBSOCKET SUBSCRIPTIONS (via JSON-RPC)
// ═══════════════════════════════════════════════════════════════════════════════

// Subscribe subscribes to a WebSocket stream via JSON-RPC.
func (c *HyperCoreClient) Subscribe(subscription map[string]any) (map[string]any, error) {
	result, err := c.call(context.Background(), "hl_subscribe", map[string]any{"subscription": subscription})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// Unsubscribe unsubscribes from a WebSocket stream.
func (c *HyperCoreClient) Unsubscribe(subscription map[string]any) (map[string]any, error) {
	result, err := c.call(context.Background(), "hl_unsubscribe", map[string]any{"subscription": subscription})
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}
