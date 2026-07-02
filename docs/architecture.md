# Architecture

## Pipeline

```
gomsi build <manifest>
    │
    ▼
[internal/manifest]  YAML/JSON → model.MSI
    │
    ▼
[internal/backend/idt]  Model → IDT files + CAB + msibuild
    │
    ▼
[msibuild] (external)   .idt → import → MSI database
[lcab]    (external)   payload → MSCF cabinet
    │
    ▼
.msi file
```

## Package layout

```
internal/
├── model/               ← backend-agnostic MSI struct
│   └── msi.go
├── manifest/            ← YAML/JSON → model.MSI
│   └── manifest.go
├── backend/
│   ├── backend.go       ← Writer interface (the contract)
│   └── idt/             ← MVP backend (knows IDT format)
│       ├── writer.go    ← orchestrator: tempdir → emit → CAB → msibuild
│       ├── table.go     ← IDT serialization (ONLY place with IDT format logic)
│       ├── tables_core.go    ← Property, Directory, Component, Feature, File, Media, sequences
│       ├── tables_service.go ← ServiceInstall, ServiceControl
│       ├── tables_upgrade.go ← Upgrade
│       ├── tables_config.go  ← CustomAction, Binary
│       ├── tables_ui.go      ← Dialog, Control, TextStyle, …
│       ├── cab.go            ← lcab invocation
│       ├── msibuild.go       ← msibuild invocation + summary info
│       ├── vbscript.go       ← VBScript CA generation
│       └── testdata/         ← golden IDT files, reference MSI
├── config/              ← (former build-time Render; superseded by VBScript CA at install)
└── cli/                 ← cobra commands
```

## msibuild invocation

```sh
msibuild <package.msi> \
  -i Property.idt \
  -i Directory.idt \
  -i Component.idt \
  -i Feature.idt \
  -i FeatureComponents.idt \
  -i File.idt \
  -i Media.idt \
  -i InstallExecuteSequence.idt \
  -i InstallUISequence.idt \
  -i ServiceInstall.idt \
  -i ServiceControl.idt \
  -i Upgrade.idt \
  -i CustomAction.idt \
  -i Binary.idt \
  -i TextStyle.idt \
  -i Dialog.idt \
  -i Control.idt \
  -i ControlEvent.idt \
  -a gomsi.cab tmp/gomsi.cab \
  -s "<ProductName>" "<Manufacturer>" ";1033" "{ProductCode}"
```

- `-i` imports each `.idt` file as a table. Order doesn't matter for msibuild.
- `-a` attaches a stream (the embedded CAB) with name matching `Media.Cabinet`.
- `-s` sets summary stream: Subject, Author, Template (`;1033` = Intel;1033), Revision (package code = ProductCode).

## CAB (lcab)

`lcab` produces a standard MSCF cabinet from a list of files. The IDT writer stages each file under its `Destination` name (via copy) before running lcab, so the cab-internal name matches the `File` table's `FileName` column. Sequence numbers in the `Media` and `File` tables correspond to CAB entry order.

## Codepage

The manifest accepts an optional top-level `codepage` field (integer). 0 (or absent) means auto-detect: CP1251 for Cyrillic text, CP1252 for Latin-1 supplement. Explicit values (1251, 1252) force the encoding for all IDT tables. Any string not representable in the selected codepage causes the build to fail.

## IDT file format

