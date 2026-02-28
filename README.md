# Hyperliquid SDK

> **Community SDKs by [QuickNode](https://quicknode.com)** — Not affiliated with Hyperliquid Foundation.

**The simplest way to trade on Hyperliquid.** One line to place orders, zero ceremony.

## Available SDKs

| Language | Package | Registry |
|----------|---------|----------|
| [Python](./python/) | `pip install hyperliquid-sdk` | [PyPI](https://pypi.org/project/hyperliquid-sdk/) |
| [TypeScript](./typescript/) | `npm install @quicknode/hyperliquid-sdk` | [npm](https://www.npmjs.com/package/@quicknode/hyperliquid-sdk) |
| [Rust](./rust/) | `cargo add quicknode-hyperliquid-sdk` | [crates.io](https://crates.io/crates/quicknode-hyperliquid-sdk) |
| [Go](./go/) | `go get github.com/quiknode-labs/hyperliquid-sdk/go` | [GitHub](https://github.com/quiknode-labs/hyperliquid-sdk) |

## Quick Example

### Python
```python
from hyperliquid_sdk import HyperliquidSDK

sdk = HyperliquidSDK(endpoint)
order = sdk.market_buy("BTC", notional=100)  # Buy $100 of BTC
```

### TypeScript
```typescript
import { HyperliquidSDK } from '@quicknode/hyperliquid-sdk';

const sdk = new HyperliquidSDK(endpoint);
const order = await sdk.marketBuy("BTC", { notional: 100 });
```

### Rust
```rust
use hyperliquid_sdk::HyperliquidSDK;

let sdk = HyperliquidSDK::new().endpoint(endpoint).build().await?;
let order = sdk.market_buy("BTC").await.notional(100.0).await?;
```

### Go
```go
import "github.com/quiknode-labs/hyperliquid-sdk/go/hyperliquid"

sdk, _ := hyperliquid.New(endpoint, hyperliquid.WithPrivateKey(privateKey))
order, _ := sdk.MarketBuy("BTC", hyperliquid.WithNotional(100))
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

- **Documentation**: https://hyperliquidapi.com
- **Examples**: https://github.com/quiknode-labs/hyperliquid-examples

## Disclaimer

These are **unofficial community SDKs** developed and maintained by [QuickNode](https://quicknode.com). They are **not affiliated with, endorsed by, or associated with Hyperliquid Foundation or Hyperliquid Labs**.

Use at your own risk. Always review transactions before signing. QuickNode is not responsible for any losses incurred through use of these SDKs.

## License

MIT License - see [LICENSE](LICENSE) for details.
