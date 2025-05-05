package telegram

import (
	"gitlab-orchestrator-bot/config"
	tele "gopkg.in/telebot.v3"
	"time"
)

func Init() *tele.Bot {
	c := config.Config
	pref := tele.Settings{
		Token:  c.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
		// Verbose: true,
	}
	bot, err := tele.NewBot(pref)
	if err != nil {
		panic(err)
	}
	return bot
}
