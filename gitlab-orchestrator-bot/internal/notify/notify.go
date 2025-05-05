package notify

import (
	"gitlab-orchestrator-bot/internal"
	"time"

	tele "gopkg.in/telebot.v3"
)

func StartNotificationScheduler(bot *tele.Bot) {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for range ticker.C {
			internal.CheckAndSendNotifications(bot)
		}
	}()
}
