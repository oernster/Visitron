# Decisions and trade-offs

The deliberate choices Visitron rests on: what was chosen, what was given up
for it and why. Each entry is the decision as the product makes it today. The
specification ([REQUIREMENTS.md](REQUIREMENTS.md)) holds the detail and the
tests behind each one; its numbered amendments record what was reversed.

## The product as a whole

### Local first, one owner, one machine

Everything Visitron keeps (the websites, the daily history, the preferences)
is one SQLite file on the computer it runs on, for the one person whose sites
they are.

- **Rather than:** an account, a server or a hosted dashboard.
- **Gains:** nothing to sign in to beyond the two services it reads; the
  figures already held still show with the network off.
- **Costs:** the history belongs to that one machine; a second machine starts
  its own from nothing.

### Windows only

Visitron is built for Windows alone, as a tray application started with
Windows.

- **Rather than:** macOS and Linux packages as well.
- **Gains:** one tray, one secret store and one setup program to get right.
- **Costs:** anyone on another desktop cannot run it.

### Go and Wails, with no C compiler

The application is Go behind a Wails window over a React page; the database
driver is written in Go alone.

- **Rather than:** a C database driver; a toolkit drawn natively.
- **Gains:** one toolchain builds the release; the gate, the setup program and
  the update check were ported from earlier projects on the same stack.
- **Costs:** the window depends on the WebView2 component being present.

### Specification before code

Every behaviour has a numbered requirement naming the test that proves it; a
change of behaviour arrives as a numbered amendment with its reason.

- **Rather than:** building first and describing afterwards.
- **Gains:** a reversal keeps its reason; a reference to a test that does not
  exist fails the suite.
- **Costs:** keeping the specification true is work of its own.

### Read only

Visitron reads GoatCounter and GitHub and changes nothing on either.

- **Rather than:** managing releases or counters from the window.
- **Gains:** a fault in Visitron can never damage what it watches; a read-only
  key is enough.
- **Costs:** every change to a site or a release is made elsewhere.

## Where the figures come from

### Page loads from GoatCounter

Page loads are read from the owner's own GoatCounter account through its
statistics API. That API reports visitors per page per day, which Visitron
shows under the name page loads; the Guide says so.

- **Rather than:** a counter of Visitron's own; another analytics service;
  GoatCounter's export of raw loads, which needs a wider permission.
- **Gains:** nothing new to host or embed; a key that can only read
  statistics; GoatCounter's own history reaches back a year from the first
  check.
- **Costs:** one person reading a page twice in a day counts once; every
  counted page must carry the tag; the owner needs a GoatCounter account.

### Downloads from GitHub releases alone

Downloads are GitHub's own count for each file attached to a release of a
public repository, grouped by platform from the file's extension.

- **Rather than:** other release hosts; private repositories.
- **Gains:** one source, readable anonymously; a token only raises the rate
  limit.
- **Costs:** a release hosted anywhere else is invisible.

### A website is an address prefix; the longest owns a path

A website is a host plus a path. A GoatCounter path belongs to the website
with the longest address that begins it, so a sub-site under a site counts
separately from the site above it.

- **Rather than:** one website per host; a path shared between websites.
- **Gains:** several applications on one domain each get their own figures;
  no page load is counted twice.
- **Costs:** the path a site records must match the address given for it,
  case included.

### Repositories are proposed by a crawl and chosen by the owner

Adding a website crawls its own pages, within fixed limits, for the GitHub
repositories they name; the owner ticks which count. A likely match is
pre-ticked; one can be typed by hand.

- **Rather than:** typing every repository; counting every one found.
- **Gains:** the common case is one address and a glance; a stray link to
  someone else's project is not counted.
- **Costs:** a site that names no repository in its pages needs them typed.

### History begins at the first check

GitHub keeps only a running total per file, so a day's downloads are the rise
between Visitron's own daily snapshots, the last check of each day kept. A
period longer than the history held says the day counting began.

- **Rather than:** estimating earlier days from release dates.
- **Gains:** every figure shown is one that was measured.
- **Costs:** downloads before the first check are a total with no shape; the
  first day has nothing to rise from.

### GitHub's counts stored; own downloads taken off on reading

The store keeps GitHub's counts as read. The owner's own downloads of each
disk image, a setting that is none until set, are taken off each disk image
file when the figures are read, never below none.

