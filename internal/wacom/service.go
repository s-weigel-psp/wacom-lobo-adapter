//go:build windows

package wacom

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const wacomServiceName = "WtabletServicePro"
const serviceStopTimeout = 10 * time.Second

// RestartWacomService stops WtabletServicePro and starts it again via the Windows SCM.
// Returns nil on success, or a structured error map on failure.
//
// This function is provided for future use; needsServiceRestart is currently false
// (Task 1 findings: PrefUtil notifies the service via COM — no restart required).
func RestartWacomService(logger *slog.Logger) map[string]interface{} {
	m, err := mgr.Connect()
	if err != nil {
		logger.Error("SCM connect failed", slog.String("error", err.Error()))
		return errMap("cannot connect to Windows SCM: "+err.Error(), "ERR_SERVICE_CONNECT")
	}
	defer m.Disconnect()

	s, err := m.OpenService(wacomServiceName)
	if err != nil {
		logger.Error("service open failed",
			slog.String("service", wacomServiceName),
			slog.String("error", err.Error()))
		return errMap(fmt.Sprintf("cannot open service %q: %v", wacomServiceName, err), "ERR_SERVICE_NOT_FOUND")
	}
	defer s.Close()

	if _, err := s.Control(svc.Stop); err != nil {
		logger.Error("service stop failed", slog.String("error", err.Error()))
		code := "ERR_SERVICE_RESTART_FAILED"
		if isAccessDenied(err) {
			code = "ERR_SERVICE_ACCESS_DENIED"
		}
		return errMap("failed to stop service: "+err.Error(), code)
	}

	// Poll until stopped (timeout serviceStopTimeout).
	deadline := time.Now().Add(serviceStopTimeout)
	stopped := false
	for time.Now().Before(deadline) {
		status, err := s.Query()
		if err != nil {
			break
		}
		if status.State == svc.Stopped {
			stopped = true
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !stopped {
		return errMap("service did not stop within 10s", "ERR_SERVICE_RESTART_TIMEOUT")
	}

	if err := s.Start(); err != nil {
		logger.Error("service start failed", slog.String("error", err.Error()))
		return errMap("failed to start service: "+err.Error(), "ERR_SERVICE_RESTART_FAILED")
	}

	logger.Info("WtabletServicePro restarted successfully")
	return nil
}

// isAccessDenied checks whether an error represents a Windows Access Denied condition.
func isAccessDenied(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "access")
}
