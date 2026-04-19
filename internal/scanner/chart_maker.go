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
	"os"
	"time"

	"github.com/fogleman/gg"
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

type candleData struct {
	kline KlineData
}

func (c candleData) Len() int        { return len(c.kline.Close) }
func (c candleData) T(i int) float64 { return float64(c.kline.Time[i]) }
func (c candleData) O(i int) float64 { return c.kline.Open[i] }
func (c candleData) H(i int) float64 { return c.kline.High[i] }
func (c candleData) L(i int) float64 { return c.kline.Low[i] }
func (c candleData) C(i int) float64 { return c.kline.Close[i] }

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

func DrawPriceChartt(symbol string, kline KlineData, filePath string, zones []float64) error {
	url := fmt.Sprintf("https://contract.mexc.com/api/v1/contract/kline/%s?interval=Min30&limit=400",
		symbol)
	resp, err := http.Get(url)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	var r KlineResp
	json.Unmarshal(body, &r)

	kline = r.Data

	_, err = config.LoadMexcConfig("internal/config/config.json")
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

		// Генерируем метки времени с заданным шагом
		for t := startT; t <= endT; t += 600 {
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

	for _, zPrice := range zones {
		if zPrice == 0 {
			continue
		}

		// Создаем линию от самого начала (p.X.Min) до самого конца (p.X.Max) графика
		zPts := plotter.XYs{
			{X: float64(kline.Time[0]), Y: zPrice},
			{X: float64(kline.Time[n-1]), Y: zPrice},
		}

		zl, err := plotter.NewLine(zPts)
		if err != nil {
			continue
		}

		// Настройка стиля полосы: Синий пунктир (как на JOE/SIREN)
		zl.Color = color.RGBA{R: 0, G: 0, B: 255, A: 180} // Полупрозрачный синий
		zl.LineStyle.Width = vg.Points(1.5)
		zl.LineStyle.Dashes = []vg.Length{vg.Points(5), vg.Points(5)} // Пунктир

		p.Add(zl)

		// Опционально: можно добавить текст с ценой уровня прямо на линию
		// Но для начала просто отрисуем полосы.
	}

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

func DrawChart(symbol string, kline KlineData, zones []PriceZone, filePath string) error {

	url := fmt.Sprintf("https://contract.mexc.com/api/v1/contract/kline/%s?interval=Hour4&limit=400",

		symbol)

	resp, _ := http.Get(url)

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var r KlineResp

	json.Unmarshal(body, &r)

	kline = r.Data

	const (
		W = 1700 // Ширина

		H = 1000 // Высота

		M = 60 // Отступы

	)

	dc := gg.NewContext(W, H)

	// 1. ФОН (Темно-синий/черный, как на скринах)

	dc.SetRGB(0.05, 0.05, 0.07)
	dc.Clear()
	n := len(kline.Close)
	if n == 0 {

		return fmt.Errorf("no data")

	}

	dayEma := GetDayEma(symbol)
	weekEma := GetWeekEma(symbol)

	// 2. ПОИСК МИНИМУМОВ И МАКСИМУМОВ ДЛЯ МАСШТАБА

	minP, maxP := kline.Low[0], kline.High[0]

	for i := 0; i < n; i++ {

		minP = math.Min(minP, kline.Low[i])

		maxP = math.Max(maxP, kline.High[i])

	}

	// Добавляем уровни EMA и зон в расчет границ

	checkLevels := []float64{

		dayEma.Ema28,

		weekEma.Ema7, weekEma.Ema14, weekEma.Ema28,
	}

	for _, z := range zones {

		checkLevels = append(checkLevels, z.Price)

	}

	for _, p := range checkLevels {

		if p > 0 {

			// Расширяем масштаб, если уровень находится за пределами свечей

			minP = math.Min(minP, p)

			maxP = math.Max(maxP, p)

		}

	}

	// Добавляем 10% отступа сверху и снизу

	padding := (maxP - minP) * 0.1

	minP -= padding

	maxP += padding

	priceRange := maxP - minP

	// Функции трансформации координат

	scaleX := func(i int) float64 {

		return M + float64(i)*(W-2*M)/float64(n)

	}

	scaleY := func(p float64) float64 {

		return (H - M) - (p-minP)*(H-2*M)/priceRange

	}

	dc.SetRGB(0.15, 0.15, 0.2)

	dc.SetLineWidth(1.5)

	for i := 0; i <= 10; i++ {

		p := minP + float64(i)*(maxP-minP)/10

		y := scaleY(p)

		dc.DrawLine(M, y, W-M, y)

		dc.Stroke()

		dc.SetRGB(0.6, 0.6, 0.6)

		precision := 4

		if p > 100 {

			precision = 2

		}

		dc.DrawString(fmt.Sprintf("%.*f", precision, p), W-M+10, y+5)

		dc.SetRGB(0.15, 0.15, 0.2)

	}

	// 3. РИСОВАНИЕ СВЕЧЕЙ

	candleW := (W - 2*M) / float64(n) * 0.8

	for i := 0; i < n; i++ {

		x := scaleX(i)

		highY := scaleY(kline.High[i])

		lowY := scaleY(kline.Low[i])

		openY := scaleY(kline.Open[i])

		closeY := scaleY(kline.Close[i])

		// Цвет свечи

		if kline.Close[i] >= kline.Open[i] {

			dc.SetRGB(0, 0.8, 0.4) // Зеленый (рост)

		} else {

			dc.SetRGB(0.9, 0.2, 0.2) // Красный (падение)

		}

		// Рисуем фитиль (тень)

		dc.DrawLine(x, highY, x, lowY)

		dc.SetLineWidth(1)

		dc.Stroke()

		// Рисуем тело свечи

		rectH := math.Abs(openY - closeY)

		if rectH < 1 {

			rectH = 1

		} // Чтобы даже плоские свечи были видны

		dc.DrawRectangle(x-candleW/2, math.Min(openY, closeY), candleW, rectH)

		dc.Fill()

	}

	// 4. РИСОВАНИЕ ЗОН И ПОДПИСЕЙ ОБЪЕМА (998K)

	// Фиолетовый для уровней

	for _, zone := range zones {

		y := scaleY(zone.Price)

		dc.SetRGB(0.5, 0.4, 0.9)
		dc.DrawLine(float64(M), y, float64(W-M), y)
		dc.Stroke()
		dc.SetDash()
		dc.DrawCircle(float64(M), y, 2)
		dc.Fill()
		// Подписываем объем чуть правее начала или в самом конце
		volText := fmt.Sprintf("%f %.4fM", zone.Price, zone.Volume/1000000)
		dc.SetRGB(1, 1, 1)
		dc.DrawString(volText, float64(M)-40, y) // Текст над линией в месте начала

	}

	dc.SetRGB(0.6, 0.6, 0.6) // Цвет текста меток

	// Выбираем шаг так, чтобы на графике было 5-7 меток времени

	step := n / 6

	if step == 0 {

		step = 1

	}

	for i := 0; i < n; i += step {

		x := scaleX(i)

		// Рисуем вертикальную линию сетки (очень тусклую)

		dc.SetRGBA(0.15, 0.15, 0.2, 0.5)

		dc.DrawLine(x, M, x, H-M)

		dc.Stroke()

		// Форматируем время из массива kline.Time

		if i < len(kline.Time) {

			t := time.Unix(kline.Time[i], 0)

			// Если данных много (limit=400), лучше выводить "День.Месяц Часы:Минуты"

			timeStr := t.Format("02.01 15:04")

			dc.SetRGB(0.7, 0.7, 0.7)

			// Рисуем текст под нижней границей графика (H-M+20)

			dc.DrawStringAnchored(timeStr, x, float64(H-M+25), 0.5, 0)

		}

	}

	drawEmaLine := func(price float64, label string, clr color.RGBA) {
		// Проверяем, попадает ли цена EMA в видимый диапазон графика

		if price < minP || price > maxP || kline.High[n-1]*0.97 > price {

			return

		}

		y := scaleY(price)

		dc.SetColor(clr)

		dc.SetLineWidth(1.2)

		dc.DrawLine(float64(M), y, float64(W-M), y)

		dc.Stroke()

		dc.SetDash()
		label += fmt.Sprintf(" %.6f", price)
		dc.DrawStringAnchored(label, float64(M+5), y+5, 0, 0.5)

	}

	drawEmaLine(dayEma.Ema28, "1D EMA 28", color.RGBA{130, 0, 255, 255})

	drawEmaLine(weekEma.Ema7, "1W EMA7", color.RGBA{255, 235, 59, 255})

	drawEmaLine(weekEma.Ema14, "1W EMA14", color.RGBA{255, 152, 0, 255})

	drawEmaLine(weekEma.Ema28, "1W EMA 28", color.RGBA{130, 0, 255, 255})

	dc.SetRGB(1, 1, 1)

	dc.DrawStringAnchored(symbol, 25, 35, 0, 0.5)

	return dc.SavePNG(filePath)

}
