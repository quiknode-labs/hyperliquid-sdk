/**
 * Tests for the gRPC stream client's new streams (mempool/priority, start_block,
 * StreamDataBytes, and the 2026 order book RPCs).
 *
 * No network: the grpc clients are replaced with fakes that capture the RPC
 * name + request and let tests emit updates. Private members are reached via
 * `as any`, mirroring signer.test.ts.
 */

import { describe, it, expect, afterEach } from 'vitest';
import { GRPCStream, GRPCStreamType } from '../src/grpc-stream';
import { GRPCStreamType as ProtoStreamType } from '../src/types';

const ENDPOINT = 'https://x.hype-mainnet.quiknode.pro/TOKEN';

type Handler = (arg?: unknown) => void;

/** Fake gRPC stream: records write() calls and lets tests emit events. */
function fakeStream() {
  const writes: any[] = [];
  const handlers: Record<string, Handler[]> = {};
  return {
    writes,
    write: (msg: any) => writes.push(msg),
    on(event: string, fn: Handler) {
      (handlers[event] = handlers[event] ?? []).push(fn);
      return this;
    },
    emit(event: string, arg?: unknown) {
      for (const fn of handlers[event] ?? []) fn(arg);
    },
    cancel: () => {},
  };
}

/** Fake client whose every RPC returns a fresh fake stream and records calls. */
function fakeClient(rpcNames: string[]) {
  const calls: Array<{ rpc: string; request: any; stream: ReturnType<typeof fakeStream> }> = [];
  const client: Record<string, unknown> = {};
  for (const rpc of rpcNames) {
    client[rpc] = (requestOrMetadata: any, _metadata?: any) => {
      const stream = fakeStream();
      // Bidi RPCs are called with metadata only; server-streaming with (request, metadata).
      const request = _metadata === undefined ? undefined : requestOrMetadata;
      calls.push({ rpc, request, stream });
      return stream;
    };
  }
  return { calls, client };
}

/** Wire fakes into a GRPCStream and run _startStreams without connecting. */
function startWithFakes(stream: GRPCStream) {
  const streaming = fakeClient(['StreamData', 'StreamDataBytes']);
  const orderbook = fakeClient([
    'StreamL2Book',
    'StreamL4Book',
    'StreamBboBook',
    'StreamL2BookDiff',
    'StreamL4BookUpdates',
    'StreamTpslUpdates',
    'StreamL2BookPacked',
    'StreamBboBookPacked',
    'StreamL4BookBytes',
  ]);
  const blocks = fakeClient(['StreamBlocks']);
  const s = stream as any;
  s._streamingClient = streaming.client;
  s._orderbookClient = orderbook.client;
  s._blockClient = blocks.client;
  s._running = true;
  s._startStreams();
  return { streaming, orderbook, blocks };
}

let active: GRPCStream | null = null;

afterEach(() => {
  // Clear ping intervals created by data streams.
  active?.stop();
  active = null;
});

function makeStream(): GRPCStream {
  active = new GRPCStream(ENDPOINT, { reconnect: false });
  return active;
}

describe('stream type maps', () => {
  it('proto enum includes the new stream types 8/9/10', () => {
    expect(ProtoStreamType.MEMPOOL_TXS).toBe(8);
    expect(ProtoStreamType.ORDER_PRIORITY).toBe(9);
    expect(ProtoStreamType.GOSSIP_PRIORITY).toBe(10);
  });

  it('client enum includes the new stream types', () => {
    expect(GRPCStreamType.MEMPOOL_TXS).toBe('MEMPOOL_TXS');
    expect(GRPCStreamType.ORDER_PRIORITY).toBe('ORDER_PRIORITY');
    expect(GRPCStreamType.GOSSIP_PRIORITY).toBe('GOSSIP_PRIORITY');
  });
});

