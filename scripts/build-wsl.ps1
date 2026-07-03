#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build an MSI using WSL (requires msitools + gcab inside WSL).
.PARAMETER Manifest
    Path to the installer manifest YAML (default: internal/manifest/testdata/installer.yaml).
.PARAMETER Output
    Path for the resulting MSI file (default: gomsi.msi in the working directory).
.PARAMETER WslDistro
    WSL distro name (default: Ubuntu-22.04).
.EXAMPLE
    .\scripts\build-wsl.ps1
    .\scripts\build-wsl.ps1 -Manifest myapp.yaml -Output out\myapp.msi
#>
param(
    [string]$Manifest = "internal/manifest/testdata/installer.yaml",
    [string]$Output = "",
    [string]$WslDistro = "Ubuntu-22.04"
)

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$linuxBin = Join-Path (Join-Path $repoRoot "bin") "gomsi-linux"

# Resolve output path.
if (-not $Output) {
    $Output = Join-Path (Get-Location) "gomsi.msi"
}
$outDir = Split-Path $Output -Parent
if ($outDir -and (Test-Path $outDir -PathType Container)) {
    $outDir = Resolve-Path $outDir
} else {
    $outDir = Get-Location
}
$outName = Split-Path $Output -Leaf
$Output = Join-Path $outDir $outName

# Resolve manifest to absolute Windows path.
$manifestAbs = Resolve-Path (Join-Path $repoRoot $Manifest)

Write-Host "=== gomsi WSL build ===" -ForegroundColor Cyan
Write-Host "Manifest : $manifestAbs"
Write-Host "Output   : $Output"
Write-Host "Distro   : $WslDistro"

# ── Step 1: cross-compile gomsi for Linux ──
Write-Host "`n[1/3] Cross-compiling gomsi for linux/amd64 ..." -ForegroundColor Yellow
$env:GOOS = "linux"
$env:GOARCH = "amd64"
try {
    $build = go build -o $linuxBin ./cmd/gomsi 2>&1
    if ($LASTEXITCODE -ne 0) { throw $build }
} finally {
    Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
}
Write-Host "       -> $linuxBin" -ForegroundColor Green

# ── Step 2: convert paths to WSL (Linux) paths ──
# D:\Projects\gomsi → /mnt/d/Projects/gomsi
function To-WslPath([string]$winPath) {
    $drive = $winPath[0].ToString().ToLower()
    $rest = $winPath.Substring(2) -replace '\\', '/'
    return "/mnt/$drive$rest"
}

$wslManifest  = To-WslPath $manifestAbs
$wslLinuxBin  = To-WslPath $linuxBin
$wslOutput    = (To-WslPath $outDir) + "/" + $outName
$wslRepoRoot  = To-WslPath $repoRoot

# ── Step 3: run gomsi inside WSL ──
Write-Host "`n[2/3] Running gomsi inside WSL ($WslDistro) ..." -ForegroundColor Yellow
$wslScript = @"
cd "$wslRepoRoot" && \
chmod +x "$wslLinuxBin" && \
"$wslLinuxBin" build "$wslManifest" -o "$wslOutput"
"@
$result = wsl -d $WslDistro -- bash -c "$wslScript" 2>&1
$exitCode = $LASTEXITCODE
if ($exitCode -ne 0) {
    Write-Host $result -ForegroundColor Red
    Write-Host "       -> WSL build FAILED (exit $exitCode)" -ForegroundColor Red
    exit $exitCode
}
Write-Host $result

# ── Step 4: verify the MSI exists ──
if (Test-Path $Output) {
    $size = (Get-Item $Output).Length
    Write-Host "       -> $Output ($size bytes)" -ForegroundColor Green
} else {
    Write-Host "       -> MSI not found at $Output" -ForegroundColor Red
    exit 1
}

Write-Host "`n=== Build complete ===" -ForegroundColor Cyan
