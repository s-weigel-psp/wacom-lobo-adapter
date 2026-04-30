package logging

import (
	"log/slog"
	"os"
	"path/filepath"
)

// OpenLogger opens (or creates) the log file at
// %LOCALAPPDATA%\WacomBridge\logs\wacom-bridge.log and returns a slog.Logger
// with JSONHandler at Info level.
// The caller MUST close the returned *os.File on shutdown.
// If the log file cannot be opened (e.g., %LOCALAPPDATA% unavailable),
// OpenLogger falls back to stderr and returns nil for the file.
func OpenLogger() (*slog.Logger, *os.File, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
		return logger, nil, nil
	}
	logDir := filepath.Join(localAppData, "WacomBridge", "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
		return logger, nil, err
	}
	logPath := filepath.Join(logDir, "wacom-bridge.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
		return logger, nil, err
	}
	logger := slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return logger, f, nil
}
