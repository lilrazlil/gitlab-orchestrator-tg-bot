package handlers

import (
	"fmt"
	"gitlab-orchestrator-back/internal"
	"gitlab-orchestrator-back/internal/database"
	"gitlab-orchestrator-back/internal/logger"
	"net/http"

	"github.com/labstack/echo/v4"
)

const (
	StatusProcess = "in_process"
	StatusPending = "pending"
)

type Handler struct {
}

// GetNotification обработчик для получения первого неотправленного уведомления
// @Summary Получить неотправленное уведомление
// @Description Получает первое неотправленное уведомление для пользователя
// @Tags notifications
// @Accept json
// @Produce json
// @Param userID query int true "ID пользователя"
// @Success 200 {object} models.Notification
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /notify [get]
func (h *Handler) GetNotification(c echo.Context) error {
	logger.InfoWithCaller("Получение неотправленных уведомлений")

	notifications, err := database.GetNotifications()
	if err != nil {
		if err.Error() == "уведомления не найдены" {
			logger.InfoWithCaller("Неотправленные уведомления не найдены")
			return c.JSON(http.StatusNoContent, nil)
		}
		logger.ErrorfWithCaller("Ошибка при получении уведомлений: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	logger.InfofWithCaller("Найдено %d неотправленных уведомлений", len(notifications))
	return c.JSON(http.StatusOK, notifications)
}

// UpdateNotification обработчик для обновления статуса уведомления
// @Summary Обновить статус уведомления
// @Description Помечает уведомление как отправленное
// @Tags notifications
// @Accept json
// @Produce json
// @Param id path int true "ID уведомления"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /notify/{id} [patch]
func (h *Handler) UpdateNotification(c echo.Context) error {
	var request struct {
		NotificationID uint `json:"notificationID"`
	}
	if err := c.Bind(&request); err != nil {
		logger.ErrorfWithCaller("Ошибка при обработке запроса: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	logger.InfofWithCaller("Обновление статуса уведомления с ID: %d", request.NotificationID)

	err := database.MarkNotificationAsSent(request.NotificationID)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при обновлении уведомления %d: %v", request.NotificationID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	logger.InfofWithCaller("Уведомление %d успешно отмечено как отправленное", request.NotificationID)
	return c.JSON(http.StatusOK, nil)
}

// GetAllUsers обработчик для получения всех пользователей
// @Summary Получить всех пользователей
// @Description Получает список всех пользователей из базы данных
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {array} models.User
// @Failure 500 {object} map[string]string
// @Router /users [get]
func (h *Handler) GetAllUsers(c echo.Context) error {
	logger.InfoWithCaller("Запрос на получение списка пользователей")

	users, err := database.GetAllUsers()
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при получении списка пользователей: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	logger.InfofWithCaller("Успешно получен список пользователей (всего: %d)", len(users))
	return c.JSON(http.StatusOK, users)
}

// HealthCheck обработчик для проверки работоспособности API
// @Summary Проверка работоспособности API
// @Description Проверяет, что API работает корректно
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *Handler) HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GetSubos обработчик для получения списка продуктов (только имя и код)
// @Summary Получить список продуктов
// @Description Получает сокращенный список всех продуктов (код и название)
// @Tags subos
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /subos [get]
func (h *Handler) GetSubos(c echo.Context) error {
	subos, err := database.GetAllSubos()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	suboMap := make(map[string]string)
	for _, subo := range subos {
		suboMap[subo.Code] = subo.Name
	}

	return c.JSON(http.StatusOK, suboMap)
}

// GetAllSubos обработчик для получения полного списка продуктов
// @Summary Получить полный список продуктов
// @Description Получает полный список всех продуктов с полными данными
// @Tags subos
// @Accept json
// @Produce json
// @Success 200 {array} models.Subos
// @Failure 500 {object} map[string]string
// @Router /subos/all [get]
func (h *Handler) GetAllSubos(c echo.Context) error {
	subos, err := database.GetAllSubos()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, subos)
}

func (h *Handler) CreateStand(c echo.Context) error {
	var request struct {
		NameStand string   `json:"nameStand"`
		Products  []string `json:"products"`
		UserID    int64    `json:"userID"`
		Ref       string   `json:"ref"`
	}
	tx := *database.DB.Begin()

	if err := c.Bind(&request); err != nil {
		logger.ErrorfWithCaller("Ошибка при обработке запроса: %v", err)
		tx.Rollback()
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	standModel, err := internal.PopulateStand(request)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при заполнении модели стенда: %v", err)
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create stand"})
	}

	// Create the stand and get the created instance with ID
	if err = database.CreateStand(standModel, &tx); err != nil {
		logger.ErrorfWithCaller("Ошибка при создании стенда: %v", err)
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	tx.Commit()

	message := fmt.Sprintf("Стенд %s добавлен в очередь на создание", request.NameStand)
	logger.InfofWithCaller("Stand creation queued: %s", request.NameStand)

	return c.JSON(http.StatusOK, map[string]string{"message": message})
}

// GetAllStands обработчик для получения всех стендов
// @Summary Получить все стенды
// @Description Получает список всех стендов с их данными
// @Tags stands
// @Accept json
// @Produce json
// @Success 200 {array} models.Stand
// @Failure 500 {object} map[string]string
// @Router /stands [get]
func (h *Handler) GetAllStands(c echo.Context) error {
	stands, err := database.GetAllStands()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, stands)
}
