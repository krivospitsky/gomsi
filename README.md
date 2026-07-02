# gomsi

Minimalist, Linux-first MSI package generator for Go applications.

`gomsi` builds Windows **MSI** installers for Go binaries **on Linux** — no Windows SDK, no Wine, no CGO. Think of it as "nfpm for MSI".

## Features

- YAML/JSON manifest input
- Installs a single Go `exe` into `Program Files`
- Registers a Windows service (auto-start, stop on uninstall)
- Generates `config.json` at install time via auto-generated VBScript (syntax: `{{.PROPERTY}}` substitution in templates)
- First-class install parameters → MSI Property, `msiexec` CLI arg, UI field, and template variable
- Non-ASCII support via Windows codepages: CP1251 (Cyrillic, Russian) / CP1252 (Latin), auto-detected or explicit in manifest
- Produces uninstallable MSI without any Windows tooling

## Quick start

```sh
gomsi build installer.yaml
```

Install silently:

```sh
msiexec /i MyAgent.msi SERVERURL=https://prod TOKEN=abc123
```

Fully silent:

```sh
msiexec /qn /i MyAgent.msi SERVERURL=https://prod TOKEN=abc123
```

## Manifest example

```yaml
codepage: 1251

product:
  name: MyAgent
  version: 1.2.3
  manufacturer: Acme
  upgradeCode: auto
  productCode: auto

install:
  directory: MyAgent

files:
  - source: dist/myagent.exe
    destination: myagent.exe

service:
  name: myagent
  displayName: My Agent
  description: Monitoring Agent
  start: auto

parameters:
  serverUrl:
    property: SERVERURL
    type: string
    title: Server URL
    required: true
    default: ""
    validate: url
    ui: auto
  token:
    property: TOKEN
    type: password
    required: false

config:
  template: installer/config.tpl
  output: config.json
```

## Backend

MVP uses [`msitools`](https://wiki.gnome.org/msitools) + [`lcab`](https://github.com/riencroonenborghs/lcab): generate `.idt` files + CAB via `lcab`, then call `msibuild`. Dependencies (both Linux-only):

```sh
apt install msitools lcab
```

Future phases may switch to `libmsi` or a pure-Go MSI writer.

## Status

Backend implementation in progress (Phases 1–3 complete: core tables, service tables, CAB, msibuild, writer orchestration). See [`docs/TODO.md`](docs/TODO.md) for the implementation plan and [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for architecture documentation.

## Non-goals

Multi-feature installers, merge modules (`.msm`), patching (`.msp`), localization, custom UI DSL, bootstrapper (Burn), complex install sequences.
