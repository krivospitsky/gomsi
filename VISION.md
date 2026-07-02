# Project Goal

gomsi is a minimalist, Linux-first MSI package generator for Go applications.

It is designed for packaging typical Windows services built via GoReleaser, without the need to use Windows SDK or build on Windows.

# Core Problem

Today, creating MSI for Go applications requires:

- a Windows build environment
- or Wine in CI

This complicates cross-platform delivery and CI/CD.

# Solution

gomsi generates MSI from a simple YAML/JSON manifest.

It:

- runs on Linux
- does not require Windows SDK
- uses msitools (IDT → MSI) as the backend at the MVP stage
- can later switch to libmsi or a custom writer

# MVP Scope

The first version supports only:

## Application
- single exe (Go binary)
- installation to Program Files
- automatic uninstall

## Windows Service
- install service
- auto-start
- stop on uninstall

## Configuration
- generate config.json
- via Go template
- with installation parameters

## Installation Parameters
- CLI / UI / defaults
- MSI properties

# Architecture

```
gomsi build
    │
    ▼
YAML manifest
    │
    ▼
Parser
    │
    ▼
Internal MSI Model
    │
    ▼
Backend Writer (MVP: IDT)
    │
    ▼
msibuild (msitools)
    │
    ▼
MSI output
```

# Internal Model

gomsi does NOT work with IDT directly.

Instead, an abstraction is used:

```go
type MSI struct {
    Product      Product
    Install      Install
    Files        []File
    Services     []Service
    Parameters   []Parameter
    Config       Config
}
```

# Manifest Format

## Product
```yaml
product:
  name: MyAgent
  version: 1.2.3
  manufacturer: Acme
  upgradeCode: auto
  productCode: auto
```

## Install
```yaml
install:
  directory: MyAgent
```

## Files
```yaml
files:
  - source: dist/myagent.exe
    destination: myagent.exe
```

## Service
```yaml
service:
  name: myagent
  displayName: My Agent
  description: Monitoring Agent
  start: auto
```

## Parameters (CORE FEATURE)

Parameters are first-class entities.

```yaml
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
```

Semantics:

Each parameter maps to:

- MSI Property
- CLI argument
- UI dialog field
- Template variable

## Config generation
```yaml
config:
  template: installer/config.tpl
  output: config.json
```

Template engine: Go text/template

# CLI Usage

Build MSI
```
gomsi build installer.yaml
```

Override parameters (silent install use-case)
```
msiexec /i MyAgent.msi SERVERURL=https://prod TOKEN=abc123
```

Fully silent
```
msiexec /qn /i MyAgent.msi SERVERURL=https://prod TOKEN=abc123
```

# UI Model (MVP minimal)

UI is auto-generated from parameters:

Rules:

- required=true → shown in dialog
- ui: always → always show
- ui: never → hidden (CLI-only)
- default != "" → prefilled
- type=password → masked input

No custom UI DSL in MVP.

# Backend Strategy

## Phase 1 (MVP)

Use msitools:

- generate .idt files
- generate CAB
- call msibuild

Pros:

- fast
- no CGO
- works on Linux

Cons:

- external dependency

## Phase 2

Replace with:

- libmsi OR
- pure Go MSI writer

# Non-goals (explicitly excluded from MVP)
- multi-feature installers
- merge modules (.msm)
- patching (.msp)
- localization
- custom UI designer
- bootstrapper (Burn)
- complex install sequences

# Design principles
- YAML-first, Go-friendly DSL
- deterministic builds
- no Windows dependency in CI
- minimal abstraction over MSI
- opinionated defaults (one app, one service)

# Future extensions (post-MVP)
- MSI signing
- MSI transform (.mst)
- multiple components/features
- rollback actions
- embedded bootstrapper
- GUI installer themes
- integration with GoReleaser plugin system

# Success criteria

MVP is successful if:

- MSI can be built on Linux CI
- installs Go binary into Program Files
- registers Windows service
- accepts SERVERURL via CLI and UI
- produces uninstallable MSI without Windows tooling

# Mental model

Think of gomsi as:

"nfpm for MSI"
