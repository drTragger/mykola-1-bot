package commands

import (
	"mykola-1-bot/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func VpnCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, utils.GetVPNDetails())
	reply.ParseMode = "Markdown"
	bot.Send(reply)
}
