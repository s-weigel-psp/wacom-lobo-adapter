# Wacom Bridge Native Messaging Protocol

**Version:** 1.0  
**Host name:** `com.brantpoint.wacombridge`  
**Scope:** JSON command/response contract between the Chrome/Edge browser extension (Phase 3) and the Go native messaging host (Phase 2).

This document is the authoritative interface contract for Phase 3 parallel development. The Phase 3 developer does not need access to the host source code — this document alone defines every message the extension may send and every response it will receive.

---

## 1. Framing

| Property | Value |
|----------|-------|
| Transport | `stdin` / `stdout` of the spawned host process |
| Length header | 4-byte unsigned integer, **little-endian** (native byte order on x86/x64 Windows) |
| Body encoding | UTF-8 JSON, no BOM |
| Max message (host → browser) | 1 MB (Chrome/Edge spec limit) |
| Direction | Both directions use identical framing |

**Wire format (each message):**

```
[ 4 bytes: uint32 LE length ] [ N bytes: UTF-8 JSON body ]
```

The length prefix encodes the byte length of the JSON body only (not including the 4-byte header itself).

---

## 2. Commands (Browser → Host)

### 2.1 `set_mapping`

Restricts the Wacom stylus to a specific screen region.

**Request:**
```json
{ "command": "set_mapping", "x": 240, "y": 270, "width": 1440, "height": 540 }
```

| Field | Type | Description |
|-------|------|-------------|
| `command` | string | `"set_mapping"` |
| `x` | integer | Left edge of target region in **physical pixels** |
| `y` | integer | Top edge of target region in **physical pixels** |
| `width` | integer | Width of target region in **physical pixels** |
| `height` | integer | Height of target region in **physical pixels** |

**Success response:**
```json
{ "ok": true }
```

**Failure response** (see Section 5 for error codes):
```json
{ "error": "human-readable message", "code": "ERR_SNAKE_CASE" }
```

---

### 2.2 `reset_mapping`

Restores the Wacom stylus to full-screen mapping (baseline profile).

**Request:**
```json
{ "command": "reset_mapping" }
```

**Success response:**
```json
{ "ok": true }
```

**Failure response:**
```json
{ "error": "human-readable message", "code": "ERR_SNAKE_CASE" }
```

---

### 2.3 `get_status`

Returns the current in-process mapping state.

**Request:**
```json
{ "command": "get_status" }
```

**Success response (when mapped):**
```json
{ "mapped": true, "x": 240, "y": 270, "width": 1440, "height": 540, "monitor": null }
```

**Success response (when not mapped / after reset):**
```json
{ "mapped": false, "x": 0, "y": 0, "width": 0, "height": 0, "monitor": null }
```

| Field | Type | Description |
|-------|------|-------------|
| `mapped` | boolean | `true` if `set_mapping` has been called and not subsequently reset |
| `x` | integer | Last `set_mapping` x value, or `0` if not mapped |
| `y` | integer | Last `set_mapping` y value, or `0` if not mapped |
| `width` | integer | Last `set_mapping` width value, or `0` if not mapped |
| `height` | integer | Last `set_mapping` height value, or `0` if not mapped |
| `monitor` | null | **Reserved — always `null` in v1.** See Section 6 (Reserved Fields). |

---

### 2.4 `ping`

Health check. Returns immediately with no side effects.

**Request:**
```json
{ "command": "ping" }
```

**Success response:**
```json
{ "ok": true }
```

---

## 3. Coordinate Semantics

All coordinate values in `set_mapping` and `get_status` are **physical pixels** — NOT logical (CSS) pixels.

| Protocol field | Wacom XML element | Semantics |
|---------------|-------------------|-----------|
| `x` | `ScreenOutputArea/Origin/X` | Left edge of region from screen origin |
| `y` | `ScreenOutputArea/Origin/Y` | Top edge of region from screen origin |
| `width` | `ScreenOutputArea/Extent/X` | Pixel width of the region |
| `height` | `ScreenOutputArea/Extent/Y` | Pixel height of the region |

**DPI scaling note:** Windows DPI scaling (125%, 150%, etc.) causes browser APIs to return logical (CSS) pixels. The extension (Phase 3, requirement EXT-02) is responsible for multiplying coordinates by `window.devicePixelRatio` before sending them. The native host writes coordinates exactly as received — it does NOT perform DPI conversion.

