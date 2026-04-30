# Phase 2: Native Messaging Host - Pattern Map

**Mapped:** 2026-04-30
**Files analyzed:** 10 (new files to be created)
**Analogs found:** 0 exact Go analogs / 10 total (greenfield Go project; spike PowerShell scripts
are the closest reference implementations for Wacom logic)

---

## File Classification

| New File | Role | Data Flow | Closest Analog | Match Quality |
|----------|------|-----------|----------------|---------------|
| `cmd/wacom-bridge/main.go` | utility (entry point) | request-response | `spike/run-tests.ps1` (orchestration pattern) | logic-match (different language) |
| `internal/messaging/host.go` | service | request-response | `spike/run-tests.ps1` + RESEARCH.md Pattern 1 | partial (protocol derived from spec) |
| `internal/wacom/xml.go` | service | file-I/O | `spike/Set-WacomMapping.ps1` lines 63–136 | role-match (direct port target) |
| `internal/wacom/service.go` | service | request-response | `spike/SPIKE-RESULTS.md` "Service Names" section | partial (service name verified; Go SCM API from RESEARCH.md) |
| `internal/state/state.go` | model | CRUD | `spike/Set-WacomMapping.ps1` param block (lines 56–58) | partial (state fields derived from params) |
| `internal/logging/logging.go` | utility | file-I/O | `spike/run-tests.ps1` Write-Host pattern | partial (logging approach from RESEARCH.md Pattern 4) |
| `docs/protocol.md` | config (contract doc) | — | `spike/SPIKE-RESULTS.md` "Recommendation for Phase 2" | partial (command names from CONTEXT.md decisions) |
| `installer/wacom-bridge.wxs` | config (installer) | — | none in codebase | no analog (RESEARCH.md Pattern 5 is sole reference) |
| `installer/manifest-chrome.json` | config | — | none in codebase | no analog (spec-defined format) |
| `installer/manifest-edge.json` | config | — | none in codebase | no analog (spec-defined format) |

---

## Pattern Assignments

### `cmd/wacom-bridge/main.go` (utility, request-response)

**Analog:** `spike/run-tests.ps1` (orchestration and prerequisite-check pattern)
**Also derives from:** RESEARCH.md Architecture Patterns section, Pattern 1 (message loop)

**Init and prerequisite-check pattern** (`spike/run-tests.ps1` lines 34–50):
```powershell
# Analog pattern — verify prerequisites before starting the main loop
if (-not (Test-Path (Join-Path $ScriptDir 'baseline-local.Export.wacomxs'))) {
    Write-Host 'ERROR: spike\baseline-local.Export.wacomxs not found.' -ForegroundColor Red
    exit 1
}
```
Port to Go as: verify baseline file exists at `%LOCALAPPDATA%\WacomBridge\baseline.Export.wacomxs`
before entering the message loop. Return structured error JSON if missing (D-12).

**Main entry pattern** (from RESEARCH.md Pattern 1):
```go
// cmd/wacom-bridge/main.go
package main

import (
    "os"
    "internal/logging"
    "internal/messaging"
    "internal/state"
    "internal/wacom"
)

func main() {
    // 1. Open logger (logging.OpenLogger)
    // 2. Verify baseline file exists
    // 3. Init in-process state
    // 4. Start messaging.MessageLoop(dispatch)
    // dispatch is a switch on msg["command"]: set_mapping, reset_mapping, get_status, ping
}
```

**Error handling on unrecoverable init failure:**
D-12 requires staying running and returning error JSON, NOT exiting.
Exception: if stdin/stdout cannot be opened, exit is unavoidable — log and exit(1).

---

### `internal/messaging/host.go` (service, request-response)

**Analog:** RESEARCH.md Pattern 1 (Native Messaging Read/Write Loop) — no codebase analog exists.

**Windows binary-mode init** (RESEARCH.md Pattern 1, confirmed from Microsoft Edge docs):
```go
//go:build windows

package messaging

import (
    "golang.org/x/sys/windows"
    "os"
)

func init() {
    // Switch stdin and stdout to binary mode on Windows.
    // O_TEXT mode (default) converts \n bytes to \r\n, corrupting the 4-byte length prefix.
    // NOTE: SetConsoleMode applies to console handles. For pipe handles spawned by Chrome/Edge,
    // use msvcrt.dll _setmode as described in Open Question 3 (RESEARCH.md).
    windows.SetConsoleMode(windows.Handle(os.Stdin.Fd()),  0)
    windows.SetConsoleMode(windows.Handle(os.Stdout.Fd()), 0)
}
```

