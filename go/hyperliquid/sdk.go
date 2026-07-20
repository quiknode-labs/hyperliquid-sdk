package hyperliquid

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultTimeout is the default request timeout.
	DefaultTimeout = 30 * time.Second

	// DefaultSlippage is the default slippage for market orders (3%).
	DefaultSlippage = 0.03

	// DefaultMaxFee is the default max builder fee.
	DefaultMaxFee = "1%"

	// DefaultWorkerURL is the public worker URL.
	DefaultWorkerURL = "https://send.hyperliquidapi.com"

	// DefaultBuilderAddress is the QuickNode builder address.
	DefaultBuilderAddress = "0x8D62d3000eF0639d1fc9667D06BE7BB98d9993F5"

	// CacheTTL is the market metadata cache TTL.
	CacheTTL = 5 * time.Minute

	// HYPEWeiDecimals is Hyperliquid's HYPE staking precision.
	HYPEWeiDecimals = 8
)

// QNSupportedInfoMethods are Info API methods supported by QuickNode nodes.
var QNSupportedInfoMethods = map[string]bool{
	"meta": true, "spotMeta": true, "clearinghouseState": true, "spotClearinghouseState": true,
	"openOrders": true, "exchangeStatus": true, "frontendOpenOrders": true, "liquidatable": true,
	"activeAssetData": true, "maxMarketOrderNtls": true, "vaultSummaries": true, "userVaultEquities": true,
	"leadingVaults": true, "extraAgents": true, "subAccounts": true, "userFees": true, "userRateLimit": true,
	"spotDeployState": true, "perpDeployAuctionStatus": true, "delegations": true, "delegatorSummary": true,
	"maxBuilderFee": true, "userToMultiSigSigners": true, "userRole": true, "perpsAtOpenInterestCap": true,
	"validatorL1Votes": true, "marginTable": true, "perpDexs": true, "webData2": true, "outcomeMeta": true,
}

// Signer signs the 32-byte order hash produced by the Hyperliquid build step
// and returns the signature components. hashHex may be 0x-prefixed. It is the
// external-signing analog of Wallet.SignHash, letting callers sign via a
// remote KMS/HSM/signing service without exposing a private key to the SDK.
//
// The ctx bounds the signer call itself by the configured Timeout (it does not
// govern the build/send HTTP calls, which keep their own per-request timeout);
// a remote signer should honour it for cancellation. Returning an error
// surfaces to the caller as a *Error with code SIGNER_FAILED (see SignerError).
type Signer func(ctx context.Context, hashHex string) (*Signature, error)

// Config holds SDK configuration options.
type Config struct {
	Endpoint      string
	PrivateKey    string
	Signer        Signer
	SignerAddress string
	Testnet       bool
	AutoApprove   bool
	MaxFee        string
	Slippage      float64
	Timeout       time.Duration
}

// Option is a functional option for configuring the SDK.
type Option func(*Config)

// WithPrivateKey sets the private key for signing transactions.
func WithPrivateKey(pk string) Option {
	return func(c *Config) {
		c.PrivateKey = pk
	}
}

// WithSigner sets an external signing callback for transactions, letting the
// caller sign Hyperliquid actions via a remote KMS/HSM/signing service without
// exposing a private key to the SDK. When a signer is set, the SDK is
// sign-capable without an in-process wallet and no private key is read.
//
// The callback receives a context bounded by the configured Timeout, so a slow
// remote signer is cancelled; the build/send HTTP calls keep their own
// per-request timeout, so their behaviour is unchanged. Builder-fee
// auto-approval is skipped under an external signer (there is no in-process
// wallet to sign the approval); call ApproveBuilderFee yourself if you need it.
func WithSigner(fn Signer) Option {
	return func(c *Config) {
		c.Signer = fn
	}
}

// WithSignerAddress sets the acting agent address used by the SDK when signing
// through an external Signer (there is no in-process wallet to derive it from).
// Address() returns this value when no wallet is present.
func WithSignerAddress(addr string) Option {
	return func(c *Config) {
		c.SignerAddress = addr
	}
}

// WithTestnet enables testnet mode.
func WithTestnet(testnet bool) Option {
	return func(c *Config) {
		c.Testnet = testnet
	}
}

// WithAutoApprove enables automatic builder fee approval.
func WithAutoApprove(autoApprove bool) Option {
	return func(c *Config) {
		c.AutoApprove = autoApprove
	}
}

// WithMaxFee sets the maximum builder fee to approve.
func WithMaxFee(maxFee string) Option {
	return func(c *Config) {
		c.MaxFee = maxFee
	}
}

