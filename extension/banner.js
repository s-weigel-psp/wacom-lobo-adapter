// banner.js — Shadow DOM banner component.
// Exported function: createBanner(targetElement) → { update(state) }
// Shadow root mode: 'closed' (prevents host-page JS from accessing shadowRoot).
// CSS: VERBATIM from UI-SPEC.md "Shadow DOM CSS — Complete Declaration". Do not modify.

import { AUTO_DISMISS_MS } from './config.js';

// VERBATIM from UI-SPEC.md lines 363–442. Do not modify any value.
const BANNER_CSS = `
:host {
  all: initial;
  font-family: system-ui, -apple-system, sans-serif;
  font-size: 14px;
  line-height: 1.4;
}

.banner {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 40px;
  padding: 0 12px;
  background: #ffffff;
  border: 1px solid #cccccc;
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  white-space: nowrap;
  color: #333333;
}

.banner.synced {
  border-color: #4caf50;
  background: #e8f5e9;
  color: #1b5e20;
}

.banner.area-changed {
  border-color: #ff9800;
  background: #fff3e0;
  color: #e65100;
}

.banner.host-not-found {
  border-color: #f44336;
  background: #ffebee;
  color: #b71c1c;
}

.banner-label {
  flex: 1;
}

.banner-hint {
  font-size: 12px;
  font-weight: 400;
  opacity: 0.85;
  flex: 1;
}

.banner-action {
  height: 32px;
  padding: 0 12px;
  background: #1976d2;
  color: #ffffff;
  border: none;
  border-radius: 4px;
  font-family: inherit;
  font-size: 14px;
  font-weight: 600;
  line-height: 1;
  cursor: pointer;
  flex-shrink: 0;
}

.banner-action:hover {
  background: #1565c0;
}

.banner-action:focus-visible {
  outline: 2px solid #1976d2;
  outline-offset: 2px;
}

.banner-action:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
`;

/**
 * Create a Shadow DOM banner anchored above targetElement.
 * Returns an update(state) function.
 *
 * @param {Element} targetElement - The DOM element the banner tracks.
 * @param {function(): void} onSyncClickCallback - Called when the user clicks
 *   "Sync now" or "Re-calibrate". Provided by content.js (syncMapping).
 * @returns {{ update: function(string): void }}
 */
export function createBanner(targetElement, onSyncClickCallback) {
  // 1. Create the host element and append to document.body.
  const host = document.createElement('div');
  host.id = 'wacom-bridge-banner-host';
  Object.assign(host.style, {
    position:      'absolute',
    zIndex:        '2147483647',  // max z-index — always on top
    pointerEvents: 'auto',
    minWidth:      '220px',
  });
  document.body.appendChild(host);

  // 2. Attach shadow root in closed mode (security: host page JS cannot access shadowRoot).
  const shadow = host.attachShadow({ mode: 'closed' });

  // 3. Inject styles from the verbatim BANNER_CSS constant above.
  const styleEl = document.createElement('style');
  styleEl.textContent = BANNER_CSS;
  shadow.appendChild(styleEl);

  // 4. Create the inner banner div with ARIA live region (UI-SPEC.md Accessibility Contract).
  const banner = document.createElement('div');
  banner.setAttribute('role', 'status');
  banner.setAttribute('aria-live', 'polite');
  banner.className = 'banner idle';
  shadow.appendChild(banner);

  let dismissTimer = null;

  /**
   * Reposition the host above the target element.
   * Must account for page scroll since host is position:absolute on body.
   */
  function reposition() {
    const rect    = targetElement.getBoundingClientRect();
    const scrollY = window.scrollY || window.pageYOffset;
    const scrollX = window.scrollX || window.pageXOffset;
    host.style.top  = `${rect.top  + scrollY - 40}px`; // 40px = banner height (UI-SPEC.md)
    host.style.left = `${rect.left + scrollX}px`;
  }

  /**
   * Update the banner to reflect the new state.
   * Repositions the host on every call (element may have moved).
   *
   * @param {'idle'|'synced'|'area-changed'|'host-not-found'|'pending'} state
   */
  function update(state) {
    // Clear any pending dismiss timer when transitioning states.
    clearTimeout(dismissTimer);

    // Restore visibility if previously hidden by synced auto-dismiss.
    if (state !== 'synced') {
      host.style.display = '';
    }

    if (state === 'pending') {
      // Disable action button only — do not change banner class or label text.
      const btn = banner.querySelector('.banner-action');
      if (btn) btn.disabled = true;
      reposition();
      return;
    }

    // Build banner inner content based on state.
    // Use createElement + textContent (never innerHTML) to prevent XSS.
    banner.innerHTML = ''; // Clear previous content.
    banner.className = `banner ${state}`;

    const label = document.createElement('span');
    label.className = 'banner-label';

    switch (state) {
      case 'idle': {
        label.textContent = 'Wacom not synced';
        const btn = document.createElement('button');
        btn.className = 'banner-action';
        btn.textContent = 'Sync now';
        btn.addEventListener('click', () => {
          if (onSyncClickCallback) onSyncClickCallback();
        });
        banner.appendChild(label);
        banner.appendChild(btn);
        break;
      }

      case 'synced': {
        label.textContent = 'Wacom area synced';
        banner.appendChild(label);
        // Auto-dismiss after AUTO_DISMISS_MS (3000ms, RESEARCH.md A3, UI-SPEC.md).
        host.style.display = '';
        dismissTimer = setTimeout(() => {
          host.style.display = 'none';
        }, AUTO_DISMISS_MS);
        break;
      }

      case 'area-changed': {
        label.textContent = 'PDF area changed';
        const btn = document.createElement('button');
        btn.className = 'banner-action';
        btn.textContent = 'Re-calibrate';
        btn.addEventListener('click', () => {
          if (onSyncClickCallback) onSyncClickCallback();
        });
        banner.appendChild(label);
        banner.appendChild(btn);
        break;
      }

      case 'host-not-found': {
        // No action button — user must install the native host (EXT-06, UI-SPEC.md).
        label.textContent = 'Native host not found';
        const hint = document.createElement('span');
        hint.className = 'banner-hint';
        hint.textContent = 'Install Wacom Bridge to activate';
        banner.appendChild(label);
        banner.appendChild(hint);
        break;
      }

      default:
        label.textContent = state;
        banner.appendChild(label);
    }

    reposition();
  }

  // Initial render: show idle state.
  update('idle');

  return { update };
}
