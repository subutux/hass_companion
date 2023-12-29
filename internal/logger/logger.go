package logger

import (
	"log/slog"
	"os"
	"sync"
)

var logger *slog.Logger
var once sync.Once

func I() *slog.Logger {
	once.Do(func() {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	})
	return logger
}
