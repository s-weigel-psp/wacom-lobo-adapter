---
phase: 02-native-messaging-host
reviewed: 2026-05-04T00:00:00Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - docs/protocol.md
  - go.mod
  - go.sum
  - cmd/wacom-bridge/main.go
  - internal/messaging/host.go
  - internal/messaging/init_windows.go
  - internal/logging/logging.go
  - internal/state/state.go
  - internal/wacom/xml.go
  - internal/wacom/service.go
  - installer/wacom-bridge.wxs
  - installer/manifest-chrome.json
  - installer/manifest-edge.json
findings:
  critical: 2
  warning: 4
  info: 3
  total: 9
status: issues_found
---

# Phase 02: Code Review Report

**Reviewed:** 2026-05-04
**Depth:** standard
**Files Reviewed:** 13
**Status:** issues_found

## Summary

Reviewed the complete Phase 2 native messaging host implementation: Go binary, Wacom XML manipulation, Windows service control, Native Messaging framing, logging, installer WXS, and protocol manifests.

The core framing and dispatch logic is clean. The XML rewriter is carefully structured, and the coordinate validation guards are reasonable. Two critical issues were found: a command injection vector via the PrefUtil temp path, and a missing `_setmode` return-value check that can silently corrupt the stdio pipes. Four warnings cover logic gaps that could cause incorrect behavior or silent failures in the field.

---

## Critical Issues

### CR-01: Command injection via unsanitized TEMP path passed to `exec.Command`

**File:** `internal/wacom/xml.go:72,150`

**Issue:** `os.Getenv("TEMP")` is written directly into `tempPath` and then passed as an argument to `exec.Command(wacomPrefPath, "/import", xmlPath)`. On Windows, `TEMP` is a user-controlled environment variable. An attacker who can set `TEMP` to a path containing shell metacharacters or spaces (e.g., `C:\foo bar` or a path with a trailing `/import` segment) can redirect PrefUtil to load an attacker-controlled file. While `exec.Command` does not invoke a shell so classic shell injection does not apply, the bigger risk is a symlink/path confusion attack: `TEMP` could be set to a directory the attacker controls, and `wacom-mapping-temp.Export.wacomxs` written there could be intercepted or pre-placed to cause PrefUtil to import a malicious preference XML.

Additionally, the file is written with a **fixed, predictable name** (`wacom-mapping-temp.Export.wacomxs`), creating a classic TOCTOU race between `os.WriteFile` and `runPrefUtil`. A low-privileged local process could replace the file between write and import.

**Fix:** Use `os.CreateTemp` to obtain a unique, unguessable temp path rather than a hardcoded filename. Verify the resolved path stays within the expected directory before passing it to PrefUtil.

```go
// In SetMapping / ResetMapping — replace fixed-path WriteFile+runPrefUtil with:
tmpFile, err := os.CreateTemp("", "wacom-bridge-*.Export.wacomxs")
if err != nil {
    return errMap("failed to create temp file: "+err.Error(), "ERR_XML_WRITE")
}
tmpPath := tmpFile.Name()
if _, err := tmpFile.Write(modified); err != nil {
    tmpFile.Close()
    os.Remove(tmpPath)
    return errMap("failed to write temp file: "+err.Error(), "ERR_XML_WRITE")
}
tmpFile.Close()
defer os.Remove(tmpPath)

if err := runPrefUtil(tmpPath); err != nil { ... }
```

---

### CR-02: `_setmode` return value not checked — silent pipe corruption on failure

**File:** `internal/messaging/init_windows.go:25-27`

**Issue:** `setmode.Call(...)` returns `(r1, r2, err)`. The Windows `_setmode` CRT function returns `-1` on failure (e.g., if the file descriptor is invalid or the DLL call fails). The current code discards all return values with `setmode.Call(...)` and no assignment. If `_setmode` fails silently, stdin/stdout remain in text mode, and every JSON body containing a `0x0A` byte will be corrupted — producing malformed length prefixes that are extraordinarily hard to diagnose. The process will not crash; it will just emit or receive corrupt data.

**Fix:** Check the return value and log a warning so failures surface in the log file.

```go
r1, _, err := setmode.Call(uintptr(os.Stdin.Fd()), 0x8000)
if r1 == ^uintptr(0) { // -1 cast to uintptr
    // Log to stderr — logger is not yet open at init() time
    fmt.Fprintf(os.Stderr, "wacom-bridge: _setmode(stdin, O_BINARY) failed: %v\n", err)
}
r1, _, err = setmode.Call(uintptr(os.Stdout.Fd()), 0x8000)
if r1 == ^uintptr(0) {
    fmt.Fprintf(os.Stderr, "wacom-bridge: _setmode(stdout, O_BINARY) failed: %v\n", err)
}
```

---

## Warnings

### WR-01: XML attribute values written without escaping — potential malformed output

**File:** `internal/wacom/xml.go:336-341`

**Issue:** In `rewriteScreenArea`, the closing tag and opening tag with attributes are assembled by manual string concatenation:

```go
tag.WriteString(fmt.Sprintf(` %s="%s"`, a.Name.Local, a.Value))
```

`a.Value` is taken directly from the parsed XML attribute value without XML-escaping it. If the baseline file contains an attribute value with `"` (a valid XML escape in the source), or if it was parsed from `'`-quoted attributes, the encoder will have unescaped it. Writing it back bare with `"` delimiters can produce malformed XML (e.g., `value="foo"bar"`). The standard library `xml.EscapeText` or using `xml.Encoder` for the whole element would avoid this.