// WithSlippage sets the default slippage for market orders.
func WithSlippage(slippage float64) Option {
	return func(c *Config) {
		c.Slippage = slippage
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// SDK is the main Hyperliquid SDK client.
type SDK struct {
	config        *Config
	http          *HTTPClient
	wallet        *Wallet
	signer        Signer
	signerAddress string

	// URLs
	endpoint        string
	exchangeURL     string
	infoURL         string
	publicWorkerURL string

	// Chain config
	chain   string
	chainID string

	// Caching
	marketsCache     *Markets
	marketsCacheTime time.Time
	szDecimalsCache  map[string]int
	cacheMu          sync.RWMutex

	// Lazy-initialized clients
	info     *InfoClient
	core     *HyperCoreClient
	evm      *EVMClient
	infoOnce sync.Once
	coreOnce sync.Once
	evmOnce  sync.Once
}

// New creates a new SDK instance.
//
// Example:
//
//	sdk, err := hyperliquid.New("https://your-endpoint.quiknode.pro/TOKEN",
//		hyperliquid.WithPrivateKey("0x..."),
//		hyperliquid.WithAutoApprove(true),
//	)
func New(endpoint string, opts ...Option) (*SDK, error) {
	config := &Config{
		Endpoint:    endpoint,
		AutoApprove: true,
		MaxFee:      DefaultMaxFee,
		Slippage:    DefaultSlippage,
		Timeout:     DefaultTimeout,
	}

	for _, opt := range opts {
		opt(config)
	}

	// Get private key from env if not provided.
	// Only fall back to the env key when no external signer was supplied —
	// a signer means the SDK must never hold a private key.
	if config.PrivateKey == "" && config.Signer == nil {
		config.PrivateKey = os.Getenv("PRIVATE_KEY")
	}

	sdk := &SDK{
		config:          config,
		http:            NewHTTPClient(config.Timeout),
		signer:          config.Signer,
		signerAddress:   config.SignerAddress,
		endpoint:        endpoint,
		publicWorkerURL: DefaultWorkerURL,
		szDecimalsCache: make(map[string]int),
	}

	// Chain configuration
	if config.Testnet {
		sdk.chain = "Testnet"
		sdk.chainID = "0x66eee" // Arbitrum Sepolia
	} else {
		sdk.chain = "Mainnet"
		sdk.chainID = "0xa4b1" // Arbitrum
	}

	// Build URLs
	if endpoint != "" {
		sdk.infoURL = buildInfoURL(endpoint)
	} else {
		sdk.infoURL = DefaultWorkerURL + "/info"
	}

	// Trading/exchange ALWAYS goes through the public worker
	// QuickNode /send endpoint is not used for trading
	sdk.exchangeURL = DefaultWorkerURL + "/exchange"

	// Initialize wallet if private key provided.
	// When an external signer is set, the SDK is sign-capable without a wallet,
	// so we skip the wallet (and the auto-approve block, which needs one).
	if config.Signer == nil && config.PrivateKey != "" {
		wallet, err := NewWallet(config.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create wallet: %w", err)
		}
		sdk.wallet = wallet

		// Auto-approve if requested
		if config.AutoApprove {
			if err := sdk.ensureApproved(config.MaxFee); err != nil {
				// Log warning but don't fail initialization
				// Approval might already exist
			}
		}
	}

	return sdk, nil
}

// Close closes the SDK and releases resources.
func (s *SDK) Close() {
	s.http.Close()
}

// Address returns the acting agent address: the in-process wallet's address,
// or the configured signer address when signing through an external Signer,
// or empty string if neither is set.
func (s *SDK) Address() string {
	if s.wallet == nil {
		return s.signerAddress
	}
	return s.wallet.AddressString()
}

// Testnet returns true if using testnet.
func (s *SDK) Testnet() bool {
	return s.config.Testnet
}

// Chain returns the chain identifier ("Mainnet" or "Testnet").
func (s *SDK) Chain() string {
	return s.chain
}

// ═══════════════════════════════════════════════════════════════════════════════
// INFO CLIENT
// ═══════════════════════════════════════════════════════════════════════════════

// Info returns the Info API client.
func (s *SDK) Info() *InfoClient {
	s.infoOnce.Do(func() {
		if s.endpoint == "" {
			panic("endpoint required for Info API")
		}
		s.info = NewInfoClient(s.endpoint, s.http)
	})
	return s.info
}

// ═══════════════════════════════════════════════════════════════════════════════
// HYPERCORE CLIENT
// ═══════════════════════════════════════════════════════════════════════════════

// Core returns the HyperCore API client.
func (s *SDK) Core() *HyperCoreClient {
	s.coreOnce.Do(func() {
		if s.endpoint == "" {
			panic("endpoint required for HyperCore API")
		}
		s.core = NewHyperCoreClient(s.endpoint, s.http)
	})
	return s.core
}

// ═══════════════════════════════════════════════════════════════════════════════
// EVM CLIENT
// ═══════════════════════════════════════════════════════════════════════════════

// EVM returns the EVM API client.
func (s *SDK) EVM() *EVMClient {
	s.evmOnce.Do(func() {
		if s.endpoint == "" {
			panic("endpoint required for EVM API")
		}
		s.evm = NewEVMClient(s.endpoint, s.http)
	})
	return s.evm
}

// ═══════════════════════════════════════════════════════════════════════════════
// STREAMING CLIENTS
// ═══════════════════════════════════════════════════════════════════════════════

// NewStream creates a new WebSocket streaming client.
// The stream client is NOT shared across invocations - each call creates a new client.
func (s *SDK) NewStream(config *StreamConfig) *Stream {
	if s.endpoint == "" {
		panic("endpoint required for WebSocket streaming")
	}
	return NewStream(s.endpoint, config)
}

// NewGRPCStream creates a new gRPC streaming client.
// The stream client is NOT shared across invocations - each call creates a new client.
func (s *SDK) NewGRPCStream(config *GRPCStreamConfig) *GRPCStream {
	if s.endpoint == "" {
		panic("endpoint required for gRPC streaming")
	}
	return NewGRPCStream(s.endpoint, config)
}

// NewEVMStream creates a new EVM WebSocket streaming client.
// The stream client is NOT shared across invocations - each call creates a new client.
func (s *SDK) NewEVMStream(config *EVMStreamConfig) *EVMStream {
	if s.endpoint == "" {
		panic("endpoint required for EVM WebSocket streaming")
	}
	return NewEVMStream(s.endpoint, config)
}

// ═══════════════════════════════════════════════════════════════════════════════
// ORDER PLACEMENT
// ═══════════════════════════════════════════════════════════════════════════════

// OrderOption is an option for order placement.
type OrderOption func(*orderParams)

type orderParams struct {
	size         string
	notional     float64
	price        string
	tif          TIF
	reduceOnly   bool
	grouping     OrderGrouping
	slippage     *float64
	priorityFee  *uint64
	vaultAddress string
	expiresAfter *int64
}

// WithSize sets the order size in asset units.
func WithSize(size any) OrderOption {
	return func(p *orderParams) {
		p.size = NewDecimal(size).String()
	}
}

// WithNotional sets the order size by USD notional value.
func WithNotional(usd float64) OrderOption {
	return func(p *orderParams) {
		p.notional = usd
	}
}

// WithPrice sets the limit price.
func WithPrice(price any) OrderOption {
	return func(p *orderParams) {
		p.price = NewDecimal(price).String()
	}
}

// WithTIF sets the time in force.
func WithTIF(tif TIF) OrderOption {
	return func(p *orderParams) {
		p.tif = tif
	}
}

// WithReduceOnly marks the order as reduce-only.
func WithReduceOnly() OrderOption {
	return func(p *orderParams) {
		p.reduceOnly = true
	}
}

// WithGrouping sets the order grouping for TP/SL attachment.
func WithGrouping(grouping OrderGrouping) OrderOption {
	return func(p *orderParams) {
		p.grouping = grouping
	}
}

// WithOrderSlippage overrides the default slippage for this order.
// Slippage is expressed as a fraction (e.g. 0.05 = 5%).
// Only applies to market orders; ignored for limit orders.
func WithOrderSlippage(slippage float64) OrderOption {
	return func(p *orderParams) {
		p.slippage = &slippage
	}
}

// WithPriorityFee sets Hyperliquid order priority p.
// p=10000 is 1 bp and is paid from undelegated staking HYPE.
func WithPriorityFee(priorityFee uint64) OrderOption {
	return func(p *orderParams) {
		p.priorityFee = &priorityFee
	}
}

// WithVaultAddress places the order on behalf of a vault (or subaccount).
// The address is sent as the top-level vaultAddress exchange field and is
// folded into the signed action hash by the build endpoint.
func WithVaultAddress(vaultAddress string) OrderOption {
	return func(p *orderParams) {
		p.vaultAddress = vaultAddress
	}
}

// WithExpiresAfter sets the action TTL as a millisecond timestamp after which
// the order is rejected by the exchange. It is sent as the top-level
// expiresAfter exchange field and is folded into the signed action hash by
// the build endpoint.
func WithExpiresAfter(expiresAfterMs int64) OrderOption {
	return func(p *orderParams) {
		p.expiresAfter = &expiresAfterMs
	}
}

// Buy places a buy order.
func (s *SDK) Buy(asset any, opts ...OrderOption) (*PlacedOrder, error) {
	return s.placeOrder(asset, SideBuy, opts...)
}

// Sell places a sell order.
func (s *SDK) Sell(asset any, opts ...OrderOption) (*PlacedOrder, error) {
	return s.placeOrder(asset, SideSell, opts...)
}

// Long is an alias for Buy (perps terminology).
func (s *SDK) Long(asset any, opts ...OrderOption) (*PlacedOrder, error) {
	return s.Buy(asset, opts...)
}

// Short is an alias for Sell (perps terminology).
func (s *SDK) Short(asset any, opts ...OrderOption) (*PlacedOrder, error) {
	return s.Sell(asset, opts...)
}

// MarketBuy places a market buy order.
func (s *SDK) MarketBuy(asset any, opts ...OrderOption) (*PlacedOrder, error) {
	opts = append(opts, WithTIF(TIFMarket))
	return s.Buy(asset, opts...)
}

// MarketSell places a market sell order.
func (s *SDK) MarketSell(asset any, opts ...OrderOption) (*PlacedOrder, error) {
	opts = append(opts, WithTIF(TIFMarket))
	return s.Sell(asset, opts...)
}

// PlaceOrder places an order using the fluent OrderBuilder.
func (s *SDK) PlaceOrder(order *OrderBuilder) (*PlacedOrder, error) {
	if err := order.Validate(); err != nil {
		return nil, err
	}

	// Handle notional orders
	if order.GetNotional() > 0 && order.GetSize() == "" {
		mid, err := s.GetMid(order.Asset())
		if err != nil || mid == 0 {
			return nil, ValidationError(fmt.Sprintf("could not fetch price for %s", order.Asset()))
		}
		size := order.GetNotional() / mid
		szDecimals := s.getSizeDecimals(order.Asset())
		if isPredictionAsset(order.Asset()) {
			size = math.Ceil(size)
		} else {
			size = math.Round(size*math.Pow(10, float64(szDecimals))) / math.Pow(10, float64(szDecimals))
		}
		order.SetSize(NewDecimal(size).String())
	}

	return s.executeOrder(order, OrderGroupingNA, &exchangeParams{
		slippage:     order.GetSlippage(),
		priorityFee:  order.GetPriorityFee(),
		vaultAddress: order.GetVaultAddress(),
		expiresAfter: order.GetExpiresAfter(),
	})
}

func (s *SDK) placeOrder(assetInput any, side Side, opts ...OrderOption) (*PlacedOrder, error) {
	asset := assetName(assetInput)
	params := &orderParams{
		tif:      TIFIOC,
		grouping: OrderGroupingNA,
	}
	for _, opt := range opts {
		opt(params)
	}

	order := Order()
	if side == SideBuy {
		order.Buy(asset)
	} else {
		order.Sell(asset)
	}

	// Handle size
	if params.notional > 0 {
		mid, err := s.GetMid(asset)
		if err != nil || mid == 0 {
			return nil, ValidationError(fmt.Sprintf("could not fetch price for %s", asset))
		}
		szDecimals := s.getSizeDecimals(asset)
		size := params.notional / mid
		if isPredictionAsset(asset) {
			size = math.Ceil(size)
		} else {
			size = math.Round(size*math.Pow(10, float64(szDecimals))) / math.Pow(10, float64(szDecimals))
		}
		order.Size(size)
	} else if params.size != "" {
		order.size = params.size
	} else {
		return nil, ValidationError("either size or notional is required")
	}

	order.tif = params.tif
	if params.tif != TIFMarket && params.price != "" {
		order.price = params.price
	}
	order.reduceOnly = params.reduceOnly

	return s.executeOrder(order, params.grouping, &exchangeParams{
		slippage:     params.slippage,
		priorityFee:  params.priorityFee,
		vaultAddress: params.vaultAddress,
		expiresAfter: params.expiresAfter,
	})
}

func (s *SDK) executeOrder(order *OrderBuilder, grouping OrderGrouping, params *exchangeParams) (*PlacedOrder, error) {
	action := order.ToAction()
	if grouping != OrderGroupingNA && params.priorityFee != nil {
		return nil, ValidationError("priorityFee cannot be combined with TP/SL grouping").
			WithGuidance("Use priorityFee on standalone IOC/market orders, or omit grouping.")
	}
	if err := s.validatePredictionOrder(order.Asset(), order.GetSize(), order.GetPrice(), order.GetTIF() == TIFMarket, order.GetSide() == SideBuy, params.priorityFee); err != nil {
		return nil, err
	}
	if grouping != OrderGroupingNA {
		action["grouping"] = string(grouping)
	}

	result, err := s.buildSignSendParams(action, params)
	if err != nil {
		return nil, err
	}

	exchangeResp, _ := result["exchangeResponse"].(map[string]any)
	return ParsePlacedOrder(exchangeResp, order.Asset(), order.GetSide(), order.GetSize(), order.GetPrice(), s), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// TRIGGER ORDERS (Stop Loss / Take Profit)
// ═══════════════════════════════════════════════════════════════════════════════

// StopLoss places a stop-loss trigger order.
func (s *SDK) StopLoss(asset string, size any, triggerPrice any, opts ...TriggerOrderOption) (*PlacedOrder, error) {
	order := TriggerOrder().StopLoss(asset).Size(size).TriggerPrice(triggerPrice).ReduceOnly()
	for _, opt := range opts {
		opt(order)
	}
	return s.executeTriggerOrder(order, OrderGroupingNA)
}

// TakeProfit places a take-profit trigger order.
func (s *SDK) TakeProfit(asset string, size any, triggerPrice any, opts ...TriggerOrderOption) (*PlacedOrder, error) {
	order := TriggerOrder().TakeProfit(asset).Size(size).TriggerPrice(triggerPrice).ReduceOnly()
	for _, opt := range opts {
		opt(order)
	}
	return s.executeTriggerOrder(order, OrderGroupingNA)
}

// SL is an alias for StopLoss.
func (s *SDK) SL(asset string, size any, triggerPrice any, opts ...TriggerOrderOption) (*PlacedOrder, error) {
	return s.StopLoss(asset, size, triggerPrice, opts...)
}

// TP is an alias for TakeProfit.
func (s *SDK) TP(asset string, size any, triggerPrice any, opts ...TriggerOrderOption) (*PlacedOrder, error) {
	return s.TakeProfit(asset, size, triggerPrice, opts...)
}

// TriggerOrderOption is an option for trigger order placement.
type TriggerOrderOption func(*TriggerOrderBuilder)

// TriggerWithLimitPrice sets the limit price for trigger order execution.
func TriggerWithLimitPrice(price any) TriggerOrderOption {
	return func(t *TriggerOrderBuilder) {
		t.Limit(price)
	}
}

// TriggerWithSide sets the side for trigger order execution.
func TriggerWithSide(side Side) TriggerOrderOption {
	return func(t *TriggerOrderBuilder) {
		t.Side(side)
	}
}

// TriggerWithGrouping sets the grouping for trigger order.
func TriggerWithGrouping(grouping OrderGrouping) TriggerOrderOption {
	return func(t *TriggerOrderBuilder) {
		// Grouping is passed to executeTriggerOrder
	}
}

// TriggerWithVaultAddress places the trigger order on behalf of a vault
// (or subaccount). See WithVaultAddress.
func TriggerWithVaultAddress(vaultAddress string) TriggerOrderOption {
	return func(t *TriggerOrderBuilder) {
		t.VaultAddress(vaultAddress)
	}
}

// TriggerWithExpiresAfter sets the action TTL (ms timestamp) for the trigger
// order. See WithExpiresAfter.
func TriggerWithExpiresAfter(expiresAfterMs int64) TriggerOrderOption {
	return func(t *TriggerOrderBuilder) {
		t.ExpiresAfter(expiresAfterMs)
	}
}

// PlaceTriggerOrder places a trigger order using the fluent TriggerOrderBuilder.
func (s *SDK) PlaceTriggerOrder(order *TriggerOrderBuilder, grouping OrderGrouping) (*PlacedOrder, error) {
	return s.executeTriggerOrder(order, grouping)
}

func (s *SDK) executeTriggerOrder(order *TriggerOrderBuilder, grouping OrderGrouping) (*PlacedOrder, error) {
	if err := order.Validate(); err != nil {
		return nil, err
	}

	action := order.ToAction(grouping)
	result, err := s.buildSignSendParams(action, &exchangeParams{
		vaultAddress: order.GetVaultAddress(),
		expiresAfter: order.GetExpiresAfter(),
	})
	if err != nil {
		return nil, err
	}

	exchangeResp, _ := result["exchangeResponse"].(map[string]any)
	return ParsePlacedOrder(exchangeResp, order.Asset(), order.GetSide(), order.GetSize(), order.GetLimitPrice(), s), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// ORDER MANAGEMENT
// ═══════════════════════════════════════════════════════════════════════════════

type cancelParams struct {
	fast         bool
	vaultAddress string
	expiresAfter *int64
}

// CancelOption is an option for order cancellation.
type CancelOption func(*cancelParams)

// CancelWithFast requests fast cancellation: the cancel action carries the
// "f": true flag (only emitted when set, matching backend semantics).
func CancelWithFast() CancelOption {
	return func(p *cancelParams) {
		p.fast = true
	}
}

// CancelWithVaultAddress cancels on behalf of a vault (or subaccount).
// See WithVaultAddress.
func CancelWithVaultAddress(vaultAddress string) CancelOption {
	return func(p *cancelParams) {
		p.vaultAddress = vaultAddress
	}
}

// CancelWithExpiresAfter sets the action TTL (ms timestamp) for the cancel.
// See WithExpiresAfter.
func CancelWithExpiresAfter(expiresAfterMs int64) CancelOption {
	return func(p *cancelParams) {
		p.expiresAfter = &expiresAfterMs
	}
}

func applyCancelOptions(opts []CancelOption) *cancelParams {
	params := &cancelParams{}
	for _, opt := range opts {
		opt(params)
	}
	return params
}

// Cancel cancels an order by OID.
func (s *SDK) Cancel(oid int64, asset string, opts ...CancelOption) (map[string]any, error) {
	if oid <= 0 {
		return nil, ValidationError(fmt.Sprintf("oid must be a positive integer, got: %d", oid))
	}

	params := applyCancelOptions(opts)

	assetIdx := 0
	if asset != "" {
		idx, err := s.resolveAssetIndex(asset)
		if err == nil {
			assetIdx = idx
		}
	}

	action := map[string]any{
		"type": "cancel",
		"cancels": []any{
			map[string]any{"a": assetIdx, "o": oid},
		},
	}
	if params.fast {
		action["f"] = true
	}

	return s.buildSignSendParams(action, &exchangeParams{
		vaultAddress: params.vaultAddress,
		expiresAfter: params.expiresAfter,
	})
}

// CancelAll cancels all open orders, optionally filtered by asset.
func (s *SDK) CancelAll(asset string, opts ...CancelOption) (map[string]any, error) {
	params := applyCancelOptions(opts)

	// When cancelling on behalf of a vault, enumerate the vault's open orders,
	// not the wallet's.
	orders, err := s.OpenOrders(params.vaultAddress)
	if err != nil {
		return nil, err
	}

	ordersList, ok := orders["orders"].([]any)
	if !ok || len(ordersList) == 0 {
		return map[string]any{"message": "No orders to cancel"}, nil
	}

	cancelActions, ok := orders["cancelActions"].(map[string]any)
	if !ok {
		return map[string]any{"message": "No orders to cancel"}, nil
	}

	var cancelAction map[string]any
	if asset != "" {
		byAsset, ok := cancelActions["byAsset"].(map[string]any)
		if !ok {
			return map[string]any{"message": fmt.Sprintf("No %s orders to cancel", asset)}, nil
		}
		cancelAction, ok = byAsset[asset].(map[string]any)
		if !ok {
			return map[string]any{"message": fmt.Sprintf("No %s orders to cancel", asset)}, nil
		}
	} else {
		cancelAction, ok = cancelActions["all"].(map[string]any)
		if !ok {
			return map[string]any{"message": "No orders to cancel"}, nil
		}
	}

	if params.fast {
		cancelAction["f"] = true
	}

	return s.buildSignSendParams(cancelAction, &exchangeParams{
		vaultAddress: params.vaultAddress,
		expiresAfter: params.expiresAfter,
	})
}

// CancelByCloid cancels an order by client order ID.
func (s *SDK) CancelByCloid(cloid string, asset string, opts ...CancelOption) (map[string]any, error) {
	params := applyCancelOptions(opts)

	assetIdx, err := s.resolveAssetIndex(asset)
	if err != nil {
		return nil, err
	}

	action := map[string]any{
		"type": "cancelByCloid",
		"cancels": []any{
			map[string]any{"asset": assetIdx, "cloid": cloid},
		},
	}
	if params.fast {
		action["f"] = true
	}

	return s.buildSignSendParams(action, &exchangeParams{
		vaultAddress: params.vaultAddress,
		expiresAfter: params.expiresAfter,
	})
}

// ScheduleCancel schedules cancellation of all orders after a delay.
// Pass 0 to cancel the scheduled cancel.
func (s *SDK) ScheduleCancel(timeMs int64) (map[string]any, error) {
	action := map[string]any{"type": "scheduleCancel"}
	if timeMs > 0 {
		action["time"] = timeMs
	}
	return s.buildSignSend(action, nil)
}

// Modify modifies an existing order.
func (s *SDK) Modify(oid int64, asset, side, price, size string, opts ...ModifyOption) (*PlacedOrder, error) {
	if oid <= 0 {
		return nil, ValidationError(fmt.Sprintf("oid must be a positive integer, got: %d", oid))
	}
	if asset == "" {
		return nil, ValidationError("asset must be a non-empty string")
	}
	if side != "buy" && side != "sell" {
		return nil, ValidationError(fmt.Sprintf("side must be 'buy' or 'sell', got: %s", side))
	}

	params := &modifyParams{
		tif: TIFGTC,
	}
	for _, opt := range opts {
		opt(params)
	}

	isBuy := side == "buy"

	// Capitalize TIF for API
	tifStr := strings.Title(string(params.tif))

	action := map[string]any{
		"type": "batchModify",
		"modifies": []any{
			map[string]any{
				"oid": oid,
				"order": map[string]any{
					"a": asset,
					"b": isBuy,
					"p": price,
					"s": size,
					"r": params.reduceOnly,
					"t": map[string]any{
						"limit": map[string]any{"tif": tifStr},
					},
				},
			},
		},
	}

	result, err := s.buildSignSendParams(action, &exchangeParams{
		vaultAddress: params.vaultAddress,
		expiresAfter: params.expiresAfter,
	})
	if err != nil {
		return nil, err
	}

	exchangeResp, _ := result["exchangeResponse"].(map[string]any)
	orderSide := SideBuy
	if side == "sell" {
		orderSide = SideSell
	}
	return ParsePlacedOrder(exchangeResp, asset, orderSide, size, price, s), nil
}

type modifyParams struct {
	tif          TIF
	reduceOnly   bool
	vaultAddress string
	expiresAfter *int64
}

// ModifyOption is an option for order modification.
type ModifyOption func(*modifyParams)

// ModifyWithTIF sets the time in force for the modified order.
func ModifyWithTIF(tif TIF) ModifyOption {
	return func(p *modifyParams) {
		p.tif = tif
	}
}

// ModifyWithReduceOnly sets the reduce-only flag for the modified order.
func ModifyWithReduceOnly() ModifyOption {
	return func(p *modifyParams) {
		p.reduceOnly = true
	}
}

// ModifyWithVaultAddress modifies on behalf of a vault (or subaccount).
// See WithVaultAddress.
func ModifyWithVaultAddress(vaultAddress string) ModifyOption {
	return func(p *modifyParams) {
		p.vaultAddress = vaultAddress
	}
}

// ModifyWithExpiresAfter sets the action TTL (ms timestamp) for the modify.
// See WithExpiresAfter.
func ModifyWithExpiresAfter(expiresAfterMs int64) ModifyOption {
	return func(p *modifyParams) {
		p.expiresAfter = &expiresAfterMs
	}
}

// CloseOption is an option for closing a position.
type CloseOption func(*closeParams)

type closeParams struct {
	slippage     *float64
	vaultAddress string
	expiresAfter *int64
}

// CloseWithSlippage overrides the default slippage for this close.
// Slippage is expressed as a fraction (e.g. 0.05 = 5%).
func CloseWithSlippage(slippage float64) CloseOption {
	return func(p *closeParams) {
		p.slippage = &slippage
	}
}

// CloseWithVaultAddress closes a vault (or subaccount) position.
// See WithVaultAddress.
func CloseWithVaultAddress(vaultAddress string) CloseOption {
	return func(p *closeParams) {
		p.vaultAddress = vaultAddress
	}
}

// CloseWithExpiresAfter sets the action TTL (ms timestamp) for the close.
// See WithExpiresAfter.
func CloseWithExpiresAfter(expiresAfterMs int64) CloseOption {
	return func(p *closeParams) {
		p.expiresAfter = &expiresAfterMs
	}
}

// ClosePosition closes an open position completely.
func (s *SDK) ClosePosition(asset string, opts ...CloseOption) (*PlacedOrder, error) {
	s.requireWallet()

	params := &closeParams{}
	for _, opt := range opts {
		opt(params)
	}

	// The backend queries action.user's position to size the close; when
	// trading on behalf of a vault, the position belongs to the vault.
	user := s.Address()
	if params.vaultAddress != "" {
		user = params.vaultAddress
	}

	action := map[string]any{
		"type":  "closePosition",
		"asset": asset,
		"user":  user,
	}

	result, err := s.buildSignSendParams(action, &exchangeParams{
		slippage:     params.slippage,
		vaultAddress: params.vaultAddress,
		expiresAfter: params.expiresAfter,
	})
	if err != nil {
		return nil, err
	}

	exchangeResp, _ := result["exchangeResponse"].(map[string]any)
	return ParsePlacedOrder(exchangeResp, asset, SideSell, "0", "", s), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// QUERIES
// ═══════════════════════════════════════════════════════════════════════════════

// OpenOrdersOption is an option for OpenOrders.
type OpenOrdersOption func(*openOrdersParams)

type openOrdersParams struct {
	dex string
}

// OpenOrdersWithDex filters open orders by HIP-3 DEX name.
func OpenOrdersWithDex(dex string) OpenOrdersOption {
	return func(p *openOrdersParams) {
		p.dex = dex
	}
}

// OpenOrders returns open orders with enriched info and pre-built cancel actions.
func (s *SDK) OpenOrders(user string, opts ...OpenOrdersOption) (map[string]any, error) {
	if user == "" {
		s.requireWallet()
		user = s.Address()
	}

	params := &openOrdersParams{}
	for _, opt := range opts {
		opt(params)
	}

	body := map[string]any{"user": user}
	if params.dex != "" {
		body["dex"] = params.dex
	}

	ctx := context.Background()
	return s.http.Post(ctx, s.publicWorkerURL+"/openOrders", body)
}

// OrderStatus returns detailed status for an order.
func (s *SDK) OrderStatus(oid int64, user string) (map[string]any, error) {
	if user == "" {
		s.requireWallet()
		user = s.Address()
	}
	ctx := context.Background()
	return s.http.Post(ctx, s.publicWorkerURL+"/orderStatus", map[string]any{"user": user, "oid": oid})
}

// Markets returns all available markets.
func (s *SDK) Markets() (*Markets, error) {
	ctx := context.Background()
	result, err := s.http.Get(ctx, s.publicWorkerURL+"/markets", nil)
	if err != nil {
		return nil, err
	}

	markets := &Markets{
		Perps: []Market{},
		Spot:  []Market{},
		HIP3:  make(map[string][]Market),
		HIP4:  []Market{},
	}

	if perps, ok := result["perps"].([]any); ok {
		for _, p := range perps {
			if m, ok := p.(map[string]any); ok {
				markets.Perps = append(markets.Perps, Market{
					Name:       m["name"].(string),
					Index:      int(m["index"].(float64)),
					SzDecimals: int(m["szDecimals"].(float64)),
				})
			}
		}
	}

	if spot, ok := result["spot"].([]any); ok {
		for _, s := range spot {
			if m, ok := s.(map[string]any); ok {
				markets.Spot = append(markets.Spot, Market{
					Name:       m["name"].(string),
					Index:      int(m["index"].(float64)),
					SzDecimals: int(m["szDecimals"].(float64)),
					IsSpot:     true,
				})
			}
		}
	}

	if hip3, ok := result["hip3"].(map[string]any); ok {
		for dex, dexMarkets := range hip3 {
			if dms, ok := dexMarkets.([]any); ok {
				var marketList []Market
				for _, dm := range dms {
					if m, ok := dm.(map[string]any); ok {
						marketList = append(marketList, Market{
							Name:       m["name"].(string),
							Index:      int(m["index"].(float64)),
							SzDecimals: int(m["szDecimals"].(float64)),
							Dex:        dex,
						})
					}
				}
				markets.HIP3[dex] = marketList
			}
		}
	}

	if hip4, ok := result["hip4"].([]any); ok {
		for _, h := range hip4 {
			if m, ok := h.(map[string]any); ok {
				markets.HIP4 = append(markets.HIP4, Market{
					Name:       m["name"].(string),
					Index:      int(m["index"].(float64)),
					SzDecimals: int(m["szDecimals"].(float64)),
				})
			}
		}
	}

	return markets, nil
}

// Dexes returns all HIP-3 DEXes.
func (s *SDK) Dexes() (map[string]any, error) {
	ctx := context.Background()
	return s.http.Get(ctx, s.publicWorkerURL+"/dexes", nil)
}

// Preflight validates an order before signing.
func (s *SDK) Preflight(asset string, side Side, price, size any, opts ...PreflightOption) (map[string]any, error) {
	params := &preflightParams{
		tif: TIFGTC,
	}
	for _, opt := range opts {
		opt(params)
	}

	// Capitalize TIF for API
	tifStr := strings.Title(string(params.tif))

	order := map[string]any{
		"a": asset,
		"b": side == SideBuy,
		"p": NewDecimal(price).String(),
		"s": NewDecimal(size).String(),
		"r": params.reduceOnly,
		"t": map[string]any{
			"limit": map[string]any{"tif": tifStr},
		},
	}

	ctx := context.Background()
	return s.http.Post(ctx, s.publicWorkerURL+"/preflight", map[string]any{
		"action": map[string]any{"type": "order", "orders": []any{order}},
	})
}

type preflightParams struct {
	tif        TIF
	reduceOnly bool
}

// PreflightOption is an option for preflight validation.
type PreflightOption func(*preflightParams)

// PreflightWithTIF sets the time in force for preflight.
func PreflightWithTIF(tif TIF) PreflightOption {
	return func(p *preflightParams) {
		p.tif = tif
	}
}

// PreflightWithReduceOnly sets the reduce-only flag for preflight.
func PreflightWithReduceOnly() PreflightOption {
	return func(p *preflightParams) {
		p.reduceOnly = true
	}
}

// ApprovalStatus checks builder fee approval status.
func (s *SDK) ApprovalStatus(user string) (map[string]any, error) {
	if user == "" {
		s.requireWallet()
		user = s.Address()
	}
	ctx := context.Background()
	return s.http.Get(ctx, s.publicWorkerURL+"/approval", map[string]string{"user": user})
}

// GetMid returns the current mid price for an asset.
func (s *SDK) GetMid(assetInput any) (float64, error) {
	asset := assetName(assetInput)
	if isPredictionAsset(asset) {
		asset = predictionSymbol(asset)
	}
	ctx := context.Background()

	body := map[string]any{"type": "allMids"}
	if strings.Contains(asset, ":") {
		dex := strings.Split(asset, ":")[0]
		body["dex"] = dex
	}

	result, err := s.postInfo(ctx, body)
	if err != nil {
		return 0, err
	}

	if m, ok := result.(map[string]any); ok {
		if midStr, ok := m[asset].(string); ok {
			return NewDecimal(midStr).Float64(), nil
		}
	}

	return 0, nil
}

// RefreshMarkets forces refresh of market metadata cache.
func (s *SDK) RefreshMarkets() (*Markets, error) {
	markets, err := s.Markets()
	if err != nil {
		return nil, err
	}

	s.cacheMu.Lock()
	s.marketsCache = markets
	s.marketsCacheTime = time.Now()
	s.cacheMu.Unlock()

	return markets, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// INTERNAL METHODS
// ═══════════════════════════════════════════════════════════════════════════════

func (s *SDK) requireWallet() {
	if s.wallet == nil && s.signer == nil {
		panic("private key or signer required for this operation")
	}
}

// validateSignerSignature checks that an external Signer returned a usable
// signature: non-nil, with non-empty r/s, and v ∈ {27,28} (the documented
// contract, matching Wallet.SignHash). It exists because the callback is
// user-supplied and may return a nil or malformed value without erroring.
func validateSignerSignature(sig *Signature) error {
	if sig == nil {
		return fmt.Errorf("signer returned a nil signature")
	}
	if sig.R == "" || sig.S == "" {
		return fmt.Errorf("signer returned a signature with empty r or s")
	}
	if sig.V != 27 && sig.V != 28 {
		return fmt.Errorf("signer returned invalid v=%d (must be 27 or 28)", sig.V)
	}
	return nil
}

// exchangeParams carries the optional top-level exchange-request fields
// (slippage, priorityFee, vaultAddress, expiresAfter). vaultAddress and
// expiresAfter participate in the action hash the build endpoint returns,
// so they must be present in the build payload AND repeated verbatim in the
// send payload for signature recovery to succeed. They are only emitted when
// set, keeping the wire format unchanged for callers that don't use them.
type exchangeParams struct {
	slippage     *float64
	priorityFee  *uint64
	vaultAddress string
	expiresAfter *int64
}

func (s *SDK) buildSignSend(action map[string]any, slippage *float64, priorityFee ...uint64) (map[string]any, error) {
	params := &exchangeParams{slippage: slippage}
	if len(priorityFee) > 0 {
		params.priorityFee = &priorityFee[0]
	}
	return s.buildSignSendParams(action, params)
}

func (s *SDK) buildSignSendParams(action map[string]any, params *exchangeParams) (map[string]any, error) {
	s.requireWallet()

	// Each HTTP call below relies on the client's own per-request timeout
	// (http.Client.Timeout), unchanged from the wallet-only path. The external
	// signer gets its own scoped deadline at Step 2 so a slow remote signer is
	// bounded without shrinking the build/send HTTP budgets.
	ctx := context.Background()

	// Step 1: Build
	buildPayload := map[string]any{"action": action}
	effectiveSlippage := s.config.Slippage
	if params.slippage != nil {
		effectiveSlippage = *params.slippage
	}
	if effectiveSlippage > 0 {
		buildPayload["slippage"] = effectiveSlippage
	}
	if params.priorityFee != nil {
		buildPayload["priorityFee"] = *params.priorityFee
	}
	if params.vaultAddress != "" {
		buildPayload["vaultAddress"] = params.vaultAddress
	}
	if params.expiresAfter != nil {
		buildPayload["expiresAfter"] = *params.expiresAfter
	}
	buildResult, err := s.http.Post(ctx, s.exchangeURL, buildPayload)
	if err != nil {
		return nil, err
	}

	hash, ok := buildResult["hash"].(string)
	if !ok {
		return nil, BuildError("build response missing hash").WithRaw(buildResult)
	}

	nonce, ok := buildResult["nonce"].(float64)
	if !ok {
		return nil, BuildError("build response missing nonce").WithRaw(buildResult)
	}

	// Step 2: Sign
	// Prefer the external signer when configured; otherwise sign in-process
	// with the wallet's private key. An external signer failure is reported as
	// SignerError (the caller's KMS/HSM/service erred), distinct from a venue
	// rejection or an in-process wallet fault (SignatureError).
	var sig *Signature
	if s.signer != nil {
		// Bound only the external signer call by the configured Timeout, so a
		// slow remote signer is cancelled. A non-positive Timeout means "no
		// deadline". The build/send HTTP calls keep their own per-request
		// timeout, so existing behaviour is unchanged for them.
		signCtx := ctx
		if s.config.Timeout > 0 {
			var cancel context.CancelFunc
			signCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
			defer cancel()
		}
		sig, err = s.signer(signCtx, hash)
		if err != nil {
			return nil, SignerError(err)
		}
		// The signer is user-supplied and has no compile-time guarantee of
		// returning a valid *Signature (unlike Wallet.SignHash). Validate it
		// here so a nil/malformed result surfaces as a clear SignerError rather
		// than a confusing venue rejection downstream.
		if verr := validateSignerSignature(sig); verr != nil {
			return nil, SignerError(verr)
		}
	} else {
		sig, err = s.wallet.SignHash(hash)
		if err != nil {
			return nil, SignatureError(fmt.Sprintf("failed to sign: %v", err))
		}
	}

	// Get action from build response (may have been normalized)
	finalAction := action
	if builtAction, ok := buildResult["action"].(map[string]any); ok {
		finalAction = builtAction
	}

	// Step 3: Send
	// vaultAddress/expiresAfter were folded into the hash at build time, so
	// they must be repeated here for the worker to recover the signer.
	sendPayload := map[string]any{
		"action":    finalAction,
		"nonce":     int64(nonce),
		"signature": sig,
	}
	if params.vaultAddress != "" {
		sendPayload["vaultAddress"] = params.vaultAddress
	}
	if params.expiresAfter != nil {
		sendPayload["expiresAfter"] = *params.expiresAfter
	}

	return s.http.Post(ctx, s.exchangeURL, sendPayload)
}

func (s *SDK) postInfo(ctx context.Context, body map[string]any) (any, error) {
	reqType, _ := body["type"].(string)

	var url string
	if QNSupportedInfoMethods[reqType] {
		url = s.infoURL
	} else {
		url = s.publicWorkerURL + "/info"
	}

	return s.http.PostRaw(ctx, url, body)
}

func (s *SDK) ensureApproved(maxFee string) error {
	status, err := s.ApprovalStatus("")
	if err != nil {
		return err
	}

	if approved, ok := status["approved"].(bool); !ok || !approved {
		_, err = s.ApproveBuilderFee(maxFee, "")
		return err
	}

	return nil
}

func (s *SDK) getSizeDecimals(asset string) int {
	if isPredictionAsset(asset) {
		s.cacheMu.Lock()
		s.szDecimalsCache[asset] = 0
		s.cacheMu.Unlock()
		return 0
	}

	s.cacheMu.RLock()
	if dec, ok := s.szDecimalsCache[asset]; ok {
		s.cacheMu.RUnlock()
		return dec
	}
	s.cacheMu.RUnlock()

	// Fetch markets if cache is stale
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	if s.marketsCache == nil || time.Since(s.marketsCacheTime) > CacheTTL {
		markets, err := s.Markets()
		if err != nil {
			return 5 // Default
		}
		s.marketsCache = markets
		s.marketsCacheTime = time.Now()
	}

	// Search perps
	for _, m := range s.marketsCache.Perps {
		if m.Name == asset {
			s.szDecimalsCache[asset] = m.SzDecimals
			return m.SzDecimals
		}
	}

	// Search spot
	for _, m := range s.marketsCache.Spot {
		if m.Name == asset {
			s.szDecimalsCache[asset] = m.SzDecimals
			return m.SzDecimals
		}
	}

	// Search HIP-3
	for _, dexMarkets := range s.marketsCache.HIP3 {
		for _, m := range dexMarkets {
			if m.Name == asset {
				s.szDecimalsCache[asset] = m.SzDecimals
				return m.SzDecimals
			}
		}
	}

	// Search HIP-4
	for _, m := range s.marketsCache.HIP4 {
		if m.Name == asset {
			s.szDecimalsCache[asset] = m.SzDecimals
			return m.SzDecimals
		}
	}

	return 5 // Default
}

func (s *SDK) resolveAssetIndex(asset string) (int, error) {
	if len(asset) > 1 && (asset[0] == '#' || asset[0] == '+') && allDecimalDigits(asset[1:]) {
		return 100000000 + int(NewDecimal(asset[1:]).Float64()), nil
	}
	if allDecimalDigits(asset) {
		return int(NewDecimal(asset).Float64()), nil
	}

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	if s.marketsCache == nil || time.Since(s.marketsCacheTime) > CacheTTL {
		markets, err := s.Markets()
		if err != nil {
			return 0, err
		}
		s.marketsCache = markets
		s.marketsCacheTime = time.Now()
	}

	// Search perps
	for i, m := range s.marketsCache.Perps {
		if m.Name == asset {
			return i, nil
		}
	}

	// Search spot (index = 10000 + position)
	for i, m := range s.marketsCache.Spot {
		if m.Name == asset {
			return 10000 + i, nil
		}
	}

	// Search HIP-3
	for _, dexMarkets := range s.marketsCache.HIP3 {
		for _, m := range dexMarkets {
			if m.Name == asset {
				return m.Index, nil
			}
		}
	}

	// Search HIP-4
	for _, m := range s.marketsCache.HIP4 {
		if m.Name == asset {
			return m.Index, nil
		}
	}

	return 0, ValidationError(fmt.Sprintf("could not resolve asset '%s' to index", asset))
}

func (s *SDK) validatePredictionOrder(asset, size, price string, isMarket bool, isBuy bool, priorityFee *uint64) error {
	if !isPredictionAsset(asset) {
		return nil
	}

	if priorityFee != nil {
		return ValidationError("priorityFee is not supported for HIP-4 prediction markets").
			WithGuidance("Omit priorityFee when trading market.Yes, market.No, or # markets.")
	}

	if size == "" {
		return nil
	}

	sizeValue := NewDecimal(size).Float64()
	if math.Trunc(sizeValue) != sizeValue {
		return ValidationError("HIP-4 prediction market size must be a whole number of contracts").
			WithGuidance("Use WithSize(20), not WithSize(20.5).")
	}

	px := NewDecimal(price).Float64()
	if px == 0 && isMarket {
		mid, err := s.GetMid(predictionSymbol(asset))
		if err == nil {
			px = mid
		}
	}

	if isBuy && px > 0 && sizeValue*px < 10 {
		return ValidationError("HIP-4 prediction market orders must have minimum value of 10 USDH").
			WithGuidance("Increase size or price, or call sdk.BuyUSDH(...) before trading.")
	}

	return nil
}
