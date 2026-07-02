# TODO — IDT Backend Implementation

## Phase 1 — IDT emitter infra

- [x] `internal/backend/idt/table.go` — Table struct, column defs, IDT text serializer (tab + `\r\n`, control-char escaping, col type mapping: `s`/`S`/`l`/`L`/`v`/`V`/`i`/`I` + size)
- [x] Golden file tests (`testdata/*.idt`) — ASCII/non-ASCII, nulls, special chars, single/multi row
- [x] Manifest `codepage` parameter — `model.MSI.CodePage`, parser wiring, testdata fixture, docs

## Phase 2 — Core → minimal installable MSI

- [x] `tables_core.go` — 9 table builders from `*model.MSI`:
  - `Property` — ProductName, ProductCode, ProductVersion, Manufacturer, UpgradeCode, ProductLanguage=1033, SecureCustomProperties
  - `Directory` — TARGETDIR (SourceDir), ProgramFilesFolder (parent=TARGETDIR, DefaultDir=.), INSTALLDIR (parent=ProgramFilesFolder, DefaultDir=install.Directory)
  - `Component` — one per File, deterministic GUID from `ProductName|C_<Destination>` via sha256 v4-like
  - `Feature` — "Complete" (parent empty, Level=1)
  - `FeatureComponents` — Complete → each Component
  - `File` — one per payload, Sequence from index `i+1`
  - `Media` — DiskId=1, LastSequence=`len(m.Files)`, Cabinet="gomsi.cab"
  - `InstallExecuteSequence` — MVP subset (11 actions, sequences 1–210)
  - `InstallUISequence` — CostInitialize, FileCost, CostFinalize, ExecuteAction
- [x] `cab.go` — `genCAB(cabPath string, files []model.File) error` — copies sources to staging dir under `Destination` names, then `lcab -n -q`
- [x] `msibuild.go` — `runMSIBuild(msiPath, tablePaths, cabPath, product) error` — invokes `msibuild <msi> -i <table>.idt ... -a gomsi.cab <cab> -s <name> <mfr> ;1033 <code>`
- [x] `writer.go` — orchestrates: stat pass → `coreTables()` → write IDTs → `genCAB()` → emit or `runMSIBuild()`; `EmitDir` field skips msibuild; tolerates missing lcab in emit mode (e.g. Windows)
- [x] CLI `--emit <dir>` flag — stop after emitting IDT+CAB (skip msibuild), for Windows/CI dev
- [x] Test: golden IDT per core table (`testdata/core/*.idt`)
- [x] Test: lcab arg construction
- [x] Test: msibuild arg construction
- [x] Test: `writer.go` emit path (all platforms) + full build (Linux only)
- [x] Integration: `go run ./cmd/gomsi build internal/manifest/testdata/installer.yaml --emit out/` (verified on Windows)

## Phase 3 — Service tables

- [x] `tables_service.go` — build from `model.Service`:
  - `ServiceInstall` — Name, Component_, DisplayName, Description, ServiceType=16 (own process), StartType, ErrorControl=normal
  - `ServiceControl` — Name, Event=stop+delete, Wait=true
  - Add InstallServices / StopServices / DeleteServices to InstallExecuteSequence
- [x] Update fixture in `testdata/` to include service
- [x] Tests: golden IDTs, end-to-end on Linux

## Phase 4 — Parameters as public properties

- [x] `Property` rows: one per `model.Parameter` with `Property.Name`, default via `Value`
  - Required parameters → note best-effort (no client-side enforcement in MVP)
- [x] `SecureCustomProperties` — list all parameter properties so they're available in deferred/machine context
- [x] Tests: golden IDT for parameter properties

## Phase 5 — Major upgrade / auto-uninstall

- [x] `tables_upgrade.go`:
  - `Upgrade` — row per upgrade detection: UpgradeCode(Product), VersionMin=0, VersionMax=current, Attributes=0 (OnlyDetect off → removal enabled), ActionProperty=OLDPRODUCTSFOUND
  - `Property["SecureCustomProperties"]` append OLDPRODUCTSFOUND
  - `FindRelatedProducts` and `RemoveExistingProducts` in InstallExecuteSequence (RemoveExistingProducts after InstallInitialize)
- [x] Tests: golden IDT

## Phase 6 — Config via VBScript CA

- [x] `vbscript.go` — generate VBScript that:
  - Reads `Session.Property("CustomActionData")` — format: `outputPath|prop1|prop2|…`
  - For each sentinel `__GOMSI_<PROPERTY>__` in the template, replaces with the live property value
  - Writes file via `CreateTextFile`
  - Build-time: read Go template, translate `{{.PROPERTY}}` → `__GOMSI_PROPERTY__` sentinel, bake skeleton into VBScript
  - Rejects unsupported Go template constructs (`range`, `if`, `with`, etc.)
- [x] `tables_config.go`:
  - `CustomAction` — immediate SetWriteConfig (Type 51, Source=WriteConfig, Formatted Target `[INSTALLDIR]output|...`), deferred WriteConfig (Type 3078 = 6|0x400|0x800, Binary stream = VBScript, Target="WriteConfig" function)
  - `Binary` — Name=WriteConfig (PK), Data=V0 → sidecar `WriteConfig.vbs`
  - 5-column CustomAction schema (Action, Condition, Type, Source, Target) matching real MSI
- [x] Add to InstallExecuteSequence: SetWriteConfig (151, after InstallFiles), WriteConfig (205, before InstallFinalize), condition `NOT REMOVE~="ALL"`
- [x] CLI resolves `config.template` to absolute path (matching `File.Source` resolution)
- [x] Document limitation: only `{{.PROPERTY}}` substitution supported, no `range`/`if`
- [x] Tests: golden IDT for CustomAction/Binary/Sequence; golden VBScript; translate/reject unit tests; emit + full-build (Linux) writer tests

## Known issues

- [ ] **Codepage + msibuild import incompatibility (Phase 1 regression)**: gomsi prefixes the codepage on IDT row 3 (e.g. `1251\tBinary\tName`), but msitools' `libmsi` does not strip it — it treats row 3's first field as the table name. msitools instead uses a separate `_ForceCodepage.idt`. Fix: emit `_ForceCodepage.idt` via the writer when `codepage≠0`, drop the row‑3 codepage prefix in `table.go`. Currently any full build with `codepage≠0` breaks at msibuild; golden‑file and e2e tests use `CodePage=0`.

## Phase 7 — Auto-UI

- [ ] `tables_ui.go`:
  - `TextStyle` — standard UI font
  - `Property` — DefaultUIFont, ButtonText_Next, ButtonText_Back, ButtonText_Finish, ButtonText_Cancel, etc.
  - `Dialog` — WelcomeDlg, ParametersDlg, VerifyReadyDlg, ProgressDlg, ExitDlg
  - `Control` — per dialog: BannerBitmap, Next/Back/Cancel/Finish buttons + per-parameter Edit control (password type → masked attr)
  - `ControlCondition` — show/hide based on parameter.ui/required
  - `ControlEvent` — EndDialog, NewDialog linking the wizard flow
  - `InstallUISequence` — WelcomeDlg → ParametersDlg → VerifyReadyDlg → ExecuteAction
- [ ] Tests: golden IDT per table group

## Phase 8 — CI + docs

- [ ] Linux CI script (`.gitlab-ci.yml` or GitHub Actions) — install msitools + lcab, run full test suite, build reference MSI
- [ ] Update README.md with lcab/msitools prerequisites and `--emit` dev workflow
- [ ] Verify `go build ./... && go vet ./... && go test ./...` pass on both Windows (--emit) and Linux (full build)
