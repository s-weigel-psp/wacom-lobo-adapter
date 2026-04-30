# Phase 1 Spike Results: Wacom Mapping via PowerShell

**Executed by:** [name]
**Date:** [date]
**Test machine:** [OS version, Wacom driver version]
**Tablet:** Wacom One M — connected: [yes/no]
**Status:** [PASS / FAIL / PARTIAL]

---

## Working Method

**Does PowerShell + PrefUtil scripting work to restrict the Wacom stylus to an arbitrary screen region?**
[YES / NO / PARTIAL — describe]

**Mechanism:**
Clone spike/baseline-local.Export.wacomxs → modify screen-mapping element via Select-Xml XPath → import modified XML via PrefUtil.exe /import → Wacom driver applies region restriction.

**GUI Dialog Note:** Every invocation of `PrefUtil.exe /import` opens a native Windows dialog requiring the user to click OK. The `/silent` flag has NO effect on `/import`. This is a known PrefUtil limitation (documented in Plan 01-01). Each test case will prompt one dialog click.

**Confidence:** [HIGH / MEDIUM / LOW] — [reason]

---

## Binary Path and Invocation

**Executable found at:**
```
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

**Note:** `/silent` flag suppresses the help-screen window only; it does NOT suppress the GUI dialog on `/import` or `/export`.

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
```
//InputScreenAreaArray/ArrayElement/ScreenArea
```

No `-Namespace` parameter needed for `Select-Xml` — confirmed no namespace on root element.

**Multiple tablet sections:** yes — test machine had 3 `ArrayElement` entries inside `InputScreenAreaArray`; all are updated by Set-WacomMapping.ps1 for consistency.

---

## Service Names

**Services found on test machine:**

| Name | DisplayName | Status |
|------|-------------|--------|
| `WtabletServicePro` | Wacom Professional Service | Running |

**Service restart required for mapping to take effect:** [YES — must restart 'WtabletServicePro'] / [NO — PrefUtil notifies service directly] / [UNKNOWN — could not confirm]

**Command to restart (if required):**
```powershell
Restart-Service -Name 'WtabletServicePro' -Force
```

---

## Measured Latency (SPIKE-02)

**Test: Three consecutive mapping changes (Set-WacomMapping.ps1 invocations via run-tests.ps1 TC-04)**

| Run | Command | Elapsed (ms) |
|-----|---------|-------------|
| 1 | `.\Set-WacomMapping.ps1 -X 0 -Y 0 -Width 960 -Height 1080` | [ms] |
| 2 | `.\Set-WacomMapping.ps1 -X 960 -Y 0 -Width 960 -Height 1080` | [ms] |
| 3 | `.\Set-WacomMapping.ps1 -X 240 -Y 270 -Width 1440 -Height 540` | [ms] |

**Mean latency:** [ms]
**Maximum latency:** [ms]
**SPIKE-02 result:** [PASS — all runs < 3000 ms] / [FAIL — run N exceeded 3000 ms]

Note: Latency measured via `Measure-Command` wrapping `$proc | Wait-Process` inside run-tests.ps1 (TC-04).
`Start-Process -Wait` was NOT used — it has a 1-second poll floor (PowerShell issue #24709).

---

## Admin Rights Requirement (SPIKE-01 dependency)

**PrefUtil import from non-elevated prompt:**
- Exit code: [0 / other]
- Mapping applied: [yes / no]
- Result: [elevation required / NOT required]

**PrefUtil import from elevated prompt:**
- Exit code: [0 / other]
- Mapping applied: [yes / no]

**Impact for Phase 2:**
[If elevation required: "Native host must run elevated — consider Windows service or scheduled task"]
[If not required: "Standard process invocation from browser extension native host is sufficient"]

---

## Test Case Results

### TC-01: Left Half Mapping (SPIKE-01, SPIKE-03)
```
Set-WacomMapping.ps1 -X 0 -Y 0 -Width 960 -Height 1080
```
- Script exited with code: [0 / other]
- PrefUtil dialog appeared: [yes / no]
- Stylus restricted to left half: [YES / NO / PARTIAL]
- Notes: [any observations]

### TC-02: Right Half Mapping (SPIKE-01, SPIKE-03)
```
Set-WacomMapping.ps1 -X 960 -Y 0 -Width 960 -Height 1080
```
- Script exited with code: [0 / other]
- PrefUtil dialog appeared: [yes / no]
- Stylus restricted to right half: [YES / NO / PARTIAL]
- Notes:

### TC-03: Centre Region (SPIKE-01, SPIKE-03)
```
Set-WacomMapping.ps1 -X 240 -Y 270 -Width 1440 -Height 540
```
- Script exited with code: [0 / other]
- PrefUtil dialog appeared: [yes / no]
- Stylus restricted to centre region: [YES / NO / PARTIAL]
- Notes:

### TC-04: Three Consecutive Changes — Latency (SPIKE-02)
See Measured Latency table above.
(run-tests.ps1 prints per-run ms and SPIKE-02 PASS/FAIL automatically)

### TC-05: Reset to Full-Screen (SPIKE-04)
```
Reset-WacomMapping.ps1
```
- Script exited with code: [0 / other]
- PrefUtil dialog appeared: [yes / no]
- Stylus covers full screen after reset: [YES / NO / PARTIAL]
- Notes:

### TC-06: Optional — Multi-Monitor (if second display available)
```
Set-WacomMapping.ps1 -X [secondary display offset X] -Y 0 -Width [secondary width] -Height [secondary height]
```
- Performed: [YES / NO — single monitor only]
- Result: [if performed]

---

## DPI / Coordinate System Finding

**Display DPI scaling on test machine:** [100% / 125% / 150% / other]
**Physical display resolution:** [e.g., 1920×1080]
**Logical display resolution (if scaled):** [e.g., 1536×864 at 125%]
**Coordinates accepted by PrefUtil:** [physical pixels / logical pixels / unknown]

**Finding:** [e.g., "Physical pixels — Set-WacomMapping.ps1 -X 0 -Y 0 -Width 960 -Height 1080 maps to physical left half of 1920×1080 display regardless of DPI scaling setting"]

**Impact for Phase 2/3:** [e.g., "Extension must send physical pixel coordinates — DPR multiplication required in content script (EXT-02)"]

---

## Recommendation for Phase 2

**Proceed with C# port of this PowerShell approach:** [YES / NO]

**Recommendation:**
[One clear sentence: e.g., "Port the clone-and-modify-XML + PrefUtil invocation approach to C# .NET 8 — all spike objectives passed, latency is well under 3s, and no elevation is required."]

**Phase 2 implementation notes:**
- Binary path to hardcode (or discover via registry): `C:\Program Files\Tablet\Wacom\PrefUtil.exe`
- Import flag to use: `/import`
- File extension: `.Export.wacomxs` (MUST match — `.xml` silently fails)
- XML element to target: `ScreenArea` (inside `//InputScreenAreaArray/ArrayElement/ScreenArea`)
- Coordinate model: child XML elements with text content (not attributes); use `Origin/X`, `Origin/Y`, `Extent/X`, `Extent/Y`
- AreaType: set to `1` for custom region (was `0` for full screen)
- Elevation requirement: [required / not required — see Admin Rights section above]
- Service restart required: [yes / no — see Service Names section above]
- XPath expression for C# XmlDocument.SelectNodes(): `//InputScreenAreaArray/ArrayElement/ScreenArea`
- PrefUtil headless limitation: GUI dialog appears on every `/import` — Phase 2 MUST find an alternative silent invocation mechanism (see 01-01-SUMMARY.md Deviations § Major Deviation)

---

## Issues Encountered

[Any deviations from expected behavior, error messages, workarounds applied]

---

*Phase 1 spike completed: [date]*
*Authored by: [name]*
