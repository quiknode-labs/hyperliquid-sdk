package hyperliquid

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// GRPCStreamConfig holds configuration for gRPC streaming.
type GRPCStreamConfig struct {
	OnError       func(error)
	OnClose       func()
	OnConnect     func()
	OnReconnect   func(attempt int)
	OnStateChange func(ConnectionState)
	Secure        bool
	Reconnect     bool
	MaxReconnect  int
}

// DefaultGRPCStreamConfig returns default gRPC stream configuration.
func DefaultGRPCStreamConfig() *GRPCStreamConfig {
	return &GRPCStreamConfig{
		Secure:       true,
		Reconnect:    true,
		MaxReconnect: 0, // Infinite
	}
}

// GRPCStream is a gRPC client for real-time data streams.
type GRPCStream struct {
	host   string
	token  string
	config *GRPCStreamConfig

	conn           *grpc.ClientConn
	streamingStub  pb.StreamingClient
	blockStub      pb.BlockStreamingClient
	orderbookStub  pb.OrderBookStreamingClient
	connMu         sync.RWMutex
	state          atomic.Int32
	running        atomic.Bool
	reconnectNum   atomic.Int32
	reconnectDelay time.Duration

	subscriptions []grpcSubscription
	subMu         sync.RWMutex

	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	stopOnce sync.Once
}

type grpcSubscription struct {
	streamType          string
	callback            func(map[string]any)
	bytesCallback       func(blockNumber uint64, timestamp uint64, data []byte)
	coins               []string
	users               []string
	coin                string
	nSigFigs            *int
	nLevels             int
	mantissa            *uint64
	skipInitialSnapshot bool
	startBlock          uint64
	raw                 bool
}

// StreamOption configures a generic gRPC data stream subscription.
type StreamOption func(*grpcSubscription)

// StreamWithStartBlock requests the stream to start from a specific block number.
func StreamWithStartBlock(block uint64) StreamOption {
	return func(s *grpcSubscription) {
		s.startBlock = block
	}
}

// StreamWithCoins filters the stream by coins (used with StreamBytes).
func StreamWithCoins(coins ...string) StreamOption {
	return func(s *grpcSubscription) {
		s.coins = coins
	}
}

// StreamWithUsers filters the stream by users (used with StreamBytes).
func StreamWithUsers(users ...string) StreamOption {
	return func(s *grpcSubscription) {
		s.users = users
	}
}

const (
	grpcPort                  = 10000
	grpcInitialReconnectDelay = 1 * time.Second
	grpcMaxReconnectDelay     = 60 * time.Second
	grpcReconnectBackoff      = 2.0
	grpcKeepaliveTime         = 30 * time.Second
	grpcKeepaliveTimeout      = 10 * time.Second
	grpcMaxRecvMsgSize        = 100 * 1024 * 1024 // 100MB
	grpcMaxSendMsgSize        = 100 * 1024 * 1024 // 100MB
)

// NewGRPCStream creates a new gRPC stream client.
func NewGRPCStream(endpoint string, config *GRPCStreamConfig) *GRPCStream {
	// Start with defaults and merge user config
	defaults := DefaultGRPCStreamConfig()
	if config == nil {
		config = defaults
	} else {
		// User can only explicitly disable Secure by setting it to false
		// Since we can't distinguish "not set" from "set to false", we default to secure (true)
		// This is the safe default - users must explicitly use insecure connections
		merged := &GRPCStreamConfig{
			OnError:       config.OnError,
			OnClose:       config.OnClose,
			OnConnect:     config.OnConnect,
			OnReconnect:   config.OnReconnect,
			OnStateChange: config.OnStateChange,
			Secure:        true, // Default to secure
			Reconnect:     config.Reconnect,
			MaxReconnect:  config.MaxReconnect,
		}
		config = merged
	}

	host, token := parseGRPCEndpoint(endpoint)

	ctx, cancel := context.WithCancel(context.Background())

	s := &GRPCStream{
		host:           host,
		token:          token,
		config:         config,
		subscriptions:  make([]grpcSubscription, 0),
		reconnectDelay: grpcInitialReconnectDelay,
		ctx:            ctx,
		cancel:         cancel,
	}

	return s
}

func parseGRPCEndpoint(endpoint string) (string, string) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint, ""
	}

	host := parsed.Host
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Extract token from path
	pathParts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	token := ""
	for _, part := range pathParts {
		if part != "" && part != "info" && part != "hypercore" && part != "evm" && part != "nanoreth" && part != "ws" {
			token = part
			break
		}
	}

	return host, token
}

func (s *GRPCStream) setState(state ConnectionState) {
	var stateInt int32
	switch state {
	case ConnectionStateDisconnected:
		stateInt = stateDisconnected
	case ConnectionStateConnecting:
		stateInt = stateConnecting
	case ConnectionStateConnected:
		stateInt = stateConnected
	case ConnectionStateReconnecting:
		stateInt = stateReconnecting
	}

	if s.state.Swap(stateInt) != stateInt && s.config.OnStateChange != nil {
		s.config.OnStateChange(state)
	}
}

func (s *GRPCStream) getState() ConnectionState {
	switch s.state.Load() {
	case stateConnected:
		return ConnectionStateConnected
	case stateConnecting:
		return ConnectionStateConnecting
	case stateReconnecting:
		return ConnectionStateReconnecting
	default:
		return ConnectionStateDisconnected
	}
}

// getMetadata returns metadata with x-token header for authentication.
func (s *GRPCStream) getMetadata() metadata.MD {
	return metadata.Pairs("x-token", s.token)
}