describe('generic data streams', () => {
  it('mempoolTxs(coins) subscribes with stream_type 8 and a coin filter', () => {
    const s = makeStream().mempoolTxs(['BTC', 'ETH'], () => {});
    const { streaming } = startWithFakes(s);

    expect(streaming.calls).toHaveLength(1);
    expect(streaming.calls[0].rpc).toBe('StreamData');
    const req = streaming.calls[0].stream.writes[0];
    expect(req.subscribe.stream_type).toBe(8);
    expect(req.subscribe.filters).toEqual({ coin: { values: ['BTC', 'ETH'] } });
  });

  it('mempoolTxs(callback) subscribes unfiltered', () => {
    const s = makeStream().mempoolTxs(() => {});
    const { streaming } = startWithFakes(s);

    const req = streaming.calls[0].stream.writes[0];
    expect(req.subscribe.stream_type).toBe(8);
    expect(req.subscribe.filters).toEqual({});
  });

  it('orderPriority and gossipPriority subscribe with stream_type 9 and 10', () => {
    const s = makeStream().orderPriority(() => {}).gossipPriority(() => {});
    const { streaming } = startWithFakes(s);

    expect(streaming.calls.map((c) => c.stream.writes[0].subscribe.stream_type)).toEqual([9, 10]);
  });

  it('plumbs startBlock into StreamSubscribe.start_block', () => {
    const s = makeStream().trades(['BTC'], () => {}, { startBlock: 750_000_000 });
    const { streaming } = startWithFakes(s);

    const req = streaming.calls[0].stream.writes[0];
    expect(req.subscribe.start_block).toBe(750_000_000);
  });

  it('omits start_block when startBlock is not given', () => {
    const s = makeStream().trades(['BTC'], () => {});
    const { streaming } = startWithFakes(s);

    const req = streaming.calls[0].stream.writes[0];
    expect('start_block' in req.subscribe).toBe(false);
  });

  it('streamDataBytes uses the StreamDataBytes RPC and passes bytes through undecoded', () => {
    const received: any[] = [];
    const s = makeStream().streamDataBytes(GRPCStreamType.MEMPOOL_TXS, (d) => received.push(d), {
      coins: ['BTC'],
      startBlock: 42,
    });
    const { streaming } = startWithFakes(s);

    expect(streaming.calls).toHaveLength(1);
    expect(streaming.calls[0].rpc).toBe('StreamDataBytes');
    const req = streaming.calls[0].stream.writes[0];
    expect(req.subscribe.stream_type).toBe(8);
    expect(req.subscribe.start_block).toBe(42);
    expect(req.subscribe.filters).toEqual({ coin: { values: ['BTC'] } });

    const payload = Buffer.from('{"not":"parsed"}');
    streaming.calls[0].stream.emit('data', {
      data: { block_number: 7, timestamp: 1234, data: payload },
    });
    expect(received).toHaveLength(1);
    expect(received[0].block_number).toBe(7);
    expect(received[0].timestamp).toBe(1234);
    expect(received[0].data).toBe(payload); // exact bytes, no JSON decoding
  });
});

