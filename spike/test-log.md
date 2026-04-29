# Spike Discovery Log

**Executed by:** [name]
**Date:** [date]
**Test machine:** [Windows version, Wacom driver version]
**Tablet:** Wacom One M (confirm model)

## 1. PrefUtil Binary

**Searched paths:**

- [x] C:\Program Files\Tablet\Wacom\PrefUtil.exe — exists: yes
- [x] C:\Program Files\Tablet\Wacom\Wacom_TabletUserPrefs.exe — exists: no
- [x] C:\Program Files (x86)\Tablet\Wacom\PrefUtil.exe — exists: no

**Actual path used:**

```path
C:\Program Files\Tablet\Wacom\PrefUtil.exe
```

**Help output:**

```sh
> $PREFUTIL_PATH = 'C:\Program Files\Tablet\Wacom\PrefUtil.exe'
> & $PREFUTIL_PATH /?
> & $PREFUTIL_PATH --help
>
```

Opens a small window but no shell output.
![window screenshot](./tablet-service-program-screenshot.png)

Adding `/silent` disables the dialog but still gives no output to the command line.

**Import flag syntax confirmed:** /import - opens the same dialog as with --help
/silent is not implemented on /import

**Export flag syntax confirmed:** /export - opens an export dialog different from the above (alson no /silent here)

[*Flag syntax specification*](https://developer-support.wacom.com/hc/en-us/articles/9354481821463-Run-the-Preferences-utility-from-the-command-line)

## 2. XML Mapping Element Discovery

**Diff result between baseline-A (full screen) and baseline-B (partial screen):**

Changed element name: `InputScreenAreaArray/ArrayElement`

Changed attributes (fill in actual names and values):

Base XML path: `InputScreenAreaArray/ArrayElement`

| Attribute                              | Baseline-A value (full screen) | Baseline-B value (right half) |
|----------------------------------------|--------------------------------|-------------------------------|
| `InputArea/OverlapArea/Extent/X`       | `21600`                        | `45078`                       |
| `InputArea/OverlapArea/Origin/X`       | `0`                            | `-23478`                      |
| `ScreenArea/AreaType`                  | `0`                            | `1`                           |
| `ScreenArea/MouseHeight`               | `5`                            | `1`                           |
| `ScreenArea/MouseSpeed`                | `5`                            | `1`                           |
| `ScreenArea/ScreenOutputArea/Extent/X` | `3840`                         | `1840`                        |
| `ScreenArea/ScreenOutputArea/Origin/X` | `0`                            | `2000`                        |

These values are located under each `ArrayElement` in the `InputScreenAreaArray` block. The main difference is that the right-half mapping shifts the overlap origin and output origin to the right, and also switches the screen area from `AreaType=0` to `AreaType=1` while reducing mouse height/speed.

**Coordinate semantics:** [Left/Top/Right/Bottom] OR [X/Y/Width/Height] — determined from attribute names above

**XML namespace on root element:** no namespace

**XPath expression to use in Set-WacomMapping.ps1:**

```txt
[e.g., //ScreenMapping  OR  //wacom:ScreenMapping with namespace hashtable]
```

**Multiple tablet entries in XML:** [yes — device IDs seen: ...] / [no — single device]

## 3. Wacom Service Names

```txt
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
