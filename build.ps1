# Builds Visitron for Windows: the application, then the setup program that
# carries it. Ported from SymDiary.
#
#   ./build.ps1                 the gate, then the application and the setup program
#   ./build.ps1 -SkipInstaller  the application alone
#   ./build.ps1 -Fast           no gate, for a working loop
#
# Outputs:
#   build/bin/Visitron.exe            the application
#   dist-installer/VisitronSetup.exe  the setup program, once it exists
#
# BuildPilot finds this file and runs it with no arguments, no profile and no
# prompts; it reads the exit code alone, 0 meaning succeeded. So nothing here
# asks a question and every native command's exit code is checked, since
# PowerShell does not stop on one by itself. BuildPilot's Launch installer
# looks for VisitronSetup.exe in dist-installer, which is where it is written.
#
# The gate runs first and cannot be skipped by any switch that ships: -Fast is
# for a working loop and says so in its output, so a release is never cut from
# a tree nobody verified.
#
# The version comes from VERSION and reaches the binary through -ldflags -X,
# which only writes to a var: against a const it silently does nothing, which is
# why main.appVersion is declared var.
param(
    [switch]$Fast,
    [switch]$SkipInstaller
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

$version = (Get-Content (Join-Path $root 'VERSION') -Raw).Trim()
if (-not $version) { throw 'VERSION is empty' }

if ($Fast) {
    Write-Host "Building $version without the gate (working build, not a release)..."
} else {
    & (Join-Path $root 'test.ps1')
    if ($LASTEXITCODE -ne 0) { throw "the gate failed with exit code $LASTEXITCODE" }
    Write-Host "Building $version..."
}

wails build -ldflags "-X main.appVersion=$version"
if ($LASTEXITCODE -ne 0) { throw "wails build failed with exit code $LASTEXITCODE" }

$binary = Join-Path $root 'build\bin\Visitron.exe'
if (-not (Test-Path $binary)) { throw "the build reported success but $binary is not there" }
Write-Host ("Built {0} ({1:N0} bytes)" -f $binary, (Get-Item $binary).Length)

if ($SkipInstaller) { return }

# The setup program is a second Wails application in the same module, under
# installer/. Until it is written there is nothing to package, which is said
# rather than failed: the application above is a complete build on its own.
if (-not (Test-Path (Join-Path $root 'installer\main.go'))) {
    Write-Host 'There is no setup program in this repository yet, so only the application was built.'
    return
}

# Section 4 of the GNU GPL asks that a copy of the licence be given to every
# recipient along with the program, so the licence travels in the payload and
# lands in the install folder beside the executable.
Copy-Item (Join-Path $root 'LICENSE') (Join-Path $root 'build\bin\LICENSE') -Force

Write-Host 'Packaging the application as the setup payload...'
$payload = Join-Path $root 'installer\payload.zip'
if (Test-Path $payload) { Remove-Item $payload -Force }
Compress-Archive -Path (Join-Path $root 'build\bin\*') -DestinationPath $payload

Write-Host 'Building the setup program...'
Push-Location (Join-Path $root 'installer')
try {
    wails build -ldflags "-X main.appVersion=$version"
    if ($LASTEXITCODE -ne 0) { throw "wails build failed with exit code $LASTEXITCODE" }
} finally {
    Pop-Location
}

Write-Host 'Collecting the setup program...'
$distDir = Join-Path $root 'dist-installer'
New-Item -ItemType Directory -Force -Path $distDir | Out-Null
$built = Join-Path $root 'installer\build\bin\VisitronSetup.exe'
if (-not (Test-Path $built)) { throw "the setup build reported success but $built is not there" }
$setup = Join-Path $distDir 'VisitronSetup.exe'
Copy-Item $built $setup -Force

# Put the empty-zip placeholder back, so `go build` and the tests keep working
# without a full build and so a payload of megabytes never reaches a commit.
$empty = [byte[]](0x50, 0x4B, 0x05, 0x06) + (New-Object byte[] 18)
[System.IO.File]::WriteAllBytes($payload, $empty)

Write-Host ("Built {0} ({1:N0} bytes)" -f $setup, (Get-Item $setup).Length)
