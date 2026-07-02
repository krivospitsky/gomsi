# Development

## Prerequisites

| Tool | Required | Install |
|---|---|---|
| Go 1.25+ | always | [go.dev](https://go.dev/) |
| [`msitools`](https://wiki.gnome.org/msitools) | real MSI build (Linux) | `apt install msitools` |
| [`lcab`](https://packages.debian.org/lcab) | real CAB build (Linux) | `apt install lcab` |


> **Note**: msitools/lcab are **Linux-only**. On Windows (your dev box) the `--emit` flag lets you develop and test the IDT emitter without them.

## Development workflow

### Windows (local dev)

1. Make changes to IDT table builders, the serializer, or other Windows-safe code.
2. Run unit/golden tests — these verify IDT text output without msibuild:
   ```
   go test ./internal/backend/idt -v
   ```
3. Use `--emit <dir>` to inspect the generated IDT files and CAB:
   ```
   go run ./cmd/gomsi build internal/manifest/testdata/installer.yaml --emit out/
   ```
   This writes all `.idt` files + cab to `out/` without calling msibuild. Inspect the IDT files and diff them with golden files.
4. Run the full Go suite:
   ```
   go build ./... && go vet ./... && go test ./...
   ```

### Linux (full build, CI)

Same workflow, but additionally `msibuild` + `lcab` are available:

1. Run all tests:
   ```
   go test ./...
   ```
   Tests that require msibuild/lcab will execute on Linux and produce real `.msi` files.

2. Build an MSI:
   ```
   go run ./cmd/gomsi build internal/manifest/testdata/installer.yaml
   ```
   Expect a `.msi` file on success.

3. Inspect the output:
   ```
   msiinfo dirs package.msi
   msidump package.msi --directory out/
   ```

## Testing strategy

| Test type | What it covers | Where it runs | Requires msibuild? |
|---|---|---|---|
| Golden IDT | Table struct → serialized `.idt` text | Windows + Linux | No |
| Arg construction | Command-line args for lcab/msibuild | Windows + Linux | No |
| Table builders | `model.MSI` → `[]Table` | Windows + Linux | No |
| Writer orchestration | Tempdir, IDT emission, CAB gen, msibuild call | Linux only | Yes |
| End-to-end | Full `gomsi build` → valid `.msi` | Linux only | Yes |

## Smoke test

```
go build ./... && go vet ./... && go test ./...

# On Windows — emit-only
go run ./cmd/gomsi build internal/manifest/testdata/installer.yaml --emit out/

# On Linux — full MSI build
go run ./cmd/gomsi build internal/manifest/testdata/installer.yaml
```

## Conventions

- IDT format logic lives **only** in `internal/backend/idt/table.go`. Every other file in the idt package uses `Table`, never raw tab/string assembly.
- Model stays backend-agnostic — no IDT column names, no table references, no MSI-specific structs leak into `internal/model`.
- Parameter property names (uppercase) are the canonical identifier everywhere: MSI properties, `msiexec` CLI args, VBScript sentinels, config skeleton substitution.
- Codepage flows from manifest → `model.MSI.CodePage` → IDT table `CodePage` field. The IDT emitter auto-detects CP1251 (Cyrillic) or CP1252 (Latin) when CodePage is 0; explicit 1251/1252 forces the codepage. Test both codepaths with inline and golden tests.
- Golden IDT files use `.idt` extension and are committed to `testdata/`.

## VBScript CA debug

If the config VBScript CA fails at install time, run the MSI with verbose logging:

```sh
msiexec /i MyAgent.msi /L*v install.log
```

Search for `WriteConfig` in the log to see CA timing, `CustomActionData` contents, and VBScript errors.
