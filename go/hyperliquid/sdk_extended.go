package hyperliquid

import (
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════════
// TWAP ORDERS
// ═══════════════════════════════════════════════════════════════════════════════

// TWAPOrder places a TWAP (Time-Weighted Average Price) order.
func (s *SDK) TWAPOrder(asset string, size any, isBuy bool, durationMinutes int, opts ...TWAPOption) (map[string]any, error) {
	params := &twapParams{
		randomize: true,
	}
	for _, opt := range opts {
		opt(params)
	}

	assetIdx, err := s.resolveAssetIndex(asset)
	if err != nil {
		return nil, err
	}

	action := map[string]any{
		"type": "twapOrder",
		"twap": map[string]any{
			"a": assetIdx,
			"b": isBuy,
			"s": NewDecimal(size).String(),
			"r": params.reduceOnly,
			"m": durationMinutes,
			"t": params.randomize,
		},
	}

	return s.buildSignSend(action, nil)
}

// TWAPCancel cancels an active TWAP order.
func (s *SDK) TWAPCancel(asset string, twapID int) (map[string]any, error) {
	assetIdx, err := s.resolveAssetIndex(asset)
	if err != nil {
		return nil, err
	}

	action := map[string]any{
		"type": "twapCancel",
		"a":    assetIdx,
		"t":    twapID,
	}

	return s.buildSignSend(action, nil)
}

type twapParams struct {
	reduceOnly bool
	randomize  bool
}

// TWAPOption is an option for TWAP orders.
type TWAPOption func(*twapParams)

// TWAPWithReduceOnly marks the TWAP as reduce-only.
func TWAPWithReduceOnly() TWAPOption {
	return func(p *twapParams) {
		p.reduceOnly = true
	}
}

// TWAPWithRandomize enables/disables slice timing randomization.
func TWAPWithRandomize(randomize bool) TWAPOption {
	return func(p *twapParams) {
		p.randomize = randomize
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// LEVERAGE MANAGEMENT
// ═══════════════════════════════════════════════════════════════════════════════

// UpdateLeverage updates leverage for an asset.
func (s *SDK) UpdateLeverage(asset string, leverage int, opts ...LeverageOption) (map[string]any, error) {
	params := &leverageParams{
		isCross: true,
	}
	for _, opt := range opts {
		opt(params)
	}

	assetIdx, err := s.resolveAssetIndex(asset)
	if err != nil {
		return nil, err
	}

	action := map[string]any{
		"type":     "updateLeverage",
		"asset":    assetIdx,
		"isCross":  params.isCross,
		"leverage": leverage,
	}

	return s.buildSignSend(action, nil)
}

type leverageParams struct {
	isCross bool
}

// LeverageOption is an option for leverage updates.
type LeverageOption func(*leverageParams)

// LeverageWithIsolated sets isolated margin mode.
func LeverageWithIsolated() LeverageOption {
	return func(p *leverageParams) {
		p.isCross = false
	}
}

// UpdateIsolatedMargin adds or removes margin from an isolated position.
func (s *SDK) UpdateIsolatedMargin(asset string, isBuy bool, amount float64) (map[string]any, error) {
	assetIdx, err := s.resolveAssetIndex(asset)
	if err != nil {
		return nil, err
	}

	// API expects amount in millionths (1000000 = 1 USD)
	ntli := int64(amount * 1_000_000)

	action := map[string]any{
		"type":  "updateIsolatedMargin",
		"asset": assetIdx,
		"isBuy": isBuy,
		"ntli":  ntli,
	}

	return s.buildSignSend(action, nil)
}

// TopUpIsolatedOnlyMargin tops up isolated margin to target leverage.
func (s *SDK) TopUpIsolatedOnlyMargin(asset string, leverage float64) (map[string]any, error) {
	assetIdx, err := s.resolveAssetIndex(asset)
	if err != nil {
		return nil, err
	}

	action := map[string]any{
		"type":     "topUpIsolatedOnlyMargin",
		"asset":    assetIdx,
		"leverage": NewDecimal(leverage).String(),
	}

	return s.buildSignSend(action, nil)
}

// ═══════════════════════════════════════════════════════════════════════════════
// TRANSFER OPERATIONS
// ═══════════════════════════════════════════════════════════════════════════════

// TransferUSD transfers USDC to another Hyperliquid address.
func (s *SDK) TransferUSD(destination string, amount any) (map[string]any, error) {
	action := map[string]any{
		"type":             "usdSend",
		"hyperliquidChain": s.chain,
		"signatureChainId": s.chainID,
		"destination":      destination,
		"amount":           NewDecimal(amount).String(),
		"time":             time.Now().UnixMilli(),
	}

	return s.buildSignSend(action, nil)
}

// TransferSpot transfers spot tokens to another Hyperliquid address.
func (s *SDK) TransferSpot(token, destination string, amount any) (map[string]any, error) {
	action := map[string]any{
		"type":             "spotSend",
		"hyperliquidChain": s.chain,
		"signatureChainId": s.chainID,
		"token":            token,
		"destination":      destination,
		"amount":           NewDecimal(amount).String(),
		"time":             time.Now().UnixMilli(),
	}

	return s.buildSignSend(action, nil)
}

// Withdraw initiates a withdrawal to Arbitrum.
func (s *SDK) Withdraw(amount any, destination string) (map[string]any, error) {
	if destination == "" {
		s.requireWallet()
		destination = s.Address()
	}

	action := map[string]any{
		"type":             "withdraw3",
		"hyperliquidChain": s.chain,
		"signatureChainId": s.chainID,
		"destination":      destination,
		"amount":           NewDecimal(amount).String(),
		"time":             time.Now().UnixMilli(),
	}

	return s.buildSignSend(action, nil)
}

// TransferSpotToPerp transfers USDC from spot balance to perp balance.
func (s *SDK) TransferSpotToPerp(amount any) (map[string]any, error) {
	action := map[string]any{
		"type":             "usdClassTransfer",
		"hyperliquidChain": s.chain,
		"signatureChainId": s.chainID,
		"amount":           NewDecimal(amount).String(),
		"toPerp":           true,
		"nonce":            time.Now().UnixMilli(),
	}

	return s.buildSignSend(action, nil)
}

// TransferPerpToSpot transfers USDC from perp balance to spot balance.
func (s *SDK) TransferPerpToSpot(amount any) (map[string]any, error) {
	action := map[string]any{
		"type":             "usdClassTransfer",
		"hyperliquidChain": s.chain,
		"signatureChainId": s.chainID,
		"amount":           NewDecimal(amount).String(),
		"toPerp":           false,
		"nonce":            time.Now().UnixMilli(),
	}

	return s.buildSignSend(action, nil)
}

// ═══════════════════════════════════════════════════════════════════════════════
// VAULT OPERATIONS
// ═══════════════════════════════════════════════════════════════════════════════

// VaultDeposit deposits USDC into a vault.
func (s *SDK) VaultDeposit(vaultAddress string, amount float64) (map[string]any, error) {
	action := map[string]any{
		"type":         "vaultTransfer",
		"vaultAddress": vaultAddress,
		"isDeposit":    true,
		"usd":          amount,
	}

	return s.buildSignSend(action, nil)
}

// VaultWithdraw withdraws USDC from a vault.
func (s *SDK) VaultWithdraw(vaultAddress string, amount float64) (map[string]any, error) {
	action := map[string]any{
		"type":         "vaultTransfer",
		"vaultAddress": vaultAddress,
		"isDeposit":    false,
		"usd":          amount,
	}

	return s.buildSignSend(action, nil)
}

// ═══════════════════════════════════════════════════════════════════════════════
// AGENT/API KEY MANAGEMENT
// ═══════════════════════════════════════════════════════════════════════════════

// ApproveAgent approves an agent (API wallet) to trade on your behalf.
func (s *SDK) ApproveAgent(agentAddress, name string) (map[string]any, error) {
	action := map[string]any{
		"type":             "approveAgent",
		"hyperliquidChain": s.chain,
		"signatureChainId": s.chainID,
		"agentAddress":     agentAddress,
		"agentName":        name,
		"nonce":            time.Now().UnixMilli(),
	}

	return s.buildSignSend(action, nil)
}

// ═══════════════════════════════════════════════════════════════════════════════
// STAKING OPERATIONS
// ═══════════════════════════════════════════════════════════════════════════════

// Stake stakes tokens.
func (s *SDK) Stake(amount float64) (map[string]any, error) {
	// Convert to wei (18 decimals)
	wei := int64(amount * 1e18)

	action := map[string]any{
		"type":             "cDeposit",
		"hyperliquidChain": s.chain,
		"signatureChainId": s.chainID,
		"wei":              wei,
		"nonce":            time.Now().UnixMilli(),
	}

	return s.buildSignSend(action, nil)
}

// Unstake unstakes tokens (7-day queue).
func (s *SDK) Unstake(amount float64) (map[string]any, error) {
	wei := int64(amount * 1e18)

	action := map[string]any{
		"type":             "cWithdraw",
		"hyperliquidChain": s.chain,
		"signatureChainId": s.chainID,
		"wei":              wei,
		"nonce":            time.Now().UnixMilli(),
	}

	return s.buildSignSend(action, nil)
}

// Delegate delegates staked tokens to a validator.
func (s *SDK) Delegate(validator string, amount float64) (map[string]any, error) {
	wei := int64(amount * 1e18)

	action := map[string]any{
		"type":             "tokenDelegate",
		"hyperliquidChain": s.chain,
		"signatureChainId": s.chainID,
		"validator":        validator,
		"isUndelegate":     false,
		"wei":              wei,
		"nonce":            time.Now().UnixMilli(),
	}

	return s.buildSignSend(action, nil)
}

// Undelegate undelegates staked tokens from a validator.
func (s *SDK) Undelegate(validator string, amount float64) (map[string]any, error) {
	wei := int64(amount * 1e18)

	action := map[string]any{
		"type":             "tokenDelegate",
		"hyperliquidChain": s.chain,
		"signatureChainId": s.chainID,
		"validator":        validator,
		"isUndelegate":     true,
		"wei":              wei,
		"nonce":            time.Now().UnixMilli(),
	}

	return s.buildSignSend(action, nil)
}

// ═══════════════════════════════════════════════════════════════════════════════
// ACCOUNT ABSTRACTION
// ═══════════════════════════════════════════════════════════════════════════════

// SetAbstraction sets account abstraction mode.
func (s *SDK) SetAbstraction(mode string, user string) (map[string]any, error) {
	if user == "" {
		s.requireWallet()
		user = s.Address()
	}

	action := map[string]any{
		"type":             "userSetAbstraction",
		"hyperliquidChain": s.chain,
		"signatureChainId": s.chainID,
		"user":             user,
		"abstraction":      mode,
		"nonce":            time.Now().UnixMilli(),
	}

	return s.buildSignSend(action, nil)
}

// AgentSetAbstraction sets account abstraction mode as an agent.
func (s *SDK) AgentSetAbstraction(mode string) (map[string]any, error) {
	// Map full mode names to short codes
	modeMap := map[string]string{
		"disabled":       "i",
		"unifiedAccount": "u",
		"portfolioMargin": "p",
		"i":              "i",
		"u":              "u",
		"p":              "p",
	}

	shortMode, ok := modeMap[mode]
	if !ok {
		return nil, ValidationError("invalid mode: use 'disabled', 'unifiedAccount', or 'portfolioMargin'")
	}

	action := map[string]any{
		"type":        "agentSetAbstraction",
		"abstraction": shortMode,
	}

	return s.buildSignSend(action, nil)
}

// ═══════════════════════════════════════════════════════════════════════════════
// ADVANCED TRANSFERS
// ═══════════════════════════════════════════════════════════════════════════════

// SendAsset performs a generalized asset transfer between DEXs and accounts.
func (s *SDK) SendAsset(token string, amount any, destination string, opts ...SendAssetOption) (map[string]any, error) {
	params := &sendAssetParams{}
	for _, opt := range opts {
		opt(params)
	}

	action := map[string]any{
		"type":             "sendAsset",
		"hyperliquidChain": s.chain,
		"signatureChainId": s.chainID,
		"destination":      destination,
		"sourceDex":        params.sourceDex,
		"destinationDex":   params.destinationDex,
		"token":            token,
		"amount":           NewDecimal(amount).String(),
		"fromSubAccount":   params.fromSubAccount,
		"nonce":            time.Now().UnixMilli(),
	}

	return s.buildSignSend(action, nil)
}

type sendAssetParams struct {
	sourceDex      string
	destinationDex string
	fromSubAccount string
}

// SendAssetOption is an option for SendAsset.
type SendAssetOption func(*sendAssetParams)

// SendAssetFromDex sets the source DEX.
func SendAssetFromDex(dex string) SendAssetOption {
	return func(p *sendAssetParams) {
		p.sourceDex = dex
	}
}

// SendAssetToDex sets the destination DEX.
func SendAssetToDex(dex string) SendAssetOption {
	return func(p *sendAssetParams) {
		p.destinationDex = dex
	}
}

// SendAssetFromSubAccount sets the source sub-account.
func SendAssetFromSubAccount(addr string) SendAssetOption {
	return func(p *sendAssetParams) {
		p.fromSubAccount = addr
	}
}

// SendToEVMWithData transfers tokens to HyperEVM with custom data payload.
func (s *SDK) SendToEVMWithData(token string, amount any, destination, data, sourceDex string, destChainID int, gasLimit int) (map[string]any, error) {
	action := map[string]any{
		"type":                 "sendToEvmWithData",
		"hyperliquidChain":     s.chain,
		"signatureChainId":     s.chainID,
		"token":                token,
		"amount":               NewDecimal(amount).String(),
		"sourceDex":            sourceDex,
		"destinationRecipient": destination,
		"addressEncoding":      "hex",
		"destinationChainId":   destChainID,
		"gasLimit":             gasLimit,
		"data":                 data,
		"nonce":                time.Now().UnixMilli(),
	}

	return s.buildSignSend(action, nil)
}

// ═══════════════════════════════════════════════════════════════════════════════
// RATE LIMITING
// ═══════════════════════════════════════════════════════════════════════════════

// ReserveRequestWeight purchases additional rate limit capacity.
func (s *SDK) ReserveRequestWeight(weight int) (map[string]any, error) {
	action := map[string]any{
		"type":   "reserveRequestWeight",
		"weight": weight,
	}

	return s.buildSignSend(action, nil)
}

// ═══════════════════════════════════════════════════════════════════════════════
// UTILITY OPERATIONS
// ═══════════════════════════════════════════════════════════════════════════════

// Noop performs a no-operation to consume a nonce.
func (s *SDK) Noop() (map[string]any, error) {
	action := map[string]any{
		"type": "noop",
	}

	return s.buildSignSend(action, nil)
}

// ValidatorL1Stream submits a validator vote for the risk-free rate.
func (s *SDK) ValidatorL1Stream(riskFreeRate string) (map[string]any, error) {
	action := map[string]any{
		"type":         "validatorL1Stream",
		"riskFreeRate": riskFreeRate,
	}

	return s.buildSignSend(action, nil)
}

// ═══════════════════════════════════════════════════════════════════════════════
// APPROVAL MANAGEMENT
// ═══════════════════════════════════════════════════════════════════════════════

// ApproveBuilderFee approves builder fee for trading.
func (s *SDK) ApproveBuilderFee(maxFee, builder string) (map[string]any, error) {
	if builder == "" {
		builder = DefaultBuilderAddress
	}

	action := map[string]any{
		"type":             "approveBuilderFee",
		"hyperliquidChain": s.chain,
		"signatureChainId": s.chainID,
		"maxFeeRate":       maxFee,
		"builder":          builder,
		"nonce":            time.Now().UnixMilli(),
	}

	return s.buildSignSend(action, nil)
}

// RevokeBuilderFee revokes builder fee approval.
func (s *SDK) RevokeBuilderFee(builder string) (map[string]any, error) {
	return s.ApproveBuilderFee("0%", builder)
}
