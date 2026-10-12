#!/usr/bin/env bash
# Builds the Visitron Flatpak for Linux. Ported from PigeonPost. Run from the repo root:
#
#   bash build_flatpak.sh
#
# Flow: install flatpak tooling if missing, add flathub, install the GNOME runtime and the
# golang and node SDK extensions, read the release repository off the origin remote,
# self-generate the desktop file, metainfo and manifest, build the app inside the sandbox
# (npm front end, then a CGO Go build against the runtime's webkit2gtk-4.1), install it for
# the current user and export a distributable bundle.
#
# The GNOME runtime is required because Wails v2 renders through webkit2gtk-4.1, which the
# freedesktop runtime does not carry. The Go build therefore uses -tags webkit2_41.
#
# Unlike PigeonPost there is no host-side `wails generate module` step: Visitron's page
# never imports the generated bindings (frontend/src/api.ts is the wire contract), so the
# sandbox builds the page from the checkout as it stands.
#
# Outputs: visitron.flatpak (installable anywhere) and a user install of ${APP_ID}.
set -euo pipefail

APP_ID="uk.codecrafter.Visitron"
BIN_NAME="visitron"
APP_NAME="Visitron"
APP_SUMMARY="Page loads and downloads for every website you run, side by side"
RUNTIME="org.gnome.Platform"
SDK="org.gnome.Sdk"
RUNTIME_VERSION="50"
SDK_EXT_VERSION="25.08"   # the freedesktop base of GNOME 50; extensions pair with it
GOLANG_EXT="org.freedesktop.Sdk.Extension.golang"
NODE_EXT="org.freedesktop.Sdk.Extension.node22"
BUILD_DIR=".flatpak-build"
REPO_DIR=".flatpak-repo"
BUNDLE="${BIN_NAME}.flatpak"
MANIFEST="${APP_ID}.yml"
PACKAGING_DIR="packaging"
# The one committed square icon. tools/genicons.py writes it from the master artwork; a
# single 256 entry in hicolor is valid and the desktop scales it, so the flatpak needs no
# icon generator of its own (and so no Python or Pillow in the sandbox).
APPICON="build/appicon.png"
APPICON_SIZE="256"
# Wails' single-instance lock on Linux owns a D-Bus name built from product.UniqueID
# (uk.codecrafter.visitron): "org.wails_app_" + the id with dots and dashes as
# underscores + ".SingleInstance". Without owning it the sandbox refuses the name and a
# second launch cannot hand over to the first.
SINGLE_INSTANCE_NAME="org.wails_app_uk_codecrafter_visitron.SingleInstance"
VERSION="$(tr -d '[:space:]' < VERSION)"
RELEASE_DATE="$(date +%F)"

section() { printf '\n\033[1m== %s ==\033[0m\n' "$1"; }

install_if_missing() {
    local tool="$1"
    command -v "$tool" > /dev/null 2>&1 && return 0
    section "Installing missing tool: $tool"
    if command -v apt-get > /dev/null 2>&1; then sudo apt-get install -y "$tool"
    elif command -v dnf > /dev/null 2>&1; then sudo dnf install -y "$tool"
    elif command -v pacman > /dev/null 2>&1; then sudo pacman -S --noconfirm "$tool"
    elif command -v zypper > /dev/null 2>&1; then sudo zypper install -y "$tool"
    else echo "error: install $tool with your package manager and re-run" >&2; exit 1
    fi
}

[ -n "${VERSION}" ] || { echo "error: VERSION is empty" >&2; exit 1; }
[ -f "${APPICON}" ] || { echo "error: ${APPICON} is missing; run python tools/genicons.py" >&2; exit 1; }

section "Tooling"
install_if_missing flatpak
install_if_missing flatpak-builder
command -v go > /dev/null 2>&1 || { echo "error: go is required on the host to read the release repository (https://go.dev/dl/)" >&2; exit 1; }

section "Flathub remote and runtimes"
flatpak remote-add --if-not-exists --user flathub https://dl.flathub.org/repo/flathub.flatpakrepo
flatpak install --user --noninteractive flathub \
    "${RUNTIME}//${RUNTIME_VERSION}" \
    "${SDK}//${RUNTIME_VERSION}" \
    "${GOLANG_EXT}//${SDK_EXT_VERSION}" \
    "${NODE_EXT}//${SDK_EXT_VERSION}"

section "Release repository"
# Read on the host, where the checkout's .git is: the sandbox is handed the tree without it.
RELEASE_REPO="$(go run ./tools/releaserepo)"
if [ -n "${RELEASE_REPO}" ]; then
    echo "Updates will be read from ${RELEASE_REPO}."
else
    echo "No GitHub origin remote: this build has no update source and its window says so."
fi

section "Writing packaging files"
mkdir -p "${PACKAGING_DIR}"

