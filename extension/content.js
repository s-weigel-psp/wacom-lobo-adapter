// content.js — MV3 content script.
// Injected into pages matching manifest.json content_scripts.matches.
// Responsibilities: DOM element detection, coordinate calculation (Plan 03-02),
// staleness detection (Plan 03-02), banner rendering (Plan 03-03 via banner.js).

import {
  DEFAULT_TARGETS,
  HOST_NAME,       // used indirectly via sendNativeCommand
  POLL_INTERVAL_MS,
  DEBOUNCE_MS,
  AUTO_DISMISS_MS,
} from './config.js';

import { createBanner } from './banner.js';

// ---------------------------------------------------------------------------
// Module-level state — persists for the lifetime of the content script.
// ---------------------------------------------------------------------------

let hasPriorSync   = false;   // true after first successful set_mapping (D-08)
let debounceTimer  = null;    // used by onStale debounce
let pollInterval   = null;    // screenX/Y poll interval handle

// Banner update callback — set by initObservers(). Replaced in main() when banner is created.
let updateBannerState = (state) => {
  // Temporary console stub — replaced with Shadow DOM banner update in main().
  console.log('[WacomBridge] banner state:', state);
};

// ---------------------------------------------------------------------------
// Native messaging relay — content scripts cannot call sendNativeMessage directly.
// Relay through the service worker via chrome.runtime.sendMessage.
// ---------------------------------------------------------------------------

/**
 * Send a command to the native host via the service worker relay.
 * @param {Object} payload - Protocol command object (see docs/protocol.md Section 2)
 * @returns {Promise<{ok: boolean, data?: Object, error?: string, code?: string}>}
 */
function sendNativeCommand(payload) {
  return new Promise((resolve, reject) => {
    chrome.runtime.sendMessage(
      { type: 'NATIVE_COMMAND', payload },
      (response) => {
        if (chrome.runtime.lastError) {
          // Service worker wake failure — treat as host unavailable.
          reject(new Error(chrome.runtime.lastError.message));
        } else {
          resolve(response);
        }
      }
    );
  });
}

// ---------------------------------------------------------------------------
// Storage — read target tuples from chrome.storage.sync (D-01, D-03)
// ---------------------------------------------------------------------------

/**
 * Returns the array of {elementId, urlPattern} tuples from storage.
 * Falls back to DEFAULT_TARGETS if storage is empty.
 * @returns {Promise<Array<{elementId: string, urlPattern: string}>>}
 */
async function getTargets() {
  const result = await chrome.storage.sync.get({ targets: DEFAULT_TARGETS });
  return result.targets;
}

// ---------------------------------------------------------------------------
// Element detection — MutationObserver + initial getElementById check (EXT-01)
// ---------------------------------------------------------------------------

/**
 * Wait for a DOM element with the given ID to appear, then invoke callback.
 * Uses MutationObserver as a fallback for lazily-rendered elements (RESEARCH.md).
 * @param {string} id
 * @param {function(Element): void} callback
 */
function waitForElement(id, callback) {
  const el = document.getElementById(id);
  if (el) {
    callback(el);
    return;
  }
  const observer = new MutationObserver(() => {
    const found = document.getElementById(id);
    if (found) {
      observer.disconnect();
      callback(found);
    }
  });
  observer.observe(document.body, { childList: true, subtree: true });
}

// ---------------------------------------------------------------------------
// Coordinate calculation — Plan 03-02 (EXT-02, docs/protocol.md Section 3)
// ---------------------------------------------------------------------------

/**
 * Convert the target element's CSS pixel bounding rect to physical screen pixels.
 * Formula from docs/protocol.md Section 3 and RESEARCH.md Pattern 3.
 *
 * Both getBoundingClientRect() values and window.screenX/Y are CSS pixels in Chrome.
 * window.screenX/Y give the browser window's position from the screen origin (CSS px).
 * Adding them gives the element's CSS distance from the screen origin.
 * Multiplying by devicePixelRatio converts to physical pixels.
 *
 * @param {Element} element
 * @returns {{ x: number, y: number, width: number, height: number }} Physical pixel coords
 */
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

// ---------------------------------------------------------------------------
// Staleness detection (D-07, D-08, RESEARCH.md Pattern 5)
// ---------------------------------------------------------------------------

/**
 * Called by any staleness trigger. Debounces to prevent rapid transitions
 * during active window-resize drags (RESEARCH.md A1, UI-SPEC.md).
 * Transitions banner to area-changed ONLY if a prior sync exists (D-08).
 */
