package commands

import (
	"fmt"
	"log"
	"os/exec"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func RebootCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	sendAndRun(bot, msg, "🔄 Перезавантажую mykola-1...", "sudo", "/usr/sbin/reboot")
}

func ShutdownCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	sendAndRun(bot, msg, "🛑 Вимикаю mykola-1...", "sudo", "/usr/sbin/poweroff")
}

func UpdateCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	cmd := exec.Command("sudo", "apt-get", "update", "-y")
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := string(output)
		if len(text) > 3500 {
			text = text[:3500] + "\n... (обрізано)"
		}
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
			fmt.Sprintf("❌ Помилка (код %d):\n```\n%s\n```",
				cmd.ProcessState.ExitCode(), text)))
		return
	}

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Оновлення завершене"))
}

func ServicesCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	cmd := exec.Command("systemctl", "list-units", "--type=service", "--no-pager", "--state=running")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Помилка виконання:", err)
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Помилка: %v", err)))
		return
	}

	text := string(output)
	if len(text) > 4000 {
		text = text[:4000] + "\n... (обрізано)"
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, "📋 Запущені сервіси:\n```\n"+text+"\n```")
	reply.ParseMode = "Markdown"

	bot.Send(reply)
}
