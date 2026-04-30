---
phase: 01-wacom-mapping-spike
plan: "01"
subsystem: infra
tags: [powershell, wacom, prefutil, xml, windows, spike]

# Dependency graph
requires: []
provides:
  - PrefUtil.exe actual path and CLI behavior (GUI-only, no silent mode)
  - Wacom preference XML schema and XPath for screen mapping region
  - Two committed baseline exports (full-screen and partial-screen)
  - Coordinate semantics for ScreenOutputArea (physical pixels, Origin + Extent)
  - Wacom service name for Phase 2 service-restart approach
affects:
  - 01-02 (Set-WacomMapping.ps1 depends on XPath and coordinate semantics discovered here)
  - 01-03 (SPIKE-RESULTS.md collects binary path, service names, and XML tags from test-log.md)
  - 02-native-host (PrefUtil headless limitation requires alternative silent invocation strategy)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Wacom preference file format: .Export.wacomxs extension (valid XML internally, no namespace on root)"
    - "XPath pattern: //InputScreenAreaArray/ArrayElement/ScreenArea for screen region nodes"
    - "Coordinate model: ScreenOutputArea/Origin (top-left) + ScreenOutputArea/Extent (width x height) in physical pixels"
    - "D-03 enforced: reference baseline committed, per-machine files gitignored"

key-files:
  created:
    - spike/test-log.md
    - spike/baseline-reference.Export.wacomxs
    - spike/baseline-modified.Export.wacomxs
    - spike/.gitkeep
    - .gitignore
  modified: []

key-decisions:
  - "PrefUtil.exe is the correct binary (Wacom_TabletUserPrefs.exe does not exist on driver 6.4.12-3)"
  - "File extension must be .Export.wacomxs — .xml extension silently fails to write at the specified path"
  - "AreaType must be changed from 0 to 1 when applying a custom screen region"
  - "ScreenOutputArea coordinates are physical pixels; InputArea.OverlapArea coordinates are tablet-native units and may not need manual manipulation"
  - "Phase 2 native host CANNOT rely on PrefUtil /import for silent operation — alternative strategy required"

patterns-established:
  - "Spike baseline naming: baseline-reference.Export.wacomxs (full-screen), baseline-modified.Export.wacomxs (custom region)"
  - "XML diff discipline: capture before/after exports, tabulate changed attributes with line numbers"

requirements-completed:
  - SPIKE-05

# Metrics
duration: "< 5 min (Task 2 auto) + human-action (Task 1 on Windows machine, 2026-04-30)"
completed: "2026-04-30"
tasks_completed: 2
tasks_total: 2
---

# Phase 01 Plan 01: Wacom Preference XML Baseline Discovery Summary

**PrefUtil.exe located at `C:\Program Files\Tablet\Wacom\PrefUtil.exe`; XML screen mapping confirmed at `//InputScreenAreaArray/ArrayElement/ScreenArea`; PrefUtil has no silent/headless mode — all import and export operations require user interaction in a GUI dialog.**

## Performance

- **Duration:** Task 2 < 5 min (auto, Linux dev machine); Task 1 human-executed on Windows test machine (2026-04-30)
- **Completed:** 2026-04-30
- **Tasks:** 2/2
- **Files modified:** 5

## Accomplishments

- Located PrefUtil.exe at `C:\Program Files\Tablet\Wacom\PrefUtil.exe` and documented its CLI behavior (no headless mode)
- Exported two `.Export.wacomxs` baselines (full-screen and bottom-left-corner partial) and identified the minimal XML diff for screen mapping
- Confirmed exact XPath, coordinate semantics, and file format for use in Plans 01-02 and 01-03
- Documented Wacom service name (`WtabletServicePro`) and confirmed non-elevated invocation works (exit code 0)
- Identified a critical architectural deviation: PrefUtil cannot be invoked silently — Phase 2 must find an alternative

## Task Commits

1. **Task 2: spike/ scaffold** - `cd1af89` (chore) — spike/.gitkeep, spike/test-log.md template, .gitignore entries
2. **Task 1: Windows test execution** - committed by user — spike/test-log.md (filled), spike/baseline-reference.Export.wacomxs, spike/baseline-modified.Export.wacomxs

## Files Created/Modified

