# Project Goal

gomsi is a minimalist, Linux-first MSI package generator for Go applications designed for packaging typical Windows services built via GoReleaser, without the need to use Windows SDK or build on Windows.

# Core Problem

Today, creating MSI for Go applications requires a Windows build environment or Wine in CI. This complicates cross-platform delivery and CI/CD.

# Solution

gomsi generates MSI from a simple YAML/JSON manifest. It runs on Linux, requires no Windows SDK, and uses msitools as the backend at the MVP stage (later switchable to libmsi or a pure-Go writer).

# MVP Scope

- Single exe installation to Program Files, automatic uninstall
- Windows service: install, auto-start, stop on uninstall
- Configuration file rendered at install time via auto-generated VBScript CustomAction (Go-style `{{.PROPERTY}}` substitution)
- Install parameters as MSI properties, msiexec CLI args, UI fields, and template variables
- Auto-generated UI dialog from parameters (no custom UI DSL)

See [`TODO.md`](TODO.md) for the implementation plan and [`ARCHITECTURE.md`](ARCHITECTURE.md) for detailed design.

# Architecture

```
gomsi build
    │
    ▼
YAML manifest → Parser → Internal MSI Model → Backend Writer (IDT + CAB) → msibuild → MSI output
```

The model is backend-agnostic — see [`ARCHITECTURE.md`](ARCHITECTURE.md) for package layout and table inventory.

# Design principles

- YAML-first, Go-friendly DSL
- Deterministic builds
- No Windows dependency in CI
- Minimal abstraction over MSI
- Opinionated defaults (one app, one service)

# Success criteria

- MSI can be built on Linux CI
- Installs Go binary into Program Files
- Registers Windows service
- Accepts SERVERURL via CLI and UI
- Produces uninstallable MSI without Windows tooling

# Non-goals (MVP)

Multi-feature installers, merge modules (`.msm`), patching (`.msp`), localization, custom UI designer, bootstrapper (Burn), complex install sequences. See [`../README.md`](../README.md) for the full list.

# Future extensions (post-MVP)

MSI signing, transforms (`.mst`), multiple components/features, rollback actions, embedded bootstrapper, GUI themes, GoReleaser plugin integration.

# Mental model

Think of gomsi as "nfpm for MSI".
