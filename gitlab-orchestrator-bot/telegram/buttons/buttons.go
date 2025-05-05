package buttons

import (
	"fmt"
	"gitlab-orchestrator-bot/config"
	"gitlab-orchestrator-bot/internal"
	"sort"

	tele "gopkg.in/telebot.v3"
)

// Создание клавиатуры с учетом состояния FilterSubos
func СreateKeyboard(subos map[string]string, user *config.UserContext) *tele.ReplyMarkup {
	var row []tele.InlineButton
	markup := &tele.ReplyMarkup{}

	// Create a slice of code-name pairs for sorting
	var items []struct {
		Code string
		Name string
	}
	for code, name := range subos {
		items = append(items, struct {
			Code string
			Name string
		}{code, name})
	}

	// Sort by name
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	for i, item := range items {
		emoji := internal.GetEmoji(user, item.Name)
		name := fmt.Sprintf("%d. %s", i+1, item.Name)
		button := tele.InlineButton{Unique: config.BtnTest, Text: fmt.Sprintf("%s %s", emoji, name), Data: item.Name}
		row = append(row, button)

		if (i+1)%config.NumberOfLinesSubos == 0 {
			markup.InlineKeyboard = append(markup.InlineKeyboard, row)
			row = []tele.InlineButton{}
		}
	}

	// Добавляем оставшиеся кнопки в последнюю строку
	if len(row) > 0 {
		markup.InlineKeyboard = append(markup.InlineKeyboard, row)
	}

	// Добавляем кнопку "Готово" в отдельную строку
	doneButton := tele.InlineButton{Unique: config.BtnDone, Text: "✅ Готово", Data: "done"}
	cancelButton := tele.InlineButton{Unique: config.BtnCancel, Text: "❌ Отмена", Data: "cancel"}
	markup.InlineKeyboard = append(markup.InlineKeyboard, []tele.InlineButton{doneButton, cancelButton})
	return markup
}
