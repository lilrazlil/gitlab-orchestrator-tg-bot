package logger

import "io"

// Logger представляет интерфейс для логирования
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	SetOutput(out io.Writer)
	SetLevel(level Level)
}

// Level представляет уровень логирования
type Level int

// Уровни логирования
const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	PanicLevel
)

var globalLogger Logger

// GetLogger возвращает текущий глобальный логгер
func GetLogger() Logger {
	if globalLogger == nil {
		globalLogger = NewLogrusLogger() // Логгер по умолчанию
	}
	return globalLogger
}

// SetLogger устанавливает глобальный логгер
func SetLogger(l Logger) {
	globalLogger = l
}

// Debug логирует сообщение на уровне Debug
func Debug(args ...interface{}) {
	GetLogger().Debug(args...)
}

// Debugf логирует форматированное сообщение на уровне Debug
func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

// Info логирует сообщение на уровне Info
func Info(args ...interface{}) {
	GetLogger().Info(args...)
}

// Infof логирует форматированное сообщение на уровне Info
func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

// Warn логирует сообщение на уровне Warn
func Warn(args ...interface{}) {
	GetLogger().Warn(args...)
}

// Warnf логирует форматированное сообщение на уровне Warn
func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

// Error логирует сообщение на уровне Error
func Error(args ...interface{}) {
	GetLogger().Error(args...)
}

// Errorf логирует форматированное сообщение на уровне Error
func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

// Fatal логирует сообщение на уровне Fatal
func Fatal(args ...interface{}) {
	GetLogger().Fatal(args...)
}

// Fatalf логирует форматированное сообщение на уровне Fatal
func Fatalf(format string, args ...interface{}) {
	GetLogger().Fatalf(format, args...)
}

// WithField возвращает новый логгер с добавленным полем
func WithField(key string, value interface{}) Logger {
	return GetLogger().WithField(key, value)
}

// WithFields возвращает новый логгер с добавленными полями
func WithFields(fields map[string]interface{}) Logger {
	return GetLogger().WithFields(fields)
}

// SetOutput устанавливает выходной поток для логгера
func SetOutput(out io.Writer) {
	GetLogger().SetOutput(out)
}

// SetLevel устанавливает уровень логирования
func SetLevel(level Level) {
	GetLogger().SetLevel(level)
}