func (s *GRPCStream) connect() error {
	s.setState(ConnectionStateConnecting)

	target := fmt.Sprintf("%s:%d", s.host, grpcPort)

	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(grpcMaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(grpcMaxSendMsgSize),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                grpcKeepaliveTime,
			Timeout:             grpcKeepaliveTimeout,
			PermitWithoutStream: true,
		}),
	}

	if s.config.Secure {
		// Use TLS with proper server name for SNI (Server Name Indication)
		// This matches Python's grpc.ssl_channel_credentials() behavior
		tlsConfig := &tls.Config{
			ServerName: s.host,
		}
		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Use DialContext with a timeout to ensure connection is established
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return err
	}

	s.connMu.Lock()
	s.conn = conn
	s.streamingStub = pb.NewStreamingClient(conn)
	s.blockStub = pb.NewBlockStreamingClient(conn)
	s.orderbookStub = pb.NewOrderBookStreamingClient(conn)
	s.connMu.Unlock()

	s.setState(ConnectionStateConnected)
	s.reconnectNum.Store(0)
	s.reconnectDelay = grpcInitialReconnectDelay

	if s.config.OnConnect != nil {
		s.config.OnConnect()
	}

	return nil
}

// addSubscription applies options to a subscription and registers it.
func (s *GRPCStream) addSubscription(sub grpcSubscription, opts []StreamOption) *GRPCStream {
	for _, opt := range opts {
		opt(&sub)
	}
	s.subMu.Lock()
	s.subscriptions = append(s.subscriptions, sub)
	s.subMu.Unlock()
	return s
}

// Trades subscribes to trade stream.
func (s *GRPCStream) Trades(coins []string, callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "TRADES",
		callback:   callback,
		coins:      coins,
	}, opts)
}

// RawTrades subscribes to raw trade blocks.
func (s *GRPCStream) RawTrades(coins []string, callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "TRADES",
		callback:   callback,
		coins:      coins,
		raw:        true,
	}, opts)
}

// Orders subscribes to order stream.
func (s *GRPCStream) Orders(coins []string, callback func(map[string]any), users ...string) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "ORDERS",
		callback:   callback,
		coins:      coins,
		users:      users,
	}, nil)
}

// RawOrders subscribes to raw order blocks.
func (s *GRPCStream) RawOrders(coins []string, callback func(map[string]any), users ...string) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "ORDERS",
		callback:   callback,
		coins:      coins,
		users:      users,
		raw:        true,
	}, nil)
}

// BookUpdates subscribes to order book updates.
func (s *GRPCStream) BookUpdates(coins []string, callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "BOOK_UPDATES",
		callback:   callback,
		coins:      coins,
	}, opts)
}

// RawBookUpdates subscribes to raw order book update blocks.
func (s *GRPCStream) RawBookUpdates(coins []string, callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "BOOK_UPDATES",
		callback:   callback,
		coins:      coins,
		raw:        true,
	}, opts)
}

// TWAP subscribes to TWAP execution stream.
func (s *GRPCStream) TWAP(coins []string, callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "TWAP",
		callback:   callback,
		coins:      coins,
	}, opts)
}

// RawTWAP subscribes to raw TWAP execution blocks.
func (s *GRPCStream) RawTWAP(coins []string, callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "TWAP",
		callback:   callback,
		coins:      coins,
		raw:        true,
	}, opts)
}

// Events subscribes to system events.
func (s *GRPCStream) Events(callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "EVENTS",
		callback:   callback,
	}, opts)
}

// RawEvents subscribes to raw system event blocks.
func (s *GRPCStream) RawEvents(callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "EVENTS",
		callback:   callback,
		raw:        true,
	}, opts)
}

// Blocks subscribes to block data.
func (s *GRPCStream) Blocks(callback func(map[string]any)) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "BLOCKS",
		callback:   callback,
	}, nil)
}

// WriterActions subscribes to writer actions.
func (s *GRPCStream) WriterActions(callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "WRITER_ACTIONS",
		callback:   callback,
	}, opts)
}

// RawWriterActions subscribes to raw writer action blocks.
func (s *GRPCStream) RawWriterActions(callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "WRITER_ACTIONS",
		callback:   callback,
		raw:        true,
	}, opts)
}

// MempoolTxs subscribes to pre-consensus mempool transactions.
// Pass nil/empty coins for all coins; the server applies the coin filter
// with OR logic across values for this stream type.
func (s *GRPCStream) MempoolTxs(coins []string, callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "MEMPOOL_TXS",
		callback:   callback,
		coins:      coins,
	}, opts)
}

// RawMempoolTxs subscribes to raw pre-consensus mempool transaction blocks.
// Pass nil/empty coins for all coins.
func (s *GRPCStream) RawMempoolTxs(coins []string, callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "MEMPOOL_TXS",
		callback:   callback,
		coins:      coins,
		raw:        true,
	}, opts)
}

// OrderPriority subscribes to derived order/write priority actions
// (from mempool and confirmed replica data). Events carry server-enriched
// fields: coin, market_type, sz_decimals.
func (s *GRPCStream) OrderPriority(callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "ORDER_PRIORITY",
		callback:   callback,
	}, opts)
}

// RawOrderPriority subscribes to raw order/write priority action blocks.
func (s *GRPCStream) RawOrderPriority(callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "ORDER_PRIORITY",
		callback:   callback,
		raw:        true,
	}, opts)
}

// GossipPriority subscribes to derived gossip/read priority bid actions
// (does not measure delivery latency). Events carry server-enriched
// fields: coin, market_type, sz_decimals.
func (s *GRPCStream) GossipPriority(callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "GOSSIP_PRIORITY",
		callback:   callback,
	}, opts)
}

