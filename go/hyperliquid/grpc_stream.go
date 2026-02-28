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

	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	stopOnce   sync.Once
}

type grpcSubscription struct {
	streamType string
	callback   func(map[string]any)
	coins      []string
	users      []string
	coin       string
	nSigFigs   *int
	nLevels    int
}

const (
	grpcPort                   = 10000
	grpcInitialReconnectDelay  = 1 * time.Second
	grpcMaxReconnectDelay      = 60 * time.Second
	grpcReconnectBackoff       = 2.0
	grpcKeepaliveTime          = 30 * time.Second
	grpcKeepaliveTimeout       = 10 * time.Second
	grpcMaxRecvMsgSize         = 100 * 1024 * 1024 // 100MB
	grpcMaxSendMsgSize         = 100 * 1024 * 1024 // 100MB
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

// Trades subscribes to trade stream.
func (s *GRPCStream) Trades(coins []string, callback func(map[string]any)) *GRPCStream {
	s.subMu.Lock()
	s.subscriptions = append(s.subscriptions, grpcSubscription{
		streamType: "TRADES",
		callback:   callback,
		coins:      coins,
	})
	s.subMu.Unlock()
	return s
}

// Orders subscribes to order stream.
func (s *GRPCStream) Orders(coins []string, callback func(map[string]any), users ...string) *GRPCStream {
	s.subMu.Lock()
	s.subscriptions = append(s.subscriptions, grpcSubscription{
		streamType: "ORDERS",
		callback:   callback,
		coins:      coins,
		users:      users,
	})
	s.subMu.Unlock()
	return s
}

// BookUpdates subscribes to order book updates.
func (s *GRPCStream) BookUpdates(coins []string, callback func(map[string]any)) *GRPCStream {
	s.subMu.Lock()
	s.subscriptions = append(s.subscriptions, grpcSubscription{
		streamType: "BOOK_UPDATES",
		callback:   callback,
		coins:      coins,
	})
	s.subMu.Unlock()
	return s
}

// TWAP subscribes to TWAP execution stream.
func (s *GRPCStream) TWAP(coins []string, callback func(map[string]any)) *GRPCStream {
	s.subMu.Lock()
	s.subscriptions = append(s.subscriptions, grpcSubscription{
		streamType: "TWAP",
		callback:   callback,
		coins:      coins,
	})
	s.subMu.Unlock()
	return s
}

// Events subscribes to system events.
func (s *GRPCStream) Events(callback func(map[string]any)) *GRPCStream {
	s.subMu.Lock()
	s.subscriptions = append(s.subscriptions, grpcSubscription{
		streamType: "EVENTS",
		callback:   callback,
	})
	s.subMu.Unlock()
	return s
}

// Blocks subscribes to block data.
func (s *GRPCStream) Blocks(callback func(map[string]any)) *GRPCStream {
	s.subMu.Lock()
	s.subscriptions = append(s.subscriptions, grpcSubscription{
		streamType: "BLOCKS",
		callback:   callback,
	})
	s.subMu.Unlock()
	return s
}

// WriterActions subscribes to writer actions.
func (s *GRPCStream) WriterActions(callback func(map[string]any)) *GRPCStream {
	s.subMu.Lock()
	s.subscriptions = append(s.subscriptions, grpcSubscription{
		streamType: "WRITER_ACTIONS",
		callback:   callback,
	})
	s.subMu.Unlock()
	return s
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

// L4Book subscribes to Level 4 order book updates (individual orders).
func (s *GRPCStream) L4Book(coin string, callback func(map[string]any)) *GRPCStream {
	s.subMu.Lock()
	s.subscriptions = append(s.subscriptions, grpcSubscription{
		streamType: "L4_BOOK",
		callback:   callback,
		coin:       coin,
	})
	s.subMu.Unlock()
	return s
}

func (s *GRPCStream) streamData(sub grpcSubscription) {
	defer s.wg.Done()

	streamTypeMap := map[string]pb.StreamType{
		"TRADES":         pb.StreamType_TRADES,
		"ORDERS":         pb.StreamType_ORDERS,
		"BOOK_UPDATES":   pb.StreamType_BOOK_UPDATES,
		"TWAP":           pb.StreamType_TWAP,
		"EVENTS":         pb.StreamType_EVENTS,
		"BLOCKS":         pb.StreamType_BLOCKS,
		"WRITER_ACTIONS": pb.StreamType_WRITER_ACTIONS,
	}

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
		req := &pb.SubscribeRequest{
			Request: &pb.SubscribeRequest_Subscribe{
				Subscribe: &pb.StreamSubscribe{
					StreamType: streamTypeMap[sub.streamType],
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

				// Extract events
				events, _ := parsed["events"].([]any)
				if len(events) > 0 {
					for _, event := range events {
						if eventArr, ok := event.([]any); ok && len(eventArr) >= 2 {
							user, _ := eventArr[0].(string)
							if eventData, ok := eventArr[1].(map[string]any); ok {
								eventData["_block_number"] = data.BlockNumber
								eventData["_timestamp"] = data.Timestamp
								eventData["_user"] = user
								sub.callback(eventData)
							}
						}
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
				bids := make([]map[string]any, len(snapshot.Bids))
				for i, order := range snapshot.Bids {
					bids[i] = l4OrderToMap(order)
				}
				asks := make([]map[string]any, len(snapshot.Asks))
				for i, order := range snapshot.Asks {
					asks[i] = l4OrderToMap(order)
				}

				data = map[string]any{
					"type":   "snapshot",
					"coin":   snapshot.Coin,
					"time":   snapshot.Time,
					"height": snapshot.Height,
					"bids":   bids,
					"asks":   asks,
				}
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
		"user":             order.User,
		"coin":             order.Coin,
		"side":             order.Side,
		"limit_px":         order.LimitPx,
		"sz":               order.Sz,
		"oid":              order.Oid,
		"timestamp":        order.Timestamp,
		"trigger_condition": order.TriggerCondition,
		"is_trigger":       order.IsTrigger,
		"trigger_px":       order.TriggerPx,
		"is_position_tpsl": order.IsPositionTpsl,
		"reduce_only":      order.ReduceOnly,
		"order_type":       order.OrderType,
	}
	if order.Tif != nil {
		m["tif"] = *order.Tif
	}
	if order.Cloid != nil {
		m["cloid"] = *order.Cloid
	}
	return m
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
		switch sub.streamType {
		case "L2_BOOK":
			go s.streamL2Book(sub)
		case "L4_BOOK":
			go s.streamL4Book(sub)
		case "BLOCKS":
			go s.streamBlocks(sub)
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
