# Phase 3: Browser Extension — Research

**Researched:** 2026-05-04
**Domain:** Chrome/Edge Manifest V3 extension, Native Messaging, Shadow DOM, DPR coordinate calculation
**Confidence:** HIGH (primary claims verified via Context7 + official Chrome docs)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Target element ID and URL match pattern are configurable at runtime — array of `{elementId, urlPattern}` tuples.
- **D-02:** Default config: `elementId="draw-canvas"`, `urlPattern="file:///C:/WacomTest/*.html"`. Defined in a single config file; production values are fill-in-before-ship.
- **D-03:** User-editable via extension options page using `chrome.storage.sync`. `chrome.storage.managed` may override in Phase 4.
- **D-04:** Banner anchored above the target DOM element — positioned relative to the element, not fixed to viewport. Recomputes position on element move/resize.
- **D-05:** Four banner states: not-synced (idle), synced (green), area-changed (warning), host-not-found (error).
- **D-06:** Banner auto-dismisses after "Synced" state. Reappears on area-changed, host-not-found, or not-synced transitions.
- **D-07:** Three staleness detection triggers: ResizeObserver (element resize), `window` resize event (window resize), screenX/Y polling (window move).
- **D-08:** screenX/Y polling interval: 1 second. Polling only runs after a successful sync.

### Claude's Discretion

- MV3 native messaging connection model (`connectNative` persistent port vs. `sendNativeMessage` per-call).
- Debounce/throttle on staleness triggers.
- Page Visibility API integration — pause polling when tab is backgrounded.
- Exact auto-dismiss duration for the "Synced" state.
- Banner CSS styling within the Shadow DOM.
- Options page layout and validation UX.

### Deferred Ideas (OUT OF SCOPE)

- MV3 persistent connection / keep-alive (connectNative with keep-alive pings).
- Multi-monitor support (MULTI-01) — `monitor` field is reserved, not implemented.
- OPS-01 configurable log verbosity — v2 requirement.
- `chrome.storage.managed` for enterprise target config — Phase 4.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| EXT-01 | Content script detects target DOM element by configured ID on page load | MutationObserver wait-for-element pattern; `run_at: "document_idle"` covers most cases but dynamic apps need MutationObserver |
| EXT-02 | Extension calculates element's screen coordinates accounting for Windows DPI scaling | `getBoundingClientRect()` returns CSS pixels; multiply by `window.devicePixelRatio` for physical pixels; `window.screenX/Y` also CSS pixels — multiply by DPR for physical screen offset |
| EXT-03 | Extension sends mapping coordinates to native host when user activates PDF mode | `chrome.runtime.sendNativeMessage("com.brantpoint.wacombridge", msg)` from service worker; content script relays via `chrome.runtime.sendMessage` |
| EXT-04 | Extension shows a Shadow DOM banner when PDF area position or size changes | Shadow host element attached to `document.body`; positioned using `getBoundingClientRect()` of target element; ResizeObserver + resize event + 1s polling |
| EXT-05 | User can re-sync Wacom mapping by clicking banner's calibration button | Re-sends `set_mapping` with freshly computed coordinates; same code path as initial sync |
| EXT-06 | Extension handles native host unavailability gracefully (shows error state in banner) | `chrome.runtime.lastError` or rejected promise on `sendNativeMessage`; also triggered by `ERR_*` error codes in the response body |
</phase_requirements>

---

## Summary

Phase 3 builds a Chrome/Edge Manifest V3 extension composed of three files: `manifest.json` (declares permissions and content script injection), a service worker (`background.js`) that owns the native messaging API calls, and a content script (`content.js`) that observes the DOM, computes coordinates, and renders the Shadow DOM banner. No framework (React, Vue, etc.) is required given the scope — vanilla JS with Shadow DOM is the correct choice.

The critical architectural constraint is that `chrome.runtime.sendNativeMessage` and `chrome.runtime.connectNative` are only available in the **service worker** (background context), not in content scripts. The content script must therefore relay coordinate data to the service worker via `chrome.runtime.sendMessage` / `chrome.runtime.onMessage`. The service worker then calls the native host and returns the result to the content script. This two-hop messaging pattern is mandatory in MV3.

DPR-corrected coordinates are computed as: `physicalX = Math.round((rect.left + window.screenX) * devicePixelRatio)`. Both `getBoundingClientRect()` values and `window.screenX/Y` report CSS (logical) pixels in Chrome on Windows — multiplying all values by `window.devicePixelRatio` is the correct and only required conversion. The host receives physical pixels and uses them verbatim (per `docs/protocol.md` Section 3).

