#!/usr/bin/env bash
# cleanup_flatpak.sh - Uninstall and purge the Visitron Flatpak. Ported from PigeonPost.
#
#   bash cleanup_flatpak.sh
#
# Scoped to flatpak artefacts only. It deliberately does NOT touch the outputs
# (build/bin, dist-installer, dist-dmg, Visitron.dmg) produced by the other build
# paths, so the build paths stay independent. The user's own data under
# ~/.var/app/uk.codecrafter.Visitron is left alone too: uninstalling a program is
# not a request to delete its history.
set -euo pipefail

APP_ID="uk.codecrafter.Visitron"
BIN_NAME="visitron"

bold=$(tput bold 2>/dev/null || true)
reset=$(tput sgr0 2>/dev/null || true)
section() { echo; echo "${bold}=== $* ===${reset}"; }

section "Uninstalling ${APP_ID}"
if flatpak list --user --app --columns=application | grep -qx "${APP_ID}"; then
    flatpak uninstall --user -y "${APP_ID}"
    echo "  Uninstalled."
else
    echo "  Not installed, skipping."
fi

section "Removing flatpak build artefacts"
rm -f "${BIN_NAME}.flatpak"
rm -rf .flatpak-build .flatpak-repo .flatpak-builder
rm -f "${APP_ID}.yml"
rm -rf packaging/
echo "  Done."

echo
echo "${bold}Purge complete.${reset}"