// RawGossipPriority subscribes to raw gossip/read priority bid action blocks.
func (s *GRPCStream) RawGossipPriority(callback func(map[string]any), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "GOSSIP_PRIORITY",
		callback:   callback,
		raw:        true,
	}, opts)
}

// StreamBytes subscribes to the raw-bytes fast path (StreamDataBytes RPC) for
// any generic stream type ("TRADES", "ORDERS", "BOOK_UPDATES", "TWAP",
// "EVENTS", "BLOCKS", "WRITER_ACTIONS", "MEMPOOL_TXS", "ORDER_PRIORITY",
// "GOSSIP_PRIORITY"). The callback receives the undecoded payload bytes — no
// JSON parsing is performed. Use StreamWithCoins/StreamWithUsers/
// StreamWithStartBlock to filter.
func (s *GRPCStream) StreamBytes(streamType string, callback func(blockNumber uint64, timestamp uint64, data []byte), opts ...StreamOption) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType:    streamType,
		bytesCallback: callback,
	}, opts)
}

// L2Book subscribes to Level 2 order book updates.
func (s *GRPCStream) L2Book(coin string, callback func(map[string]any), opts ...L2BookOption) *GRPCStream {
	sub := grpcSubscription{
		streamType: "L2_BOOK",
		callback:   callback,
		coin:       coin,
		nLevels:    20,
	}
	for _, opt := range opts {
		opt(&sub)
	}
	s.subMu.Lock()
	s.subscriptions = append(s.subscriptions, sub)
	s.subMu.Unlock()
	return s
}

// L2BookOption is an option for L2Book subscription.
type L2BookOption func(*grpcSubscription)

// L2BookNLevels sets the number of price levels.
func L2BookNLevels(n int) L2BookOption {
	return func(s *grpcSubscription) {
		s.nLevels = n
	}
}

// L2BookNSigFigs sets the number of significant figures for price aggregation.
func L2BookNSigFigs(n int) L2BookOption {
	return func(s *grpcSubscription) {
		s.nSigFigs = &n
	}
}

// L2BookMantissa sets the mantissa for price bucketing (1, 2, or 5).
func L2BookMantissa(m uint64) L2BookOption {
	return func(s *grpcSubscription) {
		s.mantissa = &m
	}
}

// L2BookSkipInitialSnapshot skips the initial per-coin snapshot (L2BookDiff only).
func L2BookSkipInitialSnapshot() L2BookOption {
	return func(s *grpcSubscription) {
		s.skipInitialSnapshot = true
	}
}

// L4Book subscribes to Level 4 order book updates (individual orders).
//
// Note: the server may send an unsolicited full snapshot at any time after
// subscribe; discard local book state and replace it with any snapshot
// received mid-stream.
func (s *GRPCStream) L4Book(coin string, callback func(map[string]any)) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "L4_BOOK",
		callback:   callback,
		coin:       coin,
	}, nil)
}

// BboBook subscribes to best bid/offer updates. Empty/nil coins = all coins.
// Emits only when the best bid or ask changes for a coin.
func (s *GRPCStream) BboBook(coins []string, callback func(map[string]any)) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "BBO_BOOK",
		callback:   callback,
		coins:      coins,
	}, nil)
}

// L2BookDiff subscribes to incremental L2 price-level changes.
// Empty/nil coins = all coins. Changed levels with sz=0 mean the level was removed.
func (s *GRPCStream) L2BookDiff(coins []string, callback func(map[string]any), opts ...L2BookOption) *GRPCStream {
	sub := grpcSubscription{
		streamType: "L2_BOOK_DIFF",
		callback:   callback,
		coins:      coins,
		nLevels:    20,
	}
	for _, opt := range opts {
		opt(&sub)
	}
	return s.addSubscription(sub, nil)
}

// L4BookUpdates subscribes to typed L4 order book updates.
// Empty/nil coins = all coins.
//
// Note: updates with snapshot=true carry a full reset snapshot; discard local
// book state and replace it whenever one arrives mid-stream.
func (s *GRPCStream) L4BookUpdates(coins []string, callback func(map[string]any)) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "L4_BOOK_UPDATES",
		callback:   callback,
		coins:      coins,
	}, nil)
}

// TpslUpdates subscribes to trigger/TP-SL order updates.
// Empty/nil coins = all perp coins.
func (s *GRPCStream) TpslUpdates(coins []string, callback func(map[string]any)) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "TPSL_UPDATES",
		callback:   callback,
		coins:      coins,
	}, nil)
}

// L2BookPacked subscribes to the fixed-point L2 book fast path.
// Prices/sizes are uint64 fixed-point integers scaled by 1e8.
func (s *GRPCStream) L2BookPacked(coin string, callback func(map[string]any), opts ...L2BookOption) *GRPCStream {
	sub := grpcSubscription{
		streamType: "L2_BOOK_PACKED",
		callback:   callback,
		coin:       coin,
		nLevels:    20,
	}
	for _, opt := range opts {
		opt(&sub)
	}
	return s.addSubscription(sub, nil)
}

// BboBookPacked subscribes to the fixed-point BBO fast path.
// Empty/nil coins = all coins. Prices/sizes are uint64 fixed-point integers
// scaled by 1e8.
func (s *GRPCStream) BboBookPacked(coins []string, callback func(map[string]any)) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "BBO_BOOK_PACKED",
		callback:   callback,
		coins:      coins,
	}, nil)
}

