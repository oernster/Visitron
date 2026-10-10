# Visitron: development

Every command is PowerShell, run from the repository root.

## Tools

| Tool | What for | Where from |
|---|---|---|
| Go, at the version `go.mod` names or later | The application and the setup program | https://go.dev/dl/ |
| Node with npm | The page | https://nodejs.org/ |
| Wails CLI | Building both windows | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| WebView2 runtime | Running both windows | Ships with Windows 11 |
| git | The release repository read at build time; the identity guard | https://git-scm.com/ |
| Python with Pillow | Regenerating the icons only | `python -m pip install pillow` |

staticcheck is not installed: `test.ps1` runs it through `go run`. The code
imports no C, so no C compiler is needed.

## Building

Builds are driven through BuildPilot, which finds `build.ps1`, runs it with no
arguments or prompts and reads its exit code alone. Its Launch installer looks
for `dist-installer/VisitronSetup.exe`. By hand:

```powershell
./build.ps1
```

In order, it:

1. Reads `VERSION`; an empty file stops the build.
2. Runs [`test.ps1`](TESTING.md) and stops on a failure, unless `-Fast` was
   given.
3. Runs `tools/releaserepo.ps1`, which reads owner/name off the origin remote;
   without a GitHub origin the build has no update source and says so.
4. Runs `wails build` with `-X main.appVersion` and `-X main.releaseRepo`,
   which installs the page's dependencies, builds the page and compiles
   `build/bin/Visitron.exe`, then checks the file is there.
5. Copies `LICENSE` beside the executable and zips `build/bin` into
   `installer/payload.zip`.
6. Builds the setup program in `installer/` with the same version flag.
7. Copies it to `dist-installer/VisitronSetup.exe`, then writes the empty-zip
   placeholder back to `installer/payload.zip` so a payload never reaches a
   commit.

`-SkipInstaller` stops after step 4. `-Fast` skips the gate for a working
loop, says so in its output and is never how a release is cut.

## Running from source

```powershell
wails dev
```

The page is served by Vite, so an edit to `frontend/src` appears at once. It
uses the real data at `%LOCALAPPDATA%\Visitron\visitron.db` and the real
secrets in Credential Manager. To leave the data alone, run the built
executable against another folder:

```powershell
$env:LOCALAPPDATA = "$env:TEMP\visitron-sandbox"; ./build/bin/Visitron.exe
```

The log is `%LOCALAPPDATA%\Visitron\Log.txt`. It holds each run's start line;
each check's start, every website's outcome, the page loads and the end with
its duration; the failures the declared writers report and any crash. Never
the key or the token.

## Generated assets

The artwork masters live in `assets/`.

```powershell
python tools/genicons.py
```

It writes `build/windows/icon.ico` with a byte-for-byte copy at
`installer/build/windows/icon.ico` for the setup program, `build/appicon.png` and the page icons in
`frontend/src/assets/icons`. The output is committed, so building needs
neither Python nor Pillow. Run it whenever a master changes.

## Versioning

`VERSION` at the root is the single source. `build.ps1` passes it to Go with
`-ldflags "-X main.appVersion=..."`. `-X` writes only to a `var`, so
`appVersion` in `main.go` is declared `var`; against a `const` the flag does
nothing.

## Cutting a release

1. Bump `VERSION` if a bump is owed against the newest tag.
2. Build through BuildPilot (or `./build.ps1`) and confirm exit code `0`.
3. Commit, tag and publish `dist-installer/VisitronSetup.exe` as a GitHub
   release of the origin repository; the update check reads its
   `releases/latest`. These steps are the owner's to run.

## Standing rules

- Structure before code: a change to behaviour amends
  [REQUIREMENTS.md](REQUIREMENTS.md) first, as a numbered amendment naming the
  test that verifies it.
- The domain performs no IO and never reads the clock; only `main.go` wires
  infrastructure to the application.
- No file over 400 lines; one landing between 381 and 400 is cut to 350 or
  fewer.
- Every colour goes in `frontend/src/theme.css`.
- A shape crossing the window boundary is written in both `dto.go` and
  `frontend/src/api.ts`.
- Nothing names the person building it: no account, domain or address in
  code, page or specification. Examples use `example.com`, `owner/name`, `youraccount`.
- A new HTTP package, outbound address, log writer or runtime dependency fails
  `tests/structural` until it is declared there.
- A new guard is not a guard until a planted violation has made it fail.

## Further reading

- [ARCHITECTURE.md](ARCHITECTURE.md): the invariants and the layers.
- [TESTING.md](TESTING.md): the gate and the floors.