- **Rather than:** a fixed allowance; storing the corrected figure.
- **Gains:** anyone can say how they test their disk images; a change applies
  to the whole history at once; GitHub's figure is never lost.
- **Costs:** the correction is worked out again at every read; it assumes the
  owner downloads only disk images.

### A falling count is not a negative day

Where a file's count falls between snapshots or the file disappears, that day
records no rise for it.

- **Rather than:** reporting the fall.
- **Gains:** a deleted or re-uploaded release cannot produce a day of negative
  downloads.
- **Costs:** downloads made between the reset and the next check are lost.

## Checking

### A daily check from the tray

A check runs once the interval (a day unless changed) has passed since the
last success, at start, on waking or when asked. One runs at a time. A failure
keeps the figures held, is shown on Refresh and waits half an hour before an
automatic retry.

- **Rather than:** checking on every open; retrying every minute.
- **Gains:** the history fills in without the window open; an outage costs a
  few requests rather than a stream of them.
- **Costs:** a day with no successful check has no snapshot; its downloads
  land on the next day that has one.

### Each snapshot is written whole

A check's figures are saved in one transaction.

- **Rather than:** writing file by file.
- **Gains:** a quit, crash or power cut leaves the last good figures intact.
- **Costs:** none recorded.

## Privacy and the network

### Secrets in Windows Credential Manager

The GoatCounter key and the GitHub token are kept by Windows, never in a file
or the log; Settings shows a stored one behind an eye when asked.

- **Rather than:** a settings file; secrets that can never be seen again.
- **Gains:** the secrets are protected by the account like any other Windows
  credential; the owner can see what is stored.
- **Costs:** the secrets do not travel with a copy of the data folder.

### Only named hosts are reached

Visitron reaches GoatCounter, GitHub, the websites it crawls and nothing else
of its own accord; one client carries every request and a structural test
lists the packages allowed to make one. A secret goes only to its own service:
a next page named on another host is refused rather than sent the token.

- **Rather than:** a promise to use the network sparingly.
- **Gains:** no telemetry is possible without a test failing first; a hostile
  answer cannot lift a token.
- **Costs:** every new outward feature has to argue for its route.

### The page never names an address

Donate and an update's Download ask the Go side, which holds the only address
and hands it to the browser; an update address not on GitHub is refused.

- **Rather than:** links written in the page.
- **Gains:** nothing from the page or from GitHub's answer has to be trusted
  before it is opened.
- **Costs:** an address changes only with a new release.

### No one's identity in the code

Visitron is released for anyone, so it names no account, domain or site of
the person who builds it; a first run starts with no websites. A structural
test learns that identity from the checkout and refuses it.

- **Rather than:** shipping the author's own sites and account as defaults.
- **Gains:** anyone can use it as their own; nothing personal leaks through
  examples, fixtures or help.
- **Costs:** every owner sets up GoatCounter and adds their sites by hand.

### Updates checked against GitHub releases, from the build's own repository

Visitron asks GitHub for its newest release shortly after opening and daily,
quiet unless there is news; it can be switched off. The repository asked is
read from the git remote the build came from, so a fork checks its own.

- **Rather than:** no check; a repository written in the source.
- **Gains:** new releases are found without nagging; the source names no one.
- **Costs:** a build with no GitHub remote cannot check for updates.

## The interface

### Closing asks; one copy runs

The close button asks whether to minimise to the tray or quit; without a tray
icon it quits. A second launch brings the running window forward.

- **Rather than:** closing silently to the tray.
- **Gains:** the history keeps filling without a process the owner cannot
  find; there is only ever one writer.
- **Costs:** one more question on closing.

## Engineering

### Layers with one place where they meet

The rules, the use cases, the parts that touch the outside world and the
window each depend only inward, wired in one place; a structural test holds
the boundaries. The rules and the use cases are fully covered.

- **Rather than:** convention alone; one coverage figure over everything.
- **Gains:** the figures are tested with no disk, network or clock.
- **Costs:** more packages and explicit wiring.

### Tests with real parts; guards proved to bite

The store is tested against a real database; fakes are written by hand; every
structural guard was proved by planting a violation.

- **Rather than:** a mocking library and guards assumed to work.
- **Gains:** a passing test means the real thing works.
- **Costs:** the fakes are Visitron's own to maintain.
