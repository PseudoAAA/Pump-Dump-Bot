package ForBot

import (
	"PumpDumpBot/internal/db"
	"PumpDumpBot/internal/scanner"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	*tgbotapi.BotAPI
}

type SignalTracker struct {
	mu       sync.Mutex
	signals  map[string]*SignalInfo
	lastDay  int
	cooldown time.Duration
}

// SignalInfo хранит данные по конкретной монете
type SignalInfo struct {
	LastSent   time.Time
	CountToday int
}

var choice = map[bool]string{
	true:  "✅",
	false: "❌",
}

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

	db, err := db.InitDB(os.Getenv("DB_STR"))
	if err != nil {
		os.Exit(1)
	}

	mexcKey := os.Getenv("MEXC_KEY")
	mexcSecret := os.Getenv("MEXC_SECRET")
	var lastAction string
	bot.Debug = true
	log.Printf("Авторизован как %s", bot.Self.UserName)

	file, err := os.ReadFile("configbot.json")
	if err != nil {
		log.Printf("Ошибка открытия файла: %v", err)
	}

	var messages map[string]string
	err = json.Unmarshal(file, &messages)
	if err != nil {
		log.Printf("Ошибка парсинга JSON: %v", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.CallbackQuery != nil {
			data := update.CallbackQuery.Data
			chatID := update.CallbackQuery.Message.Chat.ID
			messageID := update.CallbackQuery.Message.MessageID

			switch data {
			case "interval_monitoring":
				keyboard := getIntervalMonitoringKeyboard()
				editMsg := tgbotapi.NewEditMessageText(chatID, messageID, messages["⏱️Интервал мониторинга"])
				editMsg.ReplyMarkup = &keyboard
				bot.Send(editMsg)
				lastAction = "interval_monitoring"

			case "set_interval":
				keyboard := getIntervalMonitoringKeyboard()
				editMsg := tgbotapi.NewEditMessageText(chatID, messageID, messages["⏱️Интервал мониторинга"])
				editMsg.ReplyMarkup = &keyboard
				bot.Send(editMsg)
				lastAction = "set_interval"

			case "open_settings":

				keyboard := getSettingsKeyboard(chatID)
				editMsg := tgbotapi.NewEditMessageText(chatID, messageID, messages["⚙️ Настройки"])
				editMsg.ReplyMarkup = &keyboard
				bot.Send(editMsg)
				lastAction = "open_settings"

			case "open_menu":
				keyboard := getMenuKeyboard()
				editMsg := tgbotapi.NewEditMessageText(chatID, messageID, messages["open_menu"])
				editMsg.ReplyMarkup = &keyboard
				bot.Send(editMsg)
				lastAction = "open_menu"

			case "open_access":
				keyboard := getAccessKeyboard()
				str, _ := getAccessTime(db, chatID, messages["💰 Доступ"])
				editMsg := tgbotapi.NewEditMessageText(chatID, messageID, str)
				editMsg.ReplyMarkup = &keyboard
				bot.Send(editMsg)
				lastAction = "open_access"

			case "open_information":
				keyboard := getInformationKeyboard()
				editMsg := tgbotapi.NewEditMessageText(chatID, messageID, messages["ℹ️ Информация"])
				editMsg.ReplyMarkup = &keyboard
				bot.Send(editMsg)
				lastAction = "open_information"

			case "open_referral_system":
				keyboard := getInformationKeyboard()
				editMsg := tgbotapi.NewEditMessageText(chatID, messageID, messages["🔗 Реферальная система"])
				editMsg.ReplyMarkup = &keyboard
				bot.Send(editMsg)
				lastAction = "open_referral_system"

			case "update_access":
				keyboard := getUpdateAccessKeyboard()
				editMsg := tgbotapi.NewEditMessageText(chatID, messageID, messages["update_access"])
				editMsg.ReplyMarkup = &keyboard
				bot.Send(editMsg)
				lastAction = "update_access"

			case "update_access_14days":
				keyboard := getUpdateAccessPayKeyboard()
				str, _ := getPayAccessTime(14, messages["update_access_pay"])
				editMsg := tgbotapi.NewEditMessageText(chatID, messageID, str)
				editMsg.ParseMode = "HTML"
				editMsg.ReplyMarkup = &keyboard
				bot.Send(editMsg)
				lastAction = "update_access_14days"

			case "update_access_30days":
				keyboard := getUpdateAccessPayKeyboard()
				str, _ := getPayAccessTime(30, messages["update_access_pay"])
				editMsg := tgbotapi.NewEditMessageText(chatID, messageID, str)
				editMsg.ParseMode = "HTML"
				editMsg.ReplyMarkup = &keyboard
				bot.Send(editMsg)
				lastAction = "update_access_30days"

			case "update_access_90days":
				keyboard := getUpdateAccessPayKeyboard()
				str, _ := getPayAccessTime(90, messages["update_access_pay"])
				editMsg := tgbotapi.NewEditMessageText(chatID, messageID, str)
				editMsg.ParseMode = "HTML"
				editMsg.ReplyMarkup = &keyboard
				bot.Send(editMsg)
				lastAction = "update_access_90days"

			case "pay_done":
				msg := tgbotapi.NewMessage(chatID, "Отправьте ваш **TxID** ответным сообщением.\n")
				bot.Send(msg)
				lastAction = "pay_done"

			case "price_24h":
				db.SetShowPriceChange24h(chatID)
				keyboard := getSettingsKeyboard(chatID)
				edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, keyboard)
				bot.Send(edit)

			case "show_imbalance":
				db.SetShowOrderbookImbalance(chatID)
				keyboard := getSettingsKeyboard(chatID)
				edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, keyboard)
				bot.Send(edit)

			case "show_listing":
				db.SetShowListingDate(chatID)
				keyboard := getSettingsKeyboard(chatID)
				edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, keyboard)
				bot.Send(edit)

			case "volume_24h":
				db.SetShowVolume24h(chatID)
				keyboard := getSettingsKeyboard(chatID)
				edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, keyboard)
				bot.Send(edit)

			case "show_funding":
				db.SetShowFunding(chatID)
				keyboard := getSettingsKeyboard(chatID)
				edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, keyboard)
				bot.Send(edit)

			case "show_rsi":
				db.SetShowRSI(chatID)
				keyboard := getSettingsKeyboard(chatID)
				edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, keyboard)
				bot.Send(edit)
			}

			bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			continue
		}

		if update.Message != nil {
			chatID := update.Message.Chat.ID

			switch update.Message.Text {
			case "/start":
				isExist, err := db.AddUser(update.Message.Chat.ID, update.SentFrom().UserName)
				if err != nil {
					log.Println("Ошибка БД:", err)
				}
				msg := tgbotapi.NewMessage(chatID, messages[update.Message.Text])
				msg.ReplyMarkup = getMainMenu()
				bot.Send(msg)
				lastAction = "start"

				if !isExist {
					msg = tgbotapi.NewMessage(chatID, "✅Активирован пробный период на 5 дней.")
					bot.Send(msg)
				}

			case "⚙️ Настройки":
				msg := tgbotapi.NewMessage(chatID, messages[update.Message.Text])
				msg.ReplyMarkup = getSettingsKeyboard(chatID)
				bot.Send(msg)
				lastAction = "settings"

			case "ℹ️ Информация":
				msg := tgbotapi.NewMessage(chatID, messages[update.Message.Text])
				msg.ReplyMarkup = getInformationKeyboard()
				bot.Send(msg)
				lastAction = "information"

			case "💰 Доступ":
				str, _ := getAccessTime(db, chatID, messages["💰 Доступ"])
				msg := tgbotapi.NewMessage(chatID, str)
				msg.ReplyMarkup = getAccessKeyboard()
				bot.Send(msg)
				lastAction = "access"

			case "🔗 Реферальная система":
				msg := tgbotapi.NewMessage(chatID, messages[update.Message.Text])
				msg.ReplyMarkup = getReferralSystemKeyboard()
				bot.Send(msg)
				lastAction = "referral"

			default:
				//fmt.Println(lastAction)
				if lastAction == "pay_done" {
					txID := strings.TrimSpace(update.Message.Text)

					if len(txID) > 20 {
						alreadyUsed, err := db.IsTransactionUsed(txID)
						if err != nil {
							log.Printf("Ошибка БД при проверке TxID: %v", err)
							return
						}

						if alreadyUsed {
							bot.Send(tgbotapi.NewMessage(chatID, "❌ Эта транзакция уже была использована для активации подписки."))
							return
						}

						bot.Send(tgbotapi.NewMessage(chatID, "⏳ Проверяю транзакцию в системе MEXC (окно 7 дней)..."))

						amount, found, err := VerifyMexcPayment(mexcKey, mexcSecret, txID)
						if err != nil {
							log.Printf("Ошибка API MEXC: %v", err)
							bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка связи с биржей. Попробуйте позже."))
							return
						}

						if found {
							days := 0
							switch {
							case amount >= 80:
								days = 90
							case amount >= 30:
								days = 30
							case amount >= 15:
								days = 14
							}

							if days > 0 {

								err := db.SaveTransaction(txID, chatID, amount, days)
								if err != nil {
									log.Printf("Ошибка сохранения транзакции в БД: %v", err)
									bot.Send(tgbotapi.NewMessage(chatID, "❌ Ошибка при регистрации платежа. Напишите администратору."))
									return
								}

								err = db.AddSubscriptionDays(chatID, days)
								if err != nil {
									log.Printf("Ошибка продления подписки: %v", err)
									bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Оплата принята, но произошла ошибка при обновлении срока. Напишите администратору."))
									return
								}

								msg := fmt.Sprintf("✅ Оплата получена!\n💰 Сумма: %.2f USDT\n📅 Подписка продлена на %d дней.", amount, days)
								bot.Send(tgbotapi.NewMessage(chatID, msg))

							} else {
								bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("⚠️ Суммы %.2f USDT недостаточно для активации тарифа (минимум 15 USDT).", amount)))
							}
						} else {
							bot.Send(tgbotapi.NewMessage(chatID, "📭 Транзакция не найдена в истории депозитов за последние 7 дней.\n\nУбедитесь, что:\n1. Вы ввели верный ID.\n2. Статус перевода на бирже 'Success'.\n3. Прошло более 5-10 минут."))
						}
					}
				}
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