// L4BookBytes subscribes to the L4 book fast path: diffs are delivered as
// undecoded JSON bytes ({order_statuses, book_diffs}) in the "data" field
// instead of being parsed.
//
// Note: the server may send an unsolicited full snapshot at any time after
// subscribe; discard local book state and replace it with any snapshot
// received mid-stream.
func (s *GRPCStream) L4BookBytes(coin string, callback func(map[string]any)) *GRPCStream {
	return s.addSubscription(grpcSubscription{
		streamType: "L4_BOOK_BYTES",
		callback:   callback,
		coin:       coin,
	}, nil)
}

// grpcStreamTypeMap maps generic stream type names to proto enum values.
var grpcStreamTypeMap = map[string]pb.StreamType{
	"TRADES":          pb.StreamType_TRADES,
	"ORDERS":          pb.StreamType_ORDERS,
	"BOOK_UPDATES":    pb.StreamType_BOOK_UPDATES,
	"TWAP":            pb.StreamType_TWAP,
	"EVENTS":          pb.StreamType_EVENTS,
	"BLOCKS":          pb.StreamType_BLOCKS,
	"WRITER_ACTIONS":  pb.StreamType_WRITER_ACTIONS,
	"MEMPOOL_TXS":     pb.StreamType_MEMPOOL_TXS,
	"ORDER_PRIORITY":  pb.StreamType_ORDER_PRIORITY,
	"GOSSIP_PRIORITY": pb.StreamType_GOSSIP_PRIORITY,
}

// buildSubscribeRequest builds the initial StreamSubscribe request for a
// generic data subscription (StreamData / StreamDataBytes).
func buildSubscribeRequest(sub grpcSubscription) *pb.SubscribeRequest {
	req := &pb.SubscribeRequest{
		Request: &pb.SubscribeRequest_Subscribe{
			Subscribe: &pb.StreamSubscribe{
				StreamType: grpcStreamTypeMap[sub.streamType],
				StartBlock: sub.startBlock,
				Filters:    make(map[string]*pb.FilterValues),
			},
		},
	}

	if len(sub.coins) > 0 {
		req.GetSubscribe().Filters["coin"] = &pb.FilterValues{Values: sub.coins}
	}
	if len(sub.users) > 0 {
		req.GetSubscribe().Filters["user"] = &pb.FilterValues{Values: sub.users}
	}

	return req
}

func (s *GRPCStream) streamData(sub grpcSubscription) {
	defer s.wg.Done()

	retryDelay := time.Second

	for s.running.Load() {
		s.connMu.RLock()
		stub := s.streamingStub
		s.connMu.RUnlock()

		if stub == nil {
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}

		ctx := metadata.NewOutgoingContext(s.ctx, s.getMetadata())

		stream, err := stub.StreamData(ctx)
		if err != nil {
			if s.running.Load() && s.config.OnError != nil {
				s.config.OnError(err)
			}
			// Just retry with backoff, don't reconnect the whole connection
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(retryDelay):
				retryDelay = min(retryDelay*2, 30*time.Second)
				continue
			}
		}

		// Send initial subscription request
		req := buildSubscribeRequest(sub)

		if err := stream.Send(req); err != nil {
			if s.running.Load() && s.config.OnError != nil {
				s.config.OnError(err)
			}
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(retryDelay):
				retryDelay = min(retryDelay*2, 30*time.Second)
				continue
			}
		}

		// Reset retry delay on successful connection
		retryDelay = time.Second

		// Start ping goroutine
		pingDone := make(chan struct{})
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-pingDone:
					return
				case <-s.ctx.Done():
					return
				case <-ticker.C:
					pingReq := &pb.SubscribeRequest{
						Request: &pb.SubscribeRequest_Ping{
							Ping: &pb.Ping{Timestamp: time.Now().UnixMilli()},
						},
					}
					stream.Send(pingReq)
				}
			}
		}()

		// Handle responses
		for s.running.Load() {
			resp, err := stream.Recv()
			if err != nil {
				close(pingDone)
				if s.running.Load() && s.config.OnError != nil {
					s.config.OnError(err)
				}
				// Just break and retry, don't reconnect
				break
			}

			if data := resp.GetData(); data != nil {
				var parsed map[string]any
				if err := json.Unmarshal([]byte(data.Data), &parsed); err != nil {
					continue
				}

				if sub.raw {
					parsed["_block_number"] = data.BlockNumber
					parsed["_timestamp"] = data.Timestamp
					sub.callback(parsed)
					continue
				}

				// Extract events
				events, _ := parsed["events"].([]any)
				if len(events) > 0 {
					emittedEvents := false
					for i, event := range events {
						var user any
						var eventData map[string]any

						if eventArr, ok := event.([]any); ok && len(eventArr) >= 2 {
							user = eventArr[0]
							eventData, _ = eventArr[1].(map[string]any)
						} else if obj, ok := event.(map[string]any); ok {
							eventData = obj
						}

						if eventData != nil {
							eventData["_block_number"] = data.BlockNumber
							eventData["_timestamp"] = data.Timestamp
							eventData["_event_index"] = i
							if user != nil {
								eventData["_user"] = user
							}
							sub.callback(eventData)
							emittedEvents = true
						}
					}
					if !emittedEvents {
						parsed["_block_number"] = data.BlockNumber
						parsed["_timestamp"] = data.Timestamp
						sub.callback(parsed)
					}
				} else {
					parsed["_block_number"] = data.BlockNumber
					parsed["_timestamp"] = data.Timestamp
					sub.callback(parsed)
				}
			}
		}
	}
}

