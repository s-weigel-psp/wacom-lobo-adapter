#Requires -Version 5.1
<#
.SYNOPSIS
    Runs all Phase 1 spike test cases and outputs results for SPIKE-RESULTS.md.
    Execute from the spike\ directory on the Windows test machine.

.DESCRIPTION
    Executes TC-01 through TC-05 in sequence. Between each test case the script
    pauses so you can physically verify stylus behaviour before continuing.

    IMPORTANT - PrefUtil GUI dialog:
    PrefUtil.exe opens a native Windows dialog for EVERY /import operation.
    The /silent flag has NO effect on /import. You must click OK in the dialog
    before the script can continue (Wait-Process blocks until PrefUtil exits).
    Each test case that calls Set-WacomMapping.ps1 or Reset-WacomMapping.ps1
    will produce exactly one dialog click.

    Prerequisites:
    1. Wacom One M tablet connected and Wacom driver running (WtabletServicePro).
    2. spike\baseline-local.Export.wacomxs must exist in the spike directory.
       If missing: export it via PrefUtil.exe /export <path-to-spike-dir>\baseline-local.Export.wacomxs
       and click OK in the dialog.
    3. Run from an unrestricted PowerShell session:
         Set-ExecutionPolicy -ExecutionPolicy Bypass -Scope Process
         .\run-tests.ps1

.NOTES
    Copy the console output into spike/test-log.md and fill the measured values
    into spike/SPIKE-RESULTS.md Measured Latency table.
#>
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$ScriptDir = $PSScriptRoot

Write-Host '=== Phase 1 Spike Test Run ===' -ForegroundColor Cyan
Write-Host "Date: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
Write-Host ''
Write-Host 'NOTE: PrefUtil opens a GUI dialog on every /import call.' -ForegroundColor Yellow
Write-Host '      Click OK in each dialog to allow the script to continue.' -ForegroundColor Yellow
Write-Host ''

# Verify prerequisites before starting
if (-not (Test-Path (Join-Path $ScriptDir 'baseline-local.Export.wacomxs'))) {
    Write-Host 'ERROR: spike\baseline-local.Export.wacomxs not found.' -ForegroundColor Red
    Write-Host 'Export it first (elevated or non-elevated PowerShell):'
    Write-Host '  & ''C:\Program Files\Tablet\Wacom\PrefUtil.exe'' /export <path-to-spike>\baseline-local.Export.wacomxs'
    Write-Host 'Click OK in the PrefUtil dialog, then re-run this script.'
    exit 1
}

# ─────────────────────────────────────────────────────────────────────────────
# TC-01: Left Half
# ─────────────────────────────────────────────────────────────────────────────
Write-Host '--- TC-01: Left Half (X=0 Y=0 Width=960 Height=1080) ---' -ForegroundColor Yellow
Write-Host 'A PrefUtil dialog will appear - click OK to apply the mapping.'
& (Join-Path $ScriptDir 'Set-WacomMapping.ps1') -X 0 -Y 0 -Width 960 -Height 1080
Write-Host ''
Write-Host '>> Verify: draw with the stylus. Cursor should be restricted to the LEFT half of the display.'
Write-Host '>> Record in SPIKE-RESULTS.md: exit code, stylus restriction (YES/NO/PARTIAL).'
Read-Host '>> Press Enter to continue to TC-02'
Write-Host ''

# ─────────────────────────────────────────────────────────────────────────────
# TC-02: Right Half
# ─────────────────────────────────────────────────────────────────────────────
Write-Host '--- TC-02: Right Half (X=960 Y=0 Width=960 Height=1080) ---' -ForegroundColor Yellow
Write-Host 'A PrefUtil dialog will appear - click OK to apply the mapping.'
& (Join-Path $ScriptDir 'Set-WacomMapping.ps1') -X 960 -Y 0 -Width 960 -Height 1080
Write-Host ''
Write-Host '>> Verify: draw with the stylus. Cursor should be restricted to the RIGHT half of the display.'
Write-Host '>> Record in SPIKE-RESULTS.md: exit code, stylus restriction (YES/NO/PARTIAL).'
Read-Host '>> Press Enter to continue to TC-03'
Write-Host ''

# ─────────────────────────────────────────────────────────────────────────────
# TC-03: Centre Region
# ─────────────────────────────────────────────────────────────────────────────
Write-Host '--- TC-03: Centre Region (X=240 Y=270 Width=1440 Height=540) ---' -ForegroundColor Yellow
Write-Host 'A PrefUtil dialog will appear - click OK to apply the mapping.'
& (Join-Path $ScriptDir 'Set-WacomMapping.ps1') -X 240 -Y 270 -Width 1440 -Height 540
Write-Host ''
Write-Host '>> Verify: draw with the stylus. Cursor should be restricted to the CENTRE rectangle'
Write-Host '   (left 240px, top 270px, right 1680px, bottom 810px on a 1920x1080 display).'
Write-Host '>> Record in SPIKE-RESULTS.md: exit code, stylus restriction (YES/NO/PARTIAL).'
Read-Host '>> Press Enter to continue to TC-04 (latency measurement)'
Write-Host ''

