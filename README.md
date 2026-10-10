# <img width="128" height="128" alt="application-icon" src="https://github.com/user-attachments/assets/5c0abc13-03ba-4e8d-950c-ae3e4f0299d1" /> Visitron

Page loads and downloads for every website you run, side by side, with a daily history.

> **Commercial licences available.** Visitron is free and open source under the
> GPL-3.0. If those terms do not suit what you are building, such as a
> closed-source product, a commercial licence can be bought from me separately.
> It covers my own code; third-party libraries keep their own licences. See
> [commercial licensing](https://ernster.dev/commercial-licensing.html).

GoatCounter counts the visits to your sites and GitHub counts the downloads of
your releases. Neither knows which repositories belong to which site; neither
shows downloads over time. Visitron joins the two: each website you add
gets its page loads and the downloads of the releases it offers, by repository,
release and platform, with charts of every day since the first check.

## Who it is for

Anyone who publishes websites and GitHub releases and wants one window showing
how both are doing. It runs on 64-bit Windows 11.

Not for anyone wanting visitor profiles, referrers or real-time analytics:
Visitron reports counts, not people.

## What it does not do

- It does not count visits itself. Page loads come from your own GoatCounter
  account, which needs its tag on your pages and an API key in Settings.
- It does not run on macOS or Linux.
- It sends nothing about you anywhere. It makes four kinds of request: to
  GoatCounter for page loads and visitors by country, to GitHub for release
  downloads, to the website
  you add (reading its pages to find its repositories) and to GitHub once a day
  to check for a newer Visitron, which Settings can turn off.
- It cannot see downloads from before its first check: GitHub keeps only
  running totals, so the daily history starts there.

## What it does

- Lists your websites, each an address plus the GitHub repositories it offers.
  Adding one reads the site and proposes its repositories; you tick which count.
- Gives a sub-site its own figures: `example.com/app` is counted apart from
  `example.com`.
- Shows page loads and downloads over 7, 30, 90 or 365 days, plus downloads to
  date.
- Opens each website's statistics from the button at the right of its row:
  downloads to date by repository, release and platform, then visitors by
  country over the period, as GoatCounter counts them. Downloads have no
  country; GitHub does not record one.
- Checks from the tray every 24 hours by default; Refresh checks at once.
- Can leave out your own downloads of each macOS disk image, if you download
  them yourself to check notarisation; none are left out until you say so.
- Keeps the GoatCounter key and the optional GitHub token in Windows Credential
  Manager.
- Opens dark, with a button to move it to light.

## Built with

| Piece | What |
|---|---|
| Language | Go; builds with no C compiler |
| Window | Wails v2 on WebView2 |
| Page | React 18 and TypeScript, built by Vite |
| History | SQLite through `modernc.org/sqlite` (pure Go) |
| Secrets | Windows Credential Manager through `zalando/go-keyring` |
| Layering | `internal/{domain,application,infrastructure}`, enforced by tests |

## Running it

Download the setup program from the
[latest release](https://github.com/oernster/Visitron/releases/latest). The
history and the run log (`Log.txt`) live in `%LOCALAPPDATA%\Visitron`.

## Testing

```powershell
./test.ps1
```

See [TESTING.md](TESTING.md).

## Building

```powershell
./build.ps1
```

The gate runs first; see [DEVELOPMENT.md](DEVELOPMENT.md).

## Documentation

- [REQUIREMENTS.md](REQUIREMENTS.md): what Visitron must do, with the tests that verify it.
- [ARCHITECTURE.md](ARCHITECTURE.md): the invariants and the tests that enforce them.
- [TESTING.md](TESTING.md): the gate and its floors.
- [DEVELOPMENT.md](DEVELOPMENT.md): building from source.
- [DECISIONS-TRADEOFFS.md](DECISIONS-TRADEOFFS.md): the decisions Visitron rests on, with their costs.

## Supporting the project

Visitron is free and stays free. There is no paid tier, no licence key and no
feature held back behind a donation. The Donate button in the bar opens a
contribution page in your browser.

<a href="https://www.paypal.com/ncp/payment/NRXS4SP24A6C8"><img src="docs/donate.png" alt="Donate to Visitron" width="120"></a>

## Licence

GPL-3.0. See [LICENSE](LICENSE). For terms that suit a closed-source product,
see [commercial licensing](https://ernster.dev/commercial-licensing.html).
