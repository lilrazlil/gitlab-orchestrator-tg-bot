package logger

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// WithCaller добавляет информацию о вызывающей функции к логгеру
func WithCaller() Logger {
	pc, file, line, ok := runtime.Caller(2) // Пропускаем 2 фрейма: эту функцию и вызывающую функцию
	if !ok {
		return GetLogger().WithField("caller", "unknown")
	}

	// Получаем имя функции
	fn := runtime.FuncForPC(pc)
	var funcName string
	if fn == nil {
		funcName = "unknown"
	} else {
		funcName = fn.Name()
		// Обрезаем путь пакета
		if lastDot := strings.LastIndexByte(funcName, '.'); lastDot > 0 {
			funcName = funcName[lastDot+1:]
		}
	}

	// Получаем имя файла без полного пути
	fileName := filepath.Base(file)

	return GetLogger().WithField("caller", fmt.Sprintf("%s:%d:%s()", fileName, line, funcName))
}

// DebugWithCaller логирует с информацией о вызывающей функции
func DebugWithCaller(args ...interface{}) {
	WithCaller().Debug(args...)
}

// DebugfWithCaller логирует с информацией о вызывающей функции
func DebugfWithCaller(format string, args ...interface{}) {
	WithCaller().Debugf(format, args...)
}

// InfoWithCaller логирует с информацией о вызывающей функции
func InfoWithCaller(args ...interface{}) {
	WithCaller().Info(args...)
}

// InfofWithCaller логирует с информацией о вызывающей функции
func InfofWithCaller(format string, args ...interface{}) {
	WithCaller().Infof(format, args...)
}

// WarnWithCaller логирует с информацией о вызывающей функции
func WarnWithCaller(args ...interface{}) {
	WithCaller().Warn(args...)
}

// WarnfWithCaller логирует с информацией о вызывающей функции
func WarnfWithCaller(format string, args ...interface{}) {
	WithCaller().Warnf(format, args...)
}

// ErrorWithCaller логирует с информацией о вызывающей функции
func ErrorWithCaller(args ...interface{}) {
	WithCaller().Error(args...)
}

// ErrorfWithCaller логирует с информацией о вызывающей функции
func ErrorfWithCaller(format string, args ...interface{}) {
	WithCaller().Errorf(format, args...)
}

// FatalWithCaller логирует с информацией о вызывающей функции
func FatalWithCaller(args ...interface{}) {
	WithCaller().Fatal(args...)
}

// FatalfWithCaller логирует с информацией о вызывающей функции
func FatalfWithCaller(format string, args ...interface{}) {
	WithCaller().Fatalf(format, args...)
}
