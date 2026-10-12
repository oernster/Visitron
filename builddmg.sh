#!/usr/bin/env bash
# Builds the Visitron macOS DMG for Apple Silicon. Ported from PigeonPost. Run on an arm64
# Mac from the repo root:
#
#   bash builddmg.sh
#
# Flow: read the release repository off the origin remote, wails build for darwin/arm64
# with the version and release repository passed through -ldflags -X, stamp the bundle
# version from VERSION, codesign the .app (hardened runtime), notarize and staple the .app,
# stage it with ditto, create-dmg, stamp the DMG file icon, sign the DMG, then notarize and
# staple the DMG.
#
# Notarization is mandatory. A Developer ID signature alone is not enough: since macOS 10.15
# Gatekeeper rejects signed-but-unnotarized apps with "Apple could not verify ... is free of
# malware". With APPLE_ID and APPLE_APP_PASSWORD both set it notarizes as that Apple ID (a
# password not shaped like an app-specific one stops the build before anything is built);
# with either unset it notarizes through the keychain profile.
#
# No one's identity is written here (Amendment 15): the signing identity is found in the
# login keychain and the team is read off it, so whoever builds signs as themselves.
#
# Environment overrides:
#   DEVELOPER_ID_APPLICATION   signing identity; else the one Developer ID Application
#                              identity in the keychain (two or more must be named)
#   APPLE_ID, APPLE_APP_PASSWORD, APPLE_TEAM_ID   notarization credentials (else the profile);
#                              the team defaults to the one in the signing identity
#   APPLE_KEYCHAIN_PROFILE     keychain profile name (defaults to Visitron)
#   ALLOW_UNNOTARIZED=1        build without notarizing; local testing only, never released
#
# Output: Visitron.dmg in the repo root
set -euo pipefail

APP_NAME="Visitron"
PLATFORM="darwin/arm64"
APP_BUNDLE="build/bin/${APP_NAME}.app"
DIST_DIR="dist-dmg"
STAGE_DIR="${DIST_DIR}/stage"
VERSION="$(tr -d '[:space:]' < VERSION)"
# DIST_DIR/STAGE_DIR are scratch space; the final DMG lands in the repo root.
DMG_PATH="${APP_NAME}.dmg"
DEVELOPER_ID_PREFIX="Developer ID Application:"

DEVELOPER_ID="${DEVELOPER_ID_APPLICATION:-}"
APPLE_ID="${APPLE_ID:-}"
APPLE_APP_PASSWORD="${APPLE_APP_PASSWORD:-}"
APPLE_TEAM_ID="${APPLE_TEAM_ID:-}"
# Escape hatch for local test builds. Distribution builds must never set this: an
# unnotarized DMG is rejected by Gatekeeper on every machine but the one that signed
# it; the failure is invisible at build time.
ALLOW_UNNOTARIZED="${ALLOW_UNNOTARIZED:-}"
# The notarization credential for this app, created once with
#   xcrun notarytool store-credentials Visitron \
#     --apple-id <id> --team-id <team> --password <app-specific>
# One profile per app means a leaked credential can be revoked for a single app. Stated
# explicitly rather than derived from APP_NAME: the profile is a fact registered with
# Apple; deriving it would silently change which credential the build looks for if
# that name were ever edited. APPLE_KEYCHAIN_PROFILE overrides it.
NOTARY_PROFILE="${APPLE_KEYCHAIN_PROFILE:-Visitron}"
# The notary service accepts only an app-specific password from appleid.apple.com and
# rejects the Apple account password with HTTP 401. The shape is distinctive, so it is
# checked before the build rather than discovered after it.
APP_SPECIFIC_PASSWORD_RE='^[a-z]{4}-[a-z]{4}-[a-z]{4}-[a-z]{4}$'
# A Developer ID identity ends with its team in parentheses: "... Name (TEAMID)".
TEAM_IN_IDENTITY_RE='\(([A-Z0-9]+)\)$'
# Notarization is the default and the keychain profile always resolves, so the only way
# to skip it is to ask for that explicitly.
NOTARIZING=1
[ "${ALLOW_UNNOTARIZED}" = "1" ] && NOTARIZING=0

