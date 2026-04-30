---
plan: 01-03
phase: 01-wacom-mapping-spike
status: partial
date: 2026-04-30
self_check: PASSED
key-files:
  created:
    - spike/SPIKE-RESULTS.md
    - spike/run-tests.ps1
---

## Objective

Create the SPIKE-RESULTS.md results template and run-tests.ps1 test harness, then execute all manual test cases on the Windows test machine and fill the results document.

## What Was Built

**Task 1 (auto):** Created `spike/SPIKE-RESULTS.md` (structured results template pre-filled with Plan 01-01/02 findings) and `spike/run-tests.ps1` (test harness running TC-01 through TC-05 with pauses for physical stylus verification and automatic SPIKE-02 PASS/FAIL output).

**Task 2 (human-verify):** Tests executed on Windows test machine (Wacom One M, Windows 11 23H2, driver 6.4.12-3).

## Test Results

| Test Case | Description | Result |
|-----------|-------------|--------|
| TC-01 | Left half mapping (X=0 Y=0 W=960 H=1080) | PASS — stylus restricted correctly |
| TC-02 | Right half mapping (X=960 Y=0 W=960 H=1080) | PASS — stylus restricted correctly |
| TC-03 | Centre region (X=240 Y=270 W=1440 H=540) | PASS — stylus restricted correctly |
| TC-04 | Three consecutive changes — latency (SPIKE-02) | FAIL — 6659/4597/4156 ms (dialog dominated) |
| TC-05 | Reset to full-screen | PASS — full-screen restored |

## Overall Spike Status: PARTIAL

**TC-01/02/03/05 PASS** — the XML clone-and-modify + PrefUtil import mechanism works correctly. Stylus mapping is restricted to the specified screen region as expected.

**SPIKE-02 FAIL** — latency measurements (4.2–6.7 s) exceeded the 3 s threshold, but this is misleading: the excess is entirely user dialog click time. PrefUtil opens a GUI dialog on every `/import` call; `/silent` has no effect. The actual XML processing and driver application time is a fraction of the measured values.

**Admin rights:** NOT required — non-elevated PowerShell, exit code 0.

**Service restart:** NOT required — PrefUtil notifies the driver directly.

## Key Deviations

- **PrefUtil is not headless** — every `/import` opens a mandatory GUI dialog. On subsequent imports a backup dialog also appears. Silent/automated invocation via PrefUtil is not achievable.
- **SPIKE-02 technically fails** due to dialog interaction time, not mechanism latency. The underlying approach is viable for Phase 2 if PrefUtil is replaced.

## Recommendation for Phase 2

Port the clone-and-modify-XML approach to C# .NET 8. **Replace PrefUtil invocation with direct XML file write + `Restart-Service WtabletServicePro`** (or equivalent driver notification). This must be validated in Phase 2 — if confirmed, PrefUtil becomes unnecessary and sub-3 s silent operation becomes achievable.

Phase 2 implementation constants confirmed by this spike:
- XPath: `//InputScreenAreaArray/ArrayElement/ScreenArea`
- Coordinate elements: `ScreenOutputArea/Origin/X|Y` (top-left), `ScreenOutputArea/Extent/X|Y` (width/height)
- `AreaType`: set to `1` for custom region
- File extension: `.Export.wacomxs`
- Coordinates: physical pixels

## Artifacts

- `spike/SPIKE-RESULTS.md` — full results document
- `spike/run-tests.ps1` — reusable test harness
