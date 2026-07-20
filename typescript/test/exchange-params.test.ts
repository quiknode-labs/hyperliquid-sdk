/**
 * Tests for request-level trading params: vaultAddress, expiresAfter, and the
 * fast cancel flag.
 *
 * The worker folds vaultAddress/expiresAfter into the action hash at build
 * time, so both must appear in the build payload AND the send payload; the
 * fast flag is action-level (`f: true`, never emitted when false/unset).
 * Mirrors the fetch-mocking pattern of signer.test.ts.
 */

import { describe, it, expect, vi, afterEach } from 'vitest';
import { HyperliquidSDK } from '../src/client';
import { Order } from '../src/order';
import { ValidationError } from '../src/errors';

const HASH = '0x' + '00'.repeat(32);
const VAULT = '0x1234567890123456789012345678901234567890';
const EXPIRES = 1_800_000_000_000;

/** Mock fetch: build (no signature in body) returns hash+nonce; send returns ok. */
function mockExchangeFetch(): { calls: any[]; restore: () => void } {
  const calls: any[] = [];
  const fn = vi.fn(async (_url: string, init: any) => {
    const body = JSON.parse(init.body);
    calls.push(body);
    const payload =
      'signature' in body
        ? { status: 'ok' }
        : { hash: HASH, nonce: 123, action: body.action };
    return { ok: true, status: 200, json: async () => payload } as any;
  });
  const original = globalThis.fetch;
  globalThis.fetch = fn as any;
  return { calls, restore: () => { globalThis.fetch = original; } };
}

function makeSdk(): HyperliquidSDK {
  return new HyperliquidSDK('https://x.quiknode.pro/T', {
    signer: () => ({ r: '0xaa', s: '0xbb', v: 27 }),
    signerAddress: '0xabcDEF1234567890abcdef1234567890aBcDef12',
    autoApprove: false,
  });
}

afterEach(() => {
  delete process.env.PRIVATE_KEY;
});

