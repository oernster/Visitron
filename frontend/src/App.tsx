// The shell: the band of buttons, the website list beside the selected
// website's detail, the check's state line and the problem banner shown when
// the data could not be opened. Ported from SymDiary's shell.

import { useCallback, useEffect, useRef, useState } from 'react'
import { useRing } from './useRing'
import { useTheme } from './useTheme'
import { themeLabel } from './theme'
import { api, onCloseRequest, onProgress, type About, type Detail, type Overview, type Progress, type State } from './api'
import { AboutDialog, ConfirmDialog, LicenceDialog } from './Dialog'
import { Menu } from './Menu'
import { CloseChoiceDialog } from './CloseChoiceDialog'
import { UpdateDialog, useUpdateCheck } from './updates'
import { GuideDialog } from './GuideDialog'
import { SettingsDialog } from './SettingsDialog'
import { WebsiteDialog } from './WebsiteDialog'
import { DetailPane } from './DetailPane'
import { EmptyPane } from './EmptyPane'
import { missingSecrets } from './secretHelp'
import { WebsiteList } from './WebsiteList'
import addIcon from './assets/icons/add-website.png'
import editIcon from './assets/icons/edit-website.png'
import deleteIcon from './assets/icons/delete-website.png'
import refreshIcon from './assets/icons/refresh.png'
import settingsIcon from './assets/icons/settings.png'
import guideIcon from './assets/icons/help-guide.png'
import lightModeIcon from './assets/icons/light-mode.png'
import darkModeIcon from './assets/icons/dark-mode.png'
import donateIcon from './assets/icons/donate.png'
import warningIcon from './assets/icons/warning.png'

interface BandButtonProps {
  label: string
  icon: string
  hint?: string
  className?: string
  disabled?: boolean
  /** A picture drawn over the corner of the icon; the hint says what it means. */
  badge?: string
  onClick: () => void
}

function BandButton({ label, icon, hint, className, disabled, badge, onClick }: BandButtonProps) {
  return (
    <button type="button" className={className ? `band-button ${className}` : 'band-button'}
      title={hint} disabled={disabled} onClick={onClick}>
      <span className="band-picture">
        <img src={icon} alt="" />
        {badge && <img className="band-badge" src={badge} alt="" />}
      </span>
      <span>{label}</span>
    </button>
  )
}

// A rule between band groups: chrome, so no focus, no ring, hidden from a
// screen reader (FR-040).
function Separator() {
  return <span className="band-separator" aria-hidden="true" />
}

const donateHint = 'Buy the author a drink (opens your browser)'

type Open = 'add' | 'edit' | 'delete' | 'settings' | 'guide' | null

/** About and the licence both read Go's About, so one fetch serves either. */
type Shown = { kind: 'about' | 'licence'; about: About }

