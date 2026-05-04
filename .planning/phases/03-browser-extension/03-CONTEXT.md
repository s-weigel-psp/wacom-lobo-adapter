# Phase 3: Browser Extension — Context

**Gathered:** 2026-05-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Build a Chrome/Edge Manifest V3 extension that reads the position of a configured DOM element in a third-party PDF viewer, sends DPR-corrected physical pixel coordinates to the native messaging host, and shows a Shadow DOM banner reflecting sync status. The banner prompts re-calibration when the PDF element moves or resizes. Does NOT include native host code (Phase 2) or GPO/Intune deployment (Phase 4).

Three plans in scope: (1) scaffold MV3 extension — manifest, service worker, content script message routing; (2) implement coordinate calculation in the content script; (3) implement the Shadow DOM banner UI with all states and staleness detection.

</domain>

<decisions>
## Implementation Decisions

### Target Configuration

- **D-01:** Target element ID and URL match pattern are **configurable at runtime** — not a single hardcoded value. The extension supports an array of `{elementId, urlPattern}` tuples.
- **D-02:** Default configuration (used for testing and as shipped default): `elementId = "draw-canvas"`, `urlPattern = "file:///C:/WacomTest/*.html"`. These constants are defined in a single config file; downstream plans must note that production deployment replaces them before shipping.
- **D-03:** Additional target tuples are **user-editable via the extension options page** using `chrome.storage.sync`. Users (or IT administrators) can add, edit, or remove tuples from the options page. `chrome.storage.managed` may override sync storage in enterprise deployments but is not required in Phase 3.

### Banner Design

- **D-04:** Banner is **anchored above the target DOM element** — positioned relative to the element using its current screen coordinates, not fixed to the viewport. The banner must recompute its position whenever the element moves or resizes.
- **D-05:** Four visual states:
  - **Not synced (idle)** — Extension loaded but sync not yet activated. Shows "Sync now" button to trigger the first `set_mapping` call.
  - **Synced (green)** — `set_mapping` completed successfully. Shows confirmation message.
  - **Area changed (warning)** — Staleness detection fired after a previous sync. Shows "Re-calibrate" button to re-send updated coordinates.
  - **Host not found (error)** — Native host not installed or not responding (ERR codes from host). Shows "Native host not found" message (EXT-06).
- **D-06:** Banner **auto-dismisses** after showing the "Synced" state for a short interval (exact timing: Claude's discretion). Banner reappears immediately on area-changed, host-not-found, or not-synced state transitions.

### Staleness Detection

- **D-07:** Three triggers for the "area changed" state:
  1. **ResizeObserver** — fires when the target DOM element changes size (browser zoom, panel resize, etc.)
  2. **`window` resize event** — fires when the browser window is resized (shifts element's screen position)
  3. **screenX/Y polling** — detects window moves (no native event exists for this)
- **D-08:** screenX/Y polling interval: **1 second**. Polling only runs after a successful sync has been established (no point polling when "not synced").

### Claude's Discretion

- MV3 service worker connection model (`connectNative` persistent port vs. `sendNativeMessage` per-call). Recommended: `sendNativeMessage` for simplicity with MV3's ephemeral service worker model; reconnect logic is handled per-call.
- Debounce/throttle on staleness triggers to prevent rapid state transitions when the user is actively resizing the window.
- Page Visibility API integration — pause polling when the tab is backgrounded.
- Exact auto-dismiss duration for the "Synced" state.
- Banner CSS styling within the Shadow DOM.
- Options page layout and validation UX.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Protocol Contract
- `docs/protocol.md` — Full JSON command/response contract for Phase 3 to follow. Defines `set_mapping`, `reset_mapping`, `get_status`, `ping` request/response shapes, error codes, framing (4-byte LE length prefix), DPI note, and native messaging manifest format. **Read before implementing service worker message routing.**

### Project Requirements
- `.planning/REQUIREMENTS.md` — EXT-01 through EXT-06 define exactly what Phase 3 must deliver.
- `.planning/PROJECT.md` — Key Decisions table (explicit sync model, Shadow DOM for banner, extension parallel with host), Constraints section (Chrome/Edge MV3, GPO-managed, Windows-only).

### Prior Phase Context
- `.planning/phases/02-native-messaging-host/02-CONTEXT.md` — D-10 (4-byte framing), D-11/D-12/D-13 (error handling), D-14/D-15 (HKLM installer, host binary path). Establishes that error responses follow `{"error": "...", "code": "ERR_SNAKE_CASE"}` shape which EXT-06 must handle.

### Phase 2 Artifacts (integration reference)
- `installer/manifest-chrome.json` — Contains PLACEHOLDER_CHROME_EXTENSION_ID. Phase 3 MUST replace this with the real sideloaded extension ID during development and the published ID before shipping.
- `installer/manifest-edge.json` — Same, PLACEHOLDER_EDGE_EXTENSION_ID.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- No extension code exists yet — Phase 3 creates the `extension/` directory from scratch.
- `docs/protocol.md` — Complete contract; the service worker implements exactly the messages defined here.
- `installer/manifest-chrome.json` / `installer/manifest-edge.json` — Phase 3 must provide the real extension IDs to replace the placeholders in these files (or document them as post-publish replacements).

### Established Patterns
- Physical pixel convention: all coordinates are physical pixels (multiply CSS values by `window.devicePixelRatio`). This is established in Phase 1 spike and Phase 2 context. Extension must follow.
- Clone-and-modify pattern (Phase 1/2) does not apply to the extension — extension has its own architecture.
- Error shape `{"error": "...", "code": "ERR_SNAKE_CASE"}` is the host's contract. Extension reads `code` to decide banner state.

### Integration Points
- Native messaging host registered as `com.brantpoint.wacombridge` (HKLM) — extension manifest must declare this host name in `externally_connectable` or `nativeMessaging` permission.
- Extension ID must be added to `allowed_origins` in the native messaging manifest JSON files after sideloading/publishing.
- `chrome.storage.sync` — for user-configurable additional target tuples via options page.

</code_context>

<specifics>
## Specific Ideas

- Testing default is `elementId="draw-canvas"`, `urlPattern="file:///C:/WacomTest/*.html"`. The production element ID and URL pattern are a required input before shipping — the plan should note this as a fill-in-before-ship item.
- The options page supports an array of `{elementId, urlPattern}` tuples, not just a single pair. Phase 4 may push additional tuples via `chrome.storage.managed`.
- Banner is anchored *above* the target element, not fixed to the viewport — it follows the element. This requires position recomputation in the banner update logic.
- Auto-dismiss after "Synced" keeps the UX clean for users who sync once and then annotate without interruption.

</specifics>

<deferred>
## Deferred Ideas

- **MV3 persistent connection / keep-alive** — `chrome.runtime.connectNative` with keep-alive pings was considered but deferred to Claude's discretion (Phase 3 can use `sendNativeMessage` for simplicity).
- **Multi-monitor support** (MULTI-01) — `monitor` field is reserved in `get_status`, not implemented. Phase 3 reads `monitor: null` without error and must not assume it stays null.
- **OPS-01 configurable log verbosity** — v2 requirement; not in Phase 3.
- **chrome.storage.managed for enterprise target config** — IT push of additional tuples is supported architecturally (managed can override sync) but not explicitly implemented in Phase 3's options page; Phase 4 may add managed policy handling.

</deferred>

---

*Phase: 03-browser-extension*
*Context gathered: 2026-05-04*
