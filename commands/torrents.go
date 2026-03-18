package commands

import (
	"mykola-1-bot/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TorrentsCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	text := utils.GetTorrentsStatus()

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	bot.Send(reply)
}
