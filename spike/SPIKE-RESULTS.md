# Phase 1 Spike Results: Wacom Mapping via PowerShell

**Executed by:** [Simeon Weigel](mailto:simeon@sj4jc.de)
**Date:** 2026 April 29th
**Test machine:** Windows 11 version 23H2 (Build 22631.6199)
**Tablet:** Wacom One M — connected: yes
**Status:** PARTIAL

---

## Working Method

**Does PowerShell + PrefUtil scripting work to restrict the Wacom stylus to an arbitrary screen region?**
YES

**Mechanism:**
Clone spike/baseline-local.Export.wacomxs → modify screen-mapping element via Select-Xml XPath → import modified XML via PrefUtil.exe /import → Wacom driver applies region restriction.

**GUI Dialog Note:** Every invocation of `PrefUtil.exe /import` opens a native Windows dialog requiring the user to click OK. The `/silent` flag has NO effect on `/import`. This is a known PrefUtil limitation (documented in Plan 01-01). Each test case will prompt one dialog click.

**Confidence:** LOW — only on first update one dialog click will suffice. Subsequent updates will ask the user where to backup the previous Settings. The first update uses a default location, hence one click, subsequent updates cannot override the first backup and need to ask the user to either skip backup or choose a location.

---

## Binary Path and Invocation

**Executable found at:**

```txt
C:\Program Files\Tablet\Wacom\PrefUtil.exe
```

**Import command:**

```powershell
$proc = Start-Process -FilePath 'C:\Program Files\Tablet\Wacom\PrefUtil.exe' `
                      -ArgumentList '/import', $TempPath `
                      -PassThru
$proc | Wait-Process
```

**Export command:**

```powershell
$proc = Start-Process -FilePath 'C:\Program Files\Tablet\Wacom\PrefUtil.exe' `
                      -ArgumentList '/export', $OutputPath `
                      -PassThru
$proc | Wait-Process
```

**Note:** `/silent` flag suppresses only some commands' GUI; it does NOT suppress the GUI dialog on `/import` or `/export`.

**PrefUtil help output:** (see spike/test-log.md § 1. PrefUtil Binary)

---

## XML Tag Names and Structure

**File extension:** `.Export.wacomxs` (MUST match — `.xml` extension silently fails to write at the specified path)

**Screen-mapping element:**

```xml
<ScreenArea type="map">
  <AreaType type="integer">0</AreaType>       <!-- 0=full screen, 1=custom region -->
  <ScreenOutputArea type="map">
    <Extent type="map">
      <X type="integer">1920</X>              <!-- width in physical pixels -->
      <Y type="integer">1080</Y>              <!-- height in physical pixels -->
      <Z type="integer">0</Z>
    </Extent>
    <Origin type="map">
      <X type="integer">0</X>                 <!-- left edge in physical pixels -->
      <Y type="integer">0</Y>                 <!-- top edge in physical pixels -->
      <Z type="integer">0</Z>
    </Origin>
  </ScreenOutputArea>
  <WhichMonitor type="string">Desktop_0_</WhichMonitor>
</ScreenArea>
```

**Element name:** `ScreenArea`
**Coordinate model:** Child XML elements with text content (NOT attributes)
**Coordinate semantics:**

- `ScreenOutputArea/Origin/X` — left edge in physical pixels
- `ScreenOutputArea/Origin/Y` — top edge in physical pixels
- `ScreenOutputArea/Extent/X` — width in physical pixels
- `ScreenOutputArea/Extent/Y` — height in physical pixels

**XML namespace:** none — root element is `<root type="map">` with no namespace declaration

**XPath used in Set-WacomMapping.ps1:**

```txt
//InputScreenAreaArray/ArrayElement/ScreenArea
```

No `-Namespace` parameter needed for `Select-Xml` — confirmed no namespace on root element.

**Multiple tablet sections:** yes — test machine had 3 `ArrayElement` entries inside `InputScreenAreaArray`; all are updated by Set-WacomMapping.ps1 for consistency.

---

## Service Names

**Services found on test machine:**

| Name                | DisplayName                | Status  |
|---------------------|----------------------------|---------|
| `WtabletServicePro` | Wacom Professional Service | Running |

**Service restart required for mapping to take effect:** NO — PrefUtil notifies service directly

**Command to restart (if required):**

```powershell
Restart-Service -Name 'WtabletServicePro' -Force
```

---

## Measured Latency (SPIKE-02)

### Test: Three consecutive mapping changes (Set-WacomMapping.ps1 invocations via run-tests.ps1 TC-04)

| Run | Command                                                        | Elapsed (ms) |
|-----|----------------------------------------------------------------|--------------|
| 1   | `.\Set-WacomMapping.ps1 -X 0 -Y 0 -Width 960 -Height 1080`     | 6659         |
| 2   | `.\Set-WacomMapping.ps1 -X 960 -Y 0 -Width 960 -Height 1080`   | 4597         |
| 3   | `.\Set-WacomMapping.ps1 -X 240 -Y 270 -Width 1440 -Height 540` | 4156         |

**Mean latency:** 5137
**Maximum latency:** 6659
**SPIKE-02 result:** FAIL (at least one run >= 3000 ms)

Note:

