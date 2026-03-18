package commands

import (
	"fmt"
	"strings"

	"mykola-1-bot/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TorrentsCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	torrents, err := utils.GetSortedTorrents()
	if err != nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Помилка qBittorrent: %v", err))
		bot.Send(reply)
		return
	}

	text := buildTorrentsText(torrents)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = buildTorrentsKeyboard(torrents)

	bot.Send(reply)
}

func buildTorrentsText(torrents []utils.Torrent) string {
	if len(torrents) == 0 {
		return "📭 *qBittorrent*\n\nНемає торрентів."
	}

	total := len(torrents)
	activeDownloading := 0
	activeUploading := 0

	var totalDl int64
	var totalUl int64

	for _, t := range torrents {
		totalDl += t.Dlspeed
		totalUl += t.Upspeed

		if utils.IsDownloadingState(t.State) {
			activeDownloading++
		}
		if utils.IsUploadingState(t.State) {
			activeUploading++
		}
	}

	var b strings.Builder
	b.WriteString("🎬 *qBittorrent*\n\n")
	b.WriteString(fmt.Sprintf(
		"📦 *Всього:* %d\n⬇️ *Завантажуються:* %d\n⬆️ *Роздаються:* %d\n🚀 *Швидкість:* ↓ %s | ↑ %s\n\n",
		total,
		activeDownloading,
		activeUploading,
		utils.FormatSpeedOrDash(totalDl),
		utils.FormatSpeedOrDash(totalUl),
	))

	limit := 10
	for i, t := range torrents {
		if i >= limit {
			b.WriteString(fmt.Sprintf("\n_...і ще %d торрент(ів)_", len(torrents)-limit))
			break
		}

		index := i + 1
		name := fmt.Sprintf("%d. %s", index, utils.EscapeMarkdown(utils.Truncate(t.Name, 45)))
		progress := int(t.Progress * 100)
		size := utils.FormatBytesIECInt64(utils.EffectiveSize(t.Size, t.TotalSize))
		state := utils.MapState(t.State)
		bar := utils.ProgressBar(progress, 10)

		line := fmt.Sprintf(
			"*%s*\n%s %d%% • %s\n⬇️ %s • ⬆️ %s",
			name,
			bar,
			progress,
			state,
			utils.FormatSpeedOrDash(t.Dlspeed),
			utils.FormatSpeedOrDash(t.Upspeed),
		)

		if size != "0 B" {
			line += fmt.Sprintf(" • %s", size)
		}

		if utils.IsDownloadingState(t.State) && t.Eta > 0 {
			line += fmt.Sprintf("\n⏳ ETA: %s", utils.FormatETA(t.Eta))
		}

		if t.NumSeeds > 0 || t.NumLeechs > 0 {
			line += fmt.Sprintf("\n🌱 %d сидів • 🧲 %d качають", t.NumSeeds, t.NumLeechs)
		}

		if t.NumSeeds == 0 && utils.IsDownloadingState(t.State) {
			line += "\n⚠️ Немає сидів — може не скачатися"
		}

		b.WriteString(line + "\n\n")
	}

	return b.String()
}

func buildTorrentsKeyboard(torrents []utils.Torrent) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0)

	limit := 10
	for i, t := range torrents {
		if i >= limit {
			break
		}

		label := fmt.Sprintf("%d", i+1)

		var button tgbotapi.InlineKeyboardButton
		if utils.IsPausedState(t.State) {
			button = tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("▶️ %s", label),
				"torrent:resume:"+t.Hash,
			)
		} else {
			button = tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("⏸ %s", label),
				"torrent:pause:"+t.Hash,
			)
		}

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}

	if len(rows) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Оновити", "torrent:refresh"),
		))
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
