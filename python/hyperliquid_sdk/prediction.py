"""HIP-4 prediction market helpers."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Iterable, Optional


def parse_outcome_description(description: str) -> dict[str, str]:
    """Parse Hyperliquid's outcome description string into key/value fields."""
    fields: dict[str, str] = {}
    for part in description.split("|"):
        if ":" not in part:
            continue
        key, value = part.split(":", 1)
        fields[key] = value
    return fields


def format_expiry(expiry: Optional[str]) -> Optional[str]:
    """Convert YYYYMMDD-HHMM into a readable UTC timestamp."""
    if not expiry or len(expiry) != 13 or expiry[8] != "-":
        return expiry
    return f"{expiry[:4]}-{expiry[4:6]}-{expiry[6:8]}T{expiry[9:11]}:{expiry[11:13]}:00Z"


def title_from_fields(fields: dict[str, str]) -> str:
    underlying = fields.get("underlying", "Outcome")
    target_price = fields.get("targetPrice")
    expiry = format_expiry(fields.get("expiry"))

    if target_price and expiry:
        return f"{underlying} above {target_price} on {expiry}"
    if target_price:
        return f"{underlying} above {target_price}"
    return underlying


def slugify(value: str) -> str:
    chars = []
    for ch in value.lower():
        if ch.isalnum():
            chars.append(ch)
        elif chars and chars[-1] != "-":
            chars.append("-")
    return "".join(chars).strip("-")


# Only used to generate app-style slug aliases such as
# btc-above-78213-yes-may-04-0600 from outcomeMeta expiry 20260504-0600.
APP_STYLE_PREDICTION_SLUG_MONTHS = {
    "01": "jan",
    "02": "feb",
    "03": "mar",
    "04": "apr",
    "05": "may",
    "06": "jun",
    "07": "jul",
    "08": "aug",
    "09": "sep",
    "10": "oct",
    "11": "nov",
    "12": "dec",
}


def app_style_prediction_slug(fields: dict[str, str], side: Optional[str] = None) -> Optional[str]:
    underlying = fields.get("underlying")
    target_price = fields.get("targetPrice")
    expiry = fields.get("expiry")
    if not underlying or not target_price or not expiry or len(expiry) != 13:
        return None
    month = APP_STYLE_PREDICTION_SLUG_MONTHS.get(expiry[4:6])
    if month is None:
        return None
    parts = [underlying, "above", target_price]
    if side:
        parts.append(side)
    parts.extend([month, expiry[6:8], expiry[9:13]])
    return slugify("-".join(parts))


@dataclass(frozen=True)
class PredictionSide:
    """A tradeable HIP-4 outcome side."""

    outcome: int
    side: int
    name: str
    symbol: str
    token: str
    asset_id: int
    mid: Optional[str] = None

    @property
    def sz_decimals(self) -> int:
        return 0

    @property
    def supports_priority_fee(self) -> bool:
        return False

    def __str__(self) -> str:
        return self.symbol


@dataclass(frozen=True)
class PredictionMarket:
    """A HIP-4 prediction market with yes/no tradeable sides."""

    outcome: int
    name: str
    description: str
    title: str
    slug: str
    underlying: Optional[str]
    target_price: Optional[str]
    expiry: Optional[str]
    period: Optional[str]
    collateral: str
    min_order_value: str
    aliases: tuple[str, ...]
    yes: PredictionSide
    no: PredictionSide
    sides: tuple[PredictionSide, ...]

    def matches(self, query: str) -> bool:
        normalized = query.lower()
        values = [
            self.slug,
            self.title.lower(),
            self.name.lower(),
            self.underlying.lower() if self.underlying else "",
            self.yes.symbol.lower(),
            self.no.symbol.lower(),
            self.yes.token.lower(),
            self.no.token.lower(),
            *self.aliases,
        ]
        return normalized in values or any(normalized in value for value in values if value)


class PredictionMarkets(list[PredictionMarket]):
    """List wrapper with a small, readable finder."""

    def find(
        self,
        query: Optional[str] = None,
        *,
        underlying: Optional[str] = None,
        target_price: Optional[str] = None,
        expiry: Optional[str] = None,
    ) -> Optional[PredictionMarket]:
        for market in self:
            if query is not None and not market.matches(query):
                continue
            if underlying is not None and (market.underlying or "").lower() != underlying.lower():
                continue
            if target_price is not None and market.target_price != str(target_price):
                continue
            if expiry is not None and market.expiry not in {expiry, format_expiry(expiry)}:
                continue
            return market
        return None


def build_prediction_markets(outcomes: Iterable[dict[str, Any]], mids: dict[str, Any]) -> PredictionMarkets:
    markets = PredictionMarkets()
    for outcome in outcomes:
        outcome_id = int(outcome["outcome"])
        description = str(outcome.get("description", ""))
        fields = parse_outcome_description(description)
        title = title_from_fields(fields)
        side_specs = outcome.get("sideSpecs", [])

        sides = []
        for side_index, side_spec in enumerate(side_specs):
            encoding = outcome_id * 10 + side_index
            symbol = f"#{encoding}"
            token = f"+{encoding}"
            sides.append(
                PredictionSide(
                    outcome=outcome_id,
                    side=side_index,
                    name=str(side_spec.get("name", side_index)),
                    symbol=symbol,
                    token=token,
                    asset_id=100_000_000 + encoding,
                    mid=mids.get(symbol),
                )
            )

        if len(sides) < 2:
            continue

        slug = app_style_prediction_slug(fields) or slugify(title)
        aliases = tuple(
            alias
            for alias in (
                slugify(title),
                app_style_prediction_slug(fields, sides[0].name),
                app_style_prediction_slug(fields, sides[1].name),
            )
            if alias
        )
        markets.append(
            PredictionMarket(
                outcome=outcome_id,
                name=str(outcome.get("name", "")),
                description=description,
                title=title,
                slug=slug,
                underlying=fields.get("underlying"),
                target_price=fields.get("targetPrice"),
                expiry=format_expiry(fields.get("expiry")),
                period=fields.get("period"),
                collateral="USDH",
                min_order_value="10",
                aliases=aliases,
                yes=sides[0],
                no=sides[1],
                sides=tuple(sides),
            )
        )
    return markets