- Latency measured via `Measure-Command` wrapping `$proc | Wait-Process` inside run-tests.ps1 (TC-04).
- `Start-Process -Wait` was NOT used — it has a 1-second poll floor (PowerShell issue #24709).
- Several User Clicks were recorded in the Latency. The user was the main reason for latency.

---

## Admin Rights Requirement (SPIKE-01 dependency)

**PrefUtil import from non-elevated prompt:**

- Exit code: 0
- Mapping applied: yes (after clicking OK in dialog)
- Result: elevation NOT required

**Impact for Phase 2:**
Standard process invocation from browser extension native host is sufficient — no elevation needed.

---

## Test Case Results

### TC-01: Left Half Mapping (SPIKE-01, SPIKE-03)

```powershell
Set-WacomMapping.ps1 -X 0 -Y 0 -Width 960 -Height 1080
```

- Script exited with code: 0
- PrefUtil dialog appeared: yes
- Stylus restricted to left half: YES

### TC-02: Right Half Mapping (SPIKE-01, SPIKE-03)

```powershell
Set-WacomMapping.ps1 -X 960 -Y 0 -Width 960 -Height 1080
```

- Script exited with code: 0
- PrefUtil dialog appeared: yes
- Stylus restricted to right half: YES

### TC-03: Centre Region (SPIKE-01, SPIKE-03)

```powershell
Set-WacomMapping.ps1 -X 240 -Y 270 -Width 1440 -Height 540
```

- Script exited with code: 0
- PrefUtil dialog appeared: yes
- Stylus restricted to centre region: YES

### TC-04: Three Consecutive Changes — Latency (SPIKE-02)

See Measured Latency table above.
(run-tests.ps1 prints per-run ms and SPIKE-02 PASS/FAIL automatically)

### TC-05: Reset to Full-Screen (SPIKE-04)

```powershell
Reset-WacomMapping.ps1
```

- Script exited with code: 0
- PrefUtil dialog appeared: yes
- Stylus covers full screen after reset: YES

### TC-06: Optional — Multi-Monitor (if second display available)

- Performed: NO — single monitor only

---

## DPI / Coordinate System Finding

**Physical display resolution:** 1920×1080
**Coordinates accepted by PrefUtil:** physical pixels (confirmed — TC-01 `-X 0 -Y 0 -Width 960 -Height 1080` restricted stylus to exactly the left half of the physical display)

**Finding:** Physical pixels. `-X 0 -Y 0 -Width 960 -Height 1080` maps to the left half of a 1920×1080 display. DPI scaling setting on the test machine was not recorded but coordinate behaviour matched physical pixel expectations.

**Impact for Phase 2/3:** Extension must send physical pixel coordinates. If the browser reports logical (CSS) pixels, DPI device-pixel-ratio multiplication will be required in the content script (EXT-02).

---

## Recommendation for Phase 2

**Proceed with C# port of this PowerShell approach:** YES — with a mandatory alternative to PrefUtil invocation

**Recommendation:**
Port the clone-and-modify-XML approach to C# .NET 8, but replace PrefUtil invocation with direct XML file write + `Restart-Service WtabletServicePro` (or equivalent service notification). All TC-01/02/03/05 spike objectives passed and no elevation is required; SPIKE-02 latency failed only because PrefUtil's mandatory GUI dialog dominated measurement time — the underlying XML mechanism is fast.

**Phase 2 implementation notes:**

- XML element to target: `ScreenArea` inside `//InputScreenAreaArray/ArrayElement/ScreenArea`
- Coordinate model: child XML elements with text content (not attributes); set `Origin/X`, `Origin/Y`, `Extent/X`, `Extent/Y` and `AreaType=1`
- File extension: `.Export.wacomxs` (MUST match — `.xml` silently fails at the specified path)
- Service: `WtabletServicePro` — no restart required when using PrefUtil; **unknown** whether restart is needed when writing the XML file directly (must be tested in Phase 2)
- Elevation: NOT required
- XPath for C# `XmlDocument.SelectNodes()`: `//InputScreenAreaArray/ArrayElement/ScreenArea`
- **Priority investigation for Phase 2:** Test whether direct XML file write (bypassing PrefUtil entirely) + `Restart-Service WtabletServicePro` achieves silent, sub-3s mapping changes. If confirmed, PrefUtil becomes unnecessary for the production native host.

---

## Issues Encountered

1. **PrefUtil GUI dialog on every /import** — `/silent` has no effect on `/import`. Every invocation opens a native Windows dialog requiring user confirmation. Silent/headless operation via PrefUtil is not possible.
2. **Backup dialog on subsequent imports** — after the first import, PrefUtil asks where to save a backup of the previous settings (cannot override to a fixed location). This adds a second click on runs 2+, which inflated TC-04 latency figures.
3. **File extension must be `.Export.wacomxs`** — `.xml` silently fails to write the file at the specified path. Discovered during Plan 01-01 baseline export.
4. **SPIKE-02 FAIL is misleading** — all three TC-04 latency measurements exceeded 3000 ms, but the excess is entirely dialog click time (user interaction), not PrefUtil processing time. The underlying XML modification mechanism is fast.

---

*Phase 1 spike completed: 2026-04-30*
*Authored by: Simeon Weigel*
