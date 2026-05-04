---
phase: 02-native-messaging-host
plan: "02"
subsystem: native-host
tags: [go, wacom, xml, native-messaging, windows, prefutil, service-control]
dependency_graph:
  requires:
    - 02-01 — Go module scaffold, messaging I/O layer, state, logging
  provides:
    - internal/wacom/xml.go — SetMapping/ResetMapping: clone-and-modify XML, call PrefUtil subprocess
    - internal/wacom/service.go — RestartWacomService via Windows SCM (future-proofed, not called)
    - cmd/wacom-bridge/main.go — set_mapping and reset_mapping wired to wacom package
  affects:
    - HOST-01: set_mapping now applies Wacom screen region via PrefUtil /import
    - HOST-02: reset_mapping now restores baseline via PrefUtil /import
tech_stack:
  added:
    - os/exec (stdlib) — PrefUtil subprocess invocation
    - encoding/xml token stream (stdlib) — clone-and-modify XML without full struct modelling
    - golang.org/x/sys/windows/svc/mgr — Windows SCM for RestartWacomService (build tag: windows)
  patterns:
    - XML token-stream clone-and-modify: walk tokens, rewrite only ScreenArea children within InputScreenAreaArray/ArrayElement path
    - PrefUtil subprocess: exec.Command(wacomPrefPath, "/import", tempPath) — COM-based apply without restart
    - Coordinate validation guard: 0 <= x/y <= 16384, w/h > 0 before XML write (T-02-02-01)
    - 10 MB baseline file size guard (T-02-02-05)
    - Windows SCM stop/start with 200ms polling and 10s timeout
key_files:
  created:
    - internal/wacom/xml.go
    - internal/wacom/service.go
  modified:
    - cmd/wacom-bridge/main.go
decisions:
  - "wacomPrefPath = C:\\Program Files\\Tablet\\Wacom\\PrefUtil.exe — Task 1 Process Monitor revealed PrefUtil uses COM (CLSID {ff48dba4-60ef-4201-aa87-54103eef594e}) to notify WTabletServicePro; no direct XML file write path to a WTabletServicePro system location was found"
  - "needsServiceRestart = false — PrefUtil COM mechanism applies preferences without requiring WTabletServicePro restart (Task 1 finding)"
  - "XML token-stream approach chosen over full struct tree — preserves all non-mapping tablet settings byte-for-byte without needing to model 500+ lines of deeply nested XML schema"
  - "RestartWacomService implemented fully despite needsServiceRestart=false — future-proofing; will be called if a future plan discovers direct XML write requires restart"
  - "Baseline copied to temp path before PrefUtil import for both SetMapping and ResetMapping — avoids PrefUtil locking or mutating the canonical baseline"
metrics:
  duration_minutes: 15
  completed_date: "2026-05-04"
  tasks_completed: 1
  files_created: 2
  files_modified: 1
---

# Phase 2 Plan 02: Wacom XML Integration and Command Wiring Summary

**One-liner:** Wacom XML clone-and-modify via token stream + PrefUtil subprocess (COM-based, no restart), wired to set_mapping/reset_mapping in the Native Messaging host.

---

## What Was Built

### Task 1: Process Monitor investigation (human checkpoint — completed before this run)

Findings provided by user:
- **WRITE_PATH:** PrefUtil.exe subprocess mechanism — no direct Wacom system file write path exists. PrefUtil uses COM (CLSID {ff48dba4-60ef-4201-aa87-54103eef594e} InprocServer32) to notify WTabletServicePro.
- **NEEDS_RESTART:** false — WTabletServicePro is notified via COM without requiring a service restart.
- **PROCMON_MECHANISM:** After PrefUtil /import, Wacom_Tablet.exe reads from the import source path.

### Task 2: Implementation

**internal/wacom/xml.go:**
- `SetMapping(logger, x, y, w, h)` — validates coordinates (T-02-02-01), reads baseline, applies token-stream XML transformation updating all `InputScreenAreaArray/ArrayElement/ScreenArea` nodes (AreaType=1, Origin.X/Y, Extent.X/Y), writes to `%TEMP%\wacom-mapping-temp.Export.wacomxs`, calls PrefUtil /import
- `ResetMapping(logger)` — reads baseline, copies to `%TEMP%\wacom-reset-temp.Export.wacomxs`, calls PrefUtil /import
- `readBaseline` — stat + 10 MB size guard (T-02-02-05) before ReadFile
- `runPrefUtil` — `exec.Command(wacomPrefPath, "/import", xmlPath).Run()`
- `applyScreenAreaCoords` — token-stream walk: emits all tokens unchanged except ScreenArea subtrees inside the correct ancestor path
- `rewriteScreenArea` — buffers and re-encodes all ScreenArea child tokens, substituting text content for five target paths
- Constants: `wacomPrefPath`, `needsServiceRestart=false`, `maxBaselineSize=10MB`, `maxCoord=16384`
- Error codes: `ERR_BASELINE_NOT_FOUND`, `ERR_XML_PARSE`, `ERR_XML_WRITE`, `ERR_NO_SCREEN_AREA_NODES`, `ERR_INVALID_PARAMS`