**Primary recommendation:** Use `sendNativeMessage` per call rather than `connectNative` — the extension's interaction model is low-frequency (user-triggered + staleness events), not streaming. `sendNativeMessage` avoids service worker keep-alive complexity and port lifecycle management, and maps cleanly to the request/response protocol in `docs/protocol.md`.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| DOM element detection (EXT-01) | Content Script | — | Only content scripts can access the host page DOM |
| DPR coordinate calculation (EXT-02) | Content Script | — | Requires `window.devicePixelRatio`, `getBoundingClientRect()`, `window.screenX/Y` — only available in content script context |
| Native messaging calls (EXT-03) | Service Worker | — | `chrome.runtime.sendNativeMessage` is only available in service worker/extension pages, not content scripts |
| Coordinate relay content → SW | Content Script + Service Worker | — | `chrome.runtime.sendMessage` (content script) + `chrome.runtime.onMessage` (SW) is the mandatory MV3 bridge |
| Shadow DOM banner UI (EXT-04, EXT-05) | Content Script | — | Banner is injected into the host page DOM — only content scripts can do this |
| Banner state management | Content Script | — | Banner state lives with the observer loop in the content script |
| Error state display (EXT-06) | Content Script | — | Banner renders error state; content script receives error response relayed from SW |
| Staleness detection (D-07, D-08) | Content Script | — | ResizeObserver, resize event, setInterval all run in the content script context |
| User config storage (D-03) | Options Page + Service Worker | Content Script (read) | `chrome.storage.sync.set/get` available in all contexts; options page writes, content script reads on init |

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Chrome Extension MV3 APIs | Built-in (Chrome 88+) | Extension framework, native messaging, storage | The only option for Chrome/Edge extensions |
| Web APIs (Shadow DOM, ResizeObserver, MutationObserver) | Built-in | DOM interaction, UI isolation, element observation | Standard browser APIs — no library needed |

No third-party npm packages are required or recommended. The extension is pure vanilla JavaScript. Adding a bundler (webpack, Vite) or a framework (React, Preact) adds complexity without benefit for a 3-file extension of this scope.

### Supporting (Development Tooling Only)

| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| Node.js / npm | Any LTS | Development-time tooling only if a build step is introduced | Skip entirely if staying with vanilla JS |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `sendNativeMessage` (per-call) | `connectNative` (persistent port) | connectNative keeps the service worker alive but adds port lifecycle management, onDisconnect reconnect logic, and message ID correlation complexity. Not worth it for low-frequency sync events. |
| MutationObserver for element detection | `setInterval` polling | MutationObserver fires in microtask queue (~88x faster than polling, per MDN benchmark). Use MutationObserver to wait for element, then attach observers. |
| Shadow DOM `attachShadow({mode:'open'})` | iframe | Shadow DOM mounts in document tree — `getBoundingClientRect` on the host element gives coordinates directly. iframe needs manual scroll offset arithmetic. Shadow DOM is the correct choice here. |

---

## Architecture Patterns

### System Architecture Diagram

```
[Host Page (PDF viewer)]
    |
    | DOM injection point
    v
[Content Script (content.js)]
    |-- MutationObserver --> detects #draw-canvas (or configured elementId)
    |-- ResizeObserver ----> observes element size changes
    |-- window.resize -----> detects viewport resize
    |-- setInterval(1s) ---> polls window.screenX/Y for window move
    |-- getBoundingClientRect() + screenX/Y * devicePixelRatio --> physicalCoords
    |
    | chrome.runtime.sendMessage({type:'SEND_MAPPING', x,y,w,h})
    v
[Service Worker (background.js)]
    |-- chrome.runtime.onMessage listener
    |-- chrome.runtime.sendNativeMessage("com.brantpoint.wacombridge", cmd)
    |
    | JSON response (or lastError if host not found)
    v
[Native Host (wacom-bridge.exe)] <-- already exists from Phase 2
    |
    v
[Service Worker: relays response back to content script via sendResponse callback]
    |
    v
[Content Script: updates banner state (synced / area-changed / host-not-found)]
    |
    v
[Shadow DOM Banner] -- rendered inside document.body, positioned above target element
```

### Recommended Project Structure

```
extension/
├── manifest.json          # MV3 manifest: permissions, content_scripts, background
├── background.js          # Service worker: native messaging, message relay
├── content.js             # Content script: DOM observer, coord calc, banner
├── options.html           # Options page HTML
├── options.js             # Options page logic: chrome.storage.sync read/write
├── config.js              # Default constants: DEFAULT_ELEMENT_ID, DEFAULT_URL_PATTERN
└── icons/
    ├── icon-16.png
    ├── icon-48.png
    └── icon-128.png
```

`config.js` is shared between `content.js` and `options.js` via ES module import (supported when `manifest.json` `background.type = "module"`) or by inlining constants at build time. Keeping defaults in one file satisfies D-02's fill-in-before-ship requirement.

### Pattern 1: MV3 Manifest Structure

**What:** Minimal MV3 manifest with all required permissions for this extension.
**When to use:** Scaffold (Plan 03-01)

