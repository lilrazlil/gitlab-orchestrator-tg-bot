package main

import (
	"gitlab-orchestrator-bot/auth"
	"gitlab-orchestrator-bot/config"
	"gitlab-orchestrator-bot/internal/notify"
	tg "gitlab-orchestrator-bot/telegram"
	"gitlab-orchestrator-bot/telegram/handlers"
	"gitlab-orchestrator-bot/telegram/middlewares"

	tele "gopkg.in/telebot.v3"
)

func main() {
	c := config.Config
	c.Inits()
	c.Print()

	auth.InitUsers()

	bot := tg.Init()
	notify.StartNotificationScheduler(bot)
	adminOnly := bot.Group()
	//Мидлвари
	bot.Use(middlewares.UserCheckMiddleware, middlewares.Logger)
	adminOnly.Use(middlewares.AdminCheckMiddleware, middlewares.Logger)
	//Команды

	bot.Handle("/createstand", handlers.CreateNameStandHandler)

	//Обработчики
	adminOnly.Handle(&tele.InlineButton{Unique: config.BtnDone}, handlers.SelectProductsStand)
	adminOnly.Handle(&tele.InlineButton{Unique: config.BtnTest}, handlers.SelectProductsStand)
	adminOnly.Handle(&tele.InlineButton{Unique: config.BtnCancel}, handlers.CatchHandler)
	adminOnly.Handle(&tele.InlineButton{Unique: config.BtnAddStand}, handlers.CatchHandler)
	adminOnly.Handle(&tele.InlineButton{Unique: config.BtnDoneStep2}, handlers.CatchHandler)

	bot.Handle(tele.OnText, handlers.CatchHandler)

	bot.Start()
	bot.Stop()
}
