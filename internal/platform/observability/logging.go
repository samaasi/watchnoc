package observability

import (
	"log/slog"
	"os"
)

func InitLogger(debug bool) *slog.Logger {
	var opts slog.HandlerOptions
	if debug {
		opts.Level = slog.LevelDebug
	} else {
		opts.Level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
