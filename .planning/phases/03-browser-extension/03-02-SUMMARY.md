---
phase: 03-browser-extension
plan: 02
subsystem: extension/content.js
tags: [coordinate-calculation, staleness-detection, native-messaging, dpr, content-script]
dependency_graph:
  requires: [03-01]
  provides: [coordinate-calculation, staleness-detection, sync-action]
  affects: [03-03]
tech_stack:
  added: []
  patterns:
    - DPR physical pixel conversion via Math.round((rect.left + window.screenX) * devicePixelRatio)
    - ResizeObserver + window resize + setInterval three-trigger staleness detection
    - Debounced onStale() with clearTimeout guard (300ms)
    - Page Visibility API pause/resume for screenX/Y poll
    - ERR_ prefix detection for native host error mapping
key_files:
  created:
    - extension/test-coords.js
  modified:
    - extension/content.js
decisions:
  - getPhysicalCoords uses rect.width/rect.height (not rect.right/rect.bottom) to avoid double-counting viewport offsets
  - hasPriorSync gate prevents staleness transitions before first successful set_mapping
  - updateBannerState is a console.log stub until Plan 03-03 wires Shadow DOM banner
  - window.__wacomBridgeSync is a temporary integration seam; Plan 03-03 removes it
metrics:
  duration: 116s
  completed: 2026-05-04
  tasks_completed: 2
  files_modified: 2
---

# Phase 03 Plan 02: Content Script Coordinate Calculation and Staleness Detection Summary

DPR-corrected physical pixel coordinate calculation, three-trigger staleness detection with debounce, Page Visibility pause/resume, and set_mapping sync action in extension/content.js.

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | Implement getPhysicalCoords() (TDD) | 9754c35 | extension/content.js, extension/test-coords.js |
| 2 | Implement staleness detection and sync handlers | 9754c35 | extension/content.js |

## Implementation Summary

### Task 1: getPhysicalCoords()

Implemented the DPR formula from docs/protocol.md Section 3:

```javascript
function getPhysicalCoords(element) {
  const rect = element.getBoundingClientRect();
  const dpr  = window.devicePixelRatio;
  return {
    x:      Math.round((rect.left   + window.screenX) * dpr),
    y:      Math.round((rect.top    + window.screenY) * dpr),
    width:  Math.round(rect.width   * dpr),
    height: Math.round(rect.height  * dpr),
  };
}
```

Node.js test harness (`extension/test-coords.js`) verified: 3 passed, 0 failed at 100%, 125%, and 150% DPI.

### Task 2: Staleness Detection and Sync

- `onStale()`: debounced 300ms, guarded by `hasPriorSync` (no stale transition before first sync)
- `startPolling()` / `stopPolling()`: screenX/Y interval at POLL_INTERVAL_MS (1s), pauses on `document.hidden`
- `initObservers()`: wires ResizeObserver, window resize event, and visibilitychange
- `syncMapping()`: sends `{ command: 'set_mapping', x, y, width, height }` via sendNativeCommand, handles `ERR_` prefix error codes, sets `hasPriorSync = true` on success and calls `startPolling()`
- `updateBannerState`: console.log stub for Plan 03-03
- `window.__wacomBridgeSync`: temporary seam for Plan 03-03 button wiring

## Deviations from Plan

None — plan executed exactly as written. Tasks 1 and 2 were committed together in a single atomic commit since Task 2 extends the same file written in Task 1.

## Known Stubs

| Stub | File | Line | Reason |
|------|------|------|--------|
| `updateBannerState` (console.log) | extension/content.js | 119 | Plan 03-03 replaces with Shadow DOM banner update |
| `window.__wacomBridgeSync` | extension/content.js | 247 | Plan 03-03 wires button click directly; this seam is then removed |

These stubs are intentional and documented — Plan 03-03 wires the real banner. The plan's goal (coordinate calculation + staleness detection) is fully achieved.

## Threat Surface Scan

No new security surface beyond the plan's threat model. T-03-06 (elementId via getElementById only), T-03-07 (debounce coalescing rapid triggers), and T-03-09 (regex-escaped URL glob) are all implemented as planned.

## Self-Check: PASSED

- `extension/content.js` exists: FOUND
- `extension/test-coords.js` exists: FOUND
- Commit 9754c35 exists: FOUND (confirmed via git show)
- `node extension/test-coords.js`: 3 passed, 0 failed
- `grep "function getPhysicalCoords"`: line 96
- `grep "hasPriorSync" | wc -l`: 4 (declaration + guard + assignment + visibilitychange resume)
- `grep "command.*set_mapping"`: line 174
- `grep "ERR_"`: line 193 (startsWith check)
- `grep "visibilitychange"`: line 223
