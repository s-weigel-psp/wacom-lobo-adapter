# Phase 2: Native Messaging Host — Research

**Researched:** 2026-04-30
**Domain:** Go binary, Chrome/Edge Native Messaging protocol, Windows XML file I/O, Windows Service Control Manager, WiX 4 MSI
**Confidence:** MEDIUM-HIGH (protocol and Go stack HIGH; Wacom file path MEDIUM — must be verified on target machine)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Native host written in **Go** (not C# .NET 8). Single-binary, no runtime dependency.
- **D-02:** Wacom integration via **direct XML file write** — PrefUtil is NOT used at runtime.
- **D-03:** After writing the XML, host restarts `WtabletServicePro` **if required**. Whether restart is necessary (vs. hot-reload) is unknown — MUST be tested in Plan 02-02. Implement conditional restart.
- **D-04:** No PrefUtil fallback. If direct XML write fails, return structured error. PrefUtil must not appear in production code path.
- **D-05:** XML editing: clone-and-modify existing preference file. XPath: `//InputScreenAreaArray/ArrayElement/ScreenArea`. Update ALL `ArrayElement` entries. Extension MUST be `.Export.wacomxs`.
- **D-06:** `docs/protocol.md` is the first deliverable of Plan 02-01, authored before any Go code.
- **D-07:** Commands: `set_mapping`, `reset_mapping`, `get_status`, `ping`.
- **D-08:** `get_status` response: `{"mapped": bool, "x": int, "y": int, "width": int, "height": int}`. `monitor` field reserved (null).
- **D-09:** `ping` response: `{"ok": true}`.
- **D-10:** Native Messaging framing: 4-byte little-endian length prefix + UTF-8 JSON body.
- **D-11:** Failure responses: `{"error": "...", "code": "ERR_SNAKE_CASE"}`.
- **D-12:** On unrecoverable error: return error JSON and **stay running**. Do not exit.
- **D-13:** Success responses for `set_mapping` and `reset_mapping`: `{"ok": true}`.
- **D-14:** HKLM registration — Chrome: `HKLM\SOFTWARE\Google\Chrome\NativeMessagingHosts\com.eurefirma.wacombridge`; Edge: `HKLM\SOFTWARE\Microsoft\Edge\NativeMessagingHosts\com.eurefirma.wacombridge`.
- **D-15:** Binary at `C:\Program Files\WacomBridge\wacom-bridge.exe`. No elevation required at runtime.
- **D-16:** MSI installs and uninstalls silently (`/quiet`).

### Claude's Discretion

- Go module name and package structure
- Exact error codes (ERR_FILE_NOT_FOUND, ERR_SERVICE_RESTART_FAILED, etc.)
- Logging format within `%LOCALAPPDATA%\WacomBridge\logs\` (HOST-04)
- Windows Service Control Manager vs. `net stop/start` for service restart
- WiX component layout and upgrade strategy

### Deferred Ideas (OUT OF SCOPE)

- Multi-monitor support (MULTI-01)
- Log rotation / verbosity levels (OPS-01)
- Graceful fallback when host not installed (OPS-02) — handled by Phase 3
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| HOST-01 | Native messaging host processes `set_mapping` and applies Wacom screen region mapping | Native Messaging protocol (§ Protocol), XML editing patterns (§ XML), service restart (§ SCM) |
| HOST-02 | Native messaging host processes `reset_mapping` and restores baseline profile | Same as HOST-01; baseline file path section |
| HOST-03 | Host responds to `get_status` and `ping` commands | Native Messaging protocol; state tracking in process memory |
| HOST-04 | Host logs activity to `%LOCALAPPDATA%\WacomBridge\logs\` | Go `log/slog` with `JSONHandler` to file (§ Logging) |
| HOST-05 | WiX MSI installer registers Chrome and Edge native messaging manifests in HKLM | WiX 4 `RegistryKey`/`RegistryValue` pattern (§ WiX) |
| HOST-06 | Installer runs silently without user interaction | `msiexec /quiet` flag; `MajorUpgrade` element (§ WiX) |
</phase_requirements>

---

## Summary

Phase 2 builds a Go executable that bridges a Chrome/Edge extension to the Wacom driver on Windows. The three deliverables map to three plans: the protocol contract (`docs/protocol.md`), the Go native-messaging-host binary, and the WiX 4 MSI installer.

The Chrome/Edge Native Messaging protocol is well-documented and straightforward to implement in Go: a 4-byte little-endian uint32 length prefix followed by a UTF-8 JSON body in both directions, using `os.Stdin`/`os.Stdout`. The critical Windows pitfall is that stdin/stdout default to text mode (O_TEXT), which corrupts the binary length prefix by converting `\n` bytes to `\r\n`. The Go runtime on Windows requires a syscall to set binary mode before the first read, or use a library that wraps this.

The XML editing approach is a direct port of the Phase 1 PowerShell logic: load the baseline `.Export.wacomxs` file, clone it in memory, navigate to all `//InputScreenAreaArray/ArrayElement/ScreenArea` nodes via path traversal, update `AreaType`, `ScreenOutputArea/Origin/X`, `ScreenOutputArea/Origin/Y`, `ScreenOutputArea/Extent/X`, `ScreenOutputArea/Extent/Y`, then write back. Go's `encoding/xml` uses struct-based marshaling; the simplest approach for this use case is to parse into `map[string]interface{}` or to use a targeted struct for the subtree. Alternatively, treating the document as a string-based DOM via Go's `encoding/xml.Decoder`/`Encoder` is possible but brittle. The recommended approach is a targeted struct covering the `InputScreenAreaArray` subtree plus `xml:",innerxml"` passthrough for all surrounding structure.

The WiX 4 installer is straightforward: a single `.wxs` file with `ProgramFiles6432Folder`, one Component containing the `.exe` and two `RegistryKey` elements (Chrome and Edge HKLM paths), and a `MajorUpgrade` element for future upgrades.

**Primary recommendation:** Implement the native messaging loop manually using `encoding/binary` + `encoding/json` — this avoids external library dependencies, gives full control over the Windows binary-mode init sequence, and keeps the binary small. Use `log/slog` with `JSONHandler` writing to a rotated file for HOST-04.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Native Messaging I/O loop | Go binary (stdin/stdout) | — | Chrome spawns the process and pipes messages via stdin/stdout |
| JSON command dispatch | Go binary (in-process) | — | All command logic lives in the host process |
| Wacom XML editing | Go binary (filesystem) | — | Direct file write; no external process involvement |
| Service restart (conditional) | Go binary (Windows SCM) | — | Uses `golang.org/x/sys/windows/svc/mgr` |
| Baseline file storage | Filesystem (`%LOCALAPPDATA%\WacomBridge\`) | — | Per-user baseline copy shipped/stored by installer |
| Logging | Go binary (filesystem) | — | `log/slog` to `%LOCALAPPDATA%\WacomBridge\logs\` |
| Registry registration | MSI installer (HKLM) | — | Written once at install time, read by Chrome/Edge on host spawn |
| Native messaging manifest JSON | Filesystem (arbitrary path) | — | Manifest JSON file referenced by registry key default value |

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `encoding/binary` | stdlib | Read/write 4-byte length prefix | No external dep; handles `binary.LittleEndian` explicitly |
| `encoding/json` | stdlib | Marshal/unmarshal JSON command bodies | No external dep |
| `encoding/xml` | stdlib | Parse and rewrite `.Export.wacomxs` files | No external dep; sufficient for struct-based clone-and-modify |
| `log/slog` | stdlib (Go 1.21+) | Structured JSON logging to file | Official stdlib since Go 1.21; `JSONHandler` built in |
| `golang.org/x/sys/windows/svc/mgr` | latest | Windows SCM service stop/start | Official Go extended library for Windows API |
| `golang.org/x/sys/windows` | latest | `SetConsoleMode` / binary mode init for stdin/stdout | Official Windows API bindings |

[VERIFIED: pkg.go.dev/golang.org/x/sys/windows/svc/mgr — function signatures confirmed]
[VERIFIED: pkg.go.dev/log/slog — in stdlib since Go 1.21]

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `os` | stdlib | Open log file, resolve `%LOCALAPPDATA%` via `os.Getenv` | Standard I/O and filesystem operations |
| `path/filepath` | stdlib | Construct platform-correct file paths | Windows path joining |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Manual `encoding/binary` loop | `github.com/sashahilton00/native-messaging-host` or `github.com/rickypc/native-messaging-host` | Library simplifies framing but adds a dependency; manual is ~30 lines and gives direct control over Windows binary-mode init |
| `log/slog` | `github.com/rs/zerolog` | zerolog has zero-allocation JSON; not needed for this low-frequency use case |

**Installation (on Windows build machine):**
```bash
go get golang.org/x/sys
```

**Version verification:** [VERIFIED: npm view not applicable; Go module proxy used]
```bash
go list -m golang.org/x/sys
```

---

## Architecture Patterns

### System Architecture Diagram

```
Browser Extension (Phase 3)
        |
   chrome.runtime.connectNative("com.eurefirma.wacombridge")
        |
        v
[Chrome/Edge spawns process]
        |
   stdin  stdout
        |
   +-----------+
   | wacom-    |  loop: read 4-byte LE uint32 → read N bytes JSON
   | bridge.exe|        dispatch command
   |           |        write 4-byte LE uint32 → write JSON response
   |  in-proc  |
   |  state:   |
   |  mapped?  |
   |  x,y,w,h  |
   +-----------+
        |
   [set_mapping / reset_mapping]
        |
        v
   %LOCALAPPDATA%\WacomBridge\
        ├── baseline.Export.wacomxs   (copy of per-machine baseline)
        └── logs\wacom-bridge.log
        |
   [clone baseline → modify ScreenArea nodes → write temp .Export.wacomxs]
        |
        v
   WtabletServicePro (Windows service)
        |
   [conditional restart via Windows SCM if hot-reload not confirmed]
        |
        v
   Wacom driver applies new screen mapping region
```

### Recommended Project Structure

```
cmd/
  wacom-bridge/
    main.go           # entry point: init logging, start message loop
internal/
  messaging/
    host.go           # read/write Native Messaging framing (binary.LittleEndian)
  wacom/
    xml.go            # clone-and-modify XML; locate ScreenArea nodes
    service.go        # WtabletServicePro stop/start via svc/mgr
  state/
    state.go          # in-process mapping state (mapped bool, x, y, w, h)
  logging/
    logging.go        # open %LOCALAPPDATA%\WacomBridge\logs\, create slog.Logger
docs/
  protocol.md         # JSON command/response contract (first deliverable)
installer/
  wacom-bridge.wxs    # WiX 4 package definition
  manifest-chrome.json
  manifest-edge.json
go.mod
go.sum
```

### Pattern 1: Native Messaging Read/Write Loop

**What:** Read a 4-byte little-endian uint32 from stdin (message length), then read that many bytes as UTF-8 JSON. Write responses with the same framing.

**When to use:** The message loop in `main.go` and any test helpers.

**Critical Windows prerequisite:** Before the first read, stdin and stdout must be switched to binary mode (O_BINARY). Without this, `\n` bytes in the JSON or length header get converted to `\r\n`, corrupting the framing.

```go
// Source: [CITED: learn.microsoft.com/en-us/microsoft-edge/extensions/developer-guide/native-messaging]
// + [ASSUMED: Go syscall pattern for Windows binary mode — verify on Windows]

//go:build windows

package messaging

import (
    "golang.org/x/sys/windows"
    "os"
)

func init() {
    // Set stdin and stdout to binary mode on Windows.
    // Without this, O_TEXT mode converts \n -> \r\n in the length prefix bytes.
    windows.SetConsoleMode(windows.Handle(os.Stdin.Fd()),  0)
    windows.SetConsoleMode(windows.Handle(os.Stdout.Fd()), 0)
    // Alternative if SetConsoleMode is insufficient for piped handles:
    // use syscall.Open with O_BINARY via the _setmode CRT function via CGO or
    // use the x/sys/windows approach documented below.
}
```

**Note:** The exact syscall for setting binary mode on piped (non-console) handles in Go on Windows needs to be verified against the Windows test machine. `SetConsoleMode` works for console handles; piped handles may need a different approach. See Pitfall 2 below.

```go
// Source: [VERIFIED: pkg.go.dev/encoding/binary — LittleEndian confirmed]
// Source: [CITED: developer.chrome.com native-messaging spec]

package messaging

import (
    "encoding/binary"
    "encoding/json"
    "fmt"
    "io"
    "os"
)

// ReadMessage reads one Native Messaging message from r.
func ReadMessage(r io.Reader) (map[string]interface{}, error) {
    var length uint32
    if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
        return nil, err // EOF here means browser closed the port → exit
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

// WriteMessage writes one Native Messaging message to w.
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
func MessageLoop(dispatch func(map[string]interface{}) interface{}) {
    for {
        msg, err := ReadMessage(os.Stdin)
        if err == io.EOF || err == io.ErrUnexpectedEOF {
            os.Exit(0) // browser closed the port — clean shutdown
        }
        if err != nil {
            // Non-EOF read error: log and exit
            os.Exit(1)
        }
        response := dispatch(msg)
        _ = WriteMessage(os.Stdout, response) // best-effort; if pipe is gone, exit on next read
    }
}
```

### Pattern 2: XML Clone-and-Modify

**What:** Load the baseline `.Export.wacomxs` file, decode into a Go struct that preserves unmodified sections verbatim, update the `ScreenArea` subtrees, then re-encode.

**When to use:** `set_mapping` and `reset_mapping` command handlers.

**Key insight:** Go's `encoding/xml` does not have XPath. Instead, use a struct that models only the `InputScreenAreaArray` path using `xml:",innerxml"` for all other content. A simpler alternative is to unmarshal the full document into a nested struct — safe because the schema is well-known and the baseline file is controlled.

```go
// Source: [VERIFIED: pkg.go.dev/encoding/xml — struct tag syntax confirmed]
// Source: [VERIFIED: spike/baseline-reference.Export.wacomxs — schema]

// Targeted subtree types for the sections we modify.
// All other XML is preserved via xml.RawMessage or innerxml passthrough.

type ScreenOutputArea struct {
    Extent struct {
        X string `xml:"X"`
        Y string `xml:"Y"`
        Z string `xml:"Z"`
    } `xml:"Extent"`
    Origin struct {
        X string `xml:"X"`
        Y string `xml:"Y"`
        Z string `xml:"Z"`
    } `xml:"Origin"`
}

type ScreenArea struct {
    AreaType         string           `xml:"AreaType"`
    ScreenOutputArea ScreenOutputArea `xml:"ScreenOutputArea"`
    // Other children preserved via RawContent approach or full struct
}

// Full approach: unmarshal the entire file into a struct tree.
// For the planning phase, the exact struct layout is left to Plan 02-02.
// The key invariant: only ScreenArea fields are mutated; all other fields
// round-trip unchanged through marshal/unmarshal.
```

**Writing back:**
```go
// Preserve <?xml version="1.0" encoding="UTF-8"?> declaration:
out, err := xml.MarshalIndent(doc, "", "  ")
f.WriteString(xml.Header)
f.Write(out)
```

**File extension is mandatory:** must end in `.Export.wacomxs`. [VERIFIED: spike/SPIKE-RESULTS.md]

### Pattern 3: Windows Service Restart

**What:** Stop `WtabletServicePro`, wait until stopped, then start it again.

**When to use:** In `wacom/service.go`, called conditionally after XML write if testing (Plan 02-02) determines restart is needed.

```go
// Source: [VERIFIED: pkg.go.dev/golang.org/x/sys/windows/svc/mgr — function signatures]

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

    // Poll until stopped (timeout ~10s)
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

**Elevation:** Not required. Phase 1 confirmed `Wacom_TabletUserPrefs.exe` works without elevation; SCM stop/start of `WtabletServicePro` access rights need to be confirmed on test machine. [ASSUMED: non-elevated SCM restart is permitted — verify in Plan 02-02]

### Pattern 4: Structured Logging with slog

**What:** Open a log file at `%LOCALAPPDATA%\WacomBridge\logs\wacom-bridge.log`, create a `slog.Logger` with `JSONHandler`.

```go
// Source: [VERIFIED: pkg.go.dev/log/slog — JSONHandler in stdlib since Go 1.21]

package logging

import (
    "log/slog"
    "os"
    "path/filepath"
)

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

**Note:** slog does not include built-in log rotation. HOST-04 requires only basic logging; rotation is deferred to v2 (OPS-01).

### Pattern 5: WiX 4 Installer

**What:** Single `.wxs` file packaging the Go binary, two native messaging manifest JSON files, HKLM registry entries for Chrome and Edge, and a `MajorUpgrade` for future updates.

```xml
<!-- Source: [CITED: docs.firegiant.com/wix/tutorial/] -->
<!-- Source: [CITED: learn.microsoft.com/en-us/microsoft-edge/extensions/developer-guide/native-messaging] -->
<?xml version="1.0" encoding="UTF-8"?>
<Wix xmlns="http://wixtoolset.org/schemas/v4/wxs">
  <Package
    Name="WacomBridge"
    Manufacturer="Eurefirma"
    Version="1.0.0"
    UpgradeCode="GENERATE-A-NEW-GUID-HERE"
  >
    <MajorUpgrade DowngradeErrorMessage="A newer version of WacomBridge is already installed." />
    <MediaTemplate EmbedCab="yes" />

    <!-- Install to C:\Program Files\WacomBridge\ -->
    <StandardDirectory Id="ProgramFiles6432Folder">
      <Directory Id="INSTALLFOLDER" Name="WacomBridge" />
    </StandardDirectory>

    <ComponentGroup Id="WacomBridgeComponents" Directory="INSTALLFOLDER">
      <!-- Main binary -->
      <Component Id="MainExe" Guid="GENERATE-A-NEW-GUID-HERE">
        <File Source="wacom-bridge.exe" KeyPath="yes" />
      </Component>

      <!-- Chrome native messaging manifest -->
      <Component Id="ManifestChrome" Guid="GENERATE-A-NEW-GUID-HERE">
        <File Source="manifest-chrome.json" KeyPath="yes" />
        <RegistryKey Root="HKLM" Key="SOFTWARE\Google\Chrome\NativeMessagingHosts\com.eurefirma.wacombridge">
          <RegistryValue Type="string" Value="[INSTALLFOLDER]manifest-chrome.json" />
        </RegistryKey>
      </Component>

      <!-- Edge native messaging manifest -->
      <Component Id="ManifestEdge" Guid="GENERATE-A-NEW-GUID-HERE">
        <File Source="manifest-edge.json" KeyPath="yes" />
        <RegistryKey Root="HKLM" Key="SOFTWARE\Microsoft\Edge\NativeMessagingHosts\com.eurefirma.wacombridge">
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

**Build command:**
```bash
# Install WiX 4 CLI as dotnet tool (requires .NET SDK 6+)
dotnet tool install --global wix
# Build MSI
wix build installer/wacom-bridge.wxs -o wacom-bridge.msi
# Silent install
msiexec /i wacom-bridge.msi /quiet
```

**WOW6432Node note:** On 64-bit Windows, HKLM registry keys written from a 64-bit MSI go to the native (non-WOW) hive. Edge and Chrome search `HKLM\SOFTWARE\WOW6432Node\Microsoft\Edge\NativeMessagingHosts\` (32-bit view) first, then the 64-bit view. Writing to the non-WOW path (`SOFTWARE\Microsoft\Edge\...`) works because Edge searches both. WiX 4 with `ProgramFiles6432Folder` installs as 64-bit and writes to the native 64-bit hive — which Edge finds. [CITED: learn.microsoft.com — Edge registry search order documented]

### Anti-Patterns to Avoid

- **O_TEXT stdin/stdout:** The default Windows text mode converts `\n` bytes to `\r\n`, corrupting the 4-byte length prefix. Must set binary mode before first read/write.
- **Constructing minimal XML from scratch:** Phase 1 decision — always clone the existing preference file. Constructing minimal XML risks silently losing per-tablet settings.
- **Exiting on unrecoverable error:** D-12 requires staying running and returning `{"error": "...", "code": "..."}`. The extension must see the error; a crashed host produces a `Native host has exited` error in the browser with no useful detail.
- **PrefUtil in the production path:** D-04 prohibits this. PrefUtil's GUI dialog makes it production-unusable (confirmed in spike).
- **Writing `.xml` extension:** Must be `.Export.wacomxs`. The `.xml` extension causes a silent failure. [VERIFIED: spike/SPIKE-RESULTS.md]
- **Single `ArrayElement` update:** All three `ArrayElement` entries under `InputScreenAreaArray` must be updated. [VERIFIED: spike/baseline-reference.Export.wacomxs — 3 entries confirmed]

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Windows SCM service control | Custom `exec.Command("net stop ...")` | `golang.org/x/sys/windows/svc/mgr` | Proper SCM API; handles service dependencies, access rights, status polling |
| Structured logging | Custom log format | `log/slog` with `JSONHandler` | Stdlib since Go 1.21; machine-parseable JSON; no dependency |
| 4-byte framing | Custom byte manipulation | `encoding/binary.LittleEndian.Read/Write` | One-liner; correct endianness handling |
| JSON dispatch | Reflection-based router | Simple `switch msg["command"]` | Four commands total — a switch is sufficient and explicit |

**Key insight:** The native host is intentionally simple. Avoid framework-izing a 4-command switch statement.

---

## Common Pitfalls

### Pitfall 1: Windows Text Mode Corrupts the Binary Length Prefix

**What goes wrong:** The 4-byte little-endian uint32 length prefix contains bytes that the C runtime treats as newline characters (`0x0A`). In O_TEXT mode (the Windows default for stdin/stdout), these are expanded to `0x0D 0x0A`, which shifts all subsequent byte offsets and causes framing errors. Chrome reports `Error when communicating with the native messaging host`.

**Why it happens:** Windows Win32 pipes open in text mode by default. The Go runtime does not change this.

**How to avoid:** Add a `//go:build windows` init function that switches stdin and stdout to binary mode before the message loop starts. The exact mechanism depends on handle type:
- For console handles: `windows.SetConsoleMode`
- For pipe handles: the `_setmode` CRT function via `windows.NewLazySystemDLL("msvcrt.dll")` or by linking a small CGO shim.
- A practical Go-native alternative: use `os.NewFile` with `windows.Handle` and explicitly call `windows.SetFileMode` if available in the x/sys version.

**Warning signs:** Framing errors on messages containing `\n` bytes (common in any JSON with embedded strings); Chrome debug log shows `native messaging host` errors; works in testing without newlines but fails in production.

[CITED: learn.microsoft.com — "Windows-only: Make sure that the program's I/O mode is set to O_BINARY"]

### Pitfall 2: Registry Path and WOW6432Node

**What goes wrong:** The 32-bit Edge process searches `HKLM\SOFTWARE\WOW6432Node\Microsoft\Edge\NativeMessagingHosts\` before the 64-bit path. If the MSI writes the key only to the 64-bit hive, a 32-bit Edge installation will not find the host.

**Why it happens:** WiX 4 on a 64-bit machine with `ProgramFiles6432Folder` writes to the 64-bit registry view. A 32-bit process sees a different hive.

**How to avoid:** Edge's documented search order includes both WOW6432Node and native paths. Chrome and Edge both find the key if written to the **native 64-bit path** (`SOFTWARE\Microsoft\Edge\...`) because they also search there. However, add a second `RegistryKey` writing to `SOFTWARE\WOW6432Node\Microsoft\Edge\...` if testing on 32-bit Edge.

**Warning signs:** `Specified native messaging host not found` in Chrome/Edge debug log on 32-bit browser; registry key exists when viewed in regedit but browser can't find it.

[CITED: learn.microsoft.com — Edge registry search order documented; WOW6432Node listed as first in search]

### Pitfall 3: Native Messaging Manifest `allowed_origins` Uses Published Extension ID

**What goes wrong:** During development the extension is sideloaded and gets a temporary ID. The manifest `allowed_origins` entry uses that ID. After publishing to the Chrome Web Store or Edge Add-ons, the extension gets a new permanent ID. The host rejects connections.

**Why it happens:** The manifest JSON is baked into the MSI at build time. The extension ID is assigned by the store at first publication.

**How to avoid:** Reserve the extension ID in advance by submitting the extension to the store before building the production MSI. Both the Chrome Web Store and Edge Add-ons allow ID reservation. During development use the sideloaded ID; update `allowed_origins` in the manifest before MSI production build.

[CITED: learn.microsoft.com — "the extension ID of the published extension might differ from the ID that's used while sideloading"]

### Pitfall 4: Service Restart Access Rights

**What goes wrong:** The Go binary runs without elevation (D-15 confirmed by Phase 1). `mgr.Connect()` and `s.Control(svc.Stop)` on `WtabletServicePro` may fail with `Access is denied` if the service has restricted stop/start permissions.

**Why it happens:** Windows services have Security Descriptors controlling who may stop/start them. `WtabletServicePro` is a driver-level service; its default SD may restrict control to administrators.

**How to avoid:** Test in Plan 02-02. If stop/start is denied for non-elevated users:
  1. Return `{"error": "Service restart not permitted", "code": "ERR_SERVICE_ACCESS_DENIED"}` and stay running (D-12).
  2. Investigate whether direct XML write without restart causes the Wacom driver to pick up changes anyway (may be sufficient — unknown until tested).
  3. If restart is required but denied, escalate to user before adding any elevation mechanism.

**Warning signs:** `RestartWacomService` returns an error immediately on `Connect()` or `Control()`; error message contains "Access is denied".

### Pitfall 5: Wacom Preference File Path (UNKNOWN — Must Verify)

**What goes wrong:** The host writes the modified XML to the wrong path. The service reads from a different location. Mapping appears to succeed (file write returns no error) but stylus mapping does not change.

**Why it happens:** The exact path where `WtabletServicePro` reads user preferences from is not documented in sources accessible to this research session. The spike used PrefUtil as the intermediary, which knew the correct path. Direct XML write requires knowing that path explicitly.

**What is known (LOW confidence):**
- Wacom documentation mentions `%APPDATA%\WTablet\Wacom_Tablet.dat` and `Wacom_Tablet.bak` as the user preference files. [ASSUMED: the `.Export.wacomxs` that PrefUtil writes/reads is at or derived from this path]
- The spike's `baseline-local.Export.wacomxs` was stored in the spike folder on the test machine and passed to PrefUtil as an explicit path argument. PrefUtil handled writing to the correct internal location.
- Without PrefUtil, the host must know the path the service monitors for changes.

**Recommendation:** Plan 02-02 MUST include a discovery task: on the test Windows machine, use Process Monitor to observe which files `WtabletServicePro` reads/writes when PrefUtil imports a preference file. This identifies the correct write target for direct XML write.

**Warning signs:** File write succeeds (no error), `RestartWacomService` succeeds, but Wacom stylus mapping does not change.

[LOW confidence — unverified, must be investigated on test machine]

---

## Wacom Preference File — What Is Known

| Finding | Confidence | Source |
|---------|------------|--------|
| Extension MUST be `.Export.wacomxs` | HIGH | VERIFIED: spike/SPIKE-RESULTS.md — `.xml` silently fails |
| XML structure: `//InputScreenAreaArray/ArrayElement/ScreenArea` | HIGH | VERIFIED: spike/baseline-reference.Export.wacomxs + PowerShell script |
| 3 `ArrayElement` entries on test machine — all must be updated | HIGH | VERIFIED: spike/baseline-reference.Export.wacomxs |
| `AreaType=1` for custom region, `AreaType=0` for full screen | HIGH | VERIFIED: spike/SPIKE-RESULTS.md |
| Coordinates are physical pixels (Origin.X/Y = top-left, Extent.X/Y = width/height) | HIGH | VERIFIED: spike/SPIKE-RESULTS.md |
| Service name: `WtabletServicePro` | HIGH | VERIFIED: spike/SPIKE-RESULTS.md |
| Service restart was NOT required when using PrefUtil | HIGH | VERIFIED: spike/SPIKE-RESULTS.md — "PrefUtil notifies service directly" |
| Whether restart IS required for direct XML write | UNKNOWN | Not testable without Windows + Wacom device |
| Exact file path the service reads from for direct write | LOW | `%APPDATA%\WTablet\Wacom_Tablet.dat` mentioned in Wacom docs — unverified for `.Export.wacomxs` |

---

## Native Messaging Protocol — Specification

[VERIFIED: learn.microsoft.com — Edge native messaging documentation, confirmed identical to Chrome spec]

| Property | Value |
|----------|-------|
| Transport | stdin/stdout of spawned host process |
| Length header | 4-byte unsigned integer, **native byte order** (little-endian on x86/x64 Windows) |
| Body encoding | UTF-8 JSON |
| Max message host→browser | 1 MB |
| Max message browser→host | 4 GB (Edge doc) / 64 MiB (Chrome doc) |
| Host process lifecycle | Kept alive while `runtime.connectNative` port is open; exits when port is destroyed or browser closes pipe |
| On pipe close | Browser disconnects → stdin EOF → host should exit cleanly (`os.Exit(0)`) |
| First argument to host | Origin of caller: `chrome-extension://[extension-id]/` |
| Windows binary mode | **Required** — default O_TEXT mode corrupts `\n` bytes in length prefix |

**Native Messaging Manifest JSON format:**

```json
{
    "name": "com.eurefirma.wacombridge",
    "description": "Wacom Bridge — restricts stylus to PDF region",
    "path": "C:\\Program Files\\WacomBridge\\wacom-bridge.exe",
    "type": "stdio",
    "allowed_origins": [
        "chrome-extension://[CHROME-EXTENSION-ID]/",
        "chrome-extension://[EDGE-EXTENSION-ID]/"
    ]
}
```

**allowed_origins notes:**
- No wildcards permitted
- Chrome-extension URI scheme used for both Chrome and Edge
- Extension IDs assigned at publication — use sideloaded IDs during development, update before MSI release
- A single manifest file can list both Chrome and Edge extension IDs in `allowed_origins`; Chrome ignores Edge IDs and vice versa — OR use separate manifest files for each browser (simpler, required if IDs differ between browsers)

---

## Go Build for Windows

[VERIFIED: go.dev/wiki/WindowsCrossCompiling]

```bash
# Cross-compile from Linux/macOS to Windows AMD64
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o wacom-bridge.exe ./cmd/wacom-bridge/

# On Windows (native build)
go build -o wacom-bridge.exe ./cmd/wacom-bridge/
```

**CGO_ENABLED=0** is required for cross-compilation. On the Windows machine itself, CGO is not needed for this binary (all required Windows APIs are accessed via `golang.org/x/sys/windows` which uses `syscall` internally, not CGO).

**Result:** A single `.exe` with no runtime dependency — no .NET, no Go installation required on the target machine.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| PrefUtil (`/import`) | Direct XML write + conditional service restart | Phase 2 decision (spike confirmed PrefUtil GUI is production-unusable) | Eliminates GUI dialog; requires discovering correct write path |
| C# .NET 8 | Go | Phase 2 decision (D-01) | Single binary, no runtime dependency |
| `golang.org/x/exp/slog` | `log/slog` (stdlib) | Go 1.21 (2023) | Use stdlib; `x/exp/slog` is deprecated |

**Deprecated/outdated:**
- `golang.org/x/exp/slog`: superseded by `log/slog` in stdlib. Do not use.
- PrefUtil at runtime: explicitly prohibited (D-04).

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Non-elevated process can stop/start `WtabletServicePro` via SCM | Service Restart Pattern | `RestartWacomService` returns Access Denied; need alternative (no restart path or elevation) |
| A2 | Wacom service reads from `%APPDATA%\WTablet\` or an adjacent path for direct XML write | Wacom Preference File paths | XML written to wrong location; stylus mapping never applies |
| A3 | `SetConsoleMode` or equivalent is sufficient to switch pipe handles to binary mode in Go on Windows | Pattern 1 (binary mode init) | Length prefix corruption on messages containing `0x0A` bytes |
| A4 | A single native messaging manifest JSON listing both Chrome and Edge extension IDs works (i.e., Chrome-extension URI scheme is valid in Edge `allowed_origins`) | Pattern 5 WiX | If Edge requires its own manifest format, a second manifest file is needed |
| A5 | `WtabletServicePro` does NOT hot-reload the XML file (i.e., a restart IS required after direct write) | D-03 — conditional restart | If service hot-reloads, restart adds unnecessary latency |

---

## Open Questions (PARTIALLY RESOLVED)

1. **Does direct XML write require a service restart, or does WtabletServicePro hot-reload?**
   - What we know: PrefUtil notifies the service directly (no restart needed when using PrefUtil). Direct write bypasses this notification mechanism.
   - What's unclear: Whether `WtabletServicePro` watches the preference file via `ReadDirectoryChangesW` or only reads on explicit notification/restart.
   - Recommendation: Plan 02-02 must include a test task: write the XML directly, wait 2 seconds, check if mapping applied. If not, restart the service and check again.
   - **DEFERRED: resolved by Plan 02-02 Task 1 Process Monitor checkpoint (requires physical Windows + Wacom device)**

2. **What is the exact path of the preference file the Wacom service reads?**
   - What we know: `%APPDATA%\WTablet\Wacom_Tablet.dat` is the documented user preference location. The `.Export.wacomxs` format is used by PrefUtil.
   - What's unclear: Whether `WtabletServicePro` reads `.Export.wacomxs` files directly, or whether PrefUtil converts them to the `.dat` binary format before writing.
   - Recommendation: Use Process Monitor on the test machine during a PrefUtil `/import` to trace exactly which files are read/written. This must be the first task in Plan 02-02.
   - **DEFERRED: resolved by Plan 02-02 Task 1 Process Monitor checkpoint (requires physical Windows + Wacom device)**

3. **What is the exact Go mechanism to set binary mode on piped stdin/stdout on Windows?**
   - What we know: The standard C approach is `_setmode(fileno(stdin), O_BINARY)`. `SetConsoleMode` applies to console handles, not pipe handles.
   - What's unclear: Whether `golang.org/x/sys/windows` exposes a clean way to call `_setmode` on `os.Stdin.Fd()` without CGO.
   - Recommendation: Research `windows.NewLazySystemDLL("msvcrt.dll")` pattern for calling `_setmode` from pure Go, or use a `//go:linkname` trick. Test on Windows before finalizing Plan 02-01.
   - **RESOLVED: Plan 02-01 implements `msvcrt.dll _setmode` via `windows.NewLazySystemDLL` in a `//go:build windows` init function — `SetConsoleMode` is NOT used for pipe handles**

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | Build | Yes (Linux) | go1.26.2 | — |
| WiX 4 CLI (`wix`) | Plan 02-03 (installer build) | Unknown (Windows only) | — | Install via `dotnet tool install --global wix` on Windows build machine |
| .NET SDK 6+ | WiX 4 dotnet tool | Unknown (Windows only) | — | Required by WiX 4 |
| Windows test machine + Wacom One M | Plan 02-02 (Wacom integration testing) | Required — cannot test without | — | No fallback; physical device required |
| `WtabletServicePro` service | Plan 02-02 (service restart testing) | Available on test machine | Unknown version | — |

**Missing dependencies with no fallback:**
- Physical Windows test machine with Wacom One M connected — required to validate direct XML write (Open Question 2) and service restart behavior (Open Question 1).

**Missing dependencies with fallback:**
- WiX 4: installable via `dotnet tool install --global wix` on the Windows build machine.

---

## Sources

### Primary (HIGH confidence)
- [CITED: learn.microsoft.com/en-us/microsoft-edge/extensions/developer-guide/native-messaging] — Native Messaging protocol spec (byte order, size limits, manifest format, registry paths, binary mode requirement)
- [CITED: developer.chrome.com/docs/extensions/develop/concepts/native-messaging] — Chrome Native Messaging spec
- [VERIFIED: pkg.go.dev/golang.org/x/sys/windows/svc/mgr] — Windows SCM Go API function signatures
- [VERIFIED: pkg.go.dev/log/slog] — stdlib slog, confirmed in Go 1.21+
- [VERIFIED: spike/SPIKE-RESULTS.md] — XML schema, XPath, service name, extension requirement, coordinate system
- [VERIFIED: spike/baseline-reference.Export.wacomxs] — Full XML schema, 3 ArrayElement entries confirmed

### Secondary (MEDIUM confidence)
- [CITED: docs.firegiant.com/wix/tutorial/] — WiX 4 Package element structure, RegistryKey/RegistryValue syntax
- [CITED: go.dev/wiki/WindowsCrossCompiling] — Cross-compilation build flags
- [CITED: pkg.go.dev/encoding/binary] — LittleEndian encoding for length prefix
- [CITED: pkg.go.dev/encoding/xml] — XML struct tags, MarshalIndent, xml.Header constant

### Tertiary (LOW confidence)
- Wacom `%APPDATA%\WTablet\Wacom_Tablet.dat` preference file path — mentioned in Wacom dev support forum results but not directly accessible (403); not verified for `.Export.wacomxs` write target

---

## Metadata

**Confidence breakdown:**
- Native Messaging protocol: HIGH — verified from official Microsoft Edge + Chrome docs
- Go standard library stack: HIGH — verified from pkg.go.dev
- Windows SCM API: HIGH — verified from golang.org/x/sys source
- WiX 4 installer: MEDIUM — structure confirmed from firegiant docs; specific RegistryValue default value syntax from secondary sources
- Wacom file path for direct write: LOW — not accessible from available sources; requires test machine investigation
- Service restart access rights: LOW — theoretical; requires test machine verification

**Research date:** 2026-04-30
**Valid until:** 2026-05-30 (stable tech stack; Wacom driver behavior findings should be validated against actual device)
