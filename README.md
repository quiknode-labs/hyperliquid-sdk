# Hyperliquid SDK

> **Community SDKs by [QuickNode](https://quicknode.com)** — Not affiliated with Hyperliquid Foundation.

**The simplest way to trade on Hyperliquid.** One line to place orders, zero ceremony.

## Available SDKs

| Language | Package | Status |
|----------|---------|--------|
| [Python](./python/) | `pip install hyperliquid-sdk` | Available |
| [TypeScript](./typescript/) | `npm install hyperliquid-sdk` | Coming soon |
| [Rust](./rust/) | `cargo add hyperliquid-sdk` | Coming soon |

## Quick Example

### Python
```python
from hyperliquid_sdk import HyperliquidSDK

sdk = HyperliquidSDK()
order = sdk.market_buy("BTC", notional=100)  # Buy $100 of BTC
```

### TypeScript (Coming Soon)
```typescript
import { HyperliquidSDK } from 'hyperliquid-sdk';

const sdk = new HyperliquidSDK();
const order = await sdk.marketBuy("BTC", { notional: 100 });
```

### Rust (Coming Soon)
```rust
use hyperliquid_sdk::HyperliquidSDK;

let sdk = HyperliquidSDK::new()?;
let order = sdk.market_buy("BTC").notional(100.0).send().await?;
```

## Features

All SDKs share the same design philosophy:

- **One-line orders** — No build-sign-send ceremony
- **Size or notional** — Specify size in asset units or USD
- **Order management** — Modify, cancel, track orders
- **Position management** — Close positions with one call
- **Clear errors** — Actionable error messages with guidance
- **HIP-3 support** — Same API for HIP-3 markets

## Links

- **Documentation**: https://hyperliquidapi.com/docs
- **Examples**: https://github.com/quiknode-labs/hyperliquid-examples

## Disclaimer

These are **unofficial community SDKs** developed and maintained by [QuickNode](https://quicknode.com). They are **not affiliated with, endorsed by, or associated with Hyperliquid Foundation or Hyperliquid Labs**.

Use at your own risk. Always review transactions before signing. QuickNode is not responsible for any losses incurred through use of these SDKs.

## License

MIT License - see [LICENSE](LICENSE) for details.
