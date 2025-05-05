package middleware

import (
	"gitlab-orchestrator-back/internal/logger"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// RequestIDMiddleware добавляет уникальный ID для каждого запроса
func RequestIDMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		requestID := c.Request().Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			logger.DebugfWithCaller("Генерация нового request ID: %s", requestID)
		} else {
			logger.DebugfWithCaller("Используется существующий request ID: %s", requestID)
		}

		c.Set("request_id", requestID)
		c.Response().Header().Set("X-Request-ID", requestID)

		logger.InfofWithCaller("Обработка запроса %s %s (ID: %s)",
			c.Request().Method, c.Request().URL.Path, requestID)

		err := next(c)

		if err != nil {
			logger.ErrorfWithCaller("Ошибка при обработке запроса %s (ID: %s): %v",
				c.Request().URL.Path, requestID, err)
		} else {
			logger.InfofWithCaller("Успешная обработка запроса %s (ID: %s, Статус: %d)",
				c.Request().URL.Path, requestID, c.Response().Status)
		}

		return err
	}
}
