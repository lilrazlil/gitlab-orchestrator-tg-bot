package auth

import (
	"gitlab-orchestrator-bot/client"
	"gitlab-orchestrator-bot/config"
	"log"
	"strings"
)

func InitUsers() {
	users, err := client.GetUsers()
	if err != nil {
		log.Printf("Ошибка при получении списка пользователей: %v", err)
		return
	}

	for _, user := range users {
		userID := int64(user["chat_id"].(float64))
		roles := strings.Split(user["role"].(string), ",")

		for _, role := range roles {
			if role == "admin" {
				config.AllowedAdmins[userID] = true
			}
			if role == "user" {
				config.AllowedUsers[userID] = true
			}
		}
	}
	log.Printf("Admins loaded: %+v", config.AllowedAdmins)
	log.Printf("Users loaded: %+v", config.AllowedUsers)

}
