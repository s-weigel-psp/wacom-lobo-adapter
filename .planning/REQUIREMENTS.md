# Requirements: Wacom Lobo Adapter

**Defined:** 2026-04-29
**Core Value:** The Wacom stylus is confined to the PDF area with a single activation, and the user is notified via a banner when the area changes so they can re-sync with one click.

## v1 Requirements

### Spike (Phase 1 — Feasibility)

- [x] **SPIKE-01
**: PowerShell script sets Wacom tablet mapping to an arbitrary screen region specified by X, Y, Width, Height parameters
- [ ] **SPIKE-02**: Mapping change completes in under 3 seconds (measured across consecutive changes)
- [ ] **SPIKE-03**: Wacom stylus respects the new mapping region after a change (stays within mapped area)
- [x] **SPIKE-04
**: Reset script restores full-screen mapping from baseline profile
- [x] **SPIKE-05
**: SPIKE-RESULTS.md documents working method, binary paths, XML structure, service names, measured latency, and recommendation for Phase 2

### Native Host (Phase 2)

- [x] **HOST-01
**: Native messaging host processes `set_mapping` command and applies Wacom screen region mapping
- [x] **HOST-02
**: Native messaging host processes `reset_mapping` command and restores baseline profile
- [x] **HOST-03
**: Native messaging host responds to `get_status` and `ping` commands
- [x] **HOST-04
**: Host logs activity to `%LOCALAPPDATA%\WacomBridge\logs\`
- [ ] **HOST-05**: WiX MSI installer registers Chrome and Edge native messaging manifests in the Windows registry
- [ ] **HOST-06**: Installer runs silently without user interaction

### Browser Extension (Phase 3)

- [x] **EXT-01**: Content script detects the target DOM element by its configured ID on page load
- [x] **EXT-02**: Extension calculates the element's screen coordinates accounting for Windows DPI scaling
- [x] **EXT-03**: Extension sends mapping coordinates to native host when user activates PDF mode
- [x] **EXT-04**: Extension shows a banner (Shadow DOM) when the PDF area position or size changes
- [x] **EXT-05**: User can re-sync Wacom mapping by clicking the banner's calibration button
- [x] **EXT-06**: Extension handles native host unavailability gracefully (shows error state in banner)

### Deployment (Phase 4)

- [ ] **DEPLOY-01**: Chrome extension auto-installs on managed machines via `ExtensionInstallForcelist` GPO policy
- [ ] **DEPLOY-02**: Edge extension auto-installs on managed machines via equivalent Edge GPO policy
- [ ] **DEPLOY-03**: `WindowManagementAllowedForUrls` GPO policy auto-grants the Window Management permission
- [ ] **DEPLOY-04**: URL allowlist restricts extension to the target application domain via `ExtensionSettings` GPO
- [ ] **DEPLOY-05**: Native host MSI deploys silently via Intune or GPO Software Installation

## v2 Requirements

### Operations

- **OPS-01**: Configurable logging verbosity level for support cases
- **OPS-02**: Graceful fallback behavior when native host is not installed (informative user message)
- **OPS-03**: Extension self-update mechanism when the target application changes its DOM structure

### Multi-monitor

- **MULTI-01**: Full support for multi-monitor setups (mapping to secondary display)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Live/real-time tablet tracking | Wacom driver has no documented live API; ~1–2s latency per mapping change makes live tracking unusable |
| macOS / Linux support | Windows domain environment is the only target |
| Custom Wacom driver development | Using existing `Wacom_TabletUserPrefs.exe` preference XML mechanism |
| Third-party PDF viewer modifications | Integration is purely DOM observation + Wacom driver calls, no modifications to the host app |
| Firefox support | Enterprise context uses Chrome/Edge only |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| SPIKE-01 | Phase 1 | Pending |
| SPIKE-02 | Phase 1 | Pending |
| SPIKE-03 | Phase 1 | Pending |
| SPIKE-04 | Phase 1 | Pending |
| SPIKE-05 | Phase 1 | Pending |
| HOST-01 | Phase 2 | Pending |
| HOST-02 | Phase 2 | Pending |
| HOST-03 | Phase 2 | Pending |
| HOST-04 | Phase 2 | Pending |
| HOST-05 | Phase 2 | Pending |
| HOST-06 | Phase 2 | Pending |
| EXT-01 | Phase 3 | Complete |
| EXT-02 | Phase 3 | Complete |
| EXT-03 | Phase 3 | Complete |
| EXT-04 | Phase 3 | Complete |
| EXT-05 | Phase 3 | Complete |
| EXT-06 | Phase 3 | Complete |
| DEPLOY-01 | Phase 4 | Pending |
| DEPLOY-02 | Phase 4 | Pending |
| DEPLOY-03 | Phase 4 | Pending |
| DEPLOY-04 | Phase 4 | Pending |
| DEPLOY-05 | Phase 4 | Pending |

**Coverage:**
- v1 requirements: 22 total
- Mapped to phases: 22
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-29*
*Last updated: 2026-04-29 after initial definition*
