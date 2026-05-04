---
phase: 03-browser-extension
plan: "01"
subsystem: browser-extension
tags: [chrome-extension, mv3, native-messaging, service-worker, content-script]
dependency_graph:
  requires: []
  provides:
    - extension/manifest.json — MV3 extension manifest with all required permissions
    - extension/config.js — single source of truth for all extension constants
    - extension/background.js — service worker relay to native host
    - extension/content.js — content script stub with messaging relay and element detection
  affects:
    - installer/manifest-chrome.json — PLACEHOLDER_CHROME_EXTENSION_ID must be replaced after sideloading
    - installer/manifest-edge.json — PLACEHOLDER_EDGE_EXTENSION_ID must be replaced after sideloading
tech_stack:
  added:
    - Chrome/Edge Manifest V3 extension scaffold (vanilla JS, no build step)
  patterns:
    - ES module imports in service worker and content script (background.type: "module")
    - sendNativeMessage per-call pattern (not connectNative) for MV3 ephemeral service worker
    - MutationObserver + getElementById for lazy element detection (EXT-01)
    - chrome.storage.sync with DEFAULT_TARGETS fallback for configurable targets (D-01, D-03)
    - return true keepalive in onMessage for async sendResponse (RESEARCH.md Pitfall 2)
key_files:
  created:
    - extension/manifest.json
    - extension/config.js
    - extension/background.js
    - extension/content.js
    - extension/icons/icon-16.png
    - extension/icons/icon-48.png
    - extension/icons/icon-128.png
  modified: []
decisions:
  - sendNativeMessage per-call chosen over connectNative persistent port — MV3 service worker is ephemeral; low-frequency sync events do not warrant port lifecycle management overhead (RESEARCH.md recommendation)
  - Static content_scripts.matches for Phase 3 — user-added URL patterns require extension reload (RESEARCH.md Pitfall 4, Assumption A4); dynamic injection via chrome.scripting deferred to Phase 4
  - Placeholder 1x1 PNG icons — not fill-in-before-ship blocking; Chrome accepts any valid PNG
metrics:
  duration: "2 minutes"
  completed: "2026-05-04T15:00:11Z"
  tasks_completed: 3
  files_created: 7
  files_modified: 0
---

# Phase 3 Plan 01: Extension Scaffold Summary

**One-liner:** MV3 extension scaffold with manifest, config constants, service-worker relay to com.brantpoint.wacombridge, and content-script stub with MutationObserver element detection.

## What Was Built

The `extension/` directory was created from scratch with four foundational files:

1. **extension/manifest.json** — Valid MV3 manifest declaring `nativeMessaging` and `storage` permissions, background service worker with ES module type, static content_scripts injection into `file:///C:/WacomTest/*.html`, and options_ui declaration.

2. **extension/config.js** — Single source of truth for all extension constants. Exports `DEFAULT_TARGETS`, `HOST_NAME` (`com.brantpoint.wacombridge`), `POLL_INTERVAL_MS` (1000), `DEBOUNCE_MS` (300), `AUTO_DISMISS_MS` (3000). Marked FILL-IN-BEFORE-SHIP for production URL pattern and element ID replacement.

3. **extension/background.js** — MV3 service worker. Registers `chrome.runtime.onMessage` listener for `NATIVE_COMMAND` messages, relays to native host via `chrome.runtime.sendNativeMessage(HOST_NAME, payload)`, and returns `true` to keep the async message channel open. Normalizes `chrome.runtime.lastError` into `{ ok: false, error, code: 'ERR_HOST_UNAVAILABLE' }`.

4. **extension/content.js** — Content script stub. Imports all constants from config.js. Implements `sendNativeCommand()` relay, `getTargets()` storage read with DEFAULT_TARGETS fallback, `waitForElement()` with MutationObserver, and `main()` bootstrap with URL pattern matching and storage change listener. Plan 03-02 extends the `main()` body with `getPhysicalCoords()`, staleness detection, and banner integration.

5. **extension/icons/** — Three placeholder 1x1 pixel valid PNG files (icon-16, icon-48, icon-128) so Chrome accepts the extension on load.

## Sideloading Note

After sideloading via `chrome://extensions` → "Load unpacked":
1. Note the assigned extension ID (shown under the extension name)
2. For a stable development ID: use "Pack extension" → generate `.pem` key → add `"key": "<base64>"` to manifest.json
3. Replace `PLACEHOLDER_CHROME_EXTENSION_ID` in `installer/manifest-chrome.json` and `PLACEHOLDER_EDGE_EXTENSION_ID` in `installer/manifest-edge.json`
4. Reinstall the native messaging registration (re-run MSI or update registry-pointed JSON file)
5. Enable "Allow access to file URLs" on the extension's details page (required for `file://` injection — RESEARCH.md Pitfall 1)

See RESEARCH.md Pitfall 3 for the full sideloading workflow.

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

| Stub | File | Lines | Reason |
|------|------|-------|--------|
| `main()` bootstrap body | extension/content.js | 82–113 | Intentional stub — Plan 03-02 adds `getPhysicalCoords()`, ResizeObserver staleness detection, and banner.js initialization |
| Placeholder icon files | extension/icons/icon-{16,48,128}.png | — | 1x1 pixel placeholder PNGs — valid Chrome extension assets; replace before production release |
| `DEFAULT_TARGETS` values | extension/config.js | 5–9 | FILL-IN-BEFORE-SHIP: production `elementId` and `urlPattern` must replace test defaults before deployment |

These stubs do not prevent Plan 03-01's goal (extension scaffold) from being achieved. Plans 03-02 and 03-03 extend the stub implementations.

## Threat Flags

No new threat surface introduced beyond what the plan's threat model covers. The manifest is scoped to `file:///C:/WacomTest/*.html` only (T-03-03 mitigation in place). No new network endpoints, auth paths, or schema changes introduced.

## Self-Check: PASSED

Files exist:
- extension/manifest.json: FOUND
- extension/config.js: FOUND
- extension/background.js: FOUND
- extension/content.js: FOUND
- extension/icons/icon-16.png: FOUND
- extension/icons/icon-48.png: FOUND
- extension/icons/icon-128.png: FOUND

Commits:
- efd1841: feat(03-01): add MV3 manifest and placeholder icon assets
- 61ed354: feat(03-01): add config.js constants and background.js service worker relay
- 12b03b7: feat(03-01): add content.js stub with messaging relay and element detection
