# Visitron: Software Requirements Specification

Status: baselined 2026-10-09. Changes from here arrive as numbered amendments
with a reason (section 6). No question remains open.

## 1. Introduction

### 1.1 Purpose

Visitron tells its user how often each of his websites is visited and how often
each of his applications is downloaded. Both figures exist already, in two
places that never meet: GoatCounter counts page loads on the sites, while
GitHub counts downloads of each release file. Neither shows a trend for
downloads, neither knows which site belongs to which repository and neither
removes the one download of every macOS disk image that the owner makes himself
to confirm notarisation. Visitron gathers both, applies that correction and
keeps a daily history so the figures can be read over time.

### 1.2 Intended audience

The owner (Oliver Ernster), who is both the only user and the developer, plus
any AI assistant working on the code. The repository is public; the product is
built for one person and makes no attempt to serve anyone else.

### 1.3 Scope

In scope:

- A list of websites, each an address plus the GitHub repositories whose
  releases it offers.
- Adding a website by address, with a crawl of the site that proposes its
  repositories; editing an address, which repeats the crawl; deleting a
  website after a confirmation.
- First-run seeding from the record of every site that already carries the
  page counter (`sites.seed.json`).
- Page loads per website, read from GoatCounter.
- Downloads per website, repository, release and platform, read from GitHub,
  with the self-download correction.
- A daily history of both, shown as tables and charts.
- A check every 24 hours by default while Visitron runs in the tray, plus a
  manual refresh.
- Light and dark modes, dark by default, remembered across runs, in both the
  application and the setup program.
- A Settings dialog, a Guide, Help | About, a donate button and the house
  update check.
- A setup program in the house style.
- A GitHub Pages site at `ernster.dev/Visitron/` carrying the page counter.