cat > "${PACKAGING_DIR}/${APP_ID}.desktop" << DESKTOP
[Desktop Entry]
Name=${APP_NAME}
Comment=${APP_SUMMARY}
Exec=${BIN_NAME}
Icon=${APP_ID}
Terminal=false
Type=Application
Categories=Network;Development;
DESKTOP

cat > "${PACKAGING_DIR}/${APP_ID}.metainfo.xml" << METAINFO
<?xml version="1.0" encoding="UTF-8"?>
<component type="desktop-application">
  <id>${APP_ID}</id>
  <name>${APP_NAME}</name>
  <summary>${APP_SUMMARY}</summary>
  <metadata_license>CC0-1.0</metadata_license>
  <project_license>GPL-3.0-only</project_license>
  <description>
    <p>
      GoatCounter counts the visits to your sites and GitHub counts the downloads of your
      releases. Visitron joins the two: each website you add gets its page loads and the
      downloads of the releases it offers, with charts of every day since the first check.
    </p>
  </description>
  <launchable type="desktop-id">${APP_ID}.desktop</launchable>
  <content_rating type="oars-1.1"/>
  <releases>
    <release version="${VERSION}" date="${RELEASE_DATE}"/>
  </releases>
</component>
METAINFO

section "Writing manifest"
# Visitron opens no file dialog and writes only under its own XDG folders, which the
# sandbox already redirects, so it is given no filesystem access at all. It needs the
# network (GoatCounter and the GitHub API), the Secret Service for its two keys and its
# single-instance name.
cat > "${MANIFEST}" << MANIFEST_EOF
app-id: ${APP_ID}
runtime: ${RUNTIME}
runtime-version: '${RUNTIME_VERSION}'
sdk: ${SDK}
sdk-extensions:
  - ${GOLANG_EXT}
  - ${NODE_EXT}
command: ${BIN_NAME}
finish-args:
  - --share=ipc
  - --socket=fallback-x11
  - --socket=wayland
  - --device=dri
  - --share=network
  - --talk-name=org.freedesktop.secrets
  - --own-name=${SINGLE_INSTANCE_NAME}
build-options:
  append-path: /usr/lib/sdk/golang/bin:/usr/lib/sdk/node22/bin
  build-args:
    - --share=network
  env:
    GOPATH: /run/build/${BIN_NAME}/gopath
    GOCACHE: /run/build/${BIN_NAME}/gocache
    GOFLAGS: -buildvcs=false
    GOTOOLCHAIN: auto
    npm_config_cache: /run/build/${BIN_NAME}/npm-cache
modules:
  - name: ${BIN_NAME}
    buildsystem: simple
    build-commands:
      # npm install, not npm ci, to match wails.json frontend:install, which is what every
      # other platform runs through wails build. Keep the two in lockstep.
      # GOTOOLCHAIN auto above lets go.mod's go line fetch a newer toolchain than the SDK
      # extension carries, over the build's network share.
      # NOTE: this heredoc is unquoted, so backticks and dollar-parens here are shell-evaluated
      # into the manifest. Never put backticks or command substitution in these comments.
      - cd frontend && npm install --no-audit --no-fund && npm run build
      - go build -tags desktop,production,webkit2_41 -ldflags "-s -w -X main.appVersion=${VERSION} -X main.releaseRepo=${RELEASE_REPO}" -o ${BIN_NAME} .
      - install -Dm755 ${BIN_NAME} /app/bin/${BIN_NAME}
      - chmod -R u+w gopath gocache 2>/dev/null || true
      - install -Dm644 LICENSE /app/share/licenses/${APP_ID}/LICENSE
      - install -Dm644 packaging/${APP_ID}.desktop /app/share/applications/${APP_ID}.desktop
      - install -Dm644 packaging/${APP_ID}.metainfo.xml /app/share/metainfo/${APP_ID}.metainfo.xml
      - install -Dm644 ${APPICON} /app/share/icons/hicolor/${APPICON_SIZE}x${APPICON_SIZE}/apps/${APP_ID}.png
    sources:
      - type: dir
        path: .
        skip:
          - .git
          - ${BUILD_DIR}
          - ${REPO_DIR}
          - .flatpak-builder
          - ${BUNDLE}
          - frontend/node_modules
          - build/bin
          - dist-installer
          - dist-dmg
MANIFEST_EOF

section "Building ${APP_NAME} ${VERSION} with flatpak-builder"
flatpak-builder --user --install --force-clean \
    --install-deps-from=flathub \
    --repo="${REPO_DIR}" \
    "${BUILD_DIR}" "${MANIFEST}"

section "Exporting bundle"
flatpak build-bundle \
    --runtime-repo=https://dl.flathub.org/repo/flathub.flatpakrepo \
    "${REPO_DIR}" "${BUNDLE}" "${APP_ID}"

section "Done"
echo "Installed for the current user: flatpak run ${APP_ID}"
echo "Distributable bundle: ${BUNDLE}"
