package middlewares

import (
	"fmt"
	"gitlab-orchestrator-bot/config"
	tele "gopkg.in/telebot.v3"
	"log"
)

func UserCheckMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if !config.AllowedUsers[c.Sender().ID] {
			return c.Send("Не авторизирован")
		}
		return next(c)
	}
}

func AdminCheckMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if !config.AllowedAdmins[c.Sender().ID] {
			return c.Send("Не авторизирован")
		}
		return next(c)
	}
}

func Logger(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		user := c.Sender().Username
		id := c.Sender().ID
		_, exist := config.UserStates[id]
		if !exist {
			config.UserStates[id] = &config.UserContext{}
		}
		role := ""
		if config.AllowedAdmins[id] {
			role += "admin "
		}
		if config.AllowedUsers[id] {
			role += "user "
		}

		if c.Update().Callback != nil {
			log.Println(fmt.Sprintf("user:\"%s\" id:\"%d\" role:\"%s\" выбрал кнопку %s", user, id, role, c.Update().Callback.Data))
			return next(c)
		}
		if c.Update().Message.Entities == nil {
			log.Println(fmt.Sprintf("user:\"%s\" id:\"%d\" role:\"%s\" написал текст %s", user, id, role, c.Message().Text))
			return next(c)
		}
		if c.Update().Message.Entities != nil {
			log.Println(fmt.Sprintf("user:\"%s\" id:\"%d\" role:\"%s\" написал команду %s", user, id, role, c.Update().Message.Text))
			return next(c)
		}
		return next(c)
	}
}
