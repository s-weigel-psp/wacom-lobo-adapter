---
phase: 01-wacom-mapping-spike
plan: "02"
subsystem: spike
tags: [powershell, wacom, prefutil, xml, windows, spike]

# Dependency graph
requires:
  - 01-01 (PrefUtil path, import flag, XPath, coordinate semantics)
provides:
  - spike/Set-WacomMapping.ps1 — sets Wacom screen mapping to arbitrary X/Y/Width/Height region
  - spike/Reset-WacomMapping.ps1 — restores full-screen mapping from baseline
  - Reference implementation for Phase 2 C# port
affects:
  - 01-03 (test execution uses both scripts as primary spike deliverables)
  - 02-native-host (C# port of these PowerShell patterns)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Clone-and-modify baseline: [xml]$doc = Get-Content baseline -Raw; modify; $doc.Save($TempPath)"
    - "XPath node selection: Select-Xml -Xml $doc -XPath '//InputScreenAreaArray/ArrayElement/ScreenArea' | Select-Object -ExpandProperty Node"
    - "PrefUtil invocation: Start-Process -FilePath $PrefUtilPath -ArgumentList '/import', $TempPath -PassThru then Wait-Process"
    - "Accurate timing: Measure-Command { ... } wrapping PrefUtil invocation (avoids 1-second poll floor of -Wait flag)"
    - "Child-element assignment: $screenArea.ScreenOutputArea.Origin.X = [string]$X (not attribute assignment)"

key-files:
  created:
    - spike/Set-WacomMapping.ps1
    - spike/Reset-WacomMapping.ps1
  modified: []

key-decisions:
  - "Iterate ALL ScreenArea ArrayElement entries (3 on test machine) rather than only AreaType=0 entries, for consistency"
  - "Use .Export.wacomxs extension for temp file (plain .xml silently fails per Plan 01-01 finding)"
  - "Document PrefUtil GUI dialog limitation in both script headers and Write-Host output so test engineer knows what to expect"
  - "No -Namespace parameter needed for Select-Xml — baseline XML has no namespace on root element"

requirements-completed:
  - SPIKE-01
  - SPIKE-04

# Metrics
duration: "< 5 min (fully automated, Linux dev machine)"
completed: "2026-04-30"
tasks_completed: 2
tasks_total: 2
---

# Phase 01 Plan 02: Write Spike Scripts Summary

**Set-WacomMapping.ps1 and Reset-WacomMapping.ps1 authored with confirmed PrefUtil path (`C:\Program Files\Tablet\Wacom\PrefUtil.exe`), `/import` flag, XPath `//InputScreenAreaArray/ArrayElement/ScreenArea`, child-element coordinate assignment, and GUI dialog warnings; ready for Plan 01-03 test execution.**

## Performance

- **Duration:** < 5 minutes (automated authoring on Linux dev machine)
- **Completed:** 2026-04-30
- **Tasks:** 2/2
- **Files modified:** 2

## Accomplishments

- Authored `spike/Set-WacomMapping.ps1` with correct XPath, child-element coordinate assignment, AreaType=1 enforcement across all ArrayElement entries, Measure-Command timing, and Wait-Process invocation pattern
- Authored `spike/Reset-WacomMapping.ps1` that re-imports `baseline-local.Export.wacomxs` without modification, consistent error handling and PrefUtil path with Set-WacomMapping.ps1
- Both scripts document the PrefUtil GUI dialog limitation with clear user-facing Write-Host messages
- Both scripts fail fast with descriptive errors if prerequisites (baseline file, PrefUtil.exe) are missing

## Task Commits

1. **Task 1: Set-WacomMapping.ps1** - `ff372b2` (feat) — spike/Set-WacomMapping.ps1
2. **Task 2: Reset-WacomMapping.ps1** - `82e0c19` (feat) — spike/Reset-WacomMapping.ps1

## Files Created/Modified

- `spike/Set-WacomMapping.ps1` — Restricts stylus to arbitrary screen region via clone-and-modify baseline import
- `spike/Reset-WacomMapping.ps1` — Restores full-screen mapping by re-importing baseline unmodified

## Spike Script Reference

### PrefUtil Invocation Details

| Item | Value |
|------|-------|
| Path | `C:\Program Files\Tablet\Wacom\PrefUtil.exe` |
| Import flag | `/import` |
| File extension | `.Export.wacomxs` (MUST match — `.xml` silently fails) |
| GUI dialog | Opens for every `/import` — user must click OK; `/silent` has no effect |
| Admin rights | Not required (non-elevated exit code 0 confirmed in Plan 01-01) |

### XPath Expression (Set-WacomMapping.ps1)

```
//InputScreenAreaArray/ArrayElement/ScreenArea
```

No namespace prefix needed — the baseline XML has no namespace on the root element (`<root type="map">`).

### Coordinate Assignment (Set-WacomMapping.ps1)

Coordinates are child XML elements with text content, NOT attributes. Assignment uses dot-notation:

```powershell
$screenArea.AreaType                      = [string]1        # 0=full screen, 1=custom
$screenArea.ScreenOutputArea.Origin.X     = [string]$X       # left edge (physical px)
$screenArea.ScreenOutputArea.Origin.Y     = [string]$Y       # top edge (physical px)
$screenArea.ScreenOutputArea.Extent.X     = [string]$Width   # width (physical px)
$screenArea.ScreenOutputArea.Extent.Y     = [string]$Height  # height (physical px)
```

All 3 `ArrayElement` entries in `InputScreenAreaArray` are updated for consistency.

### Baseline File Reference

| File | Purpose | Committed |
|------|---------|-----------|
| `baseline-local.Export.wacomxs` | Per-machine baseline; source for clone in Set-WacomMapping.ps1; reset target for Reset-WacomMapping.ps1 | No (gitignored) |
| `baseline-reference.Export.wacomxs` | Schema reference for Phase 2; NEVER imported | Yes |

## Deviations from Plan

### Expected Deviations — Extension Correction

The plan template (`01-02-PLAN.md`) was authored before Plan 01-01 confirmed the `.Export.wacomxs` file extension requirement. The plan's `<verify>` block and inline template code reference `baseline-local.xml`. All such references were updated to `baseline-local.Export.wacomxs` and `wacom-mapping-temp.Export.wacomxs` per the critical context directive and 01-01-SUMMARY.md findings.

This is not a bug — it is the expected outcome of a spike: Plan 01-01 discovered the correct extension, and Plan 01-02 applied it.

**Files affected:** Both scripts, this SUMMARY.

### Auto-fixed — All Three ArrayElement Entries Updated

The plan task description says "target only the ones with AreaType=0 (full screen)" as an option. Since the confirmed baseline has 3 entries (all with AreaType=0 in full-screen state), the simpler and more consistent approach is to iterate ALL entries unconditionally rather than filtering by current AreaType value. This avoids a state-dependency where a previously-set entry would retain its old coordinates because its AreaType was already 1.

## Known Stubs

None. Both scripts are fully functional reference implementations. `baseline-local.Export.wacomxs` is a runtime dependency (per-machine, gitignored) that must be exported on the test machine before running — this is documented in both script headers and the precondition error messages.

## Threat Flags

No new security-relevant surface introduced beyond the threat model in the plan:
- T-01-02-01: Temp file in `%TEMP%` — accepted (ephemeral, low-value spike context)
- T-01-02-02: Baseline integrity — mitigated (scripts never write to baseline; Reset re-imports it read-only)
- T-01-02-03: PrefUtil non-zero exit — mitigated (both scripts validate exit code and warn)
- T-01-02-04: PrefUtil path spoofing — accepted (developer-controlled test machine)

## Next Phase Readiness

**Ready for Plan 01-03:**
- Both scripts are complete with actual PrefUtil path, XPath, flags, and coordinate assignment
- No further editing required before test execution on the Windows machine
- Test engineer needs to: (1) ensure `baseline-local.Export.wacomxs` exists in `spike/`, (2) run `Set-WacomMapping.ps1 -X 0 -Y 0 -Width 960 -Height 1080`, click OK in dialog, verify stylus is restricted to left half of screen

**Open question carried from Plan 01-01:**
- Does writing the XML directly + restarting `WtabletServicePro` apply the mapping silently (Option A for Phase 2 silent operation)?
- These scripts use PrefUtil and will show the GUI dialog — Plan 01-03 confirms they work, but Phase 2 must find the silent alternative.

## Self-Check: PASSED

- spike/Set-WacomMapping.ps1: FOUND
- spike/Reset-WacomMapping.ps1: FOUND
- Commit ff372b2: verified (Task 1)
- Commit 82e0c19: verified (Task 2)

---
*Phase: 01-wacom-mapping-spike*
*Completed: 2026-04-30*
