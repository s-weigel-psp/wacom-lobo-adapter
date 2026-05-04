---
phase: 03-browser-extension
verified: 2026-05-04T18:00:00Z
status: human_needed
score: 13/13 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 4/13
  gaps_closed:
    - "getPhysicalCoords() multiplying all four values by devicePixelRatio"
    - "Staleness detection with hasPriorSync gate, three triggers, debounced onStale()"
    - "ResizeObserver, window resize, screenX/Y setInterval funneling through onStale()"
    - "screenX/Y polling pauses on document.hidden, resumes on visibilitychange"
    - "Clicking Sync now or Re-calibrate sends set_mapping with freshly computed coordinates"
    - "Banner transitions to area-changed state after staleness trigger (post first sync)"
    - "Shadow DOM banner: position:absolute, z-index:2147483647"
    - "Banner repositions above target element on every state update using rect.top + scrollY - 40px"
    - "Banner renders four states: idle, synced, area-changed, host-not-found"
    - "Synced state auto-dismisses after 3000ms"
    - "Options page reads and writes tuples to chrome.storage.sync"
    - "Options page validates empty fields and invalid URL patterns"
    - "host-not-found state shows 'Native host not found' + 'Install Wacom Bridge to activate' hint, no button"
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "Load extension in Chrome (chrome://extensions → Load unpacked → extension/ directory)"
    expected: "Extension loads without errors; banner appears above #draw-canvas on pages matching file:///C:/WacomTest/*.html"
    why_human: "Cannot run Chrome extension host in a headless environment"
  - test: "Open a test page matching file:///C:/WacomTest/*.html with a #draw-canvas element; observe idle banner; click 'Sync now'"
    expected: "Banner transitions idle → pending (button disabled) → synced; auto-dismisses after 3 seconds; native host receives set_mapping with DPR-corrected physical pixel coordinates"
    why_human: "Requires Windows environment with Wacom driver and native host installed; real-time UI behavior"
  - test: "After syncing, resize the browser window"
    expected: "Banner reappears in area-changed state within ~300ms; 'Re-calibrate' button visible; clicking it re-sends set_mapping and transitions back to synced"
    why_human: "Real-time browser event behavior requires manual observation"
---

# Phase 3: Browser Extension Verification Report

**Phase Goal:** Chrome/Edge Manifest V3 extension that detects the target DOM element, computes its screen coordinates (DPR-corrected), drives the native host, and shows a banner when the area changes.
**Verified:** 2026-05-04T18:00:00Z
**Status:** HUMAN NEEDED
**Re-verification:** Yes — after Wave 2 merge into worktree branch (commit 8647452)

## Re-verification Summary

Previous verification (2026-05-04T17:30:00Z) found 9 gaps, all with a single root cause: commit `8647452` was not on `main`. The user confirmed Wave 2 work is now merged. The merge commit `8647452` is on the current worktree branch (`worktree-agent-a3ff39afe2d0b41c9`).

**Note on git state:** As of this verification, `main` at `97f0e24` still only contains the Wave 1 (03-01) scaffold. The completed implementation (Wave 2) lives on `worktree-agent-a3ff39afe2d0b41c9` at commit `8647452`. All 13 must-haves are verified against that branch. Once the worktree is merged to `main`, the implementation is correct and complete.

All 9 previously-failed items are now VERIFIED. No regressions on the 4 previously-passing items. Score advances from 4/13 to 13/13.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | manifest.json is valid MV3 with nativeMessaging and storage permissions | VERIFIED | JSON valid; manifest_version:3; permissions: ["nativeMessaging","storage"]; background.service_worker: "background.js" |
| 2 | background.js registers onMessage listener, calls sendNativeMessage to com.brantpoint.wacombridge, returns true | VERIFIED | Lines 8–24: listener, sendNativeMessage(HOST_NAME, payload), return true present |
| 3 | config.js exports DEFAULT_TARGETS, HOST_NAME, POLL_INTERVAL_MS, DEBOUNCE_MS, AUTO_DISMISS_MS | VERIFIED | All 5 exports with correct values (HOST_NAME='com.brantpoint.wacombridge', POLL=1000, DEBOUNCE=300, DISMISS=3000) |
| 4 | content.js reads target tuples from chrome.storage.sync on load | VERIFIED | getTargets() at line 65 calls chrome.storage.sync.get({targets: DEFAULT_TARGETS}); main() calls getTargets() |
| 5 | getPhysicalCoords() multiplies ALL four values (x, y, width, height) by window.devicePixelRatio | VERIFIED | Lines 116–119; node test-coords.js: 3 passed, 0 failed at 100%, 125%, 150% DPI |
| 6 | Staleness detection activates three triggers only after a successful set_mapping call | VERIFIED | hasPriorSync gate at line 133; startPolling() called only on success (line 202); 4 occurrences of hasPriorSync |
| 7 | ResizeObserver, window resize event, and screenX/Y setInterval all funnel through debounced onStale() | VERIFIED | initObservers(): ResizeObserver(onStale) line 217, addEventListener('resize', onStale) line 221, setInterval in startPolling() line 148 |
| 8 | screenX/Y polling pauses when document.hidden, resumes on visibilitychange | VERIFIED | setInterval body: `if (document.hidden) return` line 149; visibilitychange listener at line 224; stopPolling()/startPolling() called on visibility change |
| 9 | Clicking Sync now or Re-calibrate sends set_mapping with freshly computed coordinates | VERIFIED | syncMapping() at line 172 builds payload with command:'set_mapping'; called via onSyncClickCallback in banner.js; createBanner wired at content.js line 265 |
| 10 | Banner transitions to area-changed state after any staleness trigger fires (post first sync) | VERIFIED | onStale() → setTimeout → updateBannerState('area-changed'); updateBannerState = banner.update (line 268) |
| 11 | Shadow DOM banner attaches to document.body with position:absolute and z-index:2147483647 | VERIFIED | banner.js lines 101–108: host div appended to body, position:'absolute', zIndex:'2147483647' |
| 12 | Banner repositions above target element on every state update using rect.top + scrollY - 40px | VERIFIED | reposition() at banner.js line 132: `host.style.top = rect.top + scrollY - 40 + 'px'`; called in update() and on every state transition |
| 13 | Banner renders four states with correct CSS classes and copy; synced auto-dismisses after 3000ms | VERIFIED | All four switch cases present (idle/synced/area-changed/host-not-found); exact copy strings match UI-SPEC.md; AUTO_DISMISS_MS setTimeout at line 190–192 |

