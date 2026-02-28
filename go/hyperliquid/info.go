package hyperliquid

import (
	"context"
)

// InfoClient provides access to the HyperCore Info API.
type InfoClient struct {
	infoURL       string
	workerInfoURL string
	http          *HTTPClient
}

// NewInfoClient creates a new Info API client.
func NewInfoClient(endpoint string, http *HTTPClient) *InfoClient {
	return &InfoClient{
		infoURL:       buildInfoURL(endpoint),
		workerInfoURL: DefaultWorkerURL + "/info",
		http:          http,
	}
}

func (c *InfoClient) post(ctx context.Context, body map[string]any) (any, error) {
	reqType, _ := body["type"].(string)

	var url string
	if QNSupportedInfoMethods[reqType] {
		url = c.infoURL
	} else {
		url = c.workerInfoURL
	}

	return c.http.PostRaw(ctx, url, body)
}

// ═══════════════════════════════════════════════════════════════════════════════
// MARKET DATA
// ═══════════════════════════════════════════════════════════════════════════════

// AllMids returns all asset mid prices.
func (c *InfoClient) AllMids(opts ...InfoOption) (map[string]any, error) {
	params := applyInfoOptions(opts...)
	body := map[string]any{"type": "allMids"}
	if params.dex != "" {
		body["dex"] = params.dex
	}
	result, err := c.post(context.Background(), body)
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// L2Book returns the Level 2 order book for an asset.
func (c *InfoClient) L2Book(coin string, opts ...InfoOption) (map[string]any, error) {
	params := applyInfoOptions(opts...)
	body := map[string]any{"type": "l2Book", "coin": coin}
	if params.nSigFigs > 0 {
		body["nSigFigs"] = params.nSigFigs
	}
	if params.mantissa > 0 {
		body["mantissa"] = params.mantissa
	}
	result, err := c.post(context.Background(), body)
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// RecentTrades returns recent trades for an asset.
func (c *InfoClient) RecentTrades(coin string) ([]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "recentTrades", "coin": coin})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// Candles returns historical OHLCV candlestick data.
func (c *InfoClient) Candles(coin, interval string, startTime, endTime int64) ([]any, error) {
	body := map[string]any{
		"type": "candleSnapshot",
		"req": map[string]any{
			"coin":      coin,
			"interval":  interval,
			"startTime": startTime,
			"endTime":   endTime,
		},
	}
	result, err := c.post(context.Background(), body)
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// FundingHistory returns historical funding rates for an asset.
func (c *InfoClient) FundingHistory(coin string, startTime int64, endTime *int64) ([]any, error) {
	body := map[string]any{"type": "fundingHistory", "coin": coin, "startTime": startTime}
	if endTime != nil {
		body["endTime"] = *endTime
	}
	result, err := c.post(context.Background(), body)
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// PredictedFundings returns predicted funding rates for all assets.
func (c *InfoClient) PredictedFundings() ([]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "predictedFundings"})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// METADATA
// ═══════════════════════════════════════════════════════════════════════════════

// Meta returns exchange metadata including assets and margin configurations.
func (c *InfoClient) Meta() (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "meta"})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// SpotMeta returns spot trading metadata.
func (c *InfoClient) SpotMeta() (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "spotMeta"})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// MetaAndAssetCtxs returns metadata + real-time asset context.
func (c *InfoClient) MetaAndAssetCtxs() (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "metaAndAssetCtxs"})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// SpotMetaAndAssetCtxs returns spot metadata + real-time asset context.
func (c *InfoClient) SpotMetaAndAssetCtxs() (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "spotMetaAndAssetCtxs"})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// ExchangeStatus returns current exchange status.
func (c *InfoClient) ExchangeStatus() (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "exchangeStatus"})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// PerpDexs returns perpetual DEX information.
func (c *InfoClient) PerpDexs() ([]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "perpDexs"})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// USER ACCOUNT
// ═══════════════════════════════════════════════════════════════════════════════

