package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/eurefirma/wacom-bridge/internal/logging"
	"github.com/eurefirma/wacom-bridge/internal/messaging"
	"github.com/eurefirma/wacom-bridge/internal/state"
)

func main() {
	// 1. Open logger — falls back to stderr if %LOCALAPPDATA% unavailable (D-12)
	logger, logFile, _ := logging.OpenLogger()
	if logFile != nil {
		defer logFile.Close()
	}

	// 2. Verify baseline file exists at %LOCALAPPDATA%\WacomBridge\baseline.Export.wacomxs
	//    Return structured error and stay running if missing (D-12).
	//    Wacom XML logic is wired in Plan 02-02 — this is a stub check.
	localAppData := os.Getenv("LOCALAPPDATA")
	baselinePath := filepath.Join(localAppData, "WacomBridge", "baseline.Export.wacomxs")

	// 3. Init in-process state
	st := &state.State{}

	logger.Info("wacom-bridge started", slog.String("baseline", baselinePath))

	// 4. Command dispatch — Wacom handlers are stubs until Plan 02-02
	dispatch := func(msg map[string]interface{}) interface{} {
		cmd, _ := msg["command"].(string)
		logger.Info("command received", slog.String("command", cmd))

		switch cmd {
		case "set_mapping":
			// TODO (Plan 02-02): validate params and call wacom.SetMapping
			x, xOK := toInt(msg["x"])
			y, yOK := toInt(msg["y"])
			w, wOK := toInt(msg["width"])
			h, hOK := toInt(msg["height"])
			if !xOK || !yOK || !wOK || !hOK {
				return errResponse("missing or invalid x/y/width/height", "ERR_INVALID_PARAMS")
			}
			st.Set(x, y, w, h)
			logger.Info("set_mapping stub", slog.Int("x", x), slog.Int("y", y), slog.Int("w", w), slog.Int("h", h))
			return map[string]interface{}{"ok": true}

		case "reset_mapping":
			// TODO (Plan 02-02): call wacom.ResetMapping
			st.Reset()
			logger.Info("reset_mapping stub")
			return map[string]interface{}{"ok": true}

		case "get_status":
			return st.StatusResponse()

		case "ping":
			return map[string]interface{}{"ok": true}

		default:
			logger.Warn("unknown command", slog.String("command", cmd))
			return errResponse("unknown command: "+cmd, "ERR_UNKNOWN_COMMAND")
		}
	}

	_ = baselinePath // will be used in Plan 02-02

	// 5. Start message loop — blocks until browser closes port
	messaging.MessageLoop(dispatch)
}

func errResponse(msg, code string) map[string]interface{} {
	return map[string]interface{}{"error": msg, "code": code}
}

// toInt converts a JSON number (float64 from json.Unmarshal) to int.
func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	default:
		return 0, false
	}
}
