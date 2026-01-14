package adapters

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Реализует интерфейс TelegramSender для отправки сообщений в Telegram.
type TelegramAdapter struct {
	bot    *tgbotapi.BotAPI // Экземпляр бота API Telegram
	chatID int64            // ID чата, куда будут отправляться сообщения
}

// NewTelegramAdapter создаёт новый экземпляр TelegramAdapter.
// Принимает токен бота (botToken) и ID чата (chatID).
// Возвращает ошибку, если не удалось инициализировать бота.
func NewTelegramAdapter(botToken string, chatID int64) (*TelegramAdapter, error) {
	// Создаём новый экземпляр BotAPI с использованием токена бота
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		// Если произошла ошибка при инициализации бота, возвращаем её
		return nil, err
	}
	// Возвращаем инициализированный адаптер с ботом и ID чата
	return &TelegramAdapter{bot: bot, chatID: chatID}, nil
}

// Отправляет текстовое сообщение в Telegram-чат.
func (t *TelegramAdapter) SendMessage(message string) error {
	// Создаём новое текстовое сообщение для отправки в указанный чат
	msg := tgbotapi.NewMessage(t.chatID, message)

	// Отправляем сообщение через API Telegram
	_, err := t.bot.Send(msg)

	// Возвращаем ошибку, если отправка не удалась
	return err
}

func (t *TelegramAdapter) SendPhotoByURL(chatID int64, botToken, filePath, caption string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	_ = writer.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	_ = writer.WriteField("caption", caption)

	part, err := writer.CreateFormFile("photo", filepath.Base(filePath))
	if err != nil {
		return err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	writer.Close()

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", botToken),
		&buf,
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err = http.DefaultClient.Do(req)
	return err
}
