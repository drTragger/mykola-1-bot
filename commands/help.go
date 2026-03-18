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
		"• `/metrics` — Показує ключові метрики (CPU, RAM, ZRAM/SWAP, диски, навантаження, температура, аптайм, IP, ping, Fail2Ban).\n\n" +
		"*👥 Сесії*\n" +
		"• `/logins` — Перелік активних користувачів (ім’я, TTY, час входу, спосіб: SSH/Local, IP/хост).\n\n" +
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
		"• `/update` — Оновлює пакети через apt (акуратний безвзаємодійний запуск).\n\n" +
		"_Нотатки_\n" +
		"• Команди доступні лише власнику.\n" +
		"• Для `/status /start /stop /restart` вказуй точну назву юніта, напр.: `nginx`, `php8.3-fpm`, `mykola-bot`.\n" +
		"• Спробуй `/help`."
}

// надсилає довгі тексти шматками < 4000 символів
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
