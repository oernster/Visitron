# Visitron: development

Every command is PowerShell, run from the repository root, apart from the macOS
and Linux builds, which are bash on those machines.

## Tools

| Tool | What for | Where from |
|---|---|---|
| Go, at the version `go.mod` names or later | The application and the setup program | https://go.dev/dl/ |
| Node with npm | The page | https://nodejs.org/ |
| Wails CLI | Building both windows | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| WebView2 runtime | Running both windows on Windows | Ships with Windows 11 |
| git | The release repository read at build time; the identity guard | https://git-scm.com/ |
| Python with Pillow | Regenerating the icons only | `python -m pip install pillow` |

staticcheck is not installed: `test.ps1` runs it through `go run`. The code
imports no C, so no C compiler is needed on Windows. The macOS build needs the
Xcode command line tools (Wails compiles against WebKit through cgo); the Linux
build happens inside the Flatpak SDK, which carries its own.

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
3. Runs `go run ./tools/releaserepo`, which reads owner/name off the origin remote;
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

### macOS (Apple Silicon)

Needs Go, Node with npm, the Xcode command line tools and a Developer ID
Application certificate in the login keychain. The script installs the Wails
CLI and `create-dmg` (through Homebrew) when they are missing.

```bash
bash builddmg.sh
```

It reads owner/name off the origin remote with `go run ./tools/releaserepo`,
runs `wails build` for `darwin/arm64` with the same `-X` flags as `build.ps1`,
stamps the bundle version from `VERSION`, copies `LICENSE` into the bundle,
signs it with the hardened runtime, notarizes and staples it, then wraps it in
`Visitron.dmg`, which is signed, notarized and stapled in turn.

The signing identity is the one `Developer ID Application` certificate in the
keychain; with two or more, name one in `DEVELOPER_ID_APPLICATION`. Nothing in
the script names whose it is. Notarization uses the keychain profile
`Visitron`, made once with
`xcrun notarytool store-credentials Visitron --apple-id <id> --team-id <team> --password <app-specific>`;
else `APPLE_ID` and `APPLE_APP_PASSWORD` (an app-specific password, checked for
its shape before anything is built). `ALLOW_UNNOTARIZED=1` skips notarization
for a local test build that is never published.

### Linux (Flatpak)

Needs `flatpak` and `flatpak-builder` (installed through apt, dnf, pacman or
zypper when missing) and Go on the host, for the release repository. Node and
the Go that builds the application come from the Flatpak SDK extensions.

```bash
bash build_flatpak.sh
```

It installs the GNOME 50 runtime with the Go and Node SDK extensions from
Flathub, reads the release repository, writes the desktop file, the metainfo
and the manifest, builds the page and the application inside the sandbox
against WebKitGTK 4.1 and installs it for the current user
(`flatpak run uk.codecrafter.Visitron`). It also exports `visitron.flatpak`.
The sandbox gets the network, the Secret Service and the single-instance
D-Bus name; it gets no access to your files.

```bash
bash cleanup_flatpak.sh
```

removes the user install and every flatpak build output; the other platforms'
outputs and the user's own history are left alone.

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

The log is `%LOCALAPPDATA%\Visitron\Log.txt` (`~/Library/Logs/Visitron` on
macOS; `$XDG_STATE_HOME/Visitron`, else `~/.local/state/Visitron`, on Linux). It holds each run's start line;
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
`frontend/src/assets/icons`, with byte-for-byte copies of the ones the site shows in `docs/`
(GitHub Pages serves that folder alone). The output is committed, so building needs
neither Python nor Pillow. Run it whenever a master changes.

## Versioning

`VERSION` at the root is the single source. `build.ps1`, `builddmg.sh` and
`build_flatpak.sh` all pass it to Go with `-ldflags "-X main.appVersion=..."`;
the DMG also stamps it into the bundle's `Info.plist` and the Flatpak into its
metainfo release entry. `-X` writes only to a `var`, so
`appVersion` in `main.go` is declared `var`; against a `const` the flag does
nothing.

## Cutting a release

1. Bump `VERSION` if a bump is owed against the newest tag.
2. Build through BuildPilot (or `./build.ps1`) and confirm exit code `0`;
   build `Visitron.dmg` with `bash builddmg.sh` on an Apple Silicon Mac (never
   with `ALLOW_UNNOTARIZED=1`) and `visitron.flatpak` with
   `bash build_flatpak.sh` on Linux, all from the same commit.
3. Commit, tag and publish `dist-installer/VisitronSetup.exe`,
   `Visitron.dmg` and `visitron.flatpak` as one GitHub release of the origin
   repository. The update check reads its `releases/latest` and offers each
   platform the file with its ending, so the three names must keep `.exe`,
   `.dmg` and `.flatpak`. These steps are the owner's to run.

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
