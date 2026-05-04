// background.js — MV3 service worker.
// Owns chrome.runtime.sendNativeMessage (not available in content scripts).
// Receives NATIVE_COMMAND messages from content.js, calls the native host,
// and relays the response back via sendResponse.

import { HOST_NAME } from './config.js';

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'NATIVE_COMMAND') {
    chrome.runtime.sendNativeMessage(HOST_NAME, message.payload)
      .then(response => {
        sendResponse({ ok: true, data: response });
      })
      .catch(err => {
        // Normalize chrome.runtime.lastError (host not installed) into the
        // same error shape as protocol.md Section 4 error responses.
        sendResponse({ ok: false, error: err.message, code: 'ERR_HOST_UNAVAILABLE' });
      });
    // REQUIRED: return true keeps the message channel open for the async sendResponse.
    // Without this, Chrome invalidates sendResponse before the Promise resolves.
    // See RESEARCH.md Pitfall 2.
    return true;
  }
});
