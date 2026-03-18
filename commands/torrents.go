package commands

import (
	"fmt"
	"strconv"
	"strings"

	"mykola-1-bot/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TorrentsCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	text := utils.GetTorrentsStatus()

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	bot.Send(reply)
}

func PauseTorrentCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	index, err := getTorrentIndex(msg.Text)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Формат: `/pause <номер>`"))
		return
	}

	torrent, err := utils.FindTorrentByIndex(index)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ "+err.Error()))
		return
	}

	if err := utils.PauseTorrent(torrent.Hash); err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Не вдалося поставити на паузу: %v", err)))
		return
	}

	reply := tgbotapi.NewMessage(
		msg.Chat.ID,
		fmt.Sprintf("⏸ Поставив на паузу: *%s*", utils.EscapeMarkdown(torrent.Name)),
	)
	reply.ParseMode = "Markdown"
	bot.Send(reply)
}

func ResumeTorrentCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	index, err := getTorrentIndex(msg.Text)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Формат: `/resume <номер>`"))
		return
	}

	torrent, err := utils.FindTorrentByIndex(index)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ "+err.Error()))
		return
	}

	if err := utils.ResumeTorrent(torrent.Hash); err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Не вдалося відновити: %v", err)))
		return
	}

	reply := tgbotapi.NewMessage(
		msg.Chat.ID,
		fmt.Sprintf("▶️ Відновив: *%s*", utils.EscapeMarkdown(torrent.Name)),
	)
	reply.ParseMode = "Markdown"
	bot.Send(reply)
}

func getTorrentIndex(input string) (int, error) {
	parts := strings.Fields(input)
	if len(parts) < 2 {
		return 0, fmt.Errorf("missing index")
	}

	index, err := strconv.Atoi(parts[1])
	if err != nil || index <= 0 {
		return 0, fmt.Errorf("invalid index")
	}

	return index, nil
}