section() { printf '\n\033[1m== %s ==\033[0m\n' "$1"; }

# notarytool_submit uploads a file to Apple and waits for the verdict. notarytool exits
# non-zero on an Invalid verdict; set -e then fails the build rather than leaving an
# artifact that looks distributable. Stapling is a separate step because the file that is
# submitted and the file that carries the ticket differ for a bundle: a zip goes up, the
# .app gets stapled.
# The password never reaches the echoed command: with a keychain profile it is not on
# the command line at all; the fallback branch prints a masked form.
notarytool_submit() {
    if [ -n "${APPLE_ID}" ] && [ -n "${APPLE_APP_PASSWORD}" ]; then
        echo "\$ xcrun notarytool submit $1 --apple-id ${APPLE_ID} --password ******** --team-id ${APPLE_TEAM_ID} --wait"
        xcrun notarytool submit "$1" \
            --apple-id "${APPLE_ID}" \
            --password "${APPLE_APP_PASSWORD}" \
            --team-id "${APPLE_TEAM_ID}" \
            --wait
        return
    fi
    echo "\$ xcrun notarytool submit $1 --keychain-profile ${NOTARY_PROFILE} --wait"
    xcrun notarytool submit "$1" \
        --keychain-profile "${NOTARY_PROFILE}" \
        --wait
}

# require ensures a tool is on PATH, running the given install command if it is
# not. The install command is a full shell command (not just a brew formula) so
# go-installed tools like wails bootstrap the same way brew-installed ones do.
require() {
    local tool="$1" install="$2"
    if ! command -v "$tool" > /dev/null 2>&1; then
        section "Installing missing tool: $tool"
        eval "$install"
    fi
    command -v "$tool" > /dev/null 2>&1 || { echo "error: $tool is required (install: $install)" >&2; exit 1; }
}

# stamp_dmg_icon gives the .dmg file itself a custom Finder icon. create-dmg only
# sets the mounted volume icon (.VolumeIcon.icns), so without this the .dmg shows
# the generic disk-image icon in Finder/Downloads. Cosmetic: warn and skip if the
# icns or the classic resource tools are unavailable rather than failing the build.
stamp_dmg_icon() {
    local dmg="$1" icns="$2" tool
    [ -f "$icns" ] || { echo "warning: ${icns} missing; DMG file icon not set" >&2; return 0; }
    for tool in sips DeRez Rez SetFile; do
        command -v "$tool" > /dev/null 2>&1 || { echo "warning: ${tool} missing; DMG file icon not set" >&2; return 0; }
    done
    local work
    work="$(mktemp -d)"
    cp "$icns" "${work}/icon.icns"
    sips -i "${work}/icon.icns" > /dev/null
    DeRez -only icns "${work}/icon.icns" > "${work}/icon.rsrc"
    Rez -append "${work}/icon.rsrc" -o "$dmg"
    SetFile -a C "$dmg"
    rm -rf "$work"
}

section "Platform guard"
[ "$(uname -s)" = "Darwin" ] || { echo "error: this script must run on macOS" >&2; exit 1; }
[ "$(uname -m)" = "arm64" ] || { echo "error: this script targets Apple Silicon (arm64)" >&2; exit 1; }
[ -n "${VERSION}" ] || { echo "error: VERSION is empty" >&2; exit 1; }

