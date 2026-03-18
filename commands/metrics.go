package commands

import (
	"mykola-1-bot/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func MetricsCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	sysMetrics := utils.GetSystemMetrics()
	upsMetrics := utils.GetUpsStatus()

	reply := tgbotapi.NewMessage(msg.Chat.ID, sysMetrics+"\n\n"+upsMetrics)
	reply.ParseMode = "Markdown"
	bot.Send(reply)
}