```json
// Source: https://developer.chrome.com/docs/extensions/reference/manifest
{
  "manifest_version": 3,
  "name": "Wacom Bridge",
  "version": "1.0.0",
  "description": "Restricts Wacom stylus to PDF annotation region",
  "permissions": ["nativeMessaging", "storage"],
  "background": {
    "service_worker": "background.js",
    "type": "module"
  },
  "content_scripts": [
    {
      "matches": ["file:///C:/WacomTest/*.html"],
      "js": ["content.js"],
      "run_at": "document_idle"
    }
  ],
  "options_ui": {
    "page": "options.html",
    "open_in_tab": false
  },
  "icons": {
    "16": "icons/icon-16.png",
    "48": "icons/icon-48.png",
    "128": "icons/icon-128.png"
  }
}
```

**Notes on permissions:**
- `"nativeMessaging"` — required for `sendNativeMessage`/`connectNative`. [VERIFIED: Context7 / developer.chrome.com/docs/extensions/reference/api/runtime]
- `"storage"` — required for `chrome.storage.sync` options persistence. [VERIFIED: Context7 / developer.chrome.com/docs/extensions/reference/api/storage]
- `"host_permissions"` for `file:///` — **NOT required** in the manifest as a separate entry when using content_scripts.matches. The match pattern in `content_scripts` is sufficient. [VERIFIED: developer.chrome.com/docs/extensions/develop/concepts/match-patterns]
- The `matches` array in `content_scripts` is populated from `chrome.storage.sync` at injection time only for **dynamically injected** scripts; for statically declared scripts, it is baked into the manifest. The options-page-driven tuples therefore require `chrome.scripting.executeScript` for dynamic injection OR the user must add URL patterns before installing. See Pitfall 3 below.

### Pattern 2: Content Script → Service Worker Relay

**What:** Content script cannot call native messaging directly — must relay through service worker.
**When to use:** Every time the content script needs to send/receive from the native host.

```javascript
// Source: https://developer.chrome.com/docs/extensions/reference/api/runtime

// --- In content.js ---
async function sendMapping(coords) {
  return new Promise((resolve, reject) => {
    chrome.runtime.sendMessage(
      { type: 'SEND_MAPPING', payload: coords },
      (response) => {
        if (chrome.runtime.lastError) {
          reject(chrome.runtime.lastError);
        } else {
          resolve(response);
        }
      }
    );
  });
}

// --- In background.js (service worker) ---
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'SEND_MAPPING') {
    chrome.runtime.sendNativeMessage(
      'com.brantpoint.wacombridge',
      { command: 'set_mapping', ...message.payload }
    ).then(sendResponse)
     .catch(err => sendResponse({ error: err.message, code: 'ERR_HOST_UNAVAILABLE' }));
    return true; // Keep channel open for async response
  }
});
```

**Critical:** `return true` in `onMessage` listener is required to keep the response channel open for the async `sendNativeMessage` call. Without it, `sendResponse` is invalidated before the promise resolves. [VERIFIED: Context7 / developer.chrome.com/docs/extensions/reference/api/runtime]

### Pattern 3: DPR-Corrected Physical Pixel Coordinates

**What:** Convert CSS pixel DOM rect to physical screen pixels for the Wacom driver.
**When to use:** Plan 03-02 — coordinate calculation in content script.

```javascript
// Source: Verified against MDN Web API docs (getBoundingClientRect, devicePixelRatio, screenX)
// Protocol contract: docs/protocol.md Section 3

function getPhysicalCoords(element) {
  const rect = element.getBoundingClientRect(); // CSS pixels, relative to viewport
  const dpr = window.devicePixelRatio;          // e.g., 1.25 at 125%, 1.5 at 150%

  // rect.left/top are viewport-relative CSS pixels.
  // window.screenX/Y are the browser window's position from screen origin, in CSS pixels.
  // Adding them gives the element's CSS position from the physical screen origin.
  // Multiply by DPR to convert CSS pixels -> physical pixels.
  return {
    x:      Math.round((rect.left   + window.screenX) * dpr),
    y:      Math.round((rect.top    + window.screenY) * dpr),
    width:  Math.round(rect.width   * dpr),
    height: Math.round(rect.height  * dpr),
  };
}

// Example: 150% DPI, element at CSS (160, 180), size (960, 360), window at CSS (0, 0)
// -> { x: 240, y: 270, width: 1440, height: 540 }
// Matches protocol.md Section 3 example exactly.
```

**DPR values on Windows:**
- 100% scaling → `devicePixelRatio = 1.0`
- 125% scaling → `devicePixelRatio = 1.25`
- 150% scaling → `devicePixelRatio = 1.5`

**Important caveat:** `window.screenX/Y` return CSS pixels in Chrome (confirmed by MDN). The calculation above is correct. However, there is a known edge case with multi-monitor setups at mixed DPI scaling where the OS coordinate system and the CSS coordinate system can diverge. Since multi-monitor is deferred (MULTI-01), this is not a concern for Phase 3. [CITED: developer.mozilla.org/en-US/docs/Web/API/Window/screenX]

### Pattern 4: Shadow DOM Banner Attachment and Positioning

**What:** Attach a style-isolated banner above the target element.
**When to use:** Plan 03-03 — banner implementation.

