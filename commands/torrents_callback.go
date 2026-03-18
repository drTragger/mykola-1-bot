package commands

import (
	"strings"

	"mykola-1-bot/config"
	"mykola-1-bot/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	if callback == nil || callback.From == nil {
		return
	}

	bot.Request(tgbotapi.NewCallback(callback.ID, ""))

	if callback.From.ID != config.Cfg.Bot.OwnerId {
		bot.Request(tgbotapi.NewCallback(callback.ID, "⛔ Недостатньо прав"))
		return
	}

	data := callback.Data

	switch {
	case data == "torrent:refresh":
		updateTorrentsMessage(bot, callback, "🔄 Оновлено")
		return

	case strings.HasPrefix(data, "torrent:pause:"):
		hash := strings.TrimPrefix(data, "torrent:pause:")
		utils.PauseTorrent(hash)
		updateTorrentsMessage(bot, callback, "⏸ Пауза")

	case strings.HasPrefix(data, "torrent:resume:"):
		hash := strings.TrimPrefix(data, "torrent:resume:")
		utils.ResumeTorrent(hash)
		updateTorrentsMessage(bot, callback, "▶️ Відновлено")
	}
}

func updateTorrentsMessage(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, notice string) {
	torrents, err := utils.GetSortedTorrents()
	if err != nil {
		answer := tgbotapi.NewCallback(callback.ID, "❌ Помилка qBittorrent")
		bot.Request(answer)
		return
	}

	text := buildTorrentsText(torrents)
	keyboard := buildTorrentsKeyboard(torrents)

	edit := tgbotapi.NewEditMessageTextAndMarkup(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		text,
		keyboard,
	)
	edit.ParseMode = "Markdown"

	bot.Send(edit)

	answer := tgbotapi.NewCallback(callback.ID, notice)
	bot.Request(answer)
}
