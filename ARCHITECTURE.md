# Visitron: architecture

Visitron reads page loads from GoatCounter and release downloads from GitHub,
keeps a daily history in a local SQLite file and shows both from the Windows
tray. It only reads; it never changes anything on either service.

## Invariants

Each is enforced by a test in `tests/structural`. Every one was proved to bite
by planting a violation and reading the exit code.

| # | Invariant | Enforced by |
|---|---|---|
| 1 | The domain imports no other layer. | `boundary_test.go::TestDomainHasNoOutwardImports` |
| 2 | The domain performs no IO and never reads the wall clock or the global random source. | `boundary_test.go::TestDomainIsPure` |
| 3 | The application layer imports neither infrastructure nor Wails. | `boundary_test.go::TestApplicationDoesNotImportInfrastructure` |
| 4 | Only `main.go` wires infrastructure to the application. | `boundary_test.go::TestCompositionRootIsWhitelisted` |
| 5 | No Go, page or setup-page file exceeds 400 lines; none sits between 381 and 400. | `boundary_test.go::TestNoFileExceedsLineLimit`, `TestNoFileInDangerBand` |
| 6 | Every exported type carries a doc comment. | `boundary_test.go::TestEveryExportedTypeIsDocumented` |
| 7 | The page opens no connection of its own (`connect-src 'none'`). Only `web`, `github` and `update` import `net/http`; no raw socket; every web address in the Go source is listed with the requirement that allows it. | `boundary_test.go::TestThePageOpensNoConnection`, `network_test.go` |
| 8 | Only declared places write to the run log, so the key and the token cannot reach it unnoticed (NFR-SEC-001). | `logscan_test.go::TestOnlyTheKnownPlacesWriteToTheLog` |
| 9 | Nothing shipped names the person building it; the identity to refuse is read from the origin remote. | `owner_test.go::TestNothingShippedNamesTheBuilder` |
| 10 | The wire is stated twice, in `dto.go` and `frontend/src/api.ts`; the two agree field for field. | `wire_test.go::TestTheWireContractMatchesOnBothSides` |
| 11 | Every colour lives in `frontend/src/theme.css`; every text pairing meets WCAG AA and every ring 3:1, in both modes. | `colours_test.go` |
| 12 | The focus ring belongs to controls: no container takes focus or a ring; a scrolling surface suppresses the native ring; a ringed control says when it is disabled; a scrolling dialog pins its actions and reads itself. | `focus_test.go` |
| 13 | Every text box applies on Enter. | `enter_test.go::TestEveryTextBoxAppliesOnEnter` |
| 14 | Every runtime dependency is credited in About; no credit names something unshipped. | `credits_test.go::TestEveryDependencyIsCredited` |
| 15 | The licence the setup program embeds is the root `LICENSE`, byte for byte. | `licence_test.go::TestTheEmbeddedLicenceIsThePublishedOne` |
| 16 | The setup page looks up only elements its markup holds and reads only fields Go sends. | `setuppage_test.go` |
| 17 | Every "Verified by" in REQUIREMENTS.md names a test that exists. | `traceability_test.go::TestEveryVerificationNamesARealTest` |

## Dependency direction

```text
UI (root package, frontend/)  ->  Application  ->  Domain  <-  Infrastructure
```

## Components

**Domain, `internal/domain`.** Pure rules: site addresses (`address.go`),
repositories found in a page (`links.go`, `repo.go`), which website owns a
GoatCounter path (`owner.go`), platforms and counted downloads
(`downloads.go`), days (`history.go`), setting bounds (`settings.go`), the
GoatCounter account name (`goatcounter.go`), the order countries are listed
in (`countries.go`) and version comparison (`version.go`).

**Application, `internal/application`.** One service per thing the owner does,
over the ports in `ports.go`: `websites.go` (add, edit, delete, crawl),
`check.go` (one pass over every website), `scheduler.go` (when a check is due,
the wait after a failure), `figures.go`, `countries.go` (one website's visitors
by country; also the one rule for whether GoatCounter is set up), `settings.go`,
`update.go`, `about.go`.

**Infrastructure, `internal/infrastructure`.**

| Package | What it implements |
|---|---|
| `web` | The one HTTP client, plus the crawl's page reader. |
| `github` | Release download counts, anonymous or with the owner's token. |
| `goatcounter` | Daily page loads, page paths and visitors by country from the owner's GoatCounter site. |
| `update` | Visitron's latest published release, unauthenticated. |
| `store` | SQLite at `%LOCALAPPDATA%\Visitron\visitron.db`; `Unavailable` stands in when it cannot open. |
| `secrets` | The key and the token in Windows Credential Manager. |
| `startup` | The start-with-Windows value under the HKCU Run key; Settings and the setup program both write it through here. |
| `tray` | The Win32 notification-area icon: Open, Refresh now, Quit. |
| `runlog` | The run log; points error output at it before anything can fail. |
| `windowfocus` | Hands the keyboard to the WebView2 child window on DOM ready. |
| `setup` | The per-user install policy the setup program runs. |

`internal/product` holds the name and file names; `internal/licence` the
embedded licence and its plain reading.

**UI, the root package and `frontend/`.** `app.go` and `actions.go` are the
facade: one bound method per action, converting shapes and calling one service.
`dto.go` holds the wire shapes; `closing.go` the close choice; `updates.go` the
update offer; `window.go` the Wails runtime seams. The React page in
`frontend/src` talks to the facade through `api.ts` alone.

**Setup program, `installer/`.** A second Wails application in the same module
that embeds the built application as `payload.zip`. `installer/app.go` is a
facade over `internal/infrastructure/setup`, holding a `setup.Machine` so its
sequences can be tested against a recorder. Its page under
`installer/frontend/dist` is hand-written with no build step.

## Execution flow

1. `main` opens the run log and points error output at it.
2. It opens the store, else uses `store.Unavailable` and carries the failure for
   the window to show.
3. It builds the adapters and the services, then starts the tray icon. Without
   one, closing the window quits.
4. `wails.Run` starts the window with the single-instance lock; the Run-key
   flag starts it hidden in the tray.
5. On startup the scheduler asks once a minute whether a check is due; the tray
   is followed on its own goroutine.
6. On DOM ready the facade hands the page the keyboard.
7. Each page action is one bound method; a panic becomes an error the page
   shows, with the stack in the log.
8. On shutdown the scheduler stops and the store closes.

## Key decisions

- **Secrets stay in Credential Manager** and cross into the window only when
  Settings asks for them (Amendment 8).
- **The store keeps GitHub's raw counts**; the owner's own disk-image downloads
  are subtracted when read, so changing the setting applies to all history
  (Amendment 19).
- **The release repository comes from the build**, read off the origin remote
  and passed through `-ldflags -X`, so no account is written in the source and
  a fork checks its own releases (Amendment 15).
- **A failed check waits 30 minutes** before retrying on its own, so an outage
  is not a check a minute; Refresh still runs at once (Amendment 1).
- **A panic in a check is recorded as a failed check**, so the warning on
  Refresh covers it (Amendment 3).
- **The close button asks** whether to minimise to the tray or quit, only while
  the tray icon is up (Amendment 4).
- **Windows only.** Win32 work sits behind build tags with a stub beside it,
  so each package's portable half stays testable.

## Further reading

- [TESTING.md](TESTING.md): the gate, the floors and what each suite proves.
- [DEVELOPMENT.md](DEVELOPMENT.md): tools, building and releasing.
