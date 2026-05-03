package hyperliquid

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

func parseOutcomeDescription(description string) map[string]string {
	fields := make(map[string]string)
	for _, part := range strings.Split(description, "|") {
		pieces := strings.SplitN(part, ":", 2)
		if len(pieces) != 2 {
			continue
		}
		fields[pieces[0]] = pieces[1]
	}
	return fields
}

func formatPredictionExpiry(expiry string) string {
	if len(expiry) != 13 || expiry[8] != '-' {
		return expiry
	}
	return fmt.Sprintf("%s-%s-%sT%s:%s:00Z", expiry[:4], expiry[4:6], expiry[6:8], expiry[9:11], expiry[11:13])
}

func predictionTitle(fields map[string]string) string {
	underlying := fields["underlying"]
	if underlying == "" {
		underlying = "Outcome"
	}
	targetPrice := fields["targetPrice"]
	expiry := formatPredictionExpiry(fields["expiry"])
	if targetPrice != "" && expiry != "" {
		return fmt.Sprintf("%s above %s on %s", underlying, targetPrice, expiry)
	}
	if targetPrice != "" {
		return fmt.Sprintf("%s above %s", underlying, targetPrice)
	}
	return underlying
}

func predictionSlug(value string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// appStylePredictionSlugMonths is only used to generate app-style slug aliases such as
// btc-above-78213-yes-may-04-0600 from outcomeMeta expiry 20260504-0600.
var appStylePredictionSlugMonths = map[string]string{
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

func appStylePredictionSlug(fields map[string]string, side string) string {
	underlying := fields["underlying"]
	targetPrice := fields["targetPrice"]
	expiry := fields["expiry"]
	if underlying == "" || targetPrice == "" || len(expiry) != 13 {
		return ""
	}
	month := appStylePredictionSlugMonths[expiry[4:6]]
	if month == "" {
		return ""
	}
	parts := []string{underlying, "above", targetPrice}
	if side != "" {
		parts = append(parts, side)
	}
	parts = append(parts, month, expiry[6:8], expiry[9:13])
	return predictionSlug(strings.Join(parts, "-"))
}

func (p PredictionMarket) matches(query string) bool {
	normalized := strings.ToLower(query)
	values := []string{
		p.Slug,
		strings.ToLower(p.Title),
		strings.ToLower(p.Name),
		strings.ToLower(p.Underlying),
		strings.ToLower(p.Yes.Symbol),
		strings.ToLower(p.No.Symbol),
		strings.ToLower(p.Yes.Token),
		strings.ToLower(p.No.Token),
	}
	for _, alias := range p.Aliases {
		values = append(values, strings.ToLower(alias))
	}
	for _, value := range values {
		if value == normalized || strings.Contains(value, normalized) {
			return true
		}
	}
	return false
}

func assetName(asset any) string {
	switch v := asset.(type) {
	case string:
		return v
	case PredictionSide:
		return v.Symbol
	case *PredictionSide:
		if v == nil {
			return ""
		}
		return v.Symbol
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func isPredictionAsset(asset string) bool {
	if len(asset) > 1 && (asset[0] == '#' || asset[0] == '+') {
		return allDecimalDigits(asset[1:])
	}
	if allDecimalDigits(asset) {
		return NewDecimal(asset).Float64() >= 100000000
	}
	return false
}

func predictionSymbol(asset string) string {
	if len(asset) > 1 && asset[0] == '+' {
		return "#" + asset[1:]
	}
	if allDecimalDigits(asset) {
		id := int(NewDecimal(asset).Float64())
		if id >= 100000000 {
			return fmt.Sprintf("#%d", id-100000000)
		}
	}
	return asset
}

// PredictionMarkets lists active HIP-4 prediction markets with tradeable yes/no sides.
func (s *SDK) PredictionMarkets() (PredictionMarkets, error) {
	ctx := context.Background()
	outcomeRaw, err := s.postInfo(ctx, map[string]any{"type": "outcomeMeta"})
	if err != nil {
		return nil, err
	}
	midsRaw, err := s.postInfo(ctx, map[string]any{"type": "allMids"})
	if err != nil {
		return nil, err
	}

	mids, _ := midsRaw.(map[string]any)
	outcomeMeta, _ := outcomeRaw.(map[string]any)
	outcomes, _ := outcomeMeta["outcomes"].([]any)

	markets := PredictionMarkets{}
	for _, item := range outcomes {
		outcome, ok := item.(map[string]any)
		if !ok {
			continue
		}
		outcomeID := int(NewDecimal(outcome["outcome"]).Float64())
		description, _ := outcome["description"].(string)
		fields := parseOutcomeDescription(description)
		title := predictionTitle(fields)
		sideSpecs, _ := outcome["sideSpecs"].([]any)

		sides := make([]PredictionSide, 0, len(sideSpecs))
		for sideIndex, sideItem := range sideSpecs {
			sideSpec, _ := sideItem.(map[string]any)
			encoding := outcomeID*10 + sideIndex
			symbol := fmt.Sprintf("#%d", encoding)
			mid, _ := mids[symbol].(string)
			name, _ := sideSpec["name"].(string)
			if name == "" {
				name = fmt.Sprintf("%d", sideIndex)
			}
			sides = append(sides, PredictionSide{
				Outcome:             outcomeID,
				Side:                sideIndex,
				Name:                name,
				Symbol:              symbol,
				Token:               fmt.Sprintf("+%d", encoding),
				AssetID:             100000000 + encoding,
				Mid:                 mid,
				SzDecimals:          0,
				SupportsPriorityFee: false,
			})
		}
		if len(sides) < 2 {
			continue
		}

		name, _ := outcome["name"].(string)
		slug := appStylePredictionSlug(fields, "")
		if slug == "" {
			slug = predictionSlug(title)
		}
		aliases := []string{predictionSlug(title)}
		for _, side := range sides[:2] {
			if alias := appStylePredictionSlug(fields, side.Name); alias != "" {
				aliases = append(aliases, alias)
			}
		}

		markets = append(markets, PredictionMarket{
			Outcome:       outcomeID,
			Name:          name,
			Description:   description,
			Title:         title,
			Slug:          slug,
			Underlying:    fields["underlying"],
			TargetPrice:   fields["targetPrice"],
			Expiry:        formatPredictionExpiry(fields["expiry"]),
			Period:        fields["period"],
			Collateral:    "USDH",
			MinOrderValue: "10",
			Aliases:       aliases,
			Yes:           sides[0],
			No:            sides[1],
			Sides:         sides,
		})
	}

	return markets, nil
}

// Predictions is an alias for PredictionMarkets.
func (s *SDK) Predictions() (PredictionMarkets, error) {
	return s.PredictionMarkets()
}

// PredictionMarket finds one active HIP-4 prediction market.
func (s *SDK) PredictionMarket(filter PredictionMarketFilter) (*PredictionMarket, error) {
	markets, err := s.PredictionMarkets()
	if err != nil {
		return nil, err
	}
	if market, ok := markets.Find(filter); ok {
		return market, nil
	}
	return nil, ValidationError("no matching prediction market found").
		WithGuidance("Call sdk.PredictionMarkets() to list active HIP-4 markets.")
}

// PredictionSides returns a flat list of tradeable HIP-4 sides.
func (s *SDK) PredictionSides() ([]PredictionSide, error) {
	markets, err := s.PredictionMarkets()
	if err != nil {
		return nil, err
	}
	sides := []PredictionSide{}
	for _, market := range markets {
		sides = append(sides, market.Sides...)
	}
	return sides, nil
}

// BuyUSDH buys USDH with USDC for HIP-4 prediction markets.
func (s *SDK) BuyUSDH(amountUSDC any, opts ...OrderOption) (*PlacedOrder, error) {
	opts = append(opts, WithNotional(NewDecimal(amountUSDC).Float64()), WithTIF(TIFMarket))
	return s.Buy("@230", opts...)
}

// SellUSDH sells USDH back to USDC.
func (s *SDK) SellUSDH(amountUSDH any, opts ...OrderOption) (*PlacedOrder, error) {
	opts = append(opts, WithSize(amountUSDH), WithTIF(TIFMarket))
	return s.Sell("@230", opts...)
}