Out of scope (decided; see also 3.11 Won't this time):

- Click events. Page loads only (decided 2026-10-09).
- Every operating system other than Windows (decided 2026-10-09).
- Any host for releases other than GitHub.
- Any analytics service other than GoatCounter; equally, a second GoatCounter
  account.
- Changing anything on GoatCounter or GitHub. Visitron only reads.
- Excluding the owner's own page loads (decided 2026-10-09: not wanted).
- Recovering download history from before Visitron's first check. GitHub
  publishes a running total only, so earlier days cannot be reconstructed.
- Private repositories.

### 1.4 Definitions

| Term | Meaning |
|---|---|
| Website | One record in Visitron: a site address plus a set of chosen repositories. |
| Site address | An `https` URL reduced to host plus path, lower-case host, no query, no fragment, ending in `/`. `https://ernster.dev/WhatDay/` and `https://symdiary.com/` are site addresses. The path keeps its case, because GitHub Pages paths are case sensitive. |
| Sub-site | A website whose address begins with another website's address: `ernster.dev/WhatDay/` is a sub-site of `ernster.dev/`. |
| Page load | One count recorded by GoatCounter when a page carrying the tag loads in a browser. GoatCounter stores it under the path `host + path`, for example `symdiary.com/download.html`. |
| Owned path | A GoatCounter path belongs to the website with the longest address that is a prefix of it. |
| Repository | A public GitHub repository named `owner/name`. |
| Chosen repository | A repository ticked for a website; only chosen repositories count towards its downloads. |
| Release file | One asset attached to a GitHub release. |
| Raw downloads | GitHub's `download_count` for a release file. |
| Self-download allowance | The downloads the owner makes himself: exactly 1 per `.dmg` release file, a named constant (decided 2026-10-09: not a setting). |
| Counted downloads | Raw downloads less the self-download allowance, never below 0. |
| Platform | Windows for a file ending `.exe`, macOS for `.dmg`, Linux for `.flatpak`, Other for anything else. Case is ignored. |
| Check | One pass over every website: page loads from GoatCounter, downloads from GitHub, a snapshot saved. |
| Snapshot | The counted downloads of every release file at the moment of a check. |
| Day | A calendar day in the time zone Windows is set to. |
| Check interval | The time between automatic checks; 24 hours unless changed in Settings. |
| Crawl | The fetch of a website's pages that finds the repositories it mentions. |
| Seed file | `sites.seed.json`, the record of every site carrying the counter, embedded in the binary. |
| Reference machine | Oliver's desktop, Windows 11. |

### 1.5 References

- `sites.seed.json` in this repository: the 32 sites measured 2026-10-09.
- GoatCounter JSON API, `https://www.goatcounter.com/api.json`, read
  2026-10-09.
- GitHub REST API, releases endpoints.
- `C:\Users\Oliver\Development\ed-voyage-companion\installer\`: the setup
  program to port.
- `C:\Users\Oliver\Development\SymDiary\frontend\src\GuideDialog.tsx` and
  `guideContent.ts`: the Guide to port. It follows the shape of ClearBudget's
  How It Works dialog (one scrolling page naming each control with its real
  picture, then the rules the window cannot state, with the house auto-scroll).
- The house skills `keeb`, `noborderfocus`, `scroll`, `installer`, `updates`
  and `donate`.
- Appendix A of this document: feasibility measurements taken 2026-10-09.

## 2. Overall description

### 2.1 Product perspective

A new, standalone Windows desktop application. It talks to two services over
HTTPS and nothing else besides its own update check: GoatCounter's API with the
owner's key and GitHub's API, anonymously or with an optional token. It also
fetches the pages of the websites it crawls. It keeps its data in one local
SQLite file.

### 2.2 User classes

One: the owner. No administrator rights are needed or requested at any point.

### 2.3 Operating environment

Windows 11 on x64, with WebView2 (present on every Windows 11 install). A
network connection is needed for a check; without one Visitron shows the last
figures it holds (FR-034).

### 2.4 Constraints

- C-1 Language: Go with Wails v2 and a React plus TypeScript front end; no C
  import.
- C-2 Layering: `internal/{domain,application,infrastructure,ui}` with a
  `main.go` composition root, enforced by a structural test.
- C-3 Storage: `modernc.org/sqlite` under `%LOCALAPPDATA%\Visitron`.
- C-4 Secrets: the GoatCounter key and the GitHub token live in Windows
  Credential Manager (`zalando/go-keyring`), never in a file or the log.
- C-5 The setup program, the Guide, the update check and the donate button are
  ported from the house references, not designed afresh.
- C-6 Licence: GPL-3.0 (already in the repository).
- C-7 `VERSION` at the repository root is the single source of the version.
- C-8 All artwork comes from `assets/`; no picture is drawn by Visitron.

### 2.5 Assumptions and dependencies

| ID | Assumption | Owner | Status |
|---|---|---|---|
| A-1 | Every site to be measured carries the GoatCounter tag that records `host + path`. | Oliver | Met 2026-10-09 for the 32 seeded sites (Appendix A, M-4). |
| A-2 | GoatCounter stays free for this volume ("reasonable public usage"). | Oliver | Accepted 2026-10-09. |
| A-3 | The owner downloads each `.dmg` exactly once per release to test notarisation; he never downloads other files. | Oliver | Stated 2026-10-09; consistent with Appendix A, M-2. |
| A-4 | Artwork is supplied as master PNGs with transparent backgrounds. | Oliver | Met: ten RGBA PNGs in `assets/` (Appendix A, M-1). |
| A-5 | A site names its repositories in links or scripts somewhere within its own pages. Where it does not, the owner adds them by hand (FR-009). | Oliver | Holds for symdiary.com (Appendix A, M-3); unmeasured elsewhere. |

## 3. Requirements

Priorities use MoSCoW. Every requirement names the test or the manual check
that verifies it; test names are the intended ones until the code exists.

### 3.1 Functional requirements: websites

**FR-001 Add website**
- Priority: Must
- Requirement: When the owner submits an address in the Add website dialog,
  the website service shall reduce it to a site address, crawl it (FR-005) and
  offer the repositories found for choosing (FR-007).
- Acceptance: Given the entry `SymDiary.com`, the dialog shows the site
  address `https://symdiary.com/` and offers `oernster/SymDiary`.
- Verified by: `internal/domain/address_test.go::TestNormalise`,
  `internal/application/websites_test.go::TestAddCrawlsAndOffers`

**FR-002 Address not usable**
- Priority: Must
- Requirement: If the entry cannot be read as an `http` or `https` address with
  a host, then the dialog shall refuse it, name what is wrong and keep the
  entry for correction.
- Acceptance: `ftp://x.com`, `symdiary` and an empty entry are each refused
  with a reason; nothing is recorded.
- Verified by: `address_test.go::TestRefusals`

**FR-003 Address already recorded**
- Priority: Must
- Requirement: If the site address is already a website, then the dialog shall
  refuse it and name the existing website.
- Verified by: `websites_test.go::TestDuplicateRefused`

**FR-004 Edit website**
- Priority: Must
- Requirement: When the owner changes a website's address in the Edit website
  dialog and submits it, the website service shall do what FR-001 does for the
  new address, with each previously chosen repository still ticked when the new
  crawl finds it.
- Verified by: `websites_test.go::TestEditRecrawlsKeepingTicks`

**FR-005 Crawl extent**
- Priority: Must
- Requirement: The crawler shall fetch the site address, then follow links in
  `href` and `src` attributes that stay on the same host and under the same
  path, up to 50 pages, reading at most 2 MiB of each response, with a 20 s
  limit per request.
- Rationale: symdiary.com names its repository in `site.js` and keeps its
  download buttons on `download.html` (Appendix A, M-3), so the home page alone
  is not enough. The caps keep a hostile or broken site from holding Visitron.
- Verified by: `internal/domain/links_test.go::TestSiteLinksStayOnSite`,
  `internal/domain/links_test.go::TestSiteLinksStayUnderPath`,
  `internal/application/websites_test.go::TestCrawlStopsAtPageLimit`, plus
  `internal/infrastructure/web/web_test.go::TestTooLargeIsRefused` and
  `internal/infrastructure/web/web_test.go::TestTimeout` against a local
  test server

**FR-006 Repository discovery**
- Priority: Must
- Requirement: The crawler shall report every repository named by a
  `github.com/<owner>/<name>` link or an `api.github.com/repos/<owner>/<name>`
  address in the fetched pages, once each, ignoring case.
- Acceptance: Given the pages of symdiary.com, the crawl reports exactly
  `oernster/SymDiary`, although the home page names it twice.
- Verified by: `internal/domain/repo_test.go::TestFindsBothForms`

**FR-007 Choosing repositories**
- Priority: Must
- Requirement: The Add and Edit dialogs shall list each repository found with a
  tick box, pre-ticked where it is the only one found or where its name, with
  punctuation removed, equals the site's last path segment or the first label
  of its host.
- Acceptance: `snarkapi.com` pre-ticks `oernster/snark-api`; `ernster.dev/WhatDay/`
  pre-ticks `oernster/WhatDay`; `ernster.dev/` pre-ticks nothing.
- Verified by: `repo_test.go::TestPreTick`

**FR-008 Crawl fails**
- Priority: Must
- Requirement: If the crawl cannot fetch the site address, then the dialog
  shall say why and still allow the website to be saved with repositories added
  by hand.
- Verified by: `websites_test.go::TestUnreachableSiteCanBeSaved`

**FR-009 Repository added by hand**
- Priority: Should
- Requirement: The Add and Edit dialogs shall accept a repository typed as
  `owner/name` and add it, ticked, once GitHub confirms it exists.
- Verified by: `websites_test.go::TestManualRepository`

**FR-010 Delete website**
- Priority: Must
- Requirement: When the owner presses Delete website and confirms in a dialog
  naming the selected website, the website service shall remove it with its
  history.
- Source: Q-2, answered 2026-10-09; artwork `assets/delete-website.png`.
- Verified by: `store_test.go::TestDeleteRemovesHistory` plus the house
  confirmation check.

**FR-011 First-run seeding**
- Priority: Must
- Requirement: When Visitron starts with no websites and has never seeded, the
  website service shall record every site in the seed file with the
  repositories it lists, then start a check.
- Acceptance: A fresh install shows 32 websites, ernster.dev with none chosen.
  Deleting all of them and restarting seeds nothing.
- Verified by: `websites_test.go::TestSeedsOnceOnly`

### 3.2 Functional requirements: the figures

**FR-020 Page loads per website**
- Priority: Must
- Requirement: For each website and each day, the figures service shall report
  the sum of GoatCounter page loads over the paths that website owns.
- Acceptance: Given paths `ernster.dev/index.html` (3 loads) and
  `ernster.dev/WhatDay/index.html` (5 loads) with both websites recorded,
  ernster.dev reports 3 and ernster.dev/WhatDay reports 5.
- Verified by: `internal/domain/figures_test.go::TestLongestPrefixOwns`

**FR-021 Counted downloads**
- Priority: Must
- Requirement: The figures service shall report each release file's counted
  downloads as its raw downloads less 1 when it ends `.dmg`, never below 0.
- Acceptance: `SymDiary.dmg` with 1 raw download counts 0; with 0 it counts 0;
  `SymDiarySetup.exe` with 4 counts 4.
- Verified by: `internal/domain/figures_test.go::TestSelfDownloadAllowance`

**FR-022 Download totals**
- Priority: Must
- Requirement: The figures service shall total counted downloads per website,
  per chosen repository, per release and per platform.
- Verified by: `internal/domain/figures_test.go::TestTotals`

**FR-023 Daily history**
- Priority: Must
- Requirement: The history service shall keep the last snapshot of each day
  and report a day's downloads as the rise in counted downloads since the
  previous kept snapshot.
- Acceptance: Snapshots totalling 10 on 1 October and 14 on 3 October report 4
  downloads on 3 October.
- Verified by: `internal/domain/figures_test.go::TestDailyRise`

**FR-024 A count that falls**
- Priority: Must
- Requirement: If a release file's counted downloads fall between snapshots
  (or the file disappears), then the history service shall record no rise for that
  file that day.
- Rationale: a deleted release or a re-uploaded file resets GitHub's count; a
  negative day is not a real event.
- Verified by: `internal/domain/figures_test.go::TestFallIsNotNegative`

### 3.3 Functional requirements: checking

**FR-030 Automatic check**
- Priority: Must
- Requirement: While Visitron runs, the scheduler shall start a check when the
  check interval has passed since the last successful check.
- Verified by: `internal/application/scheduler_test.go::TestDueAfterInterval`
  (fake clock)

**FR-031 Overdue at start or resume**
- Priority: Must
- Requirement: When Visitron starts or the machine resumes from sleep while a
  check is overdue, the scheduler shall start one within 1 minute.
- Verified by: `scheduler_test.go::TestOverdueOnStart`,
  `TestOverdueOnResume`

**FR-032 Manual refresh**
- Priority: Must
- Requirement: When the owner presses Refresh, the scheduler shall start a
  check at once, unless one is running.
- Verified by: `scheduler_test.go::TestRefreshStartsCheck`

**FR-033 One check at a time**
- Priority: Must
- Requirement: While a check runs, the window shall show its progress by
  website and the Refresh button shall be disabled.
- Verified by: `scheduler_test.go::TestNoOverlap` plus a manual check.

**FR-034 Service unreachable**
- Priority: Must
- Requirement: If GoatCounter or GitHub cannot be reached during a check, then
  the check shall keep the figures it already holds for the affected websites,
  state the failure with its time and count the check as unsuccessful.
- Verified by: `internal/application/check_test.go::TestFailureKeepsFigures`

**FR-035 GitHub rate limit**
- Priority: Must
- Requirement: If GitHub refuses a request for its rate limit, then the check
  shall stop asking GitHub, keep what it has, state the reset time GitHub gave
  and resume the remaining websites after it.
- Verified by: `check_test.go::TestRateLimitWaitsForReset`

**FR-036 No GoatCounter key**
- Priority: Must
- Requirement: While no GoatCounter key is set, the window shall show downloads
  as normal with a line saying a key is needed in Settings where the page
  loads would be.
- Verified by: `check_test.go::TestNoKeyDownloadsStillWork`

### 3.4 Functional requirements: the window

**FR-040 Toolbar**
- Priority: Must
- Requirement: The window shall show Add website, Edit website, Delete
  website, Refresh and Settings at the top left, each with its picture from
  `assets/`; at the top right, from left to right, it shall show Donate, a
  vertical separator, the theme button and the help button.
- Source: Q-1, answered 2026-10-09.
- Verified by: `App.test.tsx::orders the band as FR-040 states` plus manual inspection.

**FR-041 Website list**
- Priority: Must
- Requirement: The window shall list every website with its page loads and
  counted downloads for the chosen period, plus the change since the previous
  check.
- Verified by: `frontend` component test `WebsiteList.test.tsx`

**FR-042 Website detail**
- Priority: Must
- Requirement: When a website is selected, the window shall show its
  downloads by repository, release and platform as tables, plus charts of page
  loads per day and downloads per day over the chosen period.
- Verified by: `DetailPane.test.tsx`

**FR-043 Periods**
- Priority: Must
- Requirement: The window shall offer the periods 7 days, 30 days, 90 days and
  1 year, defaulting to 30 days, remembered across runs.
- Verified by: `internal/application/settings_test.go::TestPeriodRemembered`

**FR-044 Last check stated**
- Priority: Must
- Requirement: The window shall state when the last successful check finished,
  plus any failure from a later one (FR-034).
- Verified by: `WebsiteList.test.tsx`

### 3.5 Functional requirements: tray and lifecycle

**FR-050 Close to tray**
- Priority: Must
- Requirement: When the window is closed, Visitron shall keep running in the
  notification area with its application icon.
- Verified by: manual check.

**FR-051 Tray menu**
- Priority: Must
- Requirement: The tray menu shall offer Open, Refresh now and Quit.
- Verified by: manual check.

**FR-052 Start with Windows**
- Priority: Must
- Requirement: Where start with Windows is on, Visitron shall start at sign-in
  with its window hidden; it is on by default.
- Verified by: `internal/infrastructure/startup` test of the HKCU Run entry
  plus a manual sign-in.

**FR-053 Single instance**
- Priority: Must
- Requirement: When Visitron is started while already running, the running
  copy shall show its window and the new copy shall exit.
- Verified by: manual check.

### 3.6 Functional requirements: settings

**FR-060 Settings dialog**
- Priority: Must
- Requirement: The Settings button shall open a dialog holding the GoatCounter
  key, the GitHub token, the check interval, start with Windows and the update
  check.
- Verified by: `SettingsDialog.test.tsx`

**FR-061 Keys verified on save**
- Priority: Must
- Requirement: When a GoatCounter key or GitHub token is saved, the settings
  service shall try it once and report whether it works, storing it either way.
- Verified by: `internal/application/settings_test.go::TestKeyVerified`

**FR-062 Keys never shown**
- Priority: Must
- Requirement: The Settings dialog shall show a stored key or token only as
  "set", with a button to replace or remove it.
- Verified by: `SettingsDialog.test.tsx::shows each secret only as set or not`

**FR-063 Check interval range**
- Priority: Must
- Requirement: The check interval shall accept whole hours from 1 to 168,
  defaulting to 24.
- Verified by: `settings_test.go::TestIntervalRange`

### 3.7 Functional requirements: appearance and help

**FR-070 Theme**
- Priority: Must
- Requirement: The theme button shall switch between dark and light, showing
  the picture of the mode it would switch to, as SymDiary does; the choice is
  remembered and dark is the default.
- Verified by: `theme.test.ts::opens dark when nothing has been remembered`,
  `useTheme.test.tsx::opens in what was remembered`

**FR-071 Guide**
- Priority: Must
- Requirement: The help button shall open the Guide, ported from SymDiary: one
  scrolling page naming each control with its real picture, then the rules
  behind the figures (the self-download allowance, owned paths, history only
  from the first check), self-reading per the house auto-scroll.
- Verified by: `GuideDialog.test.tsx` plus a manual read.

**FR-072 Help | About**
- Priority: Must
- Requirement: The Guide shall lead to About, which shows the application icon,
  the name Visitron, "by Oliver Ernster", "© Oliver Ernster", the version from
  `VERSION`, the GPL-3.0 licence and a credit to every open source component
  shipped, each with its licence.
- Verified by: `Dialog.test.tsx` plus a structural test that every module
  in `go.mod` and every runtime package in `package.json` is credited.

**FR-073 Donate**
- Priority: Must
- Requirement: The donate button shall open
  `https://www.paypal.com/ncp/payment/NRXS4SP24A6C8` in the default browser.
- Verified by: `app_test.go::TestDonateOpensTheOneAddressOrSaysWhyNot`

**FR-074 Keyboard**
- Priority: Must
- Requirement: Every control in the window and in every dialog shall be
  reachable and usable from the keyboard per the house `keeb` model, with the
  focus ring on controls only (`noborderfocus`).
- Verified by: `frontend` keyboard ring tests plus a manual pass.

**FR-075 Update check**
- Priority: Should
- Requirement: Where the update check is on, Visitron shall check its own
  GitHub releases per the house `updates` model and offer a newer version.
- Verified by: ported tests from the reference implementation.

### 3.8 Functional requirements: setup program

**FR-080 Setup program**
- Priority: Must
- Requirement: The setup program shall install, update, repair and uninstall
  Visitron per user without administrator rights, ported from ED Voyage
  Companion per the house `installer` model.
- Verified by: ported tests plus a build-and-install on the reference machine.

**FR-081 Setup theme**
- Priority: Must
- Requirement: The setup program shall open in dark mode, offer the same theme
  button and remember the choice across runs.
- Verified by: manual check across two runs.

### 3.9 Functional requirements: the site

**FR-090 Project site**
- Priority: Must
- Requirement: Visitron shall have a GitHub Pages site at
  `ernster.dev/Visitron/`, every page carrying the GoatCounter tag, recorded in
  the seed file.
- Verified by: the seed measurement script (Appendix A, M-4) run against the
  live site.

### 3.10 Non-functional requirements

**NFR-PERF-001 Check duration**: With the 32 seeded websites and a GitHub
token, a full check shall finish within 2 minutes on the reference machine's
connection. Measured by timing a check in the log.

**NFR-PERF-002 Start**: The window shall be usable within 3 s of being opened
from the tray. Measured by hand.

**NFR-REL-001 Interrupted check**: A check that is interrupted (quit, crash,
power loss) shall leave the figures of the last successful check intact; a
snapshot is written in one transaction. Verified by
`internal/infrastructure/store/store_test.go::TestSnapshotIsAtomic`.

**NFR-REL-002 No silent death**: Every failure before the window opens shall be
shown in the window; a crash shall leave a record in the log (house
robustness rules 1 and 2). Verified by manual fault injection.

**NFR-SEC-001 Secrets**: The GoatCounter key and GitHub token shall never be
written to disk by Visitron, nor to the log, nor sent to any host other than
their own service. Verified by a structural test scanning log calls plus
`TestKeyGoesOnlyToItsHost`.

**NFR-PRIV-001 Network**: Visitron shall contact only GoatCounter, GitHub, the
websites it crawls and the donate link when pressed. It sends no telemetry.
Verified by a structural test listing every outbound host.

**NFR-MAINT-001 Coverage**: Domain and application layers at 100% line
coverage, gated in `test.ps1`. Verified by the gate.

**NFR-MAINT-002 Structure**: Layering, the 400-line module limit with its
danger band, one home for every colour and the wire stated on both sides,
each enforced by a structural test proved by planting a violation.

**NFR-OBS-001 Log**: Each check shall log its start, each website's outcome
and its end with duration, to `%LOCALAPPDATA%\Visitron\Log.txt`.

### 3.11 Won't this time

- Click tracking or any event other than a page load.
- macOS and Linux builds.
- Per-visitor detail (referrers, browsers, countries), though GoatCounter
  holds it.
- A self-download allowance setting (fixed at 1 per `.dmg`).
- Notifications when figures change.
- Export of figures.

## 4. Other requirements

### 4.1 Legal

GPL-3.0 for Visitron. Each third-party component keeps its own licence and is
credited in About (FR-072).

### 4.2 Internationalisation

English only.

### 4.3 Risk

A personal utility with no safety, financial or medical consequence; FMEA is
disproportionate and not done. The one material risk is a leaked key, covered
by NFR-SEC-001.

## 5. Appendices

### Appendix A: Feasibility measurements (2026-10-09)

- M-1 `assets/` holds ten RGBA PNGs with transparent corners:
  application-icon, add-website, edit-website, delete-website, refresh,
  settings, help-guide, light-mode, dark-mode (each 1254x1254) and donate
  (1312x1199).
- M-2 Every `SymDiary.dmg` release file shows exactly 1 download while the
  `.exe` and `.flatpak` files show 0: the self-download.
- M-3 symdiary.com names `github.com/oernster/SymDiary` on its home page and
  `api.github.com/repos/oernster/SymDiary` in `site.js`; its download buttons
  are on `download.html`.
- M-4 All 32 seeded sites answered 200 and served the tag; 9 have no releases
  (crankthecode, snark-api, CommandFixer, FuckWhatDay, locus, MMSP-Spec,
  elevator, coin-analysis, snark3Dprinter-discord-bot).
- M-5 Across every seeded repository the release files are `.exe` (384),
  `.flatpak` (257) and `.dmg` (177), with no pre-releases. Not every release
  has a `.dmg` (ClearBudget: 46 releases, 26 `.dmg` files).
- M-6 GoatCounter's `/api/v0/stats/hits` accepts `start`, `end`, `daily` and
  path filters; `/api/v0/paths` lists every path. Keys are created per user
  with a "Read statistics" permission and a choice of sites.
- M-7 GitHub Pages publishes no visitor figures for a site; the repository
  traffic API covers the repository page only and the last 14 days.

### Appendix B: Open questions

None open. Answered on 2026-10-09:

| ID | Question | Answer |
|---|---|---|
| Q-1 | Top-right order. | Donate, a vertical separator, theme, help (FR-040). |
| Q-2 | Keep Delete website? | Yes, with its own artwork (FR-010). |
| Q-3 | Periods 7, 30, 90 days and 1 year, default 30? | Yes (FR-043). |

### Appendix C: Build order

1. Domain: address reduction, discovery, pre-tick rule, owned paths, counted
   downloads, totals, daily rise.
2. Application: website service, figures, history, scheduler, settings, each
   drivable headlessly from a test.
3. Infrastructure: SQLite store, GoatCounter and GitHub clients, crawler,
   Credential Manager, HKCU startup, log.
4. UI: window, dialogs, tray, then the Guide and About.
5. Setup program, update check, site.

## 6. Amendments

**Amendment 1 (2026-10-09): retry after a failed check.** FR-030 measures the
interval from the last successful check, so after a failure every one-minute
tick would find a check due and an outage would mean a check a minute. The
scheduler now waits 30 minutes after a failed check before trying again on its
own (`FailureRetry`); Refresh still runs at once. "Change since the previous
check" (FR-041) is the rise between the last two kept snapshots, since a later
check on the same day replaces that day's snapshot. Verified by
`scheduler_test.go::TestFailureRetriesAfterWait` and
`figures_test.go::TestOverviewSeparatesSubSites`.

**Amendment 2 (2026-10-09): what GoatCounter's figure is.** GoatCounter's
statistics API reports visitors, not raw page loads: its `count` is "number of
visitors" and each day's `daily` is "total visitors for this day" (read from
`api.json` on 2026-10-09). Raw loads are only in its export, which needs a key
with the Export permission. Visitron therefore shows each page's visitors per
day under the name page loads; one person reading a page twice in a day counts
once. The Guide states this (FR-071). Verified by
`internal/infrastructure/goatcounter/client_test.go::TestDailyPagesThroughPaths`.

