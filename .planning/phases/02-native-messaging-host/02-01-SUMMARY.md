---
phase: 02-native-messaging-host
plan: "01"
subsystem: native-host
tags: [go, native-messaging, protocol, windows, binary-framing]
dependency_graph:
  requires: []
  provides:
    - docs/protocol.md — stable JSON interface contract for Phase 3 extension development
    - go.mod — Go module definition (github.com/eurefirma/wacom-bridge, go 1.25.0)
    - cmd/wacom-bridge/main.go — entry point with logging, baseline check stub, message loop
    - internal/messaging/host.go — ReadMessage/WriteMessage/MessageLoop + 1 MB security guard
    - internal/messaging/init_windows.go — Windows binary mode init via msvcrt.dll _setmode
    - internal/logging/logging.go — slog JSONHandler to %LOCALAPPDATA%\WacomBridge\logs\
    - internal/state/state.go — in-process mapping state with StatusResponse
  affects: []
tech_stack:
  added:
    - Go 1.25.0 (module go 1.21 minimum satisfied)
    - golang.org/x/sys v0.43.0 (Windows API: NewLazySystemDLL, NewProc)
    - log/slog stdlib (JSONHandler for structured logging)
    - encoding/binary (4-byte LE framing)
  patterns:
    - Native Messaging read/write loop with 4-byte little-endian uint32 framing
    - Windows binary mode init via msvcrt.dll _setmode in //go:build windows init function
    - Structured error responses — host stays running on errors (D-12)
    - Command dispatch via switch statement (4 commands)
    - slog JSONHandler with stderr fallback when %LOCALAPPDATA% unavailable
key_files:
  created:
    - docs/protocol.md
    - go.mod
    - go.sum
    - cmd/wacom-bridge/main.go
    - internal/messaging/host.go
    - internal/messaging/init_windows.go
    - internal/logging/logging.go
    - internal/state/state.go
  modified: []
decisions:
  - "Used msvcrt.dll _setmode (not SetConsoleMode) for binary mode on pipe handles — Chrome/Edge spawn the host with pipe handles, not console handles. RESEARCH.md Open Question 3 resolved."
  - "go get golang.org/x/sys@latest upgraded go directive from 1.21 to 1.25.0 — expected Go toolchain behavior. slog availability requirement (>= 1.21) satisfied."
  - "Added 1 MB message size guard in ReadMessage (T-02-01-01/03) — zero-length and oversized messages return ERR_INVALID_PARAMS and host stays running per D-12."
metrics:
  duration_minutes: 3
  completed_date: "2026-04-30"
  tasks_completed: 2
  files_created: 8
---

# Phase 2 Plan 01: Go Module Scaffold and Native Messaging I/O Summary

**One-liner:** Go module scaffolded with 4-byte LE Native Messaging framing, Windows binary mode init via msvcrt.dll _setmode, slog JSON logging, and stable docs/protocol.md contract.

---

## What Was Built

### Task 1: docs/protocol.md
The authoritative JSON interface contract for Phase 3 parallel development. Covers:
- Wire framing (4-byte LE uint32 length prefix + UTF-8 JSON body)
- All four command request shapes (set_mapping, reset_mapping, get_status, ping)
- All response shapes including get_status with six-field shape and monitor:null reserved field
- Complete error code table (11 codes)
- Physical pixel coordinate semantics with DPR note
- Host lifecycle (EOF clean shutdown, stay running on errors)
- Native messaging manifest format with allowed_origins (no wildcards)

### Task 2: Go module and source files
- **go.mod / go.sum:** Module `github.com/eurefirma/wacom-bridge` with `golang.org/x/sys v0.43.0`
- **internal/messaging/host.go:** `ReadMessage` / `WriteMessage` with 4-byte LE framing; `MessageLoop` with EOF→exit(0), read error→ERR_INVALID_PARAMS stay running; 1 MB max message guard
- **internal/messaging/init_windows.go:** `//go:build windows` init function calls `msvcrt.dll _setmode` to switch stdin/stdout to binary mode (O_BINARY=0x8000) — prevents 0x0A byte corruption in the 4-byte length prefix when Chrome/Edge use pipe handles
- **internal/logging/logging.go:** `OpenLogger` returns slog.Logger writing to `%LOCALAPPDATA%\WacomBridge\logs\wacom-bridge.log` with JSONHandler; falls back to stderr if LOCALAPPDATA unavailable
- **internal/state/state.go:** `State` struct with mutex; `Set`/`Reset`/`StatusResponse`; monitor field always nil (MULTI-01 deferred)
- **cmd/wacom-bridge/main.go:** Entry point: init logging → baseline path setup → init state → message loop with switch dispatch (set_mapping, reset_mapping, get_status, ping); Wacom XML handlers are stubs returning `{"ok":true}` until Plan 02-02

Cross-compiles successfully: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/wacom-bridge/`

---

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Security] Added 1 MB message size guard in ReadMessage**
- **Found during:** Task 2 implementation (threat model T-02-01-01 and T-02-01-03)
- **Issue:** The plan's task description had a ReadMessage with no length validation. The threat model required mitigating zero-length prefix (hang/panic) and oversized messages (memory exhaustion).
- **Fix:** Added `maxMessageSize = 1 MB` constant; guard rejects length == 0 and length > maxMessageSize with ERR_INVALID_PARAMS; MessageLoop continues running (D-12 compliance).
- **Files modified:** `internal/messaging/host.go`
- **Commit:** 9f1791f

**2. [Rule 1 - Deviation] go directive upgraded from 1.21 to 1.25.0 by go toolchain**
- **Found during:** Task 2, `go get golang.org/x/sys@latest`
- **Issue:** `go get` upgraded the go directive to 1.25.0 (the installed toolchain version). The acceptance criterion `grep "go 1.21" go.mod` will fail literally.
- **Fix:** Accepted upgrade — 1.25.0 satisfies the intent (slog available since 1.21). Documented as known deviation. The criterion's intent (slog stdlib availability) is fully satisfied.
- **Files modified:** `go.mod`
- **Commit:** 9f1791f

---

## Known Stubs

| Stub | File | Line | Reason |
|------|------|------|--------|
| `set_mapping` handler updates in-process state only; does not write Wacom XML | `cmd/wacom-bridge/main.go` | 38 | Wacom XML write logic belongs to Plan 02-02 (`internal/wacom/xml.go`) |
| `reset_mapping` handler updates in-process state only; does not write Wacom XML | `cmd/wacom-bridge/main.go` | 51 | Same — deferred to Plan 02-02 |
| Baseline file check is path construction only; not validated against actual file | `cmd/wacom-bridge/main.go` | 29-30 | Plan 02-02 wires ERR_BASELINE_NOT_FOUND check into set_mapping/reset_mapping handlers |

These stubs are intentional per the plan design. Plan 02-01 goal (protocol contract + messaging layer) is fully achieved. Plan 02-02 will replace stubs with Wacom XML write logic.

---

## Threat Flags

None — no new security surface beyond what is documented in the plan's threat model.

The threat model mitigations implemented in this plan:
- **T-02-01-01 / T-02-01-03:** 1 MB message size guard + zero-length guard in ReadMessage
- **T-02-01-02:** docs/protocol.md explicitly states wildcards are NOT permitted in allowed_origins

---

## Self-Check

Verifying created files exist and commits are present.

### Files Check

All 8 created files confirmed present on disk. Commits 02de671 (Task 1) and 9f1791f (Task 2) confirmed in git log.

## Self-Check: PASSED
