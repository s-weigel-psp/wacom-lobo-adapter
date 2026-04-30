# Phase 2: Native Messaging Host — Context

**Gathered:** 2026-04-30
**Status:** Ready for planning

<domain>
## Phase Boundary

Build a production-ready Go binary that receives Native Messaging JSON commands from a Chrome/Edge browser extension and drives the Wacom driver by writing preference XML directly. Delivered as a silent WiX MSI installer. Does NOT include browser extension code (Phase 3) or GPO/Intune deployment scripts (Phase 4).

Three plans in scope: (1) scaffold Go project + author `docs/protocol.md` JSON contract, (2) implement XML write + service restart Wacom integration, (3) WiX 4 MSI installer with HKLM registry entries.

</domain>

<decisions>
## Implementation Decisions

### Language / Stack

- **D-01:** Native host is written in **Go** (not C# .NET 8). Rationale: single-binary deployment with no runtime dependency, good Windows API access via `golang.org/x/sys/windows`, developer familiarity. CLAUDE.md and PROJECT.md updated to reflect this.

### Wacom Driver Integration

- **D-02:** The host drives Wacom by **writing the preference XML file directly** — PrefUtil (`Wacom_TabletUserPrefs.exe`) is **not used** at runtime. PrefUtil's mandatory GUI dialog on every `/import` makes it production-unusable (confirmed in spike).
- **D-03:** After writing the XML, the host restarts `WtabletServicePro` if required. Whether the restart is necessary (vs. the service hot-reloading the file) is **unknown** and MUST be tested in Plan 02-02. Implement conditional restart based on test findings.
- **D-04:** No PrefUtil fallback path. If direct XML write fails, return a structured error response (see Error Handling below). PrefUtil must not appear in the production code path.
- **D-05:** XML editing follows Phase 1 decisions: clone-and-modify the existing preference file (do not construct minimal XML from scratch). Preserve all other tablet settings. XPath: `//InputScreenAreaArray/ArrayElement/ScreenArea`. Update all `ArrayElement` entries (test machine had 3). File extension must be `.Export.wacomxs`.

### Protocol Contract

- **D-06:** `docs/protocol.md` is the **first deliverable** of Plan 02-01 — authored before any Go implementation code. Phase 3 (browser extension) depends on this contract for parallel development.
- **D-07:** Commands: `set_mapping`, `reset_mapping`, `get_status`, `ping`.
- **D-08:** `get_status` response (minimal): `{"mapped": bool, "x": int, "y": int, "width": int, "height": int}`. A `monitor` field is **reserved** (optional, null by default) for future multi-monitor support (v2 MULTI-01). Do not implement multi-monitor logic in Phase 2.
- **D-09:** `ping` response: `{"ok": true}`.
- **D-10:** Native Messaging framing: 4-byte little-endian length prefix + UTF-8 JSON body (standard Chrome native messaging protocol).

### Error Handling

- **D-11:** All failure responses use structured JSON: `{"error": "human-readable message", "code": "ERR_SNAKE_CASE"}`. Never return a bare exit or silent failure.
- **D-12:** On unrecoverable errors (preference file not found, service unavailable, XML parse failure): **return error JSON and stay running**. Do not exit. The extension handles error display in the banner; the host does not crash.
- **D-13:** Success responses for `set_mapping` and `reset_mapping`: `{"ok": true}`.

### Installer

- **D-14:** Native messaging manifests registered under **HKLM** (`HKEY_LOCAL_MACHINE`) for both Chrome and Edge. Machine-wide registration is required for GPO/Intune deployment (Phase 4).
  - Chrome: `HKLM\SOFTWARE\Google\Chrome\NativeMessagingHosts\com.eurefirma.wacombridge`
  - Edge: `HKLM\SOFTWARE\Microsoft\Edge\NativeMessagingHosts\com.eurefirma.wacombridge`
- **D-15:** Binary installed at `C:\Program Files\WacomBridge\wacom-bridge.exe`. MSI requires admin elevation at install time. The host binary itself runs without elevation (confirmed in spike: PrefUtil import works without elevation).
- **D-16:** MSI installs and uninstalls silently (`/quiet` flag).

### Claude's Discretion

- Go module name and package structure
- Exact error codes (ERR_FILE_NOT_FOUND, ERR_SERVICE_RESTART_FAILED, etc.)
- Logging format within `%LOCALAPPDATA%\WacomBridge\logs\` (HOST-04)
- Windows Service Control Manager vs. `net stop/start` for service restart
- WiX component layout and upgrade strategy

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 1 Findings (XML schema, binary path, service names)
- `spike/SPIKE-RESULTS.md` — Working method, PrefUtil binary path, XML tag names and XPath, service name (`WtabletServicePro`), latency measurements, admin-rights finding, Phase 2 recommendation. **Most important reference for Plan 02-02.**
- `spike/baseline-reference.Export.wacomxs` — Reference preference XML exported from test machine. Use to develop and test Go XML parser without a live Wacom device.

### Project Requirements
- `.planning/REQUIREMENTS.md` — HOST-01 through HOST-06 define exactly what Phase 2 must deliver.
- `.planning/PROJECT.md` — Key Decisions table (Go language decision, explicit sync model), Constraints section (latency < 3s, Windows-only, HKLM installer).

### Phase 1 Context (XML editing decisions)
- `.planning/phases/01-wacom-mapping-spike/01-CONTEXT.md` — D-01 (clone-and-modify), D-02 (XPath expression), D-03 (baseline reference vs. per-machine local copy).

### Protocol Contract (to be created)
- `docs/protocol.md` — JSON command/response schema. **Does not exist yet — Plan 02-01 creates it.** Once written, Phase 3 references it as the stable interface contract.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `spike/Set-WacomMapping.ps1` — PowerShell reference implementation. The Go Plan 02-02 ports this logic: clone XML, Select-Xml XPath, modify Origin/Extent/AreaType, write temp file, invoke import. Read this before writing Go XML code.
- `spike/Reset-WacomMapping.ps1` — Reference for reset flow: re-import the baseline local file.
- `spike/baseline-reference.Export.wacomxs` — XML schema reference; use as test fixture for Go XML parsing.

### Established Patterns
- Clone-and-modify XML (not construct minimal XML) — preserves all non-mapping tablet settings.
- All `ArrayElement` entries under `InputScreenAreaArray` are updated, not just the first.
- File must be written with `.Export.wacomxs` extension — `.xml` silently fails.
- Coordinates are **physical pixels** — extension must send DPR-corrected values (EXT-02).

### Integration Points
- `WtabletServicePro` Windows service — may need restart after direct XML write (test in Plan 02-02).
- Chrome/Edge registry: `NativeMessagingHosts\com.eurefirma.wacombridge` → path to manifest JSON.
- Native Messaging manifest JSON → points to `wacom-bridge.exe` absolute path.

</code_context>

<specifics>
## Specific Ideas

- The direct XML write approach (bypassing PrefUtil) was explicitly chosen because PrefUtil's GUI dialog is production-unusable. If Plan 02-02 finds that direct XML write does not reliably apply mappings, escalate to the user before adding PrefUtil back.
- `docs/protocol.md` is a contract document for Phase 3 parallelism, not just internal documentation. It should be written to be read by the Phase 3 developer independently.
- `monitor` field in `get_status` is reserved but not implemented. Document it as "reserved, always null" in `docs/protocol.md` so Phase 3 doesn't need updating when MULTI-01 lands.

</specifics>

<deferred>
## Deferred Ideas

- **Multi-monitor support** (MULTI-01) — `monitor` field reserved in `get_status` schema, not implemented in Phase 2. Belongs in v2.
- **Log rotation / verbosity levels** (OPS-01) — HOST-04 requires basic logging; rotation and configurable verbosity are v2 operations requirements.
- **Graceful fallback when host not installed** (OPS-02) — handled by Phase 3 extension (EXT-06), not the host itself.

</deferred>

---

*Phase: 02-native-messaging-host*
*Context gathered: 2026-04-30*
