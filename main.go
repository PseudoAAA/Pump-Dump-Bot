package main

import (
	"PumpDumpBot/internal/ForBot"
	"PumpDumpBot/internal/adapters"
	"PumpDumpBot/internal/config"
	"PumpDumpBot/internal/db"
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

	db, err := db.InitDB(os.Getenv("DB_STR"))
	if err != nil {
		logger.Error("database init error", "err", err)
		os.Exit(1)
	}

	fmt.Println(db.GetActiveUsers())

	// Конфиг
	tgCfg, err := config.LoadTelegramConfig(logger)
	if err != nil {
		logger.Error("telegram config load error", "err", err)
		os.Exit(1)
	}

	// Telegram adapter
	_, err = adapters.NewTelegramAdapter(tgCfg.BotToken, tgCfg.ChatID)
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

			pump, open, close, kline := scanner.FindPump(symbol, cfg)

			if pump >= cfg.PriceMonitoring.MinPriceChangePercent {
				pumpParams := scanner.PumpParams{Pct: pump, Open: open, Close: close, Kline: kline}
				//output := scanner.FinalOutput(symbol, scanner.PumpParams{Pct: pump, Open: open, Close: close, Kline: kline}, cfg)
				file := fmt.Sprintf(
					"charts/%s_%d.png",
					symbol,
					time.Now().Unix(),
				)

				if err := scanner.DrawPriceChart(symbol, kline, file); err != nil {
					logger.Warn("DrawPriceChart", err)
				}

				users, _ := db.GetActiveUsers()
				ForBot.SendMessageToActiveUsers(users, symbol, file, pumpParams)
				/*if err := telegram.SendPhoto(file, output); err != nil {
					logger.Warn("SendPhoto error: ", err)
				}
				*/
				if err := os.Remove(file); err != nil {
					logger.Warn("Remove photo error: ", err)
				}

			}
		}
	}
}
