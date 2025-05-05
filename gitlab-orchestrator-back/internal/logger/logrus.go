package logger

import (
	"github.com/sirupsen/logrus"
	"io"
)

// LogrusLogger - реализация интерфейса Logger с использованием logrus
type LogrusLogger struct {
	logger *logrus.Logger
	entry  *logrus.Entry
}

// NewLogrusLogger создает новый логгер на основе logrus
func NewLogrusLogger() Logger {
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "@timestamp",
			logrus.FieldKeyLevel: "@level",
			logrus.FieldKeyMsg:   "@message",
			logrus.FieldKeyFunc:  "@caller",
		},
	})
	return &LogrusLogger{
		logger: l,
		entry:  logrus.NewEntry(l),
	}
}

// mapLogrusLevel преобразует наш уровень логирования в уровень logrus
func mapLogrusLevel(level Level) logrus.Level {
	switch level {
	case DebugLevel:
		return logrus.DebugLevel
	case InfoLevel:
		return logrus.InfoLevel
	case WarnLevel:
		return logrus.WarnLevel
	case ErrorLevel:
		return logrus.ErrorLevel
	case FatalLevel:
		return logrus.FatalLevel
	case PanicLevel:
		return logrus.PanicLevel
	default:
		return logrus.InfoLevel
	}
}

// Debug логирует сообщение на уровне Debug
func (l *LogrusLogger) Debug(args ...interface{}) {
	l.entry.Debug(args...)
}

// Debugf логирует форматированное сообщение на уровне Debug
func (l *LogrusLogger) Debugf(format string, args ...interface{}) {
	l.entry.Debugf(format, args...)
}

// Info логирует сообщение на уровне Info
func (l *LogrusLogger) Info(args ...interface{}) {
	l.entry.Info(args...)
}

// Infof логирует форматированное сообщение на уровне Info
func (l *LogrusLogger) Infof(format string, args ...interface{}) {
	l.entry.Infof(format, args...)
}

// Warn логирует сообщение на уровне Warn
func (l *LogrusLogger) Warn(args ...interface{}) {
	l.entry.Warn(args...)
}

// Warnf логирует форматированное сообщение на уровне Warn
func (l *LogrusLogger) Warnf(format string, args ...interface{}) {
	l.entry.Warnf(format, args...)
}

// Error логирует сообщение на уровне Error
func (l *LogrusLogger) Error(args ...interface{}) {
	l.entry.Error(args...)
}

// Errorf логирует форматированное сообщение на уровне Error
func (l *LogrusLogger) Errorf(format string, args ...interface{}) {
	l.entry.Errorf(format, args...)
}

// Fatal логирует сообщение на уровне Fatal
func (l *LogrusLogger) Fatal(args ...interface{}) {
	l.entry.Fatal(args...)
}

// Fatalf логирует форматированное сообщение на уровне Fatal
func (l *LogrusLogger) Fatalf(format string, args ...interface{}) {
	l.entry.Fatalf(format, args...)
}

// WithField возвращает новый логгер с добавленным полем
func (l *LogrusLogger) WithField(key string, value interface{}) Logger {
	return &LogrusLogger{
		logger: l.logger,
		entry:  l.entry.WithField(key, value),
	}
}

// WithFields возвращает новый логгер с добавленными полями
func (l *LogrusLogger) WithFields(fields map[string]interface{}) Logger {
	return &LogrusLogger{
		logger: l.logger,
		entry:  l.entry.WithFields(fields),
	}
}

// SetOutput устанавливает выходной поток для логгера
func (l *LogrusLogger) SetOutput(out io.Writer) {
	l.logger.SetOutput(out)
}

// SetLevel устанавливает уровень логирования
func (l *LogrusLogger) SetLevel(level Level) {
	l.logger.SetLevel(mapLogrusLevel(level))
}
