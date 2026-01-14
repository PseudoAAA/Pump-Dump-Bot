package validators

import (
	"errors"
	"strings"
)

// ValidateBotToken проверяет валидность токена бота
func ValidateBotToken(token string) error {
	if strings.TrimSpace(token) == "" {
		return errors.New("токен бота не может быть пустым")
	}

	// Проверяем формат токена Telegram бота (должен содержать двоеточие)
	if !strings.Contains(token, ":") {
		return errors.New("неверный формат токена бота")
	}

	// Проверяем длину токена
	if len(token) < 20 || len(token) > 100 {
		return errors.New("неверная длина токена бота")
	}

	return nil
}

// ValidateChatID проверяет валидность ID чата
func ValidateChatID(chatID int64) error {
	if chatID == 0 {
		return errors.New("ID чата не может быть равен нулю")
	}

	// Проверяем, что ID чата в разумных пределах для Telegram
	// Telegram ID чатов могут быть от -2^63 до 2^63-1, но ограничиваем разумными пределами
	if chatID < -999999999999999 || chatID > 999999999999999 {
		return errors.New("ID чата вне допустимого диапазона")
	}

	return nil
}