section "Signing identity"
# Found before any build work so a missing certificate costs a second rather than a
# full wails build.
if [ -z "${DEVELOPER_ID}" ]; then
    IDENTITIES="$(security find-identity -v -p codesigning \
        | sed -n "s/.*\"\(${DEVELOPER_ID_PREFIX} .*\)\"$/\1/p")"
    COUNT="$(printf '%s' "${IDENTITIES}" | grep -c . || true)"
    if [ "${COUNT}" -eq 0 ]; then
        echo "error: no '${DEVELOPER_ID_PREFIX}' identity in the keychain." >&2
        echo "Install your Developer ID certificate; else name one with DEVELOPER_ID_APPLICATION." >&2
        exit 1
    fi
    if [ "${COUNT}" -gt 1 ]; then
        echo "error: more than one '${DEVELOPER_ID_PREFIX}' identity in the keychain:" >&2
        printf '  %s\n' "${IDENTITIES}" >&2
        echo "Name the one to sign with in DEVELOPER_ID_APPLICATION." >&2
        exit 1
    fi
    DEVELOPER_ID="${IDENTITIES}"
fi
echo "Signing as ${DEVELOPER_ID}"
if [ -z "${APPLE_TEAM_ID}" ] && [[ "${DEVELOPER_ID}" =~ ${TEAM_IN_IDENTITY_RE} ]]; then
    APPLE_TEAM_ID="${BASH_REMATCH[1]}"
fi

section "Notarization credentials"
if [ "${NOTARIZING}" -eq 0 ]; then
    echo "warning: ALLOW_UNNOTARIZED=1; local test build only, do not release the result" >&2
elif [ -n "${APPLE_ID}" ] && [ -n "${APPLE_APP_PASSWORD}" ]; then
    if ! [[ "${APPLE_APP_PASSWORD}" =~ ${APP_SPECIFIC_PASSWORD_RE} ]]; then
        cat >&2 <<EOF
error: APPLE_APP_PASSWORD is not an app-specific password.
Expected four lowercase groups of four, like abcd-efgh-ijkl-mnop.
An Apple account password is rejected by the notary service with
'HTTP status code: 401. Invalid credentials'.
Generate one at https://appleid.apple.com (Sign-In and Security, App-Specific
Passwords). Alternatively leave both variables unset and store the credential
in the keychain as profile ${NOTARY_PROFILE}.
EOF
        exit 1
    fi
    if [ -z "${APPLE_TEAM_ID}" ]; then
        echo "error: no team found in '${DEVELOPER_ID}'; set APPLE_TEAM_ID" >&2
        exit 1
    fi
    echo "Notarizing as ${APPLE_ID} (team ${APPLE_TEAM_ID})"
else
    echo "Notarizing with keychain profile ${NOTARY_PROFILE}"
fi

section "Tooling"
command -v go > /dev/null 2>&1 || { echo "error: go is required (install from https://go.dev/dl/)" >&2; exit 1; }
# wails installs into the Go bin dir; put it on PATH so a fresh install resolves.
GOBIN_DIR="$(go env GOPATH)/bin"
case ":${PATH}:" in *":${GOBIN_DIR}:"*) ;; *) PATH="${GOBIN_DIR}:${PATH}" ;; esac
export PATH
require wails "go install github.com/wailsapp/wails/v2/cmd/wails@latest"
require npm "brew install node"
require create-dmg "brew install create-dmg"

section "Release repository"
RELEASE_REPO="$(go run ./tools/releaserepo)"
if [ -n "${RELEASE_REPO}" ]; then
    echo "Updates will be read from ${RELEASE_REPO}."
else
    echo "No GitHub origin remote: this build has no update source and its window says so."
fi

section "Building ${APP_NAME} ${VERSION} (${PLATFORM})"
# The icons under build/ are committed (tools/genicons.py writes them), so nothing is
# generated here; wails turns build/appicon.png into the bundle's iconfile.icns.
wails build -clean -platform "${PLATFORM}" \
    -ldflags "-X main.appVersion=${VERSION} -X main.releaseRepo=${RELEASE_REPO}"
[ -d "${APP_BUNDLE}" ] || { echo "error: ${APP_BUNDLE} was not produced" >&2; exit 1; }

section "Stamping bundle version from VERSION"
PLIST="${APP_BUNDLE}/Contents/Info.plist"
/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString ${VERSION}" "${PLIST}" \
    || /usr/libexec/PlistBuddy -c "Add :CFBundleShortVersionString string ${VERSION}" "${PLIST}"
