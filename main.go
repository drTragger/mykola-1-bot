package main

import (
	"log"
	"mykola-1-bot/commands"
	"mykola-1-bot/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	config.LoadConfig("config.toml")

	bot, err := tgbotapi.NewBotAPI(config.Cfg.Bot.Token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			commands.HandleCommand(bot, update.Message)
		}
	}
}
