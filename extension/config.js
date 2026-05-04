// config.js — single source of truth for extension constants.
// FILL-IN-BEFORE-SHIP: Replace DEFAULT_TARGETS values with production
// element ID and URL pattern before deploying to end users (D-02).

export const DEFAULT_TARGETS = [
  {
    elementId:  'draw-canvas',
    urlPattern: 'file:///C:/WacomTest/*.html',
  },
];

// Must match installer/manifest-chrome.json and installer/manifest-edge.json "name" field exactly.
export const HOST_NAME        = 'com.brantpoint.wacombridge';

export const POLL_INTERVAL_MS = 1000;  // D-08: screenX/Y poll interval after successful sync
export const DEBOUNCE_MS      = 300;   // RESEARCH.md A1: staleness trigger debounce
export const AUTO_DISMISS_MS  = 3000;  // RESEARCH.md A3: synced banner auto-dismiss