export function App() {
  const [state, setState] = useState<State | null>(null)
  const [overview, setOverview] = useState<Overview | null>(null)
  const [selected, setSelected] = useState<number | null>(null)
  const [detail, setDetail] = useState<Detail | null>(null)
  const [progress, setProgress] = useState<Progress | null>(null)
  const [open, setOpen] = useState<Open>(null)
  const [shown, setShown] = useState<Shown | null>(null)
  const [closing, setClosing] = useState(false)
  const updates = useUpdateCheck()
  const [message, setMessage] = useState('')
  const refused = useCallback((text: string) => setMessage(text), [])

  useRing()
  const [theme, toggleTheme] = useTheme()

  // The window opens neutral (keeb invariant 3): a sink out of the tab order
  // holds the first focus, so nothing is ringed until the first Tab.
  const start = useRef<HTMLDivElement>(null)
  useEffect(() => start.current?.focus(), [])

  const reload = useCallback(() => {
    void api.overview(refused).then((found) => found && setOverview(found))
  }, [refused])

  useEffect(() => {
    void api.state(refused).then((found) => found && setState(found))
    reload()
    return onProgress((p) => {
      setProgress(p.total > 0 ? p : null)
      if (p.total === 0) reload()
    })
  }, [refused, reload])

  // The window's close button asks rather than assumes (FR-050, Amendment 4).
  useEffect(() => onCloseRequest(() => setClosing(true)), [])

  useEffect(() => {
    if (selected === null) return setDetail(null)
    void api.detail(selected, refused).then((found) => found && setDetail(found))
  }, [selected, overview, refused])

  const row = overview?.rows.find((r) => r.id === selected)
  const running = (overview?.running ?? false) || progress !== null
  // The warning stays on Refresh until a later check succeeds, which is when
  // the facade stops reporting the failure (Amendment 3).
  const failed = Boolean(overview?.lastFailure)
  const saved = () => {
    setOpen(null)
    reload()
  }
  const remove = async () => {
    if (selected !== null && (await api.deleteWebsite(selected, refused))) {
      setSelected(null)
      saved()
    }
  }
  const period = async (days: number) => {
    if (await api.savePeriod(days, refused)) reload()
  }
  const show = (kind: Shown['kind']) => {
    void api.about(refused).then((about) => about && setShown({ kind, about }))
  }
  const name = state?.name ?? 'Visitron'

  return (
    <div className="shell">
      <div ref={start} className="focus-sink" tabIndex={-1} aria-hidden="true" />
      <nav className="band" aria-label={name}>
        <div className="band-group">
          <BandButton label="Add website" icon={addIcon} onClick={() => setOpen('add')} />
          <BandButton label="Edit website" icon={editIcon} disabled={!row} onClick={() => setOpen('edit')} />
          <BandButton label="Delete website" icon={deleteIcon} disabled={!row} onClick={() => setOpen('delete')} />
          <BandButton label={running ? 'Checking' : 'Refresh'} icon={refreshIcon} disabled={running}
            badge={failed ? warningIcon : undefined}
            hint={failed ? `The check at ${overview?.lastFailure} failed: ${overview?.failure}` : undefined}
            onClick={() => void api.refresh(refused)} />
          <BandButton label="Settings" icon={settingsIcon} onClick={() => setOpen('settings')} />
        </div>
        <div className="band-group">
          <BandButton label="Donate" icon={donateIcon} hint={donateHint} className="donate"
            onClick={() => void api.donate(refused)} />
          <Separator />
          <BandButton label={themeLabel(theme)} icon={theme === 'dark' ? lightModeIcon : darkModeIcon}
            hint={`${themeLabel(theme)} (the picture is the one you would move to)`} onClick={toggleTheme} />
          <Menu label="Help" icon={guideIcon} items={[
            { label: 'Guide', onClick: () => setOpen('guide') },
            { separator: true },
            { label: `About ${name}`, onClick: () => show('about') },
            { label: 'Licence', onClick: () => show('licence') },
            { label: 'Check for updates', onClick: updates.checkNow },
          ]} />
        </div>
      </nav>

      {state?.problem && <p className="problem" role="alert">{state.problem}</p>}

      <div className="status" role="status" aria-live="polite">
        {progress && <span>Checking {progress.site} ({progress.done + 1} of {progress.total})</span>}
        {!progress && overview && (
          <span>
            {overview.lastSuccess ? `Last checked ${overview.lastSuccess}.` : 'Not checked yet.'}
            {overview.lastFailure && ` The check at ${overview.lastFailure} failed: ${overview.failure}`}
            {overview.noKey && ' Page loads need your GoatCounter account name and key in Settings.'}
          </span>
        )}
        {message && (
          <>
            <span className="refusal">{message}</span>
            <button type="button" className="dismiss" onClick={() => setMessage('')}>
              Dismiss
            </button>
          </>
        )}
      </div>

      <main className="split">
        {overview && state && (
          <WebsiteList overview={overview} periods={state.periods} selected={selected}
            onSelect={setSelected} onPeriod={(d) => void period(d)} />
        )}
        {detail ? (
          <DetailPane detail={detail} />
        ) : (
          overview && <EmptyPane hasWebsites={overview.rows.length > 0} missing={missingSecrets(overview)} />
        )}
      </main>

      {(open === 'add' || (open === 'edit' && row)) && (
        <WebsiteDialog editing={open === 'edit' && row ? { id: row.id, url: row.url } : undefined}
          refused={refused} onSaved={saved} onClose={() => setOpen(null)} />
      )}
      {open === 'delete' && row && (
        <ConfirmDialog text={`Delete ${row.url} and its download history?`} confirmLabel="Delete"
          onConfirm={() => void remove()} onCancel={() => setOpen(null)} />
      )}
      {open === 'settings' && <SettingsDialog refused={refused} onClose={() => { setOpen(null); reload() }} />}
      {open === 'guide' && <GuideDialog onClose={() => setOpen(null)} />}
      {shown?.kind === 'about' && <AboutDialog about={shown.about} onClose={() => setShown(null)} />}
      {shown?.kind === 'licence' && <LicenceDialog text={shown.about.licence} onClose={() => setShown(null)} />}
      {updates.found && <UpdateDialog name={name} found={updates.found} onClose={updates.dismiss} />}
      {closing && (
        <CloseChoiceDialog onCancel={() => setClosing(false)}
          onMinimise={() => { setClosing(false); void api.minimiseToTray(refused) }}
          onQuit={() => { setClosing(false); void api.requestQuit(refused) }} />
      )}
    </div>
  )
}
