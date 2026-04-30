# Phase 2: Native Messaging Host — Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in 02-CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-30
**Phase:** 02-native-messaging-host
**Areas discussed:** Language/Stack, Wacom driver integration, Protocol contract, Error handling, Installer registry scope

---

## Language / Stack

| Option | Description | Selected |
|--------|-------------|----------|
| C# .NET 8 | Original project spec — production-grade Windows integration, single-file deployment | |
| Go | Single binary, no runtime, good Windows API access, developer familiarity | ✓ |
| PowerShell | Same as spike, simple — packaging and native messaging integration awkward | |
| C++ | Smallest binary, no runtime — complex to write and maintain | |

**User's choice:** Go  
**Notes:** User questioned the C# origin ("why C#"), learned it was from project initialization, and decided to switch now before any C# code was written. CLAUDE.md and PROJECT.md updated to reflect the change.

---

## Wacom Driver Integration

| Option | Description | Selected |
|--------|-------------|----------|
| Direct XML write only | Write XML directly to preference file; restart WtabletServicePro if needed. PrefUtil not used at runtime. | ✓ |
| Direct XML write + PrefUtil fallback | Try direct XML write first; fall back to PrefUtil only on failure. | |
| Keep PrefUtil, suppress via other means | Suppress dialog via window automation. Risky/fragile. | |

**User's choice:** Direct XML write only  
**Notes:** PrefUtil confirmed production-unusable from spike (GUI dialog on every import, no suppression flag). Service restart requirement (when writing XML directly) is unknown — must be tested in Plan 02-02.

---

## Protocol Contract

### authorship

| Option | Description | Selected |
|--------|-------------|----------|
| Author docs/protocol.md in Plan 02-01 | First deliverable before Go code. Phase 3 can start immediately. | ✓ |
| Document inline | Schema in code comments / README. No stable artifact for Phase 3. | |
| You decide | Leave to planner. | |

**User's choice:** Author docs/protocol.md in Plan 02-01

### get_status response

| Option | Description | Selected |
|--------|-------------|----------|
| Minimal: mapping state only | {mapped, x, y, width, height} | ✓ (with note) |
| Extended: mapping + host info | Add version, last_command, timestamp | |
| You decide | Leave to planner | |

**User's choice:** Minimal, but noted that a `monitor` field may be needed for multi-monitor setups. Decided to reserve the field (optional, null) in v1 and implement in v2 (MULTI-01).

---

## Error Handling

### Response format

| Option | Description | Selected |
|--------|-------------|----------|
| Structured JSON error response | {error: "message", code: "ERR_CODE"} | ✓ |
| Boolean ok field | {ok: false} — can't distinguish error types | |
| You decide | Leave to planner | |

**User's choice:** Structured JSON error response

### On unrecoverable errors

| Option | Description | Selected |
|--------|-------------|----------|
| Return error JSON, stay running | Host continues; extension handles error display | ✓ |
| Exit with non-zero code | Host exits; browser detects disconnect | |
| You decide | Leave to planner | |

**User's choice:** Return error JSON, stay running

---

## Installer Registry Scope

### Registry hive

| Option | Description | Selected |
|--------|-------------|----------|
| HKLM only | Machine-wide, required for GPO/Intune (Phase 4) | ✓ |
| HKCU only | Per-user, no elevation at install — incompatible with GPO deployment | |
| Both HKLM + HKCU | Redundant in enterprise environment | |

**User's choice:** HKLM only

### Binary install path

| Option | Description | Selected |
|--------|-------------|----------|
| Program Files | C:\Program Files\WacomBridge\ — standard, trusted location for Chrome | ✓ |
| ProgramData | Non-standard for executables | |
| You decide | Leave to planner | |

**User's choice:** Program Files (`C:\Program Files\WacomBridge\wacom-bridge.exe`)

---

## Claude's Discretion

- Go module name and package structure
- Exact error codes
- Logging format within `%LOCALAPPDATA%\WacomBridge\logs\`
- Windows Service Control Manager vs. `net stop/start` for service restart
- WiX component layout and upgrade strategy

## Deferred Ideas

- Multi-monitor support (MULTI-01) — v2
- Log rotation / verbosity levels (OPS-01) — v2
- Graceful fallback when host not installed (OPS-02) — Phase 3 / EXT-06
