package log

import (
	"go.uber.org/zap"
	"sync"
)

// Logger is a struct that representes a logger with certain methods
type Logger struct {
	zapLogger *zap.Logger
}

var instance *Logger
var once sync.Once

// GetLogger is a function to get a single instance of a zap logger
func GetLogger() *Logger {
	once.Do(func() {
		logger, _ := zap.NewProduction()
		instance = &Logger{
			zapLogger: logger,
		}
	})
	return instance
}

// Error logs an error to stdou with its value
func (logger *Logger) Error(message string, err error) {
	logger.zapLogger.Error(
		message,
		zap.Error(err),
	)
}

// SimpleError shows only a simple error string
func (logger *Logger) SimpleError(message string) {
	logger.zapLogger.Error(message)
}

// Fatal similar to SimpleError but it calls os.Exit(1)
func (logger *Logger) Fatal(message string) {
	logger.zapLogger.Fatal(message)
}

// FatalError logs a message and a golang error
func (logger *Logger) FatalError(message string, err error) {
	logger.zapLogger.Fatal(message, zap.Error(err))
}

// InitMessage logging function to show initialization of resources
func (logger *Logger) InitMessage(resource, message string) {
	logger.zapLogger.Info(
		"initialazing resource",
		zap.String("resource", resource),
		zap.String("info", message),
	)
}

// Sync flushes zap logger buffer
func (logger *Logger) Sync() {
	instance.zapLogger.Sync()
}
