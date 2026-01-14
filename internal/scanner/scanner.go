package scanner

import (
	"PumpDumpBot/internal/config"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func GetContracts() ([]string, error) {
	resp, err := http.Get("https://contract.mexc.com/api/v1/contract/detail")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r ContractResp
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}

	var symbols []string
	for _, c := range r.Data {
		if c.State == 0 {
			symbols = append(symbols, c.Symbol)
		}
	}
	return symbols, nil
}

func FindPump(symbol string) (pct float64, open float64, close float64) {
	cfg, err := config.LoadMexcConfig("internal/config/config.json")
	if err != nil {
		log.Fatal("mexc config load error", "err", err)
	}

	url := fmt.Sprintf("https://contract.mexc.com/api/v1/contract/kline/%s?interval=Min%f&limit=100",
		symbol,
		cfg.PriceMonitoring.IntervalMinutes)

	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	var r KlineResp
	json.Unmarshal(body, &r)

	if len(r.Data.Open) < 2 || len(r.Data.Close) < 0 || !isVolumeSpike(r, 30, 1) {
		return 0, 0, 0
	}

	open = r.Data.Open[0]
	close = r.Data.Close[len(r.Data.Close)-1]
	return ((close - open) / open) * 100, open, close
}

func isVolumeSpike(kline KlineResp, lookback int, multiplier float64) bool {
	volumes := kline.Data.Volume
	length := len(volumes)

	if length < lookback+2 {
		return false
	}

	currentIndex := length - 2
	currentVolume := volumes[currentIndex]

	var sum float64
	start := currentIndex - lookback

	for i := start; i < currentIndex; i++ {
		sum += volumes[i]
	}

	avgVolume := sum / float64(lookback)
	if avgVolume == 0 {
		return false
	}

	return currentVolume >= avgVolume*multiplier
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

	return r.Data.Volume24, nil
}

func GetRSI(symbol string) (float64, error) {
	cfg, err := config.LoadMexcConfig("internal/config/config.json")
	if err != nil {
		log.Fatal("mexc config load error", "err", err)
	}

	if !cfg.RSI.Enabled {
		return 0, nil
	}

	url := fmt.Sprintf("https://contract.mexc.com/api/v1/contract/kline/%s?interval=Min%f&limit=100",
		symbol,
		cfg.RSI.TimeframeMinutes)

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

func ListingDate(symbol string) (time.Time, error) {
	cfg, err := config.LoadMexcConfig("internal/config/config.json")
	if err != nil {
		log.Fatal("mexc config load error", "err", err)
	}

	if !cfg.ExtraInfo.ShowListingDate {
		return time.Time{}, nil
	}

	url := fmt.Sprintf("https://contract.mexc.com/api/v1/contract/kline/%s?interval=Month1&limit=100",
		symbol)

	resp, err := http.Get(url)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	var r ListingResp
	json.Unmarshal(body, &r)

	listingTime := time.Unix(r.Data.Time[0], 0)
	return listingTime.UTC(), nil
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

/*
func FinalOutput(cfg config.Config) {

}
*/
