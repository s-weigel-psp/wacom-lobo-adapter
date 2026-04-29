# Phase 1: Wacom Mapping Spike — Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-29
**Phase:** 01-wacom-mapping-spike
**Areas discussed:** XML editing strategy

---

## XML Editing Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Modify baseline (Recommended) | Clone baseline.xml, find mapping node, update values, import. Preserves all other tablet settings; guaranteed format compatibility. | ✓ |
| Build minimal XML from scratch | Construct only the mapping element. Simpler, but unknown whether Wacom_TabletUserPrefs.exe accepts partial profiles. | |
| Try both, document which works | Test minimal first, fall back to baseline-clone, record finding in SPIKE-RESULTS.md. | |

**User's choice:** Modify baseline

---

| Option | Description | Selected |
|--------|-------------|----------|
| XPath via Select-Xml (Recommended) | Use Select-Xml with XPath to locate the mapping node. Precise, handles namespaces. | ✓ |
| Cast to [xml] and navigate as object | $xml = [xml](Get-Content), navigate by property. Works when schema is known. | |
| Regex / string replacement | Find-and-replace coordinate values as strings. Fragile but trivial. | |

**User's choice:** XPath via Select-Xml

---

| Option | Description | Selected |
|--------|-------------|----------|
| spike/ folder in the repo (Recommended) | Export baseline.xml to spike/baseline.xml, commit to repo. Both scripts reference via $PSScriptRoot. | ✓ (with nuance) |
| Fixed path on the test machine | Hardcoded path like C:\temp\wacom-baseline.xml. Simpler but requires manual setup. | |

**User's choice:** Commit a reference copy to the repo for documentation; per-machine baselines are local and never committed (per-user, .gitignore'd).

**Notes:** User clarified that two distinct baseline artifacts exist: (1) a reference copy for Phase 2 to study the XML schema — committed once to `spike/baseline-reference.xml`; (2) a per-machine copy used by Reset-WacomMapping.ps1 — stored locally as `spike/baseline-local.xml` and excluded from version control.

---

## Claude's Discretion

- Exact XPath expression for the mapping element
- Multi-tablet handling
- Logging verbosity level
- DPI validation and admin-rights investigation depth (not explicitly scoped — document findings as encountered)

## Deferred Ideas

- DPI scaling tests in Phase 1 — deferred to Phase 3 (EXT-02)
- Admin rights workaround design — deferred to Phase 2
- Extended SPIKE-RESULTS.md sections — welcomed as bonus findings, not required scope
