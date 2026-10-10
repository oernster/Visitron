// The update check (FR-075, Amendment 5), ported from Bridge Talk: when it
// runs and what the window says about it.
//
// The facade makes the check and holds the release it offered, so nothing here
// names an address: the page asks, reads the answer and hands Download and Skip
// back to the facade.

import { useCallback, useEffect, useState } from 'react'
import { api, type Refused, type Update, type UpdateOutcome } from './api'
import { Modal } from './Modal'

/** How long after the page loads the first check runs, so it never contends with the start. */
export const firstCheckMs = 3000

/** How often the check runs again while Visitron stays open. */
export const checkEveryMs = 24 * 60 * 60 * 1000

/** unreached is what a check asked for says when no answer came at all. */
const unreached: Update = { outcome: 'unreachable', running: '', latest: '' }

/**
 * useUpdateCheck runs the automatic checks and answers what the dialog should
 * show, with the check the Help menu asks for. An automatic check shows only a release
 * it offers; one asked for shows whatever it found.
 */
export function useUpdateCheck() {
  const [found, setFound] = useState<Update | null>(null)

  const check = useCallback((manual: boolean) => {
    // A refusal is an answer that did not come. An automatic check says
    // nothing about it; a check asked for is told GitHub could not be reached,
    // which is what the owner can act on.
    void api
      .checkForUpdates(manual, () => undefined)
      .then((update) => {
        const answer = update ?? (manual ? unreached : null)
        if (answer && (manual || answer.outcome === 'available')) setFound(answer)
      })
  }, [])

  useEffect(() => {
    const first = window.setTimeout(() => check(false), firstCheckMs)
    const daily = window.setInterval(() => check(false), checkEveryMs)
    return () => {
      window.clearTimeout(first)
      window.clearInterval(daily)
    }
  }, [check])

  return { found, checkNow: () => check(true), dismiss: () => setFound(null) }
}

/**
 * said words every outcome a dialog can show but an offer, which is a question
 * of its own. Skipped and off are never shown: a check asked for sends no skip
 * and ignores the switch, while an automatic one shows only an offer.
 */
function said(outcome: UpdateOutcome, update: Update, name: string): string {
  switch (outcome) {
    case 'current':
      return 'You are running the latest version.'
    case 'none':
      return `No release of ${name} has been published yet.`
    case 'uncomparable':
      return `This copy was built from source as ${update.running}, so there is no released version to compare it with.`
    case 'nosource':
      return `This copy of ${name} was built without a repository to look for releases in, so it cannot check for updates.`
    default:
      return 'The update check could not reach GitHub. Please try again later.'
  }
}

interface Props {
  /** The product's name, as Go states it. */
  name: string
  found: Update
  onClose: () => void
}

/**
 * UpdateDialog says what a check found. An offer asks Download, Skip this
 * version or Later, Download first so the dialog opens on it; anything else is
 * read and closed. A Download or a Skip the facade refused keeps the dialog
 * open saying why.
 */
export function UpdateDialog({ name, found, onClose }: Props) {
  const [problem, setProblem] = useState('')
  const offered = found.outcome === 'available'
  const act = async (call: (refused: Refused) => Promise<true | null>) => {
    if (await call(setProblem)) onClose()
  }
  return (
    <Modal labelId="update-title" role="dialog" onClose={onClose}>
      <h2 id="update-title">{offered ? 'Update available' : 'Check for updates'}</h2>
      <p>{offered ? `${name} ${found.latest} is available. You are running ${found.running}.` : said(found.outcome, found, name)}</p>
      {problem && (
        <p className="problem" role="alert">
          {problem}
        </p>
      )}
      <div className="actions">
        {offered ? (
          <>
            <button type="button" onClick={() => void act(api.downloadUpdate)}>
              Download
            </button>
            <button type="button" onClick={() => void act(api.skipUpdate)}>
              Skip this version
            </button>
            <button type="button" onClick={onClose}>
              Later
            </button>
          </>
        ) : (
          <button type="button" onClick={onClose}>
            Close
          </button>
        )}
      </div>
    </Modal>
  )
}