**Score: 13/13 truths verified**

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `extension/manifest.json` | MV3 manifest | VERIFIED | 27 lines; valid JSON; manifest_version 3; all required permissions |
| `extension/background.js` | Service worker relay | VERIFIED | 24 lines; onMessage listener; sendNativeMessage; return true |
| `extension/config.js` | Shared constants | VERIFIED | 17 lines; all 5 exports correct |
| `extension/content.js` | Full content script (03-02+03-03) | VERIFIED | 275 lines; getPhysicalCoords, onStale, startPolling, syncMapping, initObservers, createBanner import, updateBannerState = banner.update; no __wacomBridgeSync seam |
| `extension/banner.js` | Shadow DOM banner (min 100 lines) | VERIFIED | 232 lines; closed shadow DOM; all 4 states; correct copy; AUTO_DISMISS_MS; ARIA attributes |
| `extension/options.html` | Options page HTML | VERIFIED | 32 lines; title correct; tuple-list, btn-add, btn-save, status-message IDs present; type=module |
| `extension/options.js` | Options page logic (min substantive) | VERIFIED | 200 lines; chrome.storage.sync.get/set; validateTuple with Required and URL pattern error; Saved./quota error |
| `extension/options.css` | Options page styles | VERIFIED | 139 lines; .btn-primary, .btn-destructive, .tuple-row, .tuple-field input.invalid |
| `extension/test-coords.js` | DPR formula test harness | VERIFIED | Exists; node test-coords.js exits 0, 3 passed 0 failed |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| background.js | com.brantpoint.wacombridge | chrome.runtime.sendNativeMessage | VERIFIED | Line 10: sendNativeMessage(HOST_NAME, message.payload) |
| content.js | config.js | ES module import | VERIFIED | Lines 6–12: imports DEFAULT_TARGETS, HOST_NAME, POLL_INTERVAL_MS, DEBOUNCE_MS, AUTO_DISMISS_MS |
| content.js | banner.js | import { createBanner } | VERIFIED | Line 14: import { createBanner } from './banner.js' |
| manifest.json | background.js | background.service_worker | VERIFIED | "service_worker": "background.js", "type": "module" |
| content.js getPhysicalCoords() | protocol.md Section 3 DPR formula | Math.round((rect.left + window.screenX) * dpr) | VERIFIED | Lines 116–119; formula matches spec exactly; test harness confirms |
| content.js sendNativeCommand() | background.js onMessage | NATIVE_COMMAND message | VERIFIED | sendNativeCommand sends {type:'NATIVE_COMMAND', payload}; called from syncMapping(); syncMapping called from createBanner callback |
| banner.js createBanner() | content.js syncMapping() | onSyncClickCallback parameter | VERIFIED | content.js line 265: createBanner(element, () => syncMapping(element)); banner buttons call onSyncClickCallback() |
| options.js | chrome.storage.sync | chrome.storage.sync.set({ targets }) | VERIFIED | Line 16 of options.js; saveTargets() called in btn-save handler after validation passes |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| content.js | targets | chrome.storage.sync.get({targets: DEFAULT_TARGETS}) | Yes — reads from sync storage with DEFAULT_TARGETS fallback | FLOWING |
| content.js | matchingTarget | targets.find() with regex glob match | Yes — URL pattern matching logic correct and escaped | FLOWING |
| content.js | coords (in syncMapping) | getPhysicalCoords(element) via getBoundingClientRect() + DPR | Yes — live DOM geometry, not hardcoded | FLOWING |
| banner.js | DOM state | update(state) called with real state values from syncMapping/onStale | Yes — state driven by actual native host response and resize events | FLOWING |
| options.js | tuples | chrome.storage.sync.get on load; written back with chrome.storage.sync.set on save | Yes — real chrome.storage.sync reads and writes | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| DPR formula at 150% DPI | `node extension/test-coords.js` | 3 passed, 0 failed | PASS |
| manifest.json parses as valid JSON | python3 json.load check | No error | PASS |
| HOST_NAME matches installer manifests | grep com.brantpoint.wacombridge config.js | Line 13 match | PASS |
| return true keepalive present | grep "return true" background.js | Line 22 match | PASS |
| __wacomBridgeSync seam removed | grep __wacomBridgeSync content.js | 0 matches | PASS |
| attachShadow mode closed | grep attachShadow banner.js | Line 112 match | PASS |
| banner copy strings present | grep "Wacom not synced\|Native host not found" banner.js | Lines 173, 211 match | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| EXT-01 | 03-02 | Content script detects target DOM element by configured ID | SATISFIED | waitForElement() uses getElementById + MutationObserver fallback; called in main() after URL pattern match |
| EXT-02 | 03-02 | Extension calculates screen coordinates accounting for DPI scaling | SATISFIED | getPhysicalCoords() implements protocol.md Section 3 formula exactly; DPR test harness passes 3/3 |
| EXT-03 | 03-01, 03-02 | Extension sends mapping coordinates to native host on user activation | SATISFIED | syncMapping() sends {command:'set_mapping', x, y, width, height} via sendNativeCommand → background.js relay → sendNativeMessage |
| EXT-04 | 03-03 | Extension shows Shadow DOM banner when PDF area position or size changes | SATISFIED | banner.js: closed shadow DOM; onStale() triggers updateBannerState('area-changed') after debounce; all triggers wired |
| EXT-05 | 03-03 | User can re-sync by clicking banner calibration button | SATISFIED | Re-calibrate button in area-changed state calls onSyncClickCallback → syncMapping(element) |
| EXT-06 | 03-03 | Extension handles native host unavailability gracefully | SATISFIED | syncMapping() maps ERR_ codes and catch errors to updateBannerState('host-not-found'); banner renders "Native host not found" + hint, no button |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| extension/options.js | 79, 105 | `placeholder = 'e.g. ...'` | INFO | HTML input placeholder attributes — not stubs, correct UX copy from UI-SPEC.md |
| extension/content.js | 25–28 | `let updateBannerState = (state) => { console.log(...) }` | INFO | Initial stub value — intentionally overwritten at line 268 with `banner.update` after element found; not a shipped stub |
| extension/banner.js | 165 | `banner.innerHTML = ''` | INFO | Used to clear previous state DOM before rebuilding with createElement+textContent — not user-data injection; safe |