func getSettingsKeyboard(chatID int64) tgbotapi.InlineKeyboardMarkup {
	db, _ := db.InitDB(os.Getenv("DB_STR"))
	cfg, _ := db.GetUserConfig(chatID)
	return tgbotapi.NewInlineKeyboardMarkup(

		/*tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏱️Интервал мониторинга", "interval_monitoring"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📈Порог изменения цены", "pump"),
		),*/
		/*tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔥RSI", "rsi"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰Фандинг", "funding"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🛒Имбаланс", "imbalance"),
		),*/

		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📊Изменение цены за 24 часа %s", choice[cfg.ExtraInfo.ShowPriceChange24h]), "price_24h"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🛒Отображение имбаланса %s", choice[cfg.ExtraInfo.ShowOrderbookImbalance]), "show_imbalance"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📅Отображение Листинга %s", choice[cfg.ExtraInfo.ShowListingDate]), "show_listing"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📊Объем за 24 часа %s", choice[cfg.ExtraInfo.ShowVolume24h]), "volume_24h"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("💰Отображение фандинга %s", choice[cfg.ExtraInfo.ShowFundingRate]), "show_funding"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🔥RSI %s", choice[cfg.ExtraInfo.ShowRSI]), "show_rsi"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Вернуться в меню⬅️", "open_menu"),
		),
	)
}

func getIntervalMonitoringKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("5 мин", "set_interval"),
			tgbotapi.NewInlineKeyboardButtonData("10 мин", "set_interval"),
		),
		tgbotapi.NewInlineKeyboardRow(

			tgbotapi.NewInlineKeyboardButtonData("15 мин", "set_interval"),
			tgbotapi.NewInlineKeyboardButtonData("30 мин", "set_interval"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Вернуться в меню⬅️", "open_settings"),
		),
	)
}

func getInformationKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Вернуться в меню⬅️", "open_menu"),
		),
	)
}

func getAccessKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Продлить подписку💳", "update_access"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Вернуться в меню⬅️", "open_menu"),
		),
	)
}

func getReferralSystemKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Вернуться в меню⬅️", "open_menu"),
		),
	)
}

func getMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "open_settings"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Доступ", "open_access"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Информация", "open_information"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔗 Реферальная система", "open_referral_system"),
		),
	)
}

func getUpdateAccessKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳Оплатить 14 дней (15$)", "update_access_14days"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳Оплатить 30 дней (30$)", "update_access_30days"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳Оплатить 90 дней (80$)", "update_access_90days"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Вернуться⬅️", "open_access"),
		),
	)
}
func getUpdateAccessPayKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Оплачено💳", "pay_done"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Вернуться⬅️", "update_access"),
		),
	)
}
func getAccessTime(db *db.DB, chatID int64, tmp string) (output string, err error) {
	daysLeft, err := db.GetTrialDaysLeft(chatID)
	if err != nil {
		log.Println("Ошибка получения дней:", err)
		daysLeft = 0
	}
	status := "активна✅"
	if daysLeft <= 0 {
		status = "истекла❌"
	}

	replacer := strings.NewReplacer(
		"{status}", status,
		"{daysleft}", strconv.Itoa(daysLeft),
	)

	return replacer.Replace(tmp), err
}

