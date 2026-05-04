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
// Bootstrap — Plan 03-02 replaces this stub body with full implementation.
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
  // Full staleness detection and banner initialization added in Plan 03-02.
  const pageUrl = window.location.href;
  const matchingTarget = targets.find(t => {
    // Simple glob match: replace * with .* for URL pattern matching.
    const pattern = t.urlPattern.replace(/[.+?^${}()|[\]\\]/g, '\\$&').replace(/\*/g, '.*');
    return new RegExp(`^${pattern}$`).test(pageUrl);
  });

  if (!matchingTarget) return; // No configured target matches this page.

  waitForElement(matchingTarget.elementId, (element) => {
    // Plan 03-02 adds: getPhysicalCoords(element), staleness detection loop,
    // banner initialization via banner.js createBanner(element).
    console.log('[WacomBridge] Target element found:', matchingTarget.elementId);
  });
}

main();