**Read/Write framing pattern** (RESEARCH.md Pattern 1, `encoding/binary` + `encoding/json`):
```go
package messaging

import (
    "encoding/binary"
    "encoding/json"
    "fmt"
    "io"
    "os"
)

// ReadMessage reads one Native Messaging message (4-byte LE uint32 length + UTF-8 JSON body).
func ReadMessage(r io.Reader) (map[string]interface{}, error) {
    var length uint32
    if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
        return nil, err // io.EOF here means browser closed the port — clean shutdown
    }
    buf := make([]byte, length)
    if _, err := io.ReadFull(r, buf); err != nil {
        return nil, fmt.Errorf("read body: %w", err)
    }
    var msg map[string]interface{}
    if err := json.Unmarshal(buf, &msg); err != nil {
        return nil, fmt.Errorf("unmarshal: %w", err)
    }
    return msg, nil
}

// WriteMessage writes one Native Messaging message (4-byte LE uint32 length + UTF-8 JSON body).
func WriteMessage(w io.Writer, v interface{}) error {
    body, err := json.Marshal(v)
    if err != nil {
        return err
    }
    if err := binary.Write(w, binary.LittleEndian, uint32(len(body))); err != nil {
        return err
    }
    _, err = w.Write(body)
    return err
}

// MessageLoop reads commands until EOF (browser disconnect) and dispatches them.
// On io.EOF: browser closed port — os.Exit(0) (clean shutdown).
// On other read errors: log and os.Exit(1).
// dispatch must never return nil; always return a map with "ok" or "error"+"code".
func MessageLoop(dispatch func(map[string]interface{}) interface{}) {
    for {
        msg, err := ReadMessage(os.Stdin)
        if err == io.EOF || err == io.ErrUnexpectedEOF {
            os.Exit(0)
        }
        if err != nil {
            os.Exit(1)
        }
        response := dispatch(msg)
        _ = WriteMessage(os.Stdout, response)
    }
}
```

**Command dispatch pattern** (D-07, RESEARCH.md "Don't Hand-Roll" section):
```go
// Simple switch — 4 commands total, no framework needed.
func dispatch(msg map[string]interface{}) interface{} {
    cmd, _ := msg["command"].(string)
    switch cmd {
    case "set_mapping":
        // extract x, y, width, height from msg; call wacom.SetMapping; update state
    case "reset_mapping":
        // call wacom.ResetMapping; update state
    case "get_status":
        // return state as {"mapped": bool, "x": int, "y": int, "width": int, "height": int, "monitor": null}
    case "ping":
        return map[string]interface{}{"ok": true}
    default:
        return map[string]interface{}{
            "error": "unknown command: " + cmd,
            "code":  "ERR_UNKNOWN_COMMAND",
        }
    }
}
```

---

### `internal/wacom/xml.go` (service, file-I/O)

**Analog:** `spike/Set-WacomMapping.ps1` — direct port target.

**Path and precondition pattern** (`spike/Set-WacomMapping.ps1` lines 63–77):
```powershell
$BaselinePath = Join-Path $ScriptDir 'baseline-local.Export.wacomxs'
$TempPath     = Join-Path $env:TEMP 'wacom-mapping-temp.Export.wacomxs'

if (-not (Test-Path $BaselinePath)) {
    Write-Error "baseline-local.Export.wacomxs not found at: $BaselinePath"
    exit 1
}
```
Port to Go as:
```go
baselinePath := filepath.Join(os.Getenv("LOCALAPPDATA"), "WacomBridge", "baseline.Export.wacomxs")
tempPath     := filepath.Join(os.Getenv("TEMP"), "wacom-mapping-temp.Export.wacomxs")
// If baseline missing: return structured error, do NOT exit (D-12)
// Error code: ERR_BASELINE_NOT_FOUND
```

