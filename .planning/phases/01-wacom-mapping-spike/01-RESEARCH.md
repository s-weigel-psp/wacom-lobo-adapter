# Phase 1: Wacom Mapping Spike - Research

**Researched:** 2026-04-29
**Domain:** Windows Wacom driver scripting via PowerShell + preference XML manipulation
**Confidence:** MEDIUM — official docs are gated behind 403s; findings assembled from developer community articles, official developer-support.wacom.com search summaries, and verified PowerShell documentation.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** `Set-WacomMapping.ps1` modifies a **cloned copy** of the baseline XML rather than constructing minimal XML from scratch. Full profile is preserved.
- **D-02:** The script locates the screen-mapping element using PowerShell's `Select-Xml` cmdlet with an XPath expression.
- **D-03:** Two copies of the baseline XML exist:
  - **Reference copy** — exported once, committed to `spike/baseline-reference.xml`. Never re-imported; for documentation and Phase 2 schema study only.
  - **Per-machine copy** — exported to `spike/baseline-local.xml` on the test machine. Listed in `.gitignore`. `Reset-WacomMapping.ps1` re-imports this to restore full-screen mapping.

### Claude's Discretion
- Exact XPath expression to target the mapping element (determined at spike time from `baseline-reference.xml`)
- Handling of multiple connected tablets (decided from what's found in baseline XML)
- Logging verbosity (`Write-Host` progress lines; level of detail is Claude's call)
- DPI validation depth and admin-rights investigation (document findings in SPIKE-RESULTS.md as encountered)

### Deferred Ideas (OUT OF SCOPE)
- DPI scaling validation (125%/150% display) — belongs in Phase 3 (EXT-02)
- Admin rights workaround design (service/scheduled-task) — belongs in Phase 2
- SPIKE-RESULTS.md scope beyond required fields — bonus appendix sections only
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SPIKE-01 | PowerShell script sets Wacom tablet mapping to arbitrary screen region (X, Y, Width, Height) | PrefUtil.exe `/import` with modified XML clone enables this; XML mapping tags TBD from baseline export |
| SPIKE-02 | Mapping change completes in under 3 seconds (consecutive) | PrefUtil exits quickly after import; `Wait-Process` (not `-Wait`) avoids 1-second poll floor; real-world latency is empirical |
| SPIKE-03 | Wacom stylus respects the new mapping region after change | Validated manually during Plan 01-03 test cases |
| SPIKE-04 | Reset script restores full-screen mapping from baseline profile | `Reset-WacomMapping.ps1` re-imports `spike/baseline-local.xml` via PrefUtil |
| SPIKE-05 | SPIKE-RESULTS.md documents method, binary paths, XML structure, service names, latency, recommendation | Plan 01-03 fills this; research pre-documents what fields to look for |
</phase_requirements>

---

## Summary

The Wacom driver exposes a command-line preference utility (`PrefUtil.exe`) that can export and import the full tablet preference profile as XML. The canonical import/export mechanism is: export current settings to a `.wacompref` file (which is XML), edit the XML to change the screen-mapping region, then re-import the modified file. The utility reads from and writes to `%APPDATA%\WTablet\Wacom_Tablet.dat` as its live settings store.

The exact XML tags controlling screen-to-tablet mapping are **not publicly documented** and must be discovered empirically by exporting a baseline, making a mapping change via the Wacom Tablet Properties UI, exporting again, and diffing the two files. This is the core discovery work in Plan 01-01. The developer-support.wacom.com articles confirm that partial XML imports (containing only changed settings) are accepted — the import only overrides settings present in the XML file, leaving all others intact. This is the mechanism that allows Plan D-01's "clone and modify" approach to work safely.

The Windows Wacom service is named `WTabletServicePro` (PowerShell) / `TabletServiceWacom` (net commands). Community evidence suggests the service does not need to be stopped before importing preferences — `PrefUtil.exe` communicates with the running service. Whether a service bounce is needed for mapping changes to take effect is a spike finding. A critical PowerShell timing pitfall was found: `Start-Process -Wait` polls on a 1-second interval, inflating measured latency by up to 1 second. The correct pattern is `$p = Start-Process ...; $p | Wait-Process`.

**Primary recommendation:** Run Plan 01-01 first to discover the actual XML diff between two mapping states. Everything else depends on those tag names.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Discover XML mapping tags | Developer workstation (spike) | — | One-time empirical diff on baseline vs. modified export |
| Modify screen-mapping XML | PowerShell script (local execution) | — | `Select-Xml` XPath modification of cloned baseline |
| Apply preference change | `PrefUtil.exe` (Windows process) | Wacom service | PrefUtil communicates change to the running driver service |
| Measure latency | PowerShell `Measure-Command` + `Wait-Process` | — | Script-level timing around PrefUtil invocation |
| Restore full-screen mapping | PowerShell `Reset-WacomMapping.ps1` | — | Re-imports `spike/baseline-local.xml` via PrefUtil |
| Document findings | `SPIKE-RESULTS.md` / `test-log.md` | — | Manually filled during Plan 01-03 test execution |

---

## Standard Stack

### Core

| Tool/Library | Version | Purpose | Why Standard |
|---|---|---|---|
| PowerShell | 5.1 (built-in Windows) | Script host for both scripts | Pre-installed on all Windows machines; no dependency install needed |
| PrefUtil.exe | Ships with Wacom driver (≥ 6.3) | Export/import Wacom preference XML | Official Wacom mechanism; path: `C:\Program Files\Tablet\Wacom\PrefUtil.exe` [ASSUMED path - verify on test machine] |
| `Select-Xml` (built-in cmdlet) | Part of `Microsoft.PowerShell.Utility` | XPath query and node selection in preference XML | Standard PowerShell XML tool; handles namespaces cleanly via `-Namespace` hashtable |
| `[xml]` accelerator (built-in) | PowerShell built-in | Load/save XML documents | Faster than `Select-Xml` for full-document loads; used for `$doc.Save()` |

### Alternative Utility Name

Older Wacom driver versions (pre-6.3) shipped `Wacom_TabletUserPrefs.exe` instead of `PrefUtil.exe`. The spec references this older name. Plan 01-01 must verify which executable is present and confirm the actual path. [VERIFIED: search results confirm PrefUtil.exe is the current name; Wacom_TabletUserPrefs.exe appears in older documentation]

### Supporting

| Tool | Purpose | When to Use |
|---|---|---|
| `Measure-Command` | Wrap PrefUtil invocations for wall-clock latency | Plan 01-03 latency test (SPIKE-02) |
| `Wait-Process` | Block until PrefUtil exits without the 1-second poll penalty | Replaces `-Wait` flag on `Start-Process` |
| `net stop` / `net start` | Start/stop Wacom service if restart is needed | Only if empirical testing shows service restart is required for mapping to take effect |

### Environment Availability

This phase cannot execute on the current Linux development machine. All scripts are authored here (cross-platform text editing), but execution requires:

| Dependency | Required By | Available (dev machine) | Available (test machine) | Fallback |
|---|---|---|---|---|
| Windows 10/11 | All plans | ✗ (Linux) | ✓ (assumed) | None — Windows-only |
| Wacom One M tablet | SPIKE-03 physical validation | ✗ | Must be connected | None — hardware required |
| Wacom driver ≥ 6.3 | PrefUtil.exe | ✗ | Must be installed | None |
| PowerShell 5.1+ | Script host | ✗ (Linux has pwsh) | ✓ (built-in) | None |
| Admin / elevated session | PrefUtil import (TBD) | N/A | TBD — spike finding | Scheduled task workaround (Phase 2 scope) |

**Missing dependencies with no fallback:** Physical Wacom One M + Windows test machine. The spike cannot be validated without this hardware. Script authoring (Plans 01-01 partial, 01-02) can be done on any machine; Plans 01-01 (baseline export) and 01-03 (test execution) require Windows + tablet.

---

## Architecture Patterns

### System Architecture Diagram

```
Plan 01-01: Environment Prep
  Developer ──export──> PrefUtil.exe ──reads──> Wacom_Tablet.dat
                            │
                            └──writes──> baseline-reference.xml (committed)
                                         baseline-local.xml (gitignored)

Plan 01-02: Script Authoring
  baseline-reference.xml ──diff study──> discover XML mapping tags
  Set-WacomMapping.ps1:
    param(X, Y, Width, Height)
    clone baseline-local.xml --> modified.xml
    Select-Xml XPath modify --> mapping nodes updated
    Start-Process PrefUtil /import modified.xml | Wait-Process

Plan 01-03: Test Execution
  Set-WacomMapping.ps1 ──invokes──> PrefUtil.exe ──notifies──> WTabletServicePro
                                                                     │
                                                                     └──applies──> Wacom driver
                                                                                      │
                                                                                      └──restricts──> stylus to region
  Measure-Command wraps above for SPIKE-02 latency measurement
  Reset-WacomMapping.ps1 ──invokes──> PrefUtil.exe /import baseline-local.xml
```

### Recommended Project Structure

```
spike/
├── Set-WacomMapping.ps1        # params: -X -Y -Width -Height
├── Reset-WacomMapping.ps1      # re-imports baseline-local.xml
├── baseline-reference.xml      # committed; schema reference for Phase 2
├── baseline-local.xml          # gitignored; per-machine baseline
├── SPIKE-RESULTS.md            # filled in Plan 01-03
└── test-log.md                 # raw test execution log
```

### Pattern 1: Clone-and-Modify Baseline (D-01)

**What:** Copy `baseline-local.xml` to a temp path, modify mapping nodes, pass the temp file to PrefUtil for import.
**When to use:** Every call to `Set-WacomMapping.ps1`.
**Why:** Guarantees PrefUtil receives its own export format with all required fields intact. Avoids constructing partial XML that PrefUtil might reject.

```powershell
# Source: CONTEXT.md D-01 + PowerShell [xml] docs
param(
    [int]$X,
    [int]$Y,
    [int]$Width,
    [int]$Height
)

$BaselinePath  = Join-Path $PSScriptRoot 'baseline-local.xml'
$TempPath      = Join-Path $env:TEMP 'wacom-mapping-temp.xml'

Write-Host "[Set-WacomMapping] Cloning baseline..."
[xml]$doc = Get-Content -Path $BaselinePath -Raw

# XPath expression TBD from baseline-reference.xml diff study in Plan 01-01
# Placeholder: replace with actual tag names discovered on test machine
$mappingNode = Select-Xml -Xml $doc -XPath '//PLACEHOLDER_MAPPING_NODE' |
               Select-Object -First 1 -ExpandProperty Node

$mappingNode.Left   = $X
$mappingNode.Top    = $Y
$mappingNode.Right  = $X + $Width
$mappingNode.Bottom = $Y + $Height

$doc.Save($TempPath)
Write-Host "[Set-WacomMapping] Saved modified XML to $TempPath"
```

### Pattern 2: PrefUtil Invocation with Accurate Timing

**What:** Launch PrefUtil and block until it exits, using `Wait-Process` not `-Wait`.
**When to use:** Both `Set-WacomMapping.ps1` and `Reset-WacomMapping.ps1`.
**Why:** `Start-Process -Wait` polls on a 1-second interval and will report ≥1000 ms even for fast operations, corrupting SPIKE-02 measurements. [VERIFIED: GitHub issue #24709 on PowerShell/PowerShell]

```powershell
# Source: https://github.com/PowerShell/PowerShell/issues/24709
$PrefUtilPath = 'C:\Program Files\Tablet\Wacom\PrefUtil.exe'  # verify path in Plan 01-01

$elapsed = Measure-Command {
    $proc = Start-Process -FilePath $PrefUtilPath `
                          -ArgumentList '--import', $TempPath `
                          -PassThru
    $proc | Wait-Process
}

Write-Host "[Set-WacomMapping] PrefUtil completed in $($elapsed.TotalMilliseconds) ms"
```

**Note:** The `--import` flag syntax (`--import` vs `/import`) is assumed from Wacom developer support article summaries. Verify by running `PrefUtil.exe /?` or `PrefUtil.exe --help` in Plan 01-01.

### Pattern 3: Baseline Export (Plan 01-01)

```powershell
# Export current settings to discover file path and format
$PrefUtilPath    = 'C:\Program Files\Tablet\Wacom\PrefUtil.exe'
$BaselinePath    = Join-Path $PSScriptRoot 'baseline-local.xml'

$proc = Start-Process -FilePath $PrefUtilPath `
                      -ArgumentList '--export', $BaselinePath `
                      -PassThru
$proc | Wait-Process
Write-Host "Baseline exported to: $BaselinePath"
```

### Anti-Patterns to Avoid

- **`Start-Process PrefUtil -Wait`:** The `-Wait` flag polls on a 1-second clock. This inflates measured latency by 0–999 ms and will corrupt SPIKE-02 measurements. Use `$proc | Wait-Process` instead. [VERIFIED: PowerShell/PowerShell#24709]
- **Constructing minimal XML from scratch:** PrefUtil may require specific root elements, namespace declarations, or device-ID nodes. A hand-crafted minimal XML is likely to be rejected. Always clone and modify the baseline. (D-01)
- **Hardcoding `C:\Program Files\Tablet\Wacom\PrefUtil.exe`:** The install path can vary. In Plan 01-01, discover the actual path via `Get-Command PrefUtil.exe -ErrorAction SilentlyContinue` or registry query. Then hardcode the verified path in the scripts.
- **Editing `Wacom_Tablet.dat` directly:** This binary/proprietary file format is not safe to edit directly. The official mechanism is export → edit XML → import.
- **Stopping the Wacom service before import:** Community evidence suggests PrefUtil communicates with the live service to apply changes. Stopping the service first may prevent the change from being applied. Verify empirically in Plan 01-03.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---|---|---|---|
| XML node selection with namespaces | Custom regex or string replace | `Select-Xml` with `-Namespace` hashtable | Namespace-aware, handles default xmlns, returns proper XmlNode objects |
| Process timing | `[DateTime]::Now` subtraction | `Measure-Command { ... }` | Stopwatch-backed, sub-millisecond precision, idiomatic PowerShell |
| XML document load/save | `System.IO.File` with string manipulation | `[xml]` accelerator + `.Save()` | Preserves encoding, handles entities, round-trips cleanly |
| Screen coordinate validation | Custom bounds check | `[System.Windows.Forms.Screen]::PrimaryScreen.Bounds` | Returns actual pixel dimensions; use if clamping is needed |

**Key insight:** The XML format is Wacom's proprietary schema. Never assume you can construct valid import XML from scratch — always start from an export.

---

## Common Pitfalls

### Pitfall 1: PrefUtil executable name mismatch
**What goes wrong:** Script references `Wacom_TabletUserPrefs.exe` (old name) but installed driver only has `PrefUtil.exe` (new name), or vice versa.
**Why it happens:** Wacom renamed the utility between driver generations. The project spec uses the old name; current drivers (≥ 6.3) use `PrefUtil.exe`.
**How to avoid:** In Plan 01-01, scan the install directory: `Get-ChildItem 'C:\Program Files\Tablet\Wacom\' -Filter '*.exe' | Select-Object Name`
**Warning signs:** `Start-Process` throws "The system cannot find the file specified."

### Pitfall 2: Incorrect `--import` flag syntax
**What goes wrong:** PrefUtil fails silently or returns an error because the flag is `/import` not `--import` (or vice versa).
**Why it happens:** Wacom's documentation is gated (403), so exact flag syntax is unconfirmed. [ASSUMED flag name]
**How to avoid:** In Plan 01-01, run `PrefUtil.exe /?` or `PrefUtil.exe --help` and capture output to `test-log.md`.
**Warning signs:** PrefUtil exits immediately with no error; mapping does not change.

### Pitfall 3: `Start-Process -Wait` adds ~1 second to every latency measurement
**What goes wrong:** Three consecutive mapping changes always measure ≥ 3 000 ms even when actual PrefUtil execution is fast, causing SPIKE-02 to appear to fail.
**Why it happens:** PowerShell's `Start-Process -Wait` internally uses a Win32 job object with a 1-second polling interval. [VERIFIED: PowerShell/PowerShell issue #24709]
**How to avoid:** Always use `$p = Start-Process ... -PassThru; $p | Wait-Process` pattern. Wrap with `Measure-Command`.
**Warning signs:** Three runs consistently measure 1 000–1 999 ms instead of expected sub-500 ms.

### Pitfall 4: XML mapping tags unknown until baseline diff
**What goes wrong:** Scripts contain placeholder XPath and fail at first run because real tag names differ.
**Why it happens:** Wacom does not publicly document the preference XML schema. Tag names (e.g., `ScreenLeft`, `MapArea`, `ScreenRect`) vary by driver version and are only discoverable empirically.
**How to avoid:** Plan 01-01 must export two profiles (one default, one after moving mapping via Wacom Tablet Properties UI) and diff them to find the exact tag name and attribute structure before writing any XPath in the scripts.
**Warning signs:** `Select-Xml` returns `$null` for the mapping node.

### Pitfall 5: Admin rights requirement blocks non-elevated scripts
**What goes wrong:** PrefUtil exits with "Access Denied" or silently fails because the calling PowerShell session is not elevated.
**Why it happens:** Unknown — this is an open spike question. Service-level operations often require elevation.
**How to avoid:** In Plan 01-01, run PrefUtil export from both an elevated and non-elevated prompt; compare results. Document in SPIKE-RESULTS.md.
**Warning signs:** PrefUtil exits immediately with exit code ≠ 0; mapping does not change.

### Pitfall 6: Multiple connected tablets — ambiguous import target
**What goes wrong:** If two Wacom tablets are connected, the preference XML contains entries for both. The import may apply to the wrong device or fail.
**Why it happens:** Preference XML is keyed by device identifier. The script must target the correct device's mapping section.
**How to avoid:** Document whether multiple tablets are connected during spike. The baseline diff will show how device sections are identified (device name, USB ID, or similar). Claude's discretion applies (CONTEXT.md).
**Warning signs:** Mapping change applies to a different tablet than intended.

### Pitfall 7: Wacom_Tablet.dat path varies by driver/user profile
**What goes wrong:** Script hardcodes `%APPDATA%\WTablet\Wacom_Tablet.dat` but actual file is elsewhere.
**Why it happens:** Path may differ between Wacom driver versions and product lines.
**How to avoid:** Use PrefUtil's own export to write to a known location (the script controls the output path). Never read `Wacom_Tablet.dat` directly.
**Warning signs:** `baseline-local.xml` is empty or not created.

---

## Code Examples

### Verified PowerShell Patterns

#### Select-Xml with namespace (Microsoft Learn)
```powershell
# Source: https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.utility/select-xml
# If the Wacom XML uses a default xmlns, wrap it with an arbitrary prefix:
$ns = @{ wacom = 'http://schemas.wacom.com/preferences' }  # replace URI with actual
$node = Select-Xml -Xml $doc -XPath '//wacom:ScreenMapping' -Namespace $ns |
        Select-Object -First 1 -ExpandProperty Node
# If no namespace: Select-Xml -Xml $doc -XPath '//ScreenMapping' | ...
```

#### XML save round-trip
```powershell
# Source: PowerShell [xml] accelerator docs (ASSUMED no namespace conflicts)
[xml]$doc = Get-Content -Path $BaselinePath -Raw
# ... modify nodes ...
$doc.Save($TempPath)  # preserves encoding declaration
```

#### Accurate process timing (avoids 1-second floor)
```powershell
# Source: https://github.com/PowerShell/PowerShell/issues/24709
$elapsed = Measure-Command {
    $proc = Start-Process $PrefUtilPath -ArgumentList '--import', $TempPath -PassThru
    $proc | Wait-Process
}
Write-Host "Elapsed: $($elapsed.TotalMilliseconds) ms"
```

#### Wacom service restart (if empirically required)
```powershell
# Source: https://github.com/retorillo/restart-wacom-service (service name verified)
# Service name: WTabletServicePro (PowerShell) OR TabletServiceWacom (net commands)
Restart-Service -Name WTabletServicePro -Force  # requires elevation
# Alternative via net commands:
# net stop TabletServiceWacom && net start TabletServiceWacom
```

---

## Key Unknowns (Must Discover in Plan 01-01)

These are genuinely unresolvable without a live Wacom machine. Plans 01-01 and 01-03 exist specifically to answer them.

| Unknown | How to Discover | SPIKE-RESULTS.md Field |
|---|---|---|
| Exact XML tag names for screen mapping region | Export before/after UI mapping change; diff the two files | "XML tag names" |
| Whether XML uses namespaces | Inspect the root element of the baseline export | Affects XPath pattern |
| PrefUtil exact flag syntax (`/import` vs `--import`) | Run `PrefUtil.exe /?` and capture output | "Binary path and invocation" |
| PrefUtil actual install path | `Get-ChildItem 'C:\Program Files\Tablet\Wacom\'` | "Binary path and invocation" |
| Admin rights required? | Test PrefUtil from non-elevated prompt | "Admin-rights requirement" |
| Service restart required? | Change mapping via PrefUtil; observe whether stylus responds immediately or only after service restart | "Service name, restart behavior" |
| Coordinate system (logical vs physical pixels) | Set a known region (e.g., half-screen), measure with physical ruler on display | "DPI / coordinate system finding" |
| Latency (actual) | Three consecutive `Measure-Command` runs | "Measured latency" |

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|---|---|---|---|
| `Wacom_TabletUserPrefs.exe` | `PrefUtil.exe` | Wacom driver ≥ 6.3 (approx. 2018+) | CLAUDE.md and original spec reference old name; test machine may have either |
| GUI-only mapping (AutoIt screen scraping) | PrefUtil CLI export/import | N/A | PrefUtil is the documented headless method for enterprise deployment |
| `.wacomxs` preference file format | `.wacompref` (preferred) | Driver modernization | Both still accepted; PrefUtil searches `.wacompref` first if extension omitted |

**Deprecated/outdated:**
- `Wacom_TabletUserPrefs.exe`: Older name still present in some driver installations; treat as alias to verify in Plan 01-01.
- Direct `Wacom_Tablet.dat` editing: Proprietary binary/XML hybrid — community sources confirm it cannot be safely hand-edited.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|---|---|---|
| A1 | PrefUtil.exe install path is `C:\Program Files\Tablet\Wacom\PrefUtil.exe` | Standard Stack, Code Examples | Script will fail with "file not found"; must discover actual path in Plan 01-01 |
| A2 | PrefUtil import flag is `--import <path>` (double-dash) | Code Examples | Script invocation fails; must verify with `PrefUtil.exe /?` in Plan 01-01 |
| A3 | Wacom service name is `WTabletServicePro` (PowerShell) / `TabletServiceWacom` (net) | Common Pitfalls, Code Examples | Service restart commands fail; verify via `Get-Service *wacom* -ErrorAction SilentlyContinue` |
| A4 | Partial XML import (only changed settings) is accepted by PrefUtil | Summary, Standard Stack | Must use full-profile clone approach (D-01); if partial import is rejected too, D-01's clone approach is still safe |
| A5 | No service restart is needed for mapping to take effect | Architecture Patterns | If restart IS required, latency will increase significantly and SPIKE-02 may fail; document and flag for Phase 2 workaround |
| A6 | Screen-mapping coordinates in preference XML are in physical pixels (not logical/DPI-scaled) | Common Pitfalls | If logical pixels are used, the spike scripts will produce offset/wrong regions on scaled displays; document finding |
| A7 | A single PrefUtil import completes in under 3 seconds of real execution time | Summary | If PrefUtil itself is slow (> 3 s), SPIKE-02 fails and Phase 2 architecture must change; empirical measurement is the spike's core deliverable |

---

## Open Questions

1. **XML tag names for screen-to-tablet mapping**
   - What we know: Tags exist (confirmed by UI and export/import mechanism)
   - What's unclear: Exact element names, attribute names, coordinate semantics (left/top/right/bottom vs x/y/width/height)
   - Recommendation: Perform the export-diff in Plan 01-01 as step 1; everything else depends on this

2. **Admin rights requirement for PrefUtil /import**
   - What we know: Export can likely run without elevation; import may need elevation to communicate with the service
   - What's unclear: Whether a domain user running the browser extension can invoke PrefUtil non-elevated
   - Recommendation: Test both elevated and non-elevated; document. Phase 2 design (scheduled task vs. service) depends on this answer.

3. **Whether Wacom_TabletUserPrefs.exe still exists on current driver installations**
   - What we know: PrefUtil.exe is the current name; older name is referenced in the project spec
   - What's unclear: Whether the test machine's driver is old or new
   - Recommendation: In Plan 01-01, check for both names and document which is present

---

## Sources

### Primary (HIGH confidence)
- [Microsoft Learn — Select-Xml](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.utility/select-xml?view=powershell-7.5) — `-Namespace` parameter, XPath syntax, `.Node` property access
- [PowerShell/PowerShell issue #24709](https://github.com/PowerShell/PowerShell/issues/24709) — `Start-Process -Wait` 1-second poll floor confirmed; `Wait-Process` workaround

### Secondary (MEDIUM confidence)
- Wacom developer-support.wacom.com article summaries (403 on direct fetch; content recovered via search snippets) — PrefUtil.exe path, `.wacompref` format, enterprise import behavior, partial-XML import semantics
- [restart-wacom-service batch script](https://github.com/retorillo/restart-wacom-service/blob/master/restart-wacom-service.bat) — service name `WTabletServicePro` confirmed
- [yal.cc — Wacom portrait orientation](https://yal.cc/wacom-one-portrait-orientation/) — `%APPDATA%\WTablet\Wacom_Tablet.dat` as live settings store; `<Orientation>` tag format as example of XML structure
- Search snippet: Wacom developer support — `WacomTabletDefaults.xml` + `PrefsLocation` tag structure; partial XML import behavior; enterprise deployment model

### Tertiary (LOW confidence — flag for validation)
- [yal.cc — Wacom auto-adjust screen area](https://yal.cc/wacom-auto-adjust-screen-area/) — demonstrates that GUI scripting via AutoIt is an alternative if PrefUtil fails; coordinate field IDs 53–56 (Left/Right/Top/Bottom) but this is Windows control IDs, not XML tags
- General web search results — service name `TabletServiceWacom` (net commands); general community corroboration of PrefUtil usage

---

## Metadata

**Confidence breakdown:**
- Standard stack: MEDIUM — PrefUtil.exe confirmed; flag syntax assumed
- Architecture: MEDIUM — Export/import mechanism confirmed; XML tag names unconfirmed
- Pitfalls: HIGH for timing pitfall (verified); MEDIUM for others (inferred from mechanism)

**Research date:** 2026-04-29
**Valid until:** 2027-04-29 (Wacom driver scripting API is stable; PrefUtil has been the mechanism since ~2018)
