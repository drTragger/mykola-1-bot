package commands

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func sendAndRun(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, text string, cmdName string, args ...string) {
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))

	cmd := exec.Command(cmdName, args...)
	if err := cmd.Run(); err != nil {
		log.Println("Помилка виконання:", err)
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Помилка: %v", err)))
	}
}

func getServiceName(input string) string {
	parts := strings.Fields(input)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}