**Clone-and-modify pattern** (`spike/Set-WacomMapping.ps1` lines 89–136):
```powershell
# 1. Load baseline into memory (clone — never write back to $BaselinePath)
[xml]$doc = Get-Content -Path $BaselinePath -Raw -Encoding UTF8

# 2. Navigate to all ScreenArea nodes via XPath
$screenAreaNodes = Select-Xml -Xml $doc -XPath '//InputScreenAreaArray/ArrayElement/ScreenArea' |
                   Select-Object -ExpandProperty Node

# 3. Update ALL entries (test machine had 3 ArrayElement entries)
foreach ($screenArea in @($screenAreaNodes)) {
    $screenArea.AreaType.InnerText = [string]1              # 1 = custom region
    $screenArea.ScreenOutputArea.Origin.X.InnerText  = [string]$X
    $screenArea.ScreenOutputArea.Origin.Y.InnerText  = [string]$Y
    $screenArea.ScreenOutputArea.Extent.X.InnerText  = [string]$Width
    $screenArea.ScreenOutputArea.Extent.Y.InnerText  = [string]$Height
}

# 4. Save to temp path (MUST use .Export.wacomxs extension)
$doc.Save($TempPath)
```

**Go XML struct pattern** (RESEARCH.md Pattern 2, derived from `spike/baseline-reference.Export.wacomxs` lines 84–108):

The XML structure for one `ArrayElement` (confirmed from `baseline-reference.Export.wacomxs`):
```xml
<ScreenArea type="map">
  <AreaType type="integer">0</AreaType>
  <Ballistic type="integer">1</Ballistic>
  <CursorMode type="bool">false</CursorMode>
  <Locks type="integer">0</Locks>
  <MouseHeight type="integer">5</MouseHeight>
  <MouseOrientation type="bool">false</MouseOrientation>
  <MouseSpeed type="integer">5</MouseSpeed>
  <Options type="integer">1</Options>
  <ScreenOutputArea type="map">
    <Extent type="map">
      <X type="integer">1920</X>
      <Y type="integer">1080</Y>
      <Z type="integer">0</Z>
    </Extent>
    <Origin type="map">
      <X type="integer">0</X>
      <Y type="integer">0</Y>
      <Z type="integer">0</Z>
    </Origin>
  </ScreenOutputArea>
  <SensitivityX type="double">1.000000</SensitivityX>
  <SensitivityY type="double">1.000000</SensitivityY>
  <WhichMonitor type="string">Desktop_0_</WhichMonitor>
</ScreenArea>
```

Port to Go as targeted structs covering only mutated fields, with `xml:",innerxml"` passthrough
for the rest (RESEARCH.md Pattern 2 recommendation):
```go
// Only the subtree we modify — everything else round-trips via innerxml.
type XYZ struct {
    X string `xml:"X"`
    Y string `xml:"Y"`
    Z string `xml:"Z"`
}
type ScreenOutputArea struct {
    Extent XYZ `xml:"Extent"`
    Origin XYZ `xml:"Origin"`
}
type ScreenArea struct {
    AreaType         string           `xml:"AreaType"`
    ScreenOutputArea ScreenOutputArea `xml:"ScreenOutputArea"`
    // All other children (Ballistic, CursorMode, etc.) preserved via innerxml or full struct
}
```

**Write-back pattern** (RESEARCH.md Pattern 2):
```go
// Preserve <?xml ...?> declaration when writing back:
out, err := xml.MarshalIndent(doc, "", "  ")
f.WriteString(xml.Header)
f.Write(out)
// File extension: MUST be .Export.wacomxs — .xml silently fails
// [VERIFIED: spike/SPIKE-RESULTS.md "File extension" section]
```

**Reset pattern** (`spike/Reset-WacomMapping.ps1` lines 70–97):
```powershell
# Reset: re-import the unmodified baseline — no XML modification needed
$proc = Start-Process -FilePath $PrefUtilPath -ArgumentList '/import', $BaselinePath -PassThru
$proc | Wait-Process
```
Port to Go as: copy baseline file directly to the Wacom preference write path (no XML
modification required for reset_mapping).

