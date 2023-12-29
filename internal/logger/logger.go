package logger

import (
	"log/slog"
	"os"
	"sync"
)

var logger *slog.Logger
var once sync.Once

// I gives back an instance of the slog logger
func I() *slog.Logger {
	once.Do(func() {
		logger = slog.New(slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{
				AddSource: true,
			}),
		)
	})

	return logger
}