// ClearinghouseState returns user's perpetual positions and margin info.
func (c *InfoClient) ClearinghouseState(user string, opts ...InfoOption) (map[string]any, error) {
	params := applyInfoOptions(opts...)
	body := map[string]any{"type": "clearinghouseState", "user": user}
	if params.dex != "" {
		body["dex"] = params.dex
	}
	result, err := c.post(context.Background(), body)
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// SpotClearinghouseState returns user's spot token balances.
func (c *InfoClient) SpotClearinghouseState(user string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "spotClearinghouseState", "user": user})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// OpenOrders returns user's open orders.
func (c *InfoClient) OpenOrders(user string, opts ...InfoOption) ([]any, error) {
	params := applyInfoOptions(opts...)
	body := map[string]any{"type": "openOrders", "user": user}
	if params.dex != "" {
		body["dex"] = params.dex
	}
	result, err := c.post(context.Background(), body)
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// FrontendOpenOrders returns user's open orders with enhanced info.
func (c *InfoClient) FrontendOpenOrders(user string, opts ...InfoOption) ([]any, error) {
	params := applyInfoOptions(opts...)
	body := map[string]any{"type": "frontendOpenOrders", "user": user}
	if params.dex != "" {
		body["dex"] = params.dex
	}
	result, err := c.post(context.Background(), body)
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// OrderStatus returns status of a specific order.
func (c *InfoClient) OrderStatus(user string, oid int64, opts ...InfoOption) (map[string]any, error) {
	params := applyInfoOptions(opts...)
	body := map[string]any{"type": "orderStatus", "user": user, "oid": oid}
	if params.dex != "" {
		body["dex"] = params.dex
	}
	result, err := c.post(context.Background(), body)
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// HistoricalOrders returns user's historical orders.
func (c *InfoClient) HistoricalOrders(user string) ([]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "historicalOrders", "user": user})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// UserFills returns user's trade fills.
func (c *InfoClient) UserFills(user string, aggregateByTime bool) ([]any, error) {
	body := map[string]any{"type": "userFills", "user": user}
	if aggregateByTime {
		body["aggregateByTime"] = true
	}
	result, err := c.post(context.Background(), body)
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// UserFillsByTime returns user's trade fills within a time range.
func (c *InfoClient) UserFillsByTime(user string, startTime int64, endTime *int64) ([]any, error) {
	body := map[string]any{"type": "userFillsByTime", "user": user, "startTime": startTime}
	if endTime != nil {
		body["endTime"] = *endTime
	}
	result, err := c.post(context.Background(), body)
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// UserFunding returns user's funding payments.
func (c *InfoClient) UserFunding(user string, startTime, endTime *int64) ([]any, error) {
	body := map[string]any{"type": "userFunding", "user": user}
	if startTime != nil {
		body["startTime"] = *startTime
	}
	if endTime != nil {
		body["endTime"] = *endTime
	}
	result, err := c.post(context.Background(), body)
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// UserFees returns user's fee structure.
func (c *InfoClient) UserFees(user string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "userFees", "user": user})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// UserRateLimit returns user's rate limit status.
func (c *InfoClient) UserRateLimit(user string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "userRateLimit", "user": user})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// Portfolio returns user's portfolio history.
func (c *InfoClient) Portfolio(user string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "portfolio", "user": user})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// WebData2 returns comprehensive account snapshot.
func (c *InfoClient) WebData2(user string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "webData2", "user": user})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// SubAccounts returns user's sub-accounts.
func (c *InfoClient) SubAccounts(user string) ([]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "subAccounts", "user": user})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// ExtraAgents returns user's extra agents (API keys).
func (c *InfoClient) ExtraAgents(user string) ([]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "extraAgents", "user": user})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// BATCH QUERIES
// ═══════════════════════════════════════════════════════════════════════════════

// BatchClearinghouseStates returns clearinghouse states for multiple users.
func (c *InfoClient) BatchClearinghouseStates(users []string) ([]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "batchClearinghouseStates", "users": users})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// VAULTS
// ═══════════════════════════════════════════════════════════════════════════════

