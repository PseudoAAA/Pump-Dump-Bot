package main

import (
	"PumpDumpBot/internal/adapters"
	"PumpDumpBot/internal/config"
	"PumpDumpBot/internal/scanner"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

func setupLogger() *slog.Logger {
	return slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	)
}

func main() {
	logger := setupLogger()

	// Загружаем .env
	if err := godotenv.Load(); err != nil {
		logger.Warn(".env не найден, используем env окружения")
	}

	// Конфиг
	tgCfg, err := config.LoadTelegramConfig(logger)
	if err != nil {
		logger.Error("telegram config load error", "err", err)
		os.Exit(1)
	}

	// Telegram adapter
	telegram, err := adapters.NewTelegramAdapter(tgCfg.BotToken, tgCfg.ChatID)
	if err != nil {
		logger.Error("telegram init error", "err", err)
		os.Exit(1)
	}

	cfg, err := config.LoadMexcConfig("internal/config/config.json")
	if err != nil {
		log.Fatal("mexc config load error", "err", err)
	}

	// Проверка: бот жив
	telegram.SendMessage("")

	symbols, err := scanner.GetContracts()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Количество торговых пар:", len(symbols))

	for {
		symbols, err = scanner.GetContracts()
		for _, symbol := range symbols {

			pump, open, close, kline := scanner.FindPump(symbol)
			if pump >= cfg.PriceMonitoring.MinPriceChangePercent {
				vol, _ := scanner.Get24hVolume(symbol)
				RSI, _ := scanner.GetRSI(symbol)
				funding, _ := scanner.GetFundingRate(symbol)
				listingTime, _ := scanner.ListingDate(symbol)
				price24h, _ := scanner.GetPrice24h(symbol)
				imbalance, _ := scanner.GetImbalance(symbol)

				file := fmt.Sprintf(
					"charts/%s_%d.png",
					symbol,
					time.Now().Unix(),
				)
				scanner.DrawPriceChart(symbol, kline, file)

				fmt.Printf("Coin: %s, Pump: %.2f%%, Price: (%.5f->%.5f) Volume(24h): %.2fm, RSI: %.2f\n",
					symbol,
					pump,
					open,
					close,
					vol/1_000_000.,
					RSI)

				strttg := fmt.Sprintf("🤡Coin: %s\n"+
					"🥵Pump: %.2f%%\n"+
					"🤯Price: (%.5f->%.5f)\n"+
					"📊PriceChange: %.3f%%\n"+
					"🤑Volume(24h): %.2fm\n"+
					"🙀RSI: %.2f\n"+
					"✉️Funding Rate: %.3f%%\n"+
					"🕑Listing: %d days ago\n"+
					"Imbalance: %.2f%%\n",
					symbol,
					pump,
					open,
					close,
					price24h,
					vol/1_000_000,
					RSI,
					funding,
					listingTime,
					imbalance)

				err := telegram.SendPhoto(
					file,
					strttg,
				)
				if err != nil {
					log.Println("telegram error:", err)
				}
				os.Remove(file)
				//telegram.SendMessage(strttg)
				//path, _ := rndPicture()
				//telegram.SendPhotoByURL(tgCfg.ChatID, tgCfg.BotToken, path, strttg)
				//fmt.Printf("listingTime: %s, price24h: %.3f%%\n", listingTime.Format("2006-01-02"), price24h)
			}
		}
	}
}

func rndPicture() (string, error) {
	dir := "pictures"
	entries, err := os.ReadDir("pictures")
	if err != nil {
		return "", err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		files = append(files, filepath.Join(dir, e.Name()))
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no files in directory %s", dir)
	}

	rand.Seed(time.Now().UnixNano())
	return files[rand.Intn(len(files))], nil
}
