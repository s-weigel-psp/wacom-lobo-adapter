#Requires -Version 5.1
<#
.SYNOPSIS
    Restricts the Wacom stylus to an arbitrary screen region by importing a modified
    preference XML via PrefUtil.exe.

.PARAMETER X
    Left edge of the mapping region in screen pixels (physical pixels).

.PARAMETER Y
    Top edge of the mapping region in screen pixels (physical pixels).

.PARAMETER Width
    Width of the mapping region in screen pixels (physical pixels).

.PARAMETER Height
    Height of the mapping region in screen pixels (physical pixels).

.EXAMPLE
    .\Set-WacomMapping.ps1 -X 0 -Y 0 -Width 960 -Height 1080
    # Restricts stylus to left half of a 1920x1080 display.

.NOTES
    Requires spike\baseline-local.Export.wacomxs to exist.
    Export it in Plan 01-01 setup: PrefUtil.exe /export <path>.

    Per D-01 (CONTEXT.md): modifies a clone of the baseline, never the baseline itself.
    Per D-02 (CONTEXT.md): uses Select-Xml XPath for element selection.

    PrefUtil path and import flag verified in Plan 01-01 (see test-log.md):
        Path : C:\Program Files\Tablet\Wacom\PrefUtil.exe
        Flag : /import

    FILE EXTENSION: The preference file MUST use the .Export.wacomxs extension.
    Using .xml causes PrefUtil to silently fail to write the file at the specified path.

    GUI DIALOG WARNING:
    PrefUtil.exe opens a native Windows dialog for every /import operation.
    The /silent flag has NO effect on /import - it only suppresses the help-screen window.
    A dialog will appear each time this script runs. Click OK to apply the mapping.
    This is a known PrefUtil limitation documented in spike/test-log.md (Plan 01-01).
    Phase 2 (C# native host) must use an alternative mechanism - see 01-01-SUMMARY.md.

    MULTIPLE ArrayElement ENTRIES:
    The preference XML typically contains 3 ArrayElement entries inside
    InputScreenAreaArray (confirmed on test machine). This script iterates ALL of them
    to ensure the mapping is applied consistently across all entries.
    AreaType is set to 1 (custom region) for every entry; AreaType 0 means full screen.

    Admin rights are NOT required (exit code 0 confirmed from non-elevated PowerShell).
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory)][int]$X,
    [Parameter(Mandatory)][int]$Y,
    [Parameter(Mandatory)][int]$Width,
    [Parameter(Mandatory)][int]$Height
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# --- Paths ---
$ScriptDir    = $PSScriptRoot
$BaselinePath = Join-Path $ScriptDir 'baseline-local.Export.wacomxs'
$TempPath     = Join-Path $env:TEMP 'wacom-mapping-temp.Export.wacomxs'

# PrefUtil path verified in Plan 01-01 (spike/test-log.md).
$PrefUtilPath = 'C:\Program Files\Tablet\Wacom\PrefUtil.exe'

# --- Precondition checks ---
Write-Host "[Set-WacomMapping] Checking prerequisites..."

if (-not (Test-Path $BaselinePath)) {
    Write-Error "baseline-local.Export.wacomxs not found at: $BaselinePath`nRun Plan 01-01 to export the baseline first:`n  PrefUtil.exe /export '$BaselinePath'"
    exit 1
}

if (-not (Test-Path $PrefUtilPath)) {
    Write-Error "PrefUtil not found at: $PrefUtilPath`nVerify the path in this script matches the test machine installation."
    exit 1
}

Write-Host "[Set-WacomMapping] Parameters: X=$X Y=$Y Width=$Width Height=$Height"
Write-Host "[Set-WacomMapping] Computed right edge: $($X + $Width)  bottom edge: $($Y + $Height)"

# --- Clone baseline (D-01) ---
# Load into [xml] so we can manipulate nodes; never write back to $BaselinePath.
Write-Host "[Set-WacomMapping] Loading baseline clone into memory..."
[xml]$doc = Get-Content -Path $BaselinePath -Raw -Encoding UTF8

