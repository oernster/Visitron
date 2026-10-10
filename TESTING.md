# Visitron: testing

Every command is PowerShell, run from the repository root.

## Before the first run

- `npm install --prefix frontend` once; `test.ps1` does not install.
- staticcheck is fetched through `go run ...@latest`, so the first run needs
  the network.
- On Windows, allow Go's scratch directory (`go env GOTMPDIR`) in the
  anti-virus. A quarantined test binary stops the suite rather than failing a
  test.

## The gate

```powershell
./test.ps1
```

In order, stopping at the first failure:

1. `gofmt -l` over the Go, ignoring the front end.
2. `go vet` over every package outside `node_modules`.
3. staticcheck over the same packages.
4. The whole Go suite.
5. Coverage of `internal/domain` and `internal/application` together, against
   100%. A shortfall names each uncovered function.
6. Every other measured package against its floor.
7. The front-end build: `eslint`, `tsc --noEmit`, `vite build`.
8. The front-end suite: Vitest over jsdom.
9. `node --check` over every script in `installer/frontend/dist`, since the
   setup page has no build step to catch a typo.

`build.ps1` runs it first. BuildPilot reads its exit code.

## Reading a result

Read the exit code, never the text:

```powershell
./test.ps1; "EXIT=$LASTEXITCODE"
```

`0` means every step passed and every floor held. Anything else: read the
error the script threw.

## Coverage floors

A floor is the measured number, never a target, so it fails the moment cover
is lost.

| Package | Floor | Why not 100 |
|---|---|---|
| `internal/domain` + `internal/application` | 100 | |
| `internal/infrastructure/github` | 100 | |
| `internal/infrastructure/goatcounter` | 100 | |
| `internal/infrastructure/secrets` | 100 | |
| `internal/infrastructure/web` | 100 | |
| `internal/infrastructure/update` | 100 | |
| `internal/infrastructure/store` | 88 | Operating-system failures that cannot be forced without breaking the disk. |
| `internal/infrastructure/startup` | 85 | Registry failures that cannot be forced. |
| `internal/infrastructure/runlog` | 81 | Win32 standard-handle work. |
| root package (facade) | 72 | `main`, the window's runtime calls and the single-instance lock need a window. |
| `./installer` | 69 | The Wails runtime beneath the setup facade. |
| `internal/infrastructure/setup` | 63 | The registry and shortcut writes need a real profile. |
| `internal/infrastructure/windowfocus` | 27 | Win32 calls against a real window; the portable half alone. |
| `internal/infrastructure/tray` | 4 | Win32 calls against a real desktop; the portable half alone. |

Not gated: `internal/product` (constants), `internal/licence` (compared with
`LICENSE` by `tests/structural`), `internal/infrastructure/setup/setuptest`
(a test double) and `tests/structural` (the guard itself).

## What each suite proves

| Suite | What it settles |
|---|---|
| `internal/domain` | Address reduction, repository discovery, path ownership, counted downloads, days, setting bounds, version comparison, the order of countries and releases. |
| `internal/application` | Every use case over hand-written fakes: the crawl, a check, the scheduler's retry and fault recording, figures over the history, visitors by country, settings, the update check. |
| Infrastructure | Each adapter against something real: a local HTTP server for the web, GitHub, GoatCounter and update clients; SQLite in a temporary folder; an in-memory keyring; a Run value of the test's own. |
| root package | The facade end to end over a real store: conversions, the statistics, refusals, the close choice, the update offer, a panic becoming an error. |
| `./installer`, `setup` | The route setup opens on, install, repair and removal against the `setuptest` machine; a payload entry climbing out of the install folder refused; the data folder never cleared unasked. |
| `tests/structural` | The invariants in [ARCHITECTURE.md](ARCHITECTURE.md). |
| `frontend` | The page over a fake facade: every dialog, the list, the charts, the Help menu, the ring and the self-reading cycle. |

## What the tests never do

- Reach the network. The one live test, `TestLiveRepository` in
  `internal/infrastructure/github`, skips unless `VISITRON_LIVE_REPO` names a
  repository; the gate never sets it.
- Touch the real data or profile. Store tests use `t.TempDir()`; every setup
  test runs inside a scratch profile (`internal/infrastructure/setup/main_test.go`).
- Touch the real secrets: the keyring is the library's in-memory mock.
- Mock SQLite. The store is tested against the real engine.
- Assert against Wails' generated bindings. The wire contract is
  `frontend/src/api.ts`, compared with `dto.go`.

One exception: the startup test writes the HKCU Run key under a value name of
its own, then removes it.

## Running part of the suite

```powershell
./test.ps1 -SkipFrontend
./test.ps1 -Floor 95
go test ./internal/domain/...
go test ./tests/structural/...
npm --prefix frontend test
```

`-SkipFrontend` runs the Go half alone. `-Floor` changes the gated layers'
floor for a deliberate check, never for a release.

## Further reading

- [ARCHITECTURE.md](ARCHITECTURE.md): the invariants the structural tests enforce.
- [DEVELOPMENT.md](DEVELOPMENT.md): tools, building and releasing.