No blockers or warnings found. All three flagged patterns are benign on inspection.

### Human Verification Required

#### 1. Extension Load Test

**Test:** Sideload the extension via `chrome://extensions` → Load unpacked → point to the `extension/` directory
**Expected:** Extension loads without errors; extension ID assigned; banner visible in top-right of pages matching `file:///C:/WacomTest/*.html`
**Why human:** Cannot run Chrome extension host in a headless environment

#### 2. Banner Render and Full Sync Flow

**Test:** Open a test page matching `file:///C:/WacomTest/*.html` containing `<canvas id="draw-canvas">`; observe the idle banner; click "Sync now"
**Expected:** Banner transitions idle → pending (button disabled during request) → synced ("Wacom area synced"); auto-dismisses after 3 seconds; native host receives `set_mapping` with DPR-corrected pixel coordinates
**Why human:** Requires Windows environment with Wacom driver and native host installed; real-time UI state transitions require visual observation

#### 3. Staleness Detection and Re-calibrate

**Test:** After a successful sync, resize the browser window or move the browser to a different screen position
**Expected:** Banner reappears in area-changed state ("PDF area changed" + "Re-calibrate" button) within ~300ms; clicking "Re-calibrate" transitions back to synced
**Why human:** Real-time browser event behavior and Wacom driver response require manual observation on target hardware

### Gaps Summary

No gaps remain. All 13 must-haves are verified. The 3 human verification items are environment-dependent (Windows + Wacom driver + Chrome extension host) and cannot be verified programmatically.

**Git note for attention:** The complete implementation exists on `worktree-agent-a3ff39afe2d0b41c9` at commit `8647452`. The `main` branch remains at `97f0e24` (Wave 1 only). The orchestrator should ensure `8647452` is merged to `main` before considering Phase 3 closed.

---

_Verified: 2026-05-04T18:00:00Z_
_Verifier: Claude (gsd-verifier)_