describe('vaultAddress / expiresAfter threading', () => {
  it('sends both fields in the build AND send payloads for cancel', async () => {
    const sdk = makeSdk();
    const { calls, restore } = mockExchangeFetch();
    try {
      await sdk.cancel(42, 0, { vaultAddress: VAULT, expiresAfter: EXPIRES });
    } finally {
      restore();
    }

    expect(calls).toHaveLength(2);
    // Build payload — the worker hashes these alongside the action.
    expect(calls[0].vaultAddress).toBe(VAULT);
    expect(calls[0].expiresAfter).toBe(EXPIRES);
    expect('signature' in calls[0]).toBe(false);
    // Send payload — signer recovery + forwarded exchange body.
    expect(calls[1].vaultAddress).toBe(VAULT);
    expect(calls[1].expiresAfter).toBe(EXPIRES);
    expect('signature' in calls[1]).toBe(true);
  });

  it('never emits the fields when unset (wire compat)', async () => {
    const sdk = makeSdk();
    const { calls, restore } = mockExchangeFetch();
    try {
      await sdk.cancel(42, 0);
    } finally {
      restore();
    }

    for (const call of calls) {
      expect('vaultAddress' in call).toBe(false);
      expect('expiresAfter' in call).toBe(false);
    }
  });

  it('threads buy() options into both exchange payloads', async () => {
    const sdk = makeSdk();
    const { calls, restore } = mockExchangeFetch();
    try {
      await sdk.buy('BTC', {
        size: 0.001,
        price: 50000,
        tif: 'gtc',
        vaultAddress: VAULT,
        expiresAfter: EXPIRES,
      });
    } finally {
      restore();
    }

    expect(calls).toHaveLength(2);
    expect(calls[0].vaultAddress).toBe(VAULT);
    expect(calls[0].expiresAfter).toBe(EXPIRES);
    expect(calls[1].vaultAddress).toBe(VAULT);
    expect(calls[1].expiresAfter).toBe(EXPIRES);
  });

  it('threads Order builder .vaultAddress()/.expiresAfter() through order()', async () => {
    const sdk = makeSdk();
    const order = Order.buy('BTC')
      .size(0.001)
      .price(50000)
      .gtc()
      .vaultAddress(VAULT)
      .expiresAfter(EXPIRES);

    const { calls, restore } = mockExchangeFetch();
    try {
      await sdk.order(order);
    } finally {
      restore();
    }

    expect(calls[0].vaultAddress).toBe(VAULT);
    expect(calls[0].expiresAfter).toBe(EXPIRES);
    expect(calls[1].vaultAddress).toBe(VAULT);
    expect(calls[1].expiresAfter).toBe(EXPIRES);
  });

  it('threads modify() options into both exchange payloads', async () => {
    const sdk = makeSdk();
    const { calls, restore } = mockExchangeFetch();
    try {
      await sdk.modify(42, 'BTC', 'buy', '50000', '0.001', {
        vaultAddress: VAULT,
        expiresAfter: EXPIRES,
      });
    } finally {
      restore();
    }

    expect(calls[0].vaultAddress).toBe(VAULT);
    expect(calls[0].expiresAfter).toBe(EXPIRES);
    expect(calls[1].vaultAddress).toBe(VAULT);
    expect(calls[1].expiresAfter).toBe(EXPIRES);
  });

  it('closePosition() targets the vault position and threads the fields', async () => {
    const sdk = makeSdk();
    const { calls, restore } = mockExchangeFetch();
    try {
      await sdk.closePosition('BTC', { vaultAddress: VAULT, expiresAfter: EXPIRES });
    } finally {
      restore();
    }

    // The worker queries action.user to size the close — must be the vault.
    expect(calls[0].action.user).toBe(VAULT);
    expect(calls[0].vaultAddress).toBe(VAULT);
    expect(calls[0].expiresAfter).toBe(EXPIRES);
    expect(calls[1].vaultAddress).toBe(VAULT);
    expect(calls[1].expiresAfter).toBe(EXPIRES);
  });

  it('cancelAll() enumerates the vault orders, not the wallet', async () => {
    const sdk = makeSdk();
    const openOrdersUsers: Array<string | undefined> = [];
    sdk.openOrders = async (user?: string) => {
      openOrdersUsers.push(user);
      return {
        orders: [{ oid: 1 }],
        cancelActions: { all: { type: 'cancel', cancels: [{ a: 0, o: 1 }] } },
      };
    };
    const { calls, restore } = mockExchangeFetch();
    try {
      await sdk.cancelAll(undefined, { vaultAddress: VAULT });
    } finally {
      restore();
    }

    expect(openOrdersUsers).toEqual([VAULT]);
    expect(calls[0].vaultAddress).toBe(VAULT);
  });

  it('rejects a non-integer expiresAfter before any network call', async () => {
    const sdk = makeSdk();
    const { calls, restore } = mockExchangeFetch();
    let err: unknown;
    try {
      err = await sdk.cancel(42, 0, { expiresAfter: 1.5 }).catch((e: unknown) => e);
    } finally {
      restore();
    }

    expect(err).toBeInstanceOf(ValidationError);
    expect(calls).toHaveLength(0);
  });
});

describe('fast cancel flag', () => {
  it('emits action-level f: true on cancel when fast is set', async () => {
    const sdk = makeSdk();
    const { calls, restore } = mockExchangeFetch();
    try {
      await sdk.cancel(42, 0, { fast: true });
    } finally {
      restore();
    }

    expect(calls[0].action).toEqual({
      type: 'cancel',
      cancels: [{ a: 0, o: 42 }],
      f: true,
    });
    expect(calls[1].action.f).toBe(true);
  });

  it.each([
    ['unset', {}],
    ['false', { fast: false }],
  ])('never emits f when fast is %s (backend strips f:false)', async (_label, opts) => {
    const sdk = makeSdk();
    const { calls, restore } = mockExchangeFetch();
    try {
      await sdk.cancel(42, 0, opts as { fast?: boolean });
    } finally {
      restore();
    }

    for (const call of calls) {
      expect('f' in call.action).toBe(false);
    }
  });

  it('emits f: true on cancelByCloid when fast is set', async () => {
    const sdk = makeSdk();
    const cloid = '0x' + '12'.repeat(16);
    const { calls, restore } = mockExchangeFetch();
    try {
      await sdk.cancelByCloid(cloid, 0, { fast: true });
    } finally {
      restore();
    }

    expect(calls[0].action).toEqual({
      type: 'cancelByCloid',
      cancels: [{ asset: 0, cloid }],
      f: true,
    });
  });

  it('omits f on cancelByCloid when fast is unset', async () => {
    const sdk = makeSdk();
    const { calls, restore } = mockExchangeFetch();
    try {
      await sdk.cancelByCloid('0x' + '12'.repeat(16), 0);
    } finally {
      restore();
    }

    for (const call of calls) {
      expect('f' in call.action).toBe(false);
    }
  });
});
