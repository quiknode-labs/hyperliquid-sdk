"""
HyperEVM Client — Ethereum JSON-RPC for Hyperliquid's EVM.

Standard Ethereum JSON-RPC methods plus debug/trace capabilities.

Example:
    >>> from hyperliquid_sdk import EVM
    >>> evm = EVM("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")
    >>> print(evm.block_number())
    >>> print(evm.get_balance("0x..."))
"""

from __future__ import annotations
from typing import Optional, List, Dict, Any, Union
from urllib.parse import urlparse

import requests

from .errors import HyperliquidError


class EVM:
    """
    HyperEVM Client — Ethereum JSON-RPC for Hyperliquid's EVM.

    Standard Ethereum methods plus debug/trace APIs (nanoreth path).

    Examples:
        evm = EVM("https://your-endpoint.hype-mainnet.quiknode.pro/TOKEN")

        # Standard methods
        evm.block_number()
        evm.chain_id()
        evm.get_balance("0x...")
        evm.get_block_by_number(12345)

        # Debug/Trace (mainnet only)
        evm = EVM(endpoint, debug=True)
        evm.trace_transaction("0x...")
    """

    def __init__(self, endpoint: str, *, debug: bool = False, timeout: int = 30):
        """
        Initialize the EVM client.

        Args:
            endpoint: Hyperliquid endpoint URL
            debug: Use nanoreth path for debug/trace APIs (mainnet only)
            timeout: Request timeout in seconds (default: 30)
        """
        self._base_url = self._build_url(endpoint, debug)
        self._timeout = timeout
        self._session = requests.Session()
        self._request_id = 0

    def _build_url(self, url: str, use_nanoreth: bool) -> str:
        """Build the RPC URL with appropriate path."""
        parsed = urlparse(url)
        base = f"{parsed.scheme}://{parsed.netloc}"
        path_parts = parsed.path.strip("/").split("/")
        token = ""
        for part in path_parts:
            if part not in ("info", "hypercore", "evm", "nanoreth"):
                token = part
                break
        path = "nanoreth" if use_nanoreth else "evm"
        if token:
            return f"{base}/{token}/{path}"
        return f"{base}/{path}"

    def _rpc(self, method: str, params: Optional[List[Any]] = None) -> Any:
        """Make a JSON-RPC call."""
        self._request_id += 1
        payload = {"jsonrpc": "2.0", "method": method, "params": params or [], "id": self._request_id}
        resp = self._session.post(self._base_url, json=payload, timeout=self._timeout)
        if resp.status_code != 200:
            raise HyperliquidError(f"Request failed: {resp.status_code}", code="HTTP_ERROR", raw={"body": resp.text})
        data = resp.json()
        if "error" in data:
            error = data["error"]
            raise HyperliquidError(error.get("message", "RPC error"), code=str(error.get("code", "RPC_ERROR")), raw=data)
        return data.get("result")

    # ═══════════════════════════════════════════════════════════════════════════
    # STANDARD ETHEREUM METHODS
    # ═══════════════════════════════════════════════════════════════════════════

    def block_number(self) -> int:
        """Get the current block number."""
        return int(self._rpc("eth_blockNumber"), 16)

    def chain_id(self) -> int:
        """Get the chain ID (999 mainnet, 998 testnet)."""
        return int(self._rpc("eth_chainId"), 16)

    def gas_price(self) -> int:
        """Get the current gas price in wei."""
        return int(self._rpc("eth_gasPrice"), 16)

    def get_balance(self, address: str, block: Union[int, str] = "latest") -> int:
        """Get account balance in wei."""
        block_param = block if isinstance(block, str) else hex(block)
        return int(self._rpc("eth_getBalance", [address, block_param]), 16)

    def get_transaction_count(self, address: str, block: Union[int, str] = "latest") -> int:
        """Get the nonce (transaction count) for an address."""
        block_param = block if isinstance(block, str) else hex(block)
        return int(self._rpc("eth_getTransactionCount", [address, block_param]), 16)

    def get_code(self, address: str, block: Union[int, str] = "latest") -> str:
        """Get the contract bytecode at an address."""
        block_param = block if isinstance(block, str) else hex(block)
        return self._rpc("eth_getCode", [address, block_param])

    def get_storage_at(self, address: str, position: str, block: Union[int, str] = "latest") -> str:
        """Get storage at a specific position."""
        block_param = block if isinstance(block, str) else hex(block)
        return self._rpc("eth_getStorageAt", [address, position, block_param])

    def call(self, tx: Dict[str, Any], block: Union[int, str] = "latest") -> str:
        """Execute a read-only call."""
        block_param = block if isinstance(block, str) else hex(block)
        return self._rpc("eth_call", [tx, block_param])

    def estimate_gas(self, tx: Dict[str, Any]) -> int:
        """Estimate gas for a transaction."""
        return int(self._rpc("eth_estimateGas", [tx]), 16)

    def send_raw_transaction(self, signed_tx: str) -> str:
        """Submit a signed transaction."""
        return self._rpc("eth_sendRawTransaction", [signed_tx])

    def get_transaction_by_hash(self, tx_hash: str) -> Optional[Dict[str, Any]]:
        """Get transaction by hash."""
        return self._rpc("eth_getTransactionByHash", [tx_hash])

    def get_transaction_receipt(self, tx_hash: str) -> Optional[Dict[str, Any]]:
        """Get transaction receipt."""
        return self._rpc("eth_getTransactionReceipt", [tx_hash])

    def get_block_by_number(self, block_number: Union[int, str], full_transactions: bool = False) -> Optional[Dict[str, Any]]:
        """Get block by number."""
        block_param = block_number if isinstance(block_number, str) else hex(block_number)
        return self._rpc("eth_getBlockByNumber", [block_param, full_transactions])

    def get_block_by_hash(self, block_hash: str, full_transactions: bool = False) -> Optional[Dict[str, Any]]:
        """Get block by hash."""
        return self._rpc("eth_getBlockByHash", [block_hash, full_transactions])

    def get_logs(self, filter_params: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Get logs matching filter (max 4 topics, 50 block range)."""
        return self._rpc("eth_getLogs", [filter_params])

    # ═══════════════════════════════════════════════════════════════════════════
    # DEBUG/TRACE METHODS (requires debug=True, mainnet only)
    # ═══════════════════════════════════════════════════════════════════════════

    def debug_trace_transaction(self, tx_hash: str, tracer_config: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Trace a transaction's execution."""
        params: List[Any] = [tx_hash]
        if tracer_config:
            params.append(tracer_config)
        return self._rpc("debug_traceTransaction", params)

    def trace_transaction(self, tx_hash: str) -> List[Dict[str, Any]]:
        """Get trace of a transaction."""
        return self._rpc("trace_transaction", [tx_hash])

    def trace_block(self, block_number: Union[int, str]) -> List[Dict[str, Any]]:
        """Get traces of all transactions in a block."""
        block_param = block_number if isinstance(block_number, str) else hex(block_number)
        return self._rpc("trace_block", [block_param])

    # ═══════════════════════════════════════════════════════════════════════════
    # ADDITIONAL STANDARD METHODS
    # ═══════════════════════════════════════════════════════════════════════════

    def net_version(self) -> str:
        """Get the network version."""
        return self._rpc("net_version")

    def web3_client_version(self) -> str:
        """Get the client version."""
        return self._rpc("web3_clientVersion")

    def syncing(self) -> Union[bool, Dict[str, Any]]:
        """Check if the node is syncing."""
        return self._rpc("eth_syncing")

    def accounts(self) -> List[str]:
        """Get list of accounts (usually empty for remote nodes)."""
        return self._rpc("eth_accounts")

    def fee_history(
        self, block_count: int, newest_block: Union[int, str], reward_percentiles: Optional[List[float]] = None
    ) -> Dict[str, Any]:
        """Get fee history for a range of blocks."""
        block_param = newest_block if isinstance(newest_block, str) else hex(newest_block)
        params: List[Any] = [hex(block_count), block_param]
        if reward_percentiles:
            params.append(reward_percentiles)
        return self._rpc("eth_feeHistory", params)

    def max_priority_fee_per_gas(self) -> int:
        """Get max priority fee per gas."""
        return int(self._rpc("eth_maxPriorityFeePerGas"), 16)

    def get_block_receipts(self, block_number: Union[int, str]) -> List[Dict[str, Any]]:
        """Get all receipts for a block."""
        block_param = block_number if isinstance(block_number, str) else hex(block_number)
        return self._rpc("eth_getBlockReceipts", [block_param])

    def get_block_transaction_count_by_hash(self, block_hash: str) -> int:
        """Get transaction count in a block by hash."""
        return int(self._rpc("eth_getBlockTransactionCountByHash", [block_hash]), 16)

    def get_block_transaction_count_by_number(self, block_number: Union[int, str]) -> int:
        """Get transaction count in a block by number."""
        block_param = block_number if isinstance(block_number, str) else hex(block_number)
        return int(self._rpc("eth_getBlockTransactionCountByNumber", [block_param]), 16)

    def get_transaction_by_block_hash_and_index(self, block_hash: str, index: int) -> Optional[Dict[str, Any]]:
        """Get transaction by block hash and index."""
        return self._rpc("eth_getTransactionByBlockHashAndIndex", [block_hash, hex(index)])

    def get_transaction_by_block_number_and_index(
        self, block_number: Union[int, str], index: int
    ) -> Optional[Dict[str, Any]]:
        """Get transaction by block number and index."""
        block_param = block_number if isinstance(block_number, str) else hex(block_number)
        return self._rpc("eth_getTransactionByBlockNumberAndIndex", [block_param, hex(index)])

    # ═══════════════════════════════════════════════════════════════════════════
    # HYPERLIQUID-SPECIFIC EVM METHODS
    # ═══════════════════════════════════════════════════════════════════════════

    def big_block_gas_price(self) -> int:
        """Get gas price for big blocks."""
        return int(self._rpc("eth_bigBlockGasPrice"), 16)

    def using_big_blocks(self) -> bool:
        """Check if using big blocks."""
        return self._rpc("eth_usingBigBlocks")

    def get_system_txs_by_block_hash(self, block_hash: str) -> List[Dict[str, Any]]:
        """Get system transactions by block hash."""
        return self._rpc("eth_getSystemTxsByBlockHash", [block_hash])

    def get_system_txs_by_block_number(self, block_number: Union[int, str]) -> List[Dict[str, Any]]:
        """Get system transactions by block number."""
        block_param = block_number if isinstance(block_number, str) else hex(block_number)
        return self._rpc("eth_getSystemTxsByBlockNumber", [block_param])

    # ═══════════════════════════════════════════════════════════════════════════
    # ADDITIONAL DEBUG METHODS
    # ═══════════════════════════════════════════════════════════════════════════

    def debug_get_bad_blocks(self) -> List[Dict[str, Any]]:
        """Get bad blocks."""
        return self._rpc("debug_getBadBlocks")

    def debug_get_raw_block(self, block_number: Union[int, str]) -> str:
        """Get raw block data."""
        block_param = block_number if isinstance(block_number, str) else hex(block_number)
        return self._rpc("debug_getRawBlock", [block_param])

    def debug_get_raw_header(self, block_number: Union[int, str]) -> str:
        """Get raw block header."""
        block_param = block_number if isinstance(block_number, str) else hex(block_number)
        return self._rpc("debug_getRawHeader", [block_param])

    def debug_get_raw_receipts(self, block_number: Union[int, str]) -> List[str]:
        """Get raw receipts for a block."""
        block_param = block_number if isinstance(block_number, str) else hex(block_number)
        return self._rpc("debug_getRawReceipts", [block_param])

    def debug_get_raw_transaction(self, tx_hash: str) -> str:
        """Get raw transaction data."""
        return self._rpc("debug_getRawTransaction", [tx_hash])

    def debug_storage_range_at(
        self,
        block_hash: str,
        tx_index: int,
        contract_address: str,
        key_start: str,
        max_result: int,
    ) -> Dict[str, Any]:
        """Get storage range at a specific point."""
        return self._rpc("debug_storageRangeAt", [block_hash, tx_index, contract_address, key_start, max_result])

    def debug_trace_block(self, block_rlp: str, tracer_config: Optional[Dict[str, Any]] = None) -> List[Dict[str, Any]]:
        """Trace a block by RLP."""
        params: List[Any] = [block_rlp]
        if tracer_config:
            params.append(tracer_config)
        return self._rpc("debug_traceBlock", params)

    def debug_trace_block_by_hash(
        self, block_hash: str, tracer_config: Optional[Dict[str, Any]] = None
    ) -> List[Dict[str, Any]]:
        """Trace a block by hash."""
        params: List[Any] = [block_hash]
        if tracer_config:
            params.append(tracer_config)
        return self._rpc("debug_traceBlockByHash", params)

    def debug_trace_block_by_number(
        self, block_number: Union[int, str], tracer_config: Optional[Dict[str, Any]] = None
    ) -> List[Dict[str, Any]]:
        """Trace a block by number."""
        block_param = block_number if isinstance(block_number, str) else hex(block_number)
        params: List[Any] = [block_param]
        if tracer_config:
            params.append(tracer_config)
        return self._rpc("debug_traceBlockByNumber", params)

    def debug_trace_call(
        self, tx: Dict[str, Any], block: Union[int, str] = "latest", tracer_config: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """Trace a call."""
        block_param = block if isinstance(block, str) else hex(block)
        params: List[Any] = [tx, block_param]
        if tracer_config:
            params.append(tracer_config)
        return self._rpc("debug_traceCall", params)

    # ═══════════════════════════════════════════════════════════════════════════
    # ADDITIONAL TRACE METHODS
    # ═══════════════════════════════════════════════════════════════════════════

    def trace_call(
        self, tx: Dict[str, Any], trace_types: List[str], block: Union[int, str] = "latest"
    ) -> Dict[str, Any]:
        """Trace a call with specified trace types."""
        block_param = block if isinstance(block, str) else hex(block)
        return self._rpc("trace_call", [tx, trace_types, block_param])

    def trace_call_many(
        self, calls: List[Tuple[Dict[str, Any], List[str]]], block: Union[int, str] = "latest"
    ) -> List[Dict[str, Any]]:
        """Trace multiple calls."""
        block_param = block if isinstance(block, str) else hex(block)
        return self._rpc("trace_callMany", [calls, block_param])

    def trace_filter(self, filter_params: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Filter traces."""
        return self._rpc("trace_filter", [filter_params])

    def trace_raw_transaction(self, raw_tx: str, trace_types: List[str]) -> Dict[str, Any]:
        """Trace a raw transaction."""
        return self._rpc("trace_rawTransaction", [raw_tx, trace_types])

    def trace_replay_block_transactions(
        self, block_number: Union[int, str], trace_types: List[str]
    ) -> List[Dict[str, Any]]:
        """Replay and trace all transactions in a block."""
        block_param = block_number if isinstance(block_number, str) else hex(block_number)
        return self._rpc("trace_replayBlockTransactions", [block_param, trace_types])

    def trace_replay_transaction(self, tx_hash: str, trace_types: List[str]) -> Dict[str, Any]:
        """Replay and trace a transaction."""
        return self._rpc("trace_replayTransaction", [tx_hash, trace_types])

    def __enter__(self) -> EVM:
        return self

    def __exit__(self, *args) -> None:
        self._session.close()

    def __repr__(self) -> str:
        return f"<EVM {self._base_url[:40]}...>"
