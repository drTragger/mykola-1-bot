package commands

import (
	"mykola-1-bot/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// /simple — прості метрики контейнера (CPU/RAM за cgroup, диск /)
func SimpleMetricsCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	text := utils.GetSimpleMetrics()
	out := tgbotapi.NewMessage(msg.Chat.ID, text)
	out.ParseMode = "Markdown"
	out.DisableWebPagePreview = true
	_, _ = bot.Send(out)
}