# ─────────────────────────────────────────────────────────────────────────────
# TC-04: Three Consecutive Changes - Latency (SPIKE-02)
# ─────────────────────────────────────────────────────────────────────────────
Write-Host '--- TC-04: Three Consecutive Changes (latency measurement, SPIKE-02) ---' -ForegroundColor Yellow
Write-Host 'Three /import calls will be made back-to-back. Each will show a PrefUtil dialog.'
Write-Host 'Click OK in each dialog as quickly as possible for a representative latency measurement.'
Write-Host 'Wall-clock time per run is printed below - copy these values into SPIKE-RESULTS.md.'
Write-Host ''

Write-Host 'Run 1/3 (X=0 Y=0 Width=960 Height=1080):'
$t1 = Measure-Command { & (Join-Path $ScriptDir 'Set-WacomMapping.ps1') -X 0 -Y 0 -Width 960 -Height 1080 }
Write-Host "TC-04 Run 1 total wall time: $([Math]::Round($t1.TotalMilliseconds)) ms" -ForegroundColor Cyan

Write-Host ''
Write-Host 'Run 2/3 (X=960 Y=0 Width=960 Height=1080):'
$t2 = Measure-Command { & (Join-Path $ScriptDir 'Set-WacomMapping.ps1') -X 960 -Y 0 -Width 960 -Height 1080 }
Write-Host "TC-04 Run 2 total wall time: $([Math]::Round($t2.TotalMilliseconds)) ms" -ForegroundColor Cyan

Write-Host ''
Write-Host 'Run 3/3 (X=240 Y=270 Width=1440 Height=540):'
$t3 = Measure-Command { & (Join-Path $ScriptDir 'Set-WacomMapping.ps1') -X 240 -Y 270 -Width 1440 -Height 540 }
Write-Host "TC-04 Run 3 total wall time: $([Math]::Round($t3.TotalMilliseconds)) ms" -ForegroundColor Cyan

$max  = [Math]::Max($t1.TotalMilliseconds, [Math]::Max($t2.TotalMilliseconds, $t3.TotalMilliseconds))
$mean = ($t1.TotalMilliseconds + $t2.TotalMilliseconds + $t3.TotalMilliseconds) / 3
Write-Host ''
Write-Host "Mean: $([Math]::Round($mean)) ms   Max: $([Math]::Round($max)) ms" -ForegroundColor Cyan

if ($max -lt 3000) {
    Write-Host 'SPIKE-02: PASS (all runs < 3000 ms)' -ForegroundColor Green
} else {
    Write-Host 'SPIKE-02: FAIL (at least one run >= 3000 ms)' -ForegroundColor Red
}

Write-Host ''
Write-Host '>> Copy the three run times and SPIKE-02 result into SPIKE-RESULTS.md Measured Latency table.'
Read-Host '>> Press Enter to continue to TC-05 (reset to full-screen)'
Write-Host ''

# ─────────────────────────────────────────────────────────────────────────────
# TC-05: Reset to Full-Screen (SPIKE-04)
# ─────────────────────────────────────────────────────────────────────────────
Write-Host '--- TC-05: Reset to Full-Screen ---' -ForegroundColor Yellow
Write-Host 'A PrefUtil dialog will appear - click OK to restore the full-screen baseline.'
& (Join-Path $ScriptDir 'Reset-WacomMapping.ps1')
Write-Host ''
Write-Host '>> Verify: draw with the stylus. Cursor should now move freely across the FULL display.'
Write-Host '>> Record in SPIKE-RESULTS.md: exit code, full-screen restoration (YES/NO/PARTIAL).'
Write-Host ''

Write-Host '=== Test run complete ===' -ForegroundColor Cyan
Write-Host ''
Write-Host 'Next steps:'
Write-Host '  1. Fill in all [placeholder] fields in spike/SPIKE-RESULTS.md'
Write-Host '     - Working Method (YES/NO/PARTIAL)'
Write-Host '     - Measured Latency table (copy from TC-04 output above)'
Write-Host '     - Admin Rights section (re-run Set-WacomMapping.ps1 from a non-elevated prompt)'
Write-Host '     - DPI / Coordinate System finding'
Write-Host '     - Service Names - service restart required? (YES/NO/UNKNOWN)'
Write-Host '     - Recommendation for Phase 2'
Write-Host '  2. Commit spike/SPIKE-RESULTS.md and spike/test-log.md (updated with raw output)'
Write-Host '  3. Reply with: ''passed'', ''partial [description]'', or ''failed [description]'''
Write-Host '     Include the three TC-04 latency values in your response.'
