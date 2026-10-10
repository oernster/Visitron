// The confirmation shown before a website is deleted (FR-010), About (FR-072)
// and the licence (Amendment 18), ported from SymDiary and PigeonPost. Each is
// drawn in the shared Modal.

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
  onClose: () => void
}

export function AboutDialog({ about, onClose }: AboutProps) {
  // The credits can run past the dialog's height, so the body is the scroller
  // and reads itself down gently, with Close pinned beneath it. The licence and
  // the update check each have an entry of their own in the Help menu.
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
      </div>
      <div className="actions">
        <button type="button" onClick={onClose}>
          Close
        </button>
      </div>
    </Modal>
  )
}

interface LicenceProps {
  /** The licence's full text, as Go carries it on About. */
  text: string
  onClose: () => void
}

/**
 * LicenceDialog shows the licence Visitron is released under (Amendment 18),
 * ported from PigeonPost's LicenceModal. It is far longer than the dialog, so
 * the text is the scroller and reads itself down, with Close pinned beneath.
 */
export function LicenceDialog({ text, onClose }: LicenceProps) {
  const autoScroll = useAutoScroll()
  return (
    <Modal labelId="licence-title" role="dialog" onClose={onClose} pinnedActions>
      <h2 id="licence-title">Licence</h2>
      <div className="dialog-body" ref={autoScroll}>
        <pre className="licence-text">{text}</pre>
      </div>
      <div className="actions">
        <button type="button" onClick={onClose}>
          Close
        </button>
      </div>
    </Modal>
  )
}
