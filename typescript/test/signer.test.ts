/**
 * Tests for the external-signer feature (WithSigner-equivalent).
 *
 * The build→sign→send ceremony is exercised by mocking global fetch (the only
 * network hop). Private members are reached via `as any`, mirroring the Go
 * SDK's white-box buildsignsend_test.go.
 */

import { describe, it, expect, vi, afterEach } from 'vitest';
import { HyperliquidSDK, Signer, Signature } from '../src/client';
import { SignerError, SignatureError } from '../src/errors';

const HASH = '0x' + '00'.repeat(32);

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

afterEach(() => {
  delete process.env.PRIVATE_KEY;
});

describe('external signer', () => {
  it('constructs with a signer only and exposes signerAddress', () => {
    delete process.env.PRIVATE_KEY;
    const addr = '0x1234567890123456789012345678901234567890';
    const sdk = new HyperliquidSDK('https://x.quiknode.pro/T', {
      signer: () => ({ r: '0x1', s: '0x2', v: 27 }),
      signerAddress: addr,
    });
    expect(sdk.address).toBe(addr);
    expect((sdk as any)._wallet).toBeNull();
    expect((sdk as any)._signer).not.toBeNull();
  });

  it('never reads PRIVATE_KEY env when a signer is set', () => {
    process.env.PRIVATE_KEY = '0x' + '11'.repeat(32);
    const sdk = new HyperliquidSDK('https://x.quiknode.pro/T', {
      signer: () => ({ r: '0x1', s: '0x2', v: 27 }),
    });
    expect((sdk as any)._wallet).toBeNull();
  });

  it('routes the build hash through the signer and forwards r/s/v', async () => {
    delete process.env.PRIVATE_KEY;
    let seenHash: string | undefined;
    const signer: Signer = (hashHex) => {
      seenHash = hashHex;
      return { r: '0xaa', s: '0xbb', v: 28 };
    };
    const sdk = new HyperliquidSDK('https://x.quiknode.pro/T', { signer, autoApprove: false });

    const { calls, restore } = mockExchangeFetch();
    try {
      await (sdk as any)._buildSignSend({ type: 'order' });
    } finally {
      restore();
    }

    expect(seenHash).toBe(HASH);
    expect(calls).toHaveLength(2);
    expect(calls[1].signature).toEqual({ r: '0xaa', s: '0xbb', v: 28 });
  });

  it('passes an AbortSignal to the signer', async () => {
    delete process.env.PRIVATE_KEY;
    let gotSignal: AbortSignal | undefined;
    const signer: Signer = (_hash, opts) => {
      gotSignal = opts?.signal;
      return { r: '0x1', s: '0x2', v: 27 };
    };
    const sdk = new HyperliquidSDK('https://x.quiknode.pro/T', { signer, autoApprove: false });

    const { restore } = mockExchangeFetch();
    try {
      await (sdk as any)._buildSignSend({ type: 'order' });
    } finally {
      restore();
    }
    expect(gotSignal).toBeInstanceOf(AbortSignal);
  });

  it('wraps a signer failure in SignerError (not SignatureError) with cause', async () => {
    delete process.env.PRIVATE_KEY;
    const boom = new Error('kms unavailable');
    const signer: Signer = () => { throw boom; };
    const sdk = new HyperliquidSDK('https://x.quiknode.pro/T', { signer, autoApprove: false });

    const { restore } = mockExchangeFetch();
    let err: unknown;
    try {
      err = await (sdk as any)._buildSignSend({ type: 'order' }).catch((e: unknown) => e);
    } finally {
      restore();
    }

    expect(err).toBeInstanceOf(SignerError);
    expect(err).not.toBeInstanceOf(SignatureError);
    expect((err as SignerError).code).toBe('SIGNER_FAILED');
    expect((err as Error).cause).toBe(boom);
  });

  it('skips builder-fee auto-approve under an external signer', async () => {
    delete process.env.PRIVATE_KEY;
    const sdk = new HyperliquidSDK('https://x.quiknode.pro/T', {
      signer: () => ({ r: '0x1', s: '0x2', v: 27 }),
      autoApprove: true, // would normally approve before the first trade
    });

    const { calls, restore } = mockExchangeFetch();
    try {
      await (sdk as any)._buildSignSend({ type: 'order' });
    } finally {
      restore();
    }
    // Only build + send — no approval round-trip was attempted.
    expect(calls).toHaveLength(2);
  });
});
