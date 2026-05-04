# Roadmap: Wacom Lobo Adapter

## Overview

Four phases take the project from feasibility validation to production domain rollout. Phase 1 is a high-risk spike that must succeed before any other phase begins. Phases 2 and 3 can run in parallel once Phase 1 confirms the approach is viable. Phase 4 deploys the completed system to the domain.

## Phases

- [ ] **Phase 1: Wacom Mapping Spike** - Verify that Wacom tablet mapping can be programmatically set at runtime via PowerShell + `Wacom_TabletUserPrefs.exe`
- [ ] **Phase 2: Native Messaging Host** - Build the production Go Windows helper that receives mapping commands and drives the Wacom driver
- [ ] **Phase 3: Browser Extension** - Build the Chrome/Edge MV3 extension that tracks the DOM element and sends coordinates to the native host
- [ ] **Phase 4: Domain Deployment** - Package and deploy the full system to all domain machines via GPO/Intune

## Phase Details

### Phase 1: Wacom Mapping Spike
**Goal**: Prove that `Wacom_TabletUserPrefs.exe` can be scripted to restrict the Wacom stylus to an arbitrary screen region, measure latency, and produce a documented recommendation for Phase 2.
**Depends on**: Nothing (first phase — but requires physical Wacom One M + Windows test machine)
**Requirements**: SPIKE-01, SPIKE-02, SPIKE-03, SPIKE-04, SPIKE-05
**Risk**: HIGH — if this phase fails, the overall architecture must be reconsidered
**Success Criteria** (what must be TRUE):
  1. `Set-WacomMapping.ps1 -X 0 -Y 0 -Width 960 -Height 1080` restricts stylus to left half of screen
  2. Three consecutive mapping changes complete within 3 seconds each
  3. `Reset-WacomMapping.ps1` restores full-screen stylus movement
  4. `spike/SPIKE-RESULTS.md` documents: working method, binary path, XML tag names, measured latency, admin-rights requirement, and a clear recommendation (keep PowerShell or port to C#)
**Plans**: 3 plans

Plans:
- [x] 01-01-PLAN.md — Discover PrefUtil binary, export two XML baselines, diff to identify screen-mapping tags, document service names, commit baseline-reference.xml
- [x] 01-02-PLAN.md — Write Set-WacomMapping.ps1 (clone baseline, Select-Xml XPath, PrefUtil import) and Reset-WacomMapping.ps1 using discovered tag names from Plan 01-01
- [x] 01-03-PLAN.md — Execute 5 test cases (left half, right half, centre, 3x latency, reset), fill SPIKE-RESULTS.md with all required fields and Phase 2 recommendation

### Phase 2: Native Messaging Host
**Goal**: Production-ready Windows executable that receives Native Messaging commands from a browser extension and drives the Wacom driver; delivered as a silent MSI installer.
**Depends on**: Phase 1 (Wacom XML manipulation approach validated)
**Requirements**: HOST-01, HOST-02, HOST-03, HOST-04, HOST-05, HOST-06
**Note**: Can run in parallel with Phase 3 — interface contract is the JSON protocol defined in `docs/protocol.md`
**Success Criteria** (what must be TRUE):
  1. `chrome.runtime.sendNativeMessage('com.eurefirma.wacombridge', {command:'set_mapping', x:0, y:0, width:960, height:1080})` applies Wacom mapping visibly
  2. `reset_mapping` command restores full-screen stylus movement
  3. `get_status` returns JSON with current mapping state; `ping` returns `{ok: true}`
  4. MSI installs silently (`msiexec /i WacomBridge.msi /quiet`) and appears in Chrome and Edge native messaging registry
  5. Log entries appear in `%LOCALAPPDATA%\WacomBridge\logs\` after each command
**Plans**: 3 plans

Plans:
- [ ] 02-01-PLAN.md — Author docs/protocol.md JSON contract (first deliverable), scaffold Go module, implement Native Messaging 4-byte LE framing, binary mode init, command dispatcher, slog logging to %LOCALAPPDATA%\WacomBridge\logs\
- [ ] 02-02-PLAN.md — Implement Wacom XML integration: port Set-WacomMapping.ps1 + Reset-WacomMapping.ps1 to Go (clone-and-modify, all ArrayElement entries), direct XML write to discovered preference path, conditional WtabletServicePro restart via SCM; includes Process Monitor discovery checkpoint
- [ ] 02-03-PLAN.md — WiX 4 MSI installer: wacom-bridge.exe + manifest JSON files, HKLM Chrome and Edge registry entries (64-bit + WOW6432Node), MajorUpgrade for future upgrades

### Phase 3: Browser Extension
**Goal**: Chrome/Edge Manifest V3 extension that detects the target DOM element, computes its screen coordinates (DPR-corrected), drives the native host, and shows a banner when the area changes.
**Depends on**: Phase 1 (approach validated) — can run in parallel with Phase 2
**Requirements**: EXT-01, EXT-02, EXT-03, EXT-04, EXT-05, EXT-06
**Success Criteria** (what must be TRUE):
  1. Opening the target application URL causes the extension to locate the DOM element and show a "Wacom synced" status without user action
  2. Moving or resizing the browser window changes the banner status to "PDF area changed — re-calibrate"
  3. Clicking the re-calibrate button sends updated coordinates to the native host and restricts the stylus to the new area
  4. On a 150% DPI-scaled Windows display, the stylus maps to the correct visual area (not a scaled offset)
  5. If the native host is not installed, the banner shows a clear "Native host not found" error message
**Plans**: 3 plans

Plans:
- [ ] 03-01-PLAN.md — Scaffold MV3 extension: manifest.json (nativeMessaging + storage permissions, file:///C:/WacomTest/*.html match), background.js service worker (NATIVE_COMMAND relay, sendNativeMessage to com.brantpoint.wacombridge, return true), config.js (DEFAULT_TARGETS, HOST_NAME, POLL_INTERVAL_MS, DEBOUNCE_MS, AUTO_DISMISS_MS), content.js stub with ES module imports and sendNativeCommand relay
- [ ] 03-02-PLAN.md — Implement content.js coordinate calculation and staleness detection: getPhysicalCoords() with DPR formula (Math.round((rect + screenX/Y) * devicePixelRatio)), three-trigger staleness detection (ResizeObserver + window resize + 1s screenX/Y poll), Page Visibility pause/resume, syncMapping() sending set_mapping command with ERR_ error handling
- [ ] 03-03-PLAN.md — Shadow DOM banner UI and options page: banner.js with createBanner() factory (closed mode, four states per UI-SPEC.md, auto-dismiss synced after 3000ms, 300ms debounce), options.html/js/css with chrome.storage.sync tuple CRUD, validation for empty fields and invalid URL patterns

### Phase 4: Domain Deployment
**Goal**: The full system (extension + native host) is deployed to all domain machines without per-user action, validated with a pilot group before broad rollout.
**Depends on**: Phase 2 and Phase 3
**Requirements**: DEPLOY-01, DEPLOY-02, DEPLOY-03, DEPLOY-04, DEPLOY-05
**Success Criteria** (what must be TRUE):
  1. Extension appears in Chrome and Edge on a fresh domain machine without any user installation action
  2. Opening the target application URL on a managed machine activates Wacom mapping without browser permission prompts
  3. Native host MSI deploys silently to the pilot group via Intune (or GPO Software Installation) and is confirmed installed
  4. End-to-end flow works on 2 pilot machines: open PDF -> stylus confined to PDF area -> move window -> banner appears -> re-calibrate -> stylus confined to new area
**Plans**: 3 plans

Plans:
- [ ] 04-01: Create GPO ADMX/JSON policy templates — `ExtensionInstallForcelist` for Chrome and Edge, `ExtensionSettings` URL allowlist, `WindowManagementAllowedForUrls` for Window Management permission auto-grant
- [ ] 04-02: Package native host MSI for Intune/GPO deployment — create Intune Win32 app package or GPO software installation share, test silent install on clean VM
- [ ] 04-03: Pilot test on 2 domain machines — validate end-to-end flow, document any issues, create rollout plan for remaining machines

## Progress

**Execution Order:**
Phase 1 -> Phase 2 (parallel with Phase 3) -> Phase 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Wacom Mapping Spike | 0/3 | Not started | - |
| 2. Native Messaging Host | 0/3 | Not started | - |
| 3. Browser Extension | 0/3 | Not started | - |
| 4. Domain Deployment | 0/3 | Not started | - |
