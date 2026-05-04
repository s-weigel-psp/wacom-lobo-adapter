---
status: partial
phase: 02-native-messaging-host
source: [02-VERIFICATION.md]
started: 2026-05-04T12:00:00Z
updated: 2026-05-04T12:00:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. set_mapping applies visible Wacom restriction
expected: Send `chrome.runtime.sendNativeMessage('com.eurefirma.wacombridge', {command:'set_mapping', x:0, y:0, width:960, height:1080})` — stylus is physically restricted to the mapped screen region
result: [pending]

### 2. reset_mapping restores full-screen stylus movement
expected: Send `{command:'reset_mapping'}` — stylus returns to full-screen movement; PrefUtil baseline import completes successfully end-to-end
result: [pending]

### 3. MSI builds and registers in Windows registry
expected: Run `wix build installer/wacom-bridge.wxs -o wacom-bridge.msi` on Windows build machine, then `msiexec /i wacom-bridge.msi /quiet` — HKLM Chrome and Edge NativeMessagingHosts registry keys are written and point to the correct manifest paths
result: [pending]

## Summary

total: 3
passed: 0
issues: 0
pending: 3
skipped: 0
blocked: 0

## Gaps
