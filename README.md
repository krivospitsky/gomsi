# gomsi

Minimalist, Linux-first MSI package generator for Go applications.

`gomsi` builds Windows **MSI** installers for Go binaries **on Linux** — no Windows SDK, no Wine, no CGO. Think of it as "nfpm for MSI".

## Features

- YAML/JSON manifest input
- Installs a single Go `exe` into `Program Files`
- Registers a Windows service (auto-start, stop on uninstall)
- Generates `config.json` at install time via auto-generated VBScript (syntax: `{{.PROPERTY}}` substitution in templates)
- Auto-generated wizard UI (Welcome → Parameters → Verify → Execute) — text-only dialogs, password masking, required‑field gating
- First-class install parameters → MSI Property, `msiexec` CLI arg, UI field, and template variable
- Non-ASCII support via Windows codepages: CP1251 (Cyrillic, Russian) / CP1252 (Latin), auto-detected or explicit in manifest
- Produces uninstallable MSI without any Windows tooling

## Installation

### Linux (deb/rpm)

Download the package from the [GitHub Releases](https://github.com/krivospitsky/gomsi/releases) page:

```sh
# Debian/Ubuntu
sudo dpkg -i gomsi_*.deb
# RHEL/Fedora
sudo rpm -i gomsi_*.rpm
```

The `msitools` and `lcab` dependencies are pulled automatically by the package manager.

### Docker

```sh
docker pull krivospitsky/gomsi
docker run --rm -v "$PWD:/work" -w /work krivospitsky/gomsi build installer.yaml
```

### Development (`--emit`)

If `msitools`/`lcab` are unavailable (e.g. Windows, macOS), use `--emit` to preview IDT+CAB output without building the final MSI:

```sh
gomsi build installer.yaml --emit out/
```

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

Phases 1–7 complete: core tables, service tables, parameters, upgrade/uninstall, VBScript config CA, and auto-generated UI wizards (TextStyle/Dialog/Control/ControlEvent) — gated on visible parameters. Phase 8 (CI + release) is the sole remaining phase. See [`docs/TODO.md`](docs/TODO.md) for details and [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for architecture documentation.

## Non-goals

Multi-feature installers, merge modules (`.msm`), patching (`.msp`), localization, custom UI DSL, bootstrapper (Burn), complex install sequences.