**Amendment 3 (2026-10-09): a failed check is shown on Refresh; the
references are held to the tree.** Ruled by the owner on 2026-10-09.

- While the last check has failed (FR-044: a failure later than the last
  success), the Refresh button shall carry `assets/warning.png` over the corner
  of its picture, with the failure as its hint, until a later check succeeds.
  The Guide says so (FR-071). Verified by
  `App.test.tsx::puts the warning on Refresh only while the last check has failed`.
- A check that stops on a fault it could not report itself (a panic or a
  website list that could not be read) shall be recorded as a failed check, so
  the warning above covers it too (house robustness rule 8). Verified by
  `scheduler_test.go::TestACheckThatCannotStartIsRecordedAsFailed`,
  `scheduler_test.go::TestRecordFaultWarnsUntilALaterSuccess` and
  `app_test.go::TestAPanicInACheckIsRecordedAsAFailedCheck`.
- In the website list with nothing selected, Down shall select the first row
  and Up the last (FR-041). Verified by
  `WebsiteList.test.tsx::starts at the last row when Up is pressed with nothing selected`.
- NFR-SEC-001 against GitHub's own answers: a next page of releases named on
  another host shall be refused rather than sent the token; no more than
  one hundred pages are read for one repository. Measured before the fix: the
  token reached the other host. Verified by `TestKeyGoesOnlyToItsHost` and
  `TestPagesAreCapped`.
