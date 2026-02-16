package scanner

type Contract struct {
	Symbol string `json:"symbol"`
	State  int    `json:"state"`
}

type ContractResp struct {
	Data []Contract `json:"data"`
}

type KlineData struct {
	Time   []int64   `json:"time"`
	Open   []float64 `json:"open"`
	Close  []float64 `json:"close"`
	High   []float64 `json:"high"`
	Low    []float64 `json:"low"`
	Volume []float64 `json:"vol"`
}

type KlineResp struct {
	Data KlineData `json:"data"`
}

type Ticker24h struct {
	Symbol   string  `json:"symbol"`
	Volume24 float64 `json:"volume24"`
	Amount24 float64 `json:"amount24"`
	Price24  float64 `json:"riseFallRate"`
}

type TickerResp struct {
	Data Ticker24h `json:"data"`
}

type Funding struct {
	FundingRate float64 `json:"fundingRate"`
	FundingTime int64   `json:"collectCycle"`
}

type FundingResp struct {
	Data Funding `json:"data"`
}

type ListingResp struct {
	Data struct {
		Time []int64 `json:"time"`
	} `json:"data"`
}

type RSIResp struct {
	Data struct {
		Close []float64 `json:"close"`
	} `json:"data"`
}

type DepthResp struct {
	Data struct {
		Bids [][]float64 `json:"bids"`
		Asks [][]float64 `json:"asks"`
	} `json:"data"`
}
type LiquidityTargets struct {
	UpperPrice float64
	LowerPrice float64
}

type PumpParams struct {
	Pct   float64
	Open  float64
	Close float64
	Kline KlineData
}

type OrderLevel struct {
	Price  float64
	Volume float64
}
