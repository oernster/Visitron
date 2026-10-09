# Verifies Visitron: formatting, vet, staticcheck, the Go suite and its coverage
# floors, then the front end's lint, types and suite. Ported from SymDiary.
#
#   ./test.ps1                 run everything
#   ./test.ps1 -SkipFrontend   the Go half alone, while working on it
#   ./test.ps1 -Floor 95       a different floor for the gated layers, for a deliberate check
#
# build.ps1 runs this before it builds. It asks nothing and answers with its
# exit code, 0 only when everything passed, which is what BuildPilot reads.
#
# Floors are measured numbers, never targets. A floor picked from an aspiration
# only teaches people to lower it; a floor at the measured number fails the
# moment cover is lost, which is the only moment it is worth being told.
param(
    [double]$Floor = 100,
    [switch]$SkipFrontend
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

# The gate covers the layers that can be exercised with no filesystem, no clock
# and no window: the rules and the use cases (NFR-MAINT-001). Anything short
# there is a decision nobody made. The rest of the tree is held at what it
# actually reaches, below.
$gated = './internal/domain/...', './internal/application/...'

# Visitron's own packages. A bare ./... also walks a Go package that an npm
# dependency ships inside frontend/node_modules.
$packages = go list ./... | Where-Object { $_ -notmatch '/node_modules/' }
if ($LASTEXITCODE -ne 0) { throw "go list failed with exit code $LASTEXITCODE" }

Write-Host 'Checking formatting...'
$unformatted = gofmt -l . | Where-Object { $_ -notmatch '^frontend' }
if ($unformatted) { throw "gofmt reports unformatted files:`n$($unformatted -join "`n")" }

Write-Host 'Vetting...'
go vet $packages
if ($LASTEXITCODE -ne 0) { throw "go vet failed with exit code $LASTEXITCODE" }

Write-Host 'Running staticcheck...'
go run honnef.co/go/tools/cmd/staticcheck@latest $packages
if ($LASTEXITCODE -ne 0) { throw "staticcheck failed with exit code $LASTEXITCODE" }

Write-Host 'Running the whole Go suite...'
go test $packages
if ($LASTEXITCODE -ne 0) { throw "go test failed with exit code $LASTEXITCODE" }

Write-Host "Measuring coverage of $($gated -join ', ')..."
$profilePath = Join-Path ([System.IO.Path]::GetTempPath()) 'visitron-coverage.out'
try {
    go test "-coverprofile=$profilePath" @gated
    if ($LASTEXITCODE -ne 0) { throw "the coverage run failed with exit code $LASTEXITCODE" }

    $summary = go tool cover "-func=$profilePath"
    if ($LASTEXITCODE -ne 0) { throw "go tool cover failed with exit code $LASTEXITCODE" }

    # The last line is the total across the merged profile. Read the exit code
    # and this line, never the run's own output.
    $total = ($summary | Select-Object -Last 1)
    if ($total -notmatch '([0-9]+(?:\.[0-9]+)?)%\s*$') { throw "could not read a total from: $total" }
    $percent = [double]$Matches[1]

    if ($percent -lt $Floor) {
        Write-Host 'Not covered:'
        $summary | Where-Object { $_ -notmatch '100\.0%\s*$' -and $_ -notmatch '^total:' } |
            ForEach-Object { Write-Host "  $_" }
        throw "coverage is $percent%, below the floor of $Floor%"
    }
    Write-Host "Coverage $percent%, floor $Floor%."
} finally {
    if (Test-Path $profilePath) { Remove-Item $profilePath -Force }
}

# The rest of the tree, each package at the number it actually reaches,
# measured on 2026-10-09.
#
# The four adapters at 100 talk to something a test can stand up for real: a
# local HTTP server for the web client, GitHub and GoatCounter, an in-memory
# keyring for the secrets. The root package holds the facade, which is tested,
# plus main, the window's runtime calls and the single-instance lock, which need
# a window. The store and the startup entry reach everything but operating-
# system failures that cannot be forced without breaking the disk or the
# registry. The run log's remainder is the Win32 handle work. The tray and the
# keyboard hand-over are almost wholly Win32 calls against a real desktop, so
# their figures are the portable halves alone; they are listed so that the
# little that can be tested stays tested.
$measured = [ordered]@{
    '.'                                       = 69
    './internal/infrastructure/github'        = 100
    './internal/infrastructure/goatcounter'   = 100
    './internal/infrastructure/secrets'       = 100
    './internal/infrastructure/web'           = 100
    './internal/infrastructure/store'         = 88
    './internal/infrastructure/startup'       = 85
    './internal/infrastructure/runlog'        = 81
    './internal/infrastructure/windowfocus'   = 27
    './internal/infrastructure/tray'          = 5
}

Write-Host 'Measuring the rest of the tree...'
foreach ($package in $measured.Keys) {
    $floor = $measured[$package]
    $reported = go test -cover $package
    if ($LASTEXITCODE -ne 0) { throw "$package failed with exit code $LASTEXITCODE" }

    $line = $reported | Where-Object { $_ -match 'coverage: ' } | Select-Object -First 1
    if ($line -notmatch 'coverage: ([0-9]+(?:\.[0-9]+)?)%') {
        throw "could not read a coverage figure for ${package}: $line"
    }
    $reached = [double]$Matches[1]
    if ($reached -lt $floor) {
        throw "$package is at $reached%, below its floor of $floor%"
    }
    Write-Host ("  {0,-42} {1,5}%  floor {2}%" -f $package, $reached, $floor)
}

# Not gated at all, deliberately: internal/product holds constants,
# internal/licence is one embedded file that tests/structural compares byte for
# byte with LICENSE; tests/structural is itself the guard. A floor over any
# of them asserts nothing.

if ($SkipFrontend) {
    Write-Host 'All green (the front end was skipped).'
    return
}

Write-Host 'Checking the front end: lint, types, build...'
npm --prefix frontend run build
if ($LASTEXITCODE -ne 0) { throw "the front-end build failed with exit code $LASTEXITCODE" }

Write-Host 'Running the front-end suite...'
npm --prefix frontend test
if ($LASTEXITCODE -ne 0) { throw "the front-end suite failed with exit code $LASTEXITCODE" }

Write-Host 'All green.'
