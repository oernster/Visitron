// The confirmation shown before a website is deleted (FR-010) and About
// (FR-072), ported from SymDiary. Both are drawn in the shared Modal.

import type { About } from './api'
import { Modal } from './Modal'
import { useAutoScroll } from './useAutoScroll'
import crest from './assets/icons/application-icon.png'

interface ConfirmProps {
  text: string
  confirmLabel: string
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({ text, confirmLabel, onConfirm, onCancel }: ConfirmProps) {
  // Cancel is drawn first, so the dialog's own first-stop rule lands the focus
  // on it: the safe answer is the one a stray Enter reaches.
  return (
    <Modal labelId="confirm-text" role="alertdialog" onClose={onCancel}>
      <p id="confirm-text">{text}</p>
      <div className="actions">
        <button type="button" onClick={onCancel}>
          Cancel
        </button>
        <button type="button" className="danger" onClick={onConfirm}>
          {confirmLabel}
        </button>
      </div>
    </Modal>
  )
}

interface AboutProps {
  about: About
  /** Runs the update check asked for (FR-075): About is where it lives, as there is no Help menu. */
  onCheckUpdates: () => void
  onClose: () => void
}

export function AboutDialog({ about, onCheckUpdates, onClose }: AboutProps) {
  // The credits and the licence run past the dialog's height, so the body is
  // the scroller and reads itself down gently, with Close pinned beneath it.
  const autoScroll = useAutoScroll()
  return (
    <Modal labelId="about-title" role="dialog" onClose={onClose} pinnedActions>
      <div className="dialog-body" ref={autoScroll}>
        <img className="crest" src={crest} alt="" />
        <h2 id="about-title">
          {about.name} {about.version}
        </h2>
        <p>By {about.author}</p>
        <p className="copyright">© {about.author}</p>
        <h3>Built with</h3>
        <ul className="credits">
          {about.credits.map((credit) => (
            <li key={credit.work}>
              {credit.work}: {credit.licence}, {credit.holder}
            </li>
          ))}
        </ul>
        <h3>Licence</h3>
        <pre className="licence-text">{about.licence}</pre>
      </div>
      <div className="actions">
        <button type="button" onClick={onCheckUpdates}>
          Check for updates
        </button>
        <button type="button" onClick={onClose}>
          Close
        </button>
      </div>
    </Modal>
  )
}
