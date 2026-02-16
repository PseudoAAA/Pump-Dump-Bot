package ForBot

import (
	"encoding/json"
	"log"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func getMessageForUser(userID int64) string {
	// 1. Открываем файл
	file, err := os.ReadFile("configbot.json")
	if err != nil {
		log.Printf("Ошибка открытия файла: %v", err)
		return "Ошибка системы"
	}

	// 2. Парсим JSON в мапу (словарь)
	var messages map[string]string
	err = json.Unmarshal(file, &messages)
	if err != nil {
		log.Printf("Ошибка парсинга JSON: %v", err)
		return "Ошибка конфигурации"
	}

	// 3. Преобразуем ID пользователя в строку
	uidStr := strconv.FormatInt(userID, 10)

	// 4. Ищем сообщение для конкретного пользователя
	if msg, ok := messages[uidStr]; ok {
		return msg
	}

	// 5. Если не нашли, возвращаем дефолтное
	if def, ok := messages["default"]; ok {
		return def
	}

	return "Сообщение не настроено."
}

func CheckMessages() {
	bot, _ := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))

	bot.Debug = true
	log.Printf("Авторизован как %s", bot.Self.UserName)

	file, err := os.ReadFile("configbot.json")
	if err != nil {
		log.Printf("Ошибка открытия файла: %v", err)
	}

	// 2. Парсим JSON в мапу (словарь)
	var messages map[string]string
	err = json.Unmarshal(file, &messages)
	if err != nil {
		log.Printf("Ошибка парсинга JSON: %v", err)
	}

	// Настройка получения обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		// 1. Сначала проверяем нажатие на кнопки (CallbackQuery) 🔘
		if update.CallbackQuery != nil {
			data := update.CallbackQuery.Data
			chatID := update.CallbackQuery.Message.Chat.ID
			messageID := update.CallbackQuery.Message.MessageID

			switch data {
			case "interval_monitoring":
				keyboard := getIntervalMonitoringKeyboard()
				editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "Выберите интервал мониторинга цены, рекомендуется использовать 10-15 минут:")
				editMsg.ReplyMarkup = &keyboard
				bot.Send(editMsg)

			case "set_interval":
				keyboard := getIntervalMonitoringKeyboard()
				editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "Выберите интервал мониторинга цены, рекомендуется использовать 10-15 минут:")
				editMsg.ReplyMarkup = &keyboard
				bot.Send(editMsg)

			case "open_settings":

				keyboard := getSettingsKeyboard()
				editMsg := tgbotapi.NewEditMessageText(chatID, messageID, messages["⚙️ Настройки"])
				editMsg.ReplyMarkup = &keyboard
				bot.Send(editMsg)

			}

			// Обязательно подтверждаем callback, чтобы убрать "часики" ⌛
			bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			continue // Переходим к следующему обновлению, не проверяя Message
		}

		// 2. Затем проверяем обычные сообщения (Message) ✉️
		if update.Message != nil {
			chatID := update.Message.Chat.ID

			switch update.Message.Text {
			case "/start":
				msg := tgbotapi.NewMessage(chatID, messages[update.Message.Text])
				msg.ReplyMarkup = getMainMenu()
				bot.Send(msg)

			case "⚙️ Настройки":
				msg := tgbotapi.NewMessage(chatID, messages[update.Message.Text])
				msg.ReplyMarkup = getSettingsKeyboard()
				bot.Send(msg)
			}
		}
	}
}

func getMainMenu() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⚙️ Настройки"),
			tgbotapi.NewKeyboardButton("💰 Доступ"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("ℹ️ Информация"),
			tgbotapi.NewKeyboardButton("🔗 Реферальная система"),
		),
	)
	keyboard.ResizeKeyboard = true
	return keyboard
}

func getSettingsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏱️Интервал мониторинга", "interval_monitoring"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📈Порог изменения цены", "pump"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔥RSI", "rsi"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰Фандинг", "funding"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🛒Имбаланс", "imbalance"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊Изменение цены за 24 часа", "price_24h"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🛒Отображение имбаланса", "show_imbalance"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅Листинг", "show_listing"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊Объем за 24 часа", "volume_24h"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰Отображение фандинга", "show_funding"),
		),
	)
}

func getIntervalMonitoringKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1 мин", "set_interval"),
			tgbotapi.NewInlineKeyboardButtonData("3 мин", "set_interval"),
			tgbotapi.NewInlineKeyboardButtonData("5 мин", "set_interval"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("10 мин", "set_interval"),
			tgbotapi.NewInlineKeyboardButtonData("15 мин", "set_interval"),
			tgbotapi.NewInlineKeyboardButtonData("30 мин", "set_interval"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться в меню", "open_settings"),
		),
	)
}

/*func setIntervalMonitoringKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow()
}*/

/*func getIntervalMonitoringKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏱️Интервал мониторинга", "interval_monitoring"),
		),
	)
}*/