**Error codes** for this module (Claude's discretion per CONTEXT.md):
- `ERR_BASELINE_NOT_FOUND` — baseline file missing
- `ERR_XML_PARSE` — baseline XML is malformed
- `ERR_XML_WRITE` — failed to write temp file
- `ERR_NO_SCREEN_AREA_NODES` — XPath traversal found 0 matching nodes

---

### `internal/wacom/service.go` (service, request-response)

**Analog:** `spike/SPIKE-RESULTS.md` "Service Names" section (verified service name and behavior)
**Also derives from:** RESEARCH.md Pattern 3 (Windows SCM via `golang.org/x/sys/windows/svc/mgr`)

**Service name** (`spike/SPIKE-RESULTS.md` lines 106–119):
```
Service name: WtabletServicePro
Display name: Wacom Professional Service
Status: Running
Service restart required when using PrefUtil: NO (PrefUtil notifies service directly)
Service restart required for direct XML write: UNKNOWN — must test in Plan 02-02
```

**SCM stop/start pattern** (RESEARCH.md Pattern 3):
```go
package wacom

import (
    "fmt"
    "time"
    "golang.org/x/sys/windows/svc"
    "golang.org/x/sys/windows/svc/mgr"
)

const wacomService = "WtabletServicePro"

func RestartWacomService() error {
    m, err := mgr.Connect()
    if err != nil {
        return fmt.Errorf("SCM connect: %w", err)
    }
    defer m.Disconnect()

    s, err := m.OpenService(wacomService)
    if err != nil {
        return fmt.Errorf("open service %q: %w", wacomService, err)
    }
    defer s.Close()

    if _, err := s.Control(svc.Stop); err != nil {
        return fmt.Errorf("stop service: %w", err)
    }

    // Poll until stopped (timeout 10s)
    deadline := time.Now().Add(10 * time.Second)
    for time.Now().Before(deadline) {
        status, err := s.Query()
        if err != nil {
            return fmt.Errorf("query service: %w", err)
        }
        if status.State == svc.Stopped {
            break
        }
        time.Sleep(200 * time.Millisecond)
    }

    if err := s.Start(); err != nil {
        return fmt.Errorf("start service: %w", err)
    }
    return nil
}
```

**Error codes** for this module:
- `ERR_SERVICE_CONNECT` — cannot open SCM
- `ERR_SERVICE_NOT_FOUND` — WtabletServicePro not present
- `ERR_SERVICE_ACCESS_DENIED` — stop/start denied for non-elevated process (Pitfall 4 in RESEARCH.md)
- `ERR_SERVICE_RESTART_TIMEOUT` — service did not stop within 10s
- `ERR_SERVICE_RESTART_FAILED` — start failed after stop

**Conditional invocation note:**
Per D-03, restart is called only if testing (Plan 02-02) shows it is required. The call site
in `xml.go` must be conditioned on a `needsRestart` flag that Plan 02-02 sets based on test findings.

---

### `internal/state/state.go` (model, CRUD)

**Analog:** `spike/Set-WacomMapping.ps1` param block (lines 53–58) — the four coordinates
and implied "mapped" state are the exact fields tracked in-process.

**State fields** (derived from param block):
```powershell
param(
    [Parameter(Mandatory)][int]$X,
    [Parameter(Mandatory)][int]$Y,
    [Parameter(Mandatory)][int]$Width,
    [Parameter(Mandatory)][int]$Height
)
```

Port to Go as:
```go
package state

import "sync"

// State holds the current in-process tablet mapping state.
// Access is serialized — the message loop is single-threaded, but a mutex
// is included for safety if goroutines are added later.
type State struct {
    mu      sync.Mutex
    Mapped  bool
    X       int
    Y       int
    Width   int
    Height  int
}

func (s *State) Set(x, y, w, h int) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.Mapped = true
    s.X, s.Y, s.Width, s.Height = x, y, w, h
}

func (s *State) Reset() {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.Mapped = false
    s.X, s.Y, s.Width, s.Height = 0, 0, 0, 0
}

// StatusResponse returns the get_status response body (D-08).
// "monitor" is reserved/null (MULTI-01 deferred).
func (s *State) StatusResponse() map[string]interface{} {
    s.mu.Lock()
    defer s.mu.Unlock()
    return map[string]interface{}{
        "mapped":  s.Mapped,
        "x":       s.X,
        "y":       s.Y,
        "width":   s.Width,
        "height":  s.Height,
        "monitor": nil, // reserved for MULTI-01
    }
}
```

---

### `internal/logging/logging.go` (utility, file-I/O)

**Analog:** `spike/run-tests.ps1` Write-Host output pattern — both write timestamped
structured messages to console/file. RESEARCH.md Pattern 4 is the direct code reference.

**Log directory pattern** (`spike/run-tests.ps1` lines 37–38 — date prefix convention):
```powershell
Write-Host "Date: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
```
The Go equivalent uses `slog` with `JSONHandler` which timestamps automatically.

**Logger init pattern** (RESEARCH.md Pattern 4):
```go
package logging

import (
    "log/slog"
    "os"
    "path/filepath"
)

// OpenLogger opens (or creates) the log file at %LOCALAPPDATA%\WacomBridge\logs\wacom-bridge.log
// and returns a slog.Logger with JSONHandler.
// Caller is responsible for closing the returned *os.File on shutdown.
func OpenLogger() (*slog.Logger, *os.File, error) {
    localAppData := os.Getenv("LOCALAPPDATA")
    logDir := filepath.Join(localAppData, "WacomBridge", "logs")
    if err := os.MkdirAll(logDir, 0o755); err != nil {
        return nil, nil, err
    }
    logPath := filepath.Join(logDir, "wacom-bridge.log")
    f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
    if err != nil {
        return nil, nil, err
    }
    logger := slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    return logger, f, nil
}
```

**Log-or-stderr fallback:** If `OpenLogger` fails (e.g., `%LOCALAPPDATA%` is unavailable),
fall back to `slog.New(slog.NewJSONHandler(os.Stderr, nil))` and continue running (D-12).

---

### `docs/protocol.md` (config — contract document)

**Analog:** `spike/SPIKE-RESULTS.md` "Recommendation for Phase 2" section (lines 224–238)
— the command names and coordinate semantics are directly documented there.

**Command/response schema** (assembled from D-07 through D-13):

Commands (browser → host):
```json
{ "command": "set_mapping",  "x": 240, "y": 270, "width": 1440, "height": 540 }
{ "command": "reset_mapping" }
{ "command": "get_status" }
{ "command": "ping" }
```

Responses (host → browser):
```json
// set_mapping / reset_mapping success
{ "ok": true }

// get_status
{ "mapped": true, "x": 240, "y": 270, "width": 1440, "height": 540, "monitor": null }

// ping
{ "ok": true }

// any command failure
{ "error": "human-readable message", "code": "ERR_SNAKE_CASE" }
```

**Coordinate semantics** (`spike/SPIKE-RESULTS.md` lines 82–91 and Set-WacomMapping.ps1 lines 111–118):
- All values in physical pixels (DPR-corrected, sent by extension — EXT-02)
- `x` = left edge (maps to `ScreenOutputArea/Origin/X`)
- `y` = top edge (maps to `ScreenOutputArea/Origin/Y`)
- `width` = pixel width (maps to `ScreenOutputArea/Extent/X`)
- `height` = pixel height (maps to `ScreenOutputArea/Extent/Y`)

---

### `installer/wacom-bridge.wxs` (config — WiX installer)

**Analog:** None in codebase. RESEARCH.md Pattern 5 is the sole reference.

**Registry paths** (D-14):
```
HKLM\SOFTWARE\Google\Chrome\NativeMessagingHosts\com.eurefirma.wacombridge
HKLM\SOFTWARE\Microsoft\Edge\NativeMessagingHosts\com.eurefirma.wacombridge
```

**Install path** (D-15): `C:\Program Files\WacomBridge\wacom-bridge.exe`

**WiX 4 structure** (RESEARCH.md Pattern 5):
```xml
<?xml version="1.0" encoding="UTF-8"?>
<Wix xmlns="http://wixtoolset.org/schemas/v4/wxs">
  <Package
    Name="WacomBridge"
    Manufacturer="Eurefirma"
    Version="1.0.0"
    UpgradeCode="<!-- GENERATE NEW GUID -->"
  >
    <MajorUpgrade DowngradeErrorMessage="A newer version of WacomBridge is already installed." />
    <MediaTemplate EmbedCab="yes" />

    <StandardDirectory Id="ProgramFiles6432Folder">
      <Directory Id="INSTALLFOLDER" Name="WacomBridge" />
    </StandardDirectory>

    <ComponentGroup Id="WacomBridgeComponents" Directory="INSTALLFOLDER">
      <Component Id="MainExe" Guid="<!-- GENERATE NEW GUID -->">
        <File Source="wacom-bridge.exe" KeyPath="yes" />
      </Component>

      <Component Id="ManifestChrome" Guid="<!-- GENERATE NEW GUID -->">
        <File Source="manifest-chrome.json" KeyPath="yes" />
        <RegistryKey Root="HKLM"
          Key="SOFTWARE\Google\Chrome\NativeMessagingHosts\com.eurefirma.wacombridge">
          <RegistryValue Type="string" Value="[INSTALLFOLDER]manifest-chrome.json" />
        </RegistryKey>
      </Component>

      <Component Id="ManifestEdge" Guid="<!-- GENERATE NEW GUID -->">
        <File Source="manifest-edge.json" KeyPath="yes" />
        <RegistryKey Root="HKLM"
          Key="SOFTWARE\Microsoft\Edge\NativeMessagingHosts\com.eurefirma.wacombridge">
          <RegistryValue Type="string" Value="[INSTALLFOLDER]manifest-edge.json" />
        </RegistryKey>
      </Component>
    </ComponentGroup>

    <Feature Id="App">
      <ComponentGroupRef Id="WacomBridgeComponents" />
    </Feature>
  </Package>
</Wix>
```

**WOW6432Node note** (RESEARCH.md Pitfall 2): Consider adding a second RegistryKey writing
to `SOFTWARE\WOW6432Node\Microsoft\Edge\...` if 32-bit Edge is in scope.

---

### `installer/manifest-chrome.json` and `installer/manifest-edge.json` (config)

**Analog:** None. Format is defined by the Chrome/Edge Native Messaging specification.

**Manifest format** (RESEARCH.md "Native Messaging Protocol" section):
```json
{
    "name": "com.eurefirma.wacombridge",
    "description": "Wacom Bridge - restricts stylus to PDF region",
    "path": "C:\\Program Files\\WacomBridge\\wacom-bridge.exe",
    "type": "stdio",
    "allowed_origins": [
        "chrome-extension://[EXTENSION-ID]/"
    ]
}
```

**Note on IDs** (RESEARCH.md Pitfall 3): Use sideloaded extension ID during development.
Update `allowed_origins` before MSI production build to use the published store ID.
Separate manifest files for Chrome and Edge are recommended (simpler than a shared file
with both IDs).

---

## Shared Patterns

### Error Response Structure
**Derived from:** CONTEXT.md D-11, D-12, D-13
**Apply to:** All handlers in `cmd/wacom-bridge/main.go`, `internal/wacom/xml.go`, `internal/wacom/service.go`

```go
// Success (set_mapping, reset_mapping, ping)
map[string]interface{}{"ok": true}

// Failure (all commands) — return this, do NOT exit (D-12)
map[string]interface{}{
    "error": "human-readable description",
    "code":  "ERR_SNAKE_CASE",
}
```

Error code convention (Claude's discretion — CONTEXT.md):
- `ERR_BASELINE_NOT_FOUND`
- `ERR_XML_PARSE`
- `ERR_XML_WRITE`
- `ERR_NO_SCREEN_AREA_NODES`
- `ERR_SERVICE_CONNECT`
- `ERR_SERVICE_NOT_FOUND`
- `ERR_SERVICE_ACCESS_DENIED`
- `ERR_SERVICE_RESTART_TIMEOUT`
- `ERR_SERVICE_RESTART_FAILED`
- `ERR_UNKNOWN_COMMAND`
- `ERR_INVALID_PARAMS`

### File Extension Invariant
**Source:** `spike/SPIKE-RESULTS.md` "XML Tag Names and Structure" section (line 59) + `spike/Set-WacomMapping.ps1` line 34
**Apply to:** `internal/wacom/xml.go` (all file write operations)

```
File MUST end in .Export.wacomxs
Using .xml silently fails to apply the mapping — confirmed in Phase 1 spike.
```

### All ArrayElement Entries Must Be Updated
**Source:** `spike/Set-WacomMapping.ps1` lines 44–49 (comment) + `spike/baseline-reference.Export.wacomxs` lines 46–236 (3 entries confirmed)
**Apply to:** `internal/wacom/xml.go` SetMapping function

```
Test machine has 3 ArrayElement entries under InputScreenAreaArray.
ALL must be updated — updating only the first produces inconsistent behavior.
```

### Clone-and-Modify, Never Construct
**Source:** `spike/Set-WacomMapping.ps1` lines 87–91 (D-01 comment), CONTEXT.md D-05
**Apply to:** `internal/wacom/xml.go`

```
Always load baseline.Export.wacomxs → clone in memory → modify → write to temp path.
Never construct minimal XML from scratch — risks silently losing per-tablet settings.
```

### Physical Pixel Coordinates
**Source:** `spike/SPIKE-RESULTS.md` "DPI / Coordinate System Finding" section (lines 213–218)
**Apply to:** `internal/wacom/xml.go`, `docs/protocol.md`

```
Coordinates are physical pixels (not CSS/logical pixels).
Extension (Phase 3, EXT-02) must send DPR-corrected values.
Host does NOT perform DPR conversion — it writes values as received.
```

### No PrefUtil in Production Code
**Source:** CONTEXT.md D-02, D-04; `spike/SPIKE-RESULTS.md` "Issues Encountered" #1
**Apply to:** All files

```
PrefUtil is NEVER invoked at runtime. Direct XML write + conditional service restart only.
Any reference to PrefUtil in non-spike code is a bug.
```

---

## No Analog Found (Go patterns)

All Go source files are new with no existing Go codebase to copy from.
The closest reference implementations are:

| File | Role | Data Flow | Reference Source |
|------|------|-----------|------------------|
| `cmd/wacom-bridge/main.go` | utility | request-response | RESEARCH.md arch diagram + spike/run-tests.ps1 orchestration |
| `internal/messaging/host.go` | service | request-response | RESEARCH.md Pattern 1 (spec-derived, no existing implementation) |
| `internal/wacom/xml.go` | service | file-I/O | `spike/Set-WacomMapping.ps1` + `spike/Reset-WacomMapping.ps1` (direct port) |
| `internal/wacom/service.go` | service | request-response | RESEARCH.md Pattern 3 (SCM API reference) |
| `internal/state/state.go` | model | CRUD | `spike/Set-WacomMapping.ps1` param block (field names) |
| `internal/logging/logging.go` | utility | file-I/O | RESEARCH.md Pattern 4 (slog reference) |
| `installer/wacom-bridge.wxs` | config | — | RESEARCH.md Pattern 5 (WiX 4 reference) |
| `installer/manifest-chrome.json` | config | — | Chrome/Edge Native Messaging spec |
| `installer/manifest-edge.json` | config | — | Chrome/Edge Native Messaging spec |

---

## Open Questions Affecting Pattern Implementation

The following RESEARCH.md open questions must be resolved in Plan 02-02 and may change
how certain patterns are applied:

| # | Question | Affected File | Current Pattern | Fallback if Wrong |
|---|----------|---------------|-----------------|-------------------|
| OQ-1 | Does direct XML write require service restart? | `internal/wacom/service.go` | Conditional restart | Skip restart if hot-reload confirmed |
| OQ-2 | What is the exact write path the Wacom service reads? | `internal/wacom/xml.go` | `%APPDATA%\WTablet\` (assumed) | Process Monitor investigation required |
| OQ-3 | Exact Go mechanism to set pipe handles to binary mode | `internal/messaging/host.go` | `windows.SetConsoleMode` | `msvcrt.dll _setmode` via `NewLazySystemDLL` |

---

## Metadata

**Analog search scope:** `spike/` directory (only source code in repo)
**Files scanned:** `spike/Set-WacomMapping.ps1`, `spike/Reset-WacomMapping.ps1`, `spike/run-tests.ps1`, `spike/baseline-reference.Export.wacomxs` (partial — XML schema extracted), `spike/SPIKE-RESULTS.md`
**Pattern extraction date:** 2026-04-30
**Go module:** greenfield — no existing Go files in repo