- Every text pairing in both themes is held to WCAG 2.2 AA contrast and every
  focus ring to 3:1, as in SymDiary. Verified by
  `tests/structural/colours_test.go::TestEveryTextPairingMeetsAA`.
- Fifteen "Verified by" references named files or tests that did not exist,
  mostly because the tests landed in differently named files; each now names
  the test that proves its requirement. The four that had none (the crawl's
  time limit, the band order, the Guide, About) were written. A structural test
  now fails on any reference that names nothing. Verified by
  `tests/structural/traceability_test.go::TestEveryVerificationNamesARealTest`.

**Amendment 4 (2026-10-09): closing asks.** Ruled by the owner on 2026-10-09,
ported from PigeonPost. FR-050 now reads: when the window's close button is
pressed and the tray icon is up, Visitron shall bring the window forward and
ask whether to minimise to the tray or quit; Escape cancels and leaves the
window open. The choice opens on Minimise to tray; where another dialog is
open, it warns that unsaved work may be lost and opens on Go back. Without a
tray icon the close quits, so no one is left with a process they cannot
reach; a Quit from the tray or the choice does not ask again. Verified by
`closing_test.go::TestCloseAsksWhenTheTrayIsUp`,
`closing_test.go::TestCloseQuitsWithoutATray`,
`closing_test.go::TestMinimiseHidesAndQuitDoesNotAskAgain`,
`closing_test.go::TestTheTrayRevealsAndQuitsThroughTheSameWindow` and
`CloseChoiceDialog.test.tsx::warns of open work and opens on Go back`.