// VaultSummaries returns summaries of all vaults.
func (c *InfoClient) VaultSummaries() ([]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "vaultSummaries"})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// VaultDetails returns vault details.
func (c *InfoClient) VaultDetails(vaultAddress string, user string) (map[string]any, error) {
	body := map[string]any{"type": "vaultDetails", "vaultAddress": vaultAddress}
	if user != "" {
		body["user"] = user
	}
	result, err := c.post(context.Background(), body)
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// LeadingVaults returns vaults that user is leading.
func (c *InfoClient) LeadingVaults(user string) ([]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "leadingVaults", "user": user})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// UserVaultEquities returns user's vault equities.
func (c *InfoClient) UserVaultEquities(user string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "userVaultEquities", "user": user})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// DELEGATION / STAKING
// ═══════════════════════════════════════════════════════════════════════════════

// Delegations returns user's delegations.
func (c *InfoClient) Delegations(user string) ([]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "delegations", "user": user})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// DelegatorHistory returns user's delegation history.
func (c *InfoClient) DelegatorHistory(user string) ([]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "delegatorHistory", "user": user})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// DelegatorRewards returns user's delegation rewards.
func (c *InfoClient) DelegatorRewards(user string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "delegatorRewards", "user": user})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// DelegatorSummary returns user's delegation summary.
func (c *InfoClient) DelegatorSummary(user string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "delegatorSummary", "user": user})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// OTHER
// ═══════════════════════════════════════════════════════════════════════════════

// MaxBuilderFee returns maximum builder fee for a user-builder pair.
func (c *InfoClient) MaxBuilderFee(user, builder string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "maxBuilderFee", "user": user, "builder": builder})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// Liquidatable returns list of liquidatable positions.
func (c *InfoClient) Liquidatable() ([]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "liquidatable"})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// UserTWAPHistory returns user's TWAP order history.
func (c *InfoClient) UserTWAPHistory(user string) ([]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "userTwapHistory", "user": user})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// TOKENS / SPOT
// ═══════════════════════════════════════════════════════════════════════════════

// TokenDetails returns token details.
func (c *InfoClient) TokenDetails(tokenID string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "tokenDetails", "tokenId": tokenID})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// SpotDeployState returns spot deployment state for user.
func (c *InfoClient) SpotDeployState(user string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "spotDeployState", "user": user})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// Referral returns user's referral information.
func (c *InfoClient) Referral(user string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "referral", "user": user})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// ADDITIONAL METHODS
// ═══════════════════════════════════════════════════════════════════════════════

// ActiveAssetData returns user's active asset trading parameters.
func (c *InfoClient) ActiveAssetData(user, coin string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "activeAssetData", "user": user, "coin": coin})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// UserRole returns account type (user, agent, vault, or sub-account).
func (c *InfoClient) UserRole(user string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "userRole", "user": user})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// UserNonFundingLedgerUpdates returns user's non-funding ledger updates.
func (c *InfoClient) UserNonFundingLedgerUpdates(user string, startTime, endTime *int64) ([]any, error) {
	body := map[string]any{"type": "userNonFundingLedgerUpdates", "user": user}
	if startTime != nil {
		body["startTime"] = *startTime
	}
	if endTime != nil {
		body["endTime"] = *endTime
	}
	result, err := c.post(context.Background(), body)
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// UserTWAPSliceFills returns user's TWAP slice fills.
func (c *InfoClient) UserTWAPSliceFills(user string, limit int) ([]any, error) {
	if limit <= 0 {
		limit = 500
	}
	result, err := c.post(context.Background(), map[string]any{"type": "userTwapSliceFills", "user": user, "limit": limit})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// UserToMultiSigSigners returns multi-sig signers for a user.
func (c *InfoClient) UserToMultiSigSigners(user string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "userToMultiSigSigners", "user": user})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// GossipRootIPs returns gossip root IPs for the network.
func (c *InfoClient) GossipRootIPs() ([]string, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "gossipRootIps"})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		ips := make([]string, len(arr))
		for i, v := range arr {
			if s, ok := v.(string); ok {
				ips[i] = s
			}
		}
		return ips, nil
	}
	return nil, nil
}

// MaxMarketOrderNtls returns maximum market order notionals per asset.
func (c *InfoClient) MaxMarketOrderNtls() (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "maxMarketOrderNtls"})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// PerpDeployAuctionStatus returns perpetual deploy auction status.
func (c *InfoClient) PerpDeployAuctionStatus() (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "perpDeployAuctionStatus"})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// PerpsAtOpenInterestCap returns perps that are at their open interest cap.
func (c *InfoClient) PerpsAtOpenInterestCap() ([]string, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "perpsAtOpenInterestCap"})
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]any); ok {
		perps := make([]string, len(arr))
		for i, v := range arr {
			if s, ok := v.(string); ok {
				perps[i] = s
			}
		}
		return perps, nil
	}
	return nil, nil
}