# --- Locate all ScreenArea nodes via XPath (D-02) ---
# XPath confirmed in Plan 01-01 (spike/test-log.md, 01-01-SUMMARY.md).
# No XML namespace on the root element i.e. -Namespace parameter is NOT needed.
# The test machine had 3 ArrayElement entries inside InputScreenAreaArray;
# all are updated to apply the mapping consistently.
Write-Host "[Set-WacomMapping] Locating ScreenArea nodes via XPath..."

$screenAreaNodes = Select-Xml -Xml $doc -XPath '//InputScreenAreaArray/ArrayElement/ScreenArea' |
                   Select-Object -ExpandProperty Node

if ($null -eq $screenAreaNodes -or @($screenAreaNodes).Count -eq 0) {
    Write-Error "XPath '//InputScreenAreaArray/ArrayElement/ScreenArea' returned no nodes.`nVerify that the baseline file is a valid Wacom .Export.wacomxs preference export."
    exit 1
}

$nodeCount = @($screenAreaNodes).Count
Write-Host "[Set-WacomMapping] Found $nodeCount ScreenArea node(s). Updating all entries..."

# --- Set coordinate child elements and AreaType on every ScreenArea node ---
# Coordinate model (from 01-01-SUMMARY.md, Section 4):
#   ScreenOutputArea/Origin/X  - left edge in physical pixels
#   ScreenOutputArea/Origin/Y  - top edge in physical pixels
#   ScreenOutputArea/Extent/X  - width in physical pixels
#   ScreenOutputArea/Extent/Y  - height in physical pixels
#   AreaType                   - 0 = full screen, 1 = custom region
#
# Coordinates are XML child ELEMENTS with text content, NOT XML attributes.
# Assignment syntax: $node.ChildElement.GrandchildElement = [string]$value
foreach ($screenArea in @($screenAreaNodes)) {
    # Switch to custom region mode (was 0 = full screen)
    $screenArea.AreaType.InnerText = [string]1

    # Set the screen output area coordinates
    $screenArea.ScreenOutputArea.Origin.X.InnerText  = [string]$X
    $screenArea.ScreenOutputArea.Origin.Y.InnerText  = [string]$Y
    $screenArea.ScreenOutputArea.Extent.X.InnerText  = [string]$Width
    $screenArea.ScreenOutputArea.Extent.Y.InnerText  = [string]$Height
}

Write-Host "[Set-WacomMapping] Coordinates written to all $nodeCount ScreenArea node(s)."

# --- Save modified XML to temp path ---
# MUST use .Export.wacomxs extension - PrefUtil silently ignores .xml files.
$doc.Save($TempPath)
Write-Host "[Set-WacomMapping] Modified XML saved to: $TempPath"

# --- Import via PrefUtil (Pattern 2: Wait-Process, not -Wait flag) ---
# WARNING: PrefUtil will open a GUI dialog requiring the user to click OK.
# This is a known PrefUtil limitation - /silent has no effect on /import.
Write-Host "[Set-WacomMapping] Invoking PrefUtil /import (a GUI dialog will appear - click OK)..."

$elapsed = Measure-Command {
    # Import flag '/import' verified in Plan 01-01 (spike/test-log.md).
    # -PassThru is required so that $proc is populated for Wait-Process.
    # Wait-Process (not -Wait flag) avoids the 1-second poll floor (PowerShell#24709).
    $proc = Start-Process -FilePath $PrefUtilPath `
                          -ArgumentList '/import', $TempPath `
                          -PassThru
    $proc | Wait-Process
}

Write-Host "[Set-WacomMapping] PrefUtil exit code: $($proc.ExitCode)"
Write-Host "[Set-WacomMapping] PrefUtil completed in $([Math]::Round($elapsed.TotalMilliseconds)) ms"

if ($proc.ExitCode -ne 0) {
    Write-Warning "PrefUtil returned non-zero exit code: $($proc.ExitCode). Mapping may not have been applied."
    exit $proc.ExitCode
}

Write-Host "[Set-WacomMapping] Done. Stylus should now be restricted to X=$X Y=$Y Width=$Width Height=$Height."
