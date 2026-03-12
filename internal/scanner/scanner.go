package scanner

import (
	"PumpDumpBot/internal/config"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func GetContracts() ([]string, error) {
	resp, err := http.Get("https://contract.mexc.com/api/v1/contract/detail")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	var r ContractResp
	json.Unmarshal(body, &r)

	var symbols []string
	for _, c := range r.Data {
		if c.State == 0 {
			symbols = append(symbols, c.Symbol)
		}
	}
	return symbols, nil
}

func FindPump(symbol string, cfg *config.Config) (pct float64, open float64, close float64, kline KlineData) {
	if strings.Contains(symbol, "USDC") {
		return 0, 0, 0, KlineData{}
	}

	url := fmt.Sprintf("https://contract.mexc.com/api/v1/contract/kline/%s?interval=Min1&limit=80", symbol)

	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, 0, KlineData{}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var r KlineResp
	if err := json.Unmarshal(body, &r); err != nil || len(r.Data.Close) < 80 {
		return 0, 0, 0, KlineData{}
	}

	data := r.Data
	lastIdx := len(r.Data.Close) - 1
	currentPrice := data.Close[lastIdx]

	minPrice := data.Low[lastIdx]
	minIdx := 0
	for i := 0; i <= lastIdx; i++ {
		if data.Low[i] < minPrice {
			minPrice = data.Low[i]
			minIdx = i
		}
	}

	maxPrice := data.High[lastIdx]
	highIdx := 0
	for i := 0; i <= lastIdx; i++ {
		if data.High[i] > maxPrice {
			maxPrice = data.High[i]
			highIdx = i
		}
	}

	if highIdx >= 78 && minIdx <= highIdx-10 {
		pct = ((maxPrice - minPrice) / minPrice) * 100
	} else {
		pct = ((currentPrice - minPrice) / minPrice) * 100
	}

	//fmt.Println(symbol, "pct: ", pct, " min: ", minPrice, " cur: ", currentPrice, " max: ", maxPrice)
	if pct >= cfg.PriceMonitoring.MinPriceChangePercent && IsVolumeSpike(symbol, data.Volume, 30, 10.0) {
		return pct, minPrice, maxPrice, data
	}

	return 0, 0, 0, KlineData{}
}

func IsVolumeSpike(symbol string, volumes []float64, lookback int, multiplier float64) bool {
	length := len(volumes) - 1
	if length < 2 {
		return false
	}

	var currentVolume float64
	for i := 0; i < 3; i++ {
		currentVolume += volumes[length-i]
	}

	var sum float64
	for i := 0; lookback > i; i++ {
		sum += volumes[length-i]
	}

	avgVolume := (sum - currentVolume) / float64(lookback)
	if avgVolume == 0 {
		return false
	}

	fmt.Printf("%s x: %.2f\n", symbol, currentVolume/avgVolume)
	return currentVolume/avgVolume >= multiplier
}

func Get24hVolume(symbol string) (float64, error) {
	cfg, err := config.LoadMexcConfig("internal/config/config.json")
	if err != nil {
		log.Fatal("mexc config load error", "err", err)
	}

	if !cfg.ExtraInfo.ShowVolume24h {
		return 0, nil
	}

	url := fmt.Sprintf(
		"https://contract.mexc.com/api/v1/contract/ticker?symbol=%s",
		symbol,
	)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	var r TickerResp
	json.Unmarshal(body, &r)

	return r.Data.Amount24, nil
}

func GetRSI(symbol string) (float64, error) {
	cfg, err := config.LoadMexcConfig("internal/config/config.json")
	if err != nil {
		log.Fatal("mexc config load error", "err", err)
	}

	if !cfg.ExtraInfo.ShowRSI {
		return 0, nil
	}

	url := fmt.Sprintf("https://contract.mexc.com/api/v1/contract/kline/%s?interval=Hour4&limit=100",
		symbol)

	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	var r RSIResp
	json.Unmarshal(body, &r)

	closes := r.Data.Close
	if len(closes) < 6+1 {
		return 0, err
	}

	var gain, loss float64
	for i := len(closes) - 6; i < len(closes); i++ {
		diff := closes[i] - closes[i-1]
		if diff > 0 {
			gain += diff
		} else {
			loss -= diff
		}
	}

	if loss == 0 {
		return 100, nil
	}

	rs := gain / loss
	return 100 - (100 / (1 + rs)), err
}

func GetFundingRate(symbol string) (float64, error) {
	cfg, err := config.LoadMexcConfig("internal/config/config.json")
	if err != nil {
		log.Fatal("mexc config load error", "err", err)
	}

	if !cfg.ExtraInfo.ShowFundingRate {
		return 0, nil
	}

	url := fmt.Sprintf("https://contract.mexc.com/api/v1/contract/funding_rate/%s",
		symbol)

	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	var r FundingResp
	json.Unmarshal(body, &r)

	return r.Data.FundingRate * 100, nil
}

func ListingDate(symbol string) (int64, error) {
	url := fmt.Sprintf("https://contract.mexc.com/api/v1/contract/kline/%s?interval=Month1&limit=100",
		symbol)

	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	var r ListingResp
	json.Unmarshal(body, &r)

	if r.Data.Time == nil || len(r.Data.Time) <= 0 {
		return 0, nil
	}

	listingTime := time.Unix(r.Data.Time[0], 0).UTC()
	now := time.Now().UTC()
	diff := now.Sub(listingTime)

	return int64(diff.Hours() / 24), nil
}

func GetPrice24h(symbol string) (float64, error) {
	cfg, err := config.LoadMexcConfig("internal/config/config.json")
	if err != nil {
		log.Fatal("mexc config load error", "err", err)
	}

	if !cfg.ExtraInfo.ShowPriceChange24h {
		return 0, nil
	}

	url := fmt.Sprintf(
		"https://contract.mexc.com/api/v1/contract/ticker?symbol=%s",
		symbol,
	)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	var r TickerResp
	json.Unmarshal(body, &r)

	return r.Data.Price24 * 100, nil
}

func GetImbalance(symbol string) (float64, error) {
	cfg, err := config.LoadMexcConfig("internal/config/config.json")
	if err != nil {
		log.Fatal("mexc config load error", "err", err)
	}

	if !cfg.ExtraInfo.ShowOrderbookImbalance {
		return 0, nil
	}

	url := fmt.Sprintf(
		"https://contract.mexc.com/api/v1/contract/depth/%s?limit=100",
		symbol,
	)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	var r DepthResp
	json.Unmarshal(body, &r)

	var bidVol, askVol float64
	for _, b := range r.Data.Bids {
		if len(b) < 2 {
			continue
		}
		bidVol += b[1]
	}

	for _, a := range r.Data.Asks {
		if len(a) < 2 {
			continue
		}
		askVol += a[1]
	}

	total := bidVol + askVol
	if total == 0 {
		return 0, fmt.Errorf("zero volume orderbook")
	}

	return bidVol / total, nil
}

func FinalOutput(symbol string, params PumpParams, cfg *config.DbConfig) (output string) {
	var str string

	symbolFormated := strings.TrimSuffix(symbol, "_USDT")
	symbolEscaped := html.EscapeString(symbolFormated)

	str = fmt.Sprintf("<code>%s</code>\nPump %.2f%%\nPrice: %.5f->%.5f\n", symbolEscaped, params.Pct, params.Open, params.Close)

	if cfg.ExtraInfo.ShowRSI {
		rsi, _ := GetRSI(symbol)
		str += fmt.Sprintf("RSI7 (4 hours): %.2f\n", rsi)
	}

	if cfg.ExtraInfo.ShowPriceChange24h {
		change, _ := GetPrice24h(symbol)
		str += fmt.Sprintf("Price change: %.3f%%\n", change)
	}

	if cfg.ExtraInfo.ShowVolume24h {
		volume, _ := Get24hVolume(symbol)
		str += fmt.Sprintf("Volume 24h: %.2fM\n", volume/1_000_000)
	}

	if cfg.ExtraInfo.ShowOrderbookImbalance {
		imbalance, _ := GetImbalance(symbol)
		str += fmt.Sprintf("Imbalance: %.2f%%\n", imbalance)
	}

	if cfg.ExtraInfo.ShowListingDate {
		listingTime, _ := ListingDate(symbol)
		str += fmt.Sprintf("Listing Date: %d days ago\n", listingTime)
	}

	if cfg.ExtraInfo.ShowFundingRate {
		funding, _ := GetFundingRate(symbol)
		str += fmt.Sprintf("Funding rate: %.3f%%\n", funding)
	}

	return str
}

func checkRSI(cfg *config.Config, rsiValue float64) bool {
	if !cfg.ExtraInfo.ShowRSI {
		return true
	}

	return rsiValue >= cfg.RsiParams.Value
}

func checkFunding(cfg *config.Config, fundingValue float64) bool {
	if !cfg.FundingParams.Enabled {
		return true
	}

	return fundingValue <= cfg.FundingParams.Value
}

func checkImbalance(cfg *config.Config, imbalanceValue float64) bool {
	if !cfg.ImbalanceParams.Enabled {
		return true
	}

	return imbalanceValue >= cfg.ImbalanceParams.Value
}
