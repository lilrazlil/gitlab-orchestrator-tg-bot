package internal

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gitlab-orchestrator-bot/client"
	"gitlab-orchestrator-bot/config"
	"gopkg.in/telebot.v3"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jedib0t/go-pretty/v6/table"
)

func SendNotifications(notification client.Notifications, bot *telebot.Bot) error {
	var numbers = map[int]string{
		1: "первом",
		2: "втором",
		3: "третьем",
	}
	var status = map[string]string{
		"success": "успешно",
		"error":   "с ошибкой",
	}
	succeedMessage := fmt.Sprintf("Уведомление об %s этапе создания стенда %s\nЭтап прошел %s: %s", numbers[notification.Order], notification.StandName, status[notification.Status], notification.StepName)
	if notification.Status == "error" {
		succeedMessage += fmt.Sprintf("\nСоздание стенда %s завершилось с ошибкой на одном из этапов\nРабота по созданию стенда завершена\nДля возобновления работы нужно...", notification.StandName)
	}
	_, err := bot.Send(&telebot.User{ID: notification.UserID}, succeedMessage)
	if err != nil {
		return err
	}
	return nil
}

func CheckAndSendNotifications(bot *telebot.Bot) {
	notifications, err := client.FetchNotifications()
	if err != nil {
		logrus.Errorf("Ошибка при получении уведомлений: %v", err)
		return
	}
	if len(notifications) == 0 {
		return
	}

	c := make(chan struct{}, 20)
	go func() {
		// Пытаемся отправить значение в канал
		for _, notification := range notifications {
			select {
			case c <- struct{}{}: // если канал свободен
				if err = SendNotifications(notification, bot); err != nil {
					logrus.Errorf("Ошибка при отправке сообщения: %v", err)
				}
				if err = client.SendMarkNotification(notification.ID); err != nil {
					logrus.Errorf("Ошибка при отправке патча уведомления: %v", err)
				}
				<-c // освобождаем канал после завершения
			default: // если канал занят
				logrus.Info("Канал занят")
			}
		}

	}()
}

func CutSpecAndSpaceCimbols(name string) string {
	//Чтобы у строки не было спецсимовлов и пробелов
	var re = regexp.MustCompile(`[^\w-]`)
	refactorName := re.ReplaceAllString(name, "")
	return refactorName
}

func RemoveFirstSpaces(inputString string) string {
	re := regexp.MustCompile(`^\s+`)
	output := re.ReplaceAllString(inputString, "")
	return output
}

func CapitalizeFirstLetter(name string) string {
	if name == "" {
		return name
	}
	r, size := utf8.DecodeRuneInString(name)
	return string(unicode.ToUpper(r)) + name[size:]
}

func CreateCompareText(stands []string, subos []string) []string {
	var tables []string

	allStandsData, err := client.FetchAllStands()
	if err != nil {
		return []string{fmt.Sprintf("Ошибка получения данных о стендах: %v", err)}
	}

	allStands := make(map[string]map[string]string)
	for _, standData := range allStandsData {
		name := standData["name"].(string)
		deploymentsData, ok := standData["deployments"].(map[string]interface{})
		if !ok {
			continue
		}

		deployments := make(map[string]string)
		for k, v := range deploymentsData {
			deployments[k] = v.(string)
		}
		allStands[name] = deployments
	}

	for _, subo := range subos {
		collector := make(map[string][]interface{})
		var header []interface{}
		header = append(header, "Продукты/стенды")

		for iname, vname := range stands {
			header = append(header, vname)

			deployments := allStands[vname]
			if deployments != nil {
				for keyDeployment, valueDeployments := range deployments {
					if strings.HasPrefix(keyDeployment, subo) {
						fullImage := strings.SplitAfter(valueDeployments, "/")
						splitImage := strings.Split(fullImage[len(fullImage)-1], ":")
						image, tag := splitImage[0], splitImage[1]

						if collector[image] == nil {
							collector[image] = make([]interface{}, len(stands)+1)
							collector[image][0] = image
						}
						collector[image][iname+1] = " ✅ " + tag
					}
				}
			}
		}

		t := table.NewWriter()
		t.AppendHeader(header)
		for _, row := range collector {
			for i := 1; i < len(stands)+1; i++ {
				if row[i] == nil {
					row[i] = " "
				}
				if !checkEqual(row) {
					fillArray(row)
				}
			}

			t.AppendRow(row)
			t.AppendSeparator()
		}
		t.SetStyle(table.StyleRounded)
		t.SortBy([]table.SortBy{
			{Number: 1, Mode: table.Asc},
		})
		tables = append(tables, t.Render())
	}
	return tables
}

func checkEqual(arr []interface{}) bool {
	if len(arr) == 0 {
		return true
	}
	firstElement := arr[1]
	for i, elem := range arr {
		if i == 0 {
			continue
		}
		if elem != firstElement {
			return false
		}
	}
	return true
}

func fillArray(arr []interface{}) {
	for i := 1; i < len(arr); i++ {
		if arr[i] == nil {
			continue
		}
		arr[i] = strings.Replace(arr[i].(string), " ✅ ", " ⛔ ", 1)
	}
}

func GetEmoji(user *config.UserContext, name string) string {
	if _, exist := user.FilterSubos[name]; exist {
		return "✅"
	}
	return "⬜"
}

func DropWaitingMessages(user *config.UserContext) {
	user.WaitingForMessageStand = false
	user.WaitingApproveCreateStand = false
	user.FilterSubos = nil
}
