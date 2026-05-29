"""
Tests for the external-signer feature (WithSigner-equivalent).

Run with: pytest tests/test_signer.py -v

Mirrors the Go SDK's signer_test.go / buildsignsend_test.go. The build/sign/send
ceremony is exercised by monkeypatching `sdk._exchange` (the only network hop),
following the pattern in test_sdk.py.
"""

import pytest

from hyperliquid_sdk import HyperliquidSDK, Signer, SignerError


def test_signer_is_importable_and_typed():
    # Signer is a public type alias callers annotate against.
    assert Signer is not None


def test_construct_with_signer_only_is_sign_capable(monkeypatch):
    monkeypatch.delenv("PRIVATE_KEY", raising=False)

    def signer(hash_hex):
        return {"r": "0x1", "s": "0x2", "v": 27}

    sdk = HyperliquidSDK(signer=signer, auto_approve=False)

    assert sdk._signer is signer
    assert sdk._wallet is None  # no in-process wallet under an external signer
    # _require_wallet must be satisfied by the signer (no raise).
    sdk._require_wallet()


def test_signer_address_backfills_address(monkeypatch):
    monkeypatch.delenv("PRIVATE_KEY", raising=False)
    addr = "0x1234567890123456789012345678901234567890"

    sdk = HyperliquidSDK(
        signer=lambda h: {"r": "0x1", "s": "0x2", "v": 27},
        signer_address=addr,
        auto_approve=False,
    )
    assert sdk.address == addr


def test_signer_never_reads_private_key_env(monkeypatch):
    # Even with PRIVATE_KEY set, a signer means no wallet is created.
    monkeypatch.setenv("PRIVATE_KEY", "0x" + "11" * 32)

    sdk = HyperliquidSDK(
        signer=lambda h: {"r": "0x1", "s": "0x2", "v": 27},
        auto_approve=False,
    )
    assert sdk._wallet is None


def test_build_sign_send_routes_through_signer(monkeypatch):
    monkeypatch.delenv("PRIVATE_KEY", raising=False)

    seen_hash = {}

    def signer(hash_hex):
        seen_hash["value"] = hash_hex
        return {"r": "0xaa", "s": "0xbb", "v": 28}

    sdk = HyperliquidSDK(signer=signer, auto_approve=False)

    calls = []

    def fake_exchange(body):
        calls.append(body)
        if "signature" not in body:
            return {"hash": "0x" + "00" * 32, "nonce": 123, "action": body["action"]}
        return {"success": True}

    sdk._exchange = fake_exchange
    sdk._build_sign_send({"type": "order"})

    assert seen_hash["value"] == "0x" + "00" * 32
    # The signer's signature must be forwarded in the send payload.
    assert calls[1]["signature"] == {"r": "0xaa", "s": "0xbb", "v": 28}


def test_signer_failure_raises_signer_error_not_signature_error(monkeypatch):
    monkeypatch.delenv("PRIVATE_KEY", raising=False)

    def boom(hash_hex):
        raise RuntimeError("kms unavailable")

    sdk = HyperliquidSDK(signer=boom, auto_approve=False)

    def fake_exchange(body):
        return {"hash": "0x" + "00" * 32, "nonce": 1, "action": body["action"]}

    sdk._exchange = fake_exchange

    with pytest.raises(SignerError) as exc_info:
        sdk._build_sign_send({"type": "order"})

    assert exc_info.value.code == "SIGNER_FAILED"
    # The original cause is chained for inspection.
    assert isinstance(exc_info.value.__cause__, RuntimeError)


def test_auto_approve_skipped_under_external_signer(monkeypatch):
    monkeypatch.delenv("PRIVATE_KEY", raising=False)
    approvals = []

    sdk = HyperliquidSDK(
        signer=lambda h: {"r": "0x1", "s": "0x2", "v": 27},
        auto_approve=True,  # would normally try to approve on init
    )
    # No wallet → _ensure_approved was never invoked at construction.
    sdk._ensure_approved = lambda *a, **k: approvals.append(a)
    assert approvals == []