// streamDataBytes handles the StreamDataBytes fast path: payload bytes are
// passed through to the callback without JSON decoding.
func (s *GRPCStream) streamDataBytes(sub grpcSubscription) {
	defer s.wg.Done()

	retryDelay := time.Second

	for s.running.Load() {
		s.connMu.RLock()
		stub := s.streamingStub
		s.connMu.RUnlock()

		if stub == nil {
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}

		ctx := metadata.NewOutgoingContext(s.ctx, s.getMetadata())

		stream, err := stub.StreamDataBytes(ctx)
		if err != nil {
			if s.running.Load() && s.config.OnError != nil {
				s.config.OnError(err)
			}
			// Just retry with backoff, don't reconnect the whole connection
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(retryDelay):
				retryDelay = min(retryDelay*2, 30*time.Second)
				continue
			}
		}

		// Send initial subscription request
		req := buildSubscribeRequest(sub)

		if err := stream.Send(req); err != nil {
			if s.running.Load() && s.config.OnError != nil {
				s.config.OnError(err)
			}
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(retryDelay):
				retryDelay = min(retryDelay*2, 30*time.Second)
				continue
			}
		}

		// Reset retry delay on successful connection
		retryDelay = time.Second

		// Start ping goroutine
		pingDone := make(chan struct{})
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-pingDone:
					return
				case <-s.ctx.Done():
					return
				case <-ticker.C:
					pingReq := &pb.SubscribeRequest{
						Request: &pb.SubscribeRequest_Ping{
							Ping: &pb.Ping{Timestamp: time.Now().UnixMilli()},
						},
					}
					stream.Send(pingReq)
				}
			}
		}()

		// Handle responses
		for s.running.Load() {
			resp, err := stream.Recv()
			if err != nil {
				close(pingDone)
				if s.running.Load() && s.config.OnError != nil {
					s.config.OnError(err)
				}
				// Just break and retry, don't reconnect
				break
			}

			if data := resp.GetData(); data != nil {
				sub.bytesCallback(data.BlockNumber, data.Timestamp, data.Data)
			}
		}
	}
}

func (s *GRPCStream) streamBlocks(sub grpcSubscription) {
	defer s.wg.Done()

	retryDelay := time.Second

	for s.running.Load() {
		s.connMu.RLock()
		stub := s.blockStub
		s.connMu.RUnlock()

		if stub == nil {
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}

		ctx := metadata.NewOutgoingContext(s.ctx, s.getMetadata())
		req := &pb.Timestamp{Timestamp: time.Now().UnixMilli()}

		stream, err := stub.StreamBlocks(ctx, req)
		if err != nil {
			if s.running.Load() && s.config.OnError != nil {
				s.config.OnError(err)
			}
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(retryDelay):
				retryDelay = min(retryDelay*2, 30*time.Second)
				continue
			}
		}

		// Reset retry delay on successful connection
		retryDelay = time.Second

		for s.running.Load() {
			block, err := stream.Recv()
			if err != nil {
				if s.running.Load() && s.config.OnError != nil {
					s.config.OnError(err)
				}
				break
			}

			var data map[string]any
			if err := json.Unmarshal([]byte(block.DataJson), &data); err != nil {
				continue
			}
			sub.callback(data)
		}
	}
}

func (s *GRPCStream) streamL2Book(sub grpcSubscription) {
	defer s.wg.Done()

	retryDelay := time.Second

	for s.running.Load() {
		s.connMu.RLock()
		stub := s.orderbookStub
		s.connMu.RUnlock()

		if stub == nil {
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}

		ctx := metadata.NewOutgoingContext(s.ctx, s.getMetadata())
		req := &pb.L2BookRequest{
			Coin:    sub.coin,
			NLevels: uint32(sub.nLevels),
		}
		if sub.nSigFigs != nil {
			nSigFigs := uint32(*sub.nSigFigs)
			req.NSigFigs = &nSigFigs
		}
		if sub.mantissa != nil {
			mantissa := *sub.mantissa
			req.Mantissa = &mantissa
		}

		stream, err := stub.StreamL2Book(ctx, req)
		if err != nil {
			if s.running.Load() && s.config.OnError != nil {
				s.config.OnError(err)
			}
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(retryDelay):
				retryDelay = min(retryDelay*2, 30*time.Second)
				continue
			}
		}

		// Reset retry delay on successful connection
		retryDelay = time.Second

		for s.running.Load() {
			update, err := stream.Recv()
			if err != nil {
				if s.running.Load() && s.config.OnError != nil {
					s.config.OnError(err)
				}
				break
			}

			bids := make([][]any, len(update.Bids))
			for i, level := range update.Bids {
				bids[i] = []any{level.Px, level.Sz, level.N}
			}
			asks := make([][]any, len(update.Asks))
			for i, level := range update.Asks {
				asks[i] = []any{level.Px, level.Sz, level.N}
			}

			data := map[string]any{
				"coin":         update.Coin,
				"time":         update.Time,
				"block_number": update.BlockNumber,
				"bids":         bids,
				"asks":         asks,
			}
			sub.callback(data)
		}
	}
}

func (s *GRPCStream) streamL4Book(sub grpcSubscription) {
	defer s.wg.Done()

	retryDelay := time.Second

	for s.running.Load() {
		s.connMu.RLock()
		stub := s.orderbookStub
		s.connMu.RUnlock()

		if stub == nil {
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}

		ctx := metadata.NewOutgoingContext(s.ctx, s.getMetadata())
		req := &pb.L4BookRequest{Coin: sub.coin}

		stream, err := stub.StreamL4Book(ctx, req)
		if err != nil {
			if s.running.Load() && s.config.OnError != nil {
				s.config.OnError(err)
			}
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(retryDelay):
				retryDelay = min(retryDelay*2, 30*time.Second)
				continue
			}
		}

		// Reset retry delay on successful connection
		retryDelay = time.Second

		for s.running.Load() {
			update, err := stream.Recv()
			if err != nil {
				if s.running.Load() && s.config.OnError != nil {
					s.config.OnError(err)
				}
				break
			}

			var data map[string]any

			if snapshot := update.GetSnapshot(); snapshot != nil {
				data = l4SnapshotToMap(snapshot)
			} else if diff := update.GetDiff(); diff != nil {
				var diffData map[string]any
				json.Unmarshal([]byte(diff.Data), &diffData)

				data = map[string]any{
					"type":   "diff",
					"time":   diff.Time,
					"height": diff.Height,
					"data":   diffData,
				}
			} else {
				continue
			}

			sub.callback(data)
		}
	}
}

