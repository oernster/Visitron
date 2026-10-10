/* ----------------------------------------------------------------- routes */

// One reading of the machine decides the screen, its heading, its options and
// its buttons. Deciding it once is what stops those four drifting apart.

function routeInstall(state) {
    $('install-title').textContent = `Install ${appName} ${state.thisVersion}`
    $('install-path').textContent = state.installDir
    const read = renderOptions($('install-options'), shortcutOptions(state).concat([
        launchOption(),
    ]))
    showScreen('install')
    setFooter([
        {label: 'Cancel', onClick: () => backend().Quit()},
        {
            label: 'Install', kind: 'primary',
            onClick: () => install(read, `Installing ${appName}`,
                `${appName} is installed`,
                'You are on v' + state.thisVersion + '.'),
        },
    ])
}

// routeChange serves both directions of a version change: an update and a way
// back differ only in wording and in which button is the safe one. Neither
// heading names a single version, because both are about two; the flow line
// below carries them.
function routeChange(state) {
    const goingBack = state.relation === 'older'
    $('update-title').textContent = goingBack ? 'Go back a version?' : 'Update available'
    $('update-lead').textContent = goingBack
        ? 'This setup file carries an older version than the one installed. Your websites and their history are untouched.'
        : 'A newer version is ready to install. Your websites and their history are untouched.'
    $('update-from').textContent = 'v' + state.installedVersion
    $('update-to').textContent = 'v' + state.thisVersion
    const read = renderOptions($('update-options'), shortcutOptions(state).concat([
        launchOption(),
    ]))
    showScreen('update')
    setFooter([
        {label: 'Uninstall', kind: 'danger', onClick: () => routeUninstall(state)},
        {label: 'Not now', onClick: () => backend().Quit()},
        {
            label: goingBack ? 'Go back' : 'Update', kind: 'primary',
            onClick: () => install(read,
                goingBack ? 'Going back a version' : `Updating ${appName}`,
                goingBack ? 'Version changed' : `${appName} is updated`,
                'You are on v' + state.thisVersion + '.'),
        },
    ])
}

// routeManage is the screen for a matching version. Its boxes act immediately,
// since there is nothing to install: a box that waited for a go-ahead would
// never take effect at all.
function routeManage(state) {
    $('manage-title').textContent = `${appName} ${state.installedVersion} is installed`
    const live = () => backend().SetShortcuts(read('startMenu'), read('desktop'))
    const read = renderOptions($('manage-options'), [
        {
            key: 'startMenu', label: 'Add a Start Menu entry',
            checked: state.startMenu, onChange: () => live(),
        },
        {
            key: 'desktop', label: 'Add a Desktop shortcut',
            checked: state.desktop, onChange: () => live(),
        },
        {...startOption(state), onChange: () => backend().SetStartWithWindows(read('startWithWindows'))},
        launchOption(),
    ])
    showScreen('manage')
    setFooter([
        {label: 'Uninstall', kind: 'danger', onClick: () => routeUninstall(state)},
        {label: 'Close', onClick: () => backend().Quit()},
        {
            label: 'Reinstall', onClick: () => finish(
                () => backend().Install(freshChoices), read('launch'),
                `Reinstalling ${appName}`, `${appName} is reinstalled`,
                'The files were written again and the shortcuts put back as a new install would leave them.'),
        },
        {
            label: 'Repair', kind: 'primary',
            onClick: () => finish(
                () => backend().Repair(), read('launch'),
                `Repairing ${appName}`, 'Repair complete',
                'The files have been put back and nothing else was changed.'),
        },
    ])
}

// stopReading ends the licence pane's self-reading cycle. It is held here
// because a screen stack has no unmount to hook: the one place that knows the
// screen has changed is the one place that can stop the timer.
let stopReading = null

// leaveLicence stops the cycle if one is running. Called before every route,
// so no path off the screen can leave a timer behind it.
function leaveLicence() {
    if (stopReading) {
        stopReading()
        stopReading = null
    }
}

