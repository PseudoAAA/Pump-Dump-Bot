package main

import (
	"PumpDumpBot/internal/adapters"
	"PumpDumpBot/internal/config"
	"PumpDumpBot/internal/scanner"
	"fmt"
	"log"
	"log/slog"
	"os"
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

	symbols, err := scanner.GetContracts()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Symbols count:", len(symbols))

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

				if err := scanner.DrawPriceChart(symbol, kline, file); err != nil {
					logger.Warn("DrawPriceChart", err)
				}

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

				if err := telegram.SendPhoto(file, strttg); err != nil {
					logger.Warn("SendPhoto error: ", err)
				}

				if err := os.Remove(file); err != nil {
					logger.Warn("Remove photo error: ", err)
				}
			}
		}
	}
}
