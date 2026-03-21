package commands

import (
	"mykola-1-bot/config"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var registry = map[string]func(*tgbotapi.BotAPI, *tgbotapi.Message){
	"/metrics":  MetricsCommand,
	"/simple":   SimpleMetricsCommand,
	"/torrents": TorrentsCommand,
	"/vpn":      VpnCommand,
	"/reboot":   RebootCommand,
	"/shutdown": ShutdownCommand,
	"/update":   UpdateCommand,
	"/services": ServicesCommand,
	"/logins":   LoggedInUsersCommand,
	"/help":     HelpCommand,
}

var dynamicCommands = []string{"/restart", "/stop", "/start", "/status"}

func HandleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if msg == nil || msg.From == nil {
		return
	}

	if msg.From.ID != config.Cfg.Bot.OwnerId {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Тобі не дозволено виконувати цю команду."))
		return
	}

	if handler, exists := registry[msg.Text]; exists {
		handler(bot, msg)
		return
	}

	for _, prefix := range dynamicCommands {
		if strings.HasPrefix(msg.Text, prefix+" ") {
			switch prefix {
			case "/restart":
				RestartServiceCommand(bot, msg)
			case "/stop":
				StopServiceCommand(bot, msg)
			case "/start":
				StartServiceCommand(bot, msg)
			case "/status":
				StatusServiceCommand(bot, msg)
			}
			return
		}
	}
}
