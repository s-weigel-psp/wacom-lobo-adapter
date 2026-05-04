package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/eurefirma/wacom-bridge/internal/logging"
	"github.com/eurefirma/wacom-bridge/internal/messaging"
	"github.com/eurefirma/wacom-bridge/internal/state"
	"github.com/eurefirma/wacom-bridge/internal/wacom"
)

func main() {
	// 1. Open logger — falls back to stderr if %LOCALAPPDATA% unavailable (D-12).
	logger, logFile, _ := logging.OpenLogger()
	if logFile != nil {
		defer logFile.Close()
	}

	// 2. Verify baseline file exists at %LOCALAPPDATA%\WacomBridge\baseline.Export.wacomxs.
	//    Do NOT exit if missing — stay running and return ERR_BASELINE_NOT_FOUND on
	//    set_mapping / reset_mapping commands (D-12).
	localAppData := os.Getenv("LOCALAPPDATA")
	baselinePath := filepath.Join(localAppData, "WacomBridge", "baseline.Export.wacomxs")
	if _, err := os.Stat(baselinePath); os.IsNotExist(err) {
		logger.Warn("baseline file missing at startup", slog.String("path", baselinePath))
		// Do NOT exit — stay running per D-12. Commands will return ERR_BASELINE_NOT_FOUND.
	}

	// 3. Init in-process state.
	st := &state.State{}

	logger.Info("wacom-bridge started", slog.String("baseline", baselinePath))

	// 4. Command dispatch — wired to wacom.SetMapping / wacom.ResetMapping (Plan 02-02).
	dispatch := func(msg map[string]interface{}) interface{} {
		cmd, _ := msg["command"].(string)
		logger.Info("command received", slog.String("command", cmd))

		switch cmd {
		case "set_mapping":
			x, xOK := toInt(msg["x"])
			y, yOK := toInt(msg["y"])
			w, wOK := toInt(msg["width"])
			h, hOK := toInt(msg["height"])
			if !xOK || !yOK || !wOK || !hOK {
				return errResponse("missing or invalid x/y/width/height", "ERR_INVALID_PARAMS")
			}
			result := wacom.SetMapping(logger, x, y, w, h)
			if _, ok := result["ok"]; ok {
				st.Set(x, y, w, h)
			}
			return result

		case "reset_mapping":
			result := wacom.ResetMapping(logger)
			if _, ok := result["ok"]; ok {
				st.Reset()
			}
			return result

		case "get_status":
			return st.StatusResponse()

		case "ping":
			return map[string]interface{}{"ok": true}

		default:
			logger.Warn("unknown command", slog.String("command", cmd))
			return errResponse("unknown command: "+cmd, "ERR_UNKNOWN_COMMAND")
		}
	}

	// 5. Start message loop — blocks until browser closes port.
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
