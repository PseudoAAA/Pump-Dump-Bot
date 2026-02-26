package config

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strconv"

	"PumpDumpBot/internal/validators"
)

// Конфиг для телеграмма
type TelegramConfig struct {
	BotToken string // Токен Telegram-бота
	ChatID   int64  // Идентификатор Telegram-канала
}

// Конфиг для биржи
type Config struct {
	PriceMonitoring PriceMonitoringConfig `json:"price_monitoring"`
	RsiParams       RSIConfig             `json:"rsi"`
	ExtraInfo       ExtraInfoConfig       `json:"extra_info"`
	FundingParams   FundingConfig         `json:"funding_params"`
	ImbalanceParams ImbalanceConfig       `json:"imbalance_params"`
}

type DbConfig struct {
	ExtraInfo ExtraInfoConfig `json:"extra_info"`
}

type PriceMonitoringConfig struct {
	IntervalMinutes       int     `json:"interval_minutes"`
	MinPriceChangePercent float64 `json:"min_price_change_percent"`
}

type RSIConfig struct {
	Enabled          bool    `json:"enabled"`
	Value            float64 `json:"value"`
	TimeframeMinutes int     `json:"timeframe_minutes"`
}

type FundingConfig struct {
	Enabled bool    `json:"enabled"`
	Value   float64 `json:"value"`
}

type ImbalanceConfig struct {
	Enabled bool    `json:"enabled"`
	Value   float64 `json:"value"`
}

type ExtraInfoConfig struct {
	ShowPriceChange24h     bool `json:"show_price_change_24h"`
	ShowOrderbookImbalance bool `json:"show_orderbook_imbalance"`
	ShowListingDate        bool `json:"show_listing_date"`
	ShowVolume24h          bool `json:"show_volume_24h"`
	ShowFundingRate        bool `json:"show_funding_rate"`
	ShowRSI                bool `json:"show_rsi"`
}

// Загрузка переменных из env
func LoadTelegramConfig(logger *slog.Logger) (*TelegramConfig, error) {
	// Чтение переменных окружения
	botToken := os.Getenv("BOT_TOKEN")
	chatIDStr := os.Getenv("CHAT_ID")

	//Проверка переменных для бота и чата
	if botToken == "" || chatIDStr == "" {
		logger.Error("Необходимые переменные окружения отсутствуют",
			"BOT_TOKEN", maskToken(botToken), "CHAT_ID", chatIDStr)
		return nil, errors.New("необходимые переменные окружения отсутствуют")
	}

	//Преобразование CHAT_ID
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		logger.Error("Ошибка преобразования CHAT_ID в int64", "error", err)
		return nil, err
	}

	//Валидация токена бота
	if err := validators.ValidateBotToken(botToken); err != nil {
		logger.Error("Невалидный токен бота", "error", err)
		return nil, err
	}

	//Валидация ID чата
	if err := validators.ValidateChatID(chatID); err != nil {
		logger.Error("Невалидный ID чата", "error", err)
		return nil, err
	}

	// Возвращаем конфигурацию
	return &TelegramConfig{
		BotToken: botToken,
		ChatID:   chatID,
	}, nil
}

// Маскировка токена для логирования
func maskToken(token string) string {
	if token == "" {
		return "***"
	}
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "***" + token[len(token)-4:]
}

func LoadMexcConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
