---
phase: 03-browser-extension
plan: 03
subsystem: browser-extension
tags: [shadow-dom, banner, options-page, chrome-storage, ui]
dependency_graph:
  requires: [03-01]
  provides: [banner-component, options-page]
  affects: [extension/content.js]
tech_stack:
  added: []
  patterns: [shadow-dom-closed, createElement-textContent, chrome-storage-sync]
key_files:
  created:
    - extension/banner.js
    - extension/options.html
    - extension/options.js
    - extension/options.css
  modified:
    - extension/content.js
decisions:
  - "Wrote complete content.js (combining 03-02 and 03-03 changes) since both plans run in parallel wave 2 on separate worktrees"
  - "Shadow root mode 'closed' per UI-SPEC.md and threat model T-03-13"
  - "createElement + textContent used exclusively — no innerHTML with user data"
metrics:
  duration: "~15 minutes"
  completed: 2026-05-04
  tasks_completed: 2
  tasks_total: 2
  files_created: 4
  files_modified: 1
---

# Phase 03 Plan 03: Shadow DOM Banner and Options Page Summary

One-liner: Closed-mode Shadow DOM banner with four states (idle/synced/area-changed/host-not-found) and options page for chrome.storage.sync tuple management.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create extension/banner.js + wire content.js | ca8ccbf | extension/banner.js, extension/content.js |
| 2 | Create extension/options.html + options.js + options.css | a721e9b | extension/options.html, extension/options.js, extension/options.css |

## What Was Built

### Task 1: banner.js + content.js wiring

`extension/banner.js` exports `createBanner(targetElement, onSyncClickCallback)` which:
- Appends a host `<div>` to `document.body` with `position:absolute; z-index:2147483647`
- Attaches a closed Shadow DOM (`attachShadow({ mode: 'closed' })`)
- Inlines verbatim CSS from UI-SPEC.md inside the shadow root `<style>` element
- Sets `role="status"` and `aria-live="polite"` on the inner banner div
- Renders four states using `createElement` + `textContent` (no innerHTML):
  - `idle`: "Wacom not synced" + "Sync now" button
  - `synced`: "Wacom area synced" — auto-dismisses after 3000ms via `host.style.display='none'`
  - `area-changed`: "PDF area changed" + "Re-calibrate" button
  - `host-not-found`: "Native host not found" + "Install Wacom Bridge to activate" hint, no button
- `pending` state disables the action button without rebuilding the DOM
- Repositions above target element on every `update()` call: `rect.top + scrollY - 40px`

`extension/content.js` updated to:
- Import `createBanner` from `./banner.js`
- In `waitForElement` callback: call `createBanner(element, () => syncMapping(element))`
- Assign `updateBannerState = banner.update`
- Remove `window.__wacomBridgeSync` seam (replaced by callback pattern)

Note: content.js in this worktree also includes the 03-02 implementation (coordinate calculation, staleness detection, syncMapping) since both plans modify content.js in parallel wave 2.

### Task 2: options.html + options.js + options.css

`extension/options.html`:
- Title: "Wacom Bridge — Target Elements"
- `#tuple-list` container for dynamically rendered rows
- `#empty-message` hidden initially, shown when tuple array is empty
- `#btn-add` ("Add target"), `#btn-save` ("Save targets"), `#status-message` (ARIA live region)
- `<script type="module" src="options.js">`

`extension/options.js`:
- Reads tuples from `chrome.storage.sync.get({ targets: DEFAULT_TARGETS })` on load
- Renders each tuple as a row with Element ID + URL Pattern inputs + "Remove target" button
- `validateTuple()`: "Required" for empty fields, "Must start with http://, https://, or file://" for invalid URL patterns
- Save: validates all rows, blocks save if any invalid, shows "Saved." (2000ms) or "Save failed — storage quota exceeded."
- Add target: pushes empty tuple, re-renders, focuses new row's first input
- All user-data DOM writes use `textContent` or `input.value` — no innerHTML with user data

`extension/options.css`:
- `.btn-primary`: `background: #1976d2` (accent)
- `.btn-destructive`: `color: #f44336; border: 1px solid #f44336`
- `.tuple-row`: `display: flex; gap: 8px; align-items: flex-start`
- `.tuple-field input.invalid`: `border-color: #f44336`
- `.field-error`: `font-size: 12px; color: #f44336`

## Deviations from Plan

### Deviation 1: content.js includes 03-02 implementation

- **Found during:** Task 1 planning
- **Issue:** content.js at the base commit (97f0e24) is the 03-01 stub. Plan 03-03 task 1 instructs updating content.js to use `syncMapping`, `initObservers`, and `updateBannerState` — functions that plan 03-02 adds in parallel. Writing only the banner wiring changes would leave content.js referencing undefined functions.
- **Fix:** Wrote the complete content.js incorporating both 03-02 (getPhysicalCoords, staleness detection, syncMapping, initObservers) and 03-03 (createBanner import, updateBannerState = banner.update, seam removal) implementations. The merge process will reconcile with 03-02's worktree.
- **Rule applied:** Rule 3 (auto-fix blocking issue)
- **Files modified:** extension/content.js
- **Commit:** ca8ccbf

## Threat Model Compliance

All mitigations from plan threat register applied:

| Threat | Mitigation Applied |
|--------|--------------------|
| T-03-11 Tampering — options.js renderTupleList | `input.value` assignment, `textContent` for labels/errors — no innerHTML with user data |
| T-03-12 Tampering — XSS via elementId into content.js | elementId passed only to `document.getElementById()` |
| T-03-13 EoP — host page JS accessing shadow root | `attachShadow({ mode: 'closed' })` — returns null to host scripts |
| T-03-14 DoS — storage quota | try/catch in btn-save shows "Save failed — storage quota exceeded." |

## Known Stubs

None. All banner states render real content. Options page reads and writes real chrome.storage.sync data.

## Self-Check: PASSED

- extension/banner.js: exists, 232 lines
- extension/options.html: exists, 32 lines
- extension/options.js: exists, 200 lines
- extension/options.css: exists, 139 lines
- extension/content.js: updated with createBanner import and wiring
- Commits ca8ccbf and a721e9b verified in git log
