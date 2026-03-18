package commands

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HelpCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	text := helpText()
	sendChunked(bot, msg.Chat.ID, text)
}

func helpText() string {
	return "*🧭 Бот mykola-1 — Допомога*\n\n" +
		"*📡 Моніторинг*\n" +
		"• `/metrics` — Повні метрики системи: CPU, RAM, ZRAM/SWAP, SSD, температура, мережа, Raspberry Pi health, сервіси та UPS.\n" +
		"• `/simple` — Спрощені метрики системи.\n" +

		"*🎬 Торент*\n" +
		"• `/torrents` — Список торрентів у qBittorrent: статус, прогрес, швидкість завантаження та віддачі.\n\n" +

		"*👥 Сесії*\n" +
		"• `/logins` — Активні користувачі: ім’я, TTY, час входу, спосіб входу, IP/хост.\n\n" +

		"*🛠 Сервіси*\n" +
		"• `/services` — Список запущених systemd-сервісів.\n" +
		"• `/status <service>` — Стан сервісу + останні рядки журналу.\n" +
		"• `/start <service>` — Запустити сервіс.\n" +
		"• `/stop <service>` — Зупинити сервіс.\n" +
		"• `/restart <service>` — Перезапустити сервіс.\n\n" +

		"*⚙️ Керування системою*\n" +
		"• `/reboot` — Перезавантажити mykola-1.\n" +
		"• `/shutdown` — Вимкнути mykola-1.\n\n" +

		"*🧼 Обслуговування*\n" +
		"• `/update` — Оновити список пакетів і встановити доступні оновлення.\n\n" +

		"*📝 Нотатки*\n" +
		"• Усі команди доступні лише власнику бота.\n" +
		"• Для `/status`, `/start`, `/stop`, `/restart` вказуй точну назву systemd-юніта.\n" +
		"• Приклад: `/status jellyfin` або `/restart mykola-bot`.\n"
}

func sendChunked(bot *tgbotapi.BotAPI, chatID int64, text string) {
	const limit = 3900
	runes := []rune(text)

	for i := 0; i < len(runes); i += limit {
		j := i + limit
		if j > len(runes) {
			j = len(runes)
		}

		part := string(runes[i:j])
		msg := tgbotapi.NewMessage(chatID, part)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
	}
}
