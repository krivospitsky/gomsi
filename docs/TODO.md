# TODO — IDT Backend Implementation

## Phase 1 — IDT emitter infra

- [x] `internal/backend/idt/table.go` — Table struct, column defs, IDT text serializer (tab + `\r\n`, control-char escaping, col type mapping: `s`/`S`/`l`/`L`/`v`/`V`/`i`/`I` + size)
- [x] Golden file tests (`testdata/*.idt`) — ASCII/non-ASCII, nulls, special chars, single/multi row
- [x] Manifest `codepage` parameter — `model.MSI.CodePage`, parser wiring, testdata fixture, docs

## Phase 2 — Core → minimal installable MSI

- [ ] `tables_core.go` — build from `*model.MSI`:
  - `Property` — ProductName, ProductCode, ProductVersion, Manufacturer, UpgradeCode, SecureCustomProperties
  - `Directory` — TARGETDIR (`SourceDir`), ProgramFilesFolder (`ProgramFilesFolder`), INSTALLDIR (`TARGETDIR.ProgramFilesFolder`)
  - `Component` — one per File, deterministic GUID from product+identity
  - `Feature` — "Complete" (parent = empty, Level = 1)
  - `FeatureComponents` — Complete → each Component
  - `File` — one per payload, Sequence from CAB order
  - `Media` — DiskId=1, LastSequence, Cabinet="gomsi.cab"
  - `InstallExecuteSequence` — CostInitialize, FileCost, CostFinalize, InstallValidate, InstallInitialize, ProcessComponents, InstallFiles, RegisterProduct, PublishFeatures, PublishProduct, InstallFinalize
  - `InstallUISequence` — CostInitialize, FileCost, CostFinalize, ExecuteAction
- [ ] `cab.go` — `genCAB(cabPath string, files []model.File) error` — wraps `lcab`
- [ ] `msibuild.go` — `runMSIBuild(msiPath, tempDir) error` — invokes `-i *.idt -a gomsi.cab cab.tmp -s name manufacturer ";1033" {uuid}`
- [ ] `writer.go` — orchestrate: MkdirTemp → emit IDTs → CAB → msibuild → cleanup
- [ ] CLI `--emit <dir>` flag — stop after emitting IDT+CAB (skip msibuild), for Windows/CI dev
- [ ] Test: golden IDT per table
- [ ] Test: lcab arg construction
- [ ] Test: msibuild arg construction
- [ ] Test: `writer.go` end-to-end (msibuild-only on Linux, else verify temp dir contents)
- [ ] Integration: `go run ./cmd/gomsi build testdata/installer.yaml` → produces `.msi` on Linux

## Phase 3 — Service tables

- [ ] `tables_service.go` — build from `model.Service`:
  - `ServiceInstall` — Name, Component_, DisplayName, Description, ServiceType=16 (own process), StartType, ErrorControl=normal
  - `ServiceControl` — Name, Event=stop+delete, Wait=true
  - Add InstallServices / StopServices / DeleteServices to InstallExecuteSequence
- [ ] Update fixture in `testdata/` to include service
- [ ] Tests: golden IDTs, end-to-end on Linux

## Phase 4 — Parameters as public properties

- [ ] `Property` rows: one per `model.Parameter` with `Property.Name`, default via `Value`
  - Required parameters → note best-effort (no client-side enforcement in MVP)
- [ ] `SecureCustomProperties` — list all parameter properties so they're available in deferred/machine context
- [ ] Tests: golden IDT for parameter properties

## Phase 5 — Major upgrade / auto-uninstall

- [ ] `tables_upgrade.go`:
  - `Upgrade` — row per upgrade detection: UpgradeCode(Product), VersionMin=0, VersionMax=current, Attributes=has-rom+has-rrp, ActionProperty=OLDPRODUCTSFOUND
  - `Property["SecureCustomProperties"]` append OLDPRODUCTSFOUND
  - `FindRelatedProducts` and `RemoveExistingProducts` in InstallExecuteSequence (RemoveExistingProducts after InstallInitialize)
- [ ] Tests: golden IDT

## Phase 6 — Config via VBScript CA

- [ ] `vbscript.go` — generate VBScript that:
  - Reads `Session.Property("CustomActionData")` — format: `outputPath|prop1|prop2|…`
  - For each sentinel `__GOMSI_<PROPERTY>__` in the template, replaces with the live property value
  - Writes file via `CreateTextFile`
  - Build-time: read Go template, translate `{{.PROPERTY}}` → `__GOMSI_PROPERTY__` sentinel, bake skeleton into VBScript
- [ ] `tables_config.go`:
  - `CustomAction` — immediate SetWriteConfig (Type 51, Formatted `[INSTALLDIR]output|...`), deferred WriteConfig (Type 6 + deferred, Binary stream = VBScript)
  - `Binary` — name=WriteConfig.vbs, data=generated VBScript
  - Add to InstallExecuteSequence: SetWriteConfig (immediate, after InstallFiles), WriteConfig (deferred, before InstallFinalize), condition `NOT REMOVE~="ALL"`
- [ ] Document limitation: only `{{.PROPERTY}}` substitution supported, no `range`/`if`
- [ ] Tests: golden IDT for CustomAction/Binary; end-to-end on Linux

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