**Example:** On a 1920×1080 display at 150% DPI scaling, a DOM element at CSS position (160, 180) with CSS size (960, 360) should be sent as `{ "x": 240, "y": 270, "width": 1440, "height": 540 }` (values multiplied by 1.5).

---

## 4. Error Response Shape

All command failures use the same structured error response:

```json
{ "error": "human-readable description of what went wrong", "code": "ERR_SNAKE_CASE" }
```

The host **stays running** after returning an error response — it does not exit. The extension is responsible for displaying the error in the UI (Phase 3, EXT-05).

---

## 5. Error Codes

| Code | Trigger |
|------|---------|
| `ERR_UNKNOWN_COMMAND` | `command` field is absent or does not match any known command |
| `ERR_INVALID_PARAMS` | Required fields are missing, wrong type, or out of range for `set_mapping` |
| `ERR_BASELINE_NOT_FOUND` | `%LOCALAPPDATA%\WacomBridge\baseline.Export.wacomxs` is missing |
| `ERR_XML_PARSE` | The baseline XML file is malformed and cannot be parsed |
| `ERR_XML_WRITE` | Failed to write the modified preference XML to the temp path |
| `ERR_NO_SCREEN_AREA_NODES` | XPath traversal of baseline XML found 0 `ScreenArea` nodes |
| `ERR_SERVICE_CONNECT` | Cannot connect to the Windows Service Control Manager |
| `ERR_SERVICE_NOT_FOUND` | `WtabletServicePro` service not present on this machine |
| `ERR_SERVICE_ACCESS_DENIED` | Stop/start of `WtabletServicePro` denied for non-elevated process |
| `ERR_SERVICE_RESTART_TIMEOUT` | Service did not reach Stopped state within 10 seconds |
| `ERR_SERVICE_RESTART_FAILED` | Service failed to start after being stopped |

---

## 6. Reserved Fields

### `monitor` in `get_status`

The `monitor` field is included in every `get_status` response but is **always `null`** in v1.

It is reserved for multi-monitor support (requirement MULTI-01, deferred to v2). Phase 3 code should handle `monitor: null` without error and must not assume `monitor` will remain `null` in future versions.

---

## 7. Host Lifecycle

| Event | Host behaviour |
|-------|---------------|
| Browser calls `chrome.runtime.connectNative("com.brantpoint.wacombridge")` | Chrome/Edge spawns `wacom-bridge.exe` as a child process |
| Normal command received | Host processes and responds; continues running |
| Command error | Host returns `{"error": ..., "code": ...}` and **stays running** (does not exit) |
| Browser closes the port (tab closed, extension disabled, etc.) | stdin receives EOF → host exits with code 0 |
| Unrecoverable read error (not EOF) | Host exits with code 1 |

**First argument:** Chrome/Edge passes the calling extension's origin as the first command-line argument: `chrome-extension://[extension-id]/`. The host validates this against the `allowed_origins` list in the native messaging manifest JSON (registered by the MSI installer).

---

## 8. Native Messaging Manifest

The installer registers the following manifest JSON under HKLM for both Chrome and Edge:

```json
{
    "name": "com.brantpoint.wacombridge",
    "description": "Wacom Bridge — restricts stylus to PDF annotation region",
    "path": "C:\\Program Files\\WacomBridge\\wacom-bridge.exe",
    "type": "stdio",
    "allowed_origins": [
        "chrome-extension://[CHROME-EXTENSION-ID]/",
        "chrome-extension://[EDGE-EXTENSION-ID]/"
    ]
}
```

Registry keys (HKLM):
- Chrome: `SOFTWARE\Google\Chrome\NativeMessagingHosts\com.brantpoint.wacombridge`
- Edge: `SOFTWARE\Microsoft\Edge\NativeMessagingHosts\com.brantpoint.wacombridge`

Both registry key default values point to the absolute path of the manifest JSON file installed by the MSI.

**Note on extension IDs:** The `[CHROME-EXTENSION-ID]` and `[EDGE-EXTENSION-ID]` placeholders must be replaced with the permanent IDs assigned by the Chrome Web Store and Microsoft Edge Add-ons at publication time. During development, use the sideloaded extension IDs. Wildcards are NOT permitted by the native messaging specification.

---

*Protocol version: 1.0*  
*Authored: Phase 2, Plan 02-01*  
*Depends on: spike/SPIKE-RESULTS.md (coordinate semantics, XML schema), CONTEXT.md D-06 through D-13*
