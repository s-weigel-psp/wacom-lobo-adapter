# Phase 1: Wacom Mapping Spike — Context

**Gathered:** 2026-04-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Validate that `Wacom_TabletUserPrefs.exe` can be scripted via PowerShell to restrict the Wacom stylus to an arbitrary screen region, measure latency, and produce a documented recommendation for Phase 2.

Three plans in scope: (1) environment prep and baseline XML export, (2) authoring `Set-WacomMapping.ps1` and `Reset-WacomMapping.ps1`, (3) executing manual tests and filling `SPIKE-RESULTS.md`.

New capabilities (C# porting, service installation, extension code) belong in later phases.

</domain>

<decisions>
## Implementation Decisions

### XML Editing Strategy
- **D-01:** `Set-WacomMapping.ps1` modifies a cloned copy of the baseline XML rather than constructing a minimal XML from scratch. The full profile is preserved so all other tablet settings (pressure curve, button assignments, etc.) remain intact, and `Wacom_TabletUserPrefs.exe` is guaranteed to accept its own format.
- **D-02:** The script locates the screen-mapping element using PowerShell's `Select-Xml` cmdlet with an XPath expression. This is precise, handles XML namespace quirks, and is the standard PowerShell XML manipulation approach.
- **D-03:** Two copies of the baseline XML exist:
  - **Reference copy** — exported once by the developer, committed to `spike/baseline-reference.xml` in the repo. Purpose: lets Phase 2 read the XML schema without needing a live Wacom machine. Never re-imported; purely for documentation.
  - **Per-machine copy** — exported on the test machine to `spike/baseline-local.xml`. This is the file `Reset-WacomMapping.ps1` re-imports to restore full-screen mapping. Listed in `.gitignore`; never committed (it is per-user and contains machine-specific paths).

### Claude's Discretion
- Exact XPath expression to target the mapping element — determined at spike time once the XML structure is known from `baseline-reference.xml`
- Handling of multiple connected tablets (if any) — Claude decides based on what's found in the baseline XML
- Logging verbosity in the scripts — `Write-Host` progress lines at each step; level of detail is Claude's call
- DPI validation depth and admin-rights investigation — not explicitly scoped by the user; Claude documents findings as encountered during Plan 01-03 execution

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

No external specs exist yet — all requirements are captured in REQUIREMENTS.md and this context file.

### Project Requirements
- `.planning/REQUIREMENTS.md` — SPIKE-01 through SPIKE-05 define exactly what the spike must prove; read before writing test scripts or SPIKE-RESULTS.md
- `.planning/PROJECT.md` — Key Decisions table and Constraints section (latency < 3s, Windows-only, no custom driver work)

### Wacom Environment (discovered during spike execution)
- `spike/baseline-reference.xml` — Reference XML export committed after Plan 01-01; downstream Phase 2 reads this to understand the preference XML schema before a live Wacom machine is available

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- None — this is the first phase; no existing scripts or components.

### Established Patterns
- None yet. The PowerShell patterns established here (XML manipulation via `Select-Xml`, `Wacom_TabletUserPrefs.exe /import` invocation, baseline clone approach) become the reference for Phase 2's C# port.

### Integration Points
- `Wacom_TabletUserPrefs.exe` — invoked via `Start-Process` with `/import` flag and the modified XML path
- `%ProgramData%\Tablet\Wacom\` and/or `%LOCALAPPDATA%\Wacom\` — preference file locations to identify in Plan 01-01
- Windows Wacom service (name TBD in Plan 01-01) — may need stop/start around `Wacom_TabletUserPrefs.exe` calls

</code_context>

<specifics>
## Specific Ideas

- The reference baseline (`spike/baseline-reference.xml`) is committed once after Plan 01-01 so Phase 2 can study the XML schema offline — this was explicitly called out by the user.
- The per-machine local baseline (`spike/baseline-local.xml`) is in `.gitignore` — user was explicit that per-user files must not be committed.

</specifics>

<deferred>
## Deferred Ideas

- DPI scaling validation (testing on 125%/150% display) — not scoped for Phase 1; Phase 3 (EXT-02) handles DPR correction in the extension. If the spike happens to reveal DPI behavior, document it in SPIKE-RESULTS.md as a bonus finding.
- Admin rights investigation depth — if `Wacom_TabletUserPrefs.exe` requires elevation, document the finding in SPIKE-RESULTS.md; the service/scheduled-task workaround design belongs in Phase 2.
- SPIKE-RESULTS.md scope beyond the required fields — any additional useful findings (registry paths, service restart behavior, error codes) can be included as appendix sections without expanding scope.

</deferred>

---

*Phase: 01-wacom-mapping-spike*
*Context gathered: 2026-04-29*