**internal/wacom/service.go (`//go:build windows`):**
- `RestartWacomService(logger)` — connects to SCM, opens `WtabletServicePro`, sends Stop control, polls status every 200ms up to 10s, then calls Start
- Error codes: `ERR_SERVICE_CONNECT`, `ERR_SERVICE_NOT_FOUND`, `ERR_SERVICE_ACCESS_DENIED`, `ERR_SERVICE_RESTART_TIMEOUT`, `ERR_SERVICE_RESTART_FAILED`
- `isAccessDenied` helper for SCM error classification

**cmd/wacom-bridge/main.go:**
- Replaced `set_mapping` stub with: param extraction → `wacom.SetMapping` → `st.Set` on success
- Replaced `reset_mapping` stub with: `wacom.ResetMapping` → `st.Reset` on success
- Added baseline existence check at startup (`os.Stat` → `logger.Warn` if missing, no exit per D-12)
- Added `internal/wacom` import

Cross-compile verified: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/wacom-bridge.exe ./cmd/wacom-bridge/` — BUILD OK.

---

## Deviations from Plan

### Plan-level deviation: PrefUtil subprocess replaces direct XML file write

**Context:** The plan (D-02, D-04 in CONTEXT.md) assumed direct XML file write would be discovered as the mechanism. Task 1 Process Monitor investigation revealed no direct write path exists — PrefUtil uses COM internally.

**Objective override:** The task1_findings provided with this execution explicitly instruct use of PrefUtil subprocess with the exact path `C:\Program Files\Tablet\Wacom\PrefUtil.exe`.

**Implementation adjustment:** `xml.go` writes the modified XML to `%TEMP%` then calls `exec.Command(wacomPrefPath, "/import", tempPath)` instead of `os.WriteFile` to a WTabletServicePro system path.

**Impact:** CONTEXT.md D-04 ("No PrefUtil fallback") is superseded by the Task 1 hardware findings. This is not a fallback — it is the only mechanism that works, as confirmed by the Process Monitor trace.

### Auto-fixed Issues

**1. [Rule 2 - Security] Added coordinate range validation before XML write (T-02-02-01)**
- **Found during:** Task 2 implementation (threat model T-02-02-01)
- **Issue:** Threat model requires validating x/y/width/height are non-negative integers within plausible screen bounds before writing to XML.
- **Fix:** Added guard at top of SetMapping: `x < 0 || y < 0 || w <= 0 || h <= 0` and `> maxCoord (16384)` → returns ERR_INVALID_PARAMS.
- **Files modified:** `internal/wacom/xml.go`
- **Commit:** 73108f7

**2. [Rule 2 - Security] Added 10 MB baseline file size guard (T-02-02-05)**
- **Found during:** Task 2 implementation (threat model T-02-02-05)
- **Issue:** Threat model requires rejecting baseline files > 10 MB to prevent DoS via memory allocation.
- **Fix:** `readBaseline` stat-checks size before ReadFile; returns error if > 10 MB.
- **Files modified:** `internal/wacom/xml.go`
- **Commit:** 73108f7

**3. [Rule 2 - Security] Baseline copied to temp path before PrefUtil import for ResetMapping**
- **Found during:** Task 2 implementation
- **Issue:** Passing the canonical baseline path directly to PrefUtil risks PrefUtil locking or mutating it.
- **Fix:** ResetMapping copies baseline to `%TEMP%\wacom-reset-temp.Export.wacomxs` before calling PrefUtil — same pattern as SetMapping.
- **Files modified:** `internal/wacom/xml.go`
- **Commit:** 73108f7

---

## Known Stubs

None — all stubs from Plan 02-01 have been replaced with real implementations.

| Previously Stubbed | File | Resolution |
|--------------------|------|------------|
| `set_mapping` handler (state-only) | `cmd/wacom-bridge/main.go` | Wired to `wacom.SetMapping` |
| `reset_mapping` handler (state-only) | `cmd/wacom-bridge/main.go` | Wired to `wacom.ResetMapping` |
| Baseline file check (path only) | `cmd/wacom-bridge/main.go` | `os.Stat` check with `Warn` at startup |

---

## Threat Flags

None — all new surface was covered by the plan's existing threat model. No additional security-relevant surface was introduced.

Threat model mitigations implemented:
- **T-02-02-01:** Coordinate range validation in SetMapping (0 ≤ x/y ≤ 16384, w/h > 0)
- **T-02-02-02:** `wacomPrefPath` is a compile-time constant; baseline path uses `os.Getenv("LOCALAPPDATA")` via `filepath.Join` — not user-controlled
- **T-02-02-03:** ERR_SERVICE_ACCESS_DENIED returned and host stays running (D-12); no privilege escalation
- **T-02-02-05:** 10 MB baseline file size guard in `readBaseline`

---

## Self-Check

Verifying created/modified files and commit hash.

- FOUND: internal/wacom/xml.go
- FOUND: internal/wacom/service.go
- FOUND: cmd/wacom-bridge/main.go
- FOUND: commit 73108f7

## Self-Check: PASSED
