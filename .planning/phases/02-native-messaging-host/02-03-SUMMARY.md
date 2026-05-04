---
phase: 02-native-messaging-host
plan: "03"
subsystem: infra
tags: [wix4, msi, installer, native-messaging, chrome, edge, registry, windows]

# Dependency graph
requires:
  - phase: 02-01
    provides: Go native host binary (wacom-bridge.exe) that the installer packages
  - phase: 02-02
    provides: Validated Wacom XML integration confirming host functionality

provides:
  - installer/wacom-bridge.wxs — WiX 4 Package definition: binary + manifests + HKLM registry entries
  - installer/manifest-chrome.json — Chrome Native Messaging host manifest
  - installer/manifest-edge.json — Edge Native Messaging host manifest

affects:
  - 03-browser-extension (needs final Chrome/Edge extension IDs to replace PLACEHOLDER_*_EXTENSION_ID)
  - 04-deployment (GPO/Intune uses this MSI as the deployment artefact)

# Tech tracking
tech-stack:
  added: [wix4, msiexec]
  patterns:
    - WiX 4 Package element (not Product) under http://wixtoolset.org/schemas/v4/wxs namespace
    - ProgramFiles6432Folder for 64-bit C:\Program Files\ install target
    - Separate Chrome and Edge manifest files for independent extension ID updates
    - WOW6432Node edge registry key alongside 64-bit key for complete 32-bit/64-bit coverage

key-files:
  created:
    - installer/wacom-bridge.wxs
    - installer/manifest-chrome.json
    - installer/manifest-edge.json
  modified: []

key-decisions:
  - "Separate manifest-chrome.json and manifest-edge.json files chosen over a single manifest to allow independent extension ID updates when Chrome and Edge ship at different times"
  - "WOW6432Node Edge registry key added alongside the 64-bit path — covers 32-bit Edge installations per 02-RESEARCH.md Pitfall 2"
  - "XML comment --global flag replaced with -g flag to satisfy XML 1.0 prohibition on -- inside comments (deviation auto-fix)"
  - "PLACEHOLDER extension IDs used in allowed_origins — must be replaced with real published IDs before production MSI build (Phase 3 deliverable)"

patterns-established:
  - "WiX 4 GUID format: uppercase hex wrapped in curly braces {XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}"
  - "Edge Native Messaging uses chrome-extension:// scheme (not edge://) — documented in both manifest and plan"
  - "Installer source files committed; MSI binary (wacom-bridge.msi) is a build artefact and not committed"

requirements-completed: [HOST-05, HOST-06]

# Metrics
duration: 2min
completed: 2026-05-04
---

# Phase 02 Plan 03: WiX 4 MSI Installer Source Files Summary

**WiX 4 MSI installer with HKLM Chrome/Edge NativeMessagingHosts registry entries, ProgramFiles6432Folder binary install, and separate Chrome/Edge manifest JSON files with placeholder extension IDs**

## Performance

- **Duration:** 2 min
- **Started:** 2026-05-04T09:58:17Z
- **Completed:** 2026-05-04T09:59:52Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Created installer/manifest-chrome.json and installer/manifest-edge.json as valid JSON Native Messaging manifests with com.eurefirma.wacombridge name, stdio type, C:\Program Files\WacomBridge\wacom-bridge.exe path, and distinct placeholder extension IDs
- Created installer/wacom-bridge.wxs as a valid WiX 4 Package with real GUIDs, ProgramFiles6432Folder install target, MajorUpgrade element, Chrome HKLM registry entry, and both 64-bit and WOW6432Node Edge HKLM registry entries
- All 12 plan verification checks pass; no placeholder GUID text remains in the .wxs file

## Task Commits

Each task was committed atomically:

1. **Task 1: Create Native Messaging manifest JSON files** - `f48c949` (feat)
2. **Task 2: Create installer/wacom-bridge.wxs WiX 4 Package definition** - `9a970bf` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified

- `installer/manifest-chrome.json` — Chrome NM manifest: name, description, path, type, allowed_origins with PLACEHOLDER_CHROME_EXTENSION_ID
- `installer/manifest-edge.json` — Edge NM manifest: same fields, PLACEHOLDER_EDGE_EXTENSION_ID, correct chrome-extension:// scheme
- `installer/wacom-bridge.wxs` — WiX 4 Package: MainExe + ManifestChrome + ManifestEdge components, 4 real GUIDs, HKLM Chrome and Edge (64-bit + WOW6432Node) registry keys, MajorUpgrade

## Decisions Made

- Separate manifest files for Chrome and Edge allow independent extension ID updates after Phase 3 publishes to Chrome Web Store and Edge Add-ons
- WOW6432Node registry key written alongside the standard 64-bit Edge path per 02-RESEARCH.md Pitfall 2; Chrome searches both paths natively so only one Chrome key needed
- PLACEHOLDER IDs documented prominently in both the manifest _dev_note fields and installer comment block

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed XML comment containing double-dash (--global)**
- **Found during:** Task 2 (wacom-bridge.wxs creation)
- **Issue:** XML 1.0 prohibits `--` inside comment content; `dotnet tool install --global wix` in the build instructions comment caused `xml.etree.ElementTree.ParseError: not well-formed (invalid token): line 7, column 26`
- **Fix:** Replaced `--global` with the equivalent short flag `-g` in the comment text
- **Files modified:** installer/wacom-bridge.wxs
- **Verification:** `python3 -c "import xml.etree.ElementTree as ET; ET.parse('installer/wacom-bridge.wxs')"` exits 0
- **Committed in:** 9a970bf (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - XML well-formedness bug in comment)
**Impact on plan:** Necessary correctness fix; the acceptance criteria explicitly require `ET.parse` to exit 0. No scope creep.

## Issues Encountered

None beyond the auto-fixed XML comment issue above.

## User Setup Required

WiX 4 CLI must be installed on the Windows build machine before the MSI can be compiled:

1. Install .NET SDK 6+ from https://dot.net/download
2. `dotnet tool install -g wix`
3. From the repo root: `wix build installer/wacom-bridge.wxs -o wacom-bridge.msi`
4. Deploy: `msiexec /i wacom-bridge.msi /quiet`

Before production build: replace `PLACEHOLDER_CHROME_EXTENSION_ID` and `PLACEHOLDER_EDGE_EXTENSION_ID` in the two manifest JSON files with the real published extension IDs from Phase 3.

## Known Stubs

| Stub | File | Line | Reason |
|------|------|------|--------|
| PLACEHOLDER_CHROME_EXTENSION_ID | installer/manifest-chrome.json | 7 | Real Chrome Web Store extension ID not available until Phase 3 publishes the extension |
| PLACEHOLDER_EDGE_EXTENSION_ID | installer/manifest-edge.json | 7 | Real Edge Add-ons extension ID not available until Phase 3 publishes the extension |

These stubs do not prevent the plan's goal (installer source files ready for build). Phase 3 will produce the final IDs.

## Next Phase Readiness

- installer/ directory contains all three source files needed to produce the final MSI on a Windows build machine
- HOST-05 (Chrome and Edge HKLM manifest registration) and HOST-06 (silent install/uninstall) are satisfied
- Phase 3 (browser extension) must deliver final extension IDs to complete the production MSI
- Phase 4 (deployment) can use `msiexec /i wacom-bridge.msi /quiet` for GPO/Intune distribution

---
*Phase: 02-native-messaging-host*
*Completed: 2026-05-04*

## Self-Check: PASSED

| Check | Result |
|-------|--------|
| installer/manifest-chrome.json exists | FOUND |
| installer/manifest-edge.json exists | FOUND |
| installer/wacom-bridge.wxs exists | FOUND |
| 02-03-SUMMARY.md exists | FOUND |
| commit f48c949 exists | FOUND |
| commit 9a970bf exists | FOUND |