/usr/libexec/PlistBuddy -c "Set :CFBundleVersion ${VERSION}" "${PLIST}" \
    || /usr/libexec/PlistBuddy -c "Add :CFBundleVersion string ${VERSION}" "${PLIST}"

section "Licence beside the program"
# Section 4 of the GNU GPL asks that a copy of the licence travel with every copy of the
# program, as build.ps1 puts it in the Windows payload.
cp LICENSE "${APP_BUNDLE}/Contents/Resources/LICENSE"

section "Codesigning the app bundle"
codesign --force --deep --options runtime --sign "${DEVELOPER_ID}" "${APP_BUNDLE}"
codesign --verify --deep --strict "${APP_BUNDLE}"

# Notarize the .app before it goes into the DMG. Stapling only the DMG leaves the
# copied-out .app with no local ticket, so Gatekeeper falls back to an online check and
# the app fails to launch for anyone offline or behind a restrictive network. notarytool
# takes archives only, so ditto zips the bundle first; the ticket goes on the bundle,
# since a zip cannot carry one.
if [ "${NOTARIZING}" -eq 1 ]; then
    section "Notarizing the app bundle (this waits on Apple)"
    APP_ZIP="$(mktemp -d)/${APP_NAME}.zip"
    ditto -c -k --keepParent "${APP_BUNDLE}" "${APP_ZIP}"
    notarytool_submit "${APP_ZIP}"
    xcrun stapler staple "${APP_BUNDLE}"
    rm -rf "$(dirname "${APP_ZIP}")"
fi

section "Creating the DMG"
rm -rf "${DIST_DIR}"
rm -f "${DMG_PATH}"
mkdir -p "${STAGE_DIR}"
# ditto preserves the symlinks and metadata the embedded signature depends on.
ditto "${APP_BUNDLE}" "${STAGE_DIR}/${APP_NAME}.app"
VOLICON="${APP_BUNDLE}/Contents/Resources/iconfile.icns"
CREATE_DMG_ARGS=(
    --volname "${APP_NAME}"
    --window-size 540 380
    --icon-size 128
    --icon "${APP_NAME}.app" 140 190
    --app-drop-link 400 190
)
[ -f "${VOLICON}" ] && CREATE_DMG_ARGS+=(--volicon "${VOLICON}")
set +e
create-dmg "${CREATE_DMG_ARGS[@]}" "${DMG_PATH}" "${STAGE_DIR}"
STATUS=$?
set -e
# create-dmg exits 2 when it cannot set a custom window background (headless); still a good DMG.
if [ "${STATUS}" -ne 0 ] && [ "${STATUS}" -ne 2 ]; then
    echo "error: create-dmg failed with exit ${STATUS}" >&2
    exit "${STATUS}"
fi
rm -rf "${DIST_DIR}"

# The icon stamp writes a resource fork into the DMG, so it runs before signing and
# notarization. Doing it afterwards would modify a file Gatekeeper has already been
# told the hash of.
section "Stamping the DMG file icon"
stamp_dmg_icon "${DMG_PATH}" "${VOLICON}"

section "Signing the DMG"
codesign --force --sign "${DEVELOPER_ID}" "${DMG_PATH}"
codesign --verify "${DMG_PATH}"

if [ "${NOTARIZING}" -eq 1 ]; then
    section "Notarizing the DMG (this waits on Apple)"
    notarytool_submit "${DMG_PATH}"
    xcrun stapler staple "${DMG_PATH}"
    # stapler validate proves a ticket is attached; spctl replays the check Gatekeeper
    # runs on the end user's machine. Together they catch the silent case where signing
    # succeeded but notarization never happened.
    xcrun stapler validate "${DMG_PATH}"
    spctl --assess --type install -vv "${DMG_PATH}"
else
    section "Notarization skipped: unnotarized DMG, do not publish this build"
fi

section "Done"
echo "${DMG_PATH}"
