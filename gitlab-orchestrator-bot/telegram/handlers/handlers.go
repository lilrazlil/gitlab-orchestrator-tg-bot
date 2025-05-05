package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"gitlab-orchestrator-bot/client"
	"gitlab-orchestrator-bot/config"
	"gitlab-orchestrator-bot/internal"
	"gitlab-orchestrator-bot/telegram/buttons"

	tele "gopkg.in/telebot.v3"
)

func fetchSubos() (map[string]string, error) {
	suboMap, err := client.FetchSubos()
	if err != nil {
		return nil, err
	}
	return suboMap, nil
}

func CreateNameStandHandler(c tele.Context) error {
	user := config.UserStates[c.Sender().ID]
	user.WaitingForMessageStand = true
	internal.DropWaitingMessages(user)
	err := c.Send("Вы хотите создать стенд\nНапишите его название\nНапример: sf-31")
	if err != nil {
		return err
	}
	user.WaitingForMessageStand = true
	return nil
}

func CreateButtonsVerify(data string, unique string) *tele.ReplyMarkup {
	var row []tele.InlineButton
	opts := " " + data
	markup := &tele.ReplyMarkup{}
	names := []map[string]string{
		{"text": "Да", "data": "yes"},
		{"text": "Нет", "data": "no"},
	}
	for _, v := range names {
		button := tele.InlineButton{Unique: unique, Text: v["text"], Data: v["data"] + opts}
		row = append(row, button)
	}
	markup.InlineKeyboard = append(markup.InlineKeyboard, row)
	return markup
}

func NameToCodeSubo(FilterSubos map[string]bool, subos map[string]string) []string {
	var selectedProducts []string
	// For each selected product in FilterSubos
	for product := range FilterSubos {
		// Compare with values in fetchSubos map
		for suboKey, suboValue := range subos {
			if suboValue == product {
				// If match found, add the key from fetchSubos to products
				selectedProducts = append(selectedProducts, suboKey)
				break
			}
		}
	}
	return selectedProducts
}

func CreateStand(c *config.UserContext, userID int64) (string, error) {
	// Get all available subos
	subos, err := fetchSubos()
	if err != nil {
		return "", err
	}

	// Create a slice for selected products
	selectedProducts := NameToCodeSubo(c.FilterSubos, subos)

	standData := config.StandData{
		NameStand: c.CreateStandName,
		Products:  selectedProducts,
		UserID:    userID,
		Ref:       "master",
	}

	jsonData, err := json.Marshal(&standData)
	if err != nil {
		return "", err
	}

	return client.SendStandToBackend(jsonData)
}

// UniversalHandler handles both text messages and callbacks
func CatchHandler(c tele.Context) error {
	userID := c.Sender().ID
	user := config.UserStates[userID]

	// Handle callback queries (button presses)
	if c.Callback() != nil {
		data := getCallbackData(c)

		// Handle stand name confirmation
		if user.WaitingForMessageStand && strings.HasPrefix(data, "yes ") {
			name := strings.Split(c.Callback().Data, " ")
			user.CreateStandName = name[1]
			user.WaitingForMessageStand = false
			return SelectProductsStand(c)
		}

		// Handle final stand creation approval
		if user.WaitingApproveCreateStand && strings.HasPrefix(data, "yes ") {
			response, err := CreateStand(user, userID)
			if err != nil {
				return c.Send(fmt.Sprintf("Ошибка при создании стенда: %v", err))
			}
			c.Edit(fmt.Sprintf("Вы выбрали создать стенд %s%s с продуктами\n%v", user.CreateStandName, config.Config.Domain, user.FilterSubos))
			c.Send(response)
			user.WaitingApproveCreateStand = false
			return nil
		}

		// Handle cancellations
		if strings.HasPrefix(data, "no ") || strings.HasPrefix(data, "cancel") {
			internal.DropWaitingMessages(user)
			return c.Edit("Отмена")
		}

		// Handle other callbacks (product selection etc.)
		return SelectProductsStand(c)
	}

	// Handle text messages
	message := c.Message().Text

	// Handle stand name input
	if user.WaitingForMessageStand {
		standName := internal.CutSpecAndSpaceCimbols(message)
		markup := CreateButtonsVerify(standName, config.BtnAddStand)
		return c.Send("Название стенда: "+standName+config.Config.Domain+"\nВы хотите создать стенд с таким названием?", markup)
	}

	// Default response for other text messages
	return c.Send(config.StartMessage)
}

func SelectProductsStand(c tele.Context) error {
	userID := c.Sender().ID
	user := config.UserStates[userID]

	data := getCallbackData(c)

	if user.FilterSubos == nil || data == "" {
		user.FilterSubos = make(map[string]bool)
	}

	if data == "done" {
		if len(user.FilterSubos) == 0 {
			c.Respond(&tele.CallbackResponse{
				Text:      "⚠️Вы не выбрали продукты для стенда",
				ShowAlert: true, // true для модального окна, false для маленького уведомления
			})

			return nil
		}

		return handleDoneSelection(c, user)
	}

	if data != "" && !strings.HasPrefix(data, "yes ") {
		updateFilterSubos(user, data)
	}

	return sendUpdatedKeyboard(c, user)
}

// Получение данных из Callback или команды
func getCallbackData(c tele.Context) string {
	if c.Callback() != nil {
		return c.Callback().Data
	}
	return "" // Если это команда, data будет пустой
}

// Обработка завершения выбора
func handleDoneSelection(c tele.Context, user *config.UserContext) error {
	markup := CreateButtonsVerify("test", config.BtnDoneStep2)
	err := c.Edit(fmt.Sprintf("Вы хотите создать стенд %s%s с такими продуктами?\n%v", user.CreateStandName, config.Config.Domain, user.FilterSubos), markup)
	if err != nil {
		return err
	}
	user.WaitingApproveCreateStand = true
	return nil
}

// Обновление состояния FilterSubos
func updateFilterSubos(user *config.UserContext, data string) {
	if _, exists := user.FilterSubos[data]; exists {
		// Если продукт уже в фильтре, удаляем его
		delete(user.FilterSubos, data)
	} else {
		// Если продукта нет в фильтре, добавляем его
		user.FilterSubos[data] = true
	}
}

// Создание и отправка обновленной клавиатуры
func sendUpdatedKeyboard(c tele.Context, user *config.UserContext) error {
	subos, err := fetchSubos()
	if err != nil {
		return fmt.Errorf("ошибка при получении списка продуктов: %v", err)
	}

	// Создаем клавиатуру
	markup := buttons.СreateKeyboard(subos, user)
	// Если это команда, отправляем новое сообщение, иначе редактируем существующее
	if c.Callback() != nil {
		return c.Edit(fmt.Sprintf("Вы выбрали создать стенд с названием: %s\nТеперь вы можете выбрать продукты, которые должны быть на этом стенде", user.CreateStandName), markup)
	}
	return c.Send(fmt.Sprintf("Вы выбрали создать стенд с названием: %s\nТеперь вы можете выбрать продукты, которые должны быть на этом стенде", user.CreateStandName), markup)
}
