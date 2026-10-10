// Add website and Edit website (FR-001 to FR-009). The owner types an
// address and presses Find; the crawl's repos come back with ticks; Save
// records the website with the ticked ones.

import { useState } from 'react'
import { api, type Proposal, type Refused } from './api'
import { onEnter } from './keys'
import { Modal } from './Modal'

interface Props {
  /** The website being edited; absent when adding. */
  editing?: { id: number; url: string }
  refused: Refused
  onSaved: () => void
  onClose: () => void
}

export function WebsiteDialog({ editing, refused, onSaved, onClose }: Props) {
  const [entry, setEntry] = useState(editing?.url ?? '')
  const [proposal, setProposal] = useState<Proposal | null>(null)
  const [ticked, setTicked] = useState<string[]>([])
  const [typed, setTyped] = useState('')
  const [busy, setBusy] = useState(false)
  const [problem, setProblem] = useState('')
  const say: Refused = (reason) => setProblem(reason)
  const title = editing ? 'Edit website' : 'Add website'

  const find = async () => {
    setBusy(true)
    setProblem('')
    const found = await api.propose(entry, editing?.id ?? 0, say)
    setBusy(false)
    if (!found) return
    setProposal(found)
    setTicked(found.ticked)
    if (found.problem) setProblem(`The site could not be read: ${found.problem}. Add its repositories by hand.`)
  }

  const addTyped = async () => {
    const name = await api.confirmRepo(typed, say)
    if (!name || !proposal) return
    if (!proposal.found.includes(name)) setProposal({ ...proposal, found: [...proposal.found, name] })
    if (!ticked.includes(name)) setTicked([...ticked, name])
    setTyped('')
  }

  const toggle = (name: string) =>
    setTicked(ticked.includes(name) ? ticked.filter((t) => t !== name) : [...ticked, name])

  const save = async () => {
    if (!proposal) return
    const id = await api.saveWebsite(editing?.id ?? 0, proposal.url, ticked, refused)
    if (id !== null) onSaved()
  }

  return (
    <Modal labelId="website-title" role="dialog" onClose={onClose}>
      <h2 id="website-title">{title}</h2>
      <label className="field">
        Website address
        <input type="text" value={entry} placeholder="symdiary.com"
          onChange={(e) => { setEntry(e.target.value); setProposal(null) }}
          onKeyDown={onEnter(() => void find(), !busy && entry.trim() !== '')} />
      </label>
      <div className="actions">
        <button type="button" disabled={busy || entry.trim() === ''} onClick={() => void find()}>
          {busy ? 'Reading the site' : 'Find repositories'}
        </button>
      </div>
      {problem && <p className="problem" role="alert">{problem}</p>}
      {proposal && (
        <>
          <p>
            Downloads count from the ticked repositories of <b>{proposal.url}</b>.
          </p>
          <ul className="repo-list">
            {proposal.found.map((name) => (
              <li key={name}>
                <label>
                  <input type="checkbox" checked={ticked.includes(name)} onChange={() => toggle(name)} />
                  {name}
                </label>
              </li>
            ))}
          </ul>
          <label className="field">
            Add a repository by hand
            <input type="text" value={typed} placeholder="owner/name"
              onChange={(e) => setTyped(e.target.value)}
              onKeyDown={onEnter(() => void addTyped(), typed.trim() !== '')} />
          </label>
          <div className="actions">
            <button type="button" disabled={typed.trim() === ''} onClick={() => void addTyped()}>
              Add repository
            </button>
          </div>
        </>
      )}
      <div className="actions">
        <button type="button" onClick={onClose}>
          Cancel
        </button>
        <button type="button" disabled={!proposal} onClick={() => void save()}>
          Save
        </button>
      </div>
    </Modal>
  )
}