// routeLicence is reachable from every screen that offers the header's
// controls, which is every screen but progress. Closing it re-derives the
// screen behind it from the same reading of the machine, exactly as cancelling
// a removal does, so no second copy of "where was I" has to be kept in step.
//
// Every word on the screen comes from Go. Naming a licence explains nothing to
// most people, so the plain reading is shown first and the published text sits
// beneath it, reading itself down for anyone who would rather watch than
// scroll.
function routeLicence(state) {
    const licence = backend().Licence()
    const close = () => {
        leaveLicence()
        if (state) {
            route(state)
            return
        }
        backend().Quit()
    }
    setFooter([{label: 'Close', kind: 'primary', onClick: close}])
    showScreen('licence')
    Promise.resolve(licence).then((held) => fillLicence(held))
}

// fillLicence writes what Go answered onto the screen and starts the pane
// reading itself. A screen reached again starts a fresh cycle, start hold and
// all, which is right: somebody returning to a licence is starting to read it
// again.
function fillLicence(held) {
    $('licence-lead').textContent = held.lead + ' © ' + held.holder
    const plainly = $('licence-plainly')
    plainly.replaceChildren()
    for (const line of held.plain) {
        const item = document.createElement('li')
        item.textContent = line
        plainly.appendChild(item)
    }
    // The works setup is built from go at the foot of the licence rather than
    // in a box of their own: the window is fixed; a box up here takes the
    // room the licence pane needs. They are still read out by the pane's own
    // cycle, which is more than a box that scrolled off would have managed.
    const pane = $('licence-text')
    pane.textContent = [held.text.trimEnd(), 'Built with:', ...held.notices].join('\n\n')
    pane.scrollTop = 0
    leaveLicence()
    stopReading = readItself(pane)
}

// routeUninstall is reachable from every other screen. The record is kept
// unless the box is ticked; the box names the file it would delete.
function routeUninstall(state) {
    $('record-path').textContent = state.recordFile
    const read = renderOptions($('uninstall-options'), [
        {
            key: 'record', label: 'Also delete my websites and their history',
            hint: 'Every website, its counts and the log go with it. This cannot be undone.',
            checked: false,
        },
    ])
    showScreen('uninstall')
    setFooter([
        {label: 'Cancel', onClick: () => state.installed ? route(state) : backend().Quit()},
        {
            label: 'Uninstall', kind: 'danger',
            onClick: () => withAppClosed(() => run(
                () => backend().Uninstall(read('record')),
                `Removing ${appName}`, `${appName} is removed`,
                read('record')
                    ? 'The application, its shortcuts and your data are gone.'
                    : 'The application and its shortcuts are gone. Your data is still there.')),
        },
    ])
}

function route(state) {
    // Every way off the licence screen comes through here, so this is where
    // the pane's timer stops; a screen stack has no unmount to hang it on.
    leaveLicence()
    currentState = state
    if (state.mode === 'uninstall') {
        routeUninstall(state)
    } else if (state.mode !== 'manage') {
        routeInstall(state)
    } else if (state.relation === 'same') {
        routeManage(state)
    } else {
        routeChange(state)
    }
}

async function init() {
    applyTheme(storedTheme())
    $('theme-toggle').addEventListener('click', toggleTheme)
    $('licence-open').addEventListener('click', () => routeLicence(currentState))
    let tries = 0
    while (!backend() && tries < 100) {
        await new Promise((resolve) => setTimeout(resolve, 50))
        tries++
    }
    if (!backend()) {
        $('error-msg').textContent = 'Could not reach the setup program.'
        showScreen('error')
        return
    }
    window.runtime.EventsOn('progress', onProgress)
    const state = await backend().DetectState()
    appName = state.appName
    document.title = `${appName} Setup`
    $('brand').textContent = `${appName} Setup`
    $('uninstall-title').textContent = `Remove ${appName}?`
    $('running-title').textContent = `${appName} is open`
    route(state)
    window.focus()
    focusFooter()
}

// A missing mark leaves no broken image in the header.
const markImage = $('markimg')
markImage.onerror = () => { markImage.remove() }

window.addEventListener('DOMContentLoaded', init)
