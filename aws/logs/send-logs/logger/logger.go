package logger

import (
	"log"
	"os"
)

type Logger interface {
	Info(v ...interface {})
	Error(v ...interface {})
	Fatal(v ...interface {})
}

type logger struct {
	infoLogger log.Logger
	errorLogger log.Logger
}

func (l logger) Info(v ...interface {}) {
	l.infoLogger.Println(v...)
}

func (l logger) Error(v ...interface {}) {
	l.infoLogger.Println(v...)
}

func (l logger) Fatal(v ...interface {}) {
	l.Error(v...)
	os.Exit(1)
}

func NewLogger(prefix string) (Logger) {
	return &logger {
		infoLogger: *log.New(log.Writer(), prefix + " INFO ", log.Lmsgprefix),
		errorLogger: *log.New(log.Writer(), prefix + " ERROR ", log.Lmsgprefix),
	}
}