function onStale() {
  if (!hasPriorSync) return; // No point marking stale if never synced.
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    updateBannerState('area-changed');
  }, DEBOUNCE_MS);
}

/**
 * Start the screenX/Y poll. Only call after successful sync (D-08).
 * Poll pauses when document.hidden (Page Visibility API, RESEARCH.md Pattern 5).
 */
function startPolling() {
  if (pollInterval) return; // Already running.
  let prevX = window.screenX;
  let prevY = window.screenY;
  pollInterval = setInterval(() => {
    if (document.hidden) return; // Paused — tab backgrounded.
    if (window.screenX !== prevX || window.screenY !== prevY) {
      prevX = window.screenX;
      prevY = window.screenY;
      onStale();
    }
  }, POLL_INTERVAL_MS);
}

function stopPolling() {
  clearInterval(pollInterval);
  pollInterval = null;
}

// ---------------------------------------------------------------------------
// Sync action — sends set_mapping to native host (EXT-03, EXT-05)
// ---------------------------------------------------------------------------

/**
 * Compute current physical coordinates and send set_mapping to the native host.
 * Called on: "Sync now" click (idle→synced), "Re-calibrate" click (area-changed→synced).
 * @param {Element} element - The target DOM element to map the stylus to.
 */
async function syncMapping(element) {
  const coords = getPhysicalCoords(element);
  const payload = {
    command: 'set_mapping',
    x:       coords.x,
    y:       coords.y,
    width:   coords.width,
    height:  coords.height,
  };

  updateBannerState('pending'); // Disable button while request is in-flight (UI-SPEC.md).

  let response;
  try {
    response = await sendNativeCommand(payload);
  } catch (err) {
    // Service worker wake failure — treat as host unavailable (RESEARCH.md Pitfall 6).
    updateBannerState('host-not-found');
    return;
  }

  // Check both the relay wrapper (response.ok) and the protocol error shape (response.data.code).
  const isError = !response.ok || (response.data && response.data.code && response.data.code.startsWith('ERR_'));

  if (isError) {
    // Any ERR_ code maps to host-not-found banner state (EXT-06, 03-PATTERNS.md).
    updateBannerState('host-not-found');
  } else {
    hasPriorSync = true;
    updateBannerState('synced');
    startPolling(); // Begin screenX/Y polling now that we have a successful sync (D-08).
  }
}

// ---------------------------------------------------------------------------
// Observer initialization — attach all three staleness triggers (D-07)
// ---------------------------------------------------------------------------

/**
 * Attach ResizeObserver, window resize listener, and Page Visibility handler.
 * Call once per target element after it is found in the DOM.
 * @param {Element} element
 */
function initObservers(element) {
  // Trigger 1: element size change (browser zoom, panel resize)
  const ro = new ResizeObserver(onStale);
  ro.observe(element);

  // Trigger 2: window resize (shifts element's screen position)
  window.addEventListener('resize', onStale);

  // Trigger 3: tab visibility — pause/resume screenX/Y polling
  document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
      stopPolling();
    } else if (hasPriorSync) {
      startPolling(); // Resume only if previously synced (D-08).
    }
  });

  // Trigger 3 polling itself is started in syncMapping() after first successful sync.
}

// ---------------------------------------------------------------------------
// Bootstrap
// ---------------------------------------------------------------------------

async function main() {
  const targets = await getTargets();

  // React to storage changes from the options page (D-03)
  chrome.storage.onChanged.addListener((changes, area) => {
    if (area === 'sync' && changes.targets) {
      // Re-initialize with updated targets (page reload required for new URL patterns)
      main();
    }
  });

  // Find the first matching target for the current page URL.
  const pageUrl = window.location.href;
  const matchingTarget = targets.find(t => {
    // Simple glob match: replace * with .* for URL pattern matching.
    const pattern = t.urlPattern.replace(/[.+?^${}()|[\]\\]/g, '\\$&').replace(/\*/g, '.*');
    return new RegExp(`^${pattern}$`).test(pageUrl);
  });

  if (!matchingTarget) return; // No configured target matches this page.

  waitForElement(matchingTarget.elementId, (element) => {
    // Initialize all staleness observers for this element.
    initObservers(element);

    // Create the Shadow DOM banner, wiring "Sync now" / "Re-calibrate" to syncMapping.
    const banner = createBanner(element, () => syncMapping(element));

    // Replace the console stub with the real banner update function.
    updateBannerState = banner.update;

    // Show idle state — user must click "Sync now" for first sync (D-05).
    updateBannerState('idle');
  });
}

main();