Per [MS docs](https://learn.microsoft.com/en-us/windows/win32/msi/archive-file-format):
- Row 1: column names, tab-separated
- Row 2: column type defs, tab-separated — lowercase = non-nullable, uppercase = nullable; `s`/`S` string, `l`/`L` localizable, `v`/`V` binary, `i`/`I` integer; followed by max chars
- Row 3: table name + primary key columns, tab-separated (codepage is set via a separate `_ForceCodepage.idt`)
- Rows 4+: data rows
- Control character escaping: NULL→21, BS→27, HT→16, LF→25, FF→24, CR→17
- Line ending: `\r\n`

## Required MSI tables (MVP)

| Table | Purpose | Source model |
|---|---|---|
| Property | Product properties + parameters | `model.Product` + `model.Parameter` |
| Directory | Folder tree under Program Files | `model.Install.Directory` |
| Component | Install unit (one per file) | GUIDs generated internally |
| Feature | Feature tree (single "Complete") | Constant |
| FeatureComponents | Feature → Component mapping | Constant |
| File | Payload file metadata | `model.File` |
| Media | Embedded cabinet reference | Generated |
| InstallExecuteSequence | Server-side action scheduling | Standard MSI actions |
| InstallUISequence | Client-side UI action scheduling | Standard + UI actions |
| ServiceInstall | Service registration | `model.Service` |
| ServiceControl | Service stop/delete on uninstall | `model.Service` |
| Upgrade | Major upgrade detection | `model.Product.UpgradeCode` |
| CustomAction | Immediate/deferred CAs | Generated (VBScript config) |
| Binary | Stored VBScript bytes | Generated |
| TextStyle | Font definitions | Standard |
| Dialog | UI dialogs | Auto-generated from `model.Parameter` |
| Control | UI controls inside dialogs | Auto-generated from `model.Parameter` |
| ControlEvent | Button → next dialog / end | Standard wizard structure |
| _ForceCodepage | String-table codepage declaration | Emitted when `codepage≠0` |

## Config CA design

Config rendering at install time uses a **VBScript CustomAction**:

1. **Build time**: gomsi reads `config.template` (Go text/template), replaces each `{{.PROPERTY}}` with a sentinel `__GOMSI_<PROPERTY>__`, bakes the skeleton into a VBScript CA.
2. **Immediate CA** (Type 51, sets property `WriteConfig`): uses Formatted Target `[INSTALLDIR]<output>|...` to capture the resolved install directory and property values. Because the deferred CA is named `WriteConfig`, the installer copies this property into `CustomActionData`.
3. **Deferred CA** (Type 3078 = Type 6 VBScript from Binary + InScript 0x400 + NoImpersonate 0x800): reads `Session.Property("CustomActionData")` (set from the `WriteConfig` property by the installer), replaces all sentinels with live values, writes config file via `FileSystemObject`.
4. **Sequencing**: between `InstallInitialize` and `InstallFinalize`; condition `NOT REMOVE~="ALL"`.

**Limitation**: only `{{.PROPERTY}}` variable substitution is supported. Go template constructs like `range`, `if`, `with` are not executed (future improvement).

## Auto-UI dialog flow

```
Welcome  →  Parameters  →  Ready to Install  →  (ExecuteAction + built-in progress)  →  Exit
```

- **WelcomeDlg**: title + description + Next/Cancel buttons
- **ParametersDlg**: one Edit control per visible parameter (`ui != "never"`), password type → masked input attribute, Back/Next/Cancel
- **VerifyReadyDlg**: "Ready to install" + Back/Next/Cancel → `EndDialog Return` triggers `ExecuteAction`
- **ExitDlg**: "Installed successfully" + Finish button (EndDialog Exit)
- Progress is shown by the built‑in MSI action dialog during `ExecuteAction` (no custom ProgressDlg)

Gating:
- UI tables are emitted only when at least one parameter has `ui != "never"` (empty/missing = "auto" = visible)
- `ui: never` → control omitted entirely (CLI-only parameter, still settable via `msiexec`)
- `type=password` → masked input (`msidbControlAttributesPasswordInput = 0x00200000`)

## Deterministic builds

Component GUIDs are derived deterministically (sha256 of `ProductName|C_<Destination>`, formatted as a v4-ish braced GUID). Product and upgrade codes are still randomly generated at parse time — this is flagged for a future fix. See `deterministicGUID()` in `tables_core.go`.

## See also

[`MSI-STANDARDS.md`](MSI-STANDARDS.md) — detailed reference for the MSI database schema, IDT format, column types, and standard tables.