func l4OrderToMap(order *pb.L4Order) map[string]any {
	m := map[string]any{
		"user":              order.User,
		"coin":              order.Coin,
		"side":              order.Side,
		"limit_px":          order.LimitPx,
		"sz":                order.Sz,
		"oid":               order.Oid,
		"timestamp":         order.Timestamp,
		"trigger_condition": order.TriggerCondition,
		"is_trigger":        order.IsTrigger,
		"trigger_px":        order.TriggerPx,
		"is_position_tpsl":  order.IsPositionTpsl,
		"reduce_only":       order.ReduceOnly,
		"order_type":        order.OrderType,
	}
	if order.Tif != nil {
		m["tif"] = *order.Tif
	}
	if order.Cloid != nil {
		m["cloid"] = *order.Cloid
	}
	return m
}

func l4SnapshotToMap(snapshot *pb.L4BookSnapshot) map[string]any {
	bids := make([]map[string]any, len(snapshot.Bids))
	for i, order := range snapshot.Bids {
		bids[i] = l4OrderToMap(order)
	}
	asks := make([]map[string]any, len(snapshot.Asks))
	for i, order := range snapshot.Asks {
		asks[i] = l4OrderToMap(order)
	}

	return map[string]any{
		"type":   "snapshot",
		"coin":   snapshot.Coin,
		"time":   snapshot.Time,
		"height": snapshot.Height,
		"bids":   bids,
		"asks":   asks,
	}
}

func l2LevelToArr(level *pb.L2Level) []any {
	return []any{level.Px, level.Sz, level.N}
}

func l2LevelsToArrs(levels []*pb.L2Level) [][]any {
	arrs := make([][]any, len(levels))
	for i, level := range levels {
		arrs[i] = l2LevelToArr(level)
	}
	return arrs
}

func l2LevelPackedToArr(level *pb.L2LevelPacked) []any {
	return []any{level.Px, level.Sz, level.N}
}

func l2LevelsPackedToArrs(levels []*pb.L2LevelPacked) [][]any {
	arrs := make([][]any, len(levels))
	for i, level := range levels {
		arrs[i] = l2LevelPackedToArr(level)
	}
	return arrs
}

func bboBookUpdateToMap(update *pb.BboBookUpdate) map[string]any {
	m := map[string]any{
		"coin":         update.Coin,
		"time":         update.Time,
		"block_number": update.BlockNumber,
		"bid":          nil,
		"ask":          nil,
	}
	if update.Bid != nil {
		m["bid"] = l2LevelToArr(update.Bid)
	}
	if update.Ask != nil {
		m["ask"] = l2LevelToArr(update.Ask)
	}
	return m
}

func bboBookPackedUpdateToMap(update *pb.BboBookPackedUpdate) map[string]any {
	m := map[string]any{
		"coin":         update.Coin,
		"time":         update.Time,
		"block_number": update.BlockNumber,
		"bid":          nil,
		"ask":          nil,
	}
	if update.Bid != nil {
		m["bid"] = l2LevelPackedToArr(update.Bid)
	}
	if update.Ask != nil {
		m["ask"] = l2LevelPackedToArr(update.Ask)
	}
	return m
}

func l2BookDiffUpdateToMap(update *pb.L2BookDiffUpdate) map[string]any {
	diffs := make([]map[string]any, len(update.Diffs))
	for i, diff := range update.Diffs {
		diffs[i] = map[string]any{
			"coin":     diff.Coin,
			"seq":      diff.Seq,
			"prev_seq": diff.PrevSeq,
			"bids":     l2LevelsToArrs(diff.Bids),
			"asks":     l2LevelsToArrs(diff.Asks),
			"snapshot": diff.Snapshot,
		}
	}

	return map[string]any{
		"time":     update.Time,
		"height":   update.Height,
		"snapshot": update.Snapshot,
		"diffs":    diffs,
	}
}

func l2BookPackedUpdateToMap(update *pb.L2BookPackedUpdate) map[string]any {
	return map[string]any{
		"coin":         update.Coin,
		"time":         update.Time,
		"block_number": update.BlockNumber,
		"bids":         l2LevelsPackedToArrs(update.Bids),
		"asks":         l2LevelsPackedToArrs(update.Asks),
	}
}

func l4BookUpdatesUpdateToMap(update *pb.L4BookUpdatesUpdate) map[string]any {
	diffs := make([]map[string]any, len(update.Diffs))
	for i, diff := range update.Diffs {
		diffs[i] = map[string]any{
			"diff_type": diff.DiffType.String(),
			"coin":      diff.Coin,
			"oid":       diff.Oid,
			"user":      diff.User,
			"side":      diff.Side,
			"px":        diff.Px,
			"sz":        diff.Sz,
		}
	}

	return map[string]any{
		"time":     update.Time,
		"height":   update.Height,
		"snapshot": update.Snapshot,
		"diffs":    diffs,
	}
}