- `spike/test-log.md` — Discovery log: PrefUtil path, CLI help output, XML diff table, service names, admin rights result, coordinate notes
- `spike/baseline-reference.Export.wacomxs` — Full-screen mapping export (schema reference for Phase 2 C# host)
- `spike/baseline-modified.Export.wacomxs` — Bottom-left-corner partial mapping export (diff source)
- `spike/.gitkeep` — Ensures spike/ directory is tracked in git before baselines are committed
- `.gitignore` — Entries to exclude per-machine local files from the spike directory

## Spike Findings Reference

### 1. PrefUtil Binary

| Item | Value |
|------|-------|
| Actual path | `C:\Program Files\Tablet\Wacom\PrefUtil.exe` |
| Import flag | `/import` |
| Export flag | `/export` |
| Silent flag | `/silent` — suppresses the GUI window for the help/info screen; has NO effect on `/import`; export still requires dialog OK |
| Headless mode | **Not available.** Every import and export operation opens a GUI dialog requiring user interaction |
| File extension | Must be `.Export.wacomxs` — specifying `.xml` causes the file to not be written at the requested path |
| Admin rights | Not required — non-elevated PowerShell produces exit code 0, but dialog still appears |

### 2. XML Element Structure for Screen Mapping

**File format:** `.Export.wacomxs` extension; internally valid UTF-8 XML with no namespace on the root element.

**Root declaration:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<root type="map">
  ...
</root>
```

**XPath to the screen mapping nodes (for use in `Select-Xml` in Set-WacomMapping.ps1):**
```
//InputScreenAreaArray/ArrayElement/ScreenArea
```

Full path from root (informational):
```
root/TabletArray/ArrayElement/ContextManager/MappingGroupArray/ArrayElement/MappingSetArray/ArrayElement/InputScreenAreaArray/ArrayElement/ScreenArea
```

Note: The test machine had 3 `ArrayElement` entries inside `InputScreenAreaArray`. All entries should be updated when applying a region.

**Minimal XML structure of a ScreenArea node (full-screen baseline):**
```xml
<ScreenArea type="map">
  <AreaType type="integer">0</AreaType>       <!-- 0=full screen, 1=custom region -->
  ...
  <ScreenOutputArea type="map">
    <Extent type="map">
      <X type="integer">1920</X>              <!-- width in physical pixels -->
      <Y type="integer">1080</Y>              <!-- height in physical pixels -->
      <Z type="integer">0</Z>
    </Extent>
    <Origin type="map">
      <X type="integer">0</X>                 <!-- left edge in physical pixels -->
      <Y type="integer">0</Y>                 <!-- top edge in physical pixels -->
      <Z type="integer">0</Z>
    </Origin>
  </ScreenOutputArea>
  ...
  <WhichMonitor type="string">Desktop_0_</WhichMonitor>
</ScreenArea>
```

### 3. XML Attribute Diff (Full Screen vs. Bottom-Left Corner)

| Attribute (dot-notation) | Value A — Full Screen | Value B — Bottom Left (1000x580 at Y=500) |
|---|---|---|
| `InputArea.OverlapArea.Extent.X` | 21600 | 41472 |
| `InputArea.OverlapArea.Extent.Y` | 13500 | 25137 |
| `InputArea.OverlapArea.Origin.Y` | 0 | -11637 |
| `ScreenArea.AreaType` | **0** | **1** |
| `ScreenArea.ScreenOutputArea.Extent.X` | 1920 | 1000 |
| `ScreenArea.ScreenOutputArea.Extent.Y` | 1080 | 580 |
| `ScreenArea.ScreenOutputArea.Origin.Y` | 0 | 500 |

`InputArea.OverlapArea` values (tablet-native coordinates) change automatically when the mapping changes via GUI. It is not confirmed whether these need to be written manually during import or whether the driver recalculates them. Plan 01-02 should test both approaches.

### 4. Coordinate Semantics

- `ScreenOutputArea/Origin/X` — left edge of the mapped screen region, in **physical pixels**
- `ScreenOutputArea/Origin/Y` — top edge of the mapped screen region, in **physical pixels**
- `ScreenOutputArea/Extent/X` — width of the mapped screen region, in **physical pixels**
- `ScreenOutputArea/Extent/Y` — height of the mapped screen region, in **physical pixels**
- `AreaType` must be set to `1` when a custom region is active (value `0` = full screen)
- `WhichMonitor` identifies the target display; `Desktop_0_` observed for the single-monitor test setup

### 5. Wacom Service

| Name | DisplayName | Status |
|------|-------------|--------|
| `WtabletServicePro` | Wacom Professional Service | Running |

No `*wacom*` or `*wintab*` services found beyond `WtabletServicePro`.

### 6. Environment

- Windows 11 version 23H2 (Build 22631.6199)
- Wacom driver version 6.4.12-3
- Tablet: Wacom One M

## Decisions Made

1. **PrefUtil.exe is the correct binary name.** `Wacom_TabletUserPrefs.exe` (referenced in Wacom developer docs) does not exist on driver 6.4.12-3. `PrefUtil.exe` is the current name.
2. **File extension is `.Export.wacomxs`, not `.xml`.** Specifying `.xml` in the export CLI argument causes the export to silently fail at the specified path. The extension must match exactly.
3. **`AreaType` must be changed to `1` for a custom region.** The field is present in all `ArrayElement` entries and controls whether the driver uses full-screen or the `ScreenOutputArea` coordinates.
4. **`InputArea.OverlapArea` values are calculated by the driver.** They changed automatically when the region was set via GUI. Plan 01-02 must test whether import works correctly without manually setting these values.
5. **Phase 2 native host cannot use PrefUtil for silent import.** This is a blocking architectural concern — see Deviations below.

## Deviations from Plan

### Major Deviation — PrefUtil Is Not Headless

**[Architectural Finding] PrefUtil.exe requires GUI interaction for both import and export**

- **Found during:** Task 1 (Windows test execution)
- **Issue:** The plan and prior research assumed PrefUtil supports a `/silent` mode for headless operation. In practice, `/silent` only suppresses the window for the help/info screen; it has no effect on `/import`. Every import and every export requires the user to click OK in a native Windows dialog.
- **Impact on Phase 2:** The C# native messaging host CANNOT call `PrefUtil.exe /import` as the mechanism to apply a mapping change silently. A user click would appear on screen every time the browser extension sends an update — completely unacceptable for the annotation workflow.
- **Required architectural decision before Phase 2 begins:**
  - **Option A (preferred to test in 01-02):** Write the XML file directly to the Wacom AppData preferences path and restart `WtabletServicePro` to force the driver to reload. Risk: unsupported; may corrupt driver state if format is wrong.
  - **Option B:** Investigate whether the Wacom driver exposes a COM API, named pipe, or registry hook for programmatic preference changes without PrefUtil.
  - **Option C:** Investigate whether an older PrefUtil version or an undocumented flag achieves silent import on this driver version.
  - **Option D:** Accept the dialog as a one-time setup step and redesign the user flow (not viable for automatic per-PDF annotation).
- **Status:** Unresolved — Option A should be tested in Plan 01-02 as the first experiment.

---

**Total deviations:** 1 major architectural finding (not auto-fixable — requires decision and testing in Plan 01-02)

**Impact on plan:** XML structure discovery is complete and accurate. Silent invocation is blocked. Plans 01-02 and 01-03 can proceed for the scripting and PowerShell side, but Phase 2 (C# native host) architecture must be reconsidered pending Option A test results.

## Issues Encountered

- **Export path not honored with `.xml` extension:** Initial export attempts to a custom `.xml` path produced no file. Discovered that the extension must be `.Export.wacomxs`. Resolved by using the correct extension.
- **No shell output from PrefUtil:** All three help flag variants (`/?`, `--help`, `-help`) open a GUI window with no stdout output. CLI-based flag discovery is not possible; the Wacom developer documentation link in the test log is the authoritative reference.
- **XPath typos in test-log.md:** The XPath recorded in `spike/test-log.md` section 2 contains typos (`ArraElement`, `ArrayELement`). The corrected, confirmed XPath for use in scripts is: `//InputScreenAreaArray/ArrayElement/ScreenArea`.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

**Ready for Plan 01-02:**
- XML XPath and coordinate semantics fully documented and confirmed from real baseline files
- Both baseline files committed for reference during PowerShell script development
- Service name (`WtabletServicePro`) confirmed for use in service-restart testing

**Open questions Plan 01-02 must answer:**
- Does writing the XML file directly + restarting `WtabletServicePro` apply the mapping silently (Option A test)?
- Must `InputArea.OverlapArea` values be set manually in the written XML, or does the driver recalculate them on service restart?
- What is the correct AppData path where Wacom reads its preference XML on startup?

## Self-Check: PASSED

- spike/test-log.md: FOUND (6 sections present, filled by user)
- spike/baseline-reference.Export.wacomxs: FOUND (committed by user)
- spike/baseline-modified.Export.wacomxs: FOUND (committed by user)
- spike/.gitkeep: FOUND
- .gitignore: FOUND
- Commit cd1af89: verified (Task 2 scaffold)

---
*Phase: 01-wacom-mapping-spike*
*Completed: 2026-04-30*