// ValidatorL1Votes returns L1 validator votes.
func (c *InfoClient) ValidatorL1Votes() (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "validatorL1Votes"})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// ApprovedBuilders returns list of approved builders for a user.
func (c *InfoClient) ApprovedBuilders(user string) ([]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "approvedBuilders", "user": user})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// BORROW/LEND
// ═══════════════════════════════════════════════════════════════════════════════

// BorrowLendUserState returns user's borrow/lend positions.
func (c *InfoClient) BorrowLendUserState(user string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "borrowLendUserState", "user": user})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// BorrowLendReserveState returns borrow/lend reserve state for a token.
func (c *InfoClient) BorrowLendReserveState(token int) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "borrowLendReserveState", "token": token})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// AllBorrowLendReserveStates returns borrow/lend reserve states for all tokens.
func (c *InfoClient) AllBorrowLendReserveStates() ([]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "allBorrowLendReserveStates"})
	if err != nil {
		return nil, err
	}
	return result.([]any), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// ACCOUNT ABSTRACTION
// ═══════════════════════════════════════════════════════════════════════════════

// UserAbstraction returns user's account abstraction mode.
func (c *InfoClient) UserAbstraction(user string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "userAbstraction", "user": user})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// UserDexAbstraction returns user's DEX abstraction eligibility.
func (c *InfoClient) UserDexAbstraction(user string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "userDexAbstraction", "user": user})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// EXTENDED PERP DEX INFO
// ═══════════════════════════════════════════════════════════════════════════════

// AllPerpMetas returns consolidated universe, margin tables, asset contexts across all DEXs.
func (c *InfoClient) AllPerpMetas() (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "allPerpMetas"})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// PerpCategories returns asset classifications for perps.
func (c *InfoClient) PerpCategories() (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "perpCategories"})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// PerpAnnotation returns metadata descriptions for a perp.
func (c *InfoClient) PerpAnnotation(asset int) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "perpAnnotation", "asset": asset})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// PerpDexLimits returns OI caps and transfer limits for builder-deployed markets.
func (c *InfoClient) PerpDexLimits(dex string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "perpDexLimits", "dex": dex})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// PerpDexStatus returns total net deposits for builder-deployed markets.
func (c *InfoClient) PerpDexStatus(dex string) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "perpDexStatus", "dex": dex})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// SPOT DEPLOYMENT
// ═══════════════════════════════════════════════════════════════════════════════

// SpotPairDeployAuctionStatus returns Dutch auction status for spot pair deployments.
func (c *InfoClient) SpotPairDeployAuctionStatus() (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "spotPairDeployAuctionStatus"})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// ALIGNED QUOTE TOKEN
// ═══════════════════════════════════════════════════════════════════════════════

// AlignedQuoteTokenInfo returns aligned quote token information.
func (c *InfoClient) AlignedQuoteTokenInfo(token int) (map[string]any, error) {
	result, err := c.post(context.Background(), map[string]any{"type": "alignedQuoteTokenInfo", "token": token})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// INFO OPTIONS
// ═══════════════════════════════════════════════════════════════════════════════

type infoParams struct {
	dex      string
	nSigFigs int
	mantissa int
}

// InfoOption is an option for Info API methods.
type InfoOption func(*infoParams)

// WithDex sets the DEX for HIP-3 queries.
func WithDex(dex string) InfoOption {
	return func(p *infoParams) {
		p.dex = dex
	}
}

// WithNSigFigs sets the number of significant figures for price bucketing.
func WithNSigFigs(n int) InfoOption {
	return func(p *infoParams) {
		p.nSigFigs = n
	}
}

// WithMantissa sets the bucketing mantissa multiplier.
func WithMantissa(m int) InfoOption {
	return func(p *infoParams) {
		p.mantissa = m
	}
}

func applyInfoOptions(opts ...InfoOption) *infoParams {
	p := &infoParams{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}