func getPayAccessTime(days int, tmp string) (str string, err error) {
	daysPrice := map[int]string{
		14: "15.0",
		30: "30.0",
		90: "80.0",
	}

	replacer := strings.NewReplacer(
		"{days}", strconv.Itoa(days),
		"{price}", daysPrice[days],
	)

	return replacer.Replace(tmp), err
}

func VerifyMexcPayment(apiKey, apiSecret, targetTxID string) (float64, bool, error) {
	baseURL := "https://api.mexc.com/api/v3/capital/deposit/hisrec"
	now := time.Now()

	startTime := now.AddDate(0, 0, -7).UnixMilli()
	endTime := now.UnixMilli()

	params := url.Values{}
	params.Add("startTime", fmt.Sprintf("%d", startTime))
	params.Add("endTime", fmt.Sprintf("%d", endTime))
	params.Add("timestamp", fmt.Sprintf("%d", now.UnixMilli()))
	params.Add("recvWindow", "60000")

	sig := hmac.New(sha256.New, []byte(apiSecret))
	sig.Write([]byte(params.Encode()))
	signature := hex.EncodeToString(sig.Sum(nil))

	fullURL := fmt.Sprintf("%s?%s&signature=%s", baseURL, params.Encode(), signature)

	req, _ := http.NewRequest("GET", fullURL, nil)
	req.Header.Set("X-MEXC-APIKEY", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var records []struct {
		TxID   string `json:"txId"`
		Status int    `json:"status"`
		Amount string `json:"amount"`
	}

	if err := json.Unmarshal(body, &records); err != nil {
		return 0, false, nil // Если пришел не массив, значит транзакций нет или ошибка
	}

	for _, rec := range records {
		if strings.EqualFold(rec.TxID, targetTxID) && rec.Status == 5 {
			var val float64
			fmt.Sscanf(rec.Amount, "%f", &val)
			return val, true, nil
		}
	}

	return 0, false, nil
}

func SendMessageToActiveUsers(db *db.DB, chatID []int64, symbol, photoPath string, params scanner.PumpParams, count int) {
	bot, _ := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))

	for _, userID := range chatID {
		cfg, err := db.GetUserConfig(userID)
		if err != nil {
			fmt.Println(userID, ": err: ", err)
			continue
		}
		txt := scanner.FinalOutput(symbol, params, cfg)
		txt += fmt.Sprintf("24h count: %d", count)
		msg := tgbotapi.NewPhoto(userID, tgbotapi.FilePath(photoPath))
		msg.Caption = txt
		msg.ParseMode = "HTML"
		_, err = bot.Send(msg)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func NewSignalTracker(cooldown time.Duration) *SignalTracker {
	return &SignalTracker{
		signals:  make(map[string]*SignalInfo),
		lastDay:  time.Now().Day(),
		cooldown: cooldown,
	}
}

func (st *SignalTracker) CheckAndCount(symbol string) (canSend bool, countToday int) {
	st.mu.Lock()
	defer st.mu.Unlock()

	currentDay := time.Now().Day()

	if currentDay != st.lastDay {
		st.signals = make(map[string]*SignalInfo)
		st.lastDay = currentDay
	}

	info, exists := st.signals[symbol]
	if !exists {
		info = &SignalInfo{}
		st.signals[symbol] = info
	}

	if !info.LastSent.IsZero() && time.Since(info.LastSent) < st.cooldown {
		return false, info.CountToday
	}

	info.LastSent = time.Now()
	info.CountToday++

	return true, info.CountToday
}
