package commands

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func RestartServiceCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	service := getServiceName(msg.Text)
	if service == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❗ Формат: `/restart <service>`"))
		return
	}
	sendAndRun(bot, msg, fmt.Sprintf("♻ Перезапускаю сервіс `%s`...", service),
		"sudo", "systemctl", "restart", service)
}

func StopServiceCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	service := getServiceName(msg.Text)
	if service == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❗ Формат: `/stop <service>`"))
		return
	}
	sendAndRun(bot, msg, fmt.Sprintf("⏹ Зупиняю сервіс `%s`...", service),
		"sudo", "systemctl", "stop", service)
}

func StartServiceCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	service := getServiceName(msg.Text)
	if service == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❗ Формат: `/start <service>`"))
		return
	}
	sendAndRun(bot, msg, fmt.Sprintf("▶ Запускаю сервіс `%s`...", service),
		"sudo", "systemctl", "start", service)
}

func StatusServiceCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	parts := strings.Fields(msg.Text)
	if len(parts) < 2 {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Використання: /status <service>"))
		return
	}

	service := parts[1]
	cmd := exec.Command("systemctl", "status", service, "--no-pager", "--lines=5")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Помилка виконання:", err)
	}

	text := string(output)
	if len(text) > 4000 {
		text = text[:4000] + "\n... (обрізано)"
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("📌 Статус сервісу `%s`:\n```\n%s\n```", service, text))
	reply.ParseMode = "Markdown"

	bot.Send(reply)
}
