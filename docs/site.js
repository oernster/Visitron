// The site's two small jobs, ported from SymDiary's: the appearance toggle; and
// reading the newest release so the page can say how big the setup program is.
//
// The toggle mirrors the application's own rule (FR-070). The page opens dark
// and wears light only when the reader asks for it; the choice is remembered in
// this browser. The face shown is the appearance it would move TO, so the sun
// appears while you are in the dark, exactly as the button in the window does.

const STORED = 'visitron.site.theme'
const DARK_FACE = 'dark-mode.png'
const LIGHT_FACE = 'light-mode.png'

/** stored answers the remembered appearance, else dark. */
function stored() {
    try {
        return window.localStorage.getItem(STORED) === 'light' ? 'light' : 'dark'
    } catch {
        return 'dark'
    }
}

/** apply dresses the page and the toggle in one act, so a repaint cannot leave
 *  the button offering the appearance just departed. */
function apply(theme) {
    document.documentElement.setAttribute('data-theme', theme)
    const face = document.getElementById('theme-face')
    if (!face) {
        return
    }
    const toLight = theme === 'dark'
    face.src = toLight ? LIGHT_FACE : DARK_FACE
    face.alt = toLight ? 'Switch to light mode' : 'Switch to dark mode'
}

/** toggle moves between the two and remembers which. A browser that refuses to
 *  store it still changes appearance; it simply forgets by the next visit. */
function toggle() {
    const next = stored() === 'dark' ? 'light' : 'dark'
    try {
        window.localStorage.setItem(STORED, next)
    } catch {
        /* nothing to do: the appearance still changes for this visit */
    }
    apply(next)
}

// The version shown on the page is stamped from VERSION by stamp_version.py, so
// this does not write one. The download button points at GitHub's
// releases/latest/download redirect, which always serves the newest release;
// the file name carries no version, so the link cannot go stale. This only
// decorates: where the release notes are and how big the file is. Deliberately
// NOT the date it was published: the site carries no visible dates. Where the
// request fails, what is already written stands on its own.
function decorateFromLatestRelease() {
    fetch('https://api.github.com/repos/oernster/Visitron/releases/latest')
        .then((answer) => (answer.ok ? answer.json() : null))
        .then((release) => {
            if (!release) {
                return
            }
            showNotes(release.html_url)
            showSizes(release.assets || [])
        })
        .catch(() => {
            /* no network, no decoration; the page is complete without it */
        })
}

/** showNotes points the release-notes link at this particular release rather
 *  than at whatever is newest when somebody follows it. */
function showNotes(url) {
    const link = document.getElementById('dl-whats-new')
    if (link && url) {
        link.href = url
    }
}

/** showSizes adds each file's size to the line naming what it is, so somebody
 *  on a slow connection knows what they are starting. */
function showSizes(assets) {
    const megabyte = 1024 * 1024
    for (const asset of assets) {
        const line = document.querySelector('[data-asset="' + asset.name + '"]')
        if (line && asset.size) {
            line.textContent = line.textContent + ' · ' + (asset.size / megabyte).toFixed(1) + ' MB'
        }
    }
}

apply(stored())
document.getElementById('theme-toggle').addEventListener('click', toggle)
decorateFromLatestRelease()