func tpslUpdatesUpdateToMap(update *pb.TpslUpdatesUpdate) map[string]any {
	diffs := make([]map[string]any, len(update.Diffs))
	for i, diff := range update.Diffs {
		diffs[i] = map[string]any{
			"diff_type":         diff.DiffType.String(),
			"oid":               diff.Oid,
			"coin":              diff.Coin,
			"user":              diff.User,
			"side":              diff.Side,
			"trigger_px":        diff.TriggerPx,
			"limit_px":          diff.LimitPx,
			"sz":                diff.Sz,
			"trigger_condition": diff.TriggerCondition,
			"order_type":        diff.OrderType,
			"is_position_tpsl":  diff.IsPositionTpsl,
			"reduce_only":       diff.ReduceOnly,
			"timestamp":         diff.Timestamp,
			"reason":            diff.Reason,
		}
	}

	return map[string]any{
		"time":     update.Time,
		"height":   update.Height,
		"snapshot": update.Snapshot,
		"diffs":    diffs,
	}
}

// l4BookBytesUpdateToMap maps an L4BookBytesUpdate oneof to the snapshot/diff
// shape used by L4Book, keeping the diff payload as undecoded JSON bytes.
func l4BookBytesUpdateToMap(update *pb.L4BookBytesUpdate) map[string]any {
	if snapshot := update.GetSnapshot(); snapshot != nil {
		return l4SnapshotToMap(snapshot)
	}
	if diff := update.GetDiff(); diff != nil {
		return map[string]any{
			"type":   "diff",
			"time":   diff.Time,
			"height": diff.Height,
			"data":   diff.Data, // JSON bytes, not decoded
		}
	}
	return nil
}

// runOrderbookStream runs a server-streaming order book RPC with the same
// retry/backoff behavior as the other stream handlers. Updates are converted
// by handle and skipped when it returns nil.
func runOrderbookStream[T any](
	s *GRPCStream,
	callback func(map[string]any),
	open func(ctx context.Context, stub pb.OrderBookStreamingClient) (grpc.ServerStreamingClient[T], error),
	handle func(*T) map[string]any,
) {
	defer s.wg.Done()

	retryDelay := time.Second

	for s.running.Load() {
		s.connMu.RLock()
		stub := s.orderbookStub
		s.connMu.RUnlock()

		if stub == nil {
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}

		ctx := metadata.NewOutgoingContext(s.ctx, s.getMetadata())

		stream, err := open(ctx, stub)
		if err != nil {
			if s.running.Load() && s.config.OnError != nil {
				s.config.OnError(err)
			}
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(retryDelay):
				retryDelay = min(retryDelay*2, 30*time.Second)
				continue
			}
		}

		// Reset retry delay on successful connection
		retryDelay = time.Second

		for s.running.Load() {
			update, err := stream.Recv()
			if err != nil {
				if s.running.Load() && s.config.OnError != nil {
					s.config.OnError(err)
				}
				break
			}

			if data := handle(update); data != nil {
				callback(data)
			}
		}
	}
}

func (s *GRPCStream) streamBboBook(sub grpcSubscription) {
	runOrderbookStream(s, sub.callback,
		func(ctx context.Context, stub pb.OrderBookStreamingClient) (grpc.ServerStreamingClient[pb.BboBookUpdate], error) {
			return stub.StreamBboBook(ctx, &pb.BboBookRequest{Coins: sub.coins})
		}, bboBookUpdateToMap)
}

func (s *GRPCStream) streamBboBookPacked(sub grpcSubscription) {
	runOrderbookStream(s, sub.callback,
		func(ctx context.Context, stub pb.OrderBookStreamingClient) (grpc.ServerStreamingClient[pb.BboBookPackedUpdate], error) {
			return stub.StreamBboBookPacked(ctx, &pb.BboBookRequest{Coins: sub.coins})
		}, bboBookPackedUpdateToMap)
}

func (s *GRPCStream) streamL2BookDiff(sub grpcSubscription) {
	req := &pb.L2BookDiffRequest{
		Coins:               sub.coins,
		NLevels:             uint32(sub.nLevels),
		SkipInitialSnapshot: sub.skipInitialSnapshot,
	}
	if sub.nSigFigs != nil {
		nSigFigs := uint32(*sub.nSigFigs)
		req.NSigFigs = &nSigFigs
	}
	if sub.mantissa != nil {
		mantissa := *sub.mantissa
		req.Mantissa = &mantissa
	}

	runOrderbookStream(s, sub.callback,
		func(ctx context.Context, stub pb.OrderBookStreamingClient) (grpc.ServerStreamingClient[pb.L2BookDiffUpdate], error) {
			return stub.StreamL2BookDiff(ctx, req)
		}, l2BookDiffUpdateToMap)
}

func (s *GRPCStream) streamL2BookPacked(sub grpcSubscription) {
	req := &pb.L2BookRequest{
		Coin:    sub.coin,
		NLevels: uint32(sub.nLevels),
	}
	if sub.nSigFigs != nil {
		nSigFigs := uint32(*sub.nSigFigs)
		req.NSigFigs = &nSigFigs
	}
	if sub.mantissa != nil {
		mantissa := *sub.mantissa
		req.Mantissa = &mantissa
	}

	runOrderbookStream(s, sub.callback,
		func(ctx context.Context, stub pb.OrderBookStreamingClient) (grpc.ServerStreamingClient[pb.L2BookPackedUpdate], error) {
			return stub.StreamL2BookPacked(ctx, req)
		}, l2BookPackedUpdateToMap)
}

