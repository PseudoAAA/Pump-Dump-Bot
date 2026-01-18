package scanner

import (
	"PumpDumpBot/internal/config"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"log"
	"math"
	"net/http"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
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

func FindPump(symbol string) (pct float64, open float64, close float64, kline KlineData) {
	cfg, err := config.LoadMexcConfig("internal/config/config.json")
	if err != nil {
		log.Fatal("mexc config load error", "err", err)
	}

	url := fmt.Sprintf("https://contract.mexc.com/api/v1/contract/kline/%s?interval=Min%f&limit=100",
		symbol,
		cfg.PriceMonitoring.IntervalMinutes)

	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, 0, KlineData{}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	var r KlineResp
	if err := json.Unmarshal(body, &r); err != nil {
		return 0, 0, 0, KlineData{}
	}

	if len(r.Data.Open) < 2 || len(r.Data.Close) < 0 || !isVolumeSpike(r, 30, 1) {
		return 0, 0, 0, KlineData{}
	}

	open = r.Data.Open[0]
	close = r.Data.Close[len(r.Data.Close)-1]
	return ((close - open) / open) * 100, open, close, r.Data
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
	if err := json.Unmarshal(body, &r); err != nil {
		return 0, err
	}

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
	if err := json.Unmarshal(body, &r); err != nil {
		return 0, err
	}

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
	if err := json.Unmarshal(body, &r); err != nil {
		return 0, err
	}

	return r.Data.FundingRate * 100, nil
}

func ListingDate(symbol string) (int64, error) {
	cfg, err := config.LoadMexcConfig("internal/config/config.json")
	if err != nil {
		log.Fatal(err)
	}

	if !cfg.ExtraInfo.ShowListingDate {
		return 0, nil
	}

	url := fmt.Sprintf("https://contract.mexc.com/api/v1/contract/kline/%s?interval=Month1&limit=100",
		symbol)

	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	var r ListingResp
	if err := json.Unmarshal(body, &r); err != nil {
		return 0, err
	}

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
	if err := json.Unmarshal(body, &r); err != nil {
		return 0, err
	}

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
	var r ImbalanceResp
	if err := json.Unmarshal(body, &r); err != nil {
		return 0, err
	}

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

func DrawPriceChart(symbol string, kline KlineData, filePath string) error {
	p := plot.New()
	p.Title.Text = symbol
	p.X.Tick.Marker = plot.TimeTicks{Format: "15:04"}

	n := len(kline.Close)
	if n == 0 || len(kline.Volume) != n || len(kline.Time) != n {
		return fmt.Errorf("invalid kline data")
	}

	minPrice := kline.Close[0]
	maxPrice := kline.Close[0]
	maxVol := 0.0
	for i := 0; i < n; i++ {
		minPrice = math.Min(minPrice, kline.Close[i])
		maxPrice = math.Max(maxPrice, kline.Close[i])
		maxVol = math.Max(maxVol, kline.Volume[i])
	}

	priceRange := maxPrice - minPrice
	if priceRange == 0 {
		priceRange = 1
	}

	greenPts := make(plotter.XYs, 0)
	redPts := make(plotter.XYs, 0)

	for i := 0; i < n; i++ {
		t := float64(time.Unix(kline.Time[i], 0).Unix())
		h := (kline.Volume[i] / maxVol) * priceRange * 0.7 // 70% графика

		if i == 0 || kline.Close[i] >= kline.Close[i-1] {
			greenPts = append(greenPts, plotter.XY{X: t, Y: minPrice + h})
		} else {
			redPts = append(redPts, plotter.XY{X: t, Y: minPrice + h})
		}
	}

	if len(greenPts) > 1 {
		greenLine, err := plotter.NewLine(greenPts)
		if err != nil {
			return err
		}
		greenLine.Width = vg.Points(1)
		greenLine.Color = color.RGBA{R: 0, G: 255, B: 0, A: 255}
		p.Add(greenLine)
	}

	if len(redPts) > 1 {
		redLine, err := plotter.NewLine(redPts)
		if err != nil {
			return err
		}
		redLine.Width = vg.Points(1)
		redLine.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}
		p.Add(redLine)
	}

	pricePts := make(plotter.XYs, n)
	for i := 0; i < n; i++ {
		t := time.Unix(kline.Time[i], 0)
		pricePts[i].X = float64(t.Unix())
		pricePts[i].Y = kline.Close[i]
	}

	priceLine, err := plotter.NewLine(pricePts)
	if err != nil {
		return err
	}
	priceLine.Width = vg.Points(1)
	priceLine.Color = color.RGBA{R: 0, G: 0, B: 0, A: 200}
	p.Add(priceLine)

	return p.Save(14*vg.Inch, 6*vg.Inch, filePath)
}

/*
func FinalOutput(cfg config.Config) {

}
*/
