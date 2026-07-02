# AGENTS.md

Guidance for OpenCode agents working in this repository.

## Status

Pre-implementation. Only `VISION.md` exists; there is no Go code, no `go.mod`, no build/test/lint tooling yet. Treat `VISION.md` as the authoritative source of intent until code lands. When code is added, reconcile this file with the real toolchain.

## What this project is

`gomsi` is a Go CLI that generates Windows **MSI** installers for Go binaries, built and run **on Linux**. Think of it as "nfpm for MSI".

Hard constraints from the vision (do not violate without reason):
- Host/build environment is Linux. No Windows SDK, no Wine, no CGO.
- MVP backend is `msitools`: emit `.idt` files + CAB, then shell out to `msibuild`. Later phases may switch to `libmsi` or a pure-Go writer, but the **Internal MSI Model** abstraction (see `VISION.md` → "Internal Model") must stay backend-agnostic — the code never touches IDT directly.
- Manifest input is YAML/JSON; config file rendering uses Go `text/template`.

## Architecture boundaries to respect

- `MSI` model struct (`Product`, `Install`, `Files`, `Services`, `Parameters`, `Config`) is the central abstraction. Backend writers consume this model; parsing, templating, and parameter handling must not leak backend-specific types.
- **Parameters are first-class.** Each parameter simultaneously maps to an MSI Property, a `msiexec` CLI argument, a UI dialog field, and a template variable. Keep that mapping unified.

## Explicit non-goals (out of scope for MVP)

Multi-feature installers, merge modules (`.msm`), patching (`.msp`), localization, custom UI DSL, bootstrapper/Burn, complex install sequences. Do not add these without explicit approval.

## Verification

No commands exist yet. When the Go module is created, populate this section with the exact `build` / `test` (single-test invocation) / `lint` / `typecheck` commands and the required order, and verify each one actually runs in this repo.
