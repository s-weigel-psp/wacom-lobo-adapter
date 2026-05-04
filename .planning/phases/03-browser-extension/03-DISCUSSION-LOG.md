# Phase 3: Browser Extension — Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-04
**Phase:** 03-browser-extension
**Areas discussed:** Target configuration, Banner placement & states, Staleness detection

---

## Target Configuration

| Option | Description | Selected |
|--------|-------------|----------|
| Hardcoded element ID | Single element ID baked into the code | |
| It's configurable — no hardcode | Support array of {elementId, urlPattern} tuples | ✓ |

**Config source:**

| Option | Description | Selected |
|--------|-------------|----------|
| Managed storage (GPO) | chrome.storage.managed, IT-pushed | |
| Options page (user-editable) | chrome.storage.sync via extension options page | ✓ |
| Hard-code defaults + managed override | Defaults baked in, overridable via managed | |

**Default target values:**
- elementId: `draw-canvas`
- urlPattern: `file:///C:/WacomTest/*.html`
- Context: testing/development defaults; production values are fill-in-before-ship

**Notes:** User wants a default config AND user-editable additional tuples via options page. Managed storage may override in Phase 4 enterprise deployment.

---

## Banner Placement & States

| Option | Description | Selected |
|--------|-------------|----------|
| Fixed top-right of viewport | Always visible, standard notification position | |
| Fixed bottom-center of viewport | Less intrusive top area | |
| Anchored above the PDF element | Positioned relative to target element, follows it | ✓ |

**States:**

| Option | Description | Selected |
|--------|-------------|----------|
| Synced (green / OK) | After successful set_mapping | ✓ |
| Area changed (warning) | After staleness detection fires | ✓ |
| Host not found (error) | Native host not installed/responding | ✓ |
| Not synced (idle) | Extension loaded, sync not yet activated | ✓ |

**Dismiss behavior:**

| Option | Description | Selected |
|--------|-------------|----------|
| Permanently visible | Always shows while on target page | |
| Auto-dismiss when synced | Hides after "Synced" briefly, reappears on warning/error | ✓ |
| Collapsible / minimizable | User can collapse to icon | |

**Notes:** All four states confirmed. Banner auto-dismisses after Synced state; reappears on any non-synced transition.

---

## Staleness Detection

**Triggers selected (multiSelect):**

| Option | Description | Selected |
|--------|-------------|----------|
| Element resize (ResizeObserver) | PDF canvas changes size | ✓ |
| Window resize event | Browser window resized | ✓ |
| Window move (screenX/Y polling) | User drags window | ✓ |

**Polling interval:**

| Option | Description | Selected |
|--------|-------------|----------|
| 1 second | Standard, low CPU | ✓ |
| 500ms | More responsive, slightly higher CPU | |
| 2 seconds | Least CPU, 2s lag | |

**Notes:** All three staleness triggers selected. 1-second polling chosen for screenX/Y. Debounce/throttle left to Claude's discretion.

---

## Claude's Discretion

- MV3 connection model (connectNative vs sendNativeMessage) — not selected by user; Claude decides
- Debounce/throttle on staleness triggers
- Page Visibility API integration for background tab pause
- Auto-dismiss duration for Synced state
- Banner CSS styling within Shadow DOM
- Options page layout and validation

## Deferred Ideas

- Persistent connectNative + keep-alive (MV3 complexity)
- Multi-monitor support (MULTI-01, reserved in get_status)
- OPS-01 log verbosity (v2)
- chrome.storage.managed for enterprise target config (Phase 4)