```javascript
// Source: MDN Web API docs (attachShadow, getBoundingClientRect)

function createBanner(targetElement) {
  // 1. Create a host element and append to body
  const host = document.createElement('div');
  host.id = 'wacom-bridge-banner-host';
  // Position host absolutely so it doesn't affect page layout
  Object.assign(host.style, {
    position: 'absolute',
    zIndex:   '2147483647',    // max z-index
    pointerEvents: 'auto',
  });
  document.body.appendChild(host);

  // 2. Attach shadow root (closed mode for style isolation)
  const shadow = host.attachShadow({ mode: 'closed' });

  // 3. Inject styles + inner HTML
  const style = document.createElement('style');
  style.textContent = `
    :host { all: initial; font-family: sans-serif; }
    .banner { background: #fff; border: 1px solid #ccc; padding: 6px 12px;
              border-radius: 4px; box-shadow: 0 2px 8px rgba(0,0,0,.15); }
    .banner.synced { border-color: #4caf50; background: #e8f5e9; }
    .banner.area-changed { border-color: #ff9800; background: #fff3e0; }
    .banner.host-not-found { border-color: #f44336; background: #ffebee; }
  `;
  shadow.appendChild(style);

  const banner = document.createElement('div');
  banner.className = 'banner';
  shadow.appendChild(banner);

  // 4. Return update function that repositions the host above the element
  return function updateBanner(state) {
    const rect = targetElement.getBoundingClientRect();
    const scrollY = window.scrollY || window.pageYOffset;
    const scrollX = window.scrollX || window.pageXOffset;

    // Position above the element (document-relative, matching position:absolute on body)
    host.style.top  = `${rect.top  + scrollY - 40}px`; // 40px above element
    host.style.left = `${rect.left + scrollX}px`;

    // Update visual state
    banner.className = `banner ${state}`;
    // ... update inner text/buttons based on state
  };
}
```

**Key details:**
- The host is `position: absolute` on `document.body` — this requires accounting for scroll offsets when computing position. [VERIFIED: MDN getBoundingClientRect]
- Alternatively, use `position: fixed` on the host and use raw `rect.top/left` without scroll offset. Fixed positioning means the banner follows the viewport, not the document — fine if the page doesn't scroll horizontally/vertically while the banner is shown.
- For an element-anchored banner that survives page scroll: absolute + scroll offsets is more correct.
- Banner height is ~40px — subtract that from `rect.top` to anchor above the element.

### Pattern 5: Staleness Detection Loop

**What:** Three-trigger staleness detection with debounce.
**When to use:** Plan 03-03.

```javascript
// Source: MDN (ResizeObserver, addEventListener, setInterval, Page Visibility API)
// [ASSUMED] debounce duration of 300ms — within the CONTEXT.md "Claude's discretion" scope

let lastCoords = null;
let pollInterval = null;
let debounceTimer = null;

function onStale() {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    // Transition banner to area-changed state
    updateBanner('area-changed');
  }, 300); // 300ms debounce prevents rapid flicker during resize drags
}

// Trigger 1: ResizeObserver (element size change)
const ro = new ResizeObserver(onStale);
ro.observe(targetElement);

// Trigger 2: window resize event (window resize changes element's screen position)
window.addEventListener('resize', onStale);

// Trigger 3: screenX/Y polling (window drag/move — no native event exists)
function startPolling() {
  if (pollInterval) return;
  let prevX = window.screenX, prevY = window.screenY;
  pollInterval = setInterval(() => {
    if (document.hidden) return; // Page Visibility API — skip when backgrounded
    if (window.screenX !== prevX || window.screenY !== prevY) {
      prevX = window.screenX;
      prevY = window.screenY;
      onStale();
    }
  }, 1000); // D-08: 1-second interval
}

function stopPolling() {
  clearInterval(pollInterval);
  pollInterval = null;
}

// Start polling only after successful sync (D-08)
// Pause/resume on tab visibility
document.addEventListener('visibilitychange', () => {
  if (document.hidden) {
    stopPolling();
  } else if (lastCoords !== null) {
    startPolling(); // Resume only if previously synced
  }
});
```

### Anti-Patterns to Avoid

- **Using `connectNative` with port keep-alive in service worker:** Adds onDisconnect reconnect logic and message correlation overhead for a low-frequency call pattern. Use `sendNativeMessage` instead.
- **Calling `chrome.runtime.sendNativeMessage` from the content script:** This throws a runtime error — native messaging is not available in content scripts. Always relay through the service worker.
- **Hardcoding a single elementId:** The options page writes an array of tuples; the content script must read from `chrome.storage.sync` on init, not use a hardcoded constant.
- **Using `position: fixed` for the banner without handling page scroll:** Fixed anchors to the viewport, not the element — works if the element is always visible, breaks if the page has scroll that shifts the element off-screen.
- **Not returning `true` from `onMessage` for async response:** Without `return true`, the `sendResponse` callback is invalidated before the async native messaging call resolves, causing silent failures.
- **Using `window.getScreenDetails()` without feature detection:** `getScreenDetails()` is part of the Window Management API and requires the `window-management` permission plus a user permission prompt. `window.screenX/Y * devicePixelRatio` achieves EXT-02 without extra permissions.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CSS isolation for banner | Custom CSS namespace/prefixing | Shadow DOM `attachShadow()` | Built-in, guaranteed isolation from host page styles |
| Element resize detection | `setInterval` polling element dimensions | `ResizeObserver` | Browser-native, fires on microtask queue, ~88x faster than polling |
| Wait for dynamic element | `setInterval` polling `getElementById` | `MutationObserver` + initial check | Fires on microtask queue; polling may miss elements or fire thousands of times |
| Multi-hop message correlation | Custom request ID + response map | Simple `sendResponse` callback (Promise-based `sendNativeMessage`) | Chrome's built-in one-shot response callback handles this |
| DPR conversion library | Custom scaling math library | `window.devicePixelRatio` × CSS values | Native browser property; no library needed |

**Key insight:** The Chrome Extension API and Web Platform APIs cover all non-trivial problems in this phase. Adding any third-party dependency would increase bundle size and maintenance burden without solving anything the native APIs don't already solve.

---

## Runtime State Inventory

This is a greenfield phase — no existing extension code to rename or migrate.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — no extension installed yet | None |
| Live service config | `installer/manifest-chrome.json` and `installer/manifest-edge.json` contain PLACEHOLDER extension IDs | Phase 3 must replace PLACEHOLDER_CHROME_EXTENSION_ID and PLACEHOLDER_EDGE_EXTENSION_ID after sideloading |
| OS-registered state | Native host registered under `com.brantpoint.wacombridge` (HKLM) by Phase 2 MSI | No rename needed — host name is correct |
| Secrets/env vars | None | None |
| Build artifacts | None | None |

---

## Common Pitfalls

### Pitfall 1: file:// URL Access Requires Manual User Action

**What goes wrong:** The extension is sideloaded, the manifest declares `"file:///C:/WacomTest/*.html"` in `content_scripts.matches`, but the content script never injects.
**Why it happens:** Chrome requires users to manually enable "Allow access to file URLs" on the extension's details page at `chrome://extensions/`. This setting is **not** automatically granted by the manifest declaration. Since Chrome 118, this is explicitly required.
**How to avoid:** After sideloading, open `chrome://extensions/`, click "Details" on the extension, and toggle "Allow access to file URLs". For GPO-managed installs (Phase 4), the `file_url_navigation_allowed` field in `ExtensionSettings` policy can enable this — but this is a Phase 4 concern. **Include this in the sideloading development checklist.**
**Warning signs:** `chrome://extensions/` shows the extension as active but no banner appears on the test HTML page.

### Pitfall 2: `sendResponse` Invalidated Before Async Native Messaging Resolves

**What goes wrong:** The service worker receives the relay message from the content script, calls `sendNativeMessage`, but the content script never receives the response (or receives `undefined`).
**Why it happens:** `chrome.runtime.onMessage` listeners must `return true` to indicate an asynchronous response. Without it, Chrome closes the message channel when the synchronous listener returns, invalidating `sendResponse` before the `sendNativeMessage` promise resolves.
**How to avoid:** Always `return true` from `onMessage` handlers that call `sendResponse` asynchronously.
**Warning signs:** `chrome.runtime.lastError: "The message port closed before a response was received."` in the content script.

### Pitfall 3: Extension ID Placeholder in Native Messaging Manifests

**What goes wrong:** The native host (`com.brantpoint.wacombridge`) is registered but rejects the connection with "Access denied" or the connection silently fails.
**Why it happens:** `installer/manifest-chrome.json` and `installer/manifest-edge.json` contain `PLACEHOLDER_CHROME_EXTENSION_ID` / `PLACEHOLDER_EDGE_EXTENSION_ID`. The native messaging host validates the calling extension's origin against `allowed_origins`. If the real sideloaded ID is not in that list, the connection is rejected.
**How to avoid:** 
1. Sideload the extension via `chrome://extensions/` → "Load unpacked" → note the assigned extension ID (visible under the extension name).
2. To get a **stable development ID**: use "Pack extension" in Chrome to generate a `.pem` key file, add `"key": "<base64-pem-content>"` to `manifest.json`, and the ID will be consistent across reloads.
3. Update `installer/manifest-chrome.json` and `installer/manifest-edge.json` with the real IDs and reinstall the native messaging registration (re-run the MSI or manually update the registry-pointed JSON file).
**Warning signs:** `chrome.runtime.lastError: "Specified native messaging host not found."` or "Native messaging host com.brantpoint.wacombridge is not registered."

### Pitfall 4: Static `content_scripts` Matches Cannot Be Driven by `chrome.storage.sync`

**What goes wrong:** User adds a new `{elementId, urlPattern}` tuple in the options page, but the content script never injects on the new URL.
**Why it happens:** Static `content_scripts` in `manifest.json` are baked at extension load time. They cannot be changed at runtime by writing to `chrome.storage.sync`. Only `chrome.scripting.executeScript` (requires `"scripting"` permission + `"host_permissions"`) can inject scripts dynamically into arbitrary URLs.
**How to avoid:** For the default test URL (`file:///C:/WacomTest/*.html`), static declaration in `manifest.json` is sufficient. For user-added tuples: either (a) require the extension to be reloaded after options changes (simple, acceptable for enterprise deployment), or (b) add `"scripting"` permission and use `chrome.scripting.registerContentScripts()` for dynamic URL pattern registration. **Decision is Claude's discretion — recommend option (a) for Phase 3 simplicity.**
**Warning signs:** The content script injects on the default test URL but not on any URL added through the options page.

### Pitfall 5: `window.screenX/Y` Returns CSS Pixels, Not Physical Pixels

**What goes wrong:** Coordinates sent to the native host are correct at 100% DPI but wrong at 125% or 150% — the stylus maps to a scaled-but-not-correctly-positioned area.
**Why it happens:** Developers assume `window.screenX` returns physical pixels (since it's a "screen" property). It returns CSS pixels. Only multiplying ALL values (both `getBoundingClientRect()` and `screenX/Y`) by `devicePixelRatio` gives physical pixels.
**How to avoid:** The formula is: `physicalX = Math.round((rect.left + window.screenX) * devicePixelRatio)`. Apply to all four values: x, y, width, height.
**Warning signs:** At 100% DPI the stylus maps correctly; at 150% DPI the stylus maps to an area that is offset or incorrectly sized.

### Pitfall 6: Service Worker Termination Between User Actions

**What goes wrong:** User loads the page, banner appears. User walks away for >30 seconds, returns, clicks "Sync Now" — nothing happens.
**Why it happens:** The MV3 service worker terminates after 30 seconds of inactivity. When the content script calls `chrome.runtime.sendMessage`, Chrome will wake the service worker, but if the wake fails for any reason, the message is lost.
**How to avoid:** The service worker is event-driven — Chrome automatically restarts it on incoming messages. The standard `chrome.runtime.sendMessage` → `onMessage` pattern handles this correctly; Chrome wakes the SW before delivering the message. No manual keep-alive is needed for this use case.
**Warning signs:** Intermittent failures on the first sync after the page has been idle — add error handling in the content script to retry once on `chrome.runtime.lastError`.

---

## Code Examples

### Complete manifest.json

```json
// Source: Context7 / developer.chrome.com/docs/extensions/reference/manifest
{
  "manifest_version": 3,
  "name": "Wacom Bridge",
  "version": "1.0.0",
  "description": "Restricts Wacom stylus to PDF annotation region",
  "permissions": ["nativeMessaging", "storage"],
  "background": {
    "service_worker": "background.js",
    "type": "module"
  },
  "content_scripts": [
    {
      "matches": ["file:///C:/WacomTest/*.html"],
      "js": ["content.js"],
      "run_at": "document_idle"
    }
  ],
  "options_ui": {
    "page": "options.html",
    "open_in_tab": false
  },
  "icons": {
    "16": "icons/icon-16.png",
    "48": "icons/icon-48.png",
    "128": "icons/icon-128.png"
  }
}
```

### Service Worker (background.js) — Native Messaging Relay

```javascript
// Source: Context7 / developer.chrome.com/docs/extensions/reference/api/runtime
// Pattern: sendNativeMessage per call — no persistent port

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'NATIVE_COMMAND') {
    chrome.runtime.sendNativeMessage(
      'com.brantpoint.wacombridge',
      message.payload
    ).then(response => {
      sendResponse({ ok: true, data: response });
    }).catch(err => {
      // Host not installed or not responding
      sendResponse({ ok: false, error: err.message });
    });
    return true; // REQUIRED: keep channel open for async response
  }
});
```

### Content Script Element Detection with MutationObserver

```javascript
// Source: MDN MutationObserver docs

function waitForElement(id, callback) {
  const el = document.getElementById(id);
  if (el) { callback(el); return; }

  const observer = new MutationObserver(() => {
    const el = document.getElementById(id);
    if (el) {
      observer.disconnect();
      callback(el);
    }
  });
  observer.observe(document.body, { childList: true, subtree: true });
}
```

### chrome.storage.sync Read/Write for Options Tuples

```javascript
// Source: Context7 / developer.chrome.com/docs/extensions/reference/api/storage

const DEFAULT_TARGETS = [
  { elementId: 'draw-canvas', urlPattern: 'file:///C:/WacomTest/*.html' }
];

// Read
async function getTargets() {
  const result = await chrome.storage.sync.get({ targets: DEFAULT_TARGETS });
  return result.targets;
}

// Write (options page)
async function saveTargets(targets) {
  await chrome.storage.sync.set({ targets });
}

// React to options changes in content script
chrome.storage.onChanged.addListener((changes, area) => {
  if (area === 'sync' && changes.targets) {
    // Re-initialize with new targets
    initExtension(changes.targets.newValue);
  }
});
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| MV2 background page (persistent) | MV3 service worker (ephemeral) | Chrome 88 (MV3 GA) | Service worker can terminate; native messaging via port keeps it alive |
| `background.scripts` array | `background.service_worker` string | MV3 | Single file, must be a service worker |
| `browser_action` / `page_action` | `action` | MV3 | Unified action key |
| Inline event handlers in content scripts | `addEventListener` only | Content Security Policy enforcement | Direct `onclick=...` in injected HTML blocked by CSP |
| `connectNative` requiring persistent background page | `sendNativeMessage` per-call or `connectNative` with SW keep-alive | MV3 / Chrome 105 | Both work; per-call is simpler for low-frequency use |

**Deprecated/outdated:**
- MV2 `background.persistent: true`: Not available in MV3. Background pages no longer exist.
- `chrome.extension.sendRequest`: Removed in Chrome 33. Use `chrome.runtime.sendMessage`.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | 300ms debounce duration on staleness triggers | Pattern 5 (Staleness Detection) | Too short: banner flickers during slow resize drags. Too long: banner feels sluggish. Tunable — low risk. |
| A2 | `window.screenX/Y * devicePixelRatio` gives correct physical pixel window position on Windows at all DPI scales (100%, 125%, 150%) | Pattern 3 (DPR Coordinate Calc) | If Chrome on Windows reports screenX differently at some DPI levels, coordinate offset could be wrong. The protocol.md example (Section 3) confirms the formula is correct for 150%. Not independently verified on a 125% Windows display in this session. |
| A3 | Auto-dismiss duration for "Synced" state is 3 seconds (CONTEXT.md says "Claude's discretion") | Banner behavior | If too short, user doesn't see confirmation. If too long, interferes with annotation workflow. 3s is a conventional notification pattern. |
| A4 | Static `content_scripts` approach is sufficient for Phase 3 (no `chrome.scripting` dynamic injection) | Pitfall 4 | If options-page-added URL patterns must work without extension reload, dynamic injection via `chrome.scripting.registerContentScripts` would be needed. For enterprise deployment with IT-managed config, static + reload is acceptable. |
| A5 | `run_at: "document_idle"` is sufficient for detecting the target element | Pattern 1 / EXT-01 | If the PDF viewer renders `#draw-canvas` lazily after DOMContentLoaded (e.g., via a framework), `document_idle` won't be enough. MutationObserver fallback (Pattern — Element Detection) handles this. Both should be implemented. |

---

## Open Questions

1. **Does the third-party PDF viewer render `#draw-canvas` synchronously or via a JS framework?**
   - What we know: Default test URL is `file:///C:/WacomTest/*.html` with `elementId="draw-canvas"`. This appears to be a test harness file the team controls.
   - What's unclear: Whether the element exists at `document_idle` or is dynamically inserted.
   - Recommendation: Implement both `document.getElementById` on load AND `MutationObserver` fallback — the MutationObserver pattern is essentially free and eliminates the question.

2. **Should options-page-added URL patterns work without a reload?**
   - What we know: Static `content_scripts` can't be changed at runtime (Pitfall 4). Dynamic injection via `chrome.scripting.registerContentScripts` is available in MV3.
   - What's unclear: Whether the deployment model (GPO/Intune) can handle extension reloads, or whether the user experience requires immediate activation.
   - Recommendation: For Phase 3, implement static manifest only (default URL). Document as a known limitation — Phase 4 can extend with `chrome.scripting` if needed.

3. **Does `window.screenX * devicePixelRatio` give the correct physical pixel offset on Windows when Chrome is maximized on a 125% DPI display?**
   - What we know: The formula is confirmed correct for the 150% example in `docs/protocol.md` Section 3. MDN confirms screenX returns CSS pixels.
   - What's unclear: Whether Windows DPI virtualization introduces any edge-case rounding at 125% (fractional DPR) that causes off-by-one errors for the Wacom driver.
   - Recommendation: Add a `ping` + `get_status` round-trip test in the extension that verifies the Wacom mapping visually at 125% DPI before shipping.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Chrome / Edge (Windows) | Extension testing, native messaging | Unknown (dev machine is Linux) | — | Test on Windows VM; the extension itself runs on the target Windows machine |
| Node.js | Optional build tooling | Unknown | — | Not needed if staying with vanilla JS (recommended) |
| Native host (`wacom-bridge.exe`) | EXT-03, EXT-06 integration test | Phase 2 artifact — not yet shipped | — | Use stub/mock service worker response during Phase 3 development |

**Missing dependencies with fallback:**
- `wacom-bridge.exe`: Phase 3 can be developed and tested for banner states and coordinate math independently. Service worker can be given a mock mode that returns `{"ok": true}` without a live native host. EXT-06 "host not found" state can be tested by deliberately pointing to a non-existent host name.
- Windows-specific test environment: The extension code is plain JS with no Windows-specific dependencies. Banner rendering and coordinate math can be verified in Chrome on Linux using `file://` URLs; DPR scaling at 125%/150% requires a Windows machine for final validation.

---

## Security Domain

> `security_enforcement` is not explicitly set to false in config.json — treating as enabled.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | N/A — extension communicates only with a locally registered native host, no remote auth |
| V3 Session Management | No | No sessions — stateless request/response per user action |
| V4 Access Control | Yes (limited) | Chrome's `allowed_origins` in native messaging manifest restricts which extension IDs can connect to the host |
| V5 Input Validation | Yes | `chrome.storage.sync` values from the options page must be validated before use as DOM selectors or URL patterns — no `eval`, no `innerHTML` with user-provided content |
| V6 Cryptography | No | No secrets transmitted; all communication is local (content script → service worker → local native process) |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malicious web page injecting into extension's content script context | Tampering | Shadow DOM (closed mode) + no global variables exposed; content script runs in ISOLATED world by default in MV3 |
| XSS via user-provided elementId in options page | Tampering | Never use `innerHTML` with storage values; use `textContent` or `createElement` for DOM construction |
| Content script injecting into unintended pages | Information Disclosure | Tight `content_scripts.matches` — default is `file:///C:/WacomTest/*.html` only; user-added patterns must use `chrome.storage.sync` validation to reject obviously malicious patterns |
| Native host connection spoofing | Spoofing | Chrome enforces `allowed_origins` — only the registered extension ID can connect; no additional mitigation needed |

**Shadow DOM mode choice:** Use `mode: 'closed'` for the banner's shadow root. Closed mode prevents the host page's JavaScript from accessing the shadow root via `element.shadowRoot`, reducing the attack surface for page scripts that might try to manipulate the banner.

---

## Sources

### Primary (HIGH confidence)
- Context7 library `/websites/developer_chrome_extensions_reference_api` — native messaging, runtime.sendMessage, runtime.onMessage, storage API
- Context7 library `/websites/developer_chrome_extensions_reference_manifest` — manifest.json structure, background service_worker, content_scripts, options_ui
- [developer.chrome.com/docs/extensions/develop/concepts/native-messaging](https://developer.chrome.com/docs/extensions/develop/concepts/native-messaging) — connectNative vs sendNativeMessage, host lifecycle, allowed_origins
- [developer.chrome.com/docs/extensions/develop/concepts/service-workers/lifecycle](https://developer.chrome.com/docs/extensions/develop/concepts/service-workers/lifecycle) — service worker termination rules, 30s/5min limits, native messaging keep-alive
- [developer.chrome.com/docs/extensions/develop/concepts/match-patterns](https://developer.chrome.com/docs/extensions/develop/concepts/match-patterns) — file:// pattern syntax, manual permission requirement
- [developer.mozilla.org/en-US/docs/Web/API/Window/devicePixelRatio](https://developer.mozilla.org/en-US/docs/Web/API/Window/devicePixelRatio) — DPR returns ratio of physical to CSS pixels
- [developer.mozilla.org/en-US/docs/Web/API/Window/screenX](https://developer.mozilla.org/en-US/docs/Web/API/Window/screenX) — screenX returns CSS pixels
- [developer.mozilla.org/en-US/docs/Web/API/Element/getBoundingClientRect](https://developer.mozilla.org/en-US/docs/Web/API/Element/getBoundingClientRect) — returns CSS pixels relative to viewport
- [developer.chrome.com/blog/longer-esw-lifetimes](https://developer.chrome.com/blog/longer-esw-lifetimes) — service worker lifetime improvements (Chrome 110+)

### Secondary (MEDIUM confidence)
- [support.google.com — Allow access to file URLs via GPO](https://support.google.com/chrome/a/thread/299815728) — `file_url_navigation_allowed` in ExtensionSettings policy (Chrome 119+)
- [MDN MutationObserver](https://developer.mozilla.org/en-US/docs/Web/API/MutationObserver) — microtask queue firing, childList + subtree options

### Tertiary (LOW confidence — flagged in Assumptions Log)
- A2: screenX * DPR correctness on Windows at 125% DPI — inferred from MDN + protocol.md example, not independently verified on physical 125% Windows display

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — MV3 APIs are stable, verified via Context7 + official docs
- Architecture: HIGH — content script / service worker split is a mandatory MV3 constraint confirmed by official docs
- DPR coordinate calculation: HIGH for formula; MEDIUM for edge cases at 125% DPI on Windows (see A2)
- Shadow DOM patterns: HIGH — standard web platform feature
- Pitfalls: HIGH (Pitfalls 1-4 verified from official docs); MEDIUM (Pitfall 5-6 inferred from lifecycle docs)
- file:// URL permission requirement: HIGH — confirmed by official match-patterns docs

**Research date:** 2026-05-04
**Valid until:** 2026-08-04 (Chrome Extension APIs are stable; MV3 is current standard)
