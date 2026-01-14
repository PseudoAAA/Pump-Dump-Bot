package interfaces

// TelegramSender определяет методы для отправки сообщений в Telegram.
type TelegramSender interface {
	SendMessage(message string) error
}
