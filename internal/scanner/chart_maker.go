package scanner

import (
	"PumpDumpBot/internal/config"
	"fmt"
	"image/color"
	"log"
	"math"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func DrawPriceChart(symbol string, kline KlineData, filePath string) error {
	cfg, err := config.LoadMexcConfig("internal/config/config.json")
	if err != nil {
		log.Fatal("mexc config load error", "err", err)
	}

	testv := cfg.PriceMonitoring.IntervalMinutes

	p := plot.New()
	p.Title.Text = symbol
	p.X.Tick.Marker = plot.TickerFunc(func(min, max float64) []plot.Tick {
		ticks := []plot.Tick{}
		start := time.Unix(int64(min), 0).Truncate(time.Duration(testv) * time.Minute)
		end := time.Unix(int64(max), 0)
		for t := start; t.Before(end) || t.Equal(end); t = t.Add(time.Duration(testv) * time.Minute) {
			ticks = append(ticks, plot.Tick{
				Value: float64(t.Unix()),
				Label: t.Format("15:04"),
			})
		}
		return ticks
	})

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

	p.Y.Tick.Marker = plot.TickerFunc(func(min, max float64) []plot.Tick {
		ticks := []plot.Tick{}
		numTicks := 5
		step := (max - min) / float64(numTicks)
		for i := 0; i <= numTicks; i++ {
			v := min + step*float64(i)
			ticks = append(ticks, plot.Tick{
				Value: v,
				Label: fmt.Sprintf("%.4f", v),
			})
		}
		return ticks
	})

	priceRange := maxPrice - minPrice
	if priceRange == 0 {
		priceRange = 1
	}

	greenPts := make(plotter.XYs, 0)
	redPts := make(plotter.XYs, 0)

	for i := 0; i < n; i++ {
		t := float64(time.Unix(kline.Time[i], 0).Unix())
		h := (kline.Volume[i] / maxVol) * priceRange * 0.5 // 50% графика
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
