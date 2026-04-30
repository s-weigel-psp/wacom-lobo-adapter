#Requires -Version 5.1
<#
.SYNOPSIS
    Restores the Wacom tablet to full-screen mapping by re-importing the
    per-machine baseline preference XML.

.DESCRIPTION
    Re-imports spike\baseline-local.Export.wacomxs (the per-machine full-screen
    baseline exported in Plan 01-01) without modification.

    Per D-03 (CONTEXT.md):
      baseline-local.Export.wacomxs  — per-machine baseline (gitignored); this is
                                       the only file that Reset-WacomMapping.ps1
                                       imports.
      baseline-reference.Export.wacomxs — committed reference; NEVER imported;
                                          schema documentation only.

    PrefUtil path and import flag verified in Plan 01-01 (see test-log.md):
        Path : C:\Program Files\Tablet\Wacom\PrefUtil.exe
        Flag : /import

    FILE EXTENSION: The preference file MUST use the .Export.wacomxs extension.
    Using .xml causes PrefUtil to silently fail to write the file at the specified
    path (verified during Plan 01-01 baseline export tests).

    GUI DIALOG WARNING:
    PrefUtil.exe opens a native Windows dialog for every /import operation.
    The /silent flag has NO effect on /import — it only suppresses the help-screen
    window.  A dialog will appear when this script runs.  Click OK to restore the
    full-screen mapping.
    This is a known PrefUtil limitation documented in spike/test-log.md (Plan 01-01).

    Admin rights are NOT required (exit code 0 confirmed from non-elevated PowerShell).

.EXAMPLE
    .\Reset-WacomMapping.ps1
    # Restores full-screen stylus movement.

.NOTES
    Requires spike\baseline-local.Export.wacomxs to exist.
    Export it in Plan 01-01 setup: PrefUtil.exe /export <path>
#>
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# --- Paths ---
$ScriptDir    = $PSScriptRoot
$BaselinePath = Join-Path $ScriptDir 'baseline-local.Export.wacomxs'

# PrefUtil path verified in Plan 01-01 (spike/test-log.md).
# Must match the path used in Set-WacomMapping.ps1.
$PrefUtilPath = 'C:\Program Files\Tablet\Wacom\PrefUtil.exe'

# --- Precondition checks ---
Write-Host "[Reset-WacomMapping] Checking prerequisites..."

if (-not (Test-Path $BaselinePath)) {
    Write-Error "baseline-local.Export.wacomxs not found at: $BaselinePath`nRun Plan 01-01 to export the baseline first:`n  PrefUtil.exe /export '$BaselinePath'"
    exit 1
}

if (-not (Test-Path $PrefUtilPath)) {
    Write-Error "PrefUtil not found at: $PrefUtilPath`nVerify the path in this script matches the test machine installation."
    exit 1
}

Write-Host "[Reset-WacomMapping] Importing baseline-local.Export.wacomxs to restore full-screen mapping..."

# --- Import baseline via PrefUtil (Pattern 2: Wait-Process, not -Wait flag) ---
# WARNING: PrefUtil will open a GUI dialog requiring the user to click OK.
# This is a known PrefUtil limitation — /silent has no effect on /import.
#
# Wait-Process (not -Wait flag) avoids the 1-second poll floor (PowerShell#24709),
# keeping Measure-Command timing consistent with Set-WacomMapping.ps1.
Write-Host "[Reset-WacomMapping] Invoking PrefUtil /import (a GUI dialog will appear — click OK)..."

$elapsed = Measure-Command {
    # Import flag '/import' verified in Plan 01-01 (spike/test-log.md).
    # -PassThru is required so that $proc is populated for Wait-Process.
    $proc = Start-Process -FilePath $PrefUtilPath `
                          -ArgumentList '/import', $BaselinePath `
                          -PassThru
    $proc | Wait-Process
}

Write-Host "[Reset-WacomMapping] PrefUtil exit code: $($proc.ExitCode)"
Write-Host "[Reset-WacomMapping] PrefUtil completed in $([Math]::Round($elapsed.TotalMilliseconds)) ms"

if ($proc.ExitCode -ne 0) {
    Write-Warning "PrefUtil returned non-zero exit code: $($proc.ExitCode). Full-screen mapping may not have been restored."
    exit $proc.ExitCode
}

Write-Host "[Reset-WacomMapping] Done. Full-screen stylus mapping restored."