describe('order book streams', () => {
  it('bboBook defaults to all coins and forwards updates as-is', () => {
    const received: any[] = [];
    const s = makeStream().bboBook((d) => received.push(d));
    const { orderbook } = startWithFakes(s);

    expect(orderbook.calls).toHaveLength(1);
    expect(orderbook.calls[0].rpc).toBe('StreamBboBook');
    expect(orderbook.calls[0].request).toEqual({ coins: [] });

    const update = {
      coin: 'BTC',
      time: 1,
      block_number: 2,
      bid: { px: '100', sz: '1', n: 3 },
      ask: null,
    };
    orderbook.calls[0].stream.emit('data', update);
    expect(received).toEqual([update]);
  });

  it('bboBook(coins) passes the coin list', () => {
    const s = makeStream().bboBook(['BTC'], () => {});
    const { orderbook } = startWithFakes(s);
    expect(orderbook.calls[0].request).toEqual({ coins: ['BTC'] });
  });

  it('l2BookDiff maps all options onto the request', () => {
    const s = makeStream().l2BookDiff(() => {}, {
      coins: ['BTC', 'ETH'],
      nLevels: 50,
      nSigFigs: 5,
      mantissa: 2,
      skipInitialSnapshot: true,
    });
    const { orderbook } = startWithFakes(s);

    expect(orderbook.calls[0].rpc).toBe('StreamL2BookDiff');
    expect(orderbook.calls[0].request).toEqual({
      coins: ['BTC', 'ETH'],
      n_levels: 50,
      n_sig_figs: 5,
      mantissa: 2,
      skip_initial_snapshot: true,
    });
  });

  it('l2BookDiff omits optional bucketing fields by default', () => {
    const s = makeStream().l2BookDiff(() => {});
    const { orderbook } = startWithFakes(s);
    expect(orderbook.calls[0].request).toEqual({
      coins: [],
      n_levels: 20,
      skip_initial_snapshot: false,
    });
  });

  it('l4BookUpdates and tpslUpdates use the coins request shape', () => {
    const s = makeStream().l4BookUpdates(['BTC'], () => {}).tpslUpdates(() => {});
    const { orderbook } = startWithFakes(s);

    expect(orderbook.calls.map((c) => [c.rpc, c.request])).toEqual([
      ['StreamL4BookUpdates', { coins: ['BTC'] }],
      ['StreamTpslUpdates', { coins: [] }],
    ]);
  });

  it('l2BookPacked builds an L2BookRequest with optional bucketing', () => {
    const s = makeStream().l2BookPacked('BTC', () => {}, { nSigFigs: 4, mantissa: 5, nLevels: 10 });
    const { orderbook } = startWithFakes(s);

    expect(orderbook.calls[0].rpc).toBe('StreamL2BookPacked');
    expect(orderbook.calls[0].request).toEqual({
      coin: 'BTC',
      n_levels: 10,
      n_sig_figs: 4,
      mantissa: 5,
    });
  });

  it('bboBookPacked uses the coins request shape', () => {
    const s = makeStream().bboBookPacked(['ETH'], () => {});
    const { orderbook } = startWithFakes(s);
    expect(orderbook.calls[0].rpc).toBe('StreamBboBookPacked');
    expect(orderbook.calls[0].request).toEqual({ coins: ['ETH'] });
  });

  it('l4BookBytes maps snapshots like l4Book and keeps diff bytes undecoded', () => {
    const received: any[] = [];
    const s = makeStream().l4BookBytes('BTC', (d) => received.push(d));
    const { orderbook } = startWithFakes(s);

    expect(orderbook.calls[0].rpc).toBe('StreamL4BookBytes');
    expect(orderbook.calls[0].request).toEqual({ coin: 'BTC' });

    const order = {
      user: '0xabc',
      coin: 'BTC',
      side: 'B',
      limit_px: '100',
      sz: '1',
      oid: 5,
      timestamp: 9,
      trigger_condition: 'N/A',
      is_trigger: false,
      trigger_px: '0',
      is_position_tpsl: false,
      reduce_only: false,
      order_type: 'Limit',
      tif: 'Gtc',
      cloid: undefined,
    };
    orderbook.calls[0].stream.emit('data', {
      snapshot: { coin: 'BTC', time: 1, height: 2, bids: [order], asks: [] },
    });
    const bytes = Buffer.from('{"order_statuses":[],"book_diffs":[]}');
    orderbook.calls[0].stream.emit('data', {
      diff: { time: 3, height: 4, data: bytes },
    });

    expect(received).toHaveLength(2);
    expect(received[0].type).toBe('snapshot');
    expect(received[0].coin).toBe('BTC');
    expect(received[0].bids[0].oid).toBe(5);
    expect(received[1]).toEqual({ type: 'diff', time: 3, height: 4, data: bytes });
  });

  it('l2Book still maps levels to [px, sz, n] tuples (regression)', () => {
    const received: any[] = [];
    const s = makeStream().l2Book('BTC', (d) => received.push(d));
    const { orderbook } = startWithFakes(s);

    expect(orderbook.calls[0].rpc).toBe('StreamL2Book');
    orderbook.calls[0].stream.emit('data', {
      coin: 'BTC',
      time: 1,
      block_number: 2,
      bids: [{ px: '100', sz: '1', n: 3 }],
      asks: [],
    });
    expect(received[0].bids).toEqual([['100', '1', 3]]);
  });
});
