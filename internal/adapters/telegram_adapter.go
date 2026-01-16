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

type TelegramAdapter struct {
	bot    *tgbotapi.BotAPI
	chatID int64
}

// NewTelegramAdapter создаёт новый экземпляр TelegramAdapter.
func NewTelegramAdapter(botToken string, chatID int64) (*TelegramAdapter, error) {

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		// Если произошла ошибка при инициализации бота, возвращаем её
		return nil, err
	}

	return &TelegramAdapter{bot: bot, chatID: chatID}, nil
}

// Отправляет текстовое сообщение в Telegram-чат.
func (t *TelegramAdapter) SendMessage(message string) error {
	msg := tgbotapi.NewMessage(t.chatID, message)

	_, err := t.bot.Send(msg)

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

func (t *TelegramAdapter) SendPhoto(filePath, caption string) error {
	if _, err := os.Stat(filePath); err != nil {
		return err
	}
	photo := tgbotapi.NewPhoto(
		t.chatID,
		tgbotapi.FilePath(filePath))

	photo.Caption = caption

	_, err := t.bot.Send(photo)
	return err
}
