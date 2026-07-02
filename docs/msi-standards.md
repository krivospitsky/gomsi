# MSI Standards Reference

Reference for the Windows Installer database schema, IDT format, and MSI internals relevant to gomsi. Not exhaustive — covers only what the MVP backend touches.

## MSI database structure

An `.msi` file is an OLE Structured Storage (compound document) containing:

| Stream / Table | Purpose |
|---|---|
| `_SummaryInformation` | Package metadata (codepage, title, platform, GUID) |
| `_Strems` | Binary streams (CAB, VBScript, bitmaps, etc.) |
| `_Tables` | Internal registry of table names |
| `Property`, `Directory`, etc. | Database tables (one stream per table) |

The standard table format is the `.idt` (archive) text format, but in the storage they're stored as raw table streams. `msibuild` converts between `.idt` text and table streams.

## IDT file format

Per [MS docs](https://learn.microsoft.com/en-us/windows/win32/msi/archive-file-format).

### Structure

```
<Col1>\t<Col2>\t...\t<ColN>\r\n
<Type1>\t<Type2>\t...\t<TypeN>\r\n
<TableName>\t<PKCol1>\t...\t<PKColN>\r\n
<Val11>\t<Val12>\t...\t<Val1N>\r\n
<Val21>\t<Val22>\t...\t<Val2N>\r\n
```

- **Row 1**: column names, tab-separated
- **Row 2**: column type definitions, tab-separated
- **Row 3 (ASCII data)**: table name + primary key column names, tab-separated
- **Row 3 (non-ASCII data)**: numeric codepage `\t` table name `\t` primary key columns
- **Row 4+**: data rows

### Column type syntax

`<nullable><type><size>`

| Flag | Meaning |
|---|---|
| `s` | Non-nullable string |
| `S` | Nullable string |
| `l` | Non-nullable localizable string |
| `L` | Nullable localizable string |
| `v` | Non-nullable binary |
| `V` | Nullable binary |
| `i` | Non-nullable integer |
| `I` | Nullable integer |

Size for strings/binary: `0`–`255` (0 means unlimited). For integers: `2` (short, `int16`) or `4` (long, `int32`).

Examples: `s72`, `S255`, `L50`, `v0`, `i2`, `I4`.

### Primary key

PK columns cannot contain nulls. The combination of PK column values must be unique within the table. Tables with no PK (e.g. `Property`) use the first column as de facto PK.

### Control character escaping

| Raw byte | IDT encoding | Meaning |
|---|---|---|
| `\x00` (NULL) | `21` | Null |
| `\x08` (BS) | `27` | Back Space |
| `\x09` (HT) | `16` | Tab |
| `\x0A` (LF) | `25` | Line Feed |
| `\x0C` (FF) | `24` | Form Feed |
| `\x0D` (CR) | `17` | Carriage Return |

Encoding is the decimal representation of the byte as a string.

### Line ending

`\r\n` (CR + LF).

### Codepage

For ASCII content (no non-ASCII characters), the third row is `TableName\tPKcol1\t...`. For non-ASCII content, the third row is `CodePageNumber\tTableName\tPKcol1\t...`. The ASCII range (0x00–0x7F) is always ASCII — non-ASCII characters are stored as multi-byte sequences in the file's codepage.

## Summary information stream

Set via `msibuild -s <name> [author] [template] [uuid]`. msibuild's `init_suminfo` sets defaults for the rest.

| PID | Name | Value |
|---|---|---|
| 2 | Title | `"Installation Database"` (default) |
| 3 | Subject | Set by `-s <name>` |
| 4 | Author | Set by `-s <author>` |
| 5 | Keywords | `"Installer, MSI"` (default) |
| 7 | Template | Set by `-s <template>` — `";1033"` = Intel, English |
| 9 | Revision (PackageCode) | Set by `-s <uuid>` |
| 14 | PageCount | Set to 200 by msibuild (MSI version 2.0) |
| 15 | WordCount | 0 (default) |
| 18 | AppName | `"libmsi msibuild"` (default) |

The **PackageCode** (`PID_REVNUMBER`) is a GUID that uniquely identifies the package. Each build with different properties should have a different package code. This is separate from `ProductCode` (which identifies the product family).

## MSI properties

Properties are global key-value pairs. Naming conventions:

| Convention | Example | Access from msiexec |
|---|---|---|
| Public (uppercase) | `SERVERURL` | `msiexec /i pkg.msi SERVERURL=...` |
| Private (mixed/lowercase) | `CustomActionData` | Not settable from command line |
| Secure | `SECURE_PROP` | Survives into deferred/execute sequence |

**SecureCustomProperties** — a list property (semicolon-separated property names) that tells the installer which public properties to pass to the execution phase (where they become available to deferred CAs and VBScript via `CustomActionData`). Any property accessed by a deferred CA must be listed here.

## Table reference (MVP)

### Property

| Column | Type | PK |
|---|---|---|
| Property | `s72` | Yes |
| Value | `S0` | No |

Standard entries: `ProductName`, `ProductCode`, `ProductVersion`, `ProductLanguage`, `Manufacturer`, `UpgradeCode`, `SecureCustomProperties`.

### Directory

| Column | Type | PK |
|---|---|---|
| Directory | `s72` | Yes |
| Directory_Parent | `S72` | No |
| DefaultDir | `S255` | No |

Hierarchy is a parent-pointer tree rooted at `TARGETDIR`. Constants:

| Directory | Parent | DefaultDir |
|---|---|---|
| `TARGETDIR` | (none) | `SourceDir` |
| `ProgramFilesFolder` | `TARGETDIR` | `.` |
| `INSTALLDIR` | `ProgramFilesFolder` | `<product.Install.Directory>` |

### Component

| Column | Type | PK |
|---|---|---|
| Component | `s72` | Yes |
| ComponentId | `S38` | No |
| Directory_ | `S72` | No |
| Attributes | `i2` | No |
| Condition | `S255` | No |
| KeyPath | `S72` | No |

Attributes bitmask:

| Bit | Value | Meaning |
|---|---|---|
| 0 | 0x0000 | Local only |
| 1 | 0x0001 | Source only |
| 2 | 0x0002 | Optional |
| 3 | 0x0004 | Registry key path |
| 4 | 0x0008 | COM |
| 8 | 0x0100 | Permanent (not removed on uninstall) |
| 11 | 0x0800 | Disable registry reflection |
| 15 | 0x4000 | 64-bit component |

`ComponentId` — a GUID that uniquely identifies this component version across versions. Deterministic derivation from ProductName + component name is recommended.

### Feature

| Column | Type | PK |
|---|---|---|
| Feature | `s38` | Yes |
| Feature_Parent | `S38` | No |
| Title | `S64` | No |
| Description | `S255` | No |
| Display | `i2` | No |
| Level | `i2` | No |
| Attributes | `i2` | No |

Minimal entry: `Feature=Complete`, `Level=1`.

### FeatureComponents

| Column | Type | PK |
|---|---|---|
| Feature_ | `s38` | Yes |
| Component_ | `s72` | Yes |

Maps features to components (M:N). Add one row per (Feature, Component).

### File

| Column | Type | PK |
|---|---|---|
| File | `s72` | Yes |
| Component_ | `s72` | No |
| FileName | `S255` | No |
| FileSize | `i4` | No |
| Version | `S72` | No |
| Language | `S20` | No |
| Attributes | `i4` | No |
| Sequence | `i2` | No |

`Sequence` corresponds to the position in the CAB and the `Media.LastSequence` range. For a single-cab package, files should have sequence numbers 1..N.

`File` column is the primary key — typically the file identifier (e.g. `F_myagent.exe`). `FileName` is the short\|long filename for the target (e.g. `myagent.exe`).

### Media

| Column | Type | PK |
|---|---|---|
| DiskId | `i2` | Yes |
| LastSequence | `i2` | No |
| DiskPrompt | `S64` | No |
| Cabinet | `S255` | No |
| VolumeLabel | `S32` | No |

For embedded cab: `DiskId=1`, `Cabinet=gomsi.cab`, `LastSequence=<max file Sequence>`.

### InstallExecuteSequence

| Column | Type | PK |
|---|---|---|
| Action | `s72` | Yes |
| Condition | `S255` | No |
| Sequence | `i2` | No |

Standard server-side actions (minimum viable set):

| Action | Sequence |
|---|---|
| `CostInitialize` | 1 |
| `FileCost` | 2 |
| `CostFinalize` | 3 |
| `InstallValidate` | 10 |
| `FindRelatedProducts` | 20 |
| `InstallInitialize` | 50 |
| `ProcessComponents` | 60 |
| `UnpublishFeatures` | 70 |
| `RemoveFiles` | 80 |
| `RemoveFolders` | 90 |
| `CreateFolders` | 100 |
| `MoveFiles` | 110 |
| `InstallServices` | 120 |
| `StopServices` | 130 |
| `DeleteServices` | 140 |
| `InstallFiles` | 150 |
| `WriteRegistryValues` | 160 |
| `RegisterUser` | 170 |
| `RegisterProduct` | 180 |
| `PublishFeatures` | 190 |
| `PublishProduct` | 200 |
| `InstallFinalize` | 210 |
| `RemoveExistingProducts` | 220 |

Conditions:
- `NOT Installed` — only run during install (not uninstall)
- `REMOVE~="ALL"` — full uninstall
- `NOT REMOVE~="ALL"` — during install or upgrade
- `REINSTALL` — during repair

### InstallUISequence

Same columns as InstallExecuteSequence. Standard UI actions:

| Action | Sequence |
|---|---|
| `CostInitialize` | 1 |
| `FileCost` | 2 |
| `CostFinalize` | 3 |
| `ExecuteAction` | 4 |

Custom UI dialogs replace the built-in wizard between CostInitialize and ExecuteAction.

### ServiceInstall

| Column | Type | PK |
|---|---|---|
| ServiceInstall | `s72` | Yes |
| Name | `S255` | No |
| Component_ | `s72` | No |
| DisplayName | `S255` | No |
| Description | `S255` | No |
| ServiceType | `i2` | No |
| StartType | `i2` | No |
| ErrorControl | `i2` | No |
| LoadOrderGroup | `S32` | No |
| Dependencies | `S255` | No |
| StartName | `S255` | No |
| Password | `S255` | No |

ServiceType:
- 16 = `SERVICE_WIN32_OWN_PROCESS` (standard for single-service apps)
- 32 = `SERVICE_WIN32_SHARE_PROCESS`

StartType:
- 2 = `SERVICE_AUTO_START`
- 3 = `SERVICE_DEMAND_START` (manual)
- 4 = `SERVICE_DISABLED`

ErrorControl:
- -1 = no error (deprecated)
- 0 = `SERVICE_ERROR_IGNORE` (log only)
- 1 = `SERVICE_ERROR_NORMAL` (log + message box)
- 2 = `SERVICE_ERROR_CRITICAL` (log + reboot)

### ServiceControl

| Column | Type | PK |
|---|---|---|
| ServiceControl | `s72` | Yes |
| Name | `S255` | No |
| Component_ | `s72` | No |
| Event | `i4` | No |
| Arguments | `S255` | No |
| Wait | `i2` | No |

Event bitmask:
- 1 = `SERVICE_CONTROL_STOP` (started service)
- 2 = `SERVICE_CONTROL_PAUSE` (pause)
- 4 = `SERVICE_CONTROL_DELETE` (uninstall removes service)
- 8 = `SERVICE_CONTROL_UNINSTALL_CONTINUE` (uninstall continues)

Standard uninstall: `Event=5` (`STOP | DELETE`).

Wait:
- 0 = don't wait for service to reach the requested state
- 1 = wait (recommended)

### CustomAction

| Column | Type | PK |
|---|---|---|
| Action | `s72` | Yes |
| Condition | `S255` | No |
| Type | `i2` | No |
| Source | `S72` | No |
| Target | `S255` | No |

Type encodes the action kind + flags:

| Type | Meaning |
|---|---|
| 6 | VBScript from Binary table (`Source` = Binary.Name) |
| 22 | DOS command / executable |
| 34 | JScript from Binary table |
| 38 | Inline VBScript (`Target` = script text) |
| 50 | Inline JScript |
| 51 | Property set (Formatted) |
| 3078 | Deferred + VBScript from Binary (6 + 0x400 InScript + 0x800 NoImpersonate) |
| 1026 | Deferred + set property (51 + 0x400 + 0x800) |

Key flags for CustomAction:
- `msidbCustomActionTypeInScript` = `0x400` (deferred — runs in the execute/silent sequence)
- `msidbCustomActionTypeNoImpersonate` = `0x800` (runs as LocalSystem, no user impersonation)

A deferred CA (0x400) reads its input from `Session.Property("CustomActionData")`, which the installer populates from the property named after the CA. To pass data: create a Type 51 immediate CA that sets the property `<DeferredCAName>` to a Formatted string.

### Binary

| Column | Type | PK |
|---|---|---|
| Name | `s72` | Yes |
| Data | `V0` | No |

The `V0` column type is a binary‑object reference. On import, msibuild reads the actual bytes from a sidecar file at `<TableName>/<cellValue>` (e.g. `Binary/WriteConfig.vbs`). Stores binary data (VBScript CA, bitmaps, etc.) as streams in `_Streams`.

### Upgrade

| Column | Type | PK |
|---|---|---|
| UpgradeCode | `s38` | Yes |
| VersionMin | `S255` | No |
| VersionMax | `S255` | No |
| Language | `S255` | No |
| Attributes | `i4` | No |
| Remove | `S1` | No |
| ActionProperty | `S72` | No |

Attributes bitmask:
- 1 = Migrate features (move to new product)
- 2 = Only detect, don't remove
- 4 = Ignore language
- 8 = VersionMax inclusive
- 16 = VersionMin inclusive
- 256 = Allow product downgrade

Standard major upgrade: `Attributes=9` (versionmax inclusive + language independent), `VersionMin=0`, `VersionMax=<current version>`, `UpgradeCode=<Product.UpgradeCode>`. Action property (e.g. `OLDPRODUCTSFOUND`) is set when an older product is detected.

The `RemoveExistingProducts` action uses the action property list to remove detected products.

### Dialog

| Column | Type | PK |
|---|---|---|
| Dialog | `s72` | Yes |
| HCentering | `i2` | No |
| VCentering | `i2` | No |
| Width | `i2` | No |
| Height | `i2` | No |
| Attributes | `i4` | No |
| Title | `S72` | No |
| Control_First | `S72` | No |
| Control_Default | `S72` | No |
| Control_Cancel | `S72` | No |

Dialog Attributes bitmask (key bits):
- 1 = Visible (modal dialog)
- 2 = Track disk space
- 4 = Use custom bitmap
- 8 = Error dialog
- 16 = Keep modeless
- 32 = No fail

### Control

| Column | Type | PK |
|---|---|---|
| Dialog_ | `s72` | Yes |
| Control | `s72` | Yes |
| Type | `s72` | No |
| X | `i2` | No |
| Y | `i2` | No |
| Width | `i2` | No |
| Height | `i2` | No |
| Attributes | `i4` | No |
| Property | `S72` | No |
| Text | `S0` | No |
| Control_Next | `S72` | No |
| Help | `S255` | No |

Control types for MVP:

| Type | Purpose |
|---|---|
| `Text` | Static label (e.g. parameter title) |
| `Edit` | Text input (plain or password-masked) |
| `PushButton` | Navigation / action buttons |
| `MaskedEdit` | Password input |
| `CheckBox` | Boolean option |
| `Bitmap` | Banner image |
| `GroupBox` | Visual grouping |
| `ProgressBar` | Install progress |
| `VolumeSelectCombo` | Drive selection |

Password masking: use `MaskedEdit` control or set the `msidbControlAttributesPasswordInput` bit (0x00200000) on an `Edit` control.

### ControlCondition

| Column | Type | PK |
|---|---|---|
| Dialog_ | `s72` | Yes |
| Control_ | `s72` | Yes |
| Action | `s72` | Yes |
| Condition | `S255` | No |

Actions:
- `Show` — make visible
- `Hide` — hide
- `Enable` — enable interaction
- `Disable` — disable interaction

Common condition: `SERVERURL = ""` → Disable Next button.

### ControlEvent

| Column | Type | PK |
|---|---|---|
| Dialog_ | `s72` | Yes |
| Control_ | `s72` | Yes |
| Event | `s72` | Yes |
| Argument | `S255` | No |
| Condition | `S255` | No |
| Ordering | `i2` | No |

Events:
- `EndDialog` → `"Return"`, `"Exit"`, `"Retry"`, `"Ignore"` (closes dialog)
- `NewDialog` → dialog name (navigate)
- `DoAction` → action to execute
- `SetProgress` → `"1"` / `"0"` (advance progress)

Standard wizard flow: WelcomeDlg → ParametersDlg → VerifyReadyDlg → ExecuteAction (via `DoAction`) → ExitDlg.

### TextStyle

| Column | Type | PK |
|---|---|---|
| TextStyle | `s72` | Yes |
| FaceName | `S255` | No |
| Size | `i2` | No |
| Color | `I4` | No |
| StyleBits | `i2` | No |

Minimal: one text style for the default UI font (referenced by `Property[DefaultUIFont]`).

## CAB format

An MSI cabinet is a standard [MSCF cabinet file](https://learn.microsoft.com/en-us/previous-versions/bb417343(v=msdn.10)). 

Key facts:
- The cabinet is stored as a stream in `_Streams` with a name matching `Media.Cabinet`
- Each file in the cab is identified internally by its short filename (the `File.FileName` value)
- Sequence numbers in `File` table → offset within the cab
- The `Folder` object in the cab is simple: a single folder covers all `File.Sequence` values ≤ `Folder.Offset`

`lcab` creates standard MSCF cabinets with a single folder.

## Component GUID lifecycle

- Each component has a `ComponentId` GUID that must remain stable across versions of the *same* component. If the component's identity (files, registry keys) changes, a new `ComponentId` is needed.
- The `ComponentId` is unrelated to `ProductCode`, `UpgradeCode`, or `PackageCode`.
- For deterministic builds, derive component GUIDs (e.g. hash of ProductName + component name).
- KeyPath: the file or registry value that tells the installer whether the component is present. Usually the first file for a file-only component.

## Major upgrade logic

1. `FindRelatedProducts` (Sequence ≈ 20) — scans for products matching the `Upgrade` table rows. Sets the action property (e.g. `OLDPRODUCTSFOUND`) to a list of product codes found.
2. The new product is installed normally.
3. `RemoveExistingProducts` — after `InstallInitialize` (or at another point), removes detected products.

Two common scheduling modes:
- **Schedule after `InstallInitialize`**: removes old product after new files are in place (preserves installed file modes if files have matching version). User sees no gap.
- **Schedule before `InstallInitialize`**: removes old product first, then installs new one (cleaner but there's a window with no product).

## Deferred CustomAction data flow

```
msiexec command line or UI → property values → SecureCustomProperties list
    ↓
Immediate CA (Type 51): sets "WriteConfig" property = Formatted "[INSTALLDIR]...|[SERVERURL]"
    ↓ installer passes the property value into deferred environment
Deferred CA: reads Session.Property("CustomActionData")
    (this contains the value of property "WriteConfig")
```

The immediate CA name must match the deferred CA name — the installer copies the matching property value into `CustomActionData` for the deferred CA.

## Useful links

- [MSI Archive File Format (IDT)](https://learn.microsoft.com/en-us/windows/win32/msi/archive-file-format)
- [Windows Installer Database](https://learn.microsoft.com/en-us/windows/win32/msi/windows-installer-database)
- [Windows Installer on GitHub (reference sources)](https://github.com/MicrosoftDocs/win32-pr/tree/live/desktop-src/Msi)
- [msitools / libmsi (GNOME GitLab)](https://gitlab.gnome.org/GNOME/msitools)
- [Column Definition Format](https://learn.microsoft.com/en-us/windows/win32/msi/column-definition-format)
- [Summary Information Stream](https://learn.microsoft.com/en-us/windows/win32/msi/summary-information-stream)
- [Component KeyPath](https://learn.microsoft.com/en-us/windows/win32/msi/component-table)
