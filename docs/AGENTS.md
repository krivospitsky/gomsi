# AGENTS.md

Guidance for OpenCode agents working in this repository.

## Status

Backend implementation in progress. The CLI, manifest parser, model, config renderer, and `Writer` interface are stable. The IDT backend (`internal/backend/idt`) is being implemented per the phased plan in [`TODO.md`](TODO.md) with the following resolved design decisions:

| Decision | Choice |
|----------|--------|
| CAB generation | `lcab` (external, apt-installable) |
| Config at install | VBScript CustomAction (sentinel substitution) |
| UI scope | Auto-generated dialogs from parameters (included in MVP) |
| Non-ASCII codepage | CP1251 (Cyrillic, Russian-first) / CP1252 (Latin) auto-detect; explicit `codepage` in manifest |


Resolved forks are recorded as design facts — do not revisit without reason.

## What this project is

`gomsi` is a Go CLI that generates Windows **MSI** installers for Go binaries, built and run **on Linux**. Think of it as "nfpm for MSI".

Hard constraints (do not violate without reason):
- Host/build environment is Linux. No Windows SDK, no Wine, no CGO.
- External deps: `msitools` (msibuild) + `lcab`. Both are Linux-only.
- MVP backend is `msitools`: emit `.idt` files + CAB, then shell out to `msibuild`. Later phases may switch to `libmsi` or a pure-Go writer, but the internal `MSI` model must stay backend-agnostic — the code never touches IDT directly.
- Manifest input is YAML/JSON; config rendering at install time uses an auto-generated VBScript CustomAction (the Go `config.Render` was a build-time helper; the VBScript CA is the actual install-time renderer).

## Layout & boundaries

- `internal/model` — the `MSI` struct is the central abstraction. **Backend writers consume it; nothing backend-specific (IDT rows, table names) may leak into the model, parser, or CLI.**
- `internal/manifest` — YAML/JSON → `model.MSI`.
- `internal/backend` — the `Writer` interface (the only contract between model and producers). `internal/backend/idt` is the MVP msitools implementation.
  - `internal/backend/idt/table.go` — IDT serialization (the **only** file that understands the `.idt` text format)
  - `internal/backend/idt/tables_core.go` — Property, Directory, Component, Feature, FeatureComponents, File, Media, sequences
  - `internal/backend/idt/tables_service.go` — ServiceInstall, ServiceControl
  - `internal/backend/idt/tables_upgrade.go` — Upgrade
  - `internal/backend/idt/tables_config.go` — CustomAction, Binary
  - `internal/backend/idt/tables_ui.go` — Dialog, Control, ControlCondition, ControlEvent, TextStyle
  - `internal/backend/idt/cab.go` — `lcab` invocation
  - `internal/backend/idt/msibuild.go` — `msibuild` invocation + summary info
  - `internal/backend/idt/vbscript.go` — VBScript CA generation
  - `internal/backend/idt/writer.go` — orchestrator (tempdir → emit → CAB → msibuild → cleanup)
- `internal/config` — build-time config rendering helper (superseded at install by VBScript CA).
- `internal/cli` — cobra root + `build` subcommand. `cmd/gomsi/main.go` only calls `cli.Execute`.
- **Parameters are first-class.** Each maps simultaneously to an MSI Property, a `msiexec` CLI arg, a UI field, and a VBScript sentinel. Keep that mapping unified.

## Conventions an agent would otherwise miss

- The manifest key is **`service:`** (singular) but it maps into the model's **`Services []Service`** slice. Don't "fix" the mismatch; it's deliberate to leave room for multiple services.
- `parameters` is a YAML map (unordered). The parser **sorts parameters by key** so builds are deterministic — preserve this, do not switch to unsorted map iteration.
- The manifest accepts a top-level `codepage` field (int, 0=auto). It flows into `model.MSI.CodePage` and is used by the IDT emitter for non-ASCII text encoding (CP1251 for Cyrillic, CP1252 for Latin).
- `upgradeCode`/`productCode: auto` (or empty) are resolved to freshly generated braced GUIDs **at parse time**; explicit values are preserved verbatim.
- In config templates, a parameter is referenced by its **`property`** name (e.g. `{{.SERVERURL}}`), **not** its manifest key (e.g. `serverUrl`).
- The CLI is cobra-based; add subcommands in `internal/cli` and register them on the root in an `init()`.

## Explicit non-goals (out of scope for MVP)

WiX compat, multi-feature installers, merge modules (`.msm`), patching (`.msp`), localization, custom UI DSL, bootstrapper/Burn, complex install sequences. Do not add these without explicit approval.

## Verification

Run from repo root, in this order:

```
go build ./...
go vet ./...
go test ./...
```

Single test / single package:

```
go test ./internal/manifest -run TestParse_YAML
go test ./internal/manifest
go test ./internal/backend/idt
```

Smoke-test the CLI end-to-end. On Windows (where msibuild/lcab are absent), use `--emit` to verify IDT + CAB output without the final msibuild step:

```
go run ./cmd/gomsi build internal/manifest/testdata/installer.yaml --emit out/
```

On Linux, the full backend runs and produces a `.msi`:

```
go run ./cmd/gomsi build internal/manifest/testdata/installer.yaml
```

There is no separate lint/typecheck/generate step yet — `go vet` is the only static check.
