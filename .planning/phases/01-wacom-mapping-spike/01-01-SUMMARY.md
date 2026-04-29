---
phase: 01-wacom-mapping-spike
plan: "01"
subsystem: spike-environment
tags: [wacom, spike, xml, powershell, environment-prep]
dependency_graph:
  requires: []
  provides:
    - spike/.gitkeep
    - spike/test-log.md (template)
    - .gitignore (baseline-local.xml, baseline-modified.xml excluded)
  affects:
    - spike/baseline-reference.xml (to be populated by human on Windows machine — Task 1 checkpoint)
    - Plan 01-02 (blocked until Task 1 data is available)
tech_stack:
  added: []
  patterns:
    - "D-03: two-copy baseline policy (reference committed, per-machine gitignored)"
key_files:
  created:
    - spike/.gitkeep
    - spike/test-log.md
    - .gitignore
  modified: []
decisions:
  - "D-03 enforced: spike/baseline-local.xml and spike/baseline-modified.xml added to .gitignore"
  - "spike/test-log.md structured with 6 sections to guide Windows test machine discovery"
metrics:
  duration: "< 5 minutes (Task 2 only)"
  completed: "2026-04-29"
  tasks_completed: 1
  tasks_total: 2
---

# Phase 1 Plan 1: Environment Prep and Baseline Export — Summary

**One-liner:** Spike/ directory scaffolded with test-log.md template and .gitignore exclusions; Windows baseline export step (Task 1) is a human-action checkpoint awaiting execution on a Wacom-connected Windows machine.

## What Was Built

### Task 2: spike/ scaffold (COMPLETED — commit cd1af89)

The following files were created on the Linux dev machine in preparation for the Windows test execution:

- **spike/.gitkeep** — Tracks the empty `spike/` directory in git before `baseline-reference.xml` is committed
- **spike/test-log.md** — Structured 6-section template for recording all Windows discovery data (PrefUtil path, help output, XML diff result, service names, admin rights test, coordinate notes)
- **.gitignore** — Created with entries to exclude `spike/baseline-local.xml` (per-machine baseline, D-03 policy) and `spike/baseline-modified.xml` (temp diff file)

### Task 1: Windows baseline export (PENDING — checkpoint:human-action)

Task 1 requires physical execution on a Windows machine with a Wacom One M tablet connected and driver installed. The task cannot be automated from the Linux dev machine.

**Status:** Awaiting human execution. See checkpoint message below.

**What the human needs to do on the Windows test machine:**
1. Locate `PrefUtil.exe` or `Wacom_TabletUserPrefs.exe` in `C:\Program Files\Tablet\Wacom\`
2. Run `--help` / `/?` to confirm export/import flag syntax
3. Export baseline-A (full-screen mapping) to `spike/baseline-reference.xml`
4. Change mapping via Wacom Tablet Properties GUI to a partial region
5. Export baseline-B to `spike/baseline-modified.xml`
6. Diff the two XMLs to identify the screen-mapping element and coordinate attributes
7. Run `Get-Service *wacom*` / `*tablet*` to discover service names
8. Test PrefUtil from a non-elevated prompt
9. Restore full-screen mapping and fill in `spike/test-log.md`

**After completion, the human commits `spike/baseline-reference.xml` and fills `spike/test-log.md`.**

## Deviations from Plan

None — Task 2 executed exactly as written. Task 1 is a planned checkpoint:human-action.

## Known Stubs

- `spike/test-log.md` — All 6 sections contain placeholder values (`[paste here]`, `[yes/no]`, etc.). This is intentional: the file is a template to be filled in during Task 1 on the Windows test machine. The stub is not a bug; it is the deliverable of Task 2.
- `spike/baseline-reference.xml` — Does not yet exist. Will be created by the human during Task 1 and committed separately.

## Pending Discovery Data (Task 1 Outputs — Required for Plan 01-02)

The following data is blocked on Task 1 completion and must be recorded in `spike/test-log.md` before Plan 01-02 can proceed:

| Data Point | Where Used in 01-02 |
|------------|---------------------|
| PrefUtil.exe actual path | `$PREFUTIL_PATH` variable in Set-WacomMapping.ps1 |
| Import flag syntax (`--import` / `/import`) | `Start-Process` arguments |
| Export flag syntax (`--export` / `/export`) | Reset-WacomMapping.ps1 re-export step |
| XML element name controlling screen mapping | XPath expression in `Select-Xml` call |
| Coordinate attribute names (Left/Top/Right/Bottom vs X/Y/Width/Height) | Script parameter mapping |
| Whether XML uses a namespace | Namespace hashtable in `Select-Xml` |
| Wacom service name(s) | Optional service restart around import |
| Admin rights requirement | Architecture decision (user-context vs. service) |
| XPath expression | Direct copy-paste into Set-WacomMapping.ps1 |

## Self-Check: PASSED

- spike/.gitkeep: FOUND
- spike/test-log.md: FOUND (6 sections present)
- .gitignore: FOUND (baseline-local.xml and baseline-modified.xml entries present)
- Commit cd1af89: verified in git log
