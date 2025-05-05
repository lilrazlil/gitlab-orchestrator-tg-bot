package logger

import (
	"os"
)

// Init инициализирует логгер с настройками по умолчанию
func Init() {
	// Создаем новый логгер на основе logrus
	log := NewLogrusLogger()
	
	// Устанавливаем уровень логирования из переменной окружения
	// или используем INFO по умолчанию
	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "DEBUG":
		log.SetLevel(DebugLevel)
	case "WARN":
		log.SetLevel(WarnLevel)
	case "ERROR":
		log.SetLevel(ErrorLevel)
	default:
		log.SetLevel(InfoLevel)
	}
	
	// Устанавливаем вывод в stdout
	log.SetOutput(os.Stdout)
	
	// Устанавливаем как глобальный логгер
	SetLogger(log)
	
	Info("Logger initialized")
}
