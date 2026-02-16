package main

import (
	"PumpDumpBot/internal/ForBot"
	"PumpDumpBot/internal/adapters"
	"PumpDumpBot/internal/config"
	"PumpDumpBot/internal/scanner"
	"PumpDumpBot/logger"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	logger := logger.SetupLogger()
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

	go ForBot.CheckMessages()

	for {
		symbols, err = scanner.GetContracts()
		for _, symbol := range symbols {

			pump, open, close, kline := scanner.FindPump(symbol)

			if pump >= cfg.PriceMonitoring.MinPriceChangePercent {

				output := scanner.FinalOutput(symbol, scanner.PumpParams{Pct: pump, Open: open, Close: close, Kline: kline}, cfg)
				file := fmt.Sprintf(
					"charts/%s_%d.png",
					symbol,
					time.Now().Unix(),
				)

				if err := scanner.DrawPriceChart(symbol, kline, file); err != nil {
					logger.Warn("DrawPriceChart", err)
				}

				/*fmt.Printf("Coin: %s, Pump: %.2f%%, Price: (%.5f->%.5f) Volume(24h): %.2fm, RsiParams: %.2f\n",
					symbol,
					pump,
					open,
					close,
					vol/1_000_000.,
					RsiParams)

				strttg := fmt.Sprintf("🤡Coin: <code>%s</code>\n"+
					"🥵Pump: %.2f%%\n"+
					"🤯Price: (%.5f->%.5f)\n"+
					"📊PriceChange: %.3f%%\n"+
					"🤑Volume(24h): %.2fm\n"+
					"🙀RsiParams: %.2f\n"+
					"✉️Funding Rate: %.3f%%\n"+
					"🕑Listing: %d days ago\n"+
					"Imbalance: %.2f%%\n",
					symbol,
					pump,
					open,
					close,
					price24h,
					vol/1_000_000,
					RsiParams,
					funding,
					listingTime,
					imbalance)*/

				if err := telegram.SendPhoto(file, output); err != nil {
					logger.Warn("SendPhoto error: ", err)
				}

				if err := os.Remove(file); err != nil {
					logger.Warn("Remove photo error: ", err)
				}

			}
		}
	}
}
