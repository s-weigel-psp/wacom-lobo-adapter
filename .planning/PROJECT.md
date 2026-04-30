# Wacom Lobo Adapter

## What This Is

A two-part system (Chrome/Edge browser extension + Windows native messaging host) that automatically restricts Wacom One M tablet input to the PDF rendering area in a browser-based third-party application. Users in a Windows domain annotate PDFs without manually adjusting Wacom settings — the system reads the DOM element position and programs the Wacom driver accordingly.

## Core Value

The Wacom stylus is confined to the PDF area with a single activation, and the user is notified via a banner when the area changes so they can re-sync with one click.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Wacom tablet mapping can be programmatically set to an arbitrary screen region via `Wacom_TabletUserPrefs.exe` and preference XML
- [ ] Native messaging host accepts `set_mapping`/`reset_mapping` commands from browser extension and applies them to the Wacom driver
- [ ] Browser extension reads DOM element screen coordinates (with DPR/DPI correction) and sends them to the native host
- [ ] Extension shows a calibration banner when the PDF area moves or resizes
- [ ] Entire system deploys domain-wide via GPO/Intune without per-user manual configuration

### Out of Scope

- Live/real-time tablet tracking — explicit sync model chosen; Wacom driver has no documented live API, and ~1–2s per mapping change is unacceptable for live use
- macOS/Linux support — Windows domain environment only
- Custom Wacom driver development — using existing `Wacom_TabletUserPrefs.exe` + preference XML
- Modifications to the third-party PDF application — integration is purely DOM observation + driver calls

## Context

- Target environment: Windows enterprise domain, Chrome and Edge browsers
- Third-party browser application renders PDFs inside a DOM element with a stable, known ID
- Wacom One M tablet connected to domain-joined Windows workstations
- `Wacom_TabletUserPrefs.exe` CLI imports preference XML (typical path: `C:\Program Files\Tablet\Wacom\`)
- Preference XML files at `%ProgramData%\Tablet\Wacom\` and/or `%LOCALAPPDATA%\Wacom\`
- Windows DPI scaling (125%, 150% etc.) causes PowerShell screen dimension APIs to return scaled values; Wacom likely expects physical pixels — needs verification in Phase 1
- Admin rights requirement is a key unknown: if `Wacom_TabletUserPrefs.exe` requires elevation in user context, the native host may need to run as a Windows service

## Constraints

- **Platform**: Windows only — Wacom driver APIs are Windows-specific
- **Browser**: Chrome and Edge (Manifest V3) — enterprise-managed
- **Deployment**: GPO/Intune managed — extension force-installed, no user action required
- **Latency**: Mapping change must complete in < 3 seconds (acceptable for explicit sync on activation)
- **Tech Stack**: Go single-binary exe (native host), PowerShell (Phase 1 spike), MV3 browser extension, WiX MSI packaging, Shadow DOM for banner UI

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Explicit sync model instead of live tracking | Wacom driver has no documented live API; ~1–2s per mapping change makes live tracking unusable; observed usage pattern (open PDF once, work in it) makes explicit sync functionally equivalent at far lower complexity | — Pending |
| Go for native host | Single-binary deployment (no runtime dependency), good Windows API access via golang.org/x/sys/windows, developer preference, easy to maintain | — Decided (Phase 2 discuss) |
| Shadow DOM for extension banner UI | Prevents styling conflicts with the third-party application's CSS | — Pending |
| Phase 1 is a spike first | Technical risk is HIGH — if `Wacom_TabletUserPrefs.exe` cannot be scripted reliably, the whole project design needs to change | — Pending |
| Phase 3 can run in parallel with Phase 2 | Extension and host have well-defined interface (Native Messaging JSON protocol) — teams can work independently once Phase 1 validates the approach | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-29 after initialization*
