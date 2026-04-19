package scanner

import (
	"PumpDumpBot/internal/config"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"sort"
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

func FindResistanceZones(symbol string) []PriceZone {
	url := fmt.Sprintf("https://contract.mexc.com/api/v1/contract/kline/%s?interval=Hour4&limit=400", symbol)

	resp, err := http.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var r KlineResp
	if err := json.Unmarshal(body, &r); err != nil || len(r.Data.Close) == 0 {
		return nil
	}

	data := r.Data
	lastPrice := data.Close[len(data.Close)-1]

	// 1. Считаем средний объем, чтобы найти аномальные всплески
	var totalVol float64
	for _, v := range data.Volume {
		totalVol += v
	}
	avgVol := totalVol / float64(len(data.Volume))

	// 2. Собираем "сырые" зоны (где объем в 2.5 раза выше среднего)
	var rawZones []PriceZone
	for i := 0; i < len(data.Close); i++ {
		// Фильтр: аномальный объем И цена не ниже 2% от текущей (ищем сопротивления и близкую поддержку)
		if data.Volume[i] > avgVol*2.5 && data.High[i] > lastPrice*0.98 {
			rawZones = append(rawZones, PriceZone{
				Price:      data.High[i],
				Volume:     data.Volume[i],
				StartIndex: i,
			})
		}
	}

	if len(rawZones) == 0 {
		return nil
	}

	// 3. Сортируем по цене для объединения близких точек
	sort.Slice(rawZones, func(i, j int) bool {
		return rawZones[i].Price < rawZones[j].Price
	})

	// 4. Кластеризация (объединение зон в радиусе 1.5%)
	var mergedZones []PriceZone
	const threshold = 0.015 // 1.5%

	if len(rawZones) > 0 {
		current := rawZones[0]

		for i := 1; i < len(rawZones); i++ {
			// Проверяем расстояние между текущей ценой в кластере и следующей точкой
			dist := (rawZones[i].Price - current.Price) / current.Price

			if dist < threshold {
				// Объединяем: считаем средневзвешенную цену по объему
				newTotalVol := current.Volume + rawZones[i].Volume
				// Формула: (P1*V1 + P2*V2) / (V1+V2)
				current.Price = (current.Price*current.Volume + rawZones[i].Price*rawZones[i].Volume) / newTotalVol
				current.Volume = newTotalVol
				// Index оставляем от самой первой точки всплеска
			} else {
				mergedZones = append(mergedZones, current)
				current = rawZones[i]
			}
		}
		mergedZones = append(mergedZones, current)
	}

	// 5. Выбираем ТОП-3 зоны по суммарному объему
	sort.Slice(mergedZones, func(i, j int) bool {
		return mergedZones[i].Volume > mergedZones[j].Volume
	})

	if len(mergedZones) > 5 {
		return mergedZones[:5]
	}
	return mergedZones
}

func GetTopZones(zones []PriceZone, limit int) []PriceZone {
	// Если зон меньше лимита, возвращаем как есть
	if len(zones) <= limit {
		return zones
	}

	// Сортируем слайс по убыванию объема
	sort.Slice(zones, func(i, j int) bool {
		return zones[i].Volume > zones[j].Volume
	})

	// Берем первые 5 элементов (самые крупные объемы)
	topZones := zones[:limit]

	// (Опционально) Сортируем их обратно по индексу времени,
	// чтобы при отрисовке они шли в хронологическом порядке
	sort.Slice(topZones, func(i, j int) bool {
		return topZones[i].StartIndex < topZones[j].StartIndex
	})

	return topZones
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

	if pct >= cfg.PriceMonitoring.MinPriceChangePercent && IsVolumeSpike(symbol, data.Volume, 30, 5) {
		week := GetWeekEma(symbol)
		resZones := FindResistanceZones(symbol)
		price := data.High[len(data.High)-1]
		if EmaIsHigherThenPrice(price, week) || ResistanceIsHigherThenPrice(price, resZones) {
			return pct, minPrice, maxPrice, data
		}
	}
	return 0, 0, 0, KlineData{}
}

func EmaIsHigherThenPrice(price float64, ema Ema) bool {
	if price >= ema.Ema7*0.97 || price >= ema.Ema14*0.97 || price >= ema.Ema28*0.97 {
		return true
	}
	return false
}

func ResistanceIsHigherThenPrice(price float64, res []PriceZone) bool {
	for _, r := range res {
		if price >= r.Price*0.97 {
			return true
		}
	}

	return false
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
func GetDayEma(symbol string) (result Ema) {
	url := fmt.Sprintf("https://contract.mexc.com/api/v1/contract/kline/%s?interval=Day1&limit=30", symbol)
	resp, err := http.Get(url)
	if err != nil {
		return Ema{}
	}
	defer resp.Body.Close()

	var r KlineResp
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &r)
	data := r.Data.Close

	if len(data) < 28 {
		return Ema{}
	}

	k := 2.0 / float64(7+1)
	ema := data[0]
	for _, price := range data {
		ema = (price-ema)*k + ema
	}
	result.Ema7 = ema

	k = 2.0 / float64(14+1)
	ema = data[0]
	for _, price := range data {
		ema = (price-ema)*k + ema
	}
	result.Ema14 = ema

	k = 2.0 / float64(28+1)
	ema = data[0]
	for _, price := range data {
		ema = (price-ema)*k + ema
	}
	result.Ema28 = ema

	return result
}

func GetWeekEma(symbol string) (result Ema) {
	url := fmt.Sprintf("https://contract.mexc.com/api/v1/contract/kline/%s?interval=Week1&limit=30", symbol)
	resp, err := http.Get(url)
	if err != nil {
		return Ema{}
	}
	defer resp.Body.Close()

	var r KlineResp
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &r)
	data := r.Data.Close

	if len(data) < 28 {
		return Ema{}
	}

	k := 2.0 / float64(7+1)
	ema := data[0]
	for _, price := range data {
		ema = (price-ema)*k + ema
	}
	result.Ema7 = ema

	k = 2.0 / float64(14+1)
	ema = data[0]
	for _, price := range data {
		ema = (price-ema)*k + ema
	}
	result.Ema14 = ema

	k = 2.0 / float64(28+1)
	ema = data[0]
	for _, price := range data {
		ema = (price-ema)*k + ema
	}
	result.Ema28 = ema

	return result
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
