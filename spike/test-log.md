# Spike Discovery Log

**Executed by:** [name]
**Date:** [date]
**Test machine:** [Windows version, Wacom driver version]
**Tablet:** Wacom One M (confirm model)

## 1. PrefUtil Binary

**Searched paths:**
- [ ] C:\Program Files\Tablet\Wacom\PrefUtil.exe — exists: [yes/no]
- [ ] C:\Program Files\Tablet\Wacom\Wacom_TabletUserPrefs.exe — exists: [yes/no]
- [ ] C:\Program Files (x86)\Tablet\Wacom\PrefUtil.exe — exists: [yes/no]

**Actual path used:**
```
[paste full path here]
```

**Help output:**
```
[paste PrefUtil /? or --help output here]
```

**Import flag syntax confirmed:** [--import / /import / other]
**Export flag syntax confirmed:** [--export / /export / other]

## 2. XML Mapping Element Discovery

**Diff result between baseline-A (full screen) and baseline-B (partial screen):**

Changed element name: `[element name, e.g. ScreenMapping]`

Changed attributes (fill in actual names and values):

| Attribute | Baseline-A value (full screen) | Baseline-B value (partial) |
|-----------|-------------------------------|---------------------------|
| [attr1]   | [value]                       | [value]                   |
| [attr2]   | [value]                       | [value]                   |
| [attr3]   | [value]                       | [value]                   |
| [attr4]   | [value]                       | [value]                   |

**Coordinate semantics:** [Left/Top/Right/Bottom] OR [X/Y/Width/Height] — determined from attribute names above

**XML namespace on root element:** [yes — xmlns="..."] / [no namespace]

**XPath expression to use in Set-WacomMapping.ps1:**
```
[e.g., //ScreenMapping  OR  //wacom:ScreenMapping with namespace hashtable]
```

**Multiple tablet entries in XML:** [yes — device IDs seen: ...] / [no — single device]

## 3. Wacom Service Names

```
[paste Get-Service *wacom* output here]
[paste Get-Service *tablet* output here]
```

**Service(s) confirmed running:**
- [service name] — [DisplayName] — Status: [Running/Stopped]

## 4. Admin Rights Test

**Export from non-elevated prompt:**
- Exit code: [0 / other]
- File created: [yes/no]
- Result: [elevation required / not required]

## 5. Coordinate System Notes

**Region set via GUI:** [e.g., left half of 1920×1080 = 0,0 to 960,1080]
**Values found in baseline-B:** [Left=?, Top=?, Right=?, Bottom=?]
**Are coordinates in physical pixels or logical pixels?** [physical (match physical display res) / logical (match DPI-scaled res) / unknown]

## 6. Other Findings

[Any unexpected behavior, error messages, edge cases]