func (s *GRPCStream) streamL4BookUpdates(sub grpcSubscription) {
	runOrderbookStream(s, sub.callback,
		func(ctx context.Context, stub pb.OrderBookStreamingClient) (grpc.ServerStreamingClient[pb.L4BookUpdatesUpdate], error) {
			return stub.StreamL4BookUpdates(ctx, &pb.L4BookUpdatesRequest{Coins: sub.coins})
		}, l4BookUpdatesUpdateToMap)
}

func (s *GRPCStream) streamTpslUpdates(sub grpcSubscription) {
	runOrderbookStream(s, sub.callback,
		func(ctx context.Context, stub pb.OrderBookStreamingClient) (grpc.ServerStreamingClient[pb.TpslUpdatesUpdate], error) {
			return stub.StreamTpslUpdates(ctx, &pb.TpslUpdatesRequest{Coins: sub.coins})
		}, tpslUpdatesUpdateToMap)
}

func (s *GRPCStream) streamL4BookBytes(sub grpcSubscription) {
	runOrderbookStream(s, sub.callback,
		func(ctx context.Context, stub pb.OrderBookStreamingClient) (grpc.ServerStreamingClient[pb.L4BookBytesUpdate], error) {
			return stub.StreamL4BookBytes(ctx, &pb.L4BookRequest{Coin: sub.coin})
		}, l4BookBytesUpdateToMap)
}

func (s *GRPCStream) handleReconnect() {
	if !s.running.Load() {
		return
	}

	maxReconnect := s.config.MaxReconnect
	attempt := int(s.reconnectNum.Add(1))

	if maxReconnect > 0 && attempt > maxReconnect {
		s.running.Store(false)
		s.setState(ConnectionStateDisconnected)
		if s.config.OnClose != nil {
			s.config.OnClose()
		}
		return
	}

	s.setState(ConnectionStateReconnecting)

	if s.config.OnReconnect != nil {
		s.config.OnReconnect(attempt)
	}

	select {
	case <-s.ctx.Done():
		return
	case <-time.After(s.reconnectDelay):
	}

	if s.reconnectDelay < grpcMaxReconnectDelay {
		s.reconnectDelay = time.Duration(float64(s.reconnectDelay) * grpcReconnectBackoff)
		if s.reconnectDelay > grpcMaxReconnectDelay {
			s.reconnectDelay = grpcMaxReconnectDelay
		}
	}

	if s.running.Load() {
		s.connMu.Lock()
		if s.conn != nil {
			s.conn.Close()
		}
		s.connMu.Unlock()

		s.connect()
	}
}

func (s *GRPCStream) startStreams() {
	s.subMu.RLock()
	subs := make([]grpcSubscription, len(s.subscriptions))
	copy(subs, s.subscriptions)
	s.subMu.RUnlock()

	for _, sub := range subs {
		s.wg.Add(1)
		if sub.bytesCallback != nil {
			go s.streamDataBytes(sub)
			continue
		}
		switch sub.streamType {
		case "L2_BOOK":
			go s.streamL2Book(sub)
		case "L4_BOOK":
			go s.streamL4Book(sub)
		case "BLOCKS":
			go s.streamBlocks(sub)
		case "BBO_BOOK":
			go s.streamBboBook(sub)
		case "BBO_BOOK_PACKED":
			go s.streamBboBookPacked(sub)
		case "L2_BOOK_DIFF":
			go s.streamL2BookDiff(sub)
		case "L2_BOOK_PACKED":
			go s.streamL2BookPacked(sub)
		case "L4_BOOK_UPDATES":
			go s.streamL4BookUpdates(sub)
		case "TPSL_UPDATES":
			go s.streamTpslUpdates(sub)
		case "L4_BOOK_BYTES":
			go s.streamL4BookBytes(sub)
		default:
			go s.streamData(sub)
		}
	}
}

// Ping tests connectivity.
func (s *GRPCStream) Ping() bool {
	s.connMu.RLock()
	stub := s.streamingStub
	s.connMu.RUnlock()

	if stub == nil {
		return false
	}

	ctx := metadata.NewOutgoingContext(s.ctx, s.getMetadata())
	req := &pb.PingRequest{Count: 1}

	resp, err := stub.Ping(ctx, req)
	if err != nil {
		return false
	}
	return resp.Count == 1
}

// Run runs the stream (blocking).
func (s *GRPCStream) Run() error {
	s.running.Store(true)

	if err := s.connect(); err != nil {
		return err
	}

	s.startStreams()
	s.wg.Wait()

	return nil
}

// Start starts the stream in background.
func (s *GRPCStream) Start() error {
	s.running.Store(true)

	if err := s.connect(); err != nil {
		return err
	}

	s.startStreams()
	return nil
}

// Stop stops the stream.
func (s *GRPCStream) Stop() {
	s.stopOnce.Do(func() {
		s.running.Store(false)
		s.cancel()

		s.connMu.Lock()
		if s.conn != nil {
			s.conn.Close()
			s.conn = nil
		}
		s.streamingStub = nil
		s.blockStub = nil
		s.orderbookStub = nil
		s.connMu.Unlock()

		s.wg.Wait()

		s.setState(ConnectionStateDisconnected)

		if s.config.OnClose != nil {
			s.config.OnClose()
		}
	})
}

// Connected returns true if the stream is connected.
func (s *GRPCStream) Connected() bool {
	return s.getState() == ConnectionStateConnected
}

// State returns the current connection state.
func (s *GRPCStream) State() ConnectionState {
	return s.getState()
}

// ReconnectAttempts returns the number of reconnection attempts.
func (s *GRPCStream) ReconnectAttempts() int {
	return int(s.reconnectNum.Load())
}
