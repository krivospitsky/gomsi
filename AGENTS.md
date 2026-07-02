# AGENTS.md

Guidance for OpenCode agents working in this repository.

## Status

Scaffolded, not feature-complete. The CLI, manifest parser, model, config renderer, and a backend `Writer` interface all exist and build. The MVP `idt` backend (`internal/backend/idt`) is an intentional stub — `Write` returns `not implemented`; the msitools `.idt`/CAB/`msibuild` emission is the next work item. Module path is `github.com/krivospitsky/gomsi`.

## What this project is

`gomsi` is a Go CLI that generates Windows **MSI** installers for Go binaries, built and run **on Linux**. Think of it as "nfpm for MSI".

Hard constraints (do not violate without reason):
- Host/build environment is Linux. No Windows SDK, no Wine, no CGO.
- MVP backend is `msitools`: emit `.idt` files + CAB, then shell out to `msibuild`. Later phases may switch to `libmsi` or a pure-Go writer, but the internal `MSI` model must stay backend-agnostic — the code never touches IDT directly.
- Manifest input is YAML/JSON; config file rendering uses Go `text/template`.

## Layout & boundaries

- `internal/model` — the `MSI` struct is the central abstraction. **Backend writers consume it; nothing backend-specific (IDT rows, table names) may leak into the model, parser, config, or parameters.**
- `internal/manifest` — YAML/JSON → `model.MSI`.
- `internal/backend` — the `Writer` interface (the only contract between model and producers). `internal/backend/idt` is the MVP msitools implementation (stub).
- `internal/config` — renders the config file via `text/template`.
- `internal/cli` — cobra root + `build` subcommand. `cmd/gomsi/main.go` only calls `cli.Execute`.
- **Parameters are first-class.** Each maps simultaneously to an MSI Property, a `msiexec` CLI arg, a UI field, and a template variable. Keep that mapping unified.

## Conventions an agent would otherwise miss

- The manifest key is **`service:`** (singular) but it maps into the model's **`Services []Service`** slice. Don't "fix" the mismatch; it's deliberate to leave room for multiple services.
- `parameters` is a YAML map (unordered). The parser **sorts parameters by key** so builds are deterministic — preserve this, do not switch to unsorted map iteration.
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
```

Smoke-test the CLI end-to-end (expect the backend's `not implemented` error, which confirms parsing + wiring succeeded):

```
go run ./cmd/gomsi build internal/manifest/testdata/installer.yaml
```

There is no separate lint/typecheck/generate step yet — `go vet` is the only static check.
