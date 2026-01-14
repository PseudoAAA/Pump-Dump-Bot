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
