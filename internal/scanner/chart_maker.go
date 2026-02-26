package scanner

import (
	"PumpDumpBot/internal/config"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"

	_ "github.com/lib/pq"
)

type errPoints struct {
	plotter.XYs
	plotter.YErrors
}

func DrawPriceChartOld(symbol string, kline KlineData, filePath string) error {
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
		h := (kline.Volume[i] / maxVol) * priceRange * 0.3 // 50% графика
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

func DrawPriceChart(symbol string, kline KlineData, filePath string) error {
	cfg, err := config.LoadMexcConfig("internal/config/config.json")
	if err != nil {
		log.Fatal("mexc config load error", "err", err)
	}

	p := plot.New()
	p.Title.Text = symbol

	// Настройка отступов и "коробочного" вида
	p.X.Padding = 0
	p.Y.Padding = 0
	p.Title.Padding = vg.Points(20)

	n := len(kline.Close)
	if n < 2 {
		return fmt.Errorf("not enough data")
	}

	// --- 1. ГЕНЕРАЦИЯ МЕТОК ВРЕМЕНИ ИЗ КОНФИГА ---
	var ticks []plot.Tick
	if n > 0 {
		startT := kline.Time[0]
		endT := kline.Time[n-1]
		// Шаг в секундах
		stepSec := int64(cfg.PriceMonitoring.IntervalMinutes * 60)

		// Генерируем метки времени с заданным шагом
		for t := startT; t <= endT; t += stepSec {
			ticks = append(ticks, plot.Tick{
				Value: float64(t),
				Label: time.Unix(t, 0).Format("15:04"),
			})
		}
		p.X.Tick.Marker = plot.ConstantTicks(ticks)
	}

	// --- 2. РАСЧЕТ МАСШТАБОВ ---
	minPrice, maxPrice := kline.Close[0], kline.Close[0]
	maxVol := 0.0
	for i := 0; i < n; i++ {
		minPrice = math.Min(minPrice, kline.Close[i])
		maxPrice = math.Max(maxPrice, kline.Close[i])
		maxVol = math.Max(maxVol, kline.Volume[i])
	}

	// Цена (Y) с небольшим запасом
	p.Y.Min = minPrice - (maxPrice-minPrice)*0.10
	p.Y.Max = maxPrice + (maxPrice-minPrice)*0.10
	yRange := p.Y.Max - p.Y.Min

	// Объемы (нормализованные до 85% высоты)
	normVol := func(v float64) float64 {
		if maxVol == 0 {
			return p.Y.Min
		}
		return p.Y.Min + (v/maxVol)*yRange*1
	}

	// --- 3. ПОДГОТОВКА ТОЧЕК И ПЕРЕСЕЧЕНИЙ ---
	greenPts := make(plotter.XYs, n)
	redPts := make(plotter.XYs, n)
	intersectPts := make(plotter.XYs, n)

	for i := 0; i < n; i++ {
		t := float64(kline.Time[i])
		v := normVol(kline.Volume[i])
		isGrowth := i > 0 && kline.Close[i] >= kline.Close[i-1]

		greenPts[i].X, redPts[i].X, intersectPts[i].X = t, t, t
		if isGrowth {
			greenPts[i].Y = v
			redPts[i].Y = p.Y.Min + (v-p.Y.Min)*0.3
		} else {
			redPts[i].Y = v
			greenPts[i].Y = p.Y.Min + (v-p.Y.Min)*0.3
		}
		intersectPts[i].Y = math.Min(greenPts[i].Y, redPts[i].Y)
	}

	// --- 4. РИСОВАНИЕ СТРУКТУРЫ (ЗАЛИВКА И ЛИНИИ) ---
	// Бежевая заливка пересечения
	polyPts := make(plotter.XYs, n+2)
	copy(polyPts, intersectPts)
	polyPts[n] = plotter.XY{X: intersectPts[n-1].X, Y: p.Y.Min}
	polyPts[n+1] = plotter.XY{X: intersectPts[0].X, Y: p.Y.Min}

	fill, _ := plotter.NewPolygon(polyPts)
	fill.Color = color.RGBA{R: 245, G: 225, B: 185, A: 255}
	fill.LineStyle.Width = 0
	p.Add(fill)

	// Контуры объемов
	gl, _ := plotter.NewLine(greenPts)
	gl.Color = color.RGBA{G: 140, A: 255}
	gl.Width = vg.Points(1.3)
	p.Add(gl)
	rl, _ := plotter.NewLine(redPts)
	rl.Color = color.RGBA{R: 190, A: 255}
	rl.Width = vg.Points(1.3)
	p.Add(rl)

	// Цена
	pricePts := make(plotter.XYs, n)
	for i := 0; i < n; i++ {
		pricePts[i].X, pricePts[i].Y = float64(kline.Time[i]), kline.Close[i]
	}
	lp, _ := plotter.NewLine(pricePts)
	lp.Color = color.Black
	lp.Width = vg.Points(2.2)
	p.Add(lp)

	sc, _ := plotter.NewScatter(pricePts)
	sc.GlyphStyle.Shape = draw.CircleGlyph{}
	sc.GlyphStyle.Radius = vg.Points(3)
	sc.Color = color.Black
	p.Add(sc)

	mid := (minPrice + maxPrice) / 2
	base, _ := plotter.NewLine(plotter.XYs{{X: p.X.Min, Y: mid}, {X: p.X.Max, Y: mid}})
	base.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}
	base.Width = vg.Points(2)
	p.Add(base)

	// --- 5. СОХРАНЕНИЕ С ВНЕШНИМИ ОТСТУПАМИ (CANVAS) ---
	width, height := 14*vg.Inch, 8*vg.Inch
	c, _ := draw.NewFormattedCanvas(width, height, "png")

	margin := vg.Points(60)
	inner := draw.Canvas{
		Canvas: c,
		Rectangle: vg.Rectangle{
			Min: vg.Point{X: margin, Y: margin},
			Max: vg.Point{X: width - margin, Y: height - margin},
		},
	}

	// Поворот меток времени для читаемости
	p.X.Tick.Label.Rotation = math.Pi / 4
	p.X.Tick.Label.XAlign = draw.XRight

	p.Draw(inner)

	f, _ := os.Create(filePath)
	defer f.Close()
	c.WriteTo(f)
	return nil
}