**Amendment 5 (2026-10-09): the update check.** FR-075 is built per the house
`updates` model in the form Bridge Talk takes, where the page holds no
address: the facade keeps the release it offered and Download and Skip act on
it. The automatic check runs 3 s after the window opens, then every 24 hours;
it asks nothing while Settings has it off; it shows only an offer and never
offers the version the owner skipped. Check for updates on About runs the
check asked for, ignoring the skip and the switch; it reports every outcome.
The request goes unauthenticated through the shared web client to
`releases/latest` of Visitron's own repository with a 5 s limit; a page or a
file in the answer that is not under `https://github.com/` is refused rather
than handed to the browser. The skipped version is kept with the other
preferences. Verified by
`internal/domain/version_test.go::TestNewer`,
`internal/application/update_test.go::TestASkippedReleaseIsNotOfferedUnasked`,
`internal/application/update_test.go::TestTheSwitchTurnsTheAutomaticCheckOff`,
`internal/infrastructure/update/github_test.go::TestLatestAsksTheRightPlaceUnauthenticated`,
`updates_test.go::TestAnOfferIsKeptForDownloadAndSkip` and
`updates.test.tsx::checks 3 seconds after the page loads, then once a day`.

**Amendment 6 (2026-10-10): no release yet is its own answer.** Measured on
2026-10-10: GitHub answers 404 for `releases/latest` of a repository with no
published release, which Amendment 5 read as unreachable. A 404 now reads as
no release published; a check asked for says "No release of Visitron has been
published yet." and an automatic one says nothing. Verified by
`internal/infrastructure/update/github_test.go::TestNoPublishedReleaseIsNamed` and
`internal/application/update_test.go::TestNoPublishedReleaseIsItsOwnAnswer`.