**Fix:** Use `xml.EscapeText` when writing attribute values, or reconstruct the opening tag via `enc.EncodeToken(start)` rather than raw string assembly.

```go
// Replace manual tag.WriteString loop with:
var tagBuf bytes.Buffer
tagEnc := xml.NewEncoder(&tagBuf)
_ = tagEnc.EncodeToken(start)
_ = tagEnc.Flush()
openTag := strings.TrimSuffix(tagBuf.String(), "></"+start.Name.Local+">")
// ... then close with inner content
```

---

### WR-02: `isAccessDenied` heuristic — incorrect error classification

**File:** `internal/wacom/service.go:77-79`

**Issue:** The `isAccessDenied` helper checks whether the lower-cased error message string contains `"access"`. This will false-positive on unrelated error messages (e.g., `"cannot access service"`, `"network access error"`) and will false-negative on the actual Windows error `ERROR_ACCESS_DENIED` (0x5) if the error is wrapped in a way that produces a message like `"The operation requires elevation"` or `"Access is denied."` (capital A). The correct approach for Windows errors is to use `errors.As` against `syscall.Errno` and compare to `windows.ERROR_ACCESS_DENIED`.

**Fix:**

```go
import "golang.org/x/sys/windows"

func isAccessDenied(err error) bool {
    var errno syscall.Errno
    if errors.As(err, &errno) {
        return errno == windows.ERROR_ACCESS_DENIED
    }
    return false
}
```

---

### WR-03: Zero-value `State` exported fields readable without lock from outside the package

**File:** `internal/state/state.go:8-15`

**Issue:** The struct fields `Mapped`, `X`, `Y`, `Width`, `Height` are exported (capital letter). Any code outside the `state` package can read or write these fields directly, bypassing the mutex. Currently `main.go` only calls the safe methods, but the exported fields are a footgun — future code could accidentally read `st.Mapped` directly without acquiring the lock, leading to a data race.

**Fix:** Unexport the fields (`mapped`, `x`, `y`, `width`, `height`). The `StatusResponse` method already provides the only needed read access.

```go
type State struct {
    mu     sync.Mutex
    mapped bool
    x      int
    y      int
    width  int
    height int
}
```

---

### WR-04: `WriteMessage` errors silently ignored in `MessageLoop`

**File:** `internal/messaging/host.go:73,77`

**Issue:** Both `WriteMessage` calls in `MessageLoop` use `_ = WriteMessage(...)`, discarding write errors entirely. If `os.Stdout` becomes unwritable (e.g., the browser process was killed mid-write), the host will loop silently, reading future messages and discarding all responses. This can cause the browser extension to hang waiting for a reply it will never receive and prevents detection of a broken pipe.

**Fix:** Check write errors and exit on failure, consistent with the protocol spec (Section 7: "Unrecoverable read error → exit 1"):

```go
if err := WriteMessage(os.Stdout, resp); err != nil {
    os.Exit(1)
}
```

---

## Info

### IN-01: `go.mod` declares `go 1.25.0` — non-existent Go version

**File:** `go.mod:3`

**Issue:** Go 1.25 does not exist as of the review date (latest stable is 1.23.x). This is likely a typo for `go 1.21.0` or `go 1.22.0` (the `log/slog` package was stabilized in 1.21). While `go build` typically tolerates this, the toolchain directive can cause `go` commands to attempt to download a non-existent toolchain version in module-aware mode, leading to confusing `go: toolchain not available` errors in CI.

**Fix:** Set the directive to the actual minimum required version: `go 1.21.0` (first version with `log/slog` in the standard library).

---

### IN-02: `manifest-chrome.json` and `manifest-edge.json` contain a `_dev_note` non-standard field

**File:** `installer/manifest-chrome.json:9`, `installer/manifest-edge.json:9`

**Issue:** The `_dev_note` field is not part of the Native Messaging manifest specification. Chrome and Edge ignore unknown fields, so this does not cause a functional problem. However, leaving developer notes in a production-shipped JSON file is a minor quality issue — the note may expose internal development process details to anyone who inspects the installed manifest.

**Fix:** Remove `_dev_note` from both files before production build, or add it to the pre-production checklist alongside the extension ID replacement step (already noted in `wacom-bridge.wxs` comments).

---

### IN-03: `wacom-bridge.wxs` installs manifests to `INSTALLFOLDER` root — registry path uses `[INSTALLFOLDER]` without trailing backslash separator

**File:** `installer/wacom-bridge.wxs:51,62`

**Issue:** The registry value is set to `[INSTALLFOLDER]manifest-chrome.json` (line 51) and `[INSTALLFOLDER]manifest-edge.json` (line 62). The WiX `[INSTALLFOLDER]` property always includes a trailing backslash for directories, so the concatenation produces the correct path `C:\Program Files\WacomBridge\manifest-chrome.json`. This is technically correct. However, it is fragile and non-obvious. The more robust and idiomatic WiX pattern is to use the `[#FileId]` syntax which resolves to the installed file's full path automatically, eliminating the dependency on trailing-backslash convention.

**Fix (low priority):** Replace with `[#ManifestChromeJson]` and `[#ManifestEdgeJson]` using the `File/@Id` attribute, or at minimum add a comment confirming the trailing backslash behavior is intentional.

---

_Reviewed: 2026-05-04_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
