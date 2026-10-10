// Settings (FR-060 to FR-063). Each change is saved as it is made; the key
// and token are shown only as set or not (FR-062) and tried once on save.

import { useEffect, useState } from 'react'
import { api, type Refused, type SecretName, type Settings } from './api'
import { Modal } from './Modal'

interface Props {
  refused: Refused
  onClose: () => void
}

// Each secret says where it comes from (Amendment 7). The steps were read
// from GoatCounter's API help and GitHub's token and rate-limit pages on
// 2026-10-10; re-read them there when either site changes its menus.
const secrets: { which: SecretName; label: string; set: keyof Settings; why: string; steps: string[] }[] = [
  {
    which: 'goatcounter', label: 'GoatCounter API key', set: 'goatCounterSet',
    why: 'Needed for page loads: without it every Page loads figure stays at 0.',
    steps: [
      'Open oernster.goatcounter.com in your browser and sign in.',
      'Choose your username in the top menu, then API.',
      'Create a new key with the Read statistics permission; it needs nothing else.',
      'Copy the key, paste it below and press Save. Visitron tries it at once and says whether it works.',
    ],
  },
  {
    which: 'github', label: 'GitHub token (optional)', set: 'gitHubTokenSet',
    why: 'Downloads are read without one; GitHub then allows 60 requests an hour, which a token raises to 5,000.',
    steps: [
      'On github.com, choose your profile picture, then Settings.',
      'Choose Developer settings at the foot of the left-hand list, then Personal access tokens, Fine-grained tokens, Generate new token.',
      'Name it Visitron and pick an expiry. Leave every permission unticked: a token can always read public repositories.',
      'Press Generate token, copy it (GitHub shows it only once), paste it below and press Save.',
    ],
  },
]

export function SettingsDialog({ refused, onClose }: Props) {
  const [settings, setSettings] = useState<Settings | null>(null)
  const [typed, setTyped] = useState<Record<SecretName, string>>({ goatcounter: '', github: '' })
  const [note, setNote] = useState('')
  const reload = () => void api.settings(refused).then((found) => found && setSettings(found))
  useEffect(reload, [refused])

  const saveSecret = async (which: SecretName) => {
    const problem = await api.saveSecret(which, typed[which], refused)
    if (problem === null) return
    setNote(problem ? `Saved; it did not work: ${problem}` : 'Saved; it works.')
    setTyped({ ...typed, [which]: '' })
    reload()
  }
  const removeSecret = async (which: SecretName) => {
    if (await api.removeSecret(which, refused)) reload()
  }
  const interval = async (hours: number) => {
    if (await api.saveInterval(hours, refused)) reload()
  }

  return (
    <Modal labelId="settings-title" role="dialog" onClose={onClose}>
      <h2 id="settings-title">Settings</h2>
      {settings && (
        <>
          {secrets.map(({ which, label, set, why, steps }) => (
            <div className="field" key={which}>
              <span>
                {label}: {settings[set] ? 'set' : 'not set'}
              </span>
              <p className="secret-help">{why}</p>
              <ol className="secret-steps">
                {steps.map((step) => (
                  <li key={step}>{step}</li>
                ))}
              </ol>
              <input type="password" autoComplete="off" value={typed[which]} aria-label={label}
                onChange={(e) => setTyped({ ...typed, [which]: e.target.value })} />
              <div className="actions">
                <button type="button" disabled={typed[which] === ''} onClick={() => void saveSecret(which)}>
                  Save
                </button>
                <button type="button" disabled={!settings[set]} onClick={() => void removeSecret(which)}>
                  Remove
                </button>
              </div>
            </div>
          ))}
          {note && <p role="status">{note}</p>}
          <label className="field">
            Check every (hours)
            <input type="number" min={settings.minInterval} max={settings.maxInterval}
              value={settings.intervalHours}
              onChange={(e) => void interval(Number(e.target.value))} />
          </label>
          <label className="check">
            <input type="checkbox" checked={settings.startWithWindows}
              onChange={(e) => void api.saveStartWithWindows(e.target.checked, refused).then(reload)} />
            Start with Windows
          </label>
          <label className="check">
            <input type="checkbox" checked={settings.updateCheck}
              onChange={(e) => void api.saveUpdateCheck(e.target.checked, refused).then(reload)} />
            Check for a newer Visitron
          </label>
        </>
      )}
      <div className="actions">
        <button type="button" onClick={onClose}>
          Close
        </button>
      </div>
    </Modal>
  )
}