**Amendment 7 (2026-10-10): the setup program; saying what the window
needs.** Ruled by the owner on 2026-10-10 after the first hands-on run.

- FR-080: the setup program is ported from SymDiary's. The data it keeps on an
  uninstall unless asked is the folder the store uses, %LOCALAPPDATA%\Visitron,
  which also holds the run log: setup asks the store for it, so the two cannot
  name different folders. The log therefore goes only with the data, never as
  a leftover. Verified by
  `internal/infrastructure/setup/leftovers_test.go::TestTheRecordIsNeverClearedWithTheLeftovers`.
  Every setup test runs inside a scratch profile, since one that redirected
  APPDATA alone reached the real data folder on 2026-10-10.
- FR-060: Settings shows, under each field, what the key or token is for and
  numbered steps to get it, read from GoatCounter's API help and GitHub's token
  and rate-limit pages on 2026-10-10. Verified by
  `SettingsDialog.test.tsx::says where the key and the token come from`.
- FR-041: the three download columns sit under one Downloads heading, the first
  naming the period it covers. Verified by
  `WebsiteList.test.tsx::shows each website without its scheme and its counts`.
- FR-042: a chart with nothing to draw says what it is waiting for: a
  GoatCounter key for page loads; two checks on different days for downloads.
  Verified by `Chart.test.tsx::draws nothing for no days and a flat line for days of nothing`.

